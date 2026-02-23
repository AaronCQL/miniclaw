---
name: setup
description: Interactive setup wizard for new Goclaw users who just forked the repo
disable-model-invocation: true
allowed-tools: "Read, Bash(go *), Bash(which *), Bash(claude *), Bash(mkdir *), Bash(ls *)"
---

# Goclaw Setup Wizard

You are helping a new user set up Goclaw after forking the repo. Walk through each step below **in order**. Check prerequisites first, then guide the user through configuration.

## Step 1: Check prerequisites

Run these checks silently and report the results as a checklist:

1. **Go** — run `which go` and `go version`. Require Go 1.23+.
2. **Claude CLI** — run `which claude` and `claude --version`. This is required for the agent runtime. Assume the user is authenticated since they are already using Claude to set it up - do not run any Claude commands yourself as it will fail.

If any prerequisite is missing, tell the user what to fix and stop. Do not continue until all checks pass.

## Step 2: Install Go dependencies

Run `go mod tidy` from the repo root to fetch all dependencies. Report success or failure.

## Step 3: Install binary

Run `go install ./cmd/goclaw/` to compile and install the `goclaw` binary to the user's `$GOPATH/bin`. Report success or failure.

## Step 4: Create runtime directories

Create `~/.goclaw/` and its subdirectories by running:

```
mkdir -p ~/.goclaw/{data/tasks,workspace}
```

Report that the runtime directory structure has been created.

## Step 5: Telegram bot token

Ask the user for their Telegram bot token. Tell them:

- Create a bot via [@BotFather](https://t.me/BotFather) on Telegram
- Use the `/newbot` command and follow the prompts
- Copy the token BotFather gives you

Once they provide the token, hold onto it for Step 7.

## Step 6: Agent directory

Determine the absolute path to the `agent/` directory in the current repo by running `ls` on it. Hold onto this path for Step 7 as the `GOCLAW_AGENT_DIR` value. This tells the bot where to find its CLAUDE.md and preferences.md files, so it can be run from any directory.

## Step 7: Allowed chat IDs

Ask the user for their allowed Telegram chat IDs (comma-separated). Tell them:

- After setting up, they can send `/chatid` to the bot from any chat to get the ID
- For now, they can leave this empty and add it later
- Group chats have negative IDs (e.g. `-1001234567890`)
- Private chats have positive IDs

Hold onto the value for Step 8.

## Step 8: Write .env file

Write `~/.goclaw/.env` with the collected values:

```
TELEGRAM_BOT_TOKEN=<their token>
ALLOWED_CHAT_IDS=<their chat IDs, or empty>
GOCLAW_AGENT_DIR=<absolute path to agent/ from Step 6>
```

Use the Bash tool to write this file with `0600` permissions. Do NOT use the Write tool (the path is outside the project).

## Step 9: Done

Print a summary:

```
Setup complete! To run Goclaw:

  goclaw

To find your chat ID, send /chatid to your bot on Telegram,
then add it to ~/.goclaw/.env as ALLOWED_CHAT_IDS.
```

If they left ALLOWED_CHAT_IDS empty, remind them to:

1. Start the bot without an allowlist (it will respond to anyone)
2. Send `/chatid` to the bot
3. Add the ID to `~/.goclaw/.env`
4. Restart the bot

## Rules

- Be concise and friendly
- Do NOT proceed past a failed step — fix it first
- Do NOT print raw commands unless the user asks to see them
- Do NOT modify any repo files — only create `~/.goclaw/.env`
