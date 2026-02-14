package db

import (
	"github.com/asteroid-belt/skulto/internal/models"
)

// AddInstallation records a new skill installation location.
// Uses upsert to handle reinstallation of previously installed skills.
func (db *DB) AddInstallation(installation *models.SkillInstallation) error {
	installation.ID = installation.GenerateID()
	// Use Save which does an upsert - inserts if not exists, updates if exists
	return db.Save(installation).Error
}

// RemoveInstallation removes a skill installation record.
func (db *DB) RemoveInstallation(skillID, platform, scope, basePath string) error {
	return db.Where(
		"skill_id = ? AND platform = ? AND scope = ? AND base_path = ?",
		skillID, platform, scope, basePath,
	).Delete(&models.SkillInstallation{}).Error
}

// GetInstallations returns all installation locations for a skill.
func (db *DB) GetInstallations(skillID string) ([]models.SkillInstallation, error) {
	var installations []models.SkillInstallation
	err := db.Where("skill_id = ?", skillID).Find(&installations).Error
	return installations, err
}

// RemoveAllInstallations removes all installation records for a skill.
func (db *DB) RemoveAllInstallations(skillID string) error {
	return db.Where("skill_id = ?", skillID).Delete(&models.SkillInstallation{}).Error
}

// IsInstalledAt checks if a skill is installed at a specific location.
func (db *DB) IsInstalledAt(skillID, platform, scope, basePath string) (bool, error) {
	var count int64
	err := db.Model(&models.SkillInstallation{}).Where(
		"skill_id = ? AND platform = ? AND scope = ? AND base_path = ?",
		skillID, platform, scope, basePath,
	).Count(&count).Error
	return count > 0, err
}

// GetAllInstallationsForPlatform returns all installations for a specific platform.
func (db *DB) GetAllInstallationsForPlatform(platform string) ([]models.SkillInstallation, error) {
	var installations []models.SkillInstallation
	err := db.Where("platform = ?", platform).Find(&installations).Error
	return installations, err
}

// GetAllInstallations returns all recorded skill installations.
// Used during reset to know which symlinks to remove.
func (db *DB) GetAllInstallations() ([]models.SkillInstallation, error) {
	var installations []models.SkillInstallation
	err := db.Find(&installations).Error
	return installations, err
}

// GetProjectInstallations returns all project-scope installations for a specific base path.
// Used by the manifest save command to find which skills are installed for the current project.
func (db *DB) GetProjectInstallations(basePath string) ([]models.SkillInstallation, error) {
	var installations []models.SkillInstallation
	err := db.Where("scope = ? AND base_path = ?", "project", basePath).
		Find(&installations).Error
	return installations, err
}

// GetLastInstallLocations returns the platform+scope pairs from the most recent
// install event. An "install event" is a group of installations sharing the same
// installed_at timestamp (i.e. all locations chosen in one dialog confirmation).
func (db *DB) GetLastInstallLocations() ([]models.SkillInstallation, error) {
	// Find the most recent installed_at timestamp
	var latest models.SkillInstallation
	if err := db.Order("installed_at DESC").First(&latest).Error; err != nil {
		return nil, err
	}
	// Return all installations with that same timestamp
	var installations []models.SkillInstallation
	err := db.Where("installed_at = ?", latest.InstalledAt).Find(&installations).Error
	return installations, err
}
