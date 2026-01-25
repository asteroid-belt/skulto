package views

import (
	"strings"
	"testing"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/telemetry"
)

// setupTestDB creates an in-memory test database with sample data
func setupTestDB(t *testing.T) *db.DB {
	t.Helper()

	database, err := db.New(db.Config{
		Path:        ":memory:",
		Debug:       false,
		MaxIdleConn: 1,
		MaxOpenConn: 1,
	})
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	testSkill := &models.Skill{
		ID:          "test-skill-id",
		Title:       "Test Skill Title",
		Description: "A test skill for unit testing",
		Content:     "# Test Content\n\nThis is test markdown content.",
		Slug:        "test-skill",
	}

	if err := database.CreateSkill(testSkill); err != nil {
		t.Fatalf("failed to create test skill: %v", err)
	}

	return database
}

// TestAsyncLoadingFlow simulates the complete Bubble Tea message flow:
// 1. User triggers load -> loading state shown immediately
// 2. Command executes (async in real app) -> returns SkillLoadedMsg
// 3. Message handled -> content shown
//
// This tests the actual user-visible behavior: "loading indicator appears
// before content loads"
func TestAsyncLoadingFlow(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	cfg := &config.Config{}
	view := NewDetailView(database, cfg)
	view.Init(telemetry.New(nil))
	view.SetSize(80, 24)

	// === STEP 1: User selects a skill (triggers SetSkill) ===
	cmd := view.SetSkill("test-skill-id")

	// At this point, BEFORE the command executes:
	// - loading should be true
	// - skill should be nil (not loaded yet)
	// - View() should show loading indicator
	if !view.loading {
		t.Fatal("after SetSkill: loading should be true")
	}
	if view.skill != nil {
		t.Fatal("after SetSkill: skill should be nil (async load not complete)")
	}

	loadingOutput := view.View()
	if !strings.Contains(loadingOutput, "Loading") {
		t.Errorf("after SetSkill: View() should show loading indicator\nGot: %s", loadingOutput)
	}

	// === STEP 2: Bubble Tea executes the command (simulated) ===
	// In real app, this runs in a goroutine. We execute synchronously in test.
	msg := cmd()

	// Verify the command returns the correct message type
	skillMsg, ok := msg.(SkillLoadedMsg)
	if !ok {
		t.Fatalf("command should return SkillLoadedMsg, got %T", msg)
	}

	// State should STILL be loading (message not handled yet)
	if !view.loading {
		t.Fatal("before HandleSkillLoaded: loading should still be true")
	}

	// === STEP 3: Bubble Tea delivers message to Update (simulated) ===
	view.HandleSkillLoaded(skillMsg)

	// Now the load is complete:
	// - loading should be false
	// - skill should be populated
	// - View() should show skill content
	if view.loading {
		t.Fatal("after HandleSkillLoaded: loading should be false")
	}
	if view.skill == nil {
		t.Fatal("after HandleSkillLoaded: skill should be loaded")
	}
	if view.skill.Title != "Test Skill Title" {
		t.Errorf("expected title 'Test Skill Title', got '%s'", view.skill.Title)
	}

	contentOutput := view.View()
	if strings.Contains(contentOutput, "Loading") {
		t.Error("after HandleSkillLoaded: View() should not show loading indicator")
	}
	if !strings.Contains(contentOutput, "Test Skill Title") {
		t.Errorf("after HandleSkillLoaded: View() should show skill title\nGot: %s", contentOutput)
	}
}

// TestAsyncLoadingFlowWithError verifies the error path works correctly
func TestAsyncLoadingFlowWithError(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	cfg := &config.Config{}
	view := NewDetailView(database, cfg)
	view.Init(telemetry.New(nil))
	view.SetSize(80, 24)

	// Load a non-existent skill
	cmd := view.SetSkill("non-existent-id")

	// Should be in loading state
	if !view.loading {
		t.Fatal("should be loading")
	}

	// Execute command
	msg := cmd()
	skillMsg := msg.(SkillLoadedMsg)

	// Handle the "not found" case
	view.HandleSkillLoaded(skillMsg)

	// Should show error state
	if view.loading {
		t.Fatal("should not be loading after completion")
	}
	if view.loadError == nil {
		t.Fatal("should have loadError for non-existent skill")
	}

	errorOutput := view.View()
	if !strings.Contains(errorOutput, "not found") && !strings.Contains(errorOutput, "failed") {
		t.Errorf("View() should show error message\nGot: %s", errorOutput)
	}
}

// TestLoadingStateIsRenderable verifies the loading state produces valid output
// This is important because if View() panics or returns empty during loading,
// the user would see a broken UI
func TestLoadingStateIsRenderable(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	cfg := &config.Config{}
	view := NewDetailView(database, cfg)
	view.Init(telemetry.New(nil))
	view.SetSize(80, 24)

	// Trigger load but don't complete it
	_ = view.SetSkill("test-skill-id")

	// View should render without panic and produce non-empty output
	output := view.View()

	if output == "" {
		t.Fatal("View() should not return empty string during loading")
	}
	if len(output) < 10 {
		t.Errorf("View() output seems too short: %q", output)
	}
}

// TestMultipleLoadsResetState verifies that starting a new load properly
// resets the previous state (no stale data shown)
func TestMultipleLoadsResetState(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	// Create a second skill
	skill2 := &models.Skill{
		ID:      "skill-2",
		Title:   "Second Skill",
		Content: "# Second",
		Slug:    "skill-2",
	}
	if err := database.CreateSkill(skill2); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{}
	view := NewDetailView(database, cfg)
	view.Init(telemetry.New(nil))
	view.SetSize(80, 24)

	// Load first skill completely
	cmd1 := view.SetSkill("test-skill-id")
	view.HandleSkillLoaded(cmd1().(SkillLoadedMsg))

	if view.skill.Title != "Test Skill Title" {
		t.Fatalf("expected first skill, got %s", view.skill.Title)
	}

	// Start loading second skill
	_ = view.SetSkill("skill-2")

	// During loading of second skill:
	// - Should be in loading state
	// - Previous skill data should be cleared (not shown)
	if !view.loading {
		t.Fatal("should be loading")
	}
	if view.skill != nil {
		t.Error("previous skill should be cleared when loading new skill")
	}

	output := view.View()
	if strings.Contains(output, "Test Skill Title") {
		t.Error("should not show previous skill title while loading new skill")
	}
}

// TestHandleSkillLoadedIsIdempotent verifies that handling the same message
// twice doesn't cause issues (defensive programming)
func TestHandleSkillLoadedIsIdempotent(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	cfg := &config.Config{}
	view := NewDetailView(database, cfg)
	view.Init(telemetry.New(nil))
	view.SetSize(80, 24)

	cmd := view.SetSkill("test-skill-id")
	msg := cmd().(SkillLoadedMsg)

	// Handle once
	view.HandleSkillLoaded(msg)
	firstOutput := view.View()

	// Handle again (shouldn't panic or corrupt state)
	view.HandleSkillLoaded(msg)
	secondOutput := view.View()

	if firstOutput != secondOutput {
		t.Error("handling same message twice should produce same output")
	}
}
