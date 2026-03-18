package models

type Task struct {
	Filename        string  `json:"-"` // derived from the file path, not stored in JSON
	Prompt          string  `json:"prompt"`
	ChatID          int64   `json:"chat_id"`
	ThreadID        int64   `json:"thread_id,omitempty"`
	ScheduleType    string  `json:"type"` // "once", "cron", "interval"
	ScheduleValue   string  `json:"value"`
	Status          string  `json:"status"` // "active", "paused"
	NextRun         *string `json:"next_run"`
	Expires         *string `json:"expires,omitempty"`
	IsolatedSession bool    `json:"isolated_session,omitempty"`
}
