package views

import (
	"strings"
	"testing"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// manageTestTelemetry returns a noop telemetry client for testing.
func manageTestTelemetry() telemetry.Client {
	return telemetry.New(nil)
}

// setupManageTestDB creates an in-memory test database.
func setupManageTestDB(t *testing.T) *db.DB {
	t.Helper()

	database, err := db.New(db.Config{
		Path:        ":memory:",
		Debug:       false,
		MaxIdleConn: 1,
		MaxOpenConn: 1,
	})
	require.NoError(t, err, "failed to create test database")

	return database
}

func TestManageView_HasSections(t *testing.T) {
	database := setupManageTestDB(t)
	defer func() { _ = database.Close() }()

	cfg := &config.Config{}
	tel := manageTestTelemetry()
	installSvc := installer.NewInstallService(database, cfg, tel)
	view := NewManageView(database, cfg, installSvc, tel)
	view.SetSize(80, 24)

	// Simulate loaded data
	view.HandleManageSkillsLoaded(ManageSkillsLoadedMsg{
		Skills: []installer.InstalledSkillSummary{
			{Slug: "test-skill", Locations: map[installer.Platform][]installer.InstallScope{
				installer.PlatformClaude: {installer.ScopeGlobal},
			}},
		},
		Err: nil,
	})

	// Add a discovered skill
	discovery := &models.DiscoveredSkill{
		Platform: "claude",
		Scope:    "global",
		Path:     "/test/path",
		Name:     "discovered-skill",
	}
	discovery.ID = discovery.GenerateID()
	err := database.UpsertDiscoveredSkill(discovery)
	require.NoError(t, err)

	// Simulate discoveries loaded
	view.HandleDiscoveriesLoaded(DiscoveriesLoadedMsg{
		Skills: []models.DiscoveredSkill{*discovery},
		Err:    nil,
	})

	// View should have section indicators
	output := view.View()
	assert.Contains(t, output, "INSTALLED")
	assert.Contains(t, output, "DISCOVERED")
}

func TestManageView_TabSwitchesSections(t *testing.T) {
	database := setupManageTestDB(t)
	defer func() { _ = database.Close() }()

	cfg := &config.Config{}
	tel := manageTestTelemetry()
	installSvc := installer.NewInstallService(database, cfg, tel)
	view := NewManageView(database, cfg, installSvc, tel)
	view.SetSize(80, 24)

	// Simulate loaded data
	view.HandleManageSkillsLoaded(ManageSkillsLoadedMsg{
		Skills: []installer.InstalledSkillSummary{
			{Slug: "test-skill"},
		},
		Err: nil,
	})

	// Add a discovered skill
	discovery := &models.DiscoveredSkill{
		Platform: "claude",
		Scope:    "global",
		Path:     "/test/path",
		Name:     "discovered-skill",
	}
	discovery.ID = discovery.GenerateID()
	err := database.UpsertDiscoveredSkill(discovery)
	require.NoError(t, err)

	view.HandleDiscoveriesLoaded(DiscoveriesLoadedMsg{
		Skills: []models.DiscoveredSkill{*discovery},
		Err:    nil,
	})

	// Initially should be in Installed section
	assert.Equal(t, ManageSectionInstalled, view.GetCurrentSection())

	// Tab should switch to Discovered section
	view.Update("tab")
	assert.Equal(t, ManageSectionDiscovered, view.GetCurrentSection())

	// Tab again should switch back to Installed
	view.Update("tab")
	assert.Equal(t, ManageSectionInstalled, view.GetCurrentSection())
}

func TestManageView_SectionSelectionIndependent(t *testing.T) {
	database := setupManageTestDB(t)
	defer func() { _ = database.Close() }()

	cfg := &config.Config{}
	tel := manageTestTelemetry()
	installSvc := installer.NewInstallService(database, cfg, tel)
	view := NewManageView(database, cfg, installSvc, tel)
	view.SetSize(80, 24)

	// Simulate multiple installed skills
	view.HandleManageSkillsLoaded(ManageSkillsLoadedMsg{
		Skills: []installer.InstalledSkillSummary{
			{Slug: "installed-1"},
			{Slug: "installed-2"},
			{Slug: "installed-3"},
		},
		Err: nil,
	})

	// Add multiple discovered skills
	for _, name := range []string{"discovered-1", "discovered-2", "discovered-3"} {
		d := &models.DiscoveredSkill{
			Platform: "claude",
			Scope:    "global",
			Path:     "/test/" + name,
			Name:     name,
		}
		d.ID = d.GenerateID()
		err := database.UpsertDiscoveredSkill(d)
		require.NoError(t, err)
	}

	discoveries, err := database.ListDiscoveredSkills()
	require.NoError(t, err)
	view.HandleDiscoveriesLoaded(DiscoveriesLoadedMsg{
		Skills: discoveries,
		Err:    nil,
	})

	// Move down in installed section
	view.Update("j")
	view.Update("j")
	installedIdx := view.GetSelectedIndex()
	assert.Equal(t, 2, installedIdx, "should be at index 2 in installed section")

	// Switch to discovered section
	view.Update("tab")
	assert.Equal(t, ManageSectionDiscovered, view.GetCurrentSection())
	assert.Equal(t, 0, view.GetSelectedIndex(), "discovered section should start at index 0")

	// Move down in discovered section
	view.Update("j")
	discoveredIdx := view.GetSelectedIndex()
	assert.Equal(t, 1, discoveredIdx, "should be at index 1 in discovered section")

	// Switch back to installed section
	view.Update("tab")
	assert.Equal(t, ManageSectionInstalled, view.GetCurrentSection())
	assert.Equal(t, 2, view.GetSelectedIndex(), "installed section should remember index 2")
}

func TestManageView_ShowsManagementSource(t *testing.T) {
	database := setupManageTestDB(t)
	defer func() { _ = database.Close() }()

	cfg := &config.Config{}
	tel := manageTestTelemetry()
	installSvc := installer.NewInstallService(database, cfg, tel)
	view := NewManageView(database, cfg, installSvc, tel)
	view.SetSize(100, 24)

	// Simulate installed skill
	view.HandleManageSkillsLoaded(ManageSkillsLoadedMsg{
		Skills: []installer.InstalledSkillSummary{
			{Slug: "test-skill", Locations: map[installer.Platform][]installer.InstallScope{
				installer.PlatformClaude: {installer.ScopeGlobal},
			}},
		},
		Err: nil,
	})

	// Add discovered skill (external - not managed by Skulto)
	discovery := &models.DiscoveredSkill{
		Platform: "claude",
		Scope:    "global",
		Path:     "/test/external-skill",
		Name:     "external-skill",
	}
	discovery.ID = discovery.GenerateID()
	err := database.UpsertDiscoveredSkill(discovery)
	require.NoError(t, err)

	view.HandleDiscoveriesLoaded(DiscoveriesLoadedMsg{
		Skills: []models.DiscoveredSkill{*discovery},
		Err:    nil,
	})

	// Switch to discovered section and check output
	view.Update("tab")
	output := view.View()

	// Should show the discovered skill with platform info
	assert.Contains(t, output, "external-skill")
	assert.Contains(t, output, "claude")
}

func TestManageView_DiscoveriesLoadedMsg(t *testing.T) {
	database := setupManageTestDB(t)
	defer func() { _ = database.Close() }()

	cfg := &config.Config{}
	tel := manageTestTelemetry()
	installSvc := installer.NewInstallService(database, cfg, tel)
	view := NewManageView(database, cfg, installSvc, tel)
	view.SetSize(80, 24)

	// Initially no discoveries
	assert.Empty(t, view.GetDiscoveries())

	// Add discoveries to db
	discovery := &models.DiscoveredSkill{
		Platform: "claude",
		Scope:    "global",
		Path:     "/test/skill",
		Name:     "test-discovery",
	}
	discovery.ID = discovery.GenerateID()
	err := database.UpsertDiscoveredSkill(discovery)
	require.NoError(t, err)

	// Handle loaded message
	view.HandleDiscoveriesLoaded(DiscoveriesLoadedMsg{
		Skills: []models.DiscoveredSkill{*discovery},
		Err:    nil,
	})

	// Should now have discoveries
	assert.Len(t, view.GetDiscoveries(), 1)
	assert.Equal(t, "test-discovery", view.GetDiscoveries()[0].Name)
}

func TestManageView_SectionTabsRendering(t *testing.T) {
	database := setupManageTestDB(t)
	defer func() { _ = database.Close() }()

	cfg := &config.Config{}
	tel := manageTestTelemetry()
	installSvc := installer.NewInstallService(database, cfg, tel)
	view := NewManageView(database, cfg, installSvc, tel)
	view.SetSize(80, 24)

	view.HandleManageSkillsLoaded(ManageSkillsLoadedMsg{
		Skills: []installer.InstalledSkillSummary{{Slug: "test"}},
		Err:    nil,
	})
	view.HandleDiscoveriesLoaded(DiscoveriesLoadedMsg{
		Skills: []models.DiscoveredSkill{{Name: "discovered"}},
		Err:    nil,
	})

	output := view.View()

	// Should show section tabs with counts
	assert.True(t, strings.Contains(output, "INSTALLED") || strings.Contains(output, "Installed"))
	assert.True(t, strings.Contains(output, "DISCOVERED") || strings.Contains(output, "Discovered"))
}

func TestManageView_FooterShowsTabHint(t *testing.T) {
	database := setupManageTestDB(t)
	defer func() { _ = database.Close() }()

	cfg := &config.Config{}
	tel := manageTestTelemetry()
	installSvc := installer.NewInstallService(database, cfg, tel)
	view := NewManageView(database, cfg, installSvc, tel)
	view.SetSize(80, 24)

	view.HandleManageSkillsLoaded(ManageSkillsLoadedMsg{
		Skills: []installer.InstalledSkillSummary{{Slug: "test"}},
		Err:    nil,
	})
	view.HandleDiscoveriesLoaded(DiscoveriesLoadedMsg{
		Skills: []models.DiscoveredSkill{{Name: "discovered"}},
		Err:    nil,
	})

	output := view.View()

	// Footer should mention Tab for switching sections
	assert.Contains(t, output, "Tab")
}

func TestManageView_ShowsLocalBadgeForInstalledSkills(t *testing.T) {
	database := setupManageTestDB(t)
	defer func() { _ = database.Close() }()

	cfg := &config.Config{}
	tel := manageTestTelemetry()
	installSvc := installer.NewInstallService(database, cfg, tel)
	view := NewManageView(database, cfg, installSvc, tel)
	view.SetSize(100, 24)

	// Simulate installed skills - one local, one remote
	view.HandleManageSkillsLoaded(ManageSkillsLoadedMsg{
		Skills: []installer.InstalledSkillSummary{
			{
				Slug:    "local-skill",
				Title:   "Local Skill",
				IsLocal: true,
				Locations: map[installer.Platform][]installer.InstallScope{
					installer.PlatformClaude: {installer.ScopeGlobal},
				},
			},
			{
				Slug:    "remote-skill",
				Title:   "Remote Skill",
				IsLocal: false,
				Locations: map[installer.Platform][]installer.InstallScope{
					installer.PlatformClaude: {installer.ScopeGlobal},
				},
			},
		},
		Err: nil,
	})

	output := view.View()

	// Local skill should show [local] badge
	assert.Contains(t, output, "[local]", "Local skill should show [local] badge")
	// Both skills should be visible
	assert.Contains(t, output, "local-skill")
	assert.Contains(t, output, "remote-skill")
}
