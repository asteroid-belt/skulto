package db

import (
	"time"

	"github.com/asteroid-belt/skulto/internal/models"
)

// GetAgentPreferences returns all agent preferences.
func (db *DB) GetAgentPreferences() ([]models.AgentPreference, error) {
	var prefs []models.AgentPreference
	if err := db.Find(&prefs).Error; err != nil {
		return nil, err
	}
	return prefs, nil
}

// GetEnabledAgents returns only enabled agent IDs.
func (db *DB) GetEnabledAgents() ([]string, error) {
	var agentIDs []string
	if err := db.Model(&models.AgentPreference{}).
		Where("enabled = ?", true).
		Pluck("agent_id", &agentIDs).Error; err != nil {
		return nil, err
	}
	return agentIDs, nil
}

// UpsertAgentPreference creates or updates an agent preference.
func (db *DB) UpsertAgentPreference(pref *models.AgentPreference) error {
	var existing models.AgentPreference
	result := db.Where("agent_id = ?", pref.AgentID).First(&existing)
	if result.Error != nil {
		// Not found, create
		return db.Create(pref).Error
	}
	// Update existing
	return db.Model(&existing).Updates(pref).Error
}

// SetAgentsEnabled bulk-enables a list of agents and disables all others.
// The projectPath and globalPath for each agent should be set on the
// AgentPreference records before calling this, or passed via UpsertAgentPreference.
func (db *DB) SetAgentsEnabled(agentIDs []string) error {
	return db.Transaction(func(tx *DB) error {
		// Disable all
		if err := tx.Model(&models.AgentPreference{}).
			Where("1 = 1").
			Update("enabled", false).Error; err != nil {
			return err
		}

		if len(agentIDs) == 0 {
			return nil
		}

		now := time.Now()
		for _, id := range agentIDs {
			var existing models.AgentPreference
			result := tx.Where("agent_id = ?", id).First(&existing)
			if result.Error != nil {
				// Create new
				pref := models.AgentPreference{
					AgentID:    id,
					Enabled:    true,
					SelectedAt: &now,
				}
				if err := tx.Create(&pref).Error; err != nil {
					return err
				}
			} else {
				// Update existing
				if err := tx.Model(&existing).Updates(map[string]any{
					"enabled":     true,
					"selected_at": &now,
				}).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// EnableAdditionalAgents enables the given agents without disabling others.
func (db *DB) EnableAdditionalAgents(agentIDs []string) error {
	if len(agentIDs) == 0 {
		return nil
	}

	now := time.Now()
	for _, id := range agentIDs {
		var existing models.AgentPreference
		result := db.Where("agent_id = ?", id).First(&existing)
		if result.Error != nil {
			pref := models.AgentPreference{
				AgentID:    id,
				Enabled:    true,
				SelectedAt: &now,
			}
			if err := db.Create(&pref).Error; err != nil {
				return err
			}
		} else if !existing.Enabled {
			if err := db.Model(&existing).Updates(map[string]any{
				"enabled":     true,
				"selected_at": &now,
			}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// UpdateDetectionState updates detected status for agents after detection.
func (db *DB) UpdateDetectionState(detected map[string]bool) error {
	now := time.Now()
	for agentID, isDetected := range detected {
		var existing models.AgentPreference
		result := db.Where("agent_id = ?", agentID).First(&existing)
		if result.Error != nil {
			// Create new preference for detected agent
			if isDetected {
				pref := models.AgentPreference{
					AgentID:    agentID,
					Detected:   true,
					DetectedAt: &now,
				}
				if err := db.Create(&pref).Error; err != nil {
					return err
				}
			}
		} else {
			updates := map[string]any{
				"detected": isDetected,
			}
			if isDetected && existing.DetectedAt == nil {
				updates["detected_at"] = &now
			}
			if err := db.Model(&existing).Updates(updates).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// MigrateFromAITools reads UserState.AITools and creates agent_preferences rows.
// Only runs if agent_preferences table is empty (first migration).
func (db *DB) MigrateFromAITools() error {
	// Check if agent_preferences already has data
	var count int64
	if err := db.Model(&models.AgentPreference{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil // Already migrated
	}

	// Read UserState.AITools
	state, err := db.GetUserState()
	if err != nil || state == nil {
		return nil // No state to migrate
	}

	tools := state.GetAITools()
	if len(tools) == 0 {
		return nil
	}

	now := time.Now()
	for _, tool := range tools {
		pref := &models.AgentPreference{
			AgentID:    tool,
			Enabled:    true,
			SelectedAt: &now,
		}
		if err := db.Create(pref).Error; err != nil {
			return err
		}
	}
	return nil
}
