package prompts

import (
	"testing"

	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/stretchr/testify/assert"
)

func TestBuildScopeOptions(t *testing.T) {
	options := BuildScopeOptions()

	assert.Len(t, options, 2)

	// Check global option
	assert.Equal(t, string(installer.ScopeGlobal), options[0].Value)
	assert.Contains(t, options[0].Key, "Global")

	// Check project option
	assert.Equal(t, string(installer.ScopeProject), options[1].Value)
	assert.Contains(t, options[1].Key, "Project")
}

func TestParseScopeStrings(t *testing.T) {
	t.Run("single scope", func(t *testing.T) {
		scopes := ParseScopeStrings([]string{"global"})
		assert.Len(t, scopes, 1)
		assert.Equal(t, installer.ScopeGlobal, scopes[0])
	})

	t.Run("multiple scopes", func(t *testing.T) {
		scopes := ParseScopeStrings([]string{"global", "project"})
		assert.Len(t, scopes, 2)
		assert.Equal(t, installer.ScopeGlobal, scopes[0])
		assert.Equal(t, installer.ScopeProject, scopes[1])
	})

	t.Run("invalid scope ignored", func(t *testing.T) {
		scopes := ParseScopeStrings([]string{"global", "invalid", "project"})
		assert.Len(t, scopes, 2)
	})
}
