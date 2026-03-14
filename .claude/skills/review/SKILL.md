---
name: review
description: Review git diff and suggest how to group and commit changes
allowed-tools: "Bash(git *)", "Bash(gofmt *)", "Bash(go test *)"
---

# Git Diff Review

Review the current git diff, summarise what has changed, and suggest how to group the changes into commits.

## Step 1: Determine target repo

Infer which repo the user is working with from the conversation context. Look for the most recently mentioned or worked-on repo. If unclear, default to the miniclaw repo (your working directory's parent).

Once determined, run `git rev-parse --show-toplevel` from that repo's directory to confirm it's a valid git repo. Start your response by stating the absolute path.

## Step 2: Gather state

Run a single command to detect the primary branch and gather all state:

```sh
cd <repo-root> && BASE=$(git symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@^refs/remotes/origin/@@' || git branch -l main master --format '%(refname:short)' | head -1) && echo "=== BASE: $BASE ===" && echo "=== BRANCH ===" && git branch --show-current && echo "=== STATUS ===" && git status && echo "=== DIFF vs BASE ===" && git diff $BASE...HEAD && echo "=== UNSTAGED ===" && git diff && echo "=== STAGED ===" && git diff --staged && echo "=== COMMITS SINCE BASE ===" && git log --oneline $BASE..HEAD && echo "=== RECENT COMMITS ===" && git log --oneline -5
```

## Step 3: CI checks

If the repo has CI checks you can run locally, run them. For example:

- Go: `gofmt -l .` (fix with `gofmt -w` if needed), `go test ./...`
- Node: `npm test` or `npm run lint` if available
- Python: `pytest` if available

Skip this step if no obvious CI checks exist.

## Step 4: Review

Do a comprehensive review of the diff. Look for:

- Bugs, logic errors, or edge cases
- Missing error handling
- Performance concerns
- Code style issues or inconsistencies with the rest of the codebase

Report any findings. If nothing stands out, say the diff looks clean.

## Step 5: Analyse and report

For each changed file, briefly describe what changed and why.

## Step 6: Suggest commits

Group related changes into logical commits. For each suggested commit:

- List the files to include
- Suggest a commit message following the repo's conventional commit style (e.g. `feat:`, `fix:`, `chore:`, `docs:`, `style:`)

If the working tree is clean, just say so.

## Step 7: Ask to proceed

Ask the user if they want you to commit and push, or if they want to adjust the grouping.
