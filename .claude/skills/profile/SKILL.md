---
name: profile
description: Analyse chat history to update user profile and voice guide (profile.md)
---

# Profile Update

Go through all conversation transcripts, extract user messages, and update the user profile with new observations about who the user is and how they type.

## Step 1: Find transcripts

List all JSONL transcript files:

```bash
find ~/.claude/projects/ -name "*.jsonl" -type f
```

## Step 2: Extract user messages

For each transcript file, extract all user-typed messages using this Python script:

```bash
python3 << 'PYEOF'
import json, glob
from datetime import datetime, timedelta, timezone

cutoff = datetime.now(timezone.utc) - timedelta(days=7)
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
            if ts:
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

print(f"Found {len(msgs)} user messages from the last 7 days across {len(files)} transcript(s)\n")
for i, m in enumerate(msgs):
    print(f"=== [{i}] ===")
    print(m[:800])
    print()
PYEOF
```

This output will be large. Skim through all of it to build a full picture of both WHAT the user is saying and HOW they type.

## Step 3: Read current profile

Read `~/.miniclaw/data/profile.md` to understand what's already captured.

## Step 4: Analyse and update

Compare the messages against what's already in the profile. Look for:

**Profile (who they are):**
- New personality traits or behavioural patterns
- Life updates (career changes, new hobbies, relationship developments)
- Updated opinions or preferences
- New blind spots or growth areas observed

**Voice (how they type):**
- New abbreviations or slang not yet captured
- Shifts in tone or formality
- New expressions or verbal tics
- Patterns that were wrong or overstated in the current guide
- Changes in emoji usage, punctuation habits, or sentence structure

Only update with information the user has clearly stated or demonstrated across multiple messages. Do not speculate or over-index on one-off phrasing.

## Step 5: Apply changes

Edit `~/.miniclaw/data/profile.md` with the updates. Keep it concise and well-organised. Do not duplicate existing entries.

## Step 6: Report

Tell the user what was added or changed, and why. Ask if anything should be adjusted.
