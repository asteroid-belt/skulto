package db

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/asteroid-belt/skulto/internal/models"
)

// SearchResult wraps a skill with its search rank.
type SearchResult struct {
	models.Skill
	Rank float64 `gorm:"column:rank"`
}

// CreateSkill creates a new skill.
func (db *DB) CreateSkill(skill *models.Skill) error {
	skill.IndexedAt = time.Now()
	return db.Create(skill).Error
}

// UpdateSkill updates an existing skill.
func (db *DB) UpdateSkill(skill *models.Skill) error {
	skill.IndexedAt = time.Now()
	return db.Save(skill).Error
}

// UpsertSkill creates or updates a skill.
// Only updates metadata fields - preserves user state (is_installed, viewed_at)
// and security state (security_status, threat_level, etc.).
func (db *DB) UpsertSkill(skill *models.Skill) error {
	skill.IndexedAt = time.Now()
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			// Metadata fields that should be updated on sync
			"slug", "title", "description", "content", "summary",
			"source_id", "file_path",
			"category", "difficulty",
			"stars", "forks", "downloads",
			"embedding_id",
			"version", "license", "author",
			"indexed_at", "last_sync_at",
			"updated_at",
			// NOT updated: is_local, is_installed, viewed_at
			// NOT updated: security_status, threat_level, threat_summary, scanned_at, released_at, content_hash
		}),
	}).Create(skill).Error
}

// UpsertSkillWithTags creates or updates a skill with its tags.
func (db *DB) UpsertSkillWithTags(skill *models.Skill, tags []models.Tag) error {
	return db.Transaction(func(tx *DB) error {
		skill.IndexedAt = time.Now()

		// Check if skill already exists and get its current tags
		var existingSkill models.Skill
		skillExists := tx.Preload("Tags").First(&existingSkill, "id = ?", skill.ID).Error == nil

		// Build map of old tag IDs for comparison
		oldTagIDs := make(map[string]bool)
		if skillExists {
			for _, oldTag := range existingSkill.Tags {
				oldTagIDs[oldTag.ID] = true
			}
		}

		// Upsert the skill - only update metadata fields, preserve user/security state
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"slug", "title", "description", "content", "summary",
				"source_id", "file_path",
				"category", "difficulty",
				"stars", "forks", "downloads",
				"embedding_id",
				"version", "license", "author",
				"indexed_at", "last_sync_at",
				"updated_at",
			}),
		}).Create(skill).Error; err != nil {
			return fmt.Errorf("upsert skill: %w", err)
		}

		// Track new tag IDs
		newTagIDs := make(map[string]bool)

		// Upsert tags
		for i := range tags {
			tag := &tags[i]
			newTagIDs[tag.ID] = true

			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "id"}},
				DoUpdates: clause.AssignmentColumns([]string{"name", "slug", "category", "color"}),
			}).Create(tag).Error; err != nil {
				return fmt.Errorf("upsert tag %s: %w", tag.Name, err)
			}

			// Only increment tag count if this is a new tag for this skill
			if !oldTagIDs[tag.ID] {
				if err := tx.Model(&models.Tag{}).Where("id = ?", tag.ID).
					Update("count", gorm.Expr("count + 1")).Error; err != nil {
					return fmt.Errorf("increment tag count: %w", err)
				}
			}
		}

		// Decrement count for tags that were removed
		if skillExists {
			for tagID := range oldTagIDs {
				if !newTagIDs[tagID] {
					if err := tx.Model(&models.Tag{}).Where("id = ?", tagID).
						Update("count", gorm.Expr("CASE WHEN count > 0 THEN count - 1 ELSE 0 END")).Error; err != nil {
						return fmt.Errorf("decrement tag count: %w", err)
					}
				}
			}
		}

		// Replace skill-tag associations
		if err := tx.Model(skill).Association("Tags").Replace(tags); err != nil {
			return fmt.Errorf("replace tags: %w", err)
		}

		return nil
	})
}

