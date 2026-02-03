package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiscoverCommand_Help(t *testing.T) {
	cmd := newDiscoverCmd()
	assert.Equal(t, "discover", cmd.Use)
	assert.Contains(t, cmd.Short, "unmanaged")
}

func TestDiscoverCommand_HasFlags(t *testing.T) {
	cmd := newDiscoverCmd()
	projectFlag := cmd.Flags().Lookup("project")
	globalFlag := cmd.Flags().Lookup("global")
	assert.NotNil(t, projectFlag)
	assert.NotNil(t, globalFlag)
}

func TestDiscoverCommand_FlagsAreBooleans(t *testing.T) {
	cmd := newDiscoverCmd()

	projectFlag := cmd.Flags().Lookup("project")
	assert.Equal(t, "bool", projectFlag.Value.Type())

	globalFlag := cmd.Flags().Lookup("global")
	assert.Equal(t, "bool", globalFlag.Value.Type())
}

func TestDiscoverCommand_LongDescription(t *testing.T) {
	cmd := newDiscoverCmd()
	assert.Contains(t, cmd.Long, "unmanaged")
	assert.Contains(t, cmd.Long, "symlink")
}

func TestDiscoverCommand_HasRunE(t *testing.T) {
	cmd := newDiscoverCmd()
	assert.NotNil(t, cmd.RunE)
}
