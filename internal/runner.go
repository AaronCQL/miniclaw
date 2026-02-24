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

// stream-json parse structs
type streamEvent struct {
	Type      string         `json:"type"`
	Subtype   string         `json:"subtype"`
	SessionID string         `json:"session_id"`
	Result    string         `json:"result"`
	Message   *streamMessage `json:"message"`
}

type streamMessage struct {
	Content []streamContent `json:"content"`
}

type streamContent struct {
	Type  string                 `json:"type"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

func (r *AgentRunner) Run(ctx context.Context, input models.AgentInput, onToolUse func(toolName, label string)) (models.AgentOutput, error) {
	prompt := r.buildPrompt(input)

	args := []string{
		"--print",
		"--verbose", // required by Claude CLI when using stream-json with --print
		"--output-format", "stream-json",
		"--dangerously-skip-permissions",
	}

	sessionID := r.sessions.Get(input.ChatID)
	if sessionID != "" {
		log.Printf("[agent] chat=%d resuming session=%s", input.ChatID, sessionID)
		args = append(args, "--resume", sessionID)
	} else {
		log.Printf("[agent] chat=%d starting new session", input.ChatID)
	}

	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = r.config.AgentDir
	cmd.Stdin = strings.NewReader(prompt)
	cmd.Env = append(os.Environ(), fmt.Sprintf("MINICLAW_CHAT_ID=%d", input.ChatID))

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

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024) // 1MB max line buffer

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
			if onToolUse != nil && event.Message != nil {
				for _, block := range event.Message.Content {
					if block.Type == "tool_use" && block.Name != "" {
						onToolUse(block.Name, toolLabel(block.Name, block.Input))
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
		}
	}

	if err := cmd.Wait(); err != nil {
		log.Printf("[agent] chat=%d CLI error: %v stderr=%q", input.ChatID, err, stderr.String())
		return models.AgentOutput{Status: "error", Error: stderr.String()}, err
	}

	if resultSessionID != "" {
		r.sessions.Set(input.ChatID, resultSessionID)
	}

	log.Printf("[agent] chat=%d completed session=%s result_len=%d", input.ChatID, resultSessionID, len(result))
	return models.AgentOutput{
		Result: result,
		Status: "success",
	}, nil
}

func toolLabel(name string, input map[string]interface{}) string {
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
	case "Task":
		if d := getString("description"); d != "" {
			return html.EscapeString(d)
		}
	}
	return ""
}

func codeTag(s string) string {
	return "<code>" + html.EscapeString(s) + "</code>"
}

func (r *AgentRunner) buildPrompt(input models.AgentInput) string {
	if input.ReplyToContent != "" {
		return fmt.Sprintf("[Replying to %s: %s]\n\n%s", input.ReplyToSender, input.ReplyToContent, input.Prompt)
	}
	return input.Prompt
}
