package search

import (
	"context"
	"sync"

	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/log"
	"github.com/asteroid-belt/skulto/internal/vector"
)

// BackgroundIndexer handles non-blocking embedding generation on TUI launch.
// Uses the unified VectorStore interface (chromem-go or Qdrant).
type BackgroundIndexer struct {
	db      *db.DB
	store   vector.VectorStore
	indexer *Indexer
	config  IndexerConfig

	mu         sync.Mutex
	running    bool
	cancelFunc context.CancelFunc
	done       chan struct{}
}

// IndexProgress reports current indexing state for TUI display.
type IndexProgress struct {
	Running   bool
	Total     int
	Completed int
	Failed    int
	Message   string
}

// NewBackgroundIndexer creates a new background indexer.
func NewBackgroundIndexer(database *db.DB, store vector.VectorStore, cfg IndexerConfig) *BackgroundIndexer {
	return &BackgroundIndexer{
		db:      database,
		store:   store,
		indexer: NewIndexer(database, store, cfg),
		config:  cfg,
		done:    make(chan struct{}),
	}
}

// GetPendingCount returns the number of skills needing embedding.
func (bi *BackgroundIndexer) GetPendingCount() (int, error) {
	return bi.db.CountSkillsWithoutEmbedding()
}

// Start begins background indexing if there are pending skills.
// Returns immediately - indexing happens in a background goroutine.
// The progressCh receives updates during indexing; caller should not close it.
func (bi *BackgroundIndexer) Start(ctx context.Context, progressCh chan<- IndexProgress) error {
	bi.mu.Lock()
	if bi.running {
		bi.mu.Unlock()
		return nil // Already running
	}
	bi.running = true

	// Create cancellable context for cleanup
	ctx, bi.cancelFunc = context.WithCancel(ctx)
	bi.mu.Unlock()

	go bi.runIndexing(ctx, progressCh)
	return nil
}

// Stop cancels the background indexing gracefully.
func (bi *BackgroundIndexer) Stop() {
	bi.mu.Lock()
	if bi.cancelFunc != nil {
		bi.cancelFunc()
	}
	bi.mu.Unlock()
}

// Close stops the background indexer and cleans up resources.
// This prevents goroutine leaks when the application exits.
func (bi *BackgroundIndexer) Close() error {
	bi.Stop()

	// Wait for indexing to complete
	select {
	case <-bi.done:
	default:
		// Already closed or never started
	}

	if bi.store != nil {
		if err := bi.store.Close(); err != nil {
			log.Errorf("error closing vector store: %v", err)
			return err
		}
	}
	return nil
}

// runIndexing performs the actual indexing work in a goroutine.
func (bi *BackgroundIndexer) runIndexing(ctx context.Context, progressCh chan<- IndexProgress) {
	defer func() {
		bi.mu.Lock()
		bi.running = false
		bi.mu.Unlock()
		close(bi.done)
	}()

	// Guard against nil database (for testing)
	if bi.db == nil {
		safeSendProgress(progressCh, IndexProgress{
			Running: false,
			Message: "No database configured",
		})
		return
	}

	// Check for pending skills
	pending, err := bi.db.CountSkillsWithoutEmbedding()
	if err != nil || pending == 0 {
		// Nothing to do
		safeSendProgress(progressCh, IndexProgress{
			Running: false,
			Message: "No skills to index",
		})
		return
	}

	// Send initial progress
	safeSendProgress(progressCh, IndexProgress{
		Running: true,
		Total:   pending,
		Message: "Starting embedding generation...",
	})

	// Create progress channel for the underlying indexer
	indexerProgress := make(chan Progress, 10)

	// Start a goroutine to forward progress updates
	var forwardWg sync.WaitGroup
	forwardWg.Add(1)
	go func() {
		defer forwardWg.Done()
		for p := range indexerProgress {
			// Safely send progress (channel may be closed by caller)
			safeSendProgress(progressCh, IndexProgress{
				Running:   true,
				Total:     p.Total,
				Completed: p.Completed,
				Failed:    p.Failed,
				Message:   "Indexing skills for semantic search...",
			})
		}
	}()

	// Run indexing
	err = bi.indexer.IndexPending(ctx, indexerProgress)
	close(indexerProgress)

	// Wait for forwarding goroutine to finish
	forwardWg.Wait()

	if err != nil {
		log.Errorf("background indexing error: %v", err)
		safeSendProgress(progressCh, IndexProgress{
			Running: false,
			Message: "Indexing failed: " + err.Error(),
		})
		return
	}

	// Send completion
	safeSendProgress(progressCh, IndexProgress{
		Running:   false,
		Completed: pending,
		Message:   "Indexing complete",
	})
}

// IsRunning returns whether indexing is in progress.
func (bi *BackgroundIndexer) IsRunning() bool {
	bi.mu.Lock()
	defer bi.mu.Unlock()
	return bi.running
}

// Wait blocks until indexing completes.
func (bi *BackgroundIndexer) Wait() {
	<-bi.done
}

// VectorStore returns the underlying vector store for search operations.
func (bi *BackgroundIndexer) VectorStore() vector.VectorStore {
	return bi.store
}

// safeSendProgress sends a progress update to the channel, recovering from panics
// if the channel has been closed by the caller.
func safeSendProgress(ch chan<- IndexProgress, progress IndexProgress) {
	if ch == nil {
		return
	}
	defer func() {
		// Recover from panic if channel is closed
		_ = recover()
	}()
	ch <- progress
}
