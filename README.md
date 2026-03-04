# miniclaw

A minimal and easily hackable [OpenClaw](https://github.com/openclaw/openclaw) alternative.

## Philosophy

miniclaw is deliberately small, not just for simplicity, but because a small codebase means the agent can understand and modify its own source code at runtime.

The entire project is a handful of Go files with only three dependencies (a Telegram library, a cron parser, and a dotenv loader). Claude can read, edit, and rebuild miniclaw on the fly, adding features, fixing bugs, or adapting its own behaviour to whatever you need, all through a conversation on Telegram. No plugin system required when your agent *is* the plugin system.

Fork it, read it in one sitting, and make it yours.

## What it does

- Persistent sessions per chat, with reply context
- Scheduled tasks (cron, interval, one-shot) as JSON files
- File, image, and voice message support
- Built-in and extensible skills via slash commands
- Real-time status updates while the agent works

## Prerequisites

- Go 1.23+
- [Claude CLI](https://docs.anthropic.com/en/docs/claude-code) (installed and authenticated)
- A Telegram bot token from [@BotFather](https://t.me/BotFather)
- (Optional) A [Groq API key](https://console.groq.com/) for voice transcription

## Setup

```sh
git clone https://github.com/AaronCQL/miniclaw.git
cd miniclaw
claude
# then type: /setup
```

The `/setup` command walks you through prerequisites, configuration, and optionally sets up a background service (systemd on Linux, launchd on macOS).

## Customisation

- **`agent/preferences.md`**: your bot's name, personality, timezone, and any preferences you tell it to remember
- **`agent/CLAUDE.md`**: the system prompt that defines agent behaviour, sandbox rules, and message formatting

Edit these files to make the bot your own.

## Project structure

The repo has two main concerns: the Go application that wraps Claude CLI, and the agent context that shapes how Claude behaves.

- **`agent/`**: the agent's working directory, containing its system prompt (`CLAUDE.md`) and personality (`preferences.md`). This is where Claude runs from.
- **`.claude/skills/`**: slash command definitions (e.g. `/diff`, `/setup`, `/restart`). Each skill is a markdown file that Claude follows as instructions.
- **`cmd/`** and **`internal/`**: the Go application. Telegram polling, session management, task scheduling, and the Claude CLI runner.

At runtime, all state lives in `~/.miniclaw/`: the `.env` config, session data, scheduled tasks, and a scratch workspace for file operations.