package cli

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB creates a temporary test database for startup tests.
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
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}

	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Logf("Failed to close test database: %v", err)
		}
	})

	return database
}

func TestShowStartupNotification_WithUnnotified(t *testing.T) {
	database := setupTestDB(t)

	// Create unnotified discovery
	ds := models.DiscoveredSkill{
		Platform:     "claude",
		Scope:        "project",
		Path:         ".claude/skills/my-skill",
		Name:         "my-skill",
		DiscoveredAt: time.Now(),
	}
	ds.ID = ds.GenerateID()
	require.NoError(t, database.UpsertDiscoveredSkill(&ds))

	var buf bytes.Buffer
	shown := showStartupNotification(database, &buf)

	assert.True(t, shown)
	assert.Contains(t, buf.String(), "my-skill")
	assert.Contains(t, buf.String(), "unmanaged skill")
}

func TestShowStartupNotification_NoneUnnotified(t *testing.T) {
	database := setupTestDB(t)

	var buf bytes.Buffer
	shown := showStartupNotification(database, &buf)

	assert.False(t, shown)
	assert.Empty(t, buf.String())
}

func TestShowStartupNotification_MarksAsNotified(t *testing.T) {
	database := setupTestDB(t)

	ds := models.DiscoveredSkill{
		Platform:     "claude",
		Scope:        "project",
		Path:         ".claude/skills/test-skill",
		Name:         "test-skill",
		DiscoveredAt: time.Now(),
	}
	ds.ID = ds.GenerateID()
	require.NoError(t, database.UpsertDiscoveredSkill(&ds))

	var buf bytes.Buffer
	showStartupNotification(database, &buf)

	// Verify marked as notified
	unnotified, err := database.ListUnnotifiedDiscoveredSkills()
	require.NoError(t, err)
	assert.Len(t, unnotified, 0)
}

func TestShowStartupNotification_MultipleDiscoveries(t *testing.T) {
	database := setupTestDB(t)

	// Create multiple discoveries with different scopes
	discoveries := []models.DiscoveredSkill{
		{
			Platform:     "claude",
			Scope:        "project",
			Path:         ".claude/skills/project-skill",
			Name:         "project-skill",
			DiscoveredAt: time.Now(),
		},
		{
			Platform:     "cursor",
			Scope:        "project",
			Path:         ".cursor/skills/cursor-skill",
			Name:         "cursor-skill",
			DiscoveredAt: time.Now(),
		},
		{
			Platform:     "claude",
			Scope:        "global",
			Path:         "~/.claude/skills/global-skill",
			Name:         "global-skill",
			DiscoveredAt: time.Now(),
		},
	}

	for i := range discoveries {
		discoveries[i].ID = discoveries[i].GenerateID()
		require.NoError(t, database.UpsertDiscoveredSkill(&discoveries[i]))
	}

	var buf bytes.Buffer
	shown := showStartupNotification(database, &buf)

	assert.True(t, shown)
	output := buf.String()

	// Should mention all discoveries
	assert.Contains(t, output, "3 unmanaged skill")
	assert.Contains(t, output, "project-skill")
	assert.Contains(t, output, "cursor-skill")
	assert.Contains(t, output, "global-skill")

	// Should show scopes
	assert.Contains(t, output, "(project)")
	assert.Contains(t, output, "(global)")

	// Should show help message
	assert.Contains(t, output, "skulto ingest")
}

func TestShowStartupNotification_SkipsAlreadyNotified(t *testing.T) {
	database := setupTestDB(t)

	// Create already notified discovery
	notifiedAt := time.Now().Add(-time.Hour)
	ds := models.DiscoveredSkill{
		Platform:     "claude",
		Scope:        "project",
		Path:         ".claude/skills/old-skill",
		Name:         "old-skill",
		DiscoveredAt: time.Now().Add(-2 * time.Hour),
		NotifiedAt:   &notifiedAt,
	}
	ds.ID = ds.GenerateID()
	require.NoError(t, database.UpsertDiscoveredSkill(&ds))

	var buf bytes.Buffer
	shown := showStartupNotification(database, &buf)

	assert.False(t, shown)
	assert.Empty(t, buf.String())
}

func TestShowStartupNotification_NilDatabase(t *testing.T) {
	var buf bytes.Buffer
	shown := showStartupNotification(nil, &buf)

	assert.False(t, shown)
	assert.Empty(t, buf.String())
}

func TestShowStartupNotification_OutputFormat(t *testing.T) {
	database := setupTestDB(t)

	ds := models.DiscoveredSkill{
		Platform:     "claude",
		Scope:        "project",
		Path:         ".claude/skills/format-test",
		Name:         "format-test",
		DiscoveredAt: time.Now(),
	}
	ds.ID = ds.GenerateID()
	require.NoError(t, database.UpsertDiscoveredSkill(&ds))

	var buf bytes.Buffer
	showStartupNotification(database, &buf)

	output := buf.String()

	// Should have proper formatting
	assert.Contains(t, output, "Found 1 unmanaged skill")
	assert.Contains(t, output, ".claude/skills/format-test")
	assert.Contains(t, output, "(project)")
	assert.Contains(t, output, "Run `skulto ingest` or use Manage view to import")
}
