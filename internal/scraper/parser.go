// Package scraper contains GitHub scraping and skill parsing logic.
package scraper

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

// SkillFile represents a discovered skill file from GitHub.
type SkillFile struct {
	ID       string
	Path     string
	RepoName string
	Owner    string
	Repo     string
	URL      string
	SHA      string
}

// SkillParser parses SKILL.md files and extracts metadata and content.
type SkillParser struct {
	md goldmark.Markdown
}

// NewSkillParser creates a parser with frontmatter support.
func NewSkillParser() *SkillParser {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			meta.Meta,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)

	return &SkillParser{md: md}
}

// Parse extracts skill data from SKILL.md content.
// It handles YAML frontmatter, generates unique IDs and slugs,
// and extracts description from markdown content.
func (p *SkillParser) Parse(content string, source *SkillFile) (*models.Skill, error) {
	var buf bytes.Buffer
	context := parser.NewContext()

	if err := p.md.Convert([]byte(content), &buf, parser.WithContext(context)); err != nil {
		return nil, fmt.Errorf("parse markdown: %w", err)
	}

	// Extract YAML frontmatter metadata
	frontmatter := meta.Get(context)

	skill := &models.Skill{
		ID:       source.ID,
		Content:  content,
		FilePath: source.Path,
		IsLocal:  false,
	}

	// Set SourceID - use a pointer to string since it's nullable in GORM
	sourceID := source.RepoName
	skill.SourceID = &sourceID

	// Parse title from frontmatter or extract from markdown
	if name, ok := frontmatter["name"].(string); ok && name != "" {
		skill.Title = strings.TrimSpace(name)
	} else {
		skill.Title = extractFirstHeading(content)
	}

	// Parse description from frontmatter or extract from content
	if desc, ok := frontmatter["description"].(string); ok && desc != "" {
		skill.Description = strings.TrimSpace(desc)
	} else {
		skill.Description = extractDescription(content)
	}

	// Parse metadata from nested metadata object
	if metadataRaw, ok := frontmatter["metadata"]; ok {
		// goldmark-meta may return interface{} so we need to handle type conversion
		var metadata map[string]interface{}
		switch m := metadataRaw.(type) {
		case map[string]interface{}:
			metadata = m
		case map[interface{}]interface{}:
			// Convert map[interface{}]interface{} to map[string]interface{}
			metadata = make(map[string]interface{})
			for k, v := range m {
				if str, ok := k.(string); ok {
					metadata[str] = v
				}
			}
		}

		if metadata != nil {
			if version, ok := metadata["version"].(string); ok {
				skill.Version = version
			}

			if author, ok := metadata["author"].(string); ok {
				skill.Author = author
			}

			if license, ok := metadata["license"].(string); ok {
				skill.License = license
			}
		}
	}

	// Generate slug from title
	skill.Slug = generateSlug(skill.Title)

	// Generate content hash for embedding cache
	contentHash := sha256.Sum256([]byte(content))
	skill.EmbeddingID = hex.EncodeToString(contentHash[:])[:16]

	return skill, nil
}

// extractFirstHeading finds the first H1 or H2 heading in the markdown.
// Returns "Untitled Skill" if no heading is found.
func extractFirstHeading(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip frontmatter
		if line == "---" {
			continue
		}

		// Look for H1 or H2 headings
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
		if strings.HasPrefix(line, "## ") {
			return strings.TrimPrefix(line, "## ")
		}
	}
	return "Untitled Skill"
}

// extractDescription extracts the first meaningful paragraph after the heading.
// Skips frontmatter, headings, code blocks, and lists.
// Returns up to 200 characters.
func extractDescription(content string) string {
	lines := strings.Split(content, "\n")
	inDescription := false
	var desc []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip frontmatter delimiters
		if line == "---" {
			continue
		}

		// Skip headings and set flag to start collecting description
		if strings.HasPrefix(line, "#") {
			inDescription = true
			continue
		}

		// Collect non-empty description lines
		if inDescription && line != "" {
			desc = append(desc, line)
			// Limit to 3 lines
			if len(desc) >= 3 {
				break
			}
		}

		// Stop at code blocks or lists
		if strings.HasPrefix(line, "```") || strings.HasPrefix(line, "- ") {
			break
		}
	}

	result := strings.Join(desc, " ")
	if len(result) > 200 {
		result = result[:197] + "..."
	}
	return result
}

// generateSlug creates a URL-safe slug from a title.
// Converts to lowercase, removes special characters, replaces spaces with hyphens,
// and removes consecutive hyphens. Maximum 50 characters.
func generateSlug(title string) string {
	slug := strings.ToLower(title)

	// Remove special characters except spaces and hyphens
	slug = regexp.MustCompile(`[^a-z0-9\s\-]`).ReplaceAllString(slug, "")

	// Replace spaces with hyphens
	slug = regexp.MustCompile(`\s+`).ReplaceAllString(slug, "-")

	// Remove consecutive hyphens
	slug = regexp.MustCompile(`-+`).ReplaceAllString(slug, "-")

	// Trim leading/trailing hyphens
	slug = strings.Trim(slug, "-")

	// Limit to 50 characters
	if len(slug) > 50 {
		slug = slug[:50]
	}

	return slug
}

// generateSkillID creates a unique ID for a skill based on repo name and path.
// Uses SHA256 hash truncated to 16 characters.
func generateSkillID(repoName, path string) string {
	h := sha256.New()
	h.Write([]byte(repoName + ":" + path))
	return hex.EncodeToString(h.Sum(nil))[:16]
}
