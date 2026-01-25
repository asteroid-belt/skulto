package views

import (
	"os"
	"strings"
)

// FormatPath formats a file path for display, replacing the home directory
// with "~/" for a cleaner, more readable presentation.
func FormatPath(path string) string {
	if path == "" {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}

	// Replace home directory with ~
	if after, found := strings.CutPrefix(path, home); found {
		return "~" + after
	}

	return path
}
