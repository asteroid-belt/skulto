package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIngestCommand_Help(t *testing.T) {
	cmd := newIngestCmd()
	assert.Equal(t, "ingest [skill-name]", cmd.Use)
	assert.Contains(t, cmd.Short, "Import")
}

func TestIngestCommand_HasFlags(t *testing.T) {
	cmd := newIngestCmd()
	allFlag := cmd.Flags().Lookup("all")
	projectFlag := cmd.Flags().Lookup("project")
	globalFlag := cmd.Flags().Lookup("global")
	assert.NotNil(t, allFlag)
	assert.NotNil(t, projectFlag)
	assert.NotNil(t, globalFlag)
}

func TestIngestCommand_FlagsAreBooleans(t *testing.T) {
	cmd := newIngestCmd()

	allFlag := cmd.Flags().Lookup("all")
	assert.Equal(t, "bool", allFlag.Value.Type())

	projectFlag := cmd.Flags().Lookup("project")
	assert.Equal(t, "bool", projectFlag.Value.Type())

	globalFlag := cmd.Flags().Lookup("global")
	assert.Equal(t, "bool", globalFlag.Value.Type())
}

func TestIngestCommand_AcceptsSkillName(t *testing.T) {
	cmd := newIngestCmd()
	// Should accept 0-1 args (name is optional with --all)
	assert.NoError(t, cmd.Args(cmd, []string{}))
	assert.NoError(t, cmd.Args(cmd, []string{"my-skill"}))
}

func TestIngestCommand_RejectsMultipleArgs(t *testing.T) {
	cmd := newIngestCmd()
	// Should reject more than 1 arg
	assert.Error(t, cmd.Args(cmd, []string{"skill1", "skill2"}))
}

func TestIngestCommand_LongDescription(t *testing.T) {
	cmd := newIngestCmd()
	assert.Contains(t, cmd.Long, "discovered")
	assert.Contains(t, cmd.Long, "skulto discover")
}

func TestIngestCommand_HasRunE(t *testing.T) {
	cmd := newIngestCmd()
	assert.NotNil(t, cmd.RunE)
}
