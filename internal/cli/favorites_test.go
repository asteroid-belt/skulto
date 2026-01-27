package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFavoritesCmd_Structure(t *testing.T) {
	// Verify favoritesCmd exists and has expected subcommands
	assert.Equal(t, "favorites", favoritesCmd.Use)
	assert.NotEmpty(t, favoritesCmd.Short)
	assert.NotEmpty(t, favoritesCmd.Long)

	// Verify subcommands exist
	subCmds := favoritesCmd.Commands()
	require.Len(t, subCmds, 3)

	cmdNames := make([]string, len(subCmds))
	for i, cmd := range subCmds {
		cmdNames[i] = cmd.Name()
	}

	assert.Contains(t, cmdNames, "add")
	assert.Contains(t, cmdNames, "remove")
	assert.Contains(t, cmdNames, "list")
}

func TestFavoritesAddCmd_ArgsValidation(t *testing.T) {
	// Add command requires exactly 1 argument
	assert.Equal(t, "add <slug>", favoritesAddCmd.Use)

	// Test args validation
	err := favoritesAddCmd.Args(favoritesAddCmd, []string{})
	assert.Error(t, err, "should require at least 1 arg")

	err = favoritesAddCmd.Args(favoritesAddCmd, []string{"slug"})
	assert.NoError(t, err, "should accept exactly 1 arg")

	err = favoritesAddCmd.Args(favoritesAddCmd, []string{"slug1", "slug2"})
	assert.Error(t, err, "should reject more than 1 arg")
}

func TestFavoritesRemoveCmd_ArgsValidation(t *testing.T) {
	// Remove command requires exactly 1 argument
	assert.Equal(t, "remove <slug>", favoritesRemoveCmd.Use)

	// Test args validation
	err := favoritesRemoveCmd.Args(favoritesRemoveCmd, []string{})
	assert.Error(t, err, "should require at least 1 arg")

	err = favoritesRemoveCmd.Args(favoritesRemoveCmd, []string{"slug"})
	assert.NoError(t, err, "should accept exactly 1 arg")

	err = favoritesRemoveCmd.Args(favoritesRemoveCmd, []string{"slug1", "slug2"})
	assert.Error(t, err, "should reject more than 1 arg")
}

func TestFavoritesListCmd_NoArgs(t *testing.T) {
	// List command requires no arguments
	assert.Equal(t, "list", favoritesListCmd.Use)

	// Test args validation
	err := favoritesListCmd.Args(favoritesListCmd, []string{})
	assert.NoError(t, err, "should accept 0 args")

	err = favoritesListCmd.Args(favoritesListCmd, []string{"extra"})
	assert.Error(t, err, "should reject any args")
}
