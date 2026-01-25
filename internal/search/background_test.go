package search

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBackgroundIndexer(t *testing.T) {
	store := &mockVectorStore{}
	cfg := DefaultIndexerConfig()

	bi := NewBackgroundIndexer(nil, store, cfg)

	require.NotNil(t, bi)
	assert.NotNil(t, bi.indexer)
	assert.Equal(t, store, bi.store)
	assert.False(t, bi.running)
}

func TestBackgroundIndexer_IsRunning(t *testing.T) {
	store := &mockVectorStore{}
	bi := NewBackgroundIndexer(nil, store, DefaultIndexerConfig())

	assert.False(t, bi.IsRunning())
}

func TestBackgroundIndexer_VectorStore(t *testing.T) {
	store := &mockVectorStore{}
	bi := NewBackgroundIndexer(nil, store, DefaultIndexerConfig())

	assert.Equal(t, store, bi.VectorStore())
}

func TestBackgroundIndexer_Start_NonBlocking(t *testing.T) {
	store := &mockVectorStore{}
	bi := NewBackgroundIndexer(nil, store, DefaultIndexerConfig())

	// Create a mock that returns 0 pending (simulating no DB)
	// This test verifies Start returns immediately

	ctx := context.Background()
	progressCh := make(chan IndexProgress, 10)

	start := time.Now()
	err := bi.Start(ctx, progressCh)
	elapsed := time.Since(start)

	require.NoError(t, err)
	// Should return immediately (within 50ms)
	assert.Less(t, elapsed, 50*time.Millisecond)
}

func TestBackgroundIndexer_Start_AlreadyRunning(t *testing.T) {
	store := &mockVectorStore{}
	bi := NewBackgroundIndexer(nil, store, DefaultIndexerConfig())

	ctx := context.Background()
	progressCh := make(chan IndexProgress, 10)

	// Start first time
	err := bi.Start(ctx, progressCh)
	require.NoError(t, err)

	// Try to start again - should be a no-op
	err = bi.Start(ctx, progressCh)
	require.NoError(t, err)
}

func TestBackgroundIndexer_Stop(t *testing.T) {
	store := &mockVectorStore{}
	bi := NewBackgroundIndexer(nil, store, DefaultIndexerConfig())

	ctx := context.Background()
	progressCh := make(chan IndexProgress, 10)

	_ = bi.Start(ctx, progressCh)
	bi.Stop()

	// Wait a bit for the goroutine to finish
	time.Sleep(50 * time.Millisecond)
}

func TestBackgroundIndexer_Close(t *testing.T) {
	store := &mockVectorStore{}
	bi := NewBackgroundIndexer(nil, store, DefaultIndexerConfig())

	err := bi.Close()
	assert.NoError(t, err)
}

func TestIndexProgress_Struct(t *testing.T) {
	progress := IndexProgress{
		Running:   true,
		Total:     100,
		Completed: 50,
		Failed:    5,
		Message:   "Indexing...",
	}

	assert.True(t, progress.Running)
	assert.Equal(t, 100, progress.Total)
	assert.Equal(t, 50, progress.Completed)
	assert.Equal(t, 5, progress.Failed)
	assert.Equal(t, "Indexing...", progress.Message)
}
