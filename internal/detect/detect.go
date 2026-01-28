// Package detect provides AI tool detection for common developer environments.
package detect

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/asteroid-belt/skulto/internal/installer"
)

// DetectionResult holds information about a detected AI tool.
type DetectionResult struct {
	Platform         installer.Platform
	Detected         bool
	CommandPath      string   // Path to command if found in PATH
	ProjectDir       string   // Project-level directory if found
	GlobalDir        string   // Global/home directory if found
	ConfigDir        string   // User config directory if found (backward compat)
	InstallLocations []string // All detected installation locations
}

// DetectAll detects all supported AI tools on the system.
// Uses data-driven approach: iterates the platform registry config.
func DetectAll() []DetectionResult {
	platforms := installer.AllPlatforms()
	results := make([]DetectionResult, 0, len(platforms))
	for _, p := range platforms {
		results = append(results, DetectPlatform(p))
	}
	return results
}

// DetectPlatform detects a single platform using its registry config.
func DetectPlatform(p installer.Platform) DetectionResult {
	info := p.Info()
	globalPath := ""
	if info.GlobalDir != "" {
		globalPath = expandHomePath(info.GlobalDir)
	}
	result := detectWithPaths(info.Command, p, info.ProjectDir, globalPath, info.PlatformSpecificPaths)
	result.Platform = p
	return result
}

// detectWithPaths is the core detection logic, testable with arbitrary paths.
// command: CLI command name to check in PATH
// projectDir: project-level directory to check relative to CWD
// globalDir: absolute path to global directory to check
// platformSpecificPaths: additional OS-specific paths to check
func detectWithPaths(command string, platform installer.Platform, projectDir string, globalDir string, platformSpecificPaths []string) DetectionResult {
	result := DetectionResult{Platform: platform}

	// 1. Check command in PATH
	if command != "" {
		if path, err := exec.LookPath(command); err == nil {
			result.Detected = true
			result.CommandPath = path
			result.InstallLocations = append(result.InstallLocations, path)
		}
	}

	// 2. Check project-level directory
	if projectDir != "" {
		if _, err := os.Stat(projectDir); err == nil {
			result.Detected = true
			result.ProjectDir = projectDir
			if !contains(result.InstallLocations, projectDir) {
				result.InstallLocations = append(result.InstallLocations, projectDir)
			}
		}
	}

	// 3. Check global/home directory
	if globalDir != "" {
		if _, err := os.Stat(globalDir); err == nil {
			result.Detected = true
			result.GlobalDir = globalDir
			result.ConfigDir = globalDir // backward compat
			if !contains(result.InstallLocations, globalDir) {
				result.InstallLocations = append(result.InstallLocations, globalDir)
			}
		}
	}

	// 4. Check platform-specific paths
	for _, path := range platformSpecificPaths {
		expanded := os.ExpandEnv(path)
		if _, err := os.Stat(expanded); err == nil {
			result.Detected = true
			if !contains(result.InstallLocations, expanded) {
				result.InstallLocations = append(result.InstallLocations, expanded)
			}
		}
	}

	return result
}

// expandHomePath replaces ~ with the home directory.
func expandHomePath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

// contains checks if a string slice contains a value.
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}
