package models

import (
	"strings"
	"time"
)

// OnboardingStatus represents the state of onboarding.
type OnboardingStatus string

const (
	OnboardingNotStarted OnboardingStatus = "NOT_STARTED"
	OnboardingFinished   OnboardingStatus = "FINISHED"
)

// UserState represents the user's application state including onboarding and AI tools.
// Note: The table name is "user_state" to avoid conflicts with reserved keywords.
type UserState struct {
	ID               string           `gorm:"primaryKey;size:64" json:"id"`
	OnboardingStatus OnboardingStatus `gorm:"size:20;default:NOT_STARTED" json:"onboarding_status"`
	AITools          string           `gorm:"type:text" json:"ai_tools"`
	TrackingID       string           `gorm:"size:64" json:"tracking_id"`
	UpdatedAt        time.Time        `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name for GORM.
func (UserState) TableName() string {
	return "user_state"
}

// IsOnboardingCompleted returns true if onboarding is finished.
func (s *UserState) IsOnboardingCompleted() bool {
	return s.OnboardingStatus == OnboardingFinished
}

// GetAITools returns the list of AI tools from the comma-delimited string.
func (s *UserState) GetAITools() []string {
	if s.AITools == "" {
		return []string{}
	}
	// Split by comma and trim whitespace from each tool
	parts := strings.Split(s.AITools, ",")
	tools := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			tools = append(tools, trimmed)
		}
	}
	return tools
}

// SetAITools sets the AI tools from a list.
func (s *UserState) SetAITools(tools []string) {
	s.AITools = strings.Join(tools, ",")
}
