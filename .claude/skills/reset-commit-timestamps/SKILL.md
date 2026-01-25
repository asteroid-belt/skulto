---
name: reset-commit-timestamps
description: Reset git commit timestamps to the current time. Use when the user wants to update commit dates, refresh timestamps on unpushed commits, or mentions resetting/changing commit times. Supports specifying commits by count, SHA range, or list of SHAs.
compatibility: Requires git. Only works on commits that have not been pushed to remote (rewrites history).
metadata:
  author: skulto
  version: "1.0"
allowed-tools: Bash(git:*) AskUserQuestion
---

# Reset Commit Timestamps

This skill resets both author and committer dates on git commits to the current time.

## Safety Check

Before proceeding, verify the commits have NOT been pushed to remote:

```bash
git log --oneline origin/$(git branch --show-current)..HEAD 2>/dev/null || echo "Branch not tracked or no remote"
```

If the commits exist on remote, warn the user that this will require a force push.

## Step 1: Ask for Input Format

Ask the user which format they want to use:

**Option A: Number of recent commits**
- Example: "last 5 commits" or just "5"

**Option B: Commit SHA range**
- Example: "from abc123 to def456" (oldest to newest)

**Option C: List of specific SHAs**
- Example: "abc123, def456, ghi789"

## Step 2: Determine the Rebase Target

Based on input format:

**For count (N commits):**
```bash
# Target is HEAD~N
TARGET="HEAD~<N>"
```

**For SHA range (from OLDEST to NEWEST):**
```bash
# Target is the parent of the oldest commit
TARGET="<OLDEST_SHA>~1"
```

**For list of SHAs:**
```bash
# Find the oldest commit in the list and use its parent
# Sort commits by their position in history
TARGET="<OLDEST_SHA_IN_LIST>~1"
```

## Step 3: Run the Timestamp Reset

Execute the shell script with the determined target:

```bash
scripts/reset-timestamps.sh <TARGET>
```

Or run directly:

```bash
git rebase <TARGET> --exec 'export NOW=$(date -R) && GIT_COMMITTER_DATE="$NOW" git commit --amend --no-edit --date="$NOW"'
```

## Step 4: Verify Results

Show the updated commits with their new timestamps:

```bash
git log --oneline --format="%h %ad %s" --date=short -<N>
```

## Example Interaction

User: "Reset timestamps on the last 5 commits"

1. Check if commits are unpushed
2. Run: `git rebase HEAD~5 --exec 'export NOW=$(date -R) && GIT_COMMITTER_DATE="$NOW" git commit --amend --no-edit --date="$NOW"'`
3. Show updated log with new dates

## Notes

- Commit hashes will change after rebasing (this is expected)
- Both author date and committer date are updated
- All specified commits will get the same timestamp (current time when each is processed during rebase)
