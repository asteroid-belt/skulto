package db

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/asteroid-belt/skulto/internal/models"
)

// GetUserState retrieves the current application state.
func (db *DB) GetUserState() (*models.UserState, error) {
	var state models.UserState
	err := db.Where("id = ?", "default").First(&state).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Return default state if not found
			return &models.UserState{
				ID:               "default",
				OnboardingStatus: models.OnboardingNotStarted,
				AITools:          "",
			}, nil
		}
		return nil, err
	}
	return &state, nil
}

// UpdateOnboardingStatus updates the onboarding status in the state table.
func (db *DB) UpdateOnboardingStatus(status models.OnboardingStatus) error {
	state := models.UserState{
		ID:               "default",
		OnboardingStatus: status,
	}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"onboarding_status", "updated_at"}),
	}).Create(&state).Error
}

// UpdateAITools updates the AI tools in the state table.
func (db *DB) UpdateAITools(tools string) error {
	state := models.UserState{
		ID:      "default",
		AITools: tools,
	}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"ai_tools", "updated_at"}),
	}).Create(&state).Error
}

// CompleteOnboarding marks onboarding as finished.
func (db *DB) CompleteOnboarding() error {
	return db.UpdateOnboardingStatus(models.OnboardingFinished)
}

// ResetOnboarding resets onboarding to not started.
func (db *DB) ResetOnboarding() error {
	return db.UpdateOnboardingStatus(models.OnboardingNotStarted)
}

// GetOrCreateTrackingID returns the persistent tracking ID, creating one if it doesn't exist.
// On any error, it falls back to generating a per-session ID.
func (db *DB) GetOrCreateTrackingID() string {
	state, err := db.GetUserState()
	if err != nil {
		return generateSessionID()
	}

	// If tracking ID exists, return it
	if state.TrackingID != "" {
		return state.TrackingID
	}

	// Generate new tracking ID
	trackingID := generateSessionID()

	// Save to database
	state.TrackingID = trackingID
	err = db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"tracking_id", "updated_at"}),
	}).Create(state).Error
	if err != nil {
		// Even if save fails, return the generated ID for this session
		return trackingID
	}

	return trackingID
}

// generateSessionID creates a new UUID for session-based tracking.
func generateSessionID() string {
	return uuid.New().String()
}

// GetSkipUninstallConfirm returns whether the user wants to skip uninstall confirmations.
func (db *DB) GetSkipUninstallConfirm() (bool, error) {
	state, err := db.GetUserState()
	if err != nil {
		return false, err
	}
	return state.SkipUninstallConfirm, nil
}

// SetSkipUninstallConfirm updates the skip uninstall confirmation preference.
func (db *DB) SetSkipUninstallConfirm(skip bool) error {
	state := models.UserState{
		ID:                   "default",
		SkipUninstallConfirm: skip,
	}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"skip_uninstall_confirm", "updated_at"}),
	}).Create(&state).Error
}
