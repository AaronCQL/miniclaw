package internal

import (
	"context"
	"log"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"miniclaw/internal/models"
)

// chatThreadKey identifies a unique chat+thread combination for per-thread concurrency.
type chatThreadKey struct {
	chatID   int64
	threadID int64
}

// chatState holds a per-chat/thread mutex to serialise agent runs,
// plus the cancel func of the currently running agent (if any).
type chatState struct {
	mu     sync.Mutex
	cancel atomic.Pointer[context.CancelFunc]
}

type App struct {
	config      Config
	bot         *TelegramBot
	agentRunner *AgentRunner
	scheduler   *Scheduler
	chats       sync.Map // map[int64]*chatState
	statusLevel atomic.Value
}

func NewApp(cfg Config) *App {
	a := &App{config: cfg}

	sessions := NewSessionStore(cfg.DataDir + "/sessions.json")
	a.agentRunner = NewAgentRunner(cfg, sessions)

	settings := LoadSettings(cfg.DataDir)
	a.statusLevel.Store(settings.StatusLevel)

	bot, err := NewTelegramBot(cfg.TelegramToken, filepath.Join(cfg.WorkspaceDir, "files"), a.onMessage)
	if err != nil {
		log.Fatalf("failed to create telegram bot: %v", err)
	}
	a.bot = bot
	a.bot.onCancel = a.cancelAgent
	a.bot.onRestart = a.restartAgent
	a.bot.onLogs = a.toggleLogs

	a.scheduler = NewScheduler(cfg, a.runQueuedTask, a.sendAgentOutput)

	return a
}

func (a *App) Start(ctx context.Context) error {
	if err := a.bot.Start(); err != nil {
		return err
	}
	log.Println("telegram bot started")

	go a.scheduler.Start(ctx)
	log.Println("scheduler started")

	<-ctx.Done()

	a.bot.Stop()
	log.Println("shutting down")
	return nil
}

func (a *App) getChatState(chatID, threadID int64) *chatState {
	key := chatThreadKey{chatID: chatID, threadID: threadID}
	val, _ := a.chats.LoadOrStore(key, &chatState{})
	return val.(*chatState)
}

func (a *App) onMessage(msg models.Message) {
	if !a.isAllowed(msg.ChatID) {
		log.Printf("message from unauthorised chat %d, ignoring", msg.ChatID)
		return
	}

	input := models.AgentInput{
		ChatID:          msg.ChatID,
		ThreadID:        msg.ThreadID,
		MessageID:       msg.MessageID,
		Prompt:          msg.Content,
		FilePath:        msg.FilePath,
		ReplyToSender:   msg.ReplyToSender,
		ReplyToContent:  msg.ReplyToContent,
		ReplyToFilePath: msg.ReplyToFilePath,
	}

	go a.runQueued(input)
}

// runQueuedTask is the RunFunc used by the scheduler. Acquires the mutex and runs the agent.
func (a *App) runQueuedTask(ctx context.Context, input models.AgentInput) (models.AgentOutput, error) {
	cs := a.getChatState(input.ChatID, input.ThreadID)
	cs.mu.Lock()
	defer cs.mu.Unlock()

	return a.agentRunner.Run(ctx, input, nil, nil)
}

// runQueued acquires the per-chat/thread mutex, blocking until any prior agent finishes.
func (a *App) runQueued(input models.AgentInput) {
	cs := a.getChatState(input.ChatID, input.ThreadID)
	cs.mu.Lock()
	defer cs.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	cs.cancel.Store(&cancel)
	defer cs.cancel.Store(nil)

	a.startAgent(ctx, cancel, input)
}

