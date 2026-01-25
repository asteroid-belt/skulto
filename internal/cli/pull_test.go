package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestPullCmd_Structure(t *testing.T) {
	assert.Equal(t, "pull", pullCmd.Use)
	assert.NotEmpty(t, pullCmd.Short)
	assert.NotEmpty(t, pullCmd.Long)
	assert.Contains(t, pullCmd.Long, "Pull and sync all skill repositories")
	assert.Contains(t, pullCmd.Long, "skulto pull")
}

func TestPullCmd_ArgsValidation(t *testing.T) {
	// Test the cobra.NoArgs validator
	validator := cobra.NoArgs

	// Should pass with no args
	err := validator(pullCmd, []string{})
	assert.NoError(t, err)

	// Should fail with any args
	err = validator(pullCmd, []string{"unexpected"})
	assert.Error(t, err)
}
