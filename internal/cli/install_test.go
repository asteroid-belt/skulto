package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstallCmd_Structure(t *testing.T) {
	assert.Equal(t, "install", installCmd.Use[:7])
	assert.NotEmpty(t, installCmd.Short)
	assert.NotEmpty(t, installCmd.Long)

	// Check flags exist
	pFlag := installCmd.Flags().Lookup("platform")
	assert.NotNil(t, pFlag)
	assert.Equal(t, "p", pFlag.Shorthand)

	sFlag := installCmd.Flags().Lookup("scope")
	assert.NotNil(t, sFlag)
	assert.Equal(t, "s", sFlag.Shorthand)

	yFlag := installCmd.Flags().Lookup("yes")
	assert.NotNil(t, yFlag)
	assert.Equal(t, "y", yFlag.Shorthand)
}

func TestInstallCmd_RequiresArg(t *testing.T) {
	err := installCmd.Args(installCmd, []string{})
	assert.Error(t, err, "Should require at least 1 argument")
}

func TestInstallCmd_AcceptsArg(t *testing.T) {
	err := installCmd.Args(installCmd, []string{"docker-expert"})
	assert.NoError(t, err, "Should accept 1 argument")
}

func TestIsURL(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"docker-expert", false},
		{"https://github.com/owner/repo", true},
		{"http://github.com/owner/repo", true},
		{"owner/repo", true},
		{"./local/path", false},
		{"../relative", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
