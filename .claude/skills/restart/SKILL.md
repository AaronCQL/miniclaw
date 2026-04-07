---
name: restart
description: Rebuild miniclaw and restart the background service
allowed-tools: "Bash(source *), Bash(go install *), Bash(date *), Bash(cat *), Bash(uname *), Bash(systemctl --user restart miniclaw), Bash(launchctl *)"
---

# Restart miniclaw

This skill rebuilds the miniclaw binary and restarts the background service (systemd on Linux, launchd on macOS). The restart will kill the current process (including this claude subprocess), so a scheduled task is used to confirm success afterwards.

Follow these steps **in order**. Do NOT skip steps or reorder them.

## Step 1: Load environment and detect platform

Run `source ~/.miniclaw/.env` to ensure `MINICLAW_AGENT_DIR` and other variables are available.

Run `uname -s` to detect the platform. If the result is `Darwin`, use the **macOS (launchd)** restart command in Step 4. Otherwise, use the **Linux (systemd)** command.

## Step 2: Build the binary

Run `go install $MINICLAW_AGENT_DIR/../cmd/miniclaw/` to compile and install the updated binary. Report success or failure. Do NOT continue if the build fails.

## Step 3: Schedule a post-restart confirmation

Create a one-time scheduled task for **each** chat ID in `$ALLOWED_CHAT_IDS` (comma-separated). Each task fires 10 seconds from now and sends a confirmation message.

1. Compute the timestamp:
   - **Linux:** `date -u -d '+10 seconds' --iso-8601=seconds`
   - **macOS:** `date -u -v+10S +%Y-%m-%dT%H:%M:%SZ`
2. Loop over each chat ID and write a task file per chat:

```bash
# Use the appropriate date command for the platform
if [ "$(uname -s)" = "Darwin" ]; then
  TIMESTAMP=$(date -u -v+10S +%Y-%m-%dT%H:%M:%SZ)
else
  TIMESTAMP=$(date -u -d '+10 seconds' --iso-8601=seconds)
fi
THREAD_ID=${MINICLAW_THREAD_ID:-0}
IFS=',' read -ra CHAT_IDS <<< "$ALLOWED_CHAT_IDS"
for CHAT_ID in "${CHAT_IDS[@]}"; do
  CHAT_ID=$(echo "$CHAT_ID" | tr -d ' ')
  THREAD_JSON=""
  if [ "$THREAD_ID" -gt 0 ] 2>/dev/null; then
    THREAD_JSON="\"thread_id\": ${THREAD_ID},"
  fi
  cat > ~/.miniclaw/data/tasks/restart-confirmation-${CHAT_ID}.json << EOF
{
    "prompt": "Send a short message confirming that miniclaw has been rebuilt and restarted successfully. Keep it to one or two lines.",
    "chat_id": ${CHAT_ID},
    ${THREAD_JSON}
    "type": "once",
    "value": "",
    "status": "active",
    "next_run": "${TIMESTAMP}"
}
EOF
done
```

It is fine to overwrite these files if they already exist from a previous restart.

## Step 4: Restart the service

**Linux (systemd):** Run `systemctl --user restart miniclaw`.

**macOS (launchd):** Run `launchctl kickstart -k gui/$(id -u)/com.miniclaw.agent`.

**This command kills the current miniclaw process and this claude subprocess.** This is expected behaviour. Nothing after this command will execute.
