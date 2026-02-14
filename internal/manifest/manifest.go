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
	Skills  map[string]string `json:"skills"` // slug -> "owner/repo"
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

// SkillCount returns the number of skills in the manifest.
func (mf *ManifestFile) SkillCount() int {
	return len(mf.Skills)
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
