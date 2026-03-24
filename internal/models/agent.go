package models

type AgentInput struct {
	ChatID          int64
	ThreadID        int64
	MessageID       int64 // telegram message ID of the user's message
	Prompt          string
	FilePath        string // local path to downloaded attachment, if any
	ReplyToSender   string // who sent the message being replied to (empty if not a reply)
	ReplyToContent  string // content of the message being replied to (empty if not a reply)
	ReplyToFilePath string // local path to replied-to message's attachment, if any
	IsolatedSession bool   // run in an isolated throwaway session, don't resume or save
	TaskName        string
}

type AgentOutput struct {
	Result     string
	Status     string // "success" or "error"
	Error      string
	ModelUsage map[string]ModelUsage
}

type ModelUsage struct {
	InputTokens              int     `json:"inputTokens"`
	OutputTokens             int     `json:"outputTokens"`
	CacheReadInputTokens     int     `json:"cacheReadInputTokens"`
	CacheCreationInputTokens int     `json:"cacheCreationInputTokens"`
	CostUSD                  float64 `json:"costUSD"`
	ContextWindow            int     `json:"contextWindow"`
}
