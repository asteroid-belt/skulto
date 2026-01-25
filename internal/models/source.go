package models

import (
	"time"

	"gorm.io/gorm"
)

// Source represents a GitHub repository containing skills.
type Source struct {
	ID       string `gorm:"primaryKey;size:255" json:"id"` // owner/repo
	Owner    string `gorm:"size:100;index" json:"owner"`
	Repo     string `gorm:"size:100;index" json:"repo"`
	FullName string `gorm:"size:255" json:"full_name"`

	// Repository metadata
	Description string `gorm:"size:1000" json:"description"`
	URL         string `gorm:"size:500" json:"url"`
	CloneURL    string `gorm:"size:500" json:"clone_url"`

	// Stats
	Stars    int `gorm:"default:0" json:"stars"`
	Forks    int `gorm:"default:0" json:"forks"`
	Watchers int `gorm:"default:0" json:"watchers"`

	// Tracking
	DefaultBranch string `gorm:"size:100;default:main" json:"default_branch"`
	LastCommitSHA string `gorm:"size:64" json:"last_commit_sha"`
	SkillCount    int    `gorm:"default:0" json:"skill_count"`

	// Scraping metadata
	Priority   int  `gorm:"default:5;index" json:"priority"` // 1-10, higher = more important
	IsCurated  bool `gorm:"default:false" json:"is_curated"`
	IsOfficial bool `gorm:"default:false" json:"is_official"`

	// License information
	LicenseType string `gorm:"size:50" json:"license_type"`  // Detected SPDX identifier (e.g., "MIT", "Apache-2.0")
	LicenseURL  string `gorm:"size:500" json:"license_url"`  // Direct link to LICENSE file in repository
	LicenseFile string `gorm:"size:100" json:"license_file"` // The LICENSE file name found (e.g., "LICENSE", "LICENSE.md")

	// Timestamps
	LastScrapedAt *time.Time     `json:"last_scraped_at"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

	// Has many skills
	Skills []Skill `gorm:"foreignKey:SourceID" json:"-"`
}

// TableName specifies the table name for GORM.
func (Source) TableName() string {
	return "sources"
}

// SourceType categorizes repositories.
type SourceType string

const (
	SourceTypeOfficial  SourceType = "official"  // Anthropic, OpenAI official repos
	SourceTypeCurated   SourceType = "curated"   // Hand-picked quality repos
	SourceTypeCommunity SourceType = "community" // Discovered via GitHub search
)

// SeedSource represents a known source of skills for initial scraping.
type SeedSource struct {
	Owner     string
	Repo      string
	Priority  int
	Type      SourceType
	SkillPath string // Path to skills, default is root
}

// OfficialSeeds are first-party skill repositories.
var OfficialSeeds = []SeedSource{
	{Owner: "anthropics", Repo: "anthropic-cookbook", Priority: 10, Type: SourceTypeOfficial},
	{Owner: "anthropics", Repo: "courses", Priority: 10, Type: SourceTypeOfficial},
	{Owner: "modelcontextprotocol", Repo: "servers", Priority: 10, Type: SourceTypeOfficial},
}

// CuratedSeeds are high-quality community repositories.
var CuratedSeeds = []SeedSource{
	{Owner: "pontusab", Repo: "cursor-rules", Priority: 8, Type: SourceTypeCurated},
	{Owner: "PatrickJS", Repo: "awesome-cursorrules", Priority: 8, Type: SourceTypeCurated},
}
