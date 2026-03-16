# miniclaw

A minimal Telegram agent powered by [Claude Code](https://docs.anthropic.com/en/docs/claude-code), designed to be self-modifiable.

> **Note:** miniclaw is pre-v1. Breaking changes may occur on minor version bumps.

## Why miniclaw?

1. **Official Engine**: Powered directly by [Claude Code](https://docs.anthropic.com/en/docs/claude-code). Get Anthropic's state-of-the-art tool-use, codebase awareness, and memory management out of the box, with zero maintenance and automatic updates.
2. **Telegram First**: The best UI for a personal agent. Enjoy native support for threads, file sharing, voice messages, and real-time work status.
3. **Self-Modifying**: The codebase is small enough for Claude to understand in one go. Want a new feature? Simply ask the agent to implement it for you.

## Features

- **Persistent Memory**: Context-aware sessions per Telegram chat or thread.
- **AI-Managed Tasks**: Schedule cron jobs or one-shot reminders simply by telling the agent what you need.
- **Rich Media**: Full support for images, documents, and voice messages.
- **Extensible Skills**: Use built-in skills as Telegram slash commands, or ask your agent to build new ones.

## Prerequisites

- Go 1.23+
- [Claude CLI](https://docs.anthropic.com/en/docs/claude-code) (installed and authenticated)
- A Telegram bot token from [@BotFather](https://t.me/BotFather)
- (Optional) A [Groq API key](https://console.groq.com/) for voice transcription

## Setup

**AI-Native Installation**: use the `/setup` command within Claude CLI to walk you through prerequisites, configuration, and optionally set up a persistent background service (`systemd`/`launchd`).

```sh
# 1) First clone miniclaw to your desired location
git clone https://github.com/AaronCQL/miniclaw.git

# 2) Then, change into the miniclaw directory
cd miniclaw

# 3) Launch Claude
claude

# 4) Finally, type: /setup
```

Once your bot is running, use `/commands` on Telegram to sync the agent's skills with Telegram and to see all available commands.

## Skills

Skills are slash commands the agent follows as expert instructions. Use `/commands` on Telegram to sync them.

| Skill | Description | Recommended Schedule |
|-------|-------------|---------------------|
| `/review` | Review git diff, suggest commits | Twice daily |
| `/remember` | Summarise conversations into cross-thread memory | Daily |
| `/voice` | Update typing style guide from chat history | Weekly |
| `/tweets` | Suggest tweet ideas from recent conversations | Daily |
| `/compact` | Compact conversation context | Daily |
| `/transcribe` | Transcribe voice messages via Groq Whisper | Auto (on voice message) |
| `/setup` | Interactive first-time setup wizard | One-time |
| `/commands` | Register bot commands with Telegram | One-time |
| `/release` | Create a semver release with changelog and GitHub release | On demand |
| `/migrate` | Migrate session context into current thread | On demand |

Skills accept arguments where noted (e.g. `/remember 7d`, `/voice all`). Scheduling is done by telling the agent to create a task (e.g. "run /remember every day at 3:45am").

## Customisation

- **`agent/CLAUDE.md`**: the system prompt that defines agent behaviour, sandbox rules, and message formatting

Ask your agent to edit this file to make it your own.

## Project structure

The repo has two main concerns: the Go application that wraps Claude CLI, and the agent context that shapes how Claude behaves.

- **`agent/`**: the agent's working directory, containing its system prompt (`CLAUDE.md`) and on-demand reference docs. This is where Claude runs from.
- **`.claude/skills/`**: slash command definitions (e.g. `/review`, `/remember`, `/setup`). Each skill is a directory containing a `SKILL.md` file that the agent follows as expert instructions.
- **`cmd/`** and **`internal/`**: the Go application. Telegram polling, session management, task scheduling, and the Claude CLI runner.

At runtime, all state lives in `~/.miniclaw/`: the `.env` config, session data, scheduled tasks, and a scratch workspace for file operations.
