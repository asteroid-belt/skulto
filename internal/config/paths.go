package config

import (
	"os"
	"path/filepath"
)

// Paths contains commonly used file paths.
type Paths struct {
	Database     string // Main SQLite database
	Embeddings   string // Embeddings storage
	Config       string // Config file
	Repositories string // Cloned repositories directory
	Skills       string // Local skills directory
}

// GetPaths returns all commonly used paths based on config.
func GetPaths(cfg *Config) Paths {
	return Paths{
		Database:     filepath.Join(cfg.BaseDir, "skulto.db"),
		Embeddings:   filepath.Join(cfg.BaseDir, "embeddings.db"),
		Config:       filepath.Join(cfg.BaseDir, "config.yaml"),
		Repositories: filepath.Join(cfg.BaseDir, "repositories"),
		Skills:       filepath.Join(cfg.BaseDir, "skills"),
	}
}

// DefaultBaseDir returns the default base directory (~/.skulto).
func DefaultBaseDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".skulto"
	}
	return filepath.Join(home, ".skulto")
}
