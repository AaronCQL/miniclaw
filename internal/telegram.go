package internal

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"miniclaw/internal/models"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
)

const maxMessageLength = 4096

type TelegramBot struct {
	bot       *gotgbot.Bot
	updater   *ext.Updater
	fileDir   string
	onMessage func(msg models.Message)
	onCancel  func(chatID, threadID int64)
	onRestart func(chatID, threadID int64)
	onLogs    func(chatID, threadID int64)
	onUsage   func(chatID, threadID int64)
}

func NewTelegramBot(token string, fileDir string, onMessage func(msg models.Message)) (*TelegramBot, error) {
	b, err := gotgbot.NewBot(token, nil)
	if err != nil {
		return nil, fmt.Errorf("creating bot: %w", err)
	}

	tb := &TelegramBot{
		bot:       b,
		fileDir:   fileDir,
		onMessage: onMessage,
	}

	dispatcher := ext.NewDispatcher(&ext.DispatcherOpts{
		Error: func(_ *gotgbot.Bot, _ *ext.Context, err error) ext.DispatcherAction {
			log.Printf("telegram dispatcher error: %v", err)
			return ext.DispatcherActionNoop
		},
	})

	dispatcher.AddHandler(handlers.NewCommand("chatid", tb.handleChatID))
	dispatcher.AddHandler(handlers.NewCommand("cancel", tb.handleCancel))
	dispatcher.AddHandler(handlers.NewCommand("restart", tb.handleRestart))
	dispatcher.AddHandler(handlers.NewCommand("logs", tb.handleLogs))
	dispatcher.AddHandler(handlers.NewCommand("usage", tb.handleUsage))

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
	chatID := ctx.EffectiveChat.Id
	threadID := ctx.EffectiveMessage.MessageThreadId
	log.Printf("[recv] chat=%d thread=%d command=/chatid", chatID, threadID)
	text := fmt.Sprintf("Chat ID: <code>%d</code>", chatID)
	if threadID > 0 {
		text += fmt.Sprintf("\nThread ID: <code>%d</code>", threadID)
	}
	opts := &gotgbot.SendMessageOpts{ParseMode: "HTML"}
	if threadID > 0 {
		opts.MessageThreadId = threadID
	}
	_, err := ctx.EffectiveMessage.Reply(b, text, opts)
	return err
}

func (tb *TelegramBot) handleCancel(_ *gotgbot.Bot, ctx *ext.Context) error {
	log.Printf("[recv] chat=%d thread=%d command=/cancel", ctx.EffectiveChat.Id, ctx.EffectiveMessage.MessageThreadId)
	if tb.onCancel != nil {
		tb.onCancel(ctx.EffectiveChat.Id, ctx.EffectiveMessage.MessageThreadId)
	}
	return nil
}

func (tb *TelegramBot) handleRestart(_ *gotgbot.Bot, ctx *ext.Context) error {
	log.Printf("[recv] chat=%d thread=%d command=/restart", ctx.EffectiveChat.Id, ctx.EffectiveMessage.MessageThreadId)
	if tb.onRestart != nil {
		tb.onRestart(ctx.EffectiveChat.Id, ctx.EffectiveMessage.MessageThreadId)
	}
	return nil
}

func (tb *TelegramBot) handleLogs(_ *gotgbot.Bot, ctx *ext.Context) error {
	log.Printf("[recv] chat=%d thread=%d command=/logs", ctx.EffectiveChat.Id, ctx.EffectiveMessage.MessageThreadId)
	if tb.onLogs != nil {
		tb.onLogs(ctx.EffectiveChat.Id, ctx.EffectiveMessage.MessageThreadId)
	}
	return nil
}

func (tb *TelegramBot) handleUsage(_ *gotgbot.Bot, ctx *ext.Context) error {
	log.Printf("[recv] chat=%d thread=%d command=/usage", ctx.EffectiveChat.Id, ctx.EffectiveMessage.MessageThreadId)
	if tb.onUsage != nil {
		tb.onUsage(ctx.EffectiveChat.Id, ctx.EffectiveMessage.MessageThreadId)
	}
	return nil
}

func (tb *TelegramBot) handleMessage(_ *gotgbot.Bot, ctx *ext.Context) error {
	msg := tb.parseMessage(ctx.EffectiveMessage)
	if msg.Content == "" && msg.FilePath == "" {
		return nil
	}
	log.Printf("[recv] chat=%d sender=%q text=%q file=%q", msg.ChatID, msg.Sender, msg.Content, msg.FilePath)
	tb.onMessage(msg)
	return nil
}

func (tb *TelegramBot) parseMessage(msg *gotgbot.Message) models.Message {
	m := models.Message{
		ChatID:    msg.Chat.Id,
		ThreadID:  msg.MessageThreadId,
		MessageID: msg.MessageId,
		Sender:    senderName(msg.From),
		Content:   msg.GetText(),
	}

	if fileID, fileName := extractFileID(msg); fileID != "" {
		dstDir := filepath.Join(tb.fileDir, fmt.Sprintf("%d", msg.Chat.Id))
		path, err := tb.downloadFile(fileID, fileName, dstDir)
		if err != nil {
			log.Printf("[recv] chat=%d failed to download file: %v", msg.Chat.Id, err)
		} else {
			m.FilePath = path
		}
	}

	if msg.ReplyToMessage != nil {
		m.ReplyToSender = senderName(msg.ReplyToMessage.From)
		if msg.Quote != nil && msg.Quote.Text != "" {
			m.ReplyToContent = msg.Quote.Text
		} else {
			m.ReplyToContent = msg.ReplyToMessage.GetText()
		}

		if fileID, fileName := extractFileID(msg.ReplyToMessage); fileID != "" {
			dstDir := filepath.Join(tb.fileDir, fmt.Sprintf("%d", msg.Chat.Id))
			path, err := tb.downloadFile(fileID, fileName, dstDir)
			if err != nil {
				log.Printf("[recv] chat=%d failed to download reply-to file: %v", msg.Chat.Id, err)
			} else {
				m.ReplyToFilePath = path
			}
		}
	}

	return m
}

