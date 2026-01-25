package models

// OptionalDirName defines allowed optional directory names per Agent Skills spec.
// See: https://agentskills.io/specification#optional-directories
type OptionalDirName string

const (
	OptionalDirScripts    OptionalDirName = "scripts"
	OptionalDirReferences OptionalDirName = "references"
	OptionalDirAssets     OptionalDirName = "assets"
)

// AllOptionalDirNames returns all valid optional directory names.
func AllOptionalDirNames() []OptionalDirName {
	return []OptionalDirName{
		OptionalDirScripts,
		OptionalDirReferences,
		OptionalDirAssets,
	}
}

// IsValidOptionalDirName checks if a name is a valid optional directory.
func IsValidOptionalDirName(name string) bool {
	for _, valid := range AllOptionalDirNames() {
		if string(valid) == name {
			return true
		}
	}
	return false
}

// OptionalDir represents an optional directory accompanying a SKILL.md.
type OptionalDir struct {
	// Name is the directory name (scripts, references, or assets).
	Name OptionalDirName

	// Files contains all files in the directory (recursively).
	Files []OptionalFile
}

// OptionalFile represents a file within an optional directory.
type OptionalFile struct {
	// Name is the file name (for direct children).
	Name string

	// Path is the relative path within the directory (e.g., "utils/helper.py").
	Path string

	// Content is the raw file content.
	Content []byte

	// Size is the file size in bytes.
	Size int64
}

// MaxOptionalFileSize is the maximum file size to sync (1MB).
// Files larger than this are skipped to avoid repository bloat.
const MaxOptionalFileSize int64 = 1 << 20 // 1MB

// TotalSize returns the combined size of all files in the directory.
func (d OptionalDir) TotalSize() int64 {
	var total int64
	for _, f := range d.Files {
		total += f.Size
	}
	return total
}

// FileCount returns the number of files in the directory.
func (d OptionalDir) FileCount() int {
	return len(d.Files)
}
