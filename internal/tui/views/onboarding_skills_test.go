package views

import (
	"testing"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/stretchr/testify/assert"
)

// mockOnboardingDB implements OnboardingSkillsDB for testing.
type mockOnboardingDB struct {
	installations map[string]bool // skillID -> hasInstallations
}

func newMockOnboardingDB() *mockOnboardingDB {
	return &mockOnboardingDB{
		installations: make(map[string]bool),
	}
}

func (m *mockOnboardingDB) HasInstallations(skillID string) (bool, error) {
	return m.installations[skillID], nil
}

func (m *mockOnboardingDB) setInstalled(skillID string, installed bool) {
	m.installations[skillID] = installed
}

func TestOnboardingSkillsViewInit(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	mockDB := newMockOnboardingDB()

	v := NewOnboardingSkillsView(cfg, mockDB)
	v.Init()

	assert.True(t, v.loading)
	assert.Nil(t, v.err)
	assert.Equal(t, 0, len(v.newSkills))
	assert.Equal(t, 0, len(v.installedSkills))
}

func TestOnboardingSkillsViewHandlesFetch(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	mockDB := newMockOnboardingDB()

	v := NewOnboardingSkillsView(cfg, mockDB)
	v.Init()

	skills := []models.Skill{
		{ID: "1", Slug: "skill-a", Title: "Skill A", Description: "Desc A"},
		{ID: "2", Slug: "skill-b", Title: "Skill B", Description: "Desc B"},
	}

	v.HandleSkillsFetched(skills, nil)

	assert.False(t, v.loading)
	assert.Nil(t, v.err)
	assert.Equal(t, 2, len(v.newSkills))
	assert.Equal(t, 0, len(v.installedSkills))

	// All new skills should be pre-selected
	for _, item := range v.newSkills {
		assert.True(t, item.Selected)
	}
}

func TestOnboardingSkillsViewClassifiesExisting(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	mockDB := newMockOnboardingDB()
	// Mark skill "2" as having installations
	mockDB.setInstalled("2", true)

	v := NewOnboardingSkillsView(cfg, mockDB)
	v.Init()

	skills := []models.Skill{
		{ID: "1", Slug: "new-skill", Title: "New Skill"},
		{ID: "2", Slug: "existing-skill", Title: "Existing Skill"},
	}

	v.HandleSkillsFetched(skills, nil)

	assert.Equal(t, 1, len(v.newSkills))
	assert.Equal(t, 1, len(v.installedSkills))

	// New skill should be selected
	assert.True(t, v.newSkills[0].Selected)

	// Existing skill should NOT be selected by default
	assert.False(t, v.installedSkills[0].Selected)
	assert.True(t, v.installedSkills[0].AlreadyInstalled)
}

func TestOnboardingSkillsViewNavigation(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	mockDB := newMockOnboardingDB()

	v := NewOnboardingSkillsView(cfg, mockDB)
	v.Init()

	skills := []models.Skill{
		{ID: "1", Slug: "skill-a", Title: "A"},
		{ID: "2", Slug: "skill-b", Title: "B"},
	}
	v.HandleSkillsFetched(skills, nil)

	// Initial position
	assert.Equal(t, 0, v.currentIndex)
	assert.Equal(t, 0, v.currentSection)

	// Move down
	v.Update("down")
	assert.Equal(t, 1, v.currentIndex)

	// Move up
	v.Update("up")
	assert.Equal(t, 0, v.currentIndex)
}

func TestOnboardingSkillsViewToggle(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	mockDB := newMockOnboardingDB()

	v := NewOnboardingSkillsView(cfg, mockDB)
	v.Init()

	skills := []models.Skill{
		{ID: "1", Slug: "skill-a", Title: "A"},
	}
	v.HandleSkillsFetched(skills, nil)

	// Initially selected
	assert.True(t, v.newSkills[0].Selected)

	// Toggle off
	v.Update("space")
	assert.False(t, v.newSkills[0].Selected)

	// Toggle on
	v.Update("space")
	assert.True(t, v.newSkills[0].Selected)
}