// GetSkill retrieves a skill by ID with its tags, source, and auxiliary files.
// Uses Joins for Source (one-to-one) for efficiency, Preload for Tags and AuxiliaryFiles.
func (db *DB) GetSkill(id string) (*models.Skill, error) {
	var skill models.Skill
	err := db.Joins("Source").
		Preload("Tags").
		Preload("AuxiliaryFiles").
		First(&skill, "skills.id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &skill, nil
}

// GetSkillBySlug retrieves a skill by slug.
func (db *DB) GetSkillBySlug(slug string) (*models.Skill, error) {
	var skill models.Skill
	err := db.Preload("Tags").First(&skill, "slug = ?", slug).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &skill, nil
}

// GetSkillBySlugWithPriority returns the highest priority skill matching a slug.
// CWD skills (cwd-*) take priority over local skills (local-*).
func (db *DB) GetSkillBySlugWithPriority(slug string) (*models.Skill, error) {
	var skill models.Skill

	// Try CWD first (highest priority)
	err := db.Preload("Tags").First(&skill, "id = ?", "cwd-"+slug).Error
	if err == nil {
		return &skill, nil
	}

	// Fall back to local
	err = db.Preload("Tags").First(&skill, "id = ?", "local-"+slug).Error
	if err == nil {
		return &skill, nil
	}

	// Try by slug directly
	err = db.Preload("Tags").First(&skill, "slug = ?", slug).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &skill, nil
}

// DeleteSkill soft-deletes a skill.
func (db *DB) DeleteSkill(id string) error {
	return db.Delete(&models.Skill{}, "id = ?", id).Error
}

// HardDeleteSkill permanently deletes a skill.
func (db *DB) HardDeleteSkill(id string) error {
	return db.Unscoped().Delete(&models.Skill{}, "id = ?", id).Error
}

// GetSkillsBySourceID returns all skills belonging to a source.
func (db *DB) GetSkillsBySourceID(sourceID string) ([]models.Skill, error) {
	var skills []models.Skill
	err := db.Preload("Tags").Where("source_id = ?", sourceID).Find(&skills).Error
	return skills, err
}

// HardDeleteSkillsBySource permanently deletes all skills belonging to a source.
// Returns the number of skills deleted.
func (db *DB) HardDeleteSkillsBySource(sourceID string) (int64, error) {
	// First, remove tag associations for these skills
	subQuery := db.Model(&models.Skill{}).Select("id").Where("source_id = ?", sourceID)
	if err := db.Exec("DELETE FROM skill_tags WHERE skill_id IN (?)", subQuery).Error; err != nil {
		return 0, fmt.Errorf("delete skill tags: %w", err)
	}

	// Then delete the skills themselves
	result := db.Unscoped().Where("source_id = ?", sourceID).Delete(&models.Skill{})
	return result.RowsAffected, result.Error
}

// ListSkills returns paginated skills with their tags and source.
// Uses Joins for Source (one-to-one) for efficiency, Preload for Tags (many-to-many).
func (db *DB) ListSkills(limit, offset int) ([]models.Skill, error) {
	var skills []models.Skill
	err := db.Joins("Source").
		Preload("Tags").
		Order("skills.updated_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&skills).Error
	return skills, err
}

// GetAllSkills returns all skills without pagination.
// Use with caution on large databases.
func (db *DB) GetAllSkills() ([]models.Skill, error) {
	var skills []models.Skill
	err := db.Find(&skills).Error
	return skills, err
}

// Search performs FTS5 full-text search with BM25 ranking.
func (db *DB) Search(query string, limit int) ([]SearchResult, error) {
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}

	// Prepare FTS5 query
	ftsQuery := prepareFTSQuery(query)

	var results []SearchResult
	err := db.Raw(`
		SELECT s.*, bm25(skills_fts, 10.0, 5.0, 1.0, 2.0, 3.0) as rank
		FROM skills s
		JOIN skills_fts fts ON s.rowid = fts.rowid
		WHERE skills_fts MATCH ?
		  AND s.deleted_at IS NULL
		ORDER BY rank
		LIMIT ?
	`, ftsQuery, limit).Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("fts search: %w", err)
	}

	return results, nil
}

// SearchSkills is a simpler interface that returns skills with their tags and source loaded.
// Uses Joins for Source (one-to-one) for efficiency, Preload for Tags (many-to-many).
func (db *DB) SearchSkills(query string, limit int) ([]models.Skill, error) {
	results, err := db.Search(query, limit)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return []models.Skill{}, nil
	}

	// Extract skill IDs to load tags and source efficiently
	skillIDs := make([]string, len(results))
	for i, r := range results {
		skillIDs[i] = r.ID
	}

	// Load skills with tags and source in FTS ranking order
	var skillsWithRelations []models.Skill
	err = db.Joins("Source").
		Preload("Tags").
		Where("skills.id IN ?", skillIDs).
		Find(&skillsWithRelations).Error
	if err != nil {
		return nil, err
	}

	// Create a map for quick lookup
	skillMap := make(map[string]models.Skill, len(skillsWithRelations))
	for _, s := range skillsWithRelations {
		skillMap[s.ID] = s
	}

	// Return skills in original FTS ranking order
	skills := make([]models.Skill, 0, len(results))
	for _, r := range results {
		if s, ok := skillMap[r.ID]; ok {
			skills = append(skills, s)
		}
	}
	return skills, nil
}

