package db

import (
	"fmt"
	"time"

	"github.com/asteroid-belt/skulto/internal/models"
)

// QuarantineSkill marks a skill as quarantined with threat details.
func (db *DB) QuarantineSkill(skillID string, level models.ThreatLevel, summary string) error {
	now := time.Now()
	return db.Model(&models.Skill{}).
		Where("id = ?", skillID).
		Updates(map[string]interface{}{
			"security_status": models.SecurityStatusQuarantined,
			"threat_level":    level,
			"threat_summary":  summary,
			"scanned_at":      now,
		}).Error
}

// ReleaseSkill manually releases a skill from quarantine.
func (db *DB) ReleaseSkill(skillID string) error {
	now := time.Now()

	// Get current content hash
	var skill models.Skill
	if err := db.First(&skill, "id = ?", skillID).Error; err != nil {
		return fmt.Errorf("find skill: %w", err)
	}

	return db.Model(&models.Skill{}).
		Where("id = ?", skillID).
		Updates(map[string]interface{}{
			"security_status": models.SecurityStatusReleased,
			"released_at":     now,
			"content_hash":    skill.ComputeContentHash(),
		}).Error
}

// MarkSkillClean marks a skill as clean after scanning.
func (db *DB) MarkSkillClean(skillID string) error {
	now := time.Now()

	var skill models.Skill
	if err := db.First(&skill, "id = ?", skillID).Error; err != nil {
		return fmt.Errorf("find skill: %w", err)
	}

	return db.Model(&models.Skill{}).
		Where("id = ?", skillID).
		Updates(map[string]interface{}{
			"security_status": models.SecurityStatusClean,
			"threat_level":    models.ThreatLevelNone,
			"threat_summary":  "",
			"scanned_at":      now,
			"content_hash":    skill.ComputeContentHash(),
		}).Error
}

// GetQuarantinedSkills returns all quarantined skills.
// Results are ordered by threat severity (CRITICAL > HIGH > MEDIUM > LOW > NONE).
func (db *DB) GetQuarantinedSkills() ([]models.Skill, error) {
	var skills []models.Skill
	err := db.Preload("Tags").Joins("Source").
		Where("security_status = ?", models.SecurityStatusQuarantined).
		Order("CASE skills.threat_level WHEN 'CRITICAL' THEN 5 WHEN 'HIGH' THEN 4 WHEN 'MEDIUM' THEN 3 WHEN 'LOW' THEN 2 ELSE 1 END DESC, skills.updated_at DESC").
		Find(&skills).Error
	return skills, err
}

// GetPendingSkills returns skills awaiting scan.
func (db *DB) GetPendingSkills() ([]models.Skill, error) {
	var skills []models.Skill
	err := db.Preload("Tags").Joins("Source").
		Where("security_status = ?", models.SecurityStatusPending).
		Order("skills.updated_at DESC").
		Find(&skills).Error
	return skills, err
}

// GetCleanSkills returns skills marked clean.
func (db *DB) GetCleanSkills() ([]models.Skill, error) {
	var skills []models.Skill
	err := db.Preload("Tags").Joins("Source").
		Where("security_status = ?", models.SecurityStatusClean).
		Order("skills.updated_at DESC").
		Find(&skills).Error
	return skills, err
}

// GetUsableSkills returns skills that can be installed (clean or released).
func (db *DB) GetUsableSkills() ([]models.Skill, error) {
	var skills []models.Skill
	err := db.Preload("Tags").Joins("Source").
		Where("security_status IN ?", []models.SecurityStatus{
			models.SecurityStatusClean,
			models.SecurityStatusReleased,
		}).
		Order("skills.updated_at DESC").
		Find(&skills).Error
	return skills, err
}

// CountBySecurityStatus returns counts grouped by status.
func (db *DB) CountBySecurityStatus() (map[models.SecurityStatus]int64, error) {
	type result struct {
		Status models.SecurityStatus
		Count  int64
	}
	var results []result

	err := db.Model(&models.Skill{}).
		Select("security_status as status, count(*) as count").
		Group("security_status").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[models.SecurityStatus]int64)
	for _, r := range results {
		counts[r.Status] = r.Count
	}
	return counts, nil
}

// --- Security Scan Audit ---

// CreateSecurityScan starts a new scan record.
func (db *DB) CreateSecurityScan(scan *models.SecurityScan) error {
	return db.Create(scan).Error
}

// CompleteSecurityScan marks a scan as completed.
func (db *DB) CompleteSecurityScan(scanID uint, stats models.SecurityScan) error {
	now := time.Now()
	return db.Model(&models.SecurityScan{}).
		Where("id = ?", scanID).
		Updates(map[string]interface{}{
			"completed_at":      now,
			"skills_scanned":    stats.SkillsScanned,
			"files_scanned":     stats.FilesScanned,
			"threats_found":     stats.ThreatsFound,
			"quarantined_count": stats.QuarantinedCount,
			"scan_summary":      stats.ScanSummary,
		}).Error
}

// GetRecentScans returns recent scan records.
func (db *DB) GetRecentScans(limit int) ([]models.SecurityScan, error) {
	var scans []models.SecurityScan
	err := db.Order("started_at DESC").Limit(limit).Find(&scans).Error
	return scans, err
}

// --- Additional Security Scanner Methods ---

// UpdateSkillSecurity updates security-related fields for a skill after scanning.
// This is used by the scanner to update threat level and summary without changing status.
func (db *DB) UpdateSkillSecurity(skill *models.Skill) error {
	return db.Model(skill).Updates(map[string]interface{}{
		"security_status": skill.SecurityStatus,
		"threat_level":    skill.ThreatLevel,
		"threat_summary":  skill.ThreatSummary,
		"scanned_at":      skill.ScannedAt,
		"content_hash":    skill.ContentHash,
	}).Error
}

// CountSkillsWithWarnings returns the count of skills with ThreatLevel != NONE.
// These are skills that have been scanned and found to have potential threats.
func (db *DB) CountSkillsWithWarnings() (int64, error) {
	var count int64
	err := db.Model(&models.Skill{}).
		Where("threat_level != ?", models.ThreatLevelNone).
		Count(&count).Error
	return count, err
}
