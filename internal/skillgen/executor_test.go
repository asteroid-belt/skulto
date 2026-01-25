package skillgen

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestCLIExecutor_AvailableTools(t *testing.T) {
	executor := NewCLIExecutor()
	tools := executor.AvailableTools()

	// This test verifies the method runs without error
	// Actual available tools depend on the system
	t.Logf("Available tools: %v", tools)
}

func TestCLIExecutor_findCLI_UnknownTool(t *testing.T) {
	executor := NewCLIExecutor()
	_, err := executor.findCLI(AITool("unknown"))

	if err == nil {
		t.Error("Expected error for unknown tool")
	}
}

func TestCLIExecutor_buildCommand_Claude(t *testing.T) {
	executor := NewCLIExecutor()
	ctx := context.Background()

	cmd := executor.buildCommand(ctx, AIToolClaude, "/usr/bin/claude", "test prompt")

	args := cmd.Args
	if len(args) < 4 {
		t.Fatalf("Expected at least 4 args, got %d", len(args))
	}

	// Check -p flag is present
	if !slices.Contains(args, "-p") {
		t.Error("Expected -p flag in claude command")
	}
}

func TestCLIExecutor_buildCommand_Codex(t *testing.T) {
	executor := NewCLIExecutor()
	ctx := context.Background()

	cmd := executor.buildCommand(ctx, AIToolCodex, "/usr/bin/codex", "test prompt")

	args := cmd.Args
	if !slices.Contains(args, "-q") {
		t.Error("Expected -q flag in codex command")
	}
}

func TestCLIExecutor_buildCommand_OpenCode(t *testing.T) {
	executor := NewCLIExecutor()
	ctx := context.Background()

	cmd := executor.buildCommand(ctx, AIToolOpenCode, "/usr/bin/opencode", "test prompt")

	args := cmd.Args
	// opencode uses: opencode run "prompt"
	if !slices.Contains(args, "run") {
		t.Error("Expected 'run' subcommand in opencode command")
	}
	if !slices.Contains(args, "test prompt") {
		t.Error("Expected prompt in opencode command args")
	}
}

func TestCLIExecutor_Execute_ToolNotFound(t *testing.T) {
	executor := NewCLIExecutor()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use a tool that definitely doesn't exist
	_, err := executor.Execute(ctx, AITool("nonexistent-tool-xyz"), "test")

	if err == nil {
		t.Error("Expected error when tool not found")
	}
}

func TestStreamReader_Cancel(t *testing.T) {
	sr := &StreamReader{
		done: false,
	}

	sr.Cancel()

	if !sr.done {
		t.Error("Expected done to be true after cancel")
	}
}

func TestAITool_String(t *testing.T) {
	tests := []struct {
		tool     AITool
		expected string
	}{
		{AIToolClaude, "claude"},
		{AIToolCodex, "codex"},
		{AIToolOpenCode, "opencode"},
	}

	for _, tt := range tests {
		if string(tt.tool) != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, string(tt.tool))
		}
	}
}

func TestCLIExecutor_InteractiveCommand_Claude(t *testing.T) {
	executor := NewCLIExecutor()

	cmd, err := executor.InteractiveCommand(AIToolClaude, "test prompt")

	// Only test if claude is available
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	// Check command has --system-prompt flag
	if !slices.Contains(cmd.Args, "--system-prompt") {
		t.Error("Expected --system-prompt flag in claude interactive command")
	}

	// Check user prompt is in args
	if !slices.Contains(cmd.Args, "test prompt") {
		t.Error("Expected user prompt in command args")
	}
}

func TestCLIExecutor_InteractiveCommand_ContainsAbsolutePath(t *testing.T) {
	executor := NewCLIExecutor()

	cmd, err := executor.InteractiveCommand(AIToolClaude, "test prompt")

	// Only test if claude is available
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	// Find the system prompt argument (it's after --system-prompt)
	var systemPrompt string
	for i, arg := range cmd.Args {
		if arg == "--system-prompt" && i+1 < len(cmd.Args) {
			systemPrompt = cmd.Args[i+1]
			break
		}
	}

	if systemPrompt == "" {
		t.Fatal("Could not find system prompt in command args")
	}

	// Verify it contains an absolute path, not ~
	expectedDir, _ := SkillsDir()
	if !strings.Contains(systemPrompt, expectedDir) {
		t.Errorf("System prompt should contain absolute path %s", expectedDir)
	}

	// Verify it does NOT contain ~ as a path reference
	if strings.Contains(systemPrompt, "~/.skulto") {
		t.Error("System prompt should not contain ~ as path, should use absolute path")
	}
}

func TestCLIExecutor_InteractiveCommand_UnknownTool(t *testing.T) {
	executor := NewCLIExecutor()

	_, err := executor.InteractiveCommand(AITool("unknown"), "test")

	if err == nil {
		t.Error("Expected error for unknown tool")
	}
}

