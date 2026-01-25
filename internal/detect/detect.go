// Package detect provides AI tool detection for common developer environments.
package detect

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/adrg/xdg"
	"github.com/asteroid-belt/skulto/internal/installer"
)

// DetectionResult holds information about a detected AI tool.
type DetectionResult struct {
	Platform         installer.Platform
	Detected         bool
	CommandPath      string   // Path to command if found in PATH
	ProjectDir       string   // Project-level directory if found
	ConfigDir        string   // User config directory if found
	InstallLocations []string // All detected installation locations
}

// DetectAll detects all supported AI tools on the system.
func DetectAll() []DetectionResult {
	return []DetectionResult{
		DetectClaude(),
		DetectCursor(),
		DetectCopilot(),
		DetectCodex(),
		DetectOpenCode(),
		DetectWindsurf(),
	}
}

// DetectClaude detects Claude Code installation.
func DetectClaude() DetectionResult {
	result := DetectionResult{Platform: installer.PlatformClaude}

	// Check command in PATH
	if path, err := exec.LookPath("claude"); err == nil {
		result.Detected = true
		result.CommandPath = path
		result.InstallLocations = append(result.InstallLocations, path)
	}

	// Check .claude directory in current project
	if _, err := os.Stat(".claude"); err == nil {
		result.Detected = true
		result.ProjectDir = ".claude"
		if !contains(result.InstallLocations, ".claude") {
			result.InstallLocations = append(result.InstallLocations, ".claude")
		}
	}

	// Check user config directory (XDG_CONFIG_HOME)
	claudeConfig := filepath.Join(xdg.ConfigHome, "claude")
	if _, err := os.Stat(claudeConfig); err == nil {
		result.Detected = true
		result.ConfigDir = claudeConfig
		if !contains(result.InstallLocations, claudeConfig) {
			result.InstallLocations = append(result.InstallLocations, claudeConfig)
		}
	}

	return result
}

// DetectCursor detects Cursor IDE installation.
func DetectCursor() DetectionResult {
	result := DetectionResult{Platform: installer.PlatformCursor}

	// Check command
	if path, err := exec.LookPath("cursor"); err == nil {
		result.Detected = true
		result.CommandPath = path
		result.InstallLocations = append(result.InstallLocations, path)
	}

	// Check .cursor directory in project
	if _, err := os.Stat(".cursor"); err == nil {
		result.Detected = true
		result.ProjectDir = ".cursor"
		if !contains(result.InstallLocations, ".cursor") {
			result.InstallLocations = append(result.InstallLocations, ".cursor")
		}
	}

	// Platform-specific checks
	switch runtime.GOOS {
	case "darwin":
		cursors := []string{
			"/Applications/Cursor.app",
			filepath.Join(os.Getenv("HOME"), "Applications/Cursor.app"),
		}
		for _, path := range cursors {
			if _, err := os.Stat(path); err == nil {
				result.Detected = true
				if !contains(result.InstallLocations, path) {
					result.InstallLocations = append(result.InstallLocations, path)
				}
			}
		}
	case "linux":
		linuxPaths := []string{
			filepath.Join(os.Getenv("HOME"), ".cursor"),
			"/opt/Cursor",
		}
		for _, path := range linuxPaths {
			if _, err := os.Stat(path); err == nil {
				result.Detected = true
				if !contains(result.InstallLocations, path) {
					result.InstallLocations = append(result.InstallLocations, path)
				}
			}
		}
	}

	return result
}

// DetectCopilot detects GitHub Copilot installation.
func DetectCopilot() DetectionResult {
	result := DetectionResult{Platform: installer.PlatformCopilot}

	// Check project-level config
	if _, err := os.Stat(".github/copilot-instructions.md"); err == nil {
		result.Detected = true
		result.ProjectDir = ".github"
		if !contains(result.InstallLocations, ".github/copilot-instructions.md") {
			result.InstallLocations = append(result.InstallLocations, ".github/copilot-instructions.md")
		}
	}

	if _, err := os.Stat(filepath.Join(".github", "copilot")); err == nil {
		result.Detected = true
		if result.ProjectDir == "" {
			result.ProjectDir = ".github/copilot"
		}
		if !contains(result.InstallLocations, ".github/copilot") {
			result.InstallLocations = append(result.InstallLocations, ".github/copilot")
		}
	}

	// Check VS Code extensions
	home := os.Getenv("HOME")
	vscodeExtensions := filepath.Join(home, ".vscode", "extensions")

	if entries, err := os.ReadDir(vscodeExtensions); err == nil {
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "github.copilot") {
				result.Detected = true
				fullPath := filepath.Join(vscodeExtensions, entry.Name())
				if !contains(result.InstallLocations, fullPath) {
					result.InstallLocations = append(result.InstallLocations, fullPath)
				}
			}
		}
	}

	return result
}

