package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const (
	// FileName is the manifest file name.
	FileName = "skulto.json"

	// CurrentVersion is the current manifest format version.
	CurrentVersion = 1
)

// ManifestFile is the JSON structure of skulto.json.
type ManifestFile struct {
	Version int               `json:"version"`
	Skills  map[string]string `json:"skills"`            // slug -> "owner/repo"
	Ignored []string          `json:"ignored,omitempty"` // skills explicitly excluded from manifest
}

// Read reads a manifest from the given directory.
// Returns nil, nil if the file does not exist.
func Read(dir string) (*ManifestFile, error) {
	p := Path(dir)

	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var mf ManifestFile
	if err := json.Unmarshal(data, &mf); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	if mf.Skills == nil {
		mf.Skills = make(map[string]string)
	}
	if mf.Ignored == nil {
		mf.Ignored = []string{}
	}

	return &mf, nil
}

// Write writes a manifest to the given directory using atomic file operations.
func Write(dir string, mf *ManifestFile) error {
	if mf.Version == 0 {
		mf.Version = CurrentVersion
	}
	if mf.Skills == nil {
		mf.Skills = make(map[string]string)
	}

	data, err := json.MarshalIndent(mf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	// Append trailing newline
	data = append(data, '\n')

	p := Path(dir)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Atomic write: temp file + rename
	tmpPath := p + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := os.Rename(tmpPath, p); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename manifest: %w", err)
	}

	return nil
}

// Path returns the full path to skulto.json in the given directory.
func Path(dir string) string {
	return filepath.Join(dir, FileName)
}

// New creates a new empty manifest.
func New() *ManifestFile {
	return &ManifestFile{
		Version: CurrentVersion,
		Skills:  make(map[string]string),
	}
}

// SkillEntry represents a skill eligible for inclusion in a manifest.
type SkillEntry struct {
	Slug       string
	SourceName string // "owner/repo" format
	LocalOnly  bool   // true if no source repository
}

// BuildFromSkills creates a manifest from skill entries, skipping local-only skills.
// Returns the manifest and the count of skipped local-only skills.
func BuildFromSkills(entries []SkillEntry) (*ManifestFile, int) {
	mf := New()
	skippedLocal := 0
	for _, e := range entries {
		if e.LocalOnly {
			skippedLocal++
			continue
		}
		mf.Skills[e.Slug] = e.SourceName
	}
	return mf, skippedLocal
}

// SkillCount returns the number of skills in the manifest.
func (mf *ManifestFile) SkillCount() int {
	return len(mf.Skills)
}

// SkillsEqual returns true if two manifests have identical skills maps.
func SkillsEqual(a, b *ManifestFile) bool {
	if a == nil || b == nil {
		return a == b
	}
	if len(a.Skills) != len(b.Skills) {
		return false
	}
	for slug, source := range a.Skills {
		if b.Skills[slug] != source {
			return false
		}
	}
	return true
}

// ManifestEqual returns true if two manifests have identical skills AND ignored lists.
// Used as the write gate in save.go (replaces SkillsEqual for no-op detection).
func ManifestEqual(a, b *ManifestFile) bool {
	if !SkillsEqual(a, b) {
		return false
	}
	if a == nil || b == nil {
		return a == b
	}
	if len(a.Ignored) != len(b.Ignored) {
		return false
	}
	aSet := make(map[string]bool, len(a.Ignored))
	for _, s := range a.Ignored {
		aSet[s] = true
	}
	for _, s := range b.Ignored {
		if !aSet[s] {
			return false
		}
	}
	return true
}

// SortedSlugs returns skill slugs in alphabetical order.
func (mf *ManifestFile) SortedSlugs() []string {
	slugs := make([]string, 0, len(mf.Skills))
	for slug := range mf.Skills {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)
	return slugs
}
