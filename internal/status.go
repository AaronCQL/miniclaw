package internal

import "strings"

var toolEmoji = map[string]string{
	"Read":      "📄",
	"Edit":      "✏️",
	"Write":     "✏️",
	"Bash":      "⚡",
	"Grep":      "🔎",
	"Glob":      "🔎",
	"WebSearch": "🌐",
	"WebFetch":  "🌐",
	"Task":      "🤖",
}

type statusEntry struct {
	emoji string
	label string
}

type statusTracker struct {
	entries []statusEntry
}

func newStatusTracker() *statusTracker {
	return &statusTracker{}
}

// Add appends a tool action and returns true if this is the first entry.
func (s *statusTracker) Add(toolName, label string) bool {
	emoji, ok := toolEmoji[toolName]
	if !ok {
		emoji = "⚙️"
	}
	if label == "" {
		label = toolName
	}

	first := len(s.entries) == 0
	s.entries = append(s.entries, statusEntry{emoji: emoji, label: label})
	return first
}

// Render returns the status text while the agent is still running.
func (s *statusTracker) Render() string {
	if len(s.entries) == 0 {
		return ""
	}

	var text string
	for i, e := range s.entries {
		text += e.emoji + " " + e.label
		if i < len(s.entries)-1 {
			text += "\n"
		} else {
			text += " 🟡"
		}
	}
	return text
}

// RenderDone returns all entries as completed. Used as a base for final/cancel/error states.
func (s *statusTracker) RenderDone() string {
	if len(s.entries) == 0 {
		return ""
	}

	var text string
	for _, e := range s.entries {
		text += e.emoji + " " + e.label + "\n"
	}
	return text
}

// RenderFinal returns all entries marked as complete.
func (s *statusTracker) RenderFinal() string {
	return strings.TrimRight(s.RenderDone(), "\n")
}
