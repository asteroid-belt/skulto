package db

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/asteroid-belt/skulto/internal/models"
)

// CreateSource creates a new source.
func (db *DB) CreateSource(source *models.Source) error {
	return db.Create(source).Error
}

// UpsertSource creates or updates a source.
func (db *DB) UpsertSource(source *models.Source) error {
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(source).Error
}

// GetSource retrieves a source by ID.
func (db *DB) GetSource(id string) (*models.Source, error) {
	var source models.Source
	err := db.First(&source, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &source, nil
}

// GetSourceWithSkills retrieves a source with its skills.
func (db *DB) GetSourceWithSkills(id string) (*models.Source, error) {
	var source models.Source
	err := db.Preload("Skills").First(&source, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &source, nil
}

// DeleteSource soft-deletes a source.
func (db *DB) DeleteSource(id string) error {
	return db.Delete(&models.Source{}, "id = ?", id).Error
}

// HardDeleteSource permanently deletes a source.
func (db *DB) HardDeleteSource(id string) error {
	return db.Unscoped().Delete(&models.Source{}, "id = ?", id).Error
}

// ListSources returns all sources sorted by priority.
func (db *DB) ListSources() ([]models.Source, error) {
	var sources []models.Source
	err := db.Order("priority DESC, updated_at DESC").Find(&sources).Error
	return sources, err
}

// ListSourcesByPriority returns sources with priority >= minPriority.
func (db *DB) ListSourcesByPriority(minPriority int) ([]models.Source, error) {
	var sources []models.Source
	err := db.Where("priority >= ?", minPriority).
		Order("priority DESC").
		Find(&sources).Error
	return sources, err
}

// GetOfficialSources returns official sources.
func (db *DB) GetOfficialSources() ([]models.Source, error) {
	var sources []models.Source
	err := db.Where("is_official = ?", true).
		Order("priority DESC").
		Find(&sources).Error
	return sources, err
}

// GetCuratedSources returns curated sources.
func (db *DB) GetCuratedSources() ([]models.Source, error) {
	var sources []models.Source
	err := db.Where("is_curated = ?", true).
		Order("priority DESC").
		Find(&sources).Error
	return sources, err
}

// UpdateSourceSkillCount updates the skill count for a source.
func (db *DB) UpdateSourceSkillCount(sourceID string) error {
	var count int64
	if err := db.Model(&models.Skill{}).Where("source_id = ?", sourceID).Count(&count).Error; err != nil {
		return err
	}
	return db.Model(&models.Source{}).Where("id = ?", sourceID).Update("skill_count", count).Error
}
