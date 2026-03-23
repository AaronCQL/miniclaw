<!-- omit from toc -->
# miniclaw

A minimal Telegram agent powered by [Claude Code](https://docs.anthropic.com/en/docs/claude-code), designed to be self-modifiable.

> [!WARNING]
> miniclaw is pre-v1. Breaking changes may occur on minor version bumps.

- [Why miniclaw?](#why-miniclaw)
- [Quick Start](#quick-start)
  - [Agentic Setup (Recommended)](#agentic-setup-recommended)
  - [Manual Setup](#manual-setup)
- [Features](#features)
  - [Persistent Memory](#persistent-memory)
  - [Extensible Skills](#extensible-skills)
  - [AI-Managed Tasks](#ai-managed-tasks)
  - [Rich Media](#rich-media)
- [Project Structure](#project-structure)
- [Changelog](#changelog)

## Why miniclaw?

1. **Official Harness**: Powered directly by [Claude Code](https://docs.anthropic.com/en/docs/claude-code). Get Anthropic's state-of-the-art tool-use, codebase awareness, and memory management out of the box, with zero maintenance and automatic updates.
2. **Telegram First**: The best UI for a personal agent. Enjoy native support for threads, file sharing, voice messages, and real-time work status.
3. **Self-Modifying**: The codebase is small enough for Claude to understand in one go. Want a new feature? Simply ask the agent to implement it for you.

## Quick Start

### Agentic Setup (Recommended)

Launch Claude CLI and enter the following prompt:

```md
Fork and clone `https://github.com/AaronCQL/miniclaw`, cd into `miniclaw`, then read and follow the setup instructions in `.claude/skills/setup/SKILL.md`.
```

### Manual Setup

```sh
# 1) First, fork and clone miniclaw to your desired location
gh repo fork AaronCQL/miniclaw --clone

# 2) Then, change into the miniclaw directory
cd miniclaw

# 3) Launch Claude CLI
claude

# 4) Finally, type: /setup to use the setup wizard
```

Once your bot is running, use `/commands` on Telegram to sync the agent's skills with Telegram and to see all available commands.

## Features

- [🧠 **Persistent Memory**](#persistent-memory): Context-aware sessions per Telegram chat or thread.
- [🤹 **Extensible Skills**](#extensible-skills): Use built-in skills as Telegram slash commands, or ask your agent to build new ones.
- [📋 **AI-Managed Tasks**](#ai-managed-tasks): Schedule tasks or reminders simply by telling the agent what you need.
- [📽️ **Rich Media**](#rich-media): Full support for images, documents, and voice messages.

### Persistent Memory

Each Telegram thread runs as a separate Claude session, but all sessions share the same [Claude Code auto memory](https://docs.anthropic.com/en/docs/claude-code/memory). This gives you topic isolation per thread with cross-thread recall.

miniclaw uses Claude Code's built-in file-based memory system: a `MEMORY.md` index (loaded every message) pointing to topic files (loaded on demand). The `/remember` skill scans conversation transcripts across all threads and writes structured memories into these files, organised by category: decisions, entities, cases, patterns, events, and topics.

This is intentionally simple. There's no vector database, no embedding model, no background worker. It builds on what Anthropic already provides rather than replacing it, and the memory files are plain markdown you can read and edit yourself.

**Alternatives to consider**:

- [claude-mem](https://github.com/thedotmack/claude-mem): captures every tool call, compresses via AI, injects at session start. Powerful but requires a persistent worker service and AI processing on every tool use.
- [OpenViking](https://github.com/volcengine/OpenViking): three-tier progressive loading with semantic search. Impressive architecture but requires embedding models, a running server, and heavy dependencies (Python + Go + C++).

Both are worth considering if you need semantic search or more granular recall. miniclaw's approach trades sophistication for minimal operational overhead.

### Extensible Skills

Skills are slash commands the agent follows as expert instructions:

- Use `/commands` on Telegram to sync them
- Some skills accept arguments (eg. `/remember 7d`, `/voice all`)

| Skill | Description | Recommended Schedule |
|-------|-------------|---------------------|
| `/setup` | Interactive first-time setup wizard | One-time |
| `/migrate` | Migrate your main session context into the current thread | One-time |
| `/remember` | Summarise conversations into persistent memory | Daily |
| `/voice` | Update typing style guide from chat history | Weekly |
| `/cancel` | Cancel the current request | On demand |
| `/chatid` | Show the current chat and thread IDs | On demand |
| `/commands` | Register bot commands with Telegram | On demand |
| `/restart` | Rebuild miniclaw and restart the background service | On demand |
| `/review` | Review git diff, suggest commits | On demand |
| `/logs` | Cycle status updates: off, text only, verbose | On demand |
| `/usage` | Show context window usage and cumulative cost | On demand |
| `/transcribe` | Transcribe voice messages via Groq Whisper | Auto (on voice message) |

### AI-Managed Tasks

Schedule tasks by simply telling the agent what you need and when: eg. "remind me to check emails every morning at 9am" or "give me a weekly summary of our conversation every Sunday".

Three schedule types are supported: one-shot (run once then self-delete), cron (standard 5-field expressions), and interval (fixed-duration repeat like `24h` or `30m`). Tasks can optionally have an expiry, after which they are automatically cleaned up.

Under the hood, tasks are plain JSON files in `~/.miniclaw/data/tasks/`. The agent creates, pauses, and deletes them via normal file operations. A lightweight scheduler polls every 10 seconds and runs any due tasks by invoking the agent with the task's prompt, sending results back to the originating Telegram chat and thread.

There are no external task queues, databases, or background workers beyond a single scheduler goroutine and normal file operations.

### Rich Media

When you send images, documents, videos, audio files, or voice messages to the bot, they are automatically downloaded to `~/.miniclaw/workspace/files/` and passed to the agent as a file path. The agent can then read images visually, parse documents, or process files however it sees fit.

When you receive files, it works the other way: the agent writes an `outbox.json` manifest to `~/.miniclaw/` with an absolute path and optional caption for each file, and the bot sends them as Telegram documents. Maximum file size is 50 MB per Telegram's bot API limit.

Telegram voice notes (`.oga`/`.ogg`) are downloaded like any other file, but if a [Groq API key](https://console.groq.com/) is configured, the `/transcribe` skill automatically transcribes them using Whisper and the agent responds to the transcribed text as if you had typed it. Groq's free tier allows 2,000 requests and 8 hours of audio per day.

## Project Structure

The repo has two main concerns: the Go application that wraps Claude CLI, and the agent context that shapes how Claude behaves.

- **`agent/`**: the agent's working directory, containing its system prompt (`CLAUDE.md`) and on-demand reference docs. This is where Claude runs from.
- **`.claude/skills/`**: slash command definitions (eg. `/review`, `/remember`, `/setup`). Each skill is a directory containing a `SKILL.md` file that the agent follows as expert instructions.
- **`cmd/`** and **`internal/`**: the Go application. Telegram polling, session management, task scheduling, and the Claude CLI runner.

At runtime, all state lives in `~/.miniclaw/`: the `.env` config, session data, scheduled tasks, and a scratch workspace for file operations.

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for a list of changes across releases.
