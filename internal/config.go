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

func HomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("cannot resolve home directory: %v", err)
	}

	dir := filepath.Join(home, ".miniclaw")
	for _, sub := range []string{"", "data", "data/tasks", "data/prompts", "workspace"} {
		p := filepath.Join(dir, sub)
		if err := os.MkdirAll(p, 0755); err != nil {
			log.Fatalf("cannot create directory %s: %v", p, err)
		}
	}

	// Load .env early so MINICLAW_AGENT_DIR (and other vars) are available before LoadConfig.
	_ = godotenv.Load(filepath.Join(dir, ".env"))

	return dir
}

func AgentDir() string {
	dir := os.Getenv("MINICLAW_AGENT_DIR")
	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("cannot get working directory: %v", err)
		}
		dir = filepath.Join(cwd, "agent")
	}

	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		log.Fatalf("agent directory not found at %s - set MINICLAW_AGENT_DIR or run from the repo root", dir)
	}

	return dir
}

func LoadConfig(homeDir string, agentDir string) Config {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is required")
	}

	var allowedIDs []int64
	if raw := os.Getenv("ALLOWED_CHAT_IDS"); raw != "" {
		for s := range strings.SplitSeq(raw, ",") {
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
		SchedulerInterval: 10 * time.Second,
	}
}
