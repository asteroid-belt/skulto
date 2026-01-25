#!/bin/bash
#
# reset-timestamps.sh - Reset git commit timestamps to current time
#
# Usage:
#   ./reset-timestamps.sh <target>
#
# Arguments:
#   target - The rebase target (parent of oldest commit to modify)
#            Examples: HEAD~5, abc123~1, main
#
# Examples:
#   # Reset last 5 commits
#   ./reset-timestamps.sh HEAD~5
#
#   # Reset all commits since (but not including) abc123
#   ./reset-timestamps.sh abc123
#
#   # Reset all commits since main
#   ./reset-timestamps.sh main

set -e

TARGET="${1:-}"

if [ -z "$TARGET" ]; then
    echo "Error: No target specified"
    echo ""
    echo "Usage: $0 <target>"
    echo ""
    echo "Examples:"
    echo "  $0 HEAD~5        # Reset last 5 commits"
    echo "  $0 abc123~1      # Reset commits from abc123 onwards"
    echo "  $0 main          # Reset all commits since main"
    exit 1
fi

# Verify target exists
if ! git rev-parse "$TARGET" >/dev/null 2>&1; then
    echo "Error: Invalid target '$TARGET'"
    exit 1
fi

# Count commits that will be modified
COMMIT_COUNT=$(git rev-list --count "$TARGET"..HEAD)

if [ "$COMMIT_COUNT" -eq 0 ]; then
    echo "No commits to modify between $TARGET and HEAD"
    exit 0
fi

echo "Resetting timestamps on $COMMIT_COUNT commit(s)..."
echo ""

# Show commits that will be modified
echo "Commits to be modified:"
git log --oneline "$TARGET"..HEAD
echo ""

# Perform the rebase with timestamp reset
git rebase "$TARGET" --exec 'export NOW=$(date -R) && GIT_COMMITTER_DATE="$NOW" git commit --amend --no-edit --date="$NOW"'

echo ""
echo "Done! Updated commits:"
git log --oneline --format="%h %ad %s" --date=short -"$COMMIT_COUNT"
