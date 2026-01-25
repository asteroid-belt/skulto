package models

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"gorm.io/gorm"
)

// SecurityStatus represents the security review state.
type SecurityStatus string

const (
	SecurityStatusPending     SecurityStatus = "PENDING"     // Awaiting scan
	SecurityStatusClean       SecurityStatus = "CLEAN"       // No threats detected
	SecurityStatusQuarantined SecurityStatus = "QUARANTINED" // Threats found, blocked
	SecurityStatusReleased    SecurityStatus = "RELEASED"    // Manually approved
)

// IsValid checks if the status is a known value.
func (s SecurityStatus) IsValid() bool {
	switch s {
	case SecurityStatusPending, SecurityStatusClean,
		SecurityStatusQuarantined, SecurityStatusReleased:
		return true
	}
	return false
}

// AllSecurityStatuses returns all valid status values.
func AllSecurityStatuses() []SecurityStatus {
	return []SecurityStatus{
		SecurityStatusPending,
		SecurityStatusClean,
		SecurityStatusQuarantined,
		SecurityStatusReleased,
	}
}

// IsBlocked returns true if the status prevents usage.
func (s SecurityStatus) IsBlocked() bool {
	return s == SecurityStatusQuarantined || s == SecurityStatusPending
}

// ThreatLevel represents severity of detected threats.
type ThreatLevel string

const (
	ThreatLevelNone     ThreatLevel = "NONE"
	ThreatLevelLow      ThreatLevel = "LOW"
	ThreatLevelMedium   ThreatLevel = "MEDIUM"
	ThreatLevelHigh     ThreatLevel = "HIGH"
	ThreatLevelCritical ThreatLevel = "CRITICAL"
)

// IsValid checks if the level is a known value.
func (t ThreatLevel) IsValid() bool {
	switch t {
	case ThreatLevelNone, ThreatLevelLow, ThreatLevelMedium,
		ThreatLevelHigh, ThreatLevelCritical:
		return true
	}
	return false
}

// AllThreatLevels returns all valid threat levels.
func AllThreatLevels() []ThreatLevel {
	return []ThreatLevel{
		ThreatLevelNone,
		ThreatLevelLow,
		ThreatLevelMedium,
		ThreatLevelHigh,
		ThreatLevelCritical,
	}
}

// Severity returns numeric severity (0-4) for sorting.
func (t ThreatLevel) Severity() int {
	switch t {
	case ThreatLevelNone:
		return 0
	case ThreatLevelLow:
		return 1
	case ThreatLevelMedium:
		return 2
	case ThreatLevelHigh:
		return 3
	case ThreatLevelCritical:
		return 4
	}
	return 0
}

// AuxiliaryDirType is the type of auxiliary directory.
type AuxiliaryDirType string

const (
	AuxDirScripts    AuxiliaryDirType = "scripts"
	AuxDirReferences AuxiliaryDirType = "references"
	AuxDirAssets     AuxiliaryDirType = "assets"
)

// IsValid checks if the dir type is known.
func (d AuxiliaryDirType) IsValid() bool {
	switch d {
	case AuxDirScripts, AuxDirReferences, AuxDirAssets:
		return true
	}
	return false
}

// AllAuxiliaryDirTypes returns all valid directory types.
func AllAuxiliaryDirTypes() []AuxiliaryDirType {
	return []AuxiliaryDirType{AuxDirScripts, AuxDirReferences, AuxDirAssets}
}

// AuxiliaryFile tracks individual files in auxiliary directories.
type AuxiliaryFile struct {
	ID      string `gorm:"primaryKey;size:64" json:"id"`
	SkillID string `gorm:"size:64;index;not null" json:"skill_id"`
	Skill   *Skill `gorm:"foreignKey:SkillID" json:"-"`

	DirType  AuxiliaryDirType `gorm:"size:20;index;not null" json:"dir_type"`
	FilePath string           `gorm:"size:500;not null" json:"file_path"`
	FileName string           `gorm:"size:255;not null" json:"file_name"`

	ContentHash string `gorm:"size:64" json:"content_hash"`
	FileSize    int64  `gorm:"default:0" json:"file_size"`

	// Security fields
	SecurityStatus SecurityStatus `gorm:"size:20;default:PENDING;index" json:"security_status"`
	ThreatLevel    ThreatLevel    `gorm:"size:20;default:NONE" json:"threat_level"`
	ThreatSummary  string         `gorm:"size:1000" json:"threat_summary"`
	ScannedAt      *time.Time     `json:"scanned_at"`
	ReleasedAt     *time.Time     `json:"released_at"`

	// Timestamps
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for GORM.
func (AuxiliaryFile) TableName() string { return "auxiliary_files" }

// GenerateID creates a deterministic ID from skill, dir type, and path.
func (af *AuxiliaryFile) GenerateID() string {
	data := af.SkillID + ":" + string(af.DirType) + ":" + af.FilePath
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])[:32]
}

// BeforeCreate generates the ID if not set.
func (af *AuxiliaryFile) BeforeCreate(tx *gorm.DB) error {
	if af.ID == "" {
		af.ID = af.GenerateID()
	}
	return nil
}

// SecurityScan records a scan run for audit purposes.
type SecurityScan struct {
	ID               uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	SkillID          *string    `gorm:"size:64;index" json:"skill_id"`
	ScanType         string     `gorm:"size:20;not null" json:"scan_type"`
	StartedAt        time.Time  `gorm:"not null" json:"started_at"`
	CompletedAt      *time.Time `json:"completed_at"`
	SkillsScanned    int        `gorm:"default:0" json:"skills_scanned"`
	FilesScanned     int        `gorm:"default:0" json:"files_scanned"`
	ThreatsFound     int        `gorm:"default:0" json:"threats_found"`
	QuarantinedCount int        `gorm:"default:0" json:"quarantined_count"`
	ScanSummary      string     `gorm:"type:text" json:"scan_summary"`
}

// TableName specifies the table name for GORM.
func (SecurityScan) TableName() string { return "security_scans" }

// ScanType constants.
const (
	ScanTypeFull   = "full"
	ScanTypeDelta  = "delta"
	ScanTypeSingle = "single"
)
