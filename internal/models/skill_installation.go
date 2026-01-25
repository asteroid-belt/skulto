package models

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// SkillInstallation tracks where a skill is installed.
// A skill can be installed to multiple locations (global + project, multiple platforms).
type SkillInstallation struct {
	ID          string    `gorm:"primaryKey;size:64" json:"id"`           // Hash of skill_id + platform + scope + base_path
	SkillID     string    `gorm:"size:64;index;not null" json:"skill_id"` // FK to skills.id
	Platform    string    `gorm:"size:20;index;not null" json:"platform"` // "claude", "cursor", etc.
	Scope       string    `gorm:"size:20;index;not null" json:"scope"`    // "global" or "project"
	BasePath    string    `gorm:"size:500;not null" json:"base_path"`     // Actual path used (e.g., /Users/x or /project/dir)
	SymlinkPath string    `gorm:"size:500" json:"symlink_path"`           // Full path to the created symlink
	InstalledAt time.Time `gorm:"autoCreateTime" json:"installed_at"`
}

// TableName specifies the table name for GORM.
func (SkillInstallation) TableName() string {
	return "skill_installations"
}

// GenerateID creates a unique ID for this installation based on key components.
func (si *SkillInstallation) GenerateID() string {
	data := si.SkillID + ":" + si.Platform + ":" + si.Scope + ":" + si.BasePath
	return hashString(data)
}

// hashString creates a SHA256 hash of the input (first 16 chars).
func hashString(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))[:16]
}
