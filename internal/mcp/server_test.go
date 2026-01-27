package mcp

import (
	"path/filepath"
	"testing"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/favorites"
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

func setupTestFavorites(t *testing.T) *favorites.Store {
	t.Helper()
	tmpDir := t.TempDir()
	favPath := filepath.Join(tmpDir, "favorites.json")
	store := favorites.NewStore(favPath)
	require.NoError(t, store.Load())
	return store
}

func TestNewServer(t *testing.T) {
	database := setupTestDB(t)
	cfg := &config.Config{}
	favStore := setupTestFavorites(t)

	server := NewServer(database, cfg, favStore)

	assert.NotNil(t, server)
	assert.NotNil(t, server.server)
	assert.NotNil(t, server.db)
	assert.NotNil(t, server.installer)
	assert.NotNil(t, server.favorites)
}

func TestNewServer_WithNilConfig(t *testing.T) {
	database := setupTestDB(t)
	favStore := setupTestFavorites(t)

	// Should not panic with nil config
	server := NewServer(database, nil, favStore)

	assert.NotNil(t, server)
	assert.NotNil(t, server.server)
}

func TestNewServer_WithNilFavorites(t *testing.T) {
	database := setupTestDB(t)
	cfg := &config.Config{}

	// Should not panic with nil favorites store
	server := NewServer(database, cfg, nil)

	assert.NotNil(t, server)
	assert.NotNil(t, server.server)
	assert.Nil(t, server.favorites)
}
