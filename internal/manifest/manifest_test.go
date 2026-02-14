package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRead_FileExists(t *testing.T) {
	dir := t.TempDir()

	// Write a valid manifest
	original := &ManifestFile{
		Version: 1,
		Skills: map[string]string{
			"teach":     "asteroid-belt/skills",
			"superplan": "asteroid-belt/skills",
		},
	}
	if err := Write(dir, original); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Read it back
	got, err := Read(dir)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got == nil {
		t.Fatal("Read returned nil")
	}
	if got.Version != 1 {
		t.Errorf("Version = %d, want 1", got.Version)
	}
	if len(got.Skills) != 2 {
		t.Errorf("Skills count = %d, want 2", len(got.Skills))
	}
	if got.Skills["teach"] != "asteroid-belt/skills" {
		t.Errorf("Skills[teach] = %q, want %q", got.Skills["teach"], "asteroid-belt/skills")
	}
}

func TestRead_FileNotExists(t *testing.T) {
	dir := t.TempDir()

	got, err := Read(dir)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got != nil {
		t.Fatalf("Read returned non-nil for missing file: %+v", got)
	}
}

func TestRead_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, FileName)

	if err := os.WriteFile(p, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Read(dir)
	if err == nil {
		t.Fatal("Read should return error for invalid JSON")
	}
}

func TestRead_EmptySkills(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, FileName)

	if err := os.WriteFile(p, []byte(`{"version": 1}`), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := Read(dir)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.Skills == nil {
		t.Fatal("Skills should be initialized to empty map, not nil")
	}
}

func TestWrite_CreatesFile(t *testing.T) {
	dir := t.TempDir()

	mf := &ManifestFile{
		Version: 1,
		Skills: map[string]string{
			"docker-expert": "vercel/skills",
		},
	}
	if err := Write(dir, mf); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Verify file exists
	p := Path(dir)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		t.Fatalf("File not created at %s", p)
	}

	// Read raw contents to verify JSON structure
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if content[len(content)-1] != '\n' {
		t.Error("File should end with newline")
	}
}

func TestWrite_Idempotent(t *testing.T) {
	dir := t.TempDir()

	mf := &ManifestFile{
		Version: 1,
		Skills: map[string]string{
			"b-skill": "owner/repo-b",
			"a-skill": "owner/repo-a",
		},
	}

	// Write twice
	if err := Write(dir, mf); err != nil {
		t.Fatalf("Write 1: %v", err)
	}
	data1, _ := os.ReadFile(Path(dir))

	if err := Write(dir, mf); err != nil {
		t.Fatalf("Write 2: %v", err)
	}
	data2, _ := os.ReadFile(Path(dir))

	if string(data1) != string(data2) {
		t.Error("Two writes produced different output (not idempotent)")
	}
}

func TestWrite_SetsDefaultVersion(t *testing.T) {
	dir := t.TempDir()

	mf := &ManifestFile{
		Skills: map[string]string{"test": "owner/repo"},
	}
	if err := Write(dir, mf); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, _ := Read(dir)
	if got.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d", got.Version, CurrentVersion)
	}
}

func TestPath(t *testing.T) {
	got := Path("/home/user/project")
	want := filepath.Join("/home/user/project", FileName)
	if got != want {
		t.Errorf("Path = %q, want %q", got, want)
	}
}

func TestNew(t *testing.T) {
	mf := New()
	if mf.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d", mf.Version, CurrentVersion)
	}
	if mf.Skills == nil {
		t.Error("Skills should not be nil")
	}
	if len(mf.Skills) != 0 {
		t.Error("Skills should be empty")
	}
}

func TestSortedSlugs(t *testing.T) {
	mf := &ManifestFile{
		Skills: map[string]string{
			"charlie": "o/r",
			"alpha":   "o/r",
			"bravo":   "o/r",
		},
	}
	got := mf.SortedSlugs()
	want := []string{"alpha", "bravo", "charlie"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("SortedSlugs[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
