---
name: commands
description: Register bot commands with Telegram so they appear in the command menu
allowed-tools: "Bash(curl *), Bash(ls *)"
---

# Register Telegram Bot Commands

Read all available skills and register them as Telegram bot commands via the `setMyCommands` API. This is idempotent. Running it multiple times just overwrites the command list.

## Step 1: Discover skills

List skill directories from both the repo and personal locations:

```bash
ls -1 ../.claude/skills/
ls -1 ~/.claude/skills/
```

For each directory found in either location, read its `SKILL.md` file using the Read tool. Extract the `name`, `description`, and `disable-model-invocation` fields from the YAML frontmatter. If a skill exists in both locations, the personal one (~/.claude) takes precedence. **Skip any skill that has `disable-model-invocation: true`**.

## Step 2: Build the command list

Combine the discovered skills with these hardcoded commands that don't have SKILL.md files:

- `chatid`: "Get your Telegram chat ID"
- `cancel`: "Cancel the current request"
- `compact`: "Compact conversation context to free up space"
- `effort`: "View or set the effort level (low, medium, high, max, default)"
- `logs`: "Cycle status updates: off, text only, verbose"
- `simplify`: "Review changed code for reuse, quality, and efficiency"
- `usage`: "Show context window usage and cumulative cost"
- `clear`: "Clear chat history and context window for this thread"

Telegram command descriptions have a 256-character limit. Truncate any descriptions that exceed this.

## Step 3: Register commands

Build a JSON array of `{"command": "name", "description": "desc"}` objects and call the Telegram API:

```bash
curl -s https://api.telegram.org/bot$TELEGRAM_BOT_TOKEN/setMyCommands \
  -H "Content-Type: application/json" \
  -d '{"commands": [...]}'
```

The response should contain `{"ok": true}`. If it doesn't, report the error.

## Step 4: Report

List all registered commands with their descriptions and confirm success.
