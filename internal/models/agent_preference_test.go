package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAgentPreference_TableName(t *testing.T) {
	pref := AgentPreference{}
	assert.Equal(t, "agent_preferences", pref.TableName())
}

func TestAgentPreference_Defaults(t *testing.T) {
	pref := AgentPreference{AgentID: "claude"}
	assert.Equal(t, "claude", pref.AgentID)
	assert.False(t, pref.Enabled)
	assert.False(t, pref.Detected)
	assert.Nil(t, pref.DetectedAt)
	assert.Nil(t, pref.SelectedAt)
	assert.Empty(t, pref.ProjectPath)
	assert.Empty(t, pref.GlobalPath)
}
