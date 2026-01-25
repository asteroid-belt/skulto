package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/scraper"
	"github.com/asteroid-belt/skulto/internal/telemetry"
	"github.com/asteroid-belt/skulto/internal/tui/components"
)

// setupTestTelemetry initializes a noop telemetry client for tests.
func setupTestTelemetry() {
	telemetryClient = telemetry.New(nil)
}

func TestRemoveCmd_Structure(t *testing.T) {
	assert.Equal(t, "remove [repository]", removeCmd.Use)
	assert.NotEmpty(t, removeCmd.Short)
	assert.NotEmpty(t, removeCmd.Long)
	assert.NotNil(t, removeCmd.Args)
	assert.NotNil(t, removeCmd.RunE)
}

func TestRemoveCmd_ArgsValidation(t *testing.T) {
	validator := cobra.MaximumNArgs(1)

	// Should pass with no args (interactive mode)
	err := validator(removeCmd, []string{})
	assert.NoError(t, err)

	// Should pass with exactly 1 arg
	err = validator(removeCmd, []string{"owner/repo"})
	assert.NoError(t, err)

	// Should fail with too many args
	err = validator(removeCmd, []string{"arg1", "arg2"})
	assert.Error(t, err)
}

func TestRemoveCmd_ForceFlag(t *testing.T) {
	// Verify the force flag exists
	flag := removeCmd.Flags().Lookup("force")
	require.NotNil(t, flag)
	assert.Equal(t, "f", flag.Shorthand)
	assert.Equal(t, "false", flag.DefValue)
}

func TestRemoveCmd_ValidURLFormats(t *testing.T) {
	testCases := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
	}{
		{"short format", "owner/repo", "owner", "repo"},
		{"https URL", "https://github.com/owner/repo", "owner", "repo"},
		{"https URL with .git", "https://github.com/owner/repo.git", "owner", "repo"},
		{"ssh URL", "git@github.com:owner/repo.git", "owner", "repo"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			source, err := scraper.ParseRepositoryURL(tc.url)
			require.NoError(t, err)
			assert.Equal(t, tc.wantOwner, source.Owner)
			assert.Equal(t, tc.wantRepo, source.Repo)
		})
	}
}

func TestRemoveCmd_InvalidURL(t *testing.T) {
	_, err := scraper.ParseRepositoryURL("invalid-url")
	assert.Error(t, err)
}

