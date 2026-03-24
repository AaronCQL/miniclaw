package internal

import (
	"context"
	"fmt"
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
	sessions    *SessionStore
	scheduler   *Scheduler
	chats       sync.Map // map[int64]*chatState
	statusLevel atomic.Value
}

func NewApp(cfg Config) *App {
	a := &App{config: cfg}

	a.sessions = NewSessionStore(cfg.DataDir + "/sessions.json")
	a.agentRunner = NewAgentRunner(cfg, a.sessions)

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
	a.bot.onUsage = a.showUsage
	a.bot.onClear = a.clearSession

	a.scheduler = NewScheduler(cfg, a.runQueuedTask)

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

	return a.runAgentWithFeedback(ctx, input)
}

// runQueued acquires the per-chat/thread mutex, blocking until any prior agent finishes.
func (a *App) runQueued(input models.AgentInput) {
	cs := a.getChatState(input.ChatID, input.ThreadID)
	cs.mu.Lock()
	defer cs.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	cs.cancel.Store(&cancel)
	defer cs.cancel.Store(nil)
	defer cancel()

	a.runAgentWithFeedback(ctx, input)
}

func (a *App) runAgentWithFeedback(ctx context.Context, input models.AgentInput) (models.AgentOutput, error) {
	if input.TaskName != "" {
		a.bot.SendMessage(input.ChatID, input.ThreadID, fmt.Sprintf("⏱ Running scheduled task <code>%s</code>...", input.TaskName))
	}

	typingCtx, typingCancel := context.WithCancel(ctx)
	defer typingCancel()
	go func() {
		a.bot.SendTyping(input.ChatID, input.ThreadID)
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-typingCtx.Done():
				return
			case <-ticker.C:
				a.bot.SendTyping(input.ChatID, input.ThreadID)
			}
		}
	}()

	tracker := newStatusTracker()
	var statusMsgID int64

	var mu sync.Mutex
	var debounceTimer *time.Timer
	var lastStatusText string
	var done bool

	flushStatus := func() {
		mu.Lock()
		if done {
			mu.Unlock()
			return
		}
		text := tracker.Render()
		changed := text != lastStatusText
		if changed {
			lastStatusText = text
		}
		msgID := statusMsgID
		mu.Unlock()
		if !changed {
			return
		}
		if msgID != 0 {
			a.bot.EditMessage(input.ChatID, msgID, text)
		} else {
			mu.Lock()
			if statusMsgID == 0 {
				statusMsgID = a.bot.SendStatusMessage(input.ChatID, input.ThreadID, text)
			}
			mu.Unlock()
		}
	}

	scheduleStatusUpdate := func() {
		if debounceTimer != nil {
			debounceTimer.Stop()
		}
		debounceTimer = time.AfterFunc(1*time.Second, flushStatus)
	}

	onToolUse := func(toolName, label string) {
		mu.Lock()
		defer mu.Unlock()
		tracker.Add(toolName, label)
		scheduleStatusUpdate()
	}

	onText := func(text string) {
		mu.Lock()
		defer mu.Unlock()
		tracker.AddText(text)
		scheduleStatusUpdate()
	}

	var toolCallback func(string, string)
	var textCallback func(string)
	switch a.statusLevel.Load().(string) {
	case StatusText:
		textCallback = onText
	case StatusVerbose:
		toolCallback = onToolUse
		textCallback = onText
	}

	output, err := a.agentRunner.Run(ctx, input, toolCallback, textCallback)
	if output.ModelUsage != nil {
		a.sessions.UpdateUsage(input.ChatID, input.ThreadID, output.ModelUsage, input.IsolatedSession)
	}

	mu.Lock()
	done = true
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
			if input.MessageID != 0 {
				a.bot.SendReply(input.ChatID, input.ThreadID, input.MessageID, "Cancelled.")
			}
			return output, err
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
		return output, err
	}

	// Workaround: Claude CLI's stream-json sets stop_reason=null on all assistant
	// events, so we can't distinguish intermediate text from the final response
	// during streaming. We show all text immediately, then retroactively remove
	// the final response via DropText once the result event arrives.
	// TODO: simplify if Claude CLI exposes stop_reason on assistant events.
	if statusMsgID != 0 {
		tracker.DropText(output.Result)
		if final := tracker.RenderFinal(); final != "" {
			a.bot.EditMessage(input.ChatID, statusMsgID, final)
		} else if output.Result != "" {
			// Status only had the final response - edit it to become the result
			a.bot.EditMessage(input.ChatID, statusMsgID, output.Result)
			output.Result = ""
		}
	}

	if output.Result != "" {
		a.sendAgentOutput(input.ChatID, input.ThreadID, output.Result)
	}

	return output, err
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
		next = StatusText
	case StatusText:
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
		a.bot.SendMessage(chatID, threadID, "🔕 Logs: off.")
	case StatusText:
		a.bot.SendMessage(chatID, threadID, "💬 Logs: intermediate text only.")
	case StatusVerbose:
		a.bot.SendMessage(chatID, threadID, "📢 Logs: verbose (intermediate text and tool use).")
	}
}

func (a *App) clearSession(chatID, threadID int64) {
	if !a.isAllowed(chatID) {
		return
	}
	a.sessions.Clear(chatID, threadID)
}

func (a *App) showUsage(chatID, threadID int64) {
	if !a.isAllowed(chatID) {
		return
	}

	totalCost := a.sessions.TotalCost()

	if threadID == 0 {
		usage := a.sessions.GetUsage(chatID, 0)
		text := fmt.Sprintf("💸 <b>Usage</b>\n\n%s", formatCostLine("Total Cost", totalCost, usage.LastCostUSD))
		a.bot.SendMessage(chatID, threadID, text)
		return
	}

	usage := a.sessions.GetUsage(chatID, threadID)
	if usage.ContextWindow == 0 && usage.CostUSD == 0 {
		text := fmt.Sprintf("💸 <b>Usage</b>\n\n%s", formatCostLine("Total Cost", totalCost, 0))
		a.bot.SendMessage(chatID, threadID, text)
		return
	}

	contextLine := "Context: <b>no data yet</b>"
	if usage.ContextWindow > 0 {
		pct := float64(usage.ContextTokens) / float64(usage.ContextWindow) * 100
		contextLine = fmt.Sprintf("Context: <b>%s / %s</b> (%.2f%%)",
			formatTokens(usage.ContextTokens), formatTokens(usage.ContextWindow), pct)
	}

	text := fmt.Sprintf(
		"💸 <b>Usage</b>\n\n%s\n%s\n%s",
		contextLine,
		formatCostLine("Thread Cost", usage.CostUSD, usage.LastCostUSD),
		formatCostLine("Total Cost", totalCost, 0),
	)

	a.bot.SendMessage(chatID, threadID, text)
}

func formatCostLine(label string, cost, lastCost float64) string {
	line := fmt.Sprintf("%s: <b>$%.4f</b>", label, cost)
	if lastCost > 0 {
		line += fmt.Sprintf(" (+$%.4f)", lastCost)
	}
	return line
}

func formatTokens(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
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
