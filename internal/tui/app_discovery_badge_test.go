package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestModel_UpdateDiscoveryCount_NilDB tests that updateDiscoveryCount handles nil database gracefully.
func TestModel_UpdateDiscoveryCount_NilDB(t *testing.T) {
	// Create model with nil database
	m := &Model{
		db: nil,
	}

	// Should not panic with nil db
	assert.NotPanics(t, func() {
		m.updateDiscoveryCount()
	})
}

// TestModel_UpdateDiscoveryCount_NilHomeView tests behavior when homeView is nil.
func TestModel_UpdateDiscoveryCount_NilHomeView(t *testing.T) {
	// Create model with nil homeView
	m := &Model{
		db:       nil,
		homeView: nil,
	}

	// Should not panic - updateDiscoveryCount returns early if db is nil
	assert.NotPanics(t, func() {
		m.updateDiscoveryCount()
	})
}
