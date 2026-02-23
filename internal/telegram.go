package internal

import (
	"fmt"
	"log"
	"strings"

	"goclaw/internal/models"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
)

const maxMessageLength = 4096

type TelegramBot struct {
	bot       *gotgbot.Bot
	updater   *ext.Updater
	onMessage func(msg models.Message)
}

func NewTelegramBot(token string, onMessage func(msg models.Message)) (*TelegramBot, error) {
	b, err := gotgbot.NewBot(token, nil)
	if err != nil {
		return nil, fmt.Errorf("creating bot: %w", err)
	}

	tb := &TelegramBot{
		bot:       b,
		onMessage: onMessage,
	}

	dispatcher := ext.NewDispatcher(&ext.DispatcherOpts{
		Error: func(_ *gotgbot.Bot, _ *ext.Context, err error) ext.DispatcherAction {
			log.Printf("telegram dispatcher error: %v", err)
			return ext.DispatcherActionNoop
		},
	})

	// Handle /chatid command
	dispatcher.AddHandler(handlers.NewCommand("chatid", tb.handleChatID))

	// Handle all text messages
	dispatcher.AddHandler(handlers.NewMessage(nil, tb.handleMessage))

	tb.updater = ext.NewUpdater(dispatcher, nil)

	return tb, nil
}

func (tb *TelegramBot) Start() error {
	return tb.updater.StartPolling(tb.bot, &ext.PollingOpts{
		DropPendingUpdates: true,
	})
}

func (tb *TelegramBot) Stop() {
	tb.updater.Stop()
}

func (tb *TelegramBot) handleChatID(b *gotgbot.Bot, ctx *ext.Context) error {
	_, err := ctx.EffectiveMessage.Reply(b, fmt.Sprintf("Chat ID: <code>%d</code>", ctx.EffectiveChat.Id), &gotgbot.SendMessageOpts{
		ParseMode: "HTML",
	})
	return err
}

func (tb *TelegramBot) handleMessage(_ *gotgbot.Bot, ctx *ext.Context) error {
	msg := tb.parseMessage(ctx.EffectiveMessage)
	if msg.Content == "" {
		return nil
	}
	log.Printf("[recv] chat=%d sender=%q text=%q", msg.ChatID, msg.Sender, msg.Content)
	tb.onMessage(msg)
	return nil
}

func (tb *TelegramBot) parseMessage(msg *gotgbot.Message) models.Message {
	m := models.Message{
		ChatID:  msg.Chat.Id,
		Sender:  senderName(msg.From),
		Content: msg.Text,
	}

	if msg.ReplyToMessage != nil {
		m.ReplyToSender = senderName(msg.ReplyToMessage.From)
		m.ReplyToContent = msg.ReplyToMessage.Text
	}

	return m
}

func (tb *TelegramBot) SendTyping(chatID int64) {
	tb.bot.SendChatAction(chatID, "typing", nil)
}

func (tb *TelegramBot) SendMessage(chatID int64, text string) error {
	if text == "" {
		return nil
	}

	chunks := splitMessage(text)
	log.Printf("[send] chat=%d chunks=%d len=%d", chatID, len(chunks), len(text))
	for _, chunk := range chunks {
		_, err := tb.bot.SendMessage(chatID, chunk, &gotgbot.SendMessageOpts{
			ParseMode: "HTML",
		})
		if err != nil {
			// Retry without parse mode in case of HTML formatting errors
			log.Printf("[send] chat=%d HTML parse failed, retrying plain", chatID)
			_, err = tb.bot.SendMessage(chatID, chunk, nil)
			if err != nil {
				return fmt.Errorf("sending message: %w", err)
			}
		}
	}
	return nil
}

func splitMessage(text string) []string {
	if len(text) <= maxMessageLength {
		return []string{text}
	}

	var chunks []string
	for len(text) > 0 {
		if len(text) <= maxMessageLength {
			chunks = append(chunks, text)
			break
		}

		// Find a newline to split on within the limit
		cutoff := maxMessageLength
		idx := strings.LastIndex(text[:cutoff], "\n")
		if idx > 0 {
			cutoff = idx + 1 // include the newline
		}

		chunks = append(chunks, text[:cutoff])
		text = text[cutoff:]
	}

	return chunks
}

func senderName(user *gotgbot.User) string {
	if user == nil {
		return "Unknown"
	}
	if user.FirstName != "" {
		if user.LastName != "" {
			return user.FirstName + " " + user.LastName
		}
		return user.FirstName
	}
	return user.Username
}
