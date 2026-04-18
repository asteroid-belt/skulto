package installer

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathPolicyFor_OpenCodeGlobal(t *testing.T) {
	policy, ok := PathPolicyFor(PlatformOpenCode, ScopeGlobal)
	assert.True(t, ok)
	assert.Equal(t, filepath.Join(".config", "opencode", "skills"), policy.CanonicalRelativePath)
	assert.Equal(t, []string{filepath.Join(".opencode", "skills")}, policy.DeprecatedRelativePaths)
}

func TestPathPolicyFor_OpenCodeProject_NotDefined(t *testing.T) {
	_, ok := PathPolicyFor(PlatformOpenCode, ScopeProject)
	assert.False(t, ok)
}

func TestResolveCanonicalSkillsBasePath_UsesPolicyWhenDefined(t *testing.T) {
	basePath := "/tmp/home"
	path := resolveCanonicalSkillsBasePath(PlatformOpenCode, ScopeGlobal, basePath)
	assert.Equal(t, filepath.Join(basePath, ".config", "opencode", "skills"), path)
}

func TestResolveCanonicalSkillsBasePath_FallsBackToPlatformSkillsPath(t *testing.T) {
	basePath := "/tmp/home"
	path := resolveCanonicalSkillsBasePath(PlatformClaude, ScopeGlobal, basePath)
	assert.Equal(t, filepath.Join(basePath, ".claude", "skills"), path)
}

func TestResolveDeprecatedSkillsBasePaths(t *testing.T) {
	basePath := "/tmp/home"

	openCodePaths := resolveDeprecatedSkillsBasePaths(PlatformOpenCode, ScopeGlobal, basePath)
	assert.Equal(t, []string{filepath.Join(basePath, ".opencode", "skills")}, openCodePaths)

	claudePaths := resolveDeprecatedSkillsBasePaths(PlatformClaude, ScopeGlobal, basePath)
	assert.Nil(t, claudePaths)
}
