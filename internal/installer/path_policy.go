package installer

import "path/filepath"

// PathPolicy describes canonical and deprecated relative paths for a platform/scope pair.
// All paths are relative to the resolved scope base path (home for global, cwd for project).
type PathPolicy struct {
	CanonicalRelativePath   string
	DeprecatedRelativePaths []string
}

type pathPolicyKey struct {
	platform Platform
	scope    InstallScope
}

var pathPolicies = map[pathPolicyKey]PathPolicy{
	{
		platform: PlatformOpenCode,
		scope:    ScopeGlobal,
	}: {
		CanonicalRelativePath: filepath.Join(".config", "opencode", "skills"),
		DeprecatedRelativePaths: []string{
			filepath.Join(".opencode", "skills"),
		},
	},
}

// PathPolicyFor returns the policy for a platform/scope pair if one is registered.
func PathPolicyFor(platform Platform, scope InstallScope) (PathPolicy, bool) {
	policy, ok := pathPolicies[pathPolicyKey{
		platform: platform,
		scope:    scope,
	}]
	return policy, ok
}

func resolveCanonicalSkillsBasePath(platform Platform, scope InstallScope, basePath string) string {
	info := platform.Info()
	defaultRelativePath := info.SkillsPath

	if policy, ok := PathPolicyFor(platform, scope); ok && policy.CanonicalRelativePath != "" {
		return filepath.Join(basePath, policy.CanonicalRelativePath)
	}
	if defaultRelativePath == "" {
		return ""
	}
	return filepath.Join(basePath, defaultRelativePath)
}

func resolveDeprecatedSkillsBasePaths(platform Platform, scope InstallScope, basePath string) []string {
	policy, ok := PathPolicyFor(platform, scope)
	if !ok || len(policy.DeprecatedRelativePaths) == 0 {
		return nil
	}

	paths := make([]string, 0, len(policy.DeprecatedRelativePaths))
	for _, relativePath := range policy.DeprecatedRelativePaths {
		paths = append(paths, filepath.Join(basePath, relativePath))
	}
	return paths
}
