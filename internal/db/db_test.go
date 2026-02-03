package db

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/asteroid-belt/skulto/internal/models"
)

// testDB creates a temporary test database.
func testDB(t *testing.T) *DB {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := New(Config{
		Path:        dbPath,
		Debug:       false,
		MaxIdleConn: 1,
		MaxOpenConn: 1,
	})
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Logf("Failed to close test database: %v", err)
		}
	})

	return db
}

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "skulto.db")

	db, err := New(DefaultConfig(dbPath))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("Failed to close database: %v", err)
		}
	}()

	// Verify database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was not created")
	}

	// Verify path is stored correctly
	if db.Path() != dbPath {
		t.Errorf("Path() = %v, want %v", db.Path(), dbPath)
	}
}

func TestNew_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "nested", "dirs", "skulto.db")

	db, err := New(DefaultConfig(dbPath))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("Failed to close database: %v", err)
		}
	}()

	// Verify nested directories were created
	dir := filepath.Dir(dbPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("nested directories were not created")
	}
}

func TestGetStats_EmptyDB(t *testing.T) {
	db := testDB(t)

	stats, err := db.GetStats()
	if err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}

	if stats.TotalSkills != 0 {
		t.Errorf("TotalSkills = %d, want 0", stats.TotalSkills)
	}
	// Note: "mine" tag is created on DB init, so expect 1 tag
	if stats.TotalTags != 1 {
		t.Errorf("TotalTags = %d, want 1 (mine tag)", stats.TotalTags)
	}
	if stats.TotalSources != 0 {
		t.Errorf("TotalSources = %d, want 0", stats.TotalSources)
	}
}

// --- Skill Tests ---

func TestSkillCRUD(t *testing.T) {
	db := testDB(t)

	// Create
	skill := &models.Skill{
		ID:          "test-skill-001",
		Slug:        "test-skill",
		Title:       "Test Skill",
		Description: "A test skill for unit testing",
		Content:     "# Test Skill\n\nThis is test content.",
		Category:    "testing",
		Difficulty:  models.DifficultyBeginner,
		Author:      "test-author",
	}

	err := db.CreateSkill(skill)
	if err != nil {
		t.Fatalf("CreateSkill() error = %v", err)
	}

	// Read
	retrieved, err := db.GetSkill("test-skill-001")
	if err != nil {
		t.Fatalf("GetSkill() error = %v", err)
	}
	if retrieved == nil {
		t.Fatal("GetSkill() returned nil")
	}
	if retrieved.Title != "Test Skill" {
		t.Errorf("Title = %q, want %q", retrieved.Title, "Test Skill")
	}

	// Update
	retrieved.Title = "Updated Test Skill"
	err = db.UpdateSkill(retrieved)
	if err != nil {
		t.Fatalf("UpdateSkill() error = %v", err)
	}

	updated, err := db.GetSkill("test-skill-001")
	if err != nil {
		t.Fatalf("GetSkill() after update error = %v", err)
	}
	if updated.Title != "Updated Test Skill" {
		t.Errorf("Title after update = %q, want %q", updated.Title, "Updated Test Skill")
	}

	// Delete (soft)
	err = db.DeleteSkill("test-skill-001")
	if err != nil {
		t.Fatalf("DeleteSkill() error = %v", err)
	}

	deleted, err := db.GetSkill("test-skill-001")
	if err != nil {
		t.Fatalf("GetSkill() after delete error = %v", err)
	}
	if deleted != nil {
		t.Error("GetSkill() should return nil after soft delete")
	}
}

func TestGetSkillBySlug(t *testing.T) {
	db := testDB(t)

	skill := &models.Skill{
		ID:    "slug-test-001",
		Slug:  "my-unique-slug",
		Title: "Slug Test Skill",
	}
	if err := db.CreateSkill(skill); err != nil {
		t.Fatalf("CreateSkill() error = %v", err)
	}

	retrieved, err := db.GetSkillBySlug("my-unique-slug")
	if err != nil {
		t.Fatalf("GetSkillBySlug() error = %v", err)
	}
	if retrieved == nil {
		t.Fatal("GetSkillBySlug() returned nil")
	}
	if retrieved.ID != "slug-test-001" {
		t.Errorf("ID = %q, want %q", retrieved.ID, "slug-test-001")
	}

	// Non-existent slug
	notFound, err := db.GetSkillBySlug("non-existent")
	if err != nil {
		t.Fatalf("GetSkillBySlug() error = %v", err)
	}
	if notFound != nil {
		t.Error("GetSkillBySlug() should return nil for non-existent slug")
	}
}