func (a *App) startAgent(ctx context.Context, cancel context.CancelFunc, input models.AgentInput) {
	defer cancel()

	go func() {
		a.bot.SendTyping(input.ChatID, input.ThreadID)
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.bot.SendTyping(input.ChatID, input.ThreadID)
			}
		}
	}()

	tracker := newStatusTracker()
	var statusMsgID int64

	// Debounce timer to avoid rapid edits hitting Telegram rate limits
	var mu sync.Mutex
	var debounceTimer *time.Timer
	var lastStatusText string

	scheduleStatusUpdate := func() {
		if debounceTimer != nil {
			debounceTimer.Stop()
		}
		debounceTimer = time.AfterFunc(1*time.Second, func() {
			mu.Lock()
			text := tracker.Render()
			changed := text != lastStatusText
			if changed {
				lastStatusText = text
			}
			mu.Unlock()
			if changed {
				a.bot.EditMessage(input.ChatID, statusMsgID, text)
			}
		})
	}

	onToolUse := func(toolName, label string) {
		mu.Lock()
		defer mu.Unlock()

		first := tracker.Add(toolName, label)

		if first {
			lastStatusText = tracker.Render()
			statusMsgID = a.bot.SendStatusMessage(input.ChatID, input.ThreadID, lastStatusText)
			return
		}

		if statusMsgID == 0 {
			return
		}

		scheduleStatusUpdate()
	}

	onText := func(text string) {
		mu.Lock()
		defer mu.Unlock()

		tracker.AddText(text)

		if statusMsgID == 0 {
			lastStatusText = tracker.Render()
			statusMsgID = a.bot.SendStatusMessage(input.ChatID, input.ThreadID, lastStatusText)
			return
		}

		scheduleStatusUpdate()
	}

	var toolCallback func(string, string)
	var textCallback func(string)
	switch a.statusLevel.Load().(string) {
	case StatusThinking:
		textCallback = onText
	case StatusVerbose:
		toolCallback = onToolUse
		textCallback = onText
	}
	output, err := a.agentRunner.Run(ctx, input, toolCallback, textCallback)

	mu.Lock()
	if debounceTimer != nil {
		debounceTimer.Stop()
	}
	mu.Unlock()

	if err != nil {
		if ctx.Err() == context.Canceled {
			log.Printf("agent cancelled for chat %d thread %d", input.ChatID, input.ThreadID)
			if statusMsgID != 0 {
				a.bot.EditMessage(input.ChatID, statusMsgID, tracker.RenderDone()+"❌ Cancelled")
			}
			a.bot.SendReply(input.ChatID, input.ThreadID, input.MessageID, "Cancelled.")
			return
		}
		log.Printf("agent error for chat %d thread %d: %v", input.ChatID, input.ThreadID, err)
		if statusMsgID != 0 {
			a.bot.EditMessage(input.ChatID, statusMsgID, tracker.RenderDone()+"❌ Error")
		}
		var errMsg string
		if output.Error != "" {
			errMsg = "Sorry, I encountered an error.\n\n<pre>" + output.Error + "</pre>"
		} else {
			errMsg = "Sorry, I encountered an unknown error. Check logs for details."
		}
		a.bot.SendMessage(input.ChatID, input.ThreadID, errMsg)
		return
	}

	if statusMsgID != 0 {
		tracker.DropText(output.Result)
		a.bot.EditMessage(input.ChatID, statusMsgID, tracker.RenderFinal())
	}

	if output.Result != "" {
		a.sendAgentOutput(input.ChatID, input.ThreadID, output.Result)
	}
}

func (a *App) sendAgentOutput(chatID, threadID int64, result string) {
	outboxPath := filepath.Join(a.config.HomeDir, "outbox.json")
	entries, err := ReadOutbox(outboxPath)
	if err != nil {
		log.Printf("[outbox] chat=%d error reading outbox: %v", chatID, err)
	}

	if len(entries) > 0 {
		for _, entry := range entries {
			if err := ValidateOutboxEntry(entry); err != nil {
				log.Printf("[outbox] chat=%d skipping %s: %v", chatID, entry.Path, err)
				continue
			}
			if err := a.bot.SendFile(chatID, threadID, entry.Path, entry.Caption); err != nil {
				log.Printf("[outbox] chat=%d failed to send %s: %v", chatID, entry.Path, err)
			}
		}
		RemoveOutbox(outboxPath)
	}

	if err := a.bot.SendMessage(chatID, threadID, result); err != nil {
		log.Printf("error sending message to chat %d: %v", chatID, err)
	}
}

func (a *App) restartAgent(chatID, threadID int64) {
	if !a.isAllowed(chatID) {
		log.Printf("restart from unauthorised chat %d, ignoring", chatID)
		return
	}

	// Cancel any running agent so the restart doesn't queue behind it.
	// Restart is global: cancel the non-threaded (threadID=0) agent.
	cs := a.getChatState(chatID, 0)
	if fn := cs.cancel.Load(); fn != nil {
		(*fn)()
	}

	a.bot.SendMessage(chatID, threadID, "Restarting miniclaw...")

	input := models.AgentInput{
		ChatID:   chatID,
		ThreadID: threadID,
		Prompt:   "/restart",
	}

	go a.runQueued(input)
}

func (a *App) cancelAgent(chatID, threadID int64) {
	cs := a.getChatState(chatID, threadID)
	fn := cs.cancel.Load()
	if fn == nil {
		a.bot.SendMessage(chatID, threadID, "Nothing to cancel.")
		return
	}
	(*fn)()
}

func (a *App) toggleLogs(chatID, threadID int64) {
	if !a.isAllowed(chatID) {
		return
	}

	var next string
	switch a.statusLevel.Load().(string) {
	case StatusOff:
		next = StatusThinking
	case StatusThinking:
		next = StatusVerbose
	default:
		next = StatusOff
	}

	a.statusLevel.Store(next)
	s := LoadSettings(a.config.DataDir)
	s.StatusLevel = next
	SaveSettings(a.config.DataDir, s)

	switch next {
	case StatusOff:
		a.bot.SendMessage(chatID, threadID, "🔕 Status updates off.")
	case StatusThinking:
		a.bot.SendMessage(chatID, threadID, "💭 Status: thinking.")
	case StatusVerbose:
		a.bot.SendMessage(chatID, threadID, "🔍 Status: verbose.")
	}
}

func (a *App) isAllowed(chatID int64) bool {
	if len(a.config.AllowedChatIDs) == 0 {
		return true
	}
	for _, id := range a.config.AllowedChatIDs {
		if id == chatID {
			return true
		}
	}
	return false
}
