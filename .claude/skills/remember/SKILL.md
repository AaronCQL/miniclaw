---
name: remember
description: Summarise recent conversations across all threads into auto memory (MEMORY.md + topic files)
---

# Remember

Scan all recent conversation transcripts across all sessions/threads, extract key context, and update the shared auto memory so that future sessions start with cross-thread awareness.

**Arguments:** optional time window (e.g. `1d`, `7d`, `30d`, `all`). Defaults to `1d`.

## Step 1: Find transcripts

List all JSONL transcript files:

```bash
find ~/.claude/projects/ -name "*.jsonl" -type f
```

## Step 2: Extract conversation context

Parse the time window from the skill arguments. Examples: `/remember all`, `/remember 7d`, `/remember 30d`. Default is `1d` if no argument is given.

Before running the script below, substitute `<DAYS>` with the appropriate value:
- `all` -> `0` (no cutoff)
- `7d` -> `7`
- `30d` -> `30`
- no argument -> `1`

For each transcript file, extract both user and assistant messages within the time window:

```bash
python3 << 'PYEOF'
import json, glob, os
from datetime import datetime, timedelta, timezone

DAYS = <DAYS>  # 0 means no cutoff (all time)
cutoff = datetime.now(timezone.utc) - timedelta(days=DAYS) if DAYS > 0 else None
label = "all time" if not cutoff else f"the last {DAYS}d"
files = glob.glob("/home/htpc/.claude/projects/**/*.jsonl", recursive=True)
sessions = {}

for fpath in files:
    session_id = os.path.basename(fpath).replace(".jsonl", "")
    turns = []
    with open(fpath, "r") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            try:
                obj = json.loads(line)
            except:
                continue
            msg_type = obj.get("type")
            if msg_type not in ("user", "assistant"):
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
                # Skip system injections, skill triggers, and very short messages
                if len(t) < 10:
                    continue
                if any(t.startswith(p) for p in ("<system", "<command", "<local-command", "Base directory for this skill")):
                    continue
                turns.append({"role": msg_type, "text": t[:500], "ts": ts})
    if turns:
        sessions[session_id] = turns

print(f"Found {sum(len(t) for t in sessions.values())} messages across {len(sessions)} session(s) from {label}\n")
for sid, turns in sessions.items():
    print(f"=== SESSION {sid[:12]}... ({len(turns)} messages) ===")
    for turn in turns:
        role = "USER" if turn["role"] == "user" else "ASST"
        print(f"[{role}] {turn['text'][:300]}")
    print()
PYEOF
```

This output will be large. Read through all of it to understand what was discussed, decided, and built across all threads.

## Step 3: Read current memory

Read all files in the memory directory:

```bash
ls ~/.claude/projects/-home-htpc-Desktop-dev-miniclaw/memory/
```

Read `MEMORY.md` and any existing topic files to understand what's already captured.

## Step 4: Analyse and categorise

Extract memories into these categories. Conversations can cover anything - engineering, personal life, interests, advice sought, etc.

**decisions** - choices made and why (e.g. "chose X over Y because of free tier")
**entities** - people, projects, services, books, or things referenced across threads (e.g. "Project X - a side project using Rust")
**cases** - problem + solution pairs worth remembering (e.g. "API rejects .oga files - fix: rename to .ogg")
**patterns** - reusable approaches or preferences discovered (e.g. "user prefers squash merge for PRs")
**events** - milestones, life updates, or time-bound context (e.g. "2026-03-10: started exploring new integration", "2026-03-15: user started a new job")
**topics** - ongoing themes, interests, or personal context (e.g. "user is reading a specific book", "user exploring business ideas with family")

Before creating a new memory, check existing topic files for overlap:
- If a file already covers the topic, **merge** the new information into it
- If the new information contradicts an existing entry, **replace** with the latest
- If it's already captured, **skip**
- Only **create** a new file when no existing file fits

Drop anything that's:
- Already captured in MEMORY.md or topic files
- Ephemeral (greetings, routine confirmations)
- Derivable from the code or git history
- Already documented in CLAUDE.md or profile.md

## Step 5: Update memory

Structure the memory as:

**MEMORY.md** (the index, max 200 lines): a concise list of one-line pointers to topic files with brief descriptions. Group by topic. This file is loaded every single message, so keep it lean.

**Topic files** (e.g. `gemini-integration.md`, `skill-design.md`): detailed context per topic. These are read on demand. Each topic file should have frontmatter:

```markdown
---
name: <topic name>
description: <one-line description used to decide relevance>
type: <decisions|entities|cases|patterns|events|topics>
---

<content>
```

A single topic file can contain multiple related entries. For example, `cases-cli.md` might hold several CLI-related problem/solution pairs rather than one file per case.

Rules:
- Remove entries that are stale or no longer relevant
- Do not duplicate what's in CLAUDE.md, profile.md, or git history
- Convert relative dates to absolute dates (e.g. "yesterday" -> "2026-03-16")
- Keep MEMORY.md well under 200 lines

## Step 6: Report

Tell the user:
- What was added or updated
- What was removed as stale
- Current MEMORY.md line count
