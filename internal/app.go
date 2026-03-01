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

// chatState holds a per-chat mutex to serialise agent runs,
// plus the cancel func of the currently running agent (if any).
type chatState struct {
	mu     sync.Mutex
	cancel atomic.Pointer[context.CancelFunc]
}

type App struct {
	config    Config
	bot       *TelegramBot
	agentRunner *AgentRunner
	scheduler *Scheduler
	chats     sync.Map // map[int64]*chatState
}

func NewApp(cfg Config) *App {
	a := &App{config: cfg}

	sessions, err := NewSessionStore(cfg.DataDir + "/sessions.json")
	if err != nil {
		log.Fatalf("failed to load session store: %v", err)
	}

	a.agentRunner = NewAgentRunner(cfg, sessions)

	bot, err := NewTelegramBot(cfg.TelegramToken, filepath.Join(cfg.WorkspaceDir, "files"), a.onMessage)
	if err != nil {
		log.Fatalf("failed to create telegram bot: %v", err)
	}
	a.bot = bot
	a.bot.onCancel = a.cancelAgent
	a.bot.onRestart = a.restartAgent

	a.scheduler = NewScheduler(cfg, a.runQueuedTask, a.bot)

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

func (a *App) getChatState(chatID int64) *chatState {
	val, _ := a.chats.LoadOrStore(chatID, &chatState{})
	return val.(*chatState)
}

func (a *App) onMessage(msg models.Message) {
	if !a.isAllowed(msg.ChatID) {
		log.Printf("message from unauthorised chat %d, ignoring", msg.ChatID)
		return
	}

	input := models.AgentInput{
		ChatID:          msg.ChatID,
		MessageID:       msg.MessageID,
		Prompt:          msg.Content,
		FilePath:        msg.FilePath,
		ReplyToSender:   msg.ReplyToSender,
		ReplyToContent:  msg.ReplyToContent,
		ReplyToFilePath: msg.ReplyToFilePath,
	}

	go a.runQueued(input)
}

// runQueuedTask is the RunFunc used by the scheduler — acquires the mutex and runs the agent.
func (a *App) runQueuedTask(ctx context.Context, input models.AgentInput) (models.AgentOutput, error) {
	cs := a.getChatState(input.ChatID)
	cs.mu.Lock()
	defer cs.mu.Unlock()

	return a.agentRunner.Run(ctx, input, nil)
}

// runQueued acquires the per-chat mutex, blocking until any prior agent finishes.
func (a *App) runQueued(input models.AgentInput) {
	cs := a.getChatState(input.ChatID)
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
		a.bot.SendTyping(input.ChatID)
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.bot.SendTyping(input.ChatID)
			}
		}
	}()

	tracker := newStatusTracker()
	var statusMsgID int64

	// Debounce timer to avoid rapid edits hitting Telegram rate limits
	var mu sync.Mutex
	var debounceTimer *time.Timer
	var lastStatusText string

	onToolUse := func(toolName, label string) {
		mu.Lock()
		defer mu.Unlock()

		first := tracker.Add(toolName, label)

		if first {
			lastStatusText = tracker.Render()
			statusMsgID = a.bot.SendStatusMessage(input.ChatID, lastStatusText)
			return
		}

		if statusMsgID == 0 {
			return
		}

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

	var callback func(string, string)
	if a.config.ShowStatusUpdates {
		callback = onToolUse
	}
	output, err := a.agentRunner.Run(ctx, input, callback)

	mu.Lock()
	if debounceTimer != nil {
		debounceTimer.Stop()
	}
	mu.Unlock()

	if err != nil {
		if ctx.Err() == context.Canceled {
			log.Printf("agent cancelled for chat %d", input.ChatID)
			if statusMsgID != 0 {
				a.bot.EditMessage(input.ChatID, statusMsgID, tracker.RenderDone()+"❌ Cancelled")
			}
			a.bot.SendReply(input.ChatID, input.MessageID, "Cancelled.")
			return
		}
		log.Printf("agent error for chat %d: %v", input.ChatID, err)
		if statusMsgID != 0 {
			a.bot.EditMessage(input.ChatID, statusMsgID, tracker.RenderDone()+"❌ Error")
		}
		a.bot.SendMessage(input.ChatID, "Sorry, I encountered an error. Check logs for details.")
		return
	}

	if statusMsgID != 0 {
		a.bot.EditMessage(input.ChatID, statusMsgID, tracker.RenderFinal())
	}

	if output.Result != "" {
		if err := a.bot.SendMessage(input.ChatID, output.Result); err != nil {
			log.Printf("error sending message to chat %d: %v", input.ChatID, err)
		}
	}
}

func (a *App) restartAgent(chatID int64) {
	if !a.isAllowed(chatID) {
		log.Printf("restart from unauthorised chat %d, ignoring", chatID)
		return
	}

	// Cancel any running agent so the restart doesn't queue behind it.
	cs := a.getChatState(chatID)
	if fn := cs.cancel.Load(); fn != nil {
		(*fn)()
	}

	a.bot.SendMessage(chatID, "Restarting miniclaw...")

	input := models.AgentInput{
		ChatID: chatID,
		Prompt: "/restart",
	}

	go a.runQueued(input)
}

func (a *App) cancelAgent(chatID int64) {
	cs := a.getChatState(chatID)
	fn := cs.cancel.Load()
	if fn == nil {
		a.bot.SendMessage(chatID, "Nothing to cancel.")
		return
	}
	(*fn)()
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
