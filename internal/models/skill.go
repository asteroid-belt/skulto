// Package models defines the core data structures for Skulto.
package models

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"gorm.io/gorm"
)

// Skill represents a parsed SKILL.md file from GitHub.
type Skill struct {
	ID   string `gorm:"primaryKey;size:64" json:"id"`                     // SHA256 hash of repo+path
	Slug string `gorm:"uniqueIndex:idx_slug_source;size:100" json:"slug"` // URL-safe identifier

	// Content
	Title       string `gorm:"size:255;index" json:"title"`
	Description string `gorm:"size:1000" json:"description"`
	Content     string `gorm:"type:text" json:"content"` // Full SKILL.md content
	Summary     string `gorm:"size:500" json:"summary"`  // AI-generated summary

	// Source information (foreign key to Source)
	SourceID *string `gorm:"size:255;index;uniqueIndex:idx_slug_source" json:"source_id"` // nullable foreign key
	Source   *Source `gorm:"foreignKey:SourceID" json:"-"`
	FilePath string  `gorm:"size:500" json:"file_path"`

	// Categorization
	Tags []Tag `gorm:"many2many:skill_tags" json:"tags"`
	// Category is DEPRECATED - kept for backward compatibility, not written to.
	// Use tags for categorization instead.
	Category   string `gorm:"size:50;index" json:"category,omitempty"`
	Difficulty string `gorm:"size:20;index;default:intermediate" json:"difficulty"`

	// Metrics
	Stars     int `gorm:"default:0" json:"stars"`
	Forks     int `gorm:"default:0" json:"forks"`
	Downloads int `gorm:"default:0" json:"downloads"`

	// Embedding for vector search
	EmbeddingID string `gorm:"size:64" json:"embedding_id"`

	// Metadata
	Version string `gorm:"size:50" json:"version"`
	License string `gorm:"size:100" json:"license"` // User-provided license from SKILL.md frontmatter
	Author  string `gorm:"size:255;index" json:"author"`

	// Local state
	IsLocal     bool `gorm:"default:false" json:"is_local"`
	IsInstalled bool `gorm:"default:false;index" json:"is_installed"`

	// Security fields
	SecurityStatus SecurityStatus `gorm:"size:20;default:PENDING;index" json:"security_status"`
	ThreatLevel    ThreatLevel    `gorm:"size:20;default:NONE" json:"threat_level"`
	ThreatSummary  string         `gorm:"size:1000" json:"threat_summary"`
	ScannedAt      *time.Time     `json:"scanned_at"`
	ReleasedAt     *time.Time     `json:"released_at"`
	ContentHash    string         `gorm:"size:64" json:"content_hash"`

	// Auxiliary files relationship
	AuxiliaryFiles []AuxiliaryFile `gorm:"foreignKey:SkillID" json:"auxiliary_files,omitempty"`

	// Timestamps (GORM auto-manages CreatedAt/UpdatedAt)
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"` // Soft delete support
	IndexedAt  time.Time      `json:"indexed_at"`
	LastSyncAt *time.Time     `json:"last_sync_at"`
	ViewedAt   *time.Time     `json:"viewed_at"` // Tracks when skill was last viewed by user
}

// TableName specifies the table name for GORM.
func (Skill) TableName() string {
	return "skills"
}

// IsUsable returns true if the skill can be installed/used.
func (s *Skill) IsUsable() bool {
	return !s.SecurityStatus.IsBlocked()
}

// ComputeContentHash calculates SHA256 of the skill content.
func (s *Skill) ComputeContentHash() string {
	h := sha256.Sum256([]byte(s.Content))
	return hex.EncodeToString(h[:])
}

// SkillMeta contains extracted metadata from SKILL.md frontmatter.
type SkillMeta struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Version     string   `yaml:"version"`
	Author      string   `yaml:"author"`
	License     string   `yaml:"license"`
	Tags        []string `yaml:"tags"`
	Platforms   []string `yaml:"platforms"`
	MinVersion  string   `yaml:"min_version"`
}

// SkillStats provides aggregate statistics.
type SkillStats struct {
	TotalSkills    int64     `json:"total_skills"`
	TotalTags      int64     `json:"total_tags"`
	TotalSources   int64     `json:"total_sources"`
	LastUpdated    time.Time `json:"last_updated"`
	CacheSizeBytes int64     `json:"cache_size_bytes"`
	EmbeddingCount int64     `json:"embedding_count"`
}

// Difficulty levels.
const (
	DifficultyBeginner     = "beginner"
	DifficultyIntermediate = "intermediate"
	DifficultyAdvanced     = "advanced"
)

// ValidDifficulties returns all valid difficulty levels.
func ValidDifficulties() []string {
	return []string{DifficultyBeginner, DifficultyIntermediate, DifficultyAdvanced}
}