func TestOnboardingSkillsViewSelectAll(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	mockDB := newMockOnboardingDB()

	v := NewOnboardingSkillsView(cfg, mockDB)
	v.Init()

	skills := []models.Skill{
		{ID: "1", Slug: "skill-a", Title: "A"},
		{ID: "2", Slug: "skill-b", Title: "B"},
	}
	v.HandleSkillsFetched(skills, nil)

	// Deselect all first
	v.Update("n")
	assert.False(t, v.newSkills[0].Selected)
	assert.False(t, v.newSkills[1].Selected)

	// Select all
	v.Update("a")
	assert.True(t, v.newSkills[0].Selected)
	assert.True(t, v.newSkills[1].Selected)
}

func TestOnboardingSkillsViewSkip(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	mockDB := newMockOnboardingDB()

	v := NewOnboardingSkillsView(cfg, mockDB)
	v.Init()
	v.loading = false

	done, skipped, _ := v.Update("esc")

	assert.True(t, done)
	assert.True(t, skipped)
}

func TestOnboardingSkillsViewContinue(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	mockDB := newMockOnboardingDB()

	v := NewOnboardingSkillsView(cfg, mockDB)
	v.Init()
	v.loading = false

	done, skipped, _ := v.Update("enter")

	assert.True(t, done)
	assert.False(t, skipped)
}

func TestOnboardingSkillsViewGetSelected(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	mockDB := newMockOnboardingDB()

	v := NewOnboardingSkillsView(cfg, mockDB)
	v.Init()

	skills := []models.Skill{
		{ID: "1", Slug: "skill-a", Title: "A"},
		{ID: "2", Slug: "skill-b", Title: "B"},
	}
	v.HandleSkillsFetched(skills, nil)

	// Deselect one
	v.Update("space") // Toggle first one off

	selected := v.GetSelectedSkills()
	assert.Equal(t, 1, len(selected))
	assert.Equal(t, "B", selected[0].Title)
}

func TestOnboardingSkillsViewGetReplaceSkills(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	mockDB := newMockOnboardingDB()
	mockDB.setInstalled("2", true)
	mockDB.setInstalled("3", true)

	v := NewOnboardingSkillsView(cfg, mockDB)
	v.Init()

	skills := []models.Skill{
		{ID: "1", Slug: "new-skill", Title: "New Skill"},
		{ID: "2", Slug: "installed-a", Title: "Installed A"},
		{ID: "3", Slug: "installed-b", Title: "Installed B"},
	}
	v.HandleSkillsFetched(skills, nil)

	// Installed skills start unselected, select one
	v.currentSection = 1
	v.currentIndex = 0
	v.Update("space") // Toggle first installed skill on

	replace := v.GetReplaceSkills()
	assert.Equal(t, 1, len(replace))
	assert.Equal(t, "Installed A", replace[0].Title)
}

func TestOnboardingSkillsViewHandlesError(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	mockDB := newMockOnboardingDB()

	v := NewOnboardingSkillsView(cfg, mockDB)
	v.Init()

	testErr := assert.AnError
	v.HandleSkillsFetched(nil, testErr)

	assert.False(t, v.loading)
	assert.Equal(t, testErr, v.err)
	assert.Equal(t, testErr.Error(), v.errorMsg)
}

func TestOnboardingSkillsViewNavigationCrossSection(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	mockDB := newMockOnboardingDB()
	mockDB.setInstalled("2", true)

	v := NewOnboardingSkillsView(cfg, mockDB)
	v.Init()

	skills := []models.Skill{
		{ID: "1", Slug: "new-skill", Title: "New"},
		{ID: "2", Slug: "installed", Title: "Installed"},
	}
	v.HandleSkillsFetched(skills, nil)

	// Start in new section
	assert.Equal(t, 0, v.currentSection)
	assert.Equal(t, 0, v.currentIndex)

	// Move down should go to installed section
	v.Update("down")
	assert.Equal(t, 1, v.currentSection)
	assert.Equal(t, 0, v.currentIndex)

	// Move up should go back to new section
	v.Update("up")
	assert.Equal(t, 0, v.currentSection)
	assert.Equal(t, 0, v.currentIndex)
}

