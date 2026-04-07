package internal

import (
	"fmt"
	"strings"
)

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
	"Skill":         "🦾",
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

const maxStatusLen = 4000

func (s *statusTracker) renderEntries(showSpinner bool) string {
	if len(s.entries) == 0 {
		return ""
	}

	// Build lines backwards, stopping when we'd exceed the limit
	lines := make([]string, 0, len(s.entries))
	total := 0
	for i := len(s.entries) - 1; i >= 0; i-- {
		e := s.entries[i]
		var line string
		if e.emoji != "" {
			line = e.emoji + " " + e.label
		} else {
			label := e.label
			if strings.HasSuffix(label, ":") {
				label = label[:len(label)-1] + "."
			}
			line = "<i>" + FormatTelegramHTML(label) + "</i>"
		}

		// Last entry gets spinner; text entries get a blank line before them
		if i == len(s.entries)-1 && showSpinner && e.emoji != "" {
			line += " 🟡"
		}
		if e.emoji == "" && i > 0 {
			line = "\n" + line
		}

		cost := len(line) + 1 // +1 for newline separator
		if total+cost > maxStatusLen-40 {
			break
		}
		total += cost
		lines = append(lines, line)
	}

	// Reverse to restore chronological order
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}

	skipped := len(s.entries) - len(lines)
	var b strings.Builder
	if skipped > 0 {
		b.WriteString(fmt.Sprintf("... %d earlier entries\n\n", skipped))
	}
	b.WriteString(strings.Join(lines, "\n"))
	return b.String()
}

func (s *statusTracker) Render() string {
	return s.renderEntries(true)
}

// DropText strips the final response from status since it's sent as a separate message.
func (s *statusTracker) DropText(text string) {
	text = strings.TrimSpace(text)
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
