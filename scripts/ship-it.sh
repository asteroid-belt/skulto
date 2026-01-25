#!/bin/bash
# ship-it: Push to remote after checking for unstaged files

set -e

function check_requirements() {
	if [[ -z "$(command -v git)" ]]; then
		echo "Ensure 'git' is installed before running this script..."
		exit 1
	fi
}

function main() {
	check_requirements

	echo "🚀 Checking for unstaged files..."

	# Check for unstaged changes in working directory
	if ! git diff --quiet; then
		echo "❌ Error: There are unstaged changes in the working directory"
		echo "Please stage or discard these changes before shipping:"
		echo ""
		git diff --name-only
		exit 1
	fi

	# Check for uncommitted changes in the staging area
	if ! git diff --cached --quiet; then
		echo "❌ Error: There are uncommitted changes in the staging area"
		echo "Please commit these changes before shipping:"
		echo ""
		git diff --cached --name-only
		exit 1
	fi

	echo "✅ All files are committed and nothing is staged"
	echo "🔥 Pushing to remote..."

	git push

	echo "✅ Push complete!"
}

main