func TestOnboardingSkillsViewRenderOutput(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	mockDB := newMockOnboardingDB()

	v := NewOnboardingSkillsView(cfg, mockDB)
	v.SetSize(100, 50)
	v.Init()

	// Test loading state render
	output := v.View()
	assert.Contains(t, output, "Fetching skills")

	// Test with skills
	skills := []models.Skill{
		{ID: "1", Slug: "test-skill", Title: "Test Skill"},
	}
	v.HandleSkillsFetched(skills, nil)

	output = v.View()
	assert.Contains(t, output, "Select Skills")
	assert.Contains(t, output, "Test Skill")
}

func TestOnboardingSkillsViewGetKeyboardCommands(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	mockDB := newMockOnboardingDB()

	v := NewOnboardingSkillsView(cfg, mockDB)
	commands := v.GetKeyboardCommands()

	assert.Equal(t, "Skills Onboarding", commands.ViewName)
	assert.True(t, len(commands.Commands) > 0)
}

func TestOnboardingSkillsViewRenderError(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	mockDB := newMockOnboardingDB()

	v := NewOnboardingSkillsView(cfg, mockDB)
	v.SetSize(100, 50)
	v.Init()

	testErr := assert.AnError
	v.HandleSkillsFetched(nil, testErr)

	output := v.View()
	assert.Contains(t, output, "Error")
	assert.Contains(t, output, testErr.Error())
}

func TestOnboardingSkillsViewErrorStateContinue(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	mockDB := newMockOnboardingDB()

	v := NewOnboardingSkillsView(cfg, mockDB)
	v.Init()
	v.HandleSkillsFetched(nil, assert.AnError)

	// Continue from error state
	done, skipped, _ := v.Update("enter")
	assert.True(t, done)
	assert.False(t, skipped)
}

func TestOnboardingSkillsViewErrorStateSkip(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	mockDB := newMockOnboardingDB()

	v := NewOnboardingSkillsView(cfg, mockDB)
	v.Init()
	v.HandleSkillsFetched(nil, assert.AnError)

	// Skip from error state
	done, skipped, _ := v.Update("esc")
	assert.True(t, done)
	assert.True(t, skipped)
}

func TestOnboardingSkillsViewRenderEmpty(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	mockDB := newMockOnboardingDB()

	v := NewOnboardingSkillsView(cfg, mockDB)
	v.SetSize(100, 50)
	v.Init()

	// Handle fetch with empty skills
	v.HandleSkillsFetched([]models.Skill{}, nil)

	output := v.View()
	assert.Contains(t, output, "No skills found")
}

func TestOnboardingSkillsViewEmptyStateContinue(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	mockDB := newMockOnboardingDB()

	v := NewOnboardingSkillsView(cfg, mockDB)
	v.Init()
	v.HandleSkillsFetched([]models.Skill{}, nil)

	// Continue from empty state
	done, skipped, _ := v.Update("enter")
	assert.True(t, done)
	assert.False(t, skipped)
}

func TestOnboardingSkillsViewEmptyStateSkip(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	mockDB := newMockOnboardingDB()

	v := NewOnboardingSkillsView(cfg, mockDB)
	v.Init()
	v.HandleSkillsFetched([]models.Skill{}, nil)

	// Skip from empty state
	done, skipped, _ := v.Update("esc")
	assert.True(t, done)
	assert.True(t, skipped)
}

func TestOnboardingSkillsViewVimNavigation(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	mockDB := newMockOnboardingDB()

	v := NewOnboardingSkillsView(cfg, mockDB)
	v.Init()

	skills := []models.Skill{
		{ID: "1", Slug: "skill-a", Title: "A"},
		{ID: "2", Slug: "skill-b", Title: "B"},
	}
	v.HandleSkillsFetched(skills, nil)

	// Test j to move down
	v.Update("j")
	assert.Equal(t, 1, v.currentIndex)

	// Test k to move up
	v.Update("k")
	assert.Equal(t, 0, v.currentIndex)
}