func TestListSkills(t *testing.T) {
	db := testDB(t)

	// Create multiple skills
	for i := 0; i < 5; i++ {
		skill := &models.Skill{
			ID:    "list-test-" + string(rune('a'+i)),
			Slug:  "list-skill-" + string(rune('a'+i)),
			Title: "List Test Skill " + string(rune('A'+i)),
		}
		if err := db.CreateSkill(skill); err != nil {
			t.Fatalf("CreateSkill() error = %v", err)
		}
	}

	// List with pagination
	skills, err := db.ListSkills(3, 0)
	if err != nil {
		t.Fatalf("ListSkills() error = %v", err)
	}
	if len(skills) != 3 {
		t.Errorf("ListSkills(3, 0) returned %d skills, want 3", len(skills))
	}

	// List with offset
	skills, err = db.ListSkills(10, 2)
	if err != nil {
		t.Fatalf("ListSkills() error = %v", err)
	}
	if len(skills) != 3 {
		t.Errorf("ListSkills(10, 2) returned %d skills, want 3", len(skills))
	}
}

func TestUpsertSkill(t *testing.T) {
	db := testDB(t)

	skill := &models.Skill{
		ID:    "upsert-test-001",
		Slug:  "upsert-skill",
		Title: "Original Title",
	}

	// First upsert (insert)
	if err := db.UpsertSkill(skill); err != nil {
		t.Fatalf("UpsertSkill() insert error = %v", err)
	}

	retrieved, _ := db.GetSkill("upsert-test-001")
	if retrieved.Title != "Original Title" {
		t.Errorf("Title = %q, want %q", retrieved.Title, "Original Title")
	}

	// Second upsert (update)
	skill.Title = "Updated Title"
	if err := db.UpsertSkill(skill); err != nil {
		t.Fatalf("UpsertSkill() update error = %v", err)
	}

	retrieved, _ = db.GetSkill("upsert-test-001")
	if retrieved.Title != "Updated Title" {
		t.Errorf("Title after upsert = %q, want %q", retrieved.Title, "Updated Title")
	}
}

func TestGetTopSkills(t *testing.T) {
	db := testDB(t)

	// Create skills with different star counts
	stars := []int{100, 50, 200, 25, 150}
	for i, s := range stars {
		skill := &models.Skill{
			ID:    "stars-test-" + string(rune('a'+i)),
			Slug:  "stars-skill-" + string(rune('a'+i)),
			Title: "Stars Test Skill",
			Stars: s,
		}
		if err := db.CreateSkill(skill); err != nil {
			t.Fatalf("CreateSkill() error = %v", err)
		}
	}

	top, err := db.GetTopSkills(3)
	if err != nil {
		t.Fatalf("GetTopSkills() error = %v", err)
	}
	if len(top) != 3 {
		t.Fatalf("GetTopSkills(3) returned %d skills, want 3", len(top))
	}

	// Should be sorted by stars descending
	if top[0].Stars != 200 {
		t.Errorf("First skill stars = %d, want 200", top[0].Stars)
	}
	if top[1].Stars != 150 {
		t.Errorf("Second skill stars = %d, want 150", top[1].Stars)
	}
}

func TestGetSkillsByCategory(t *testing.T) {
	db := testDB(t)

	categories := []string{"web", "web", "backend", "web", "devops"}
	for i, cat := range categories {
		skill := &models.Skill{
			ID:       "cat-test-" + string(rune('a'+i)),
			Slug:     "cat-skill-" + string(rune('a'+i)),
			Title:    "Category Test Skill",
			Category: cat,
		}
		if err := db.CreateSkill(skill); err != nil {
			t.Fatalf("CreateSkill() error = %v", err)
		}
	}

	webSkills, err := db.GetSkillsByCategory("web", 10, 0)
	if err != nil {
		t.Fatalf("GetSkillsByCategory() error = %v", err)
	}
	if len(webSkills) != 3 {
		t.Errorf("GetSkillsByCategory('web') returned %d skills, want 3", len(webSkills))
	}
}

// --- Source Tests ---

func TestSourceCRUD(t *testing.T) {
	db := testDB(t)

	source := &models.Source{
		ID:          "anthropics/test-repo",
		Owner:       "anthropics",
		Repo:        "test-repo",
		FullName:    "anthropics/test-repo",
		Description: "Test repository",
		Priority:    10,
		IsOfficial:  true,
	}

	// Create
	err := db.CreateSource(source)
	if err != nil {
		t.Fatalf("CreateSource() error = %v", err)
	}

	// Read
	retrieved, err := db.GetSource("anthropics/test-repo")
	if err != nil {
		t.Fatalf("GetSource() error = %v", err)
	}
	if retrieved == nil {
		t.Fatal("GetSource() returned nil")
	}
	if retrieved.Owner != "anthropics" {
		t.Errorf("Owner = %q, want %q", retrieved.Owner, "anthropics")
	}

	// Delete
	err = db.DeleteSource("anthropics/test-repo")
	if err != nil {
		t.Fatalf("DeleteSource() error = %v", err)
	}

	deleted, err := db.GetSource("anthropics/test-repo")
	if err != nil {
		t.Fatalf("GetSource() after delete error = %v", err)
	}
	if deleted != nil {
		t.Error("GetSource() should return nil after delete")
	}
}

