---
name: voice
description: Analyse chat history to update voice and typing style guide (voice.md in auto memory)
---

# Voice Update

This skill delegates to a sub-agent to keep large transcript data out of the main context window.

**Arguments:** optional time window (e.g. `1d`, `7d`, `30d`, `all`). Defaults to `7d`.

## Instructions

Launch a sub-agent using the Agent tool with the prompt below. Substitute `{{DAYS}}` with the parsed time window:
- `all` -> `0`
- `14d` -> `14`
- `30d` -> `30`
- no argument -> `7`

When the agent completes, relay its response directly to the user without modification.

### Sub-agent prompt

```md
Go through all conversation transcripts, extract user messages, and update the voice and typing style guide.

## Step 1: Find transcripts

List all JSONL transcript files:

find ~/.claude/projects/ -name "*.jsonl" -type f

## Step 2: Extract user messages

For each transcript file, extract all user-typed messages within the time window. Run this script with DAYS={{DAYS}}:

python3 << 'PYEOF'
import json, glob, os
from datetime import datetime, timedelta, timezone

DAYS = {{DAYS}}  # 0 means no cutoff (all time)
cutoff = datetime.now(timezone.utc) - timedelta(days=DAYS) if DAYS > 0 else None
label = "all time" if not cutoff else f"the last {DAYS}d"
files = glob.glob(os.path.expanduser("~/.claude/projects/**/*.jsonl"), recursive=True)
msgs = []

for fpath in files:
    with open(fpath, "r") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            try:
                obj = json.loads(line)
            except:
                continue
            if obj.get("type") != "user":
                continue
            ts = obj.get("timestamp", "")
            if cutoff and ts:
                try:
                    dt = datetime.fromisoformat(ts)
                    if dt < cutoff:
                        continue
                except:
                    pass
            message = obj.get("message", {})
            content = message.get("content", "")
            texts = []
            if isinstance(content, str):
                texts.append(content)
            elif isinstance(content, list):
                for block in content:
                    if isinstance(block, dict) and block.get("type") == "text":
                        texts.append(block.get("text", ""))
            for t in texts:
                t = t.strip()
                if len(t) > 5 and not t.startswith("<system") and not t.startswith("<command") and not t.startswith("<local-command") and not t.startswith("Base directory for this skill"):
                    msgs.append(t)

print(f"Found {len(msgs)} user messages from {label} across {len(files)} transcript(s)\n")
for i, m in enumerate(msgs):
    print(f"=== [{i}] ===")
    print(m[:800])
    print()
PYEOF

Skim through all of the output focusing on HOW the user types, not WHAT they're saying. Life updates and personal context are handled by the /remember skill.

## Step 3: Read current voice guide

Read the voice.md file from the auto memory directory (the path is provided in your system prompt) to understand what's already captured.

## Step 4: Analyse and update

Compare the user's actual typing patterns against what's in the voice guide. Look for:

- New abbreviations or slang not yet captured
- Shifts in tone or formality
- New expressions or verbal tics
- Patterns that were wrong or overstated in the current guide
- Changes in emoji usage, punctuation habits, or sentence structure

Only document patterns that appear consistently across multiple messages. Do not over-index on one-off phrasing.

## Step 5: Apply changes

Edit the voice.md file in the auto memory directory with the updates. Keep it concise and well-organised. Do not duplicate existing entries.

## Step 6: Report

Start your response with "/voice summary" so the user knows which skill produced this output. Then tell the user what was added or changed, and why.
```
