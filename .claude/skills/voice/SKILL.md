---
name: voice
description: Analyse chat history to update voice and typing style guide (voice.md in auto memory)
---

# Voice Update

Go through all conversation transcripts, extract user messages, and update the voice and typing style guide with new observations about how the user communicates.

**Arguments:** optional time window (e.g. `1d`, `7d`, `30d`, `all`). Defaults to `7d`.

## Step 1: Find transcripts

List all JSONL transcript files:

```bash
find ~/.claude/projects/ -name "*.jsonl" -type f
```

## Step 2: Extract user messages

Parse the time window from the skill arguments. Examples: `/voice all`, `/voice 14d`, `/voice 30d`. Default is `7d` if no argument is given.

Before running the script below, substitute `<DAYS>` with the appropriate value:
- `all` -> `0` (no cutoff)
- `14d` -> `14`
- `30d` -> `30`
- no argument -> `7`

For each transcript file, extract all user-typed messages within the time window:

```bash
python3 << 'PYEOF'
import json, glob
from datetime import datetime, timedelta, timezone

DAYS = <DAYS>  # 0 means no cutoff (all time)
cutoff = datetime.now(timezone.utc) - timedelta(days=DAYS) if DAYS > 0 else None
label = "all time" if not cutoff else f"the last {DAYS}d"
files = glob.glob("/home/htpc/.claude/projects/**/*.jsonl", recursive=True)
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
```

This output will be large. Skim through all of it focusing on HOW the user types, not WHAT they're saying. Life updates and personal context are handled by the /remember skill.

## Step 3: Read current voice guide

Read `~/.claude/projects/-home-htpc-Desktop-dev-miniclaw/memory/voice.md` to understand what's already captured.

## Step 4: Analyse and update

Compare the user's actual typing patterns against what's in the voice guide. Look for:

- New abbreviations or slang not yet captured
- Shifts in tone or formality
- New expressions or verbal tics
- Patterns that were wrong or overstated in the current guide
- Changes in emoji usage, punctuation habits, or sentence structure

Only document patterns that appear consistently across multiple messages. Do not over-index on one-off phrasing.

## Step 5: Apply changes

Edit `~/.claude/projects/-home-htpc-Desktop-dev-miniclaw/memory/voice.md` with the updates. Keep it concise and well-organised. Do not duplicate existing entries.

## Step 6: Report

Tell the user what was added or changed, and why. Ask if anything should be adjusted.
