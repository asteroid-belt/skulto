package cli

import (
	"testing"

	"github.com/asteroid-belt/skulto/internal/scraper"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddCmd_Structure(t *testing.T) {
	assert.Equal(t, "add <repository_url>", addCmd.Use)
	assert.NotEmpty(t, addCmd.Short)
	assert.NotEmpty(t, addCmd.Long)
	assert.NotNil(t, addCmd.Args)
}

func TestAddCmd_ArgsValidation(t *testing.T) {
	validator := cobra.ExactArgs(1)

	err := validator(addCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg")

	err = validator(addCmd, []string{"owner/repo"})
	assert.NoError(t, err)

	err = validator(addCmd, []string{"arg1", "arg2"})
	assert.Error(t, err)
}

func TestAddCmd_ValidURLFormats(t *testing.T) {
	testCases := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
	}{
		{"short format", "owner/repo", "owner", "repo"},
		{"https URL", "https://github.com/owner/repo", "owner", "repo"},
		{"https URL with .git", "https://github.com/owner/repo.git", "owner", "repo"},
		{"ssh URL", "git@github.com:owner/repo.git", "owner", "repo"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			source, err := scraper.ParseRepositoryURL(tc.url)
			require.NoError(t, err)
			assert.Equal(t, tc.wantOwner, source.Owner)
			assert.Equal(t, tc.wantRepo, source.Repo)
		})
	}
}

func TestAddCmd_InvalidURL(t *testing.T) {
	_, err := scraper.ParseRepositoryURL("invalid-url")
	assert.Error(t, err)
}