func TestListSources(t *testing.T) {
	db := testDB(t)

	// Create sources with different priorities
	priorities := []int{5, 10, 3, 8}
	for i, p := range priorities {
		source := &models.Source{
			ID:       "test-owner/repo-" + string(rune('a'+i)),
			Owner:    "test-owner",
			Repo:     "repo-" + string(rune('a'+i)),
			Priority: p,
		}
		if err := db.CreateSource(source); err != nil {
			t.Fatalf("CreateSource() error = %v", err)
		}
	}

	sources, err := db.ListSources()
	if err != nil {
		t.Fatalf("ListSources() error = %v", err)
	}
	if len(sources) != 4 {
		t.Fatalf("ListSources() returned %d sources, want 4", len(sources))
	}

	// Should be sorted by priority descending
	if sources[0].Priority != 10 {
		t.Errorf("First source priority = %d, want 10", sources[0].Priority)
	}
}

func TestGetOfficialSources(t *testing.T) {
	db := testDB(t)

	// Create mix of official and non-official sources
	if err := db.CreateSource(&models.Source{ID: "official/repo1", IsOfficial: true, Priority: 10}); err != nil {
		t.Fatalf("CreateSource() error = %v", err)
	}
	if err := db.CreateSource(&models.Source{ID: "community/repo1", IsOfficial: false, Priority: 5}); err != nil {
		t.Fatalf("CreateSource() error = %v", err)
	}
	if err := db.CreateSource(&models.Source{ID: "official/repo2", IsOfficial: true, Priority: 8}); err != nil {
		t.Fatalf("CreateSource() error = %v", err)
	}

	official, err := db.GetOfficialSources()
	if err != nil {
		t.Fatalf("GetOfficialSources() error = %v", err)
	}
	if len(official) != 2 {
		t.Errorf("GetOfficialSources() returned %d sources, want 2", len(official))
	}
}

// --- Tag Tests ---

func TestTagCRUD(t *testing.T) {
	db := testDB(t)

	tag := &models.Tag{
		ID:       "python",
		Name:     "Python",
		Slug:     "python",
		Category: string(models.TagCategoryLanguage),
		Color:    models.TagColors[models.TagCategoryLanguage],
	}

	// Create
	err := db.CreateTag(tag)
	if err != nil {
		t.Fatalf("CreateTag() error = %v", err)
	}

	// Read
	retrieved, err := db.GetTag("python")
	if err != nil {
		t.Fatalf("GetTag() error = %v", err)
	}
	if retrieved == nil {
		t.Fatal("GetTag() returned nil")
	}
	if retrieved.Name != "Python" {
		t.Errorf("Name = %q, want %q", retrieved.Name, "Python")
	}

	// Read by slug
	bySlug, err := db.GetTagBySlug("python")
	if err != nil {
		t.Fatalf("GetTagBySlug() error = %v", err)
	}
	if bySlug == nil {
		t.Fatal("GetTagBySlug() returned nil")
	}
}

func TestListTags(t *testing.T) {
	db := testDB(t)

	// Create tags in different categories
	tags := []models.Tag{
		{ID: "python", Name: "Python", Slug: "python", Category: "language", Count: 100},
		{ID: "go", Name: "Go", Slug: "go", Category: "language", Count: 50},
		{ID: "react", Name: "React", Slug: "react", Category: "framework", Count: 75},
	}
	for _, tag := range tags {
		if err := db.CreateTag(&tag); err != nil {
			t.Fatalf("CreateTag() error = %v", err)
		}
	}

	// List all tags (including "mine" tag created on init)
	allTags, err := db.ListTags("")
	if err != nil {
		t.Fatalf("ListTags('') error = %v", err)
	}
	if len(allTags) != 4 {
		t.Errorf("ListTags('') returned %d tags, want 4", len(allTags))
	}

	// First tag should be "mine" due to priority, then sorted by count descending
	if allTags[0].ID != "mine" {
		t.Errorf("First tag should be 'mine' (priority), got %q", allTags[0].ID)
	}
	if allTags[1].Count != 100 {
		t.Errorf("Second tag count = %d, want 100", allTags[1].Count)
	}

	// Filter by category
	langTags, err := db.ListTags("language")
	if err != nil {
		t.Fatalf("ListTags('language') error = %v", err)
	}
	if len(langTags) != 2 {
		t.Errorf("ListTags('language') returned %d tags, want 2", len(langTags))
	}
}