// DetectCodex detects OpenAI Codex installation.
func DetectCodex() DetectionResult {
	result := DetectionResult{Platform: installer.PlatformCodex}

	// Check command
	if path, err := exec.LookPath("codex"); err == nil {
		result.Detected = true
		result.CommandPath = path
		result.InstallLocations = append(result.InstallLocations, path)
	}

	// Check .codex directory in project
	if _, err := os.Stat(".codex"); err == nil {
		result.Detected = true
		result.ProjectDir = ".codex"
		if !contains(result.InstallLocations, ".codex") {
			result.InstallLocations = append(result.InstallLocations, ".codex")
		}
	}

	// Check user config
	home := os.Getenv("HOME")
	codexPaths := []string{
		filepath.Join(home, ".codex"),
		filepath.Join(xdg.ConfigHome, "codex"),
	}

	for _, path := range codexPaths {
		if _, err := os.Stat(path); err == nil {
			result.Detected = true
			if result.ConfigDir == "" {
				result.ConfigDir = path
			}
			if !contains(result.InstallLocations, path) {
				result.InstallLocations = append(result.InstallLocations, path)
			}
		}
	}

	return result
}

// DetectOpenCode detects OpenCode installation.
func DetectOpenCode() DetectionResult {
	result := DetectionResult{Platform: installer.PlatformOpenCode}

	// Check command
	if path, err := exec.LookPath("opencode"); err == nil {
		result.Detected = true
		result.CommandPath = path
		result.InstallLocations = append(result.InstallLocations, path)
	}

	// Check .opencode directory in project
	if _, err := os.Stat(".opencode"); err == nil {
		result.Detected = true
		result.ProjectDir = ".opencode"
		if !contains(result.InstallLocations, ".opencode") {
			result.InstallLocations = append(result.InstallLocations, ".opencode")
		}
	}

	// Check user config
	home := os.Getenv("HOME")
	opencodePaths := []string{
		filepath.Join(home, ".opencode"),
		filepath.Join(xdg.ConfigHome, "opencode"),
	}

	for _, path := range opencodePaths {
		if _, err := os.Stat(path); err == nil {
			result.Detected = true
			if result.ConfigDir == "" {
				result.ConfigDir = path
			}
			if !contains(result.InstallLocations, path) {
				result.InstallLocations = append(result.InstallLocations, path)
			}
		}
	}

	return result
}

// DetectWindsurf detects Windsurf installation.
func DetectWindsurf() DetectionResult {
	result := DetectionResult{Platform: installer.PlatformWindsurf}

	// Check command
	if path, err := exec.LookPath("windsurf"); err == nil {
		result.Detected = true
		result.CommandPath = path
		result.InstallLocations = append(result.InstallLocations, path)
	}

	// Check .windsurf directory in project
	if _, err := os.Stat(".windsurf"); err == nil {
		result.Detected = true
		result.ProjectDir = ".windsurf"
		if !contains(result.InstallLocations, ".windsurf") {
			result.InstallLocations = append(result.InstallLocations, ".windsurf")
		}
	}

	// Check user config
	home := os.Getenv("HOME")
	windsurfPaths := []string{
		filepath.Join(home, ".windsurf"),
		filepath.Join(xdg.ConfigHome, "windsurf"),
	}

	for _, path := range windsurfPaths {
		if _, err := os.Stat(path); err == nil {
			result.Detected = true
			if result.ConfigDir == "" {
				result.ConfigDir = path
			}
			if !contains(result.InstallLocations, path) {
				result.InstallLocations = append(result.InstallLocations, path)
			}
		}
	}

	return result
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
