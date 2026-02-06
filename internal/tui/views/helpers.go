package views

import "github.com/asteroid-belt/skulto/internal/models"

// FilterOutMineTag removes the "mine" tag from a slice of tags.
// If maxTags > 0, limits the result to that many tags.
// If maxTags <= 0, no limit is applied.
func FilterOutMineTag(tags []models.Tag, maxTags int) []models.Tag {
	result := make([]models.Tag, 0, len(tags))
	for _, tag := range tags {
		if tag.ID == "mine" || tag.Slug == "mine" {
			continue
		}
		result = append(result, tag)
		if maxTags > 0 && len(result) >= maxTags {
			break
		}
	}
	return result
}