func extractFileID(msg *gotgbot.Message) (fileID, fileName string) {
	if len(msg.Photo) > 0 {
		return msg.Photo[len(msg.Photo)-1].FileId, ""
	}
	if msg.Document != nil {
		return msg.Document.FileId, msg.Document.FileName
	}
	if msg.Video != nil {
		return msg.Video.FileId, msg.Video.FileName
	}
	if msg.Audio != nil {
		return msg.Audio.FileId, msg.Audio.FileName
	}
	if msg.Voice != nil {
		return msg.Voice.FileId, ""
	}
	return "", ""
}

func (tb *TelegramBot) downloadFile(fileID, fileName, dstDir string) (string, error) {
	file, err := tb.bot.GetFile(fileID, nil)
	if err != nil {
		return "", fmt.Errorf("getting file info: %w", err)
	}

	if fileName == "" {
		ext := filepath.Ext(file.FilePath)
		if ext == "" {
			ext = ".jpg" // default for photos with no extension
		} else if ext == ".oga" {
			ext = ".ogg" // Groq Whisper API doesn't accept .oga
		}
		fileName = file.FileUniqueId + ext
	} else {
		ext := filepath.Ext(fileName)
		fileName = strings.TrimSuffix(fileName, ext) + "_" + file.FileUniqueId + ext
	}

	dstPath := filepath.Join(dstDir, fileName)

	if _, err := os.Stat(dstPath); err == nil {
		return dstPath, nil
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(file.URL(tb.bot, nil))
	if err != nil {
		return "", fmt.Errorf("downloading file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return "", fmt.Errorf("creating file directory: %w", err)
	}

	out, err := os.Create(dstPath)
	if err != nil {
		return "", fmt.Errorf("creating file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		os.Remove(dstPath)
		return "", fmt.Errorf("writing file: %w", err)
	}

	return dstPath, nil
}

func (tb *TelegramBot) SendTyping(chatID, threadID int64) {
	opts := &gotgbot.SendChatActionOpts{}
	if threadID > 0 {
		opts.MessageThreadId = threadID
	}
	tb.bot.SendChatAction(chatID, "typing", opts)
}

// Returns 0 on error (best-effort).
func (tb *TelegramBot) SendStatusMessage(chatID, threadID int64, text string) int64 {
	opts := &gotgbot.SendMessageOpts{ParseMode: "HTML"}
	if threadID > 0 {
		opts.MessageThreadId = threadID
	}
	msg, err := tb.bot.SendMessage(chatID, text, opts)
	if err != nil {
		log.Printf("[send] chat=%d failed to send status message: %v", chatID, err)
		return 0
	}
	return msg.MessageId
}

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

func (tb *TelegramBot) SendReply(chatID, threadID, replyToMessageID int64, text string) error {
	if text == "" {
		return nil
	}
	opts := &gotgbot.SendMessageOpts{
		ReplyParameters: &gotgbot.ReplyParameters{MessageId: replyToMessageID},
	}
	if threadID > 0 {
		opts.MessageThreadId = threadID
	}
	_, err := tb.bot.SendMessage(chatID, text, opts)
	return err
}

func (tb *TelegramBot) SendFile(chatID, threadID int64, filePath string, caption string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("opening file %s: %w", filePath, err)
	}
	defer f.Close()

	fileName := filepath.Base(filePath)
	opts := &gotgbot.SendDocumentOpts{}
	if caption != "" {
		opts.Caption = caption
		opts.ParseMode = "HTML"
	}
	if threadID > 0 {
		opts.MessageThreadId = threadID
	}

	_, err = tb.bot.SendDocument(chatID, gotgbot.InputFileByReader(fileName, f), opts)
	if err != nil {
		return fmt.Errorf("sending file %s: %w", fileName, err)
	}
	log.Printf("[send] chat=%d file=%s", chatID, fileName)
	return nil
}

func (tb *TelegramBot) SendMessage(chatID, threadID int64, text string) error {
	if text == "" {
		return nil
	}

	chunks := splitMessage(text)
	log.Printf("[send] chat=%d thread=%d chunks=%d len=%d", chatID, threadID, len(chunks), len(text))
	for _, chunk := range chunks {
		opts := &gotgbot.SendMessageOpts{ParseMode: "HTML"}
		if threadID > 0 {
			opts.MessageThreadId = threadID
		}
		_, err := tb.bot.SendMessage(chatID, chunk, opts)
		if err != nil {
			// Retry without parse mode in case of HTML formatting errors
			log.Printf("[send] chat=%d HTML parse failed, retrying plain", chatID)
			retryOpts := &gotgbot.SendMessageOpts{}
			if threadID > 0 {
				retryOpts.MessageThreadId = threadID
			}
			_, err = tb.bot.SendMessage(chatID, chunk, retryOpts)
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
