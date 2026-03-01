---
name: diff
description: Review git diff and suggest how to group and commit changes
allowed-tools: "Bash(git *)", "Bash(gofmt *)", "Bash(go test *)"
---

# Git Diff Review

Review the current git diff in the miniclaw repo, summarise what has changed, and suggest how to group the changes into commits.

## Step 1: Announce repo

Start your response by stating the absolute path of the git repo root (run `git rev-parse --show-toplevel`).

## Step 2: Gather state

Run these commands from the repo root:

1. `git status` — to see all modified, staged, and untracked files
2. `git diff` — to see unstaged changes
3. `git diff --staged` — to see any already-staged changes
4. `git log --oneline -5` — to see recent commit style

## Step 3: CI checks

Run the same checks that CI runs:

1. `gofmt -l .` — if any files are listed, run `gofmt -w` on them to fix formatting before continuing
2. `go test ./...` — report any test failures

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
