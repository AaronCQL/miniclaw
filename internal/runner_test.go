package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"miniclaw/internal/models"
)

func TestToolLabel(t *testing.T) {
	tests := []struct {
		name  string
		tool  string
		input map[string]any
		want  string
	}{
		{
			name: "Read with file_path",
			tool: "Read", input: map[string]any{"file_path": "/home/user/project/main.go"},
			want: "<code>main.go</code>",
		},
		{
			name: "Edit with file_path",
			tool: "Edit", input: map[string]any{"file_path": "/tmp/config.yaml"},
			want: "<code>config.yaml</code>",
		},
		{
			name: "Write with file_path",
			tool: "Write", input: map[string]any{"file_path": "/a/b/c.txt"},
			want: "<code>c.txt</code>",
		},
		{
			name: "Read without file_path",
			tool: "Read", input: map[string]any{},
			want: "",
		},
		{
			name: "Bash single-line command",
			tool: "Bash", input: map[string]any{"command": "go build ./..."},
			want: "<code>go build ./...</code>",
		},
		{
			name: "Bash multiline command uses first line",
			tool: "Bash", input: map[string]any{"command": "echo hello\necho world"},
			want: "<code>echo hello</code>",
		},
		{
			name: "Bash empty command",
			tool: "Bash", input: map[string]any{"command": ""},
			want: "",
		},
		{
			name: "Grep with pattern",
			tool: "Grep", input: map[string]any{"pattern": "TODO"},
			want: "<code>TODO</code>",
		},
		{
			name: "Glob with pattern",
			tool: "Glob", input: map[string]any{"pattern": "**/*.go"},
			want: "<code>**/*.go</code>",
		},
		{
			name: "WebSearch with query",
			tool: "WebSearch", input: map[string]any{"query": "golang testing"},
			want: "golang testing",
		},
		{
			name: "WebSearch escapes HTML",
			tool: "WebSearch", input: map[string]any{"query": "a <b> & c"},
			want: "a &lt;b&gt; &amp; c",
		},
		{
			name: "WebFetch with URL",
			tool: "WebFetch", input: map[string]any{"url": "https://example.com/path?q=1"},
			want: "example.com",
		},
		{
			name: "WebFetch with invalid URL",
			tool: "WebFetch", input: map[string]any{"url": "://bad"},
			want: "",
		},
		{
			name: "Task with description",
			tool: "Task", input: map[string]any{"description": "Run linter"},
			want: "Run linter",
		},
		{
			name: "TodoWrite in_progress item",
			tool: "TodoWrite", input: map[string]any{
				"todos": []any{
					map[string]any{"status": "pending", "content": "first"},
					map[string]any{"status": "in_progress", "content": "doing this"},
					map[string]any{"status": "completed", "content": "done"},
				},
			},
			want: "<b>doing this</b>",
		},
		{
			name: "TodoWrite no in_progress item",
			tool: "TodoWrite", input: map[string]any{
				"todos": []any{
					map[string]any{"status": "pending", "content": "first"},
				},
			},
			want: "",
		},
		{
			name: "TodoWrite empty todos",
			tool: "TodoWrite", input: map[string]any{"todos": []any{}},
			want: "",
		},
		{
			name: "Skill with name",
			tool: "Skill", input: map[string]any{"skill": "release"},
			want: "Skill: /release",
		},
		{
			name: "EnterPlanMode",
			tool: "EnterPlanMode", input: map[string]any{},
			want: "Plan mode",
		},
		{
			name: "MCP tool",
			tool: "mcp__playwright__browser_navigate", input: map[string]any{},
			want: "MCP: Playwright Browser Navigate",
		},
		{
			name: "MCP tool with underscores",
			tool: "mcp__some_server__do_something_cool", input: map[string]any{},
			want: "MCP: Some Server Do Something Cool",
		},
		{
			name: "unknown tool",
			tool: "SomeNewTool", input: map[string]any{},
			want: "",
		},
		{
			name: "nil input",
			tool: "Read", input: nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toolLabel(tt.tool, tt.input); got != tt.want {
				t.Errorf("toolLabel(%q, %v) = %q, want %q", tt.tool, tt.input, got, tt.want)
			}
		})
	}
}

