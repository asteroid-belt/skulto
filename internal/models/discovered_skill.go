package models

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// DiscoveredSkill represents a non-symlinked skill directory found in a platform's skill folder.
type DiscoveredSkill struct {
	ID           string     `gorm:"primaryKey"`
	Platform     string     `gorm:"not null;index"`
	Scope        string     `gorm:"not null;index"`
	Path         string     `gorm:"not null;uniqueIndex"`
	Name         string     `gorm:"not null"`
	DiscoveredAt time.Time  `gorm:"not null"`
	NotifiedAt   *time.Time `gorm:"index"`
	DismissedAt  *time.Time
}

// TableName returns the database table name.
func (DiscoveredSkill) TableName() string {
	return "discovered_skills"
}

// GenerateID creates a deterministic ID from platform, scope, and path.
func (ds *DiscoveredSkill) GenerateID() string {
	data := fmt.Sprintf("%s:%s:%s", ds.Platform, ds.Scope, ds.Path)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// IsNotified returns true if the user has been notified about this discovery.
func (ds *DiscoveredSkill) IsNotified() bool {
	return ds.NotifiedAt != nil
}

// IsDismissed returns true if the user has dismissed this discovery.
func (ds *DiscoveredSkill) IsDismissed() bool {
	return ds.DismissedAt != nil
}

// ManagementSource represents who manages a skill.
type ManagementSource string

const (
	ManagementSkulto   ManagementSource = "Skulto"
	ManagementVercel   ManagementSource = "Vercel"
	ManagementExternal ManagementSource = "External"
	ManagementNone     ManagementSource = "Unmanaged"
)
