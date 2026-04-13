package internal

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"miniclaw/internal/models"
)

type AgentRunner struct {
	config   Config
	sessions *SessionStore
}

func NewAgentRunner(cfg Config, sessions *SessionStore) *AgentRunner {
	return &AgentRunner{
		config:   cfg,
		sessions: sessions,
	}
}

type streamEvent struct {
	Type       string                       `json:"type"`
	Subtype    string                       `json:"subtype"`
	SessionID  string                       `json:"session_id"`
	Result     string                       `json:"result"`
	Message    *streamMessage               `json:"message"`
	ModelUsage map[string]models.ModelUsage `json:"modelUsage"`
}

type streamMessage struct {
	Content []streamContent
}

func (m *streamMessage) UnmarshalJSON(data []byte) error {
	var raw struct {
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	// content can be a string (text response) or array (tool use blocks).
	// We only need the array form; ignore strings silently.
	if len(raw.Content) > 0 && raw.Content[0] == '[' {
		return json.Unmarshal(raw.Content, &m.Content)
	}
	return nil
}

type streamContent struct {
	Type  string         `json:"type"`
	Text  string         `json:"text"`
	Name  string         `json:"name"`
	Input map[string]any `json:"input"`
}

func (r *AgentRunner) Run(ctx context.Context, input models.AgentInput, effort string, onToolUse func(toolName, label string), onText func(text string)) (models.AgentOutput, error) {
	prompt := r.buildPrompt(input)

	args := []string{
		"--print",
		"--verbose", // required by Claude CLI when using stream-json with --print
		"--output-format", "stream-json",
		"--dangerously-skip-permissions",
	}
	if effort != EffortDefault {
		args = append(args, "--effort", effort)
	}
	if threadPrompt := r.loadThreadPrompt(input.ChatID, input.ThreadID); threadPrompt != "" {
		args = append(args, "--append-system-prompt", threadPrompt)
	}

	if input.IsolatedSession {
		log.Printf("[agent] chat=%d thread=%d starting isolated session", input.ChatID, input.ThreadID)
	} else if sessionID := r.sessions.Get(input.ChatID, input.ThreadID); sessionID != "" {
		log.Printf("[agent] chat=%d thread=%d resuming session=%s", input.ChatID, input.ThreadID, sessionID)
		args = append(args, "--resume", sessionID)
	} else {
		log.Printf("[agent] chat=%d thread=%d starting new session", input.ChatID, input.ThreadID)
	}

	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = r.config.AgentDir
	cmd.Stdin = strings.NewReader(prompt)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("MINICLAW_CHAT_ID=%d", input.ChatID),
		fmt.Sprintf("MINICLAW_THREAD_ID=%d", input.ThreadID),
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return models.AgentOutput{Status: "error", Error: "failed to create stdout pipe"}, err
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		log.Printf("[agent] chat=%d CLI start error: %v", input.ChatID, err)
		return models.AgentOutput{Status: "error", Error: "failed to start CLI"}, err
	}

	var result string
	var resultSessionID string
	var resultModelUsage map[string]models.ModelUsage

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var event streamEvent
		if err := json.Unmarshal(line, &event); err != nil {
			log.Printf("[agent] chat=%d failed to parse stream line: %v", input.ChatID, err)
			continue
		}

		switch event.Type {
		case "system":
			if event.Subtype == "init" && event.SessionID != "" {
				resultSessionID = event.SessionID
			}

		case "assistant":
			if event.Message != nil && (onToolUse != nil || onText != nil) {
				for _, block := range event.Message.Content {
					if block.Type == "tool_use" && block.Name != "" && onToolUse != nil {
						onToolUse(block.Name, toolLabel(block.Name, block.Input))
					}
					if block.Type == "text" && block.Text != "" && onText != nil {
						onText(strings.TrimSpace(block.Text))
					}
				}
			}

		case "result":
			if event.Subtype == "success" {
				result = event.Result
			}
			if event.SessionID != "" {
				resultSessionID = event.SessionID
			}
			if event.ModelUsage != nil {
				resultModelUsage = event.ModelUsage
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		log.Printf("[agent] chat=%d thread=%d CLI error: %v stderr=%q", input.ChatID, input.ThreadID, err, stderr.String())
		return models.AgentOutput{Status: "error", Error: stderr.String()}, err
	}

	if resultSessionID != "" && !input.IsolatedSession {
		r.sessions.SetIfAbsent(input.ChatID, input.ThreadID, resultSessionID)
	}

	log.Printf("[agent] chat=%d thread=%d completed session=%s result_len=%d", input.ChatID, input.ThreadID, resultSessionID, len(result))
	return models.AgentOutput{
		Result:     result,
		Status:     "success",
		ModelUsage: resultModelUsage,
	}, nil
}

func toolLabel(name string, input map[string]any) string {
	getString := func(key string) string {
		if v, ok := input[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}

	switch name {
	case "Read", "Edit", "Write":
		if fp := getString("file_path"); fp != "" {
			return codeTag(filepath.Base(fp))
		}
	case "Bash":
		if cmd := getString("command"); cmd != "" {
			if i := strings.IndexByte(cmd, '\n'); i >= 0 {
				cmd = cmd[:i]
			}
			return codeTag(cmd)
		}
	case "Grep", "Glob":
		if p := getString("pattern"); p != "" {
			return codeTag(p)
		}
	case "WebSearch":
		if q := getString("query"); q != "" {
			return html.EscapeString(q)
		}
	case "WebFetch":
		if u := getString("url"); u != "" {
			if parsed, err := url.Parse(u); err == nil {
				return html.EscapeString(parsed.Hostname())
			}
		}
	case "Agent":
		if d := getString("description"); d != "" {
			return "<b>" + html.EscapeString(d) + "</b>"
		}
	case "Task":
		if d := getString("description"); d != "" {
			return html.EscapeString(d)
		}
	case "TodoWrite":
		if todos, ok := input["todos"].([]any); ok {
			for _, t := range todos {
				if todo, ok := t.(map[string]any); ok {
					if todo["status"] == "in_progress" {
						if c, ok := todo["content"].(string); ok {
							return "<b>" + html.EscapeString(c) + "</b>"
						}
					}
				}
			}
		}
	case "Skill":
		if s := getString("skill"); s != "" {
			return "Skill: /" + html.EscapeString(s)
		}
	case "EnterPlanMode":
		return "Plan mode"
	}

	// MCP tools: mcp__server__action -> "MCP: Server Action"
	if parts := strings.SplitN(name, "__", 3); len(parts) == 3 && parts[0] == "mcp" {
		return "MCP: " + titleCase(parts[1]) + " " + titleCase(parts[2])
	}

	return ""
}

func codeTag(s string) string {
	return "<code>" + html.EscapeString(s) + "</code>"
}

// titleCase converts "browser_navigate" to "Browser Navigate".
func titleCase(s string) string {
	words := strings.Split(s, "_")
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

func (r *AgentRunner) buildPrompt(input models.AgentInput) string {
	var parts []string

	now := time.Now()
	if tz := os.Getenv("MINICLAW_TIMEZONE"); tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			now = now.In(loc)
		}
	}
	parts = append(parts, fmt.Sprintf("[Current time: %s]", now.Format("2006-01-02 15:04 -07:00")))

	if input.ReplyToContent != "" {
		parts = append(parts, fmt.Sprintf("[Replying to %s: %s]", input.ReplyToSender, input.ReplyToContent))
	}

	if input.ReplyToFilePath != "" {
		parts = append(parts, fmt.Sprintf("[Replied-to message has file attached: %s - use the Read tool to view this file]", input.ReplyToFilePath))
	}

	if input.FilePath != "" {
		parts = append(parts, fmt.Sprintf("[File attached: %s - use the Read tool to view this file]", input.FilePath))
	}

	if input.Prompt != "" {
		parts = append(parts, input.Prompt)
	} else if input.FilePath != "" {
		parts = append(parts, "The user sent a file. Please view and describe or analyse it.")
	}

	return strings.Join(parts, "\n\n")
}

func (r *AgentRunner) loadThreadPrompt(chatID, threadID int64) string {
	var file string
	if threadID == 0 {
		file = fmt.Sprintf("%d.md", chatID)
	} else {
		file = fmt.Sprintf("%d_%d.md", chatID, threadID)
	}
	data, err := os.ReadFile(filepath.Join(r.config.DataDir, "prompts", file))
	if err != nil {
		return ""
	}
	prompt := strings.TrimSpace(string(data))
	if prompt == "" {
		return ""
	}
	return "<thread-system-prompt>\n" + prompt + "\n</thread-system-prompt>"
}