func TestCLIExecutor_InteractiveCommand_OpenCode(t *testing.T) {
	executor := NewCLIExecutor()

	cmd, err := executor.InteractiveCommand(AIToolOpenCode, "test prompt")

	// Only test if opencode is available
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	// Check command has 'run' subcommand
	if !slices.Contains(cmd.Args, "run") {
		t.Error("Expected 'run' subcommand in opencode interactive command")
	}

	// Check the combined prompt is in args (contains user prompt)
	foundPrompt := false
	for _, arg := range cmd.Args {
		if strings.Contains(arg, "test prompt") {
			foundPrompt = true
			break
		}
	}
	if !foundPrompt {
		t.Error("Expected user prompt in command args")
	}
}

func TestScanSkills(t *testing.T) {
	// This test just verifies the function runs without error
	skills, err := ScanSkills()
	if err != nil {
		t.Logf("ScanSkills returned error (may be expected if no skills folder): %v", err)
	}
	t.Logf("Found %d skills", len(skills))
}

func TestFindNewSkills(t *testing.T) {
	// Use fixed times to avoid precision issues
	oldTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	newTime := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	before := []SkillInfo{
		{Slug: "skill-a", ModTime: oldTime},
		{Slug: "skill-b", ModTime: oldTime},
	}

	after := []SkillInfo{
		{Slug: "skill-a", ModTime: oldTime}, // unchanged
		{Slug: "skill-b", ModTime: newTime}, // modified
		{Slug: "skill-c", ModTime: newTime}, // new
	}

	newSkills := FindNewSkills(before, after)

	if len(newSkills) != 2 {
		t.Errorf("Expected 2 new/modified skills, got %d", len(newSkills))
	}

	// Check that skill-b and skill-c are in the result
	slugs := make(map[string]bool)
	for _, s := range newSkills {
		slugs[s.Slug] = true
	}

	if !slugs["skill-b"] {
		t.Error("Expected skill-b to be detected as modified")
	}
	if !slugs["skill-c"] {
		t.Error("Expected skill-c to be detected as new")
	}
}

// --- CWD Skill Scanning Tests ---

func TestCwdSkillsDir(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, ".skulto", "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatalf("Failed to create skills dir: %v", err)
	}

	// Change to temp directory
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}

	dir, err := CwdSkillsDir()
	if err != nil {
		t.Fatalf("CwdSkillsDir() error = %v", err)
	}

	// Resolve symlinks for comparison (macOS uses /private/var instead of /var)
	expectedDir, _ := filepath.EvalSymlinks(skillsDir)
	if dir != expectedDir {
		t.Errorf("CwdSkillsDir() = %q, want %q", dir, expectedDir)
	}
}

func TestCwdSkillsDir_NotExists(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}

	dir, err := CwdSkillsDir()
	if err != nil {
		t.Fatalf("CwdSkillsDir() error = %v", err)
	}
	if dir != "" {
		t.Errorf("CwdSkillsDir() should return empty string when dir doesn't exist, got %q", dir)
	}
}

func TestScanCwdSkills(t *testing.T) {
	// Create temp directory with skills
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, ".skulto", "skills")

	// Create two skill directories
	skill1Dir := filepath.Join(skillsDir, "my-skill")
	skill2Dir := filepath.Join(skillsDir, "another-skill")
	if err := os.MkdirAll(skill1Dir, 0755); err != nil {
		t.Fatalf("Failed to create skill1 dir: %v", err)
	}
	if err := os.MkdirAll(skill2Dir, 0755); err != nil {
		t.Fatalf("Failed to create skill2 dir: %v", err)
	}

	// Create skill.md files
	if err := os.WriteFile(filepath.Join(skill1Dir, "skill.md"), []byte("# My Skill"), 0644); err != nil {
		t.Fatalf("Failed to write skill1.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skill2Dir, "skill.md"), []byte("# Another"), 0644); err != nil {
		t.Fatalf("Failed to write skill2.md: %v", err)
	}

	// Change to temp directory
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}

	skills, err := ScanCwdSkills()
	if err != nil {
		t.Fatalf("ScanCwdSkills() error = %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("ScanCwdSkills() returned %d skills, want 2", len(skills))
	}

	slugs := []string{skills[0].Slug, skills[1].Slug}
	hasMySkill := slices.Contains(slugs, "my-skill")
	hasAnotherSkill := slices.Contains(slugs, "another-skill")

	if !hasMySkill {
		t.Error("Expected to find 'my-skill'")
	}
	if !hasAnotherSkill {
		t.Error("Expected to find 'another-skill'")
	}
}