func TestGetTopTags(t *testing.T) {
	db := testDB(t)

	counts := []int{10, 50, 25, 100, 5}
	for i, c := range counts {
		tag := &models.Tag{
			ID:    "tag-" + string(rune('a'+i)),
			Name:  "Tag " + string(rune('A'+i)),
			Slug:  "tag-" + string(rune('a'+i)),
			Count: c,
		}
		if err := db.CreateTag(tag); err != nil {
			t.Fatalf("CreateTag() error = %v", err)
		}
	}

	top, err := db.GetTopTags(3)
	if err != nil {
		t.Fatalf("GetTopTags() error = %v", err)
	}
	if len(top) != 3 {
		t.Fatalf("GetTopTags(3) returned %d tags, want 3", len(top))
	}
	// First tag should be "mine" due to priority
	if top[0].ID != "mine" {
		t.Errorf("First tag should be 'mine' (priority), got %q", top[0].ID)
	}
	// Second tag should have highest count (100)
	if top[1].Count != 100 {
		t.Errorf("Second tag count = %d, want 100", top[1].Count)
	}
}

// --- Installed Tests ---

func TestInstalled(t *testing.T) {
	db := testDB(t)

	// Create a skill first
	skill := &models.Skill{
		ID:    "fav-test-001",
		Slug:  "fav-skill",
		Title: "Installed Test Skill",
	}
	if err := db.CreateSkill(skill); err != nil {
		t.Fatalf("CreateSkill() error = %v", err)
	}

	// Add to installed
	err := db.AddInstalled("fav-test-001")
	if err != nil {
		t.Fatalf("AddInstalled() error = %v", err)
	}

	// Check is installed
	isInst, err := db.IsInstalled("fav-test-001")
	if err != nil {
		t.Fatalf("IsInstalled() error = %v", err)
	}
	if !isInst {
		t.Error("IsInstalled() = false, want true")
	}

	// Get installed
	installed, err := db.GetInstalled()
	if err != nil {
		t.Fatalf("GetInstalled() error = %v", err)
	}
	if len(installed) != 1 {
		t.Errorf("GetInstalled() returned %d, want 1", len(installed))
	}

	// Count installed
	count, err := db.CountInstalled()
	if err != nil {
		t.Fatalf("CountInstalled() error = %v", err)
	}
	if count != 1 {
		t.Errorf("CountInstalled() = %d, want 1", count)
	}

	// Update notes
	err = db.UpdateInstalledNotes("fav-test-001", "My installed skill!")
	if err != nil {
		t.Fatalf("UpdateInstalledNotes() error = %v", err)
	}

	instWithNotes, err := db.GetInstalledWithNotes()
	if err != nil {
		t.Fatalf("GetInstalledWithNotes() error = %v", err)
	}
	if len(instWithNotes) != 1 {
		t.Fatalf("GetInstalledWithNotes() returned %d, want 1", len(instWithNotes))
	}
	if instWithNotes[0].Notes != "My installed skill!" {
		t.Errorf("Notes = %q, want %q", instWithNotes[0].Notes, "My installed skill!")
	}

	// Remove from installed
	err = db.RemoveInstalled("fav-test-001")
	if err != nil {
		t.Fatalf("RemoveInstalled() error = %v", err)
	}

	isInst, _ = db.IsInstalled("fav-test-001")
	if isInst {
		t.Error("IsInstalled() after remove = true, want false")
	}
}

func TestSetInstalled(t *testing.T) {
	db := testDB(t)

	skill := &models.Skill{
		ID:    "set-inst-test",
		Slug:  "set-inst-skill",
		Title: "Set Installed Test",
	}
	if err := db.CreateSkill(skill); err != nil {
		t.Fatalf("CreateSkill() error = %v", err)
	}

	// Set installed = true
	if err := db.SetInstalled("set-inst-test", true); err != nil {
		t.Fatalf("SetInstalled(true) error = %v", err)
	}

	isInst, _ := db.IsInstalled("set-inst-test")
	if !isInst {
		t.Error("SetInstalled(true) didn't work")
	}

	// Set installed = false
	if err := db.SetInstalled("set-inst-test", false); err != nil {
		t.Fatalf("SetInstalled(false) error = %v", err)
	}

	isInst, _ = db.IsInstalled("set-inst-test")
	if isInst {
		t.Error("SetInstalled(false) didn't work")
	}
}

// --- SyncMeta Tests ---

func TestSyncMeta(t *testing.T) {
	db := testDB(t)

	// Get default values (seeded on init)
	version, err := db.GetSyncMeta(models.SyncMetaSchemaVersion)
	if err != nil {
		t.Fatalf("GetSyncMeta() error = %v", err)
	}
	if version != "1" {
		t.Errorf("Schema version = %q, want %q", version, "1")
	}

	// Set new value
	now := time.Now().Format(time.RFC3339)
	err = db.SetSyncMeta(models.SyncMetaLastFullSync, now)
	if err != nil {
		t.Fatalf("SetSyncMeta() error = %v", err)
	}

	retrieved, err := db.GetSyncMeta(models.SyncMetaLastFullSync)
	if err != nil {
		t.Fatalf("GetSyncMeta() error = %v", err)
	}
	if retrieved != now {
		t.Errorf("LastFullSync = %q, want %q", retrieved, now)
	}

	// Get all
	all, err := db.GetAllSyncMeta()
	if err != nil {
		t.Fatalf("GetAllSyncMeta() error = %v", err)
	}
	if len(all) < 4 {
		t.Errorf("GetAllSyncMeta() returned %d keys, want >= 4", len(all))
	}

	// Delete
	err = db.DeleteSyncMeta(models.SyncMetaLastFullSync)
	if err != nil {
		t.Fatalf("DeleteSyncMeta() error = %v", err)
	}

	deleted, _ := db.GetSyncMeta(models.SyncMetaLastFullSync)
	if deleted != "" {
		t.Errorf("GetSyncMeta() after delete = %q, want empty", deleted)
	}
}

