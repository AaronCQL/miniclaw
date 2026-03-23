---
name: setup
description: Interactive setup wizard for new miniclaw users who just forked the repo
disable-model-invocation: true
allowed-tools: "Read, Edit, Bash(go *), Bash(which *), Bash(claude *), Bash(mkdir *), Bash(ls *), Bash(cat *), Bash(chmod *), Bash(systemctl *), Bash(loginctl *), Bash(curl *), Bash(uname *), Bash(launchctl *)"
---

# miniclaw Setup Wizard

You are helping a new user set up miniclaw after forking the repo. Walk through each step below **in order**. Check prerequisites first, then guide the user through configuration.

## Rules

- **Idempotent**: this wizard MUST be safe to run multiple times. Before creating or writing any file, check if it already exists and whether its contents are correct. Never overwrite existing valid configuration. Only prompt the user for values that are missing or empty
- Be concise and friendly
- Do NOT proceed past a failed step. Fix it first
- Do NOT print raw commands unless the user asks to see them
- Do NOT modify any repo files except `agent/CLAUDE.md`. Only create/update `~/.miniclaw/.env` and the agent config

## Step 1: Check prerequisites

Run these checks silently and report the results as a checklist:

1. **Go**: run `which go` and `go version`. If `which go` fails, check `~/go/bin/go` and `/usr/local/go/bin/go`. Require Go 1.23+.
2. **Claude CLI**: run `which claude`. If that fails, check these common paths: `~/.claude/local/claude`, `/usr/local/bin/claude`, `~/.npm-global/bin/claude`. Once found, report the path and run `<path> --version`. This is required for the agent runtime. Assume the user is authenticated since they are already using Claude to set it up. Do not run any Claude commands yourself as it will fail.

If any prerequisite is missing, tell the user what to fix and stop. Do not continue until all checks pass.

**Important:** Remember the directories where `go` and `claude` were found. These will be needed for the PATH in the launchd plist (macOS) in Step 12.

## Step 2: Install Go dependencies

Run `go mod tidy` from the repo root to fetch all dependencies. Report success or failure.

## Step 3: Install binary

Run `go install ./cmd/miniclaw/` to compile and install the `miniclaw` binary to the user's `$GOPATH/bin`. Report success or failure.

## Step 4: Create runtime directories

Run `mkdir -p ~/.miniclaw/{data/tasks,workspace}`. This is already idempotent. Report that the runtime directory structure is ready.

## Step 5: Personalise agent

Read `agent/CLAUDE.md` and show the user the current bot name (on line 3, e.g. "You are Enki") and timezone (under `Behaviour > General`). Ask if they want to change either one. Only edit the file if they request a change.

## Step 6: Read existing .env (if any)

Before asking for configuration values, check if `~/.miniclaw/.env` already exists by reading it with `cat ~/.miniclaw/.env 2>/dev/null`. Parse out the current values of:

- `TELEGRAM_BOT_TOKEN`
- `ALLOWED_CHAT_IDS`
- `MINICLAW_AGENT_DIR`

Also check for `GROQ_API_KEY`.

Hold onto these values. Steps 7-10 will only prompt for values that are missing or empty.

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

## Step 10: Groq API key (optional)

If `GROQ_API_KEY` already has a non-empty value in the existing .env, report that a key is already configured (show a masked version) and skip.

Otherwise, tell the user:

- This is optional but required for voice message transcription
- Sign up at https://console.groq.com and create an API key
- The free tier is generous (2,000 requests/day, 8 hours of audio/day)

The user may skip this and add it later. Hold onto the value for Step 11.

## Step 11: Write .env file

If `~/.miniclaw/.env` already exists and all values (`TELEGRAM_BOT_TOKEN`, `ALLOWED_CHAT_IDS`, `MINICLAW_AGENT_DIR`, and optionally `GROQ_API_KEY`) are correct, report that the .env file is already up to date and skip writing.

