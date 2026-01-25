package vector

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/asteroid-belt/skulto/internal/models"
)

// PrepareContent concatenates skill fields for embedding.
// Title is repeated for emphasis.
func PrepareContent(skill *models.Skill) string {
	var parts []string

	if skill.Title != "" {
		parts = append(parts, skill.Title, skill.Title) // Repeated for emphasis
	}
	if skill.Description != "" {
		parts = append(parts, skill.Description)
	}
	if skill.Summary != "" {
		parts = append(parts, skill.Summary)
	}
	if skill.Content != "" {
		parts = append(parts, skill.Content)
	}
	for _, tag := range skill.Tags {
		parts = append(parts, tag.Name)
	}

	return strings.Join(parts, "\n\n")
}

// ContentHash generates SHA256 hash for cache invalidation.
func ContentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// TruncateToTokens truncates content to approximate token limit.
// Uses ~4 chars per token as rough estimate.
func TruncateToTokens(content string, maxTokens int) string {
	maxChars := maxTokens * 4
	if len(content) <= maxChars {
		return content
	}
	return content[:maxChars]
}
