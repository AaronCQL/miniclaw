---
name: release
description: Create a semver release with changelog, git tag, and GitHub release
allowed-tools: "Bash(git *)", "Bash(gh *)"
---

# Release

Create a new semver release: infer the version, generate a changelog from commits since the last tag, and publish a GitHub release.

## Step 1: Determine target repo

Infer which repo the user is working with from the conversation context. If unclear, default to the miniclaw repo (your working directory's parent).

Run `git rev-parse --show-toplevel` to confirm it's a valid git repo. Start your response by stating the absolute path.

## Step 2: Ensure on primary branch

```sh
cd <repo-root> && BASE=$(git symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@^refs/remotes/origin/@@' || git branch -l main master --format '%(refname:short)' | head -1) && CURRENT=$(git branch --show-current) && echo "BASE: $BASE" && echo "CURRENT: $CURRENT" && git pull && git status
```

If not on the primary branch, stop and tell the user to switch first.

If the working tree is dirty, list the uncommitted changes and ask the user if they want to commit everything before proceeding. If yes, stage all changes, commit with an appropriate message, and continue.

## Step 3: Determine version

Find the latest semver tag:

```sh
git tag -l 'v*' --sort=-v:refname | head -1
```

If no tags exist, the first release is `v0.1.0`. Use a brief "Initial release" changelog entry instead of listing every commit.

For subsequent releases, infer the next version by scanning commit messages since the last tag:

```sh
git log <last-tag>..HEAD --oneline
```

Apply conventional commit rules to determine the bump:

**Pre-v1 (v0.x.y):**
- `feat:` or breaking changes -> bump minor (v0.1.0 -> v0.2.0)
- `fix:`, `chore:`, `docs:`, `style:`, `refactor:` -> bump patch (v0.1.0 -> v0.1.1)

**Post-v1 (v1.0.0+):**
- Breaking changes (`BREAKING CHANGE:` or `!:` suffix) -> bump major
- `feat:` -> bump minor
- Everything else -> bump patch

State the proposed version and why.

## Step 4: Generate changelog entry

Group commits by type and write a changelog entry:

```markdown
## [vX.Y.Z] - YYYY-MM-DD

### Features
- description (from feat: commits)

### Bug Fixes
- description (from fix: commits)

### Improvements
- description (from refactor:, chore:, docs:, style: commits)
```

Only include sections that have entries. Strip the conventional commit prefix (feat:, fix:, etc.) from each description.

## Step 5: Update CHANGELOG.md

If CHANGELOG.md doesn't exist, create it with a header:

```markdown
# Changelog

All notable changes to this project will be documented in this file.
```

Prepend the new entry after the header (newest first). Keep all previous entries intact.

## Step 6: Confirm with user

Show the user:
- The proposed version
- The full changelog entry
- The list of commits included

Ask for confirmation before proceeding.

## Step 7: Tag and release

After confirmation:

```sh
git add CHANGELOG.md
git commit -m "chore: release vX.Y.Z"
git tag vX.Y.Z
git push --follow-tags
```

Then create a GitHub release:

```sh
gh release create vX.Y.Z --title "vX.Y.Z" --notes "$(cat <<'EOF'
<changelog entry content here>
EOF
)"
```

## Step 8: Report

Confirm the release was created and provide the GitHub release URL.