// --- FTS5 Search Tests ---

func TestFTS5Search(t *testing.T) {
	db := testDB(t)

	// Create searchable skills
	skills := []models.Skill{
		{ID: "fts-1", Slug: "react-hooks", Title: "React Hooks Guide", Description: "Learn React hooks", Content: "useState useEffect custom hooks"},
		{ID: "fts-2", Slug: "go-testing", Title: "Go Testing Patterns", Description: "Testing in Go", Content: "unit tests integration tests benchmarks"},
		{ID: "fts-3", Slug: "python-ml", Title: "Python Machine Learning", Description: "ML with Python", Content: "tensorflow pytorch scikit-learn"},
		{ID: "fts-4", Slug: "docker-basics", Title: "Docker Fundamentals", Description: "Container basics", Content: "dockerfile compose kubernetes"},
		{ID: "fts-5", Slug: "react-testing", Title: "React Testing Library", Description: "Testing React apps", Content: "render screen fireEvent userEvent"},
	}

	for i := range skills {
		if err := db.CreateSkill(&skills[i]); err != nil {
			t.Fatalf("CreateSkill() error = %v", err)
		}
	}

	// Search for "react" - should match skills 1 and 5
	results, err := db.Search("react", 10)
	if err != nil {
		t.Fatalf("Search('react') error = %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Search('react') returned %d results, want 2", len(results))
	}

	// Search for "testing" - should match skills 2 and 5
	results, err = db.Search("testing", 10)
	if err != nil {
		t.Fatalf("Search('testing') error = %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Search('testing') returned %d results, want 2", len(results))
	}

	// Search for "python" - should match skill 3
	results, err = db.Search("python", 10)
	if err != nil {
		t.Fatalf("Search('python') error = %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Search('python') returned %d results, want 1", len(results))
	}

	// Empty query
	results, err = db.Search("", 10)
	if err != nil {
		t.Fatalf("Search('') error = %v", err)
	}
	if results != nil {
		t.Error("Search('') should return nil")
	}

	// No matches
	results, err = db.Search("nonexistent", 10)
	if err != nil {
		t.Fatalf("Search('nonexistent') error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Search('nonexistent') returned %d results, want 0", len(results))
	}
}

func TestSearchSkills(t *testing.T) {
	db := testDB(t)

	if err := db.CreateSkill(&models.Skill{ID: "ss-1", Slug: "kubernetes-guide", Title: "Kubernetes Guide", Description: "K8s deployment"}); err != nil {
		t.Fatalf("CreateSkill() error = %v", err)
	}
	if err := db.CreateSkill(&models.Skill{ID: "ss-2", Slug: "kubernetes-advanced", Title: "Advanced Kubernetes", Description: "K8s networking"}); err != nil {
		t.Fatalf("CreateSkill() error = %v", err)
	}

	skills, err := db.SearchSkills("kubernetes", 10)
	if err != nil {
		t.Fatalf("SearchSkills() error = %v", err)
	}
	if len(skills) != 2 {
		t.Errorf("SearchSkills() returned %d results, want 2", len(skills))
	}
}

func TestSearchByCategory(t *testing.T) {
	db := testDB(t)

	if err := db.CreateSkill(&models.Skill{ID: "sbc-1", Slug: "go-web", Title: "Go Web Framework", Category: "backend", Content: "web server routing"}); err != nil {
		t.Fatalf("CreateSkill() error = %v", err)
	}
	if err := db.CreateSkill(&models.Skill{ID: "sbc-2", Slug: "react-web", Title: "React Web App", Category: "frontend", Content: "web components"}); err != nil {
		t.Fatalf("CreateSkill() error = %v", err)
	}

	// Both match "web", but filter by category
	results, err := db.SearchByCategory("web", "backend", 10)
	if err != nil {
		t.Fatalf("SearchByCategory() error = %v", err)
	}
	if len(results) != 1 {
		t.Errorf("SearchByCategory('web', 'backend') returned %d results, want 1", len(results))
	}
	if len(results) > 0 && results[0].Category != "backend" {
		t.Error("SearchByCategory returned wrong category")
	}
}

// --- Skills with Tags Tests ---

