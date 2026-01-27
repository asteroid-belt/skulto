package favorites

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_AddAndRemove(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "favorites.json")

	store := NewStore(path)
	require.NoError(t, store.Load())

	// Initially empty
	assert.Equal(t, 0, store.Count())

	// Add a favorite
	err := store.Add("docker-expert")
	require.NoError(t, err)
	assert.Equal(t, 1, store.Count())
	assert.True(t, store.IsFavorite("docker-expert"))

	// Add another
	err = store.Add("react-best-practices")
	require.NoError(t, err)
	assert.Equal(t, 2, store.Count())

	// Adding duplicate should be idempotent (no error, no duplicate)
	err = store.Add("docker-expert")
	require.NoError(t, err)
	assert.Equal(t, 2, store.Count())

	// Remove one
	err = store.Remove("docker-expert")
	require.NoError(t, err)
	assert.Equal(t, 1, store.Count())
	assert.False(t, store.IsFavorite("docker-expert"))
	assert.True(t, store.IsFavorite("react-best-practices"))

	// Removing non-existent should be idempotent (no error)
	err = store.Remove("nonexistent")
	require.NoError(t, err)
	assert.Equal(t, 1, store.Count())
}

func TestStore_IsFavorite(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "favorites.json")

	store := NewStore(path)
	require.NoError(t, store.Load())

	// Not a favorite initially
	assert.False(t, store.IsFavorite("docker-expert"))

	// Add and check
	require.NoError(t, store.Add("docker-expert"))
	assert.True(t, store.IsFavorite("docker-expert"))

	// Remove and check
	require.NoError(t, store.Remove("docker-expert"))
	assert.False(t, store.IsFavorite("docker-expert"))
}

func TestStore_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "favorites.json")

	// Create store, add favorites, save
	store1 := NewStore(path)
	require.NoError(t, store1.Load())
	require.NoError(t, store1.Add("docker-expert"))
	require.NoError(t, store1.Add("react-best-practices"))

	// Verify file exists
	_, err := os.Stat(path)
	require.NoError(t, err, "favorites.json should exist")

	// Create new store instance, load from file
	store2 := NewStore(path)
	require.NoError(t, store2.Load())

	// Should have same favorites
	assert.Equal(t, 2, store2.Count())
	assert.True(t, store2.IsFavorite("docker-expert"))
	assert.True(t, store2.IsFavorite("react-best-practices"))

	// Verify List() returns favorites with timestamps
	favorites := store2.List()
	assert.Len(t, favorites, 2)
	for _, fav := range favorites {
		assert.NotEmpty(t, fav.Slug)
		assert.False(t, fav.AddedAt.IsZero())
	}
}

func TestStore_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "favorites.json")

	// Load from non-existent file should initialize empty
	store := NewStore(path)
	require.NoError(t, store.Load())
	assert.Equal(t, 0, store.Count())

	// Should be able to add favorites
	require.NoError(t, store.Add("test-skill"))
	assert.Equal(t, 1, store.Count())
}

func TestStore_Concurrent(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "favorites.json")

	store := NewStore(path)
	require.NoError(t, store.Load())

	var wg sync.WaitGroup
	slugs := []string{"skill-1", "skill-2", "skill-3", "skill-4", "skill-5"}

	// Concurrent adds
	for _, slug := range slugs {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			err := store.Add(s)
			assert.NoError(t, err)
		}(slug)
	}
	wg.Wait()

	assert.Equal(t, 5, store.Count())

	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = store.IsFavorite("skill-1")
			_ = store.List()
			_ = store.Count()
		}()
	}
	wg.Wait()

	// Concurrent removes
	for _, slug := range slugs {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			err := store.Remove(s)
			assert.NoError(t, err)
		}(slug)
	}
	wg.Wait()

	assert.Equal(t, 0, store.Count())
}

func TestStore_List_OrderByAddedAt(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "favorites.json")

	store := NewStore(path)
	require.NoError(t, store.Load())

	// Add with small delays to ensure different timestamps
	require.NoError(t, store.Add("first"))
	time.Sleep(10 * time.Millisecond)
	require.NoError(t, store.Add("second"))
	time.Sleep(10 * time.Millisecond)
	require.NoError(t, store.Add("third"))

	favorites := store.List()
	require.Len(t, favorites, 3)

	// Should be in order added (first added = first in list)
	assert.Equal(t, "first", favorites[0].Slug)
	assert.Equal(t, "second", favorites[1].Slug)
	assert.Equal(t, "third", favorites[2].Slug)
}
