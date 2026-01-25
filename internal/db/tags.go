package db

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/asteroid-belt/skulto/internal/models"
)

// CreateTag creates a new tag.
func (db *DB) CreateTag(tag *models.Tag) error {
	return db.Create(tag).Error
}

// UpsertTag creates or updates a tag.
func (db *DB) UpsertTag(tag *models.Tag) error {
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(tag).Error
}

// GetTag retrieves a tag by ID.
func (db *DB) GetTag(id string) (*models.Tag, error) {
	var tag models.Tag
	err := db.First(&tag, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &tag, nil
}

// GetTagBySlug retrieves a tag by slug.
func (db *DB) GetTagBySlug(slug string) (*models.Tag, error) {
	var tag models.Tag
	err := db.First(&tag, "slug = ?", slug).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &tag, nil
}

// ListTags returns all tags, optionally filtered by category.
func (db *DB) ListTags(category string) ([]models.Tag, error) {
	var tags []models.Tag
	query := db.Order("priority DESC, count DESC, name ASC")

	if category != "" {
		query = query.Where("category = ?", category)
	}

	err := query.Find(&tags).Error
	return tags, err
}

// ListTagsByCategory returns tags in a specific category.
func (db *DB) ListTagsByCategory(category string) ([]models.Tag, error) {
	return db.ListTags(category)
}

// GetTopTags returns the most popular tags.
func (db *DB) GetTopTags(limit int) ([]models.Tag, error) {
	var tags []models.Tag
	err := db.Order("priority DESC, count DESC, name ASC").Limit(limit).Find(&tags).Error
	return tags, err
}

// GetTagsForSkill returns all tags for a skill.
func (db *DB) GetTagsForSkill(skillID string) ([]models.Tag, error) {
	var tags []models.Tag
	err := db.Joins("JOIN skill_tags st ON tags.id = st.tag_id").
		Where("st.skill_id = ?", skillID).
		Find(&tags).Error
	return tags, err
}

// AddTagToSkill associates a tag with a skill.
func (db *DB) AddTagToSkill(skillID, tagID string) error {
	return db.Exec("INSERT OR IGNORE INTO skill_tags (skill_id, tag_id) VALUES (?, ?)", skillID, tagID).Error
}

// RemoveTagFromSkill removes a tag association from a skill.
func (db *DB) RemoveTagFromSkill(skillID, tagID string) error {
	return db.Exec("DELETE FROM skill_tags WHERE skill_id = ? AND tag_id = ?", skillID, tagID).Error
}

// UpdateTagCounts recalculates tag counts based on skill associations.
func (db *DB) UpdateTagCounts() error {
	return db.Exec(`
		UPDATE tags SET count = (
			SELECT COUNT(*) FROM skill_tags WHERE skill_tags.tag_id = tags.id
		)
	`).Error
}

// DeleteUnusedTags removes tags with no skill associations.
func (db *DB) DeleteUnusedTags() error {
	return db.Where("count = 0").Delete(&models.Tag{}).Error
}

// EnsureMineTag creates or updates the "mine" tag with correct priority.
func (db *DB) EnsureMineTag() error {
	mineTag := models.MineTag()
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"priority", "color", "category"}),
	}).Create(&mineTag).Error
}

// RetagAllSkills re-extracts tags for all skills in the database.
// This should be called after updating the tagging algorithm.
// The extractFn parameter is the tag extraction function (e.g., scraper.ExtractTags).
func (db *DB) RetagAllSkills(extractFn func(string) []models.Tag) (int, error) {
	var skills []models.Skill
	if err := db.Find(&skills).Error; err != nil {
		return 0, err
	}

	updated := 0
	for _, skill := range skills {
		// Extract new tags from content
		newTags := extractFn(skill.Content)

		// Update skill with new tags
		if err := db.UpsertSkillWithTags(&skill, newTags); err != nil {
			// Log but continue
			continue
		}
		updated++
	}

	// Update tag counts after retagging
	_ = db.UpdateTagCounts()

	return updated, nil
}
