package internal

import (
	"context"
	"log"
	"sync"
	"time"

	"goclaw/internal/models"
)

type App struct {
	config       Config
	bot          *TelegramBot
	agentRunner  *AgentRunner
	scheduler    *Scheduler
	activeAgents sync.Map // map[int64]struct{} — chatID → running flag
}

func NewApp(cfg Config) *App {
	a := &App{config: cfg}

	sessions, err := NewSessionStore(cfg.DataDir + "/sessions.json")
	if err != nil {
		log.Fatalf("failed to load session store: %v", err)
	}

	a.agentRunner = NewAgentRunner(cfg, sessions)

	bot, err := NewTelegramBot(cfg.TelegramToken, a.onMessage)
	if err != nil {
		log.Fatalf("failed to create telegram bot: %v", err)
	}
	a.bot = bot

	a.scheduler = NewScheduler(cfg, a.agentRunner, a.bot)

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

func (a *App) onMessage(msg models.Message) {
	if !a.isAllowed(msg.ChatID) {
		log.Printf("message from unauthorized chat %d, ignoring", msg.ChatID)
		return
	}

	if _, loaded := a.activeAgents.LoadOrStore(msg.ChatID, struct{}{}); loaded {
		log.Printf("agent already running for chat %d, ignoring message", msg.ChatID)
		return
	}

	input := models.AgentInput{
		ChatID:         msg.ChatID,
		Prompt:         msg.Content,
		ReplyToSender:  msg.ReplyToSender,
		ReplyToContent: msg.ReplyToContent,
	}

	go a.startAgent(input)
}

func (a *App) startAgent(input models.AgentInput) {
	defer a.activeAgents.Delete(input.ChatID)

	// Send typing indicator every 4s until the agent finishes
	typingCtx, stopTyping := context.WithCancel(context.Background())
	defer stopTyping()
	go func() {
		a.bot.SendTyping(input.ChatID)
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-typingCtx.Done():
				return
			case <-ticker.C:
				a.bot.SendTyping(input.ChatID)
			}
		}
	}()

	output, err := a.agentRunner.Run(context.Background(), input)
	if err != nil {
		log.Printf("agent error for chat %d: %v", input.ChatID, err)
		a.bot.SendMessage(input.ChatID, "Sorry, I encountered an error. Check logs for details.")
		return
	}

	if output.Result != "" {
		if err := a.bot.SendMessage(input.ChatID, output.Result); err != nil {
			log.Printf("error sending message to chat %d: %v", input.ChatID, err)
		}
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
