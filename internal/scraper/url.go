package scraper

import (
	"fmt"
	"strings"

	"github.com/asteroid-belt/skulto/internal/models"
)

// ParseRepositoryURL parses a repository URL in various formats and returns a Source struct.
// Supported formats:
// - owner/repo
// - https://github.com/owner/repo
// - https://github.com/owner/repo.git
// - git@github.com:owner/repo.git
func ParseRepositoryURL(urlStr string) (*models.Source, error) {
	urlStr = strings.TrimSpace(urlStr)
	if urlStr == "" {
		return nil, fmt.Errorf("repository URL cannot be empty")
	}

	var owner, repo string

	// Handle short format: owner/repo
	if !strings.Contains(urlStr, "://") && !strings.Contains(urlStr, "git@") {
		parts := strings.Split(urlStr, "/")
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			owner = parts[0]
			repo = strings.TrimSuffix(parts[1], ".git")
		} else {
			return nil, fmt.Errorf("invalid repository format: expected 'owner/repo', got '%s'", urlStr)
		}
	} else if strings.HasPrefix(urlStr, "https://github.com/") || strings.HasPrefix(urlStr, "http://github.com/") {
		// Handle HTTPS URL: https://github.com/owner/repo[.git]
		// Remove protocol and domain
		parts := strings.TrimPrefix(urlStr, "https://")
		parts = strings.TrimPrefix(parts, "http://")
		parts = strings.TrimPrefix(parts, "github.com/")

		pathParts := strings.Split(parts, "/")
		if len(pathParts) >= 2 {
			owner = pathParts[0]
			repo = strings.TrimSuffix(pathParts[1], ".git")
		} else {
			return nil, fmt.Errorf("invalid GitHub HTTPS URL: %s", urlStr)
		}
	} else if strings.HasPrefix(urlStr, "git@github.com:") {
		// Handle SSH URL: git@github.com:owner/repo[.git]
		parts := strings.TrimPrefix(urlStr, "git@github.com:")
		pathParts := strings.Split(parts, "/")
		if len(pathParts) >= 2 {
			owner = pathParts[0]
			repo = strings.TrimSuffix(pathParts[1], ".git")
		} else {
			return nil, fmt.Errorf("invalid GitHub SSH URL: %s", urlStr)
		}
	} else {
		return nil, fmt.Errorf("unsupported repository URL format: %s", urlStr)
	}

	// Validate parsed components
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("could not parse owner and repo from URL: %s", urlStr)
	}

	// Validate characters (alphanumeric, hyphens, underscores)
	if !isValidGitHubName(owner) || !isValidGitHubName(repo) {
		return nil, fmt.Errorf("invalid owner or repo name: owner=%s, repo=%s", owner, repo)
	}

	// Create Source struct with sensible defaults for user-added sources
	sourceID := owner + "/" + repo
	source := &models.Source{
		ID:       sourceID,
		Owner:    owner,
		Repo:     repo,
		FullName: sourceID,
		URL:      fmt.Sprintf("https://github.com/%s/%s", owner, repo),
		CloneURL: fmt.Sprintf("https://github.com/%s/%s.git", owner, repo),

		// Default metadata for user-added sources
		Priority:      5, // Medium priority
		IsOfficial:    false,
		IsCurated:     false,
		DefaultBranch: "main",
	}

	return source, nil
}

// isValidGitHubName validates a GitHub username or repository name.
// GitHub names must:
// - Start and end with alphanumeric character
// - Contain only alphanumeric characters, hyphens, and underscores
// - Be 1-39 characters long
func isValidGitHubName(name string) bool {
	if len(name) == 0 || len(name) > 39 {
		return false
	}

	// Check first and last character are alphanumeric
	if !isAlphanumeric(rune(name[0])) || !isAlphanumeric(rune(name[len(name)-1])) {
		return false
	}

	// Check all characters are alphanumeric, hyphen, or underscore
	for _, ch := range name {
		if !isAlphanumeric(ch) && ch != '-' && ch != '_' {
			return false
		}
	}

	return true
}

// isAlphanumeric checks if a rune is alphanumeric.
func isAlphanumeric(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9')
}
