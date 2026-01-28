package models

import "time"

// AgentPreference tracks user's agent/platform preferences.
type AgentPreference struct {
	AgentID        string     `gorm:"primaryKey;size:64" json:"agent_id"`
	Enabled        bool       `gorm:"default:false" json:"enabled"`
	Detected       bool       `gorm:"default:false" json:"detected"`
	DetectedAt     *time.Time `json:"detected_at"`
	SelectedAt     *time.Time `json:"selected_at"`
	PreferredScope string     `gorm:"type:text;default:'global'" json:"preferred_scope"`
	ProjectPath    string     `gorm:"type:text" json:"project_path"`
	GlobalPath     string     `gorm:"type:text" json:"global_path"`
	UpdatedAt      time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name for GORM.
func (AgentPreference) TableName() string {
	return "agent_preferences"
}
