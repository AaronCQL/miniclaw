package models

// Message is the internal representation passed from TelegramBot to App.
// Not persisted — used only for allowlist check and agent dispatch.
type Message struct {
	ChatID         int64
	Sender         string
	Content        string
	ReplyToSender  string
	ReplyToContent string
}
