---
name: setup
description: Interactive setup wizard for new miniclaw users who just forked the repo
disable-model-invocation: true
allowed-tools: "Read, Edit, Bash(go *), Bash(which *), Bash(claude *), Bash(mkdir *), Bash(ls *), Bash(cat *), Bash(chmod *), Bash(systemctl *), Bash(loginctl *)"
---

# miniclaw Setup Wizard

You are helping a new user set up miniclaw after forking the repo. Walk through each step below **in order**. Check prerequisites first, then guide the user through configuration.

**Idempotency rule:** This wizard MUST be safe to run multiple times. Before creating or writing any file, check if it already exists and whether its contents are correct. Never overwrite existing valid configuration. Only prompt the user for values that are missing or empty.

## Step 1: Check prerequisites

Run these checks silently and report the results as a checklist:

1. **Go** — run `which go` and `go version`. Require Go 1.23+.
2. **Claude CLI** — run `which claude` and `claude --version`. This is required for the agent runtime. Assume the user is authenticated since they are already using Claude to set it up — do not run any Claude commands yourself as it will fail.

If any prerequisite is missing, tell the user what to fix and stop. Do not continue until all checks pass.

## Step 2: Install Go dependencies

Run `go mod tidy` from the repo root to fetch all dependencies. Report success or failure.

## Step 3: Install binary

Run `go install ./cmd/miniclaw/` to compile and install the `miniclaw` binary to the user's `$GOPATH/bin`. Report success or failure.

## Step 4: Create runtime directories

Run `mkdir -p ~/.miniclaw/{data/tasks,workspace}` — this is already idempotent. Report that the runtime directory structure is ready.

## Step 5: Personalise agent

Read `agent/preferences.md` and show the user the current Name and Timezone values. Ask if they want to change either one. Only edit the file if they request a change.

## Step 6: Read existing .env (if any)

Before asking for configuration values, check if `~/.miniclaw/.env` already exists by reading it with `cat ~/.miniclaw/.env 2>/dev/null`. Parse out the current values of:

- `TELEGRAM_BOT_TOKEN`
- `ALLOWED_CHAT_IDS`
- `MINICLAW_AGENT_DIR`

Hold onto these values. Steps 7–9 will only prompt for values that are missing or empty.

## Step 7: Telegram bot token

If `TELEGRAM_BOT_TOKEN` already has a non-empty value in the existing .env, report that a token is already configured (show a masked version, e.g. `714...XYz`) and skip this step.

Otherwise, ask the user for their Telegram bot token. Tell them:

- Create a bot via [@BotFather](https://t.me/BotFather) on Telegram
- Use the `/newbot` command and follow the prompts
- Copy the token BotFather gives you

The user may also choose to skip and add it later. Hold onto the value for Step 10.

## Step 8: Agent directory

Determine the absolute path to the `agent/` directory in the current repo by running `ls` on it. This is the `MINICLAW_AGENT_DIR` value.

If the existing .env already has a correct `MINICLAW_AGENT_DIR` that matches this path, skip silently. Otherwise, hold onto the new value for Step 10.

## Step 9: Allowed chat IDs

If `ALLOWED_CHAT_IDS` already has a non-empty value in the existing .env, report the current value and ask if they want to change it. If they don't, skip.

Otherwise, ask the user for their allowed Telegram chat IDs (comma-separated). Tell them:

- After setting up, they can send `/chatid` to the bot from any chat to get the ID
- For now, they can leave this empty and add it later
- Group chats have negative IDs (e.g. `-1001234567890`)
- Private chats have positive IDs

Hold onto the value for Step 10.

## Step 10: Write .env file

If `~/.miniclaw/.env` already exists and all three values (`TELEGRAM_BOT_TOKEN`, `ALLOWED_CHAT_IDS`, `MINICLAW_AGENT_DIR`) are correct, report that the .env file is already up to date and skip writing.

Otherwise, write `~/.miniclaw/.env` with the merged values (existing values for fields that didn't change, new values for fields that did):

```
TELEGRAM_BOT_TOKEN=<token>
ALLOWED_CHAT_IDS=<chat IDs>
MINICLAW_AGENT_DIR=<absolute path to agent/>
```

Use the Bash tool to write this file with `0600` permissions. Do NOT use the Write tool (the path is outside the project).

## Step 11: Systemd service (optional)

First, check if `~/.config/systemd/user/miniclaw.service` already exists. If it does, read its contents and verify the `ExecStart` path is correct (matches `which miniclaw`).

- If the service file exists and is correct, report that systemd is already configured. Check if the service is enabled (`systemctl --user is-enabled miniclaw`) and skip to asking if they want to start/restart it.
- If the service file exists but `ExecStart` is wrong, tell the user and offer to update it.
- If the service file does not exist, ask the user if they want to set up systemd.

If they decline systemd, skip to Step 12.

To set up or update the service:

1. Determine the absolute path to the `miniclaw` binary by running `which miniclaw` or falling back to `ls ~/go/bin/miniclaw`.
2. Run `mkdir -p ~/.config/systemd/user` (idempotent).
3. Write `~/.config/systemd/user/miniclaw.service` via the Bash tool with the following content:

```ini
[Unit]
Description=miniclaw Telegram Bot
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=%h/.miniclaw/.env
ExecStart=<absolute path to miniclaw binary>
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
```

4. Reload the systemd user daemon: `systemctl --user daemon-reload`
5. Enable the service so it starts on login: `systemctl --user enable miniclaw`
6. Enable lingering so it runs even when the user is not logged in: `loginctl enable-linger`
7. Ask if they want to start it now. If yes: `systemctl --user start miniclaw` and confirm it's running with `systemctl --user status miniclaw`.

## Step 12: Done

Print a summary:

```
Setup complete!
```

If they set up systemd, add:

```
miniclaw is running as a systemd user service.

  systemctl --user status miniclaw   — check status
  systemctl --user restart miniclaw  — restart after config changes
  journalctl --user -u miniclaw -f   — follow logs
```

If they skipped systemd, add:

```
To run miniclaw:

  miniclaw
```

In both cases, add:

```
To find your chat ID, send /chatid to your bot on Telegram,
then add it to ~/.miniclaw/.env as ALLOWED_CHAT_IDS.
```

If they left ALLOWED_CHAT_IDS empty, remind them to:

1. Start the bot without an allowlist (it will respond to anyone)
2. Send `/chatid` to the bot
3. Add the ID to `~/.miniclaw/.env`
4. Restart the bot (or `systemctl --user restart miniclaw` if using systemd)

## Rules

- **Idempotent** — never overwrite existing valid files or values; check first, act only if needed
- Be concise and friendly
- Do NOT proceed past a failed step — fix it first
- Do NOT print raw commands unless the user asks to see them
- Do NOT modify any repo files except `agent/preferences.md` — only create/update `~/.miniclaw/.env` and edit preferences
