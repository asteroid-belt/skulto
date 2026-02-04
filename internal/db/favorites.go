package db

import (
	"github.com/asteroid-belt/skulto/internal/models"
)

// AddInstalled marks a skill as installed.
// Uses FirstOrCreate for idempotency - safe to call multiple times.
func (db *DB) AddInstalled(skillID string) error {
	return db.Transaction(func(tx *DB) error {
		// Use FirstOrCreate for upsert semantics - handles duplicates gracefully
		installed := models.Installed{SkillID: skillID}
		if err := tx.Where("skill_id = ?", skillID).FirstOrCreate(&installed).Error; err != nil {
			return err
		}

		// Update skill's is_installed flag
		return tx.Model(&models.Skill{}).Where("id = ?", skillID).
			Update("is_installed", true).Error
	})
}

// RemoveInstalled removes a skill from installed.
func (db *DB) RemoveInstalled(skillID string) error {
	return db.Transaction(func(tx *DB) error {
		// Delete installed record
		if err := tx.Delete(&models.Installed{}, "skill_id = ?", skillID).Error; err != nil {
			return err
		}

		// Update skill's is_installed flag
		return tx.Model(&models.Skill{}).Where("id = ?", skillID).
			Update("is_installed", false).Error
	})
}

// SetInstalled sets the installed status of a skill.
func (db *DB) SetInstalled(skillID string, isInstalled bool) error {
	if isInstalled {
		return db.AddInstalled(skillID)
	}
	return db.RemoveInstalled(skillID)
}

// IsInstalled checks if a skill is marked as installed.
func (db *DB) IsInstalled(skillID string) (bool, error) {
	var count int64
	err := db.Model(&models.Installed{}).Where("skill_id = ?", skillID).Count(&count).Error
	return count > 0, err
}

// GetInstalled returns all installed skills.
// Uses skill_installations as the source of truth - a skill is "installed"
// if it has at least one installation record (symlink on disk).
func (db *DB) GetInstalled() ([]models.Skill, error) {
	var skills []models.Skill
	err := db.Preload("Tags").
		Joins("JOIN skill_installations si ON skills.id = si.skill_id").
		Group("skills.id").
		Order("MAX(si.installed_at) DESC").
		Find(&skills).Error
	return skills, err
}

// HasInstallations checks if a skill has any installation records.
// This is the source of truth for whether a skill is "installed".
func (db *DB) HasInstallations(skillID string) (bool, error) {
	var count int64
	err := db.Model(&models.SkillInstallation{}).Where("skill_id = ?", skillID).Count(&count).Error
	return count > 0, err
}

// GetInstalledWithNotes returns installed skills with their notes.
func (db *DB) GetInstalledWithNotes() ([]models.Installed, error) {
	var installed []models.Installed
	err := db.Preload("Skill").Preload("Skill.Tags").
		Order("added_at DESC").
		Find(&installed).Error
	return installed, err
}

// UpdateInstalledNotes updates the notes for an installed skill.
func (db *DB) UpdateInstalledNotes(skillID, notes string) error {
	return db.Model(&models.Installed{}).
		Where("skill_id = ?", skillID).
		Update("notes", notes).Error
}

// CountInstalled returns the number of installed skills.
// Uses skill_installations as the source of truth.
func (db *DB) CountInstalled() (int64, error) {
	var count int64
	err := db.Model(&models.SkillInstallation{}).
		Select("COUNT(DISTINCT skill_id)").
		Scan(&count).Error
	return count, err
}