func TestScanCwdSkills_FlatStructure(t *testing.T) {
	// Create temp directory with flat skills only (name/skill.md structure)
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, ".skulto", "skills")

	// Create flat skill directories: skill-name/skill.md
	skill1Dir := filepath.Join(skillsDir, "dadjoke")
	skill2Dir := filepath.Join(skillsDir, "song-lyrics")
	skill3Dir := filepath.Join(skillsDir, "teleport")

	for _, dir := range []string{skill1Dir, skill2Dir, skill3Dir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
	}

	// Create skill.md files (test both lowercase and uppercase)
	if err := os.WriteFile(filepath.Join(skill1Dir, "SKILL.md"), []byte("# Dad Joke"), 0644); err != nil {
		t.Fatalf("Failed to write SKILL.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skill2Dir, "skill.md"), []byte("# Song Lyrics"), 0644); err != nil {
		t.Fatalf("Failed to write skill.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skill3Dir, "skill.md"), []byte("# Teleport"), 0644); err != nil {
		t.Fatalf("Failed to write skill.md: %v", err)
	}

	// Change to temp directory
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}

	skills, err := ScanCwdSkillsWithCategory()
	if err != nil {
		t.Fatalf("ScanCwdSkillsWithCategory() error = %v", err)
	}
	if len(skills) != 3 {
		t.Fatalf("ScanCwdSkillsWithCategory() returned %d skills, want 3", len(skills))
	}

	// Build a map for easier testing
	skillMap := make(map[string]CwdSkillInfo)
	for _, s := range skills {
		skillMap[s.Slug] = s
	}

	// Verify all skills have empty category (flat structure)
	for slug, skill := range skillMap {
		if skill.Category != "" {
			t.Errorf("skill %q category = %q, want empty (flat structure)", slug, skill.Category)
		}
	}

	// Verify all expected skills are found
	expectedSlugs := []string{"dadjoke", "song-lyrics", "teleport"}
	for _, slug := range expectedSlugs {
		if _, ok := skillMap[slug]; !ok {
			t.Errorf("Expected to find skill %q", slug)
		}
	}
}

func TestScanCwdSkills_NestedDirectoriesSkipped(t *testing.T) {
	// Create temp directory with nested structure that should be SKIPPED
	// Skills should only be found at top-level (skills/<name>/skill.md)
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, ".skulto", "skills")

	// Create nested skill directories (should be skipped - no SKILL.md at top level)
	nestedSkillDir := filepath.Join(skillsDir, "jokes", "dadjoke")
	if err := os.MkdirAll(nestedSkillDir, 0755); err != nil {
		t.Fatalf("Failed to create nested dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nestedSkillDir, "skill.md"), []byte("# Dad Joke"), 0644); err != nil {
		t.Fatalf("Failed to write skill.md: %v", err)
	}

	// Create a valid flat skill for comparison
	flatSkillDir := filepath.Join(skillsDir, "teleport")
	if err := os.MkdirAll(flatSkillDir, 0755); err != nil {
		t.Fatalf("Failed to create flat dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(flatSkillDir, "skill.md"), []byte("# Teleport"), 0644); err != nil {
		t.Fatalf("Failed to write skill.md: %v", err)
	}

	// Change to temp directory
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}

	skills, err := ScanCwdSkillsWithCategory()
	if err != nil {
		t.Fatalf("ScanCwdSkillsWithCategory() error = %v", err)
	}

	// Should only find the flat skill (teleport), not the nested one (dadjoke)
	if len(skills) != 1 {
		t.Fatalf("ScanCwdSkillsWithCategory() returned %d skills, want 1 (only flat skills)", len(skills))
	}

	if skills[0].Slug != "teleport" {
		t.Errorf("Expected to find 'teleport', got %q", skills[0].Slug)
	}

	if skills[0].Category != "" {
		t.Errorf("Category should be empty for flat structure, got %q", skills[0].Category)
	}
}

func TestScanCwdSkills_CategoryAlwaysEmpty(t *testing.T) {
	// Verify that Category is always empty for all discovered skills
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, ".skulto", "skills")

	// Create multiple flat skills
	for _, name := range []string{"skill-a", "skill-b", "skill-c"} {
		dir := filepath.Join(skillsDir, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "skill.md"), []byte("# "+name), 0644); err != nil {
			t.Fatalf("Failed to write skill.md: %v", err)
		}
	}

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}

	skills, err := ScanCwdSkillsWithCategory()
	if err != nil {
		t.Fatalf("ScanCwdSkillsWithCategory() error = %v", err)
	}

	if len(skills) != 3 {
		t.Fatalf("Expected 3 skills, got %d", len(skills))
	}

	// Verify ALL skills have empty category
	for _, skill := range skills {
		if skill.Category != "" {
			t.Errorf("Skill %q has non-empty category %q, expected empty", skill.Slug, skill.Category)
		}
	}
}

func TestScanCwdSkills_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, ".skulto", "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatalf("Failed to create skills dir: %v", err)
	}

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}

	skills, err := ScanCwdSkills()
	if err != nil {
		t.Fatalf("ScanCwdSkills() error = %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("ScanCwdSkills() should return empty slice for empty dir, got %d skills", len(skills))
	}
}

func TestScanCwdSkills_NoCwdFolder(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}

	skills, err := ScanCwdSkills()
	if err != nil {
		t.Fatalf("ScanCwdSkills() error = %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("ScanCwdSkills() should return empty when no .skulto/skills dir, got %v", skills)
	}
}
