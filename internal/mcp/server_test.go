package mcp

import (
	"path/filepath"
	"testing"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *db.DB {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.New(db.Config{
		Path:        dbPath,
		Debug:       false,
		MaxIdleConn: 1,
		MaxOpenConn: 1,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func TestNewServer(t *testing.T) {
	database := setupTestDB(t)
	cfg := &config.Config{}

	server := NewServer(database, cfg)

	assert.NotNil(t, server)
	assert.NotNil(t, server.server)
	assert.NotNil(t, server.db)
	assert.NotNil(t, server.installer)
}

func TestNewServer_WithNilConfig(t *testing.T) {
	database := setupTestDB(t)

	// Should not panic with nil config
	server := NewServer(database, nil)

	assert.NotNil(t, server)
	assert.NotNil(t, server.server)
}
