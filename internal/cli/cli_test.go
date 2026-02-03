package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRootCmd_Structure(t *testing.T) {
	assert.Equal(t, "skulto", rootCmd.Use)
	assert.NotEmpty(t, rootCmd.Short)
	assert.NotEmpty(t, rootCmd.Long)
}

func TestRootCmd_HasSubcommands(t *testing.T) {
	commands := rootCmd.Commands()

	var names []string
	for _, cmd := range commands {
		names = append(names, cmd.Name())
	}

	assert.Contains(t, names, "add")
}

func TestRootCommand_HasIngestSubcommand(t *testing.T) {
	commands := rootCmd.Commands()

	var names []string
	for _, cmd := range commands {
		names = append(names, cmd.Name())
	}

	assert.Contains(t, names, "ingest")
}

func TestRootCommand_HasDiscoverSubcommand(t *testing.T) {
	commands := rootCmd.Commands()

	var names []string
	for _, cmd := range commands {
		names = append(names, cmd.Name())
	}

	assert.Contains(t, names, "discover")
}
