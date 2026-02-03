package db

import (
	"time"

	"github.com/asteroid-belt/skulto/internal/models"
	"gorm.io/gorm/clause"
)

// UpsertDiscoveredSkill inserts or updates a discovered skill.
func (db *DB) UpsertDiscoveredSkill(ds *models.DiscoveredSkill) error {
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"discovered_at"}),
	}).Create(ds).Error
}

// ListDiscoveredSkills returns all discovered skills.
func (db *DB) ListDiscoveredSkills() ([]models.DiscoveredSkill, error) {
	var skills []models.DiscoveredSkill
	err := db.Order("discovered_at DESC").Find(&skills).Error
	return skills, err
}

// ListUnnotifiedDiscoveredSkills returns discoveries that haven't been shown to user.
func (db *DB) ListUnnotifiedDiscoveredSkills() ([]models.DiscoveredSkill, error) {
	var skills []models.DiscoveredSkill
	err := db.Where("notified_at IS NULL").Order("discovered_at DESC").Find(&skills).Error
	return skills, err
}

// ListDiscoveredSkillsByScope returns discoveries filtered by scope.
func (db *DB) ListDiscoveredSkillsByScope(scope string) ([]models.DiscoveredSkill, error) {
	var skills []models.DiscoveredSkill
	err := db.Where("scope = ?", scope).Order("discovered_at DESC").Find(&skills).Error
	return skills, err
}

// MarkDiscoveredSkillsNotified sets NotifiedAt for the given skill IDs.
func (db *DB) MarkDiscoveredSkillsNotified(ids []string) error {
	now := time.Now()
	return db.Model(&models.DiscoveredSkill{}).
		Where("id IN ?", ids).
		Update("notified_at", now).Error
}

// DeleteDiscoveredSkill removes a discovered skill by ID.
func (db *DB) DeleteDiscoveredSkill(id string) error {
	return db.Delete(&models.DiscoveredSkill{}, "id = ?", id).Error
}

// GetDiscoveredSkillByPath finds a discovered skill by its path.
func (db *DB) GetDiscoveredSkillByPath(path string) (*models.DiscoveredSkill, error) {
	var skill models.DiscoveredSkill
	err := db.Where("path = ?", path).First(&skill).Error
	if err != nil {
		return nil, err
	}
	return &skill, nil
}

// GetDiscoveredSkillByName finds a discovered skill by name and scope.
func (db *DB) GetDiscoveredSkillByName(name, scope string) (*models.DiscoveredSkill, error) {
	var skill models.DiscoveredSkill
	err := db.Where("name = ? AND scope = ?", name, scope).First(&skill).Error
	if err != nil {
		return nil, err
	}
	return &skill, nil
}

// CountDiscoveredSkills returns the total count of discovered skills.
func (db *DB) CountDiscoveredSkills() (int64, error) {
	var count int64
	err := db.Model(&models.DiscoveredSkill{}).Count(&count).Error
	return count, err
}

// CleanupStaleDiscoveries removes discoveries whose paths no longer exist or are now symlinks.
func (db *DB) CleanupStaleDiscoveries(pathChecker func(string) bool) error {
	skills, err := db.ListDiscoveredSkills()
	if err != nil {
		return err
	}

	for _, skill := range skills {
		if !pathChecker(skill.Path) {
			if err := db.DeleteDiscoveredSkill(skill.ID); err != nil {
				return err
			}
		}
	}
	return nil
}