func TestBuildPrompt(t *testing.T) {
	r := &AgentRunner{}

	// The timestamp prefix is always injected; build the expected prefix dynamically.
	now := time.Now()
	if tz := os.Getenv("MINICLAW_TIMEZONE"); tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			now = now.In(loc)
		}
	}
	ts := fmt.Sprintf("[Current time: %s]", now.Format("2006-01-02 15:04 -07:00"))

	tests := []struct {
		name  string
		input models.AgentInput
		want  string
	}{
		{
			name:  "simple prompt",
			input: models.AgentInput{Prompt: "hello"},
			want:  ts + "\n\nhello",
		},
		{
			name:  "prompt with file",
			input: models.AgentInput{Prompt: "check this", FilePath: "/tmp/photo.jpg"},
			want:  ts + "\n\n[File attached: /tmp/photo.jpg - use the Read tool to view this file]\n\ncheck this",
		},
		{
			name:  "file only no prompt",
			input: models.AgentInput{FilePath: "/tmp/doc.pdf"},
			want:  ts + "\n\n[File attached: /tmp/doc.pdf - use the Read tool to view this file]\n\nThe user sent a file. Please view and describe or analyse it.",
		},
		{
			name: "reply context",
			input: models.AgentInput{
				Prompt:         "what about this?",
				ReplyToSender:  "Alice",
				ReplyToContent: "some earlier message",
			},
			want: ts + "\n\n[Replying to Alice: some earlier message]\n\nwhat about this?",
		},
		{
			name: "reply with file attachment on replied-to message",
			input: models.AgentInput{
				Prompt:          "see the file",
				ReplyToSender:   "Bob",
				ReplyToContent:  "here's the file",
				ReplyToFilePath: "/tmp/reply.png",
			},
			want: ts + "\n\n[Replying to Bob: here's the file]\n\n[Replied-to message has file attached: /tmp/reply.png - use the Read tool to view this file]\n\nsee the file",
		},
		{
			name: "all fields populated",
			input: models.AgentInput{
				Prompt:          "do something",
				FilePath:        "/tmp/my.txt",
				ReplyToSender:   "Eve",
				ReplyToContent:  "original",
				ReplyToFilePath: "/tmp/orig.txt",
			},
			want: ts + "\n\n[Replying to Eve: original]\n\n[Replied-to message has file attached: /tmp/orig.txt - use the Read tool to view this file]\n\n[File attached: /tmp/my.txt - use the Read tool to view this file]\n\ndo something",
		},
		{
			name:  "empty input",
			input: models.AgentInput{},
			want:  ts,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.buildPrompt(tt.input); got != tt.want {
				t.Errorf("buildPrompt() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStreamMessageUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		json      string
		wantLen   int
		wantErr   bool
		wantFirst string // type of first content block, if any
	}{
		{
			name:      "array content with tool_use",
			json:      `{"content": [{"type": "tool_use", "name": "Read", "input": {"file_path": "/tmp/x"}}]}`,
			wantLen:   1,
			wantFirst: "tool_use",
		},
		{
			name:    "string content is ignored",
			json:    `{"content": "some text response"}`,
			wantLen: 0,
		},
		{
			name:    "empty content",
			json:    `{"content": ""}`,
			wantLen: 0,
		},
		{
			name:    "missing content field",
			json:    `{}`,
			wantLen: 0,
		},
		{
			name:    "null content",
			json:    `{"content": null}`,
			wantLen: 0,
		},
		{
			name:    "multiple content blocks",
			json:    `{"content": [{"type": "text", "name": ""}, {"type": "tool_use", "name": "Bash"}]}`,
			wantLen: 2,
		},
		{
			name:    "invalid JSON",
			json:    `not json`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m streamMessage
			err := json.Unmarshal([]byte(tt.json), &m)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(m.Content) != tt.wantLen {
				t.Errorf("got %d content blocks, want %d", len(m.Content), tt.wantLen)
			}
			if tt.wantFirst != "" && len(m.Content) > 0 {
				if m.Content[0].Type != tt.wantFirst {
					t.Errorf("first block type = %q, want %q", m.Content[0].Type, tt.wantFirst)
				}
			}
		})
	}
}
