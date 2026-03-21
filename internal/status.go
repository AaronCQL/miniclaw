package internal

import "strings"

var toolEmoji = map[string]string{
	"Read":          "📄",
	"Edit":          "✏️",
	"Write":         "✏️",
	"Bash":          "⚡",
	"Grep":          "🔎",
	"Glob":          "🔎",
	"WebSearch":     "🌐",
	"WebFetch":      "🌐",
	"Agent":         "🧠",
	"Task":          "🤖",
	"EnterPlanMode": "📝",
	"TodoWrite":     "🏗️",
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

func (s *statusTracker) Add(toolName, label string) bool {
	if toolName == "ExitPlanMode" || toolName == "ToolSearch" || (toolName == "TodoWrite" && label == "") {
		return len(s.entries) == 0
	}
	emoji, ok := toolEmoji[toolName]
	if !ok {
		emoji = "⚙️"
	}
	if label == "" {
		label = toolName
	}

	if n := len(s.entries); n > 0 && s.entries[n-1].emoji == emoji && s.entries[n-1].label == label {
		return false
	}

	first := len(s.entries) == 0
	s.entries = append(s.entries, statusEntry{emoji: emoji, label: label})
	return first
}

func (s *statusTracker) AddText(text string) {
	s.entries = append(s.entries, statusEntry{emoji: "", label: text})
}

func (s *statusTracker) renderEntries(showSpinner bool) string {
	if len(s.entries) == 0 {
		return ""
	}

	var b strings.Builder
	for i, e := range s.entries {
		if e.emoji != "" {
			b.WriteString(e.emoji + " " + e.label)
		} else {
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString("<i>" + e.label + "</i>")
		}
		if i < len(s.entries)-1 {
			b.WriteString("\n")
		} else if showSpinner && e.emoji != "" {
			b.WriteString(" 🟡")
		}
	}
	return b.String()
}

func (s *statusTracker) Render() string {
	return s.renderEntries(true)
}

// DropText strips the final response from status since it's sent as a separate message.
func (s *statusTracker) DropText(text string) {
	for i := len(s.entries) - 1; i >= 0; i-- {
		if s.entries[i].emoji == "" && s.entries[i].label == text {
			s.entries = append(s.entries[:i], s.entries[i+1:]...)
			return
		}
	}
}

func (s *statusTracker) RenderDone() string {
	return s.renderEntries(false)
}

func (s *statusTracker) RenderFinal() string {
	return strings.TrimRight(s.RenderDone(), "\n")
}
