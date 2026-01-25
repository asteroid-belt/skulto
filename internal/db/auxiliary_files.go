package db

import (
	"time"

	"gorm.io/gorm/clause"

	"github.com/asteroid-belt/skulto/internal/models"
)

// UpsertAuxiliaryFile creates or updates an auxiliary file record.
func (db *DB) UpsertAuxiliaryFile(file *models.AuxiliaryFile) error {
	if file.ID == "" {
		file.ID = file.GenerateID()
	}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(file).Error
}

// GetAuxiliaryFilesForSkill returns all auxiliary files for a skill.
func (db *DB) GetAuxiliaryFilesForSkill(skillID string) ([]models.AuxiliaryFile, error) {
	var files []models.AuxiliaryFile
	err := db.Where("skill_id = ?", skillID).
		Order("dir_type, file_path").
		Find(&files).Error
	return files, err
}

// GetAuxiliaryFilesByDirType returns files of a specific type for a skill.
func (db *DB) GetAuxiliaryFilesByDirType(skillID string, dirType models.AuxiliaryDirType) ([]models.AuxiliaryFile, error) {
	var files []models.AuxiliaryFile
	err := db.Where("skill_id = ? AND dir_type = ?", skillID, dirType).
		Order("file_path").
		Find(&files).Error
	return files, err
}

// GetQuarantinedFiles returns all quarantined auxiliary files.
// Results are ordered by threat severity (CRITICAL > HIGH > MEDIUM > LOW > NONE).
func (db *DB) GetQuarantinedFiles() ([]models.AuxiliaryFile, error) {
	var files []models.AuxiliaryFile
	err := db.Preload("Skill").
		Where("security_status = ?", models.SecurityStatusQuarantined).
		Order("CASE threat_level WHEN 'CRITICAL' THEN 5 WHEN 'HIGH' THEN 4 WHEN 'MEDIUM' THEN 3 WHEN 'LOW' THEN 2 ELSE 1 END DESC, updated_at DESC").
		Find(&files).Error
	return files, err
}

// QuarantineFile marks an auxiliary file as quarantined.
func (db *DB) QuarantineFile(fileID string, level models.ThreatLevel, summary string) error {
	now := time.Now()
	return db.Model(&models.AuxiliaryFile{}).
		Where("id = ?", fileID).
		Updates(map[string]interface{}{
			"security_status": models.SecurityStatusQuarantined,
			"threat_level":    level,
			"threat_summary":  summary,
			"scanned_at":      now,
		}).Error
}

// ReleaseFile releases an auxiliary file from quarantine.
func (db *DB) ReleaseFile(fileID string) error {
	now := time.Now()
	return db.Model(&models.AuxiliaryFile{}).
		Where("id = ?", fileID).
		Updates(map[string]interface{}{
			"security_status": models.SecurityStatusReleased,
			"released_at":     now,
		}).Error
}

// MarkFileClean marks an auxiliary file as clean.
func (db *DB) MarkFileClean(fileID string) error {
	now := time.Now()
	return db.Model(&models.AuxiliaryFile{}).
		Where("id = ?", fileID).
		Updates(map[string]interface{}{
			"security_status": models.SecurityStatusClean,
			"threat_level":    models.ThreatLevelNone,
			"threat_summary":  "",
			"scanned_at":      now,
		}).Error
}

// DeleteAuxiliaryFilesForSkill soft-deletes all auxiliary files for a skill.
func (db *DB) DeleteAuxiliaryFilesForSkill(skillID string) error {
	return db.Where("skill_id = ?", skillID).Delete(&models.AuxiliaryFile{}).Error
}

// HardDeleteAuxiliaryFilesForSkill permanently deletes auxiliary files.
func (db *DB) HardDeleteAuxiliaryFilesForSkill(skillID string) error {
	return db.Unscoped().Where("skill_id = ?", skillID).Delete(&models.AuxiliaryFile{}).Error
}
