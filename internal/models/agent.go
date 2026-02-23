package models

type AgentInput struct {
	ChatID         int64
	Prompt         string
	ReplyToSender  string // who sent the message being replied to (empty if not a reply)
	ReplyToContent string // content of the message being replied to (empty if not a reply)
}

type AgentOutput struct {
	Result string
	Status string // "success" or "error"
	Error  string
}
