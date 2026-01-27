package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUninstallCmd_Structure(t *testing.T) {
	assert.Equal(t, "uninstall", uninstallCmd.Use[:9])
	assert.NotEmpty(t, uninstallCmd.Short)
	assert.NotEmpty(t, uninstallCmd.Long)

	// Check flags exist
	yFlag := uninstallCmd.Flags().Lookup("yes")
	assert.NotNil(t, yFlag)
	assert.Equal(t, "y", yFlag.Shorthand)

	aFlag := uninstallCmd.Flags().Lookup("all")
	assert.NotNil(t, aFlag)
	assert.Equal(t, "a", aFlag.Shorthand)
}

func TestUninstallCmd_RequiresArg(t *testing.T) {
	err := uninstallCmd.Args(uninstallCmd, []string{})
	assert.Error(t, err, "Should require at least 1 argument")
}

func TestUninstallCmd_AcceptsArg(t *testing.T) {
	err := uninstallCmd.Args(uninstallCmd, []string{"docker-expert"})
	assert.NoError(t, err, "Should accept 1 argument")
}