// testDB creates a temporary test database.
func testDB(t *testing.T) *db.DB {
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

// testConfig creates a test config with temp directories.
func testConfig(t *testing.T) *config.Config {
	t.Helper()

	tmpDir := t.TempDir()
	cloneDir := filepath.Join(tmpDir, "repositories")
	require.NoError(t, os.MkdirAll(cloneDir, 0755))

	return &config.Config{
		BaseDir: tmpDir,
	}
}

func TestExecuteRemoval_WithSkills(t *testing.T) {
	setupTestTelemetry()
	database := testDB(t)
	cfg := testConfig(t)

	// Create a source
	source := &models.Source{
		ID:         "test/repo",
		Owner:      "test",
		Repo:       "repo",
		FullName:   "test/repo",
		SkillCount: 3,
	}
	require.NoError(t, database.CreateSource(source))

	// Create skills belonging to this source
	sourceID := source.ID
	for i := 0; i < 3; i++ {
		skill := &models.Skill{
			ID:       "skill-" + string(rune('a'+i)),
			Slug:     "skill-" + string(rune('a'+i)),
			Title:    "Skill " + string(rune('A'+i)),
			SourceID: &sourceID,
		}
		require.NoError(t, database.CreateSkill(skill))
	}

	// Create fake git clone directory
	paths := config.GetPaths(cfg)
	repoPath := filepath.Join(paths.Repositories, "test", "repo")
	require.NoError(t, os.MkdirAll(repoPath, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("test"), 0644))

	// Execute removal
	ctx := context.Background()
	err := executeRemoval(ctx, cfg, database, source)
	require.NoError(t, err)

	// Verify source is deleted
	result, err := database.GetSource(source.ID)
	require.NoError(t, err)
	assert.Nil(t, result)

	// Verify skills are deleted
	skills, err := database.GetSkillsBySourceID(source.ID)
	require.NoError(t, err)
	assert.Empty(t, skills)

	// Verify git clone is deleted
	_, err = os.Stat(repoPath)
	assert.True(t, os.IsNotExist(err))
}

func TestExecuteRemoval_NoSkills(t *testing.T) {
	setupTestTelemetry()
	database := testDB(t)
	cfg := testConfig(t)

	// Create a source with no skills
	source := &models.Source{
		ID:         "empty/repo",
		Owner:      "empty",
		Repo:       "repo",
		FullName:   "empty/repo",
		SkillCount: 0,
	}
	require.NoError(t, database.CreateSource(source))

	// Execute removal
	ctx := context.Background()
	err := executeRemoval(ctx, cfg, database, source)
	require.NoError(t, err)

	// Verify source is deleted
	result, err := database.GetSource(source.ID)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestExecuteRemoval_NoGitClone(t *testing.T) {
	setupTestTelemetry()
	database := testDB(t)
	cfg := testConfig(t)

	// Create a source without a corresponding git clone
	source := &models.Source{
		ID:         "no-clone/repo",
		Owner:      "no-clone",
		Repo:       "repo",
		FullName:   "no-clone/repo",
		SkillCount: 0,
	}
	require.NoError(t, database.CreateSource(source))

	// Execute removal (should not error even without git clone)
	ctx := context.Background()
	err := executeRemoval(ctx, cfg, database, source)
	require.NoError(t, err)

	// Verify source is deleted
	result, err := database.GetSource(source.ID)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestRepoSelectModel_Init(t *testing.T) {
	sources := []models.Source{
		{ID: "test/repo1", SkillCount: 5},
	}
	dialog := components.NewRepoSelectDialog(sources)
	model := repoSelectModel{dialog: dialog, width: 80, height: 24}

	cmd := model.Init()
	assert.Nil(t, cmd)
}

func TestRepoSelectModel_Update_WindowSize(t *testing.T) {
	sources := []models.Source{
		{ID: "test/repo1", SkillCount: 5},
	}
	dialog := components.NewRepoSelectDialog(sources)
	model := repoSelectModel{dialog: dialog, width: 80, height: 24}

	newModel, cmd := model.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	assert.Nil(t, cmd)

	m := newModel.(repoSelectModel)
	assert.Equal(t, 100, m.width)
	assert.Equal(t, 50, m.height)
}

func TestRepoSelectModel_Update_KeyPress(t *testing.T) {
	sources := []models.Source{
		{ID: "test/repo1", SkillCount: 5},
		{ID: "test/repo2", SkillCount: 10},
	}
	dialog := components.NewRepoSelectDialog(sources)
	model := repoSelectModel{dialog: dialog, width: 80, height: 24}

	// Navigate down
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	m := newModel.(repoSelectModel)
	assert.Equal(t, "test/repo2", m.dialog.GetSelection().ID)
}

func TestRepoSelectModel_Update_Confirm(t *testing.T) {
	sources := []models.Source{
		{ID: "test/repo1", SkillCount: 5},
	}
	dialog := components.NewRepoSelectDialog(sources)
	model := repoSelectModel{dialog: dialog, width: 80, height: 24}

	// Press enter to confirm
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.NotNil(t, cmd) // Should return tea.Quit
	assert.True(t, dialog.IsConfirmed())
}

func TestRepoSelectModel_Update_Cancel(t *testing.T) {
	sources := []models.Source{
		{ID: "test/repo1", SkillCount: 5},
	}
	dialog := components.NewRepoSelectDialog(sources)
	model := repoSelectModel{dialog: dialog, width: 80, height: 24}

	// Press escape to cancel
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.NotNil(t, cmd) // Should return tea.Quit
	assert.True(t, dialog.IsCancelled())
}

func TestRepoSelectModel_View(t *testing.T) {
	sources := []models.Source{
		{ID: "test/repo1", SkillCount: 5},
	}
	dialog := components.NewRepoSelectDialog(sources)
	model := repoSelectModel{dialog: dialog, width: 80, height: 24}

	view := model.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "test/repo1")
}

func TestBuildRepoOptions_WithSkillCounts(t *testing.T) {
	database := testDB(t)

	// Create a source
	source := &models.Source{
		ID:         "test/repo",
		Owner:      "test",
		Repo:       "repo",
		FullName:   "test/repo",
		SkillCount: 3,
	}
	require.NoError(t, database.CreateSource(source))

	// Create skills - 2 installed, 1 not installed
	sourceID := source.ID
	skills := []models.Skill{
		{ID: "skill-a", Slug: "skill-a", SourceID: &sourceID, IsInstalled: true},
		{ID: "skill-b", Slug: "skill-b", SourceID: &sourceID, IsInstalled: true},
		{ID: "skill-c", Slug: "skill-c", SourceID: &sourceID, IsInstalled: false},
	}
	for _, skill := range skills {
		require.NoError(t, database.CreateSkill(&skill))
	}

	// Get skills and count
	fetchedSkills, err := database.GetSkillsBySourceID(sourceID)
	require.NoError(t, err)

	var installedCount, notInstalledCount int
	for _, skill := range fetchedSkills {
		if skill.IsInstalled {
			installedCount++
		} else {
			notInstalledCount++
		}
	}

	assert.Equal(t, 2, installedCount)
	assert.Equal(t, 1, notInstalledCount)
}

func TestRepoSelectDialog_WithInstalledCounts(t *testing.T) {
	options := []components.RepoOption{
		{
			Source:            &models.Source{ID: "test/repo1"},
			Title:             "test/repo1",
			InstalledCount:    3,
			NotInstalledCount: 7,
		},
		{
			Source:            &models.Source{ID: "test/repo2"},
			Title:             "test/repo2",
			InstalledCount:    0,
			NotInstalledCount: 5,
		},
	}

	dialog := components.NewRepoSelectDialogWithOptions(options)
	view := dialog.View()

	// Should show both repos
	assert.Contains(t, view, "test/repo1")
	assert.Contains(t, view, "test/repo2")

	// Should show installed counts
	assert.Contains(t, view, "3 installed")
	assert.Contains(t, view, "7 not installed")
	assert.Contains(t, view, "0 installed")
	assert.Contains(t, view, "5 not installed")
}
