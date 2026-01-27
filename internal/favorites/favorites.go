// Package favorites provides persistent storage for user's favorite skills.
// Favorites are stored in a JSON file that persists across database resets.
package favorites

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Favorite represents a favorited skill.
type Favorite struct {
	Slug    string    `json:"slug"`
	AddedAt time.Time `json:"added_at"`
}

// fileData represents the JSON file structure.
type fileData struct {
	Version   int        `json:"version"`
	Favorites []Favorite `json:"favorites"`
}

// Store manages favorites persistence.
type Store struct {
	path  string
	mu    sync.RWMutex
	cache *fileData
}

// NewStore creates a new favorites store.
func NewStore(path string) *Store {
	return &Store{
		path: path,
		cache: &fileData{
			Version:   1,
			Favorites: []Favorite{},
		},
	}
}

// Load reads favorites from the JSON file.
// If the file doesn't exist, initializes with empty favorites.
func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, initialize empty
			s.cache = &fileData{
				Version:   1,
				Favorites: []Favorite{},
			}
			return nil
		}
		return err
	}

	var fd fileData
	if err := json.Unmarshal(data, &fd); err != nil {
		// Corrupted file, initialize empty
		s.cache = &fileData{
			Version:   1,
			Favorites: []Favorite{},
		}
		return nil
	}

	s.cache = &fd
	return nil
}

// Save writes favorites to the JSON file atomically.
func (s *Store) Save() error {
	s.mu.RLock()
	data, err := json.MarshalIndent(s.cache, "", "  ")
	s.mu.RUnlock()

	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Atomic write: write to temp file, then rename
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}

	return os.Rename(tmpPath, s.path)
}

// Add adds a skill to favorites if not already present.
func (s *Store) Add(slug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already exists
	for _, f := range s.cache.Favorites {
		if f.Slug == slug {
			return nil // Already exists, idempotent
		}
	}

	s.cache.Favorites = append(s.cache.Favorites, Favorite{
		Slug:    slug,
		AddedAt: time.Now(),
	})

	// Save immediately
	return s.saveLocked()
}

// Remove removes a skill from favorites.
func (s *Store) Remove(slug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, f := range s.cache.Favorites {
		if f.Slug == slug {
			s.cache.Favorites = append(s.cache.Favorites[:i], s.cache.Favorites[i+1:]...)
			return s.saveLocked()
		}
	}

	return nil // Not found, idempotent
}

// IsFavorite returns true if the skill is a favorite.
func (s *Store) IsFavorite(slug string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, f := range s.cache.Favorites {
		if f.Slug == slug {
			return true
		}
	}
	return false
}

// List returns all favorites in order they were added.
func (s *Store) List() []Favorite {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]Favorite, len(s.cache.Favorites))
	copy(result, s.cache.Favorites)
	return result
}

// Count returns the number of favorites.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.cache.Favorites)
}

// saveLocked saves without acquiring the lock (caller must hold write lock).
func (s *Store) saveLocked() error {
	data, err := json.MarshalIndent(s.cache, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Atomic write
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}

	return os.Rename(tmpPath, s.path)
}
