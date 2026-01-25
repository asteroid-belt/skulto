package models

import "time"

// Installed represents a skill marked as installed for offline access.
type Installed struct {
	SkillID string    `gorm:"primaryKey;size:64" json:"skill_id"`
	Skill   Skill     `gorm:"foreignKey:SkillID" json:"-"`
	AddedAt time.Time `gorm:"autoCreateTime" json:"added_at"`
	Notes   string    `gorm:"type:text" json:"notes"`
}

// TableName specifies the table name for GORM.
func (Installed) TableName() string {
	return "installed"
}

// SyncMeta stores sync metadata as key-value pairs.
type SyncMeta struct {
	Key       string    `gorm:"primaryKey;size:100" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name for GORM.
func (SyncMeta) TableName() string {
	return "sync_meta"
}

// Common sync meta keys.
const (
	SyncMetaLastFullSync  = "last_full_sync"
	SyncMetaLastDeltaSync = "last_delta_sync"
	SyncMetaSchemaVersion = "schema_version"
	SyncMetaTotalSkills   = "total_skills"
)