// SearchByCategory performs FTS search filtered by category.
func (db *DB) SearchByCategory(query, category string, limit int) ([]SearchResult, error) {
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}

	ftsQuery := prepareFTSQuery(query)

	var results []SearchResult
	err := db.Raw(`
		SELECT s.*, bm25(skills_fts, 10.0, 5.0, 1.0, 2.0, 3.0) as rank
		FROM skills s
		JOIN skills_fts fts ON s.rowid = fts.rowid
		WHERE skills_fts MATCH ? AND s.category = ?
		  AND s.deleted_at IS NULL
		ORDER BY rank
		LIMIT ?
	`, ftsQuery, category, limit).Scan(&results).Error

	return results, err
}

// SearchByTag performs FTS search filtered by tag.
func (db *DB) SearchByTag(query, tagSlug string, limit int) ([]SearchResult, error) {
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}

	ftsQuery := prepareFTSQuery(query)

	var results []SearchResult
	err := db.Raw(`
		SELECT s.*, bm25(skills_fts, 10.0, 5.0, 1.0, 2.0, 3.0) as rank
		FROM skills s
		JOIN skills_fts fts ON s.rowid = fts.rowid
		JOIN skill_tags st ON s.id = st.skill_id
		WHERE skills_fts MATCH ? AND st.tag_id = ?
		  AND s.deleted_at IS NULL
		ORDER BY rank
		LIMIT ?
	`, ftsQuery, tagSlug, limit).Scan(&results).Error

	return results, err
}

// GetTopSkills returns skills sorted by stars.
func (db *DB) GetTopSkills(limit int) ([]models.Skill, error) {
	var skills []models.Skill
	err := db.Preload("Tags").
		Order("stars DESC").
		Limit(limit).
		Find(&skills).Error
	return skills, err
}

// GetRecentSkills returns recently viewed skills.
func (db *DB) GetRecentSkills(limit int) ([]models.Skill, error) {
	var skills []models.Skill
	err := db.Preload("Tags").
		Where("viewed_at IS NOT NULL").
		Order("viewed_at DESC").
		Limit(limit).
		Find(&skills).Error
	return skills, err
}

// RecordSkillView records that a skill was viewed by the user.
func (db *DB) RecordSkillView(skillID string) error {
	now := time.Now()
	return db.Model(&models.Skill{}).
		Where("id = ?", skillID).
		Update("viewed_at", now).Error
}

// GetSkillsByCategory returns skills in a category.
func (db *DB) GetSkillsByCategory(category string, limit, offset int) ([]models.Skill, error) {
	var skills []models.Skill
	err := db.Preload("Tags").
		Where("category = ?", category).
		Order("stars DESC, updated_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&skills).Error
	return skills, err
}

// GetSkillsByTag returns skills with a specific tag.
func (db *DB) GetSkillsByTag(tagSlug string, limit, offset int) ([]models.Skill, error) {
	var skills []models.Skill
	err := db.Preload("Tags").
		Joins("JOIN skill_tags st ON skills.id = st.skill_id").
		Where("st.tag_id = ?", tagSlug).
		Order("skills.stars DESC, skills.updated_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&skills).Error
	return skills, err
}

// prepareFTSQuery prepares a query string for FTS5.
func prepareFTSQuery(query string) string {
	terms := strings.Fields(query)
	if len(terms) == 0 {
		return ""
	}

	var escaped []string
	for _, term := range terms {
		// Remove FTS5 special characters
		term = strings.ReplaceAll(term, "\"", "")
		term = strings.ReplaceAll(term, "'", "")
		term = strings.ReplaceAll(term, "(", "")
		term = strings.ReplaceAll(term, ")", "")
		term = strings.ReplaceAll(term, "*", "")
		term = strings.ReplaceAll(term, ":", "")
		term = strings.ReplaceAll(term, "-", " ")

		if term != "" {
			// Add prefix matching
			escaped = append(escaped, term+"*")
		}
	}

	return strings.Join(escaped, " ")
}

// CountSkillsWithoutEmbedding returns the count of skills needing embedding.
func (db *DB) CountSkillsWithoutEmbedding() (int, error) {
	var count int64
	err := db.Model(&models.Skill{}).
		Where("embedding_id IS NULL OR embedding_id = ''").
		Count(&count).Error
	return int(count), err
}

// GetSkillsWithoutEmbedding returns skills that need embedding.
func (db *DB) GetSkillsWithoutEmbedding(limit int) ([]models.Skill, error) {
	var skills []models.Skill
	err := db.Preload("Tags").
		Where("embedding_id IS NULL OR embedding_id = ''").
		Order("updated_at DESC"). // Index most recent first
		Limit(limit).
		Find(&skills).Error
	return skills, err
}
