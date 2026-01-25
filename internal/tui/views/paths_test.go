package views

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Could not get home directory")
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty path",
			input:    "",
			expected: "",
		},
		{
			name:     "path in home directory",
			input:    filepath.Join(home, ".skulto", "skills"),
			expected: "~/.skulto/skills",
		},
		{
			name:     "home directory itself",
			input:    home,
			expected: "~",
		},
		{
			name:     "path outside home directory",
			input:    "/tmp/some/path",
			expected: "/tmp/some/path",
		},
		{
			name:     "relative path",
			input:    "./local/path",
			expected: "./local/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
