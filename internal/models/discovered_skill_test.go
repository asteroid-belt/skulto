package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDiscoveredSkill_GenerateID(t *testing.T) {
	ds := DiscoveredSkill{
		Platform: "claude",
		Scope:    "global",
		Path:     "/home/user/.claude/skills/my-skill",
		Name:     "my-skill",
	}

	id := ds.GenerateID()

	assert.NotEmpty(t, id)
	assert.Len(t, id, 64) // SHA256 hex
}

func TestDiscoveredSkill_GenerateID_Deterministic(t *testing.T) {
	ds1 := DiscoveredSkill{Platform: "claude", Scope: "global", Path: "/path/a"}
	ds2 := DiscoveredSkill{Platform: "claude", Scope: "global", Path: "/path/a"}

	assert.Equal(t, ds1.GenerateID(), ds2.GenerateID())
}

func TestDiscoveredSkill_GenerateID_Unique(t *testing.T) {
	ds1 := DiscoveredSkill{Platform: "claude", Scope: "global", Path: "/path/a"}
	ds2 := DiscoveredSkill{Platform: "claude", Scope: "global", Path: "/path/b"}

	assert.NotEqual(t, ds1.GenerateID(), ds2.GenerateID())
}

func TestDiscoveredSkill_IsNotified(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		notifiedAt *time.Time
		want       bool
	}{
		{"nil notifiedAt", nil, false},
		{"set notifiedAt", &now, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := DiscoveredSkill{NotifiedAt: tt.notifiedAt}
			assert.Equal(t, tt.want, ds.IsNotified())
		})
	}
}

func TestDiscoveredSkill_IsDismissed(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		dismissedAt *time.Time
		want        bool
	}{
		{"nil dismissedAt", nil, false},
		{"set dismissedAt", &now, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := DiscoveredSkill{DismissedAt: tt.dismissedAt}
			assert.Equal(t, tt.want, ds.IsDismissed())
		})
	}
}