func TestUpsertSkillWithTags(t *testing.T) {
	db := testDB(t)

	tags := []models.Tag{
		{ID: "python", Name: "Python", Slug: "python", Category: "language"},
		{ID: "testing", Name: "Testing", Slug: "testing", Category: "concept"},
	}

	skill := &models.Skill{
		ID:    "skill-with-tags",
		Slug:  "skill-with-tags",
		Title: "Skill with Tags",
	}

	err := db.UpsertSkillWithTags(skill, tags)
	if err != nil {
		t.Fatalf("UpsertSkillWithTags() error = %v", err)
	}

	// Retrieve skill with tags
	retrieved, err := db.GetSkill("skill-with-tags")
	if err != nil {
		t.Fatalf("GetSkill() error = %v", err)
	}
	if len(retrieved.Tags) != 2 {
		t.Errorf("Skill has %d tags, want 2", len(retrieved.Tags))
	}

	// Get tags for skill
	skillTags, err := db.GetTagsForSkill("skill-with-tags")
	if err != nil {
		t.Fatalf("GetTagsForSkill() error = %v", err)
	}
	if len(skillTags) != 2 {
		t.Errorf("GetTagsForSkill() returned %d tags, want 2", len(skillTags))
	}
}

func TestAddRemoveTagFromSkill(t *testing.T) {
	db := testDB(t)

	// Create skill and tag
	if err := db.CreateSkill(&models.Skill{ID: "tag-assoc-skill", Slug: "tag-assoc", Title: "Tag Association Test"}); err != nil {
		t.Fatalf("CreateSkill() error = %v", err)
	}
	if err := db.CreateTag(&models.Tag{ID: "assoc-tag", Name: "Association Tag", Slug: "assoc-tag"}); err != nil {
		t.Fatalf("CreateTag() error = %v", err)
	}

	// Add tag to skill
	err := db.AddTagToSkill("tag-assoc-skill", "assoc-tag")
	if err != nil {
		t.Fatalf("AddTagToSkill() error = %v", err)
	}

	tags, _ := db.GetTagsForSkill("tag-assoc-skill")
	if len(tags) != 1 {
		t.Errorf("After AddTagToSkill, got %d tags, want 1", len(tags))
	}

	// Remove tag from skill
	err = db.RemoveTagFromSkill("tag-assoc-skill", "assoc-tag")
	if err != nil {
		t.Fatalf("RemoveTagFromSkill() error = %v", err)
	}

	tags, _ = db.GetTagsForSkill("tag-assoc-skill")
	if len(tags) != 0 {
		t.Errorf("After RemoveTagFromSkill, got %d tags, want 0", len(tags))
	}
}

// --- Stats Tests ---

func TestGetStats(t *testing.T) {
	db := testDB(t)

	// Create some data
	if err := db.CreateSkill(&models.Skill{ID: "stats-skill-1", Slug: "stats1", Title: "Stats Skill 1"}); err != nil {
		t.Fatalf("CreateSkill() error = %v", err)
	}
	if err := db.CreateSkill(&models.Skill{ID: "stats-skill-2", Slug: "stats2", Title: "Stats Skill 2"}); err != nil {
		t.Fatalf("CreateSkill() error = %v", err)
	}
	if err := db.CreateTag(&models.Tag{ID: "stats-tag-1", Name: "Stats Tag 1", Slug: "stats-tag-1"}); err != nil {
		t.Fatalf("CreateTag() error = %v", err)
	}
	if err := db.CreateSource(&models.Source{ID: "stats/source", Owner: "stats", Repo: "source"}); err != nil {
		t.Fatalf("CreateSource() error = %v", err)
	}

	stats, err := db.GetStats()
	if err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}

	if stats.TotalSkills != 2 {
		t.Errorf("TotalSkills = %d, want 2", stats.TotalSkills)
	}
	// mine tag + stats-tag-1 = 2 tags
	if stats.TotalTags != 2 {
		t.Errorf("TotalTags = %d, want 2", stats.TotalTags)
	}
	if stats.TotalSources != 1 {
		t.Errorf("TotalSources = %d, want 1", stats.TotalSources)
	}
	if stats.CacheSizeBytes <= 0 {
		t.Error("CacheSizeBytes should be > 0")
	}
}

// --- Skill with Source Tests ---

func TestSkillWithSource(t *testing.T) {
	db := testDB(t)

	// Create source first
	source := &models.Source{
		ID:    "source-owner/source-repo",
		Owner: "source-owner",
		Repo:  "source-repo",
	}
	if err := db.CreateSource(source); err != nil {
		t.Fatalf("CreateSource() error = %v", err)
	}

	// Create skill with source reference
	sourceID := "source-owner/source-repo"
	skill := &models.Skill{
		ID:       "skill-with-source",
		Slug:     "skill-with-source",
		Title:    "Skill with Source",
		SourceID: &sourceID,
	}
	if err := db.CreateSkill(skill); err != nil {
		t.Fatalf("CreateSkill() error = %v", err)
	}

	// Retrieve skill with source preloaded
	retrieved, err := db.GetSkill("skill-with-source")
	if err != nil {
		t.Fatalf("GetSkill() error = %v", err)
	}
	if retrieved.Source == nil {
		t.Fatal("Skill.Source should not be nil")
	}
	if retrieved.Source.Owner != "source-owner" {
		t.Errorf("Source.Owner = %q, want %q", retrieved.Source.Owner, "source-owner")
	}
}

