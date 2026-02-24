# miniclaw

A minimal [Claude CLI](https://docs.anthropic.com/en/docs/claude-code) agent wrapper for Telegram.

## Philosophy

miniclaw is deliberately small. The entire codebase fits in a single sitting of reading — no frameworks, no plugin systems, no abstractions you need to trace through. Fork it, read it, make it yours. Inspired by [nanoclaw](https://github.com/qwibitai/nanoclaw) and [picoclaw](https://github.com/sipeed/picoclaw).

## What it does

- **Session persistence** — each chat maintains its own Claude conversation across restarts
- **Scheduled tasks** — cron, interval, and one-shot tasks stored as simple JSON files
- **Real-time status** — shows what tools Claude is using while it works
- **Reply chains** — replies to bot messages include prior context
- **Per-chat concurrency** — one agent per chat, no race conditions

## Prerequisites

- Go 1.23+
- [Claude CLI](https://docs.anthropic.com/en/docs/claude-code) (installed and authenticated)
- A Telegram bot token from [@BotFather](https://t.me/BotFather)

## Setup

```sh
git clone https://github.com/AaronCQL/miniclaw.git
cd miniclaw
claude
# then type: /setup
```

The `/setup` command walks you through prerequisites, configuration, and optionally sets up a systemd service.

## Customisation

- **`agent/preferences.md`** — your bot's name, personality, timezone, and any preferences you tell it to remember
- **`agent/CLAUDE.md`** — the system prompt that defines agent behaviour, sandbox rules, and message formatting

Edit these files to make the bot your own.

## Project structure

```
Repository                      Runtime (~/.miniclaw/)
├── agent/                      ├── .env
│   ├── CLAUDE.md               ├── data/
│   └── preferences.md          │   ├── sessions.json
├── cmd/miniclaw/               │   └── tasks/
│   └── main.go                 │       └── *.json
├── internal/                   └── workspace/
│   ├── app.go
│   ├── config.go
│   ├── models/
│   ├── runner.go
│   ├── scheduler.go
│   ├── session.go
│   ├── status.go
│   └── telegram.go
└── go.mod
```

## How it works

The bot long-polls Telegram for messages, runs each one through a Claude CLI subprocess, and streams tool usage back in real time as status updates. A background scheduler periodically checks for and executes due tasks.
