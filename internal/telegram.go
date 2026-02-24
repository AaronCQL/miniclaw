package internal

import (
	"fmt"
	"log"
	"strings"

	"miniclaw/internal/models"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
)

const maxMessageLength = 4096

type TelegramBot struct {
	bot       *gotgbot.Bot
	updater   *ext.Updater
	onMessage func(msg models.Message)
	onCancel  func(chatID int64)
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

	// Handle commands
	dispatcher.AddHandler(handlers.NewCommand("chatid", tb.handleChatID))
	dispatcher.AddHandler(handlers.NewCommand("cancel", tb.handleCancel))

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
	log.Printf("[recv] chat=%d command=/chatid", ctx.EffectiveChat.Id)
	_, err := ctx.EffectiveMessage.Reply(b, fmt.Sprintf("Chat ID: <code>%d</code>", ctx.EffectiveChat.Id), &gotgbot.SendMessageOpts{
		ParseMode: "HTML",
	})
	return err
}

func (tb *TelegramBot) handleCancel(_ *gotgbot.Bot, ctx *ext.Context) error {
	log.Printf("[recv] chat=%d command=/cancel", ctx.EffectiveChat.Id)
	if tb.onCancel != nil {
		tb.onCancel(ctx.EffectiveChat.Id)
	}
	return nil
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
		ChatID:    msg.Chat.Id,
		MessageID: msg.MessageId,
		Sender:    senderName(msg.From),
		Content:   msg.Text,
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

// SendStatusMessage sends a plain text message and returns its message ID.
// Returns 0 on error (best-effort).
func (tb *TelegramBot) SendStatusMessage(chatID int64, text string) int64 {
	msg, err := tb.bot.SendMessage(chatID, text, &gotgbot.SendMessageOpts{
		ParseMode: "HTML",
	})
	if err != nil {
		log.Printf("[send] chat=%d failed to send status message: %v", chatID, err)
		return 0
	}
	return msg.MessageId
}

// EditMessage edits a previously sent message in-place.
// Best-effort: logs errors but doesn't return them.
func (tb *TelegramBot) EditMessage(chatID, messageID int64, text string) {
	_, _, err := tb.bot.EditMessageText(text, &gotgbot.EditMessageTextOpts{
		ChatId:    chatID,
		MessageId: messageID,
		ParseMode: "HTML",
	})
	if err != nil {
		log.Printf("[send] chat=%d msg=%d failed to edit status message: %v", chatID, messageID, err)
	}
}

func (tb *TelegramBot) SendReply(chatID int64, replyToMessageID int64, text string) error {
	if text == "" {
		return nil
	}
	_, err := tb.bot.SendMessage(chatID, text, &gotgbot.SendMessageOpts{
		ReplyParameters: &gotgbot.ReplyParameters{MessageId: replyToMessageID},
	})
	return err
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