func TestUpdateSourceSkillCount(t *testing.T) {
	db := testDB(t)

	// Create source
	source := &models.Source{
		ID:    "count-test/repo",
		Owner: "count-test",
		Repo:  "repo",
	}
	if err := db.CreateSource(source); err != nil {
		t.Fatalf("CreateSource() error = %v", err)
	}

	// Create skills linked to source
	sourceID := "count-test/repo"
	for i := 0; i < 3; i++ {
		skill := &models.Skill{
			ID:       "count-skill-" + string(rune('a'+i)),
			Slug:     "count-skill-" + string(rune('a'+i)),
			Title:    "Count Skill",
			SourceID: &sourceID,
		}
		if err := db.CreateSkill(skill); err != nil {
			t.Fatalf("CreateSkill() error = %v", err)
		}
	}

	// Update count
	err := db.UpdateSourceSkillCount("count-test/repo")
	if err != nil {
		t.Fatalf("UpdateSourceSkillCount() error = %v", err)
	}

	// Verify count
	updated, _ := db.GetSource("count-test/repo")
	if updated.SkillCount != 3 {
		t.Errorf("SkillCount = %d, want 3", updated.SkillCount)
	}
}

// --- FTS Query Preparation Tests ---

func TestPrepareFTSQuery(t *testing.T) {
	// Note: The function removes special chars and adds * suffix to each term.
	// Dash is replaced with space AFTER split, so "go-lang" becomes "go lang*" (one term with space).
	// Other special chars like () : are simply removed, not replaced with space.
	tests := []struct {
		input    string
		expected string
	}{
		{"react", "react*"},
		{"react hooks", "react* hooks*"},
		{"go-lang", "go lang*"}, // dash replaced with space, but it's one term
		{"", ""},
		{"test(query)", "testquery*"},             // parens removed, stays one term
		{"hello:world", "helloworld*"},            // colon removed, stays one term
		{"react AND hooks", "react* AND* hooks*"}, // AND is treated as a term
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := prepareFTSQuery(tt.input)
			if result != tt.expected {
				t.Errorf("prepareFTSQuery(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// --- Hard Delete Test ---

func TestHardDeleteSkill(t *testing.T) {
	db := testDB(t)

	skill := &models.Skill{
		ID:    "hard-delete-test",
		Slug:  "hard-delete",
		Title: "Hard Delete Test",
	}
	if err := db.CreateSkill(skill); err != nil {
		t.Fatalf("CreateSkill() error = %v", err)
	}

	// Soft delete first
	if err := db.DeleteSkill("hard-delete-test"); err != nil {
		t.Fatalf("DeleteSkill() error = %v", err)
	}

	// Verify soft deleted (shouldn't be found with normal query)
	softDeleted, _ := db.GetSkill("hard-delete-test")
	if softDeleted != nil {
		t.Error("Soft deleted skill should not be found")
	}

	// Hard delete
	err := db.HardDeleteSkill("hard-delete-test")
	if err != nil {
		t.Fatalf("HardDeleteSkill() error = %v", err)
	}

	// Verify completely gone (even with unscoped query)
	var count int64
	db.Unscoped().Model(&models.Skill{}).Where("id = ?", "hard-delete-test").Count(&count)
	if count != 0 {
		t.Error("Hard deleted skill should be completely removed")
	}
}

// --- Tag Priority Tests ---

func TestTagPriorityOrdering(t *testing.T) {
	db := testDB(t)

	// Create tags with different priorities (mine tag already exists with priority 100)
	tags := []models.Tag{
		{ID: "python", Name: "Python", Slug: "python", Category: "language", Count: 100, Priority: 0},
		{ID: "go", Name: "Go", Slug: "go", Category: "language", Count: 50, Priority: 0},
	}
	for i := range tags {
		if err := db.CreateTag(&tags[i]); err != nil {
			t.Fatalf("CreateTag() error = %v", err)
		}
	}

	// Get top tags - "mine" should be first despite lower count
	result, err := db.GetTopTags(10)
	if err != nil {
		t.Fatalf("GetTopTags() error = %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("GetTopTags() returned %d tags, want 3", len(result))
	}
	if result[0].ID != "mine" {
		t.Errorf("First tag should be 'mine' due to priority, got %q", result[0].ID)
	}
	if result[1].ID != "python" {
		t.Errorf("Second tag should be 'python' (highest count), got %q", result[1].ID)
	}
	if result[2].ID != "go" {
		t.Errorf("Third tag should be 'go', got %q", result[2].ID)
	}
}

func TestEnsureMineTag(t *testing.T) {
	db := testDB(t)

	// First call is done in New(), verify it exists
	tag, err := db.GetTag("mine")
	if err != nil {
		t.Fatalf("GetTag() error = %v", err)
	}
	if tag == nil {
		t.Fatal("mine tag should exist after DB init")
	}
	if tag.Priority != 100 {
		t.Errorf("mine tag priority = %d, want 100", tag.Priority)
	}
	if tag.Color != "#DC143C" {
		t.Errorf("mine tag color = %q, want #DC143C", tag.Color)
	}

	// Second call should be idempotent
	err = db.EnsureMineTag()
	if err != nil {
		t.Fatalf("EnsureMineTag() error = %v", err)
	}

	// Verify still exists with same values
	tag, _ = db.GetTag("mine")
	if tag.Priority != 100 {
		t.Errorf("mine tag priority after re-ensure = %d, want 100", tag.Priority)
	}
}

func TestMineTagCreatedOnStartup(t *testing.T) {
	db := testDB(t) // This calls New() which should create mine tag

	tag, err := db.GetTag("mine")
	if err != nil {
		t.Fatalf("GetTag() error = %v", err)
	}
	if tag == nil {
		t.Fatal("mine tag should be created on DB init")
	}
	if tag.Priority != 100 {
		t.Errorf("mine tag priority = %d, want 100", tag.Priority)
	}
	if tag.Category != "mine" {
		t.Errorf("mine tag category = %q, want 'mine'", tag.Category)
	}
}

// --- Transaction Tests ---

func TestTransaction_Commit(t *testing.T) {
	db := testDB(t)

	// Create a skill within a transaction
	err := db.Transaction(func(tx *DB) error {
		skill := &models.Skill{
			ID:          "tx-test-skill",
			Slug:        "tx-test",
			Title:       "Transaction Test Skill",
			Description: "Created in transaction",
			Content:     "# Test",
		}
		return tx.CreateSkill(skill)
	})
	if err != nil {
		t.Fatalf("Transaction() error = %v", err)
	}

	// Verify skill was committed
	skill, err := db.GetSkill("tx-test-skill")
	if err != nil {
		t.Fatalf("GetSkill() error = %v", err)
	}
	if skill == nil {
		t.Error("Skill should exist after transaction commit")
	}
}

func TestTransaction_Rollback(t *testing.T) {
	db := testDB(t)

	// Create a skill in a transaction that will fail
	err := db.Transaction(func(tx *DB) error {
		skill := &models.Skill{
			ID:          "tx-rollback-skill",
			Slug:        "tx-rollback",
			Title:       "Rollback Test Skill",
			Description: "Should be rolled back",
			Content:     "# Test",
		}
		if err := tx.CreateSkill(skill); err != nil {
			return err
		}
		// Return an error to trigger rollback
		return os.ErrInvalid
	})

	// Transaction should have returned the error
	if err != os.ErrInvalid {
		t.Errorf("Expected os.ErrInvalid, got %v", err)
	}

	// Verify skill was NOT committed (rolled back)
	skill, err := db.GetSkill("tx-rollback-skill")
	if err != nil {
		t.Fatalf("GetSkill() error = %v", err)
	}
	if skill != nil {
		t.Error("Skill should NOT exist after transaction rollback")
	}
}

func TestTransaction_MultipleOperations(t *testing.T) {
	db := testDB(t)

	// Create multiple skills in a single transaction
	err := db.Transaction(func(tx *DB) error {
		for i := 1; i <= 3; i++ {
			skill := &models.Skill{
				ID:          "tx-multi-" + string(rune('0'+i)),
				Slug:        "tx-multi-" + string(rune('0'+i)),
				Title:       "Multi Test Skill",
				Description: "Batch created",
				Content:     "# Test",
			}
			if err := tx.CreateSkill(skill); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Transaction() error = %v", err)
	}

	// Verify all skills were committed
	for i := 1; i <= 3; i++ {
		skill, err := db.GetSkill("tx-multi-" + string(rune('0'+i)))
		if err != nil {
			t.Fatalf("GetSkill() error = %v", err)
		}
		if skill == nil {
			t.Errorf("Skill tx-multi-%d should exist after transaction commit", i)
		}
	}
}

// --- DiscoveredSkills Table Tests ---

func TestDB_DiscoveredSkillsTableExists(t *testing.T) {
	db := testDB(t)

	// Check table exists
	var count int64
	err := db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='discovered_skills'").Scan(&count).Error

	if err != nil {
		t.Fatalf("Query error: %v", err)
	}
	if count != 1 {
		t.Errorf("discovered_skills table should exist, got count = %d", count)
	}
}