func TestOnboardingSkillsViewSpaceToggle(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	mockDB := newMockOnboardingDB()

	v := NewOnboardingSkillsView(cfg, mockDB)
	v.Init()

	skills := []models.Skill{
		{ID: "1", Slug: "skill-a", Title: "A"},
	}
	v.HandleSkillsFetched(skills, nil)

	// Initially selected
	assert.True(t, v.newSkills[0].Selected)

	// Toggle with space
	v.Update("space")
	assert.False(t, v.newSkills[0].Selected)

	// Enter should continue, not toggle
	done, _, _ := v.Update("enter")
	assert.True(t, done)
}

func TestOnboardingSkillsViewLoadingIgnoresKeys(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	mockDB := newMockOnboardingDB()

	v := NewOnboardingSkillsView(cfg, mockDB)
	v.Init()

	// While loading, all keys should be ignored
	done, skipped, _ := v.Update("c")
	assert.False(t, done)
	assert.False(t, skipped)

	done, skipped, _ = v.Update("s")
	assert.False(t, done)
	assert.False(t, skipped)
}

func TestOnboardingSkillsViewRenderInstalledSection(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	mockDB := newMockOnboardingDB()
	mockDB.setInstalled("2", true)

	v := NewOnboardingSkillsView(cfg, mockDB)
	v.SetSize(100, 50)
	v.Init()

	skills := []models.Skill{
		{ID: "1", Slug: "new-skill", Title: "New Skill"},
		{ID: "2", Slug: "installed", Title: "Installed Skill"},
	}
	v.HandleSkillsFetched(skills, nil)

	output := v.View()
	assert.Contains(t, output, "New Skills")
	assert.Contains(t, output, "Already Installed")
	assert.Contains(t, output, "replace?")
}

func TestOnboardingSkillsViewOnlyInstalledSkills(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	mockDB := newMockOnboardingDB()
	mockDB.setInstalled("1", true)
	mockDB.setInstalled("2", true)

	v := NewOnboardingSkillsView(cfg, mockDB)
	v.Init()

	// Only installed skills, no new ones
	skills := []models.Skill{
		{ID: "1", Slug: "installed-a", Title: "Installed A"},
		{ID: "2", Slug: "installed-b", Title: "Installed B"},
	}
	v.HandleSkillsFetched(skills, nil)

	// Should start in installed section since there are no new skills
	assert.Equal(t, 1, v.currentSection)
	assert.False(t, v.inNewSection)
	assert.Equal(t, 0, v.currentIndex)
}

func TestOnboardingSkillsViewNavigationBounds(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	mockDB := newMockOnboardingDB()
	mockDB.setInstalled("1", true)

	v := NewOnboardingSkillsView(cfg, mockDB)
	v.Init()

	skills := []models.Skill{
		{ID: "1", Slug: "installed-a", Title: "Installed A"},
	}
	v.HandleSkillsFetched(skills, nil)

	// Try to move up from first item - should stay at 0
	v.Update("up")
	assert.Equal(t, 0, v.currentIndex)

	// Try to move down from last item - should stay at 0
	v.Update("down")
	assert.Equal(t, 0, v.currentIndex)
}

func TestOnboardingSkillsViewToggleInstalledSection(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	mockDB := newMockOnboardingDB()
	mockDB.setInstalled("1", true)

	v := NewOnboardingSkillsView(cfg, mockDB)
	v.Init()

	skills := []models.Skill{
		{ID: "1", Slug: "installed", Title: "Installed"},
	}
	v.HandleSkillsFetched(skills, nil)

	// Should start in installed section
	assert.Equal(t, 1, v.currentSection)
	assert.False(t, v.installedSkills[0].Selected)

	// Toggle on
	v.Update("space")
	assert.True(t, v.installedSkills[0].Selected)

	// Toggle off
	v.Update("space")
	assert.False(t, v.installedSkills[0].Selected)
}

func TestOnboardingSkillsViewSkillWithoutTitle(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	mockDB := newMockOnboardingDB()

	v := NewOnboardingSkillsView(cfg, mockDB)
	v.SetSize(100, 50)
	v.Init()

	skills := []models.Skill{
		{ID: "1", Slug: "my-skill", Title: ""}, // No title, should fallback to slug
	}
	v.HandleSkillsFetched(skills, nil)

	output := v.View()
	assert.Contains(t, output, "my-skill")
}
