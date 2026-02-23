package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"goclaw/internal/models"
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

func (r *AgentRunner) Run(ctx context.Context, input models.AgentInput) (models.AgentOutput, error) {
	prompt := r.buildPrompt(input)

	args := []string{
		"--print",
		"--output-format", "json",
		"--max-turns", "10",
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
	cmd.Env = append(os.Environ(), fmt.Sprintf("GOCLAW_CHAT_ID=%d", input.ChatID))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		log.Printf("[agent] chat=%d CLI error: %v stderr=%q", input.ChatID, err, stderr.String())
		return models.AgentOutput{Status: "error", Error: stderr.String()}, err
	}

	var cliResponse struct {
		Result    string `json:"result"`
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &cliResponse); err != nil {
		log.Printf("[agent] chat=%d failed to parse CLI JSON: %v", input.ChatID, err)
		return models.AgentOutput{Status: "error", Error: "failed to parse CLI output: " + stdout.String()}, err
	}

	if cliResponse.SessionID != "" {
		r.sessions.Set(input.ChatID, cliResponse.SessionID)
	}

	log.Printf("[agent] chat=%d completed session=%s result_len=%d", input.ChatID, cliResponse.SessionID, len(cliResponse.Result))
	return models.AgentOutput{
		Result: cliResponse.Result,
		Status: "success",
	}, nil
}

func (r *AgentRunner) buildPrompt(input models.AgentInput) string {
	if input.ReplyToContent != "" {
		return fmt.Sprintf("[Replying to %s: %s]\n\n%s", input.ReplyToSender, input.ReplyToContent, input.Prompt)
	}
	return input.Prompt
}
