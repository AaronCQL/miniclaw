package internal

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	HomeDir           string
	AgentDir          string
	DataDir           string
	WorkspaceDir      string
	TelegramToken     string
	AllowedChatIDs    []int64
	SchedulerInterval time.Duration
}

// HomeDir resolves ~/.goclaw/ and creates it (and data/, data/tasks/, workspace/ subdirectories) if they don't exist.
// Also loads ~/.goclaw/.env early so env vars are available to subsequent init functions.
func HomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("cannot resolve home directory: %v", err)
	}

	dir := filepath.Join(home, ".goclaw")
	for _, sub := range []string{"", "data", "data/tasks", "workspace"} {
		p := filepath.Join(dir, sub)
		if err := os.MkdirAll(p, 0755); err != nil {
			log.Fatalf("cannot create directory %s: %v", p, err)
		}
	}

	// Load .env early so GOCLAW_AGENT_DIR (and other vars) are available before LoadConfig.
	_ = godotenv.Load(filepath.Join(dir, ".env"))

	return dir
}

// AgentDir resolves the agent directory. Uses GOCLAW_AGENT_DIR env var if set,
// otherwise falls back to ./agent/ relative to CWD.
// Seeds preferences.md with default template if it doesn't exist.
func AgentDir() string {
	dir := os.Getenv("GOCLAW_AGENT_DIR")
	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("cannot get working directory: %v", err)
		}
		dir = filepath.Join(cwd, "agent")
	}

	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		log.Fatalf("agent directory not found at %s — set GOCLAW_AGENT_DIR or run from the repo root", dir)
	}

	prefsPath := filepath.Join(dir, "preferences.md")
	if _, err := os.Stat(prefsPath); os.IsNotExist(err) {
		defaultPrefs := `# Preferences

## Identity

- Name: Assistant
- Personality: Helpful, concise, friendly

## Settings

- Timezone: UTC

## User Preferences

(none yet — the user will tell you what to remember)
`
		if err := os.WriteFile(prefsPath, []byte(defaultPrefs), 0644); err != nil {
			log.Fatalf("cannot seed preferences.md: %v", err)
		}
	}

	return dir
}

// LoadConfig loads config from env vars (already loaded via HomeDir).
func LoadConfig(homeDir string, agentDir string) Config {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is required")
	}

	var allowedIDs []int64
	if raw := os.Getenv("ALLOWED_CHAT_IDS"); raw != "" {
		for _, s := range strings.Split(raw, ",") {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			id, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				log.Fatalf("invalid chat ID %q: %v", s, err)
			}
			allowedIDs = append(allowedIDs, id)
		}
	}

	return Config{
		HomeDir:           homeDir,
		AgentDir:          agentDir,
		DataDir:           filepath.Join(homeDir, "data"),
		WorkspaceDir:      filepath.Join(homeDir, "workspace"),
		TelegramToken:     token,
		AllowedChatIDs:    allowedIDs,
		SchedulerInterval: time.Minute,
	}
}
