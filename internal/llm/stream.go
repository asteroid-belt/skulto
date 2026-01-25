package llm

import (
	"strings"
	"sync"
)

// StreamChunk represents a piece of a streaming response.
type StreamChunk struct {
	Text  string // Text content in this chunk
	Done  bool   // True if this is the final chunk
	Error error  // Error if streaming failed
}

// StreamReader provides an iterator interface for streaming responses.
type StreamReader struct {
	chunks  chan StreamChunk
	current StreamChunk
	closed  bool
	mu      sync.Mutex
}

// NewStreamReader creates a new stream reader.
func NewStreamReader() *StreamReader {
	return &StreamReader{
		chunks: make(chan StreamChunk, 100), // Buffered channel for smoother streaming
	}
}

// Send sends a chunk to the stream. Called by provider implementations.
func (sr *StreamReader) Send(chunk StreamChunk) {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	if sr.closed {
		return
	}

	sr.chunks <- chunk
}

// Close closes the stream. Called by provider implementations when done.
func (sr *StreamReader) Close() {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	if !sr.closed {
		sr.closed = true
		close(sr.chunks)
	}
}

// Next advances to the next chunk. Returns false when stream is exhausted.
func (sr *StreamReader) Next() bool {
	chunk, ok := <-sr.chunks
	if !ok {
		return false
	}
	sr.current = chunk
	return true
}

// Current returns the current chunk. Call after Next() returns true.
func (sr *StreamReader) Current() StreamChunk {
	return sr.current
}

// Err returns any error from the current chunk.
func (sr *StreamReader) Err() error {
	return sr.current.Error
}

// Collect reads all chunks and returns the complete text.
// Useful for non-streaming consumption of a streaming response.
func (sr *StreamReader) Collect() (string, error) {
	var builder strings.Builder
	var lastErr error

	for sr.Next() {
		chunk := sr.Current()
		if chunk.Error != nil {
			lastErr = chunk.Error
			continue
		}
		builder.WriteString(chunk.Text)
		if chunk.Done {
			break
		}
	}

	if lastErr != nil {
		return builder.String(), lastErr
	}

	return builder.String(), nil
}

// CollectWithCallback reads all chunks, calling the callback for each.
// Useful for updating UI while collecting the full response.
func (sr *StreamReader) CollectWithCallback(callback func(chunk StreamChunk)) (string, error) {
	var builder strings.Builder
	var lastErr error

	for sr.Next() {
		chunk := sr.Current()
		if callback != nil {
			callback(chunk)
		}
		if chunk.Error != nil {
			lastErr = chunk.Error
			continue
		}
		builder.WriteString(chunk.Text)
		if chunk.Done {
			break
		}
	}

	if lastErr != nil {
		return builder.String(), lastErr
	}

	return builder.String(), nil
}