Otherwise, write `~/.miniclaw/.env` with the merged values (existing values for fields that didn't change, new values for fields that did):

```
TELEGRAM_BOT_TOKEN=<token>
ALLOWED_CHAT_IDS=<chat IDs>
MINICLAW_AGENT_DIR=<absolute path to agent/>
GROQ_API_KEY=<key, if provided>
```

Use the Bash tool to write this file with `0600` permissions. Do NOT use the Write tool (the path is outside the project).

## Step 12: Background service (optional)

First, detect the platform by running `uname -s`. If the result is `Darwin`, follow the **macOS (launchd)** path. Otherwise, follow the **Linux (systemd)** path.

### Linux (systemd)

Check if `~/.config/systemd/user/miniclaw.service` already exists. If it does, read its contents and verify the `ExecStart` path is correct (matches `which miniclaw`).

- If the service file exists and is correct, report that systemd is already configured. Check if the service is enabled (`systemctl --user is-enabled miniclaw`) and skip to asking if they want to start/restart it.
- If the service file exists but `ExecStart` is wrong, tell the user and offer to update it.
- If the service file does not exist, ask the user if they want to set up systemd.

If they decline, skip to Step 13.

To set up or update the service:

1. Determine the absolute path to the `miniclaw` binary by running `which miniclaw` or falling back to `ls ~/go/bin/miniclaw`.
2. Determine the directory where `claude` was found (from Step 1). This is the only extra PATH entry needed since `miniclaw` is referenced by absolute path in `ExecStart` and Claude's Bash tool reads the user's shell profile for its own PATH.
3. Run `mkdir -p ~/.config/systemd/user` (idempotent).
4. Write `~/.config/systemd/user/miniclaw.service` via the Bash tool with the following content:

```ini
[Unit]
Description=miniclaw Telegram Bot
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=%h/.miniclaw/.env
Environment=PATH=<directory containing claude>:/usr/local/bin:/usr/bin:/bin
ExecStart=<absolute path to miniclaw binary>
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
```

5. Reload the systemd user daemon: `systemctl --user daemon-reload`
6. Enable the service so it starts on login: `systemctl --user enable miniclaw`
7. Enable lingering so it runs even when the user is not logged in: `loginctl enable-linger`
8. Ask if they want to start it now. If yes: `systemctl --user start miniclaw` and confirm it's running with `systemctl --user status miniclaw`.

### macOS (launchd)

Check if `~/Library/LaunchAgents/com.miniclaw.agent.plist` already exists. If it does, read its contents and verify the `ProgramArguments` path is correct (matches `which miniclaw`).

- If the plist exists and is correct, report that launchd is already configured. Check if it's loaded (`launchctl list | grep com.miniclaw.agent`) and skip to asking if they want to start/restart it.
- If the plist exists but the binary path is wrong, tell the user and offer to update it.
- If the plist does not exist, ask the user if they want to set up launchd.

If they decline, skip to Step 13.

To set up or update the service:

1. Determine the absolute path to the `miniclaw` binary by running `which miniclaw` or falling back to `ls ~/go/bin/miniclaw`.
2. Read `~/.miniclaw/.env` and parse all key-value pairs. These will become `EnvironmentVariables` in the plist.
3. Build a `PATH` value that includes the directories where `go`, `claude`, and `miniclaw` were found (from Step 1 and above), plus standard paths: `/usr/local/bin`, `/usr/bin`, `/bin`, `/usr/sbin`, `/sbin`. Deduplicate entries.
4. Run `mkdir -p ~/Library/LaunchAgents` (idempotent).
5. Write `~/Library/LaunchAgents/com.miniclaw.agent.plist` via the Bash tool with the following content:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.miniclaw.agent</string>
    <key>ProgramArguments</key>
    <array>
        <string><absolute path to miniclaw binary></string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string><constructed PATH from step 3></string>
        <key>TELEGRAM_BOT_TOKEN</key>
        <string><value></string>
        <key>ALLOWED_CHAT_IDS</key>
        <string><value></string>
        <key>MINICLAW_AGENT_DIR</key>
        <string><value></string>
        <!-- include GROQ_API_KEY if set -->
    </dict>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/miniclaw.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/miniclaw.log</string>
</dict>
</plist>
```

5. If the service was previously loaded, unload it first: `launchctl unload ~/Library/LaunchAgents/com.miniclaw.agent.plist 2>/dev/null`
6. Load the service: `launchctl load ~/Library/LaunchAgents/com.miniclaw.agent.plist`
7. Verify it's running: `launchctl list | grep com.miniclaw.agent`

## Step 13: Register bot commands (optional)

If `TELEGRAM_BOT_TOKEN` is configured, ask the user if they want to register bot commands with Telegram so they appear in the command menu when typing `/`.

If they agree, invoke the `/commands` skill (read `.claude/skills/commands/SKILL.md` and follow its instructions).

If they decline, tell them they can run `/commands` later.

## Step 14: Done

Print a summary:

```
Setup complete!
```

If they set up systemd (Linux), add:

```
miniclaw is running as a systemd user service.

  systemctl --user status miniclaw   # check status
  systemctl --user restart miniclaw  # restart after config changes
  journalctl --user -u miniclaw -f   # follow logs
```

If they set up launchd (macOS), add:

```
miniclaw is running as a launchd agent.

  launchctl list | grep com.miniclaw.agent   # check status
  launchctl kickstart -k gui/$(id -u)/com.miniclaw.agent   # restart
  tail -f /tmp/miniclaw.log   # follow logs
```

If they skipped the background service, add:

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
4. Restart the bot

