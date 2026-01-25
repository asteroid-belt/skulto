package llm

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamReader_Basic(t *testing.T) {
	sr := NewStreamReader()

	// Send chunks in a goroutine
	go func() {
		sr.Send(StreamChunk{Text: "Hello "})
		sr.Send(StreamChunk{Text: "World"})
		sr.Send(StreamChunk{Text: "!", Done: true})
		sr.Close()
	}()

	// Collect all chunks
	var result string
	for sr.Next() {
		chunk := sr.Current()
		result += chunk.Text
	}

	assert.Equal(t, "Hello World!", result)
}

func TestStreamReader_Collect(t *testing.T) {
	sr := NewStreamReader()

	go func() {
		sr.Send(StreamChunk{Text: "Hello "})
		sr.Send(StreamChunk{Text: "World"})
		sr.Send(StreamChunk{Text: "!", Done: true})
		sr.Close()
	}()

	result, err := sr.Collect()
	require.NoError(t, err)
	assert.Equal(t, "Hello World!", result)
}

func TestStreamReader_CollectWithError(t *testing.T) {
	sr := NewStreamReader()
	testErr := errors.New("test error")

	go func() {
		sr.Send(StreamChunk{Text: "Hello "})
		sr.Send(StreamChunk{Error: testErr})
		sr.Send(StreamChunk{Text: "World", Done: true})
		sr.Close()
	}()

	result, err := sr.Collect()
	assert.Error(t, err)
	assert.Equal(t, testErr, err)
	// Should still have partial content
	assert.Equal(t, "Hello World", result)
}

func TestStreamReader_CollectWithCallback(t *testing.T) {
	sr := NewStreamReader()

	go func() {
		sr.Send(StreamChunk{Text: "A"})
		sr.Send(StreamChunk{Text: "B"})
		sr.Send(StreamChunk{Text: "C", Done: true})
		sr.Close()
	}()

	var chunks []string
	result, err := sr.CollectWithCallback(func(chunk StreamChunk) {
		chunks = append(chunks, chunk.Text)
	})

	require.NoError(t, err)
	assert.Equal(t, "ABC", result)
	assert.Equal(t, []string{"A", "B", "C"}, chunks)
}

func TestStreamReader_CloseMultipleTimes(t *testing.T) {
	sr := NewStreamReader()

	// Should not panic
	sr.Close()
	sr.Close()
	sr.Close()

	assert.True(t, sr.closed)
}

func TestStreamReader_SendAfterClose(t *testing.T) {
	sr := NewStreamReader()
	sr.Close()

	// Should not panic, just be a no-op
	sr.Send(StreamChunk{Text: "ignored"})

	// Channel should be closed
	_, ok := <-sr.chunks
	assert.False(t, ok)
}

func TestStreamReader_ConcurrentAccess(t *testing.T) {
	sr := NewStreamReader()

	var wg sync.WaitGroup
	wg.Add(2)

	// Writer goroutine
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			sr.Send(StreamChunk{Text: "x"})
		}
		sr.Send(StreamChunk{Done: true})
		sr.Close()
	}()

	// Reader goroutine
	var count int
	go func() {
		defer wg.Done()
		for sr.Next() {
			count++
		}
	}()

	wg.Wait()
	assert.Equal(t, 101, count) // 100 "x" chunks + 1 done chunk
}

func TestStreamReader_Err(t *testing.T) {
	sr := NewStreamReader()
	testErr := errors.New("stream error")

	go func() {
		sr.Send(StreamChunk{Error: testErr})
		sr.Close()
	}()

	require.True(t, sr.Next())
	assert.Equal(t, testErr, sr.Err())
}

func TestStreamReader_EmptyStream(t *testing.T) {
	sr := NewStreamReader()

	go func() {
		sr.Close()
	}()

	// Should return false immediately
	result := sr.Next()
	assert.False(t, result)
}

func TestStreamReader_DoneStopsCollect(t *testing.T) {
	sr := NewStreamReader()

	go func() {
		sr.Send(StreamChunk{Text: "First "})
		sr.Send(StreamChunk{Text: "Second", Done: true})
		// These should be ignored by Collect since Done was sent
		time.Sleep(10 * time.Millisecond)
		sr.Send(StreamChunk{Text: " Third"})
		sr.Close()
	}()

	result, err := sr.Collect()
	require.NoError(t, err)
	assert.Equal(t, "First Second", result)
}

func TestStreamChunk_Fields(t *testing.T) {
	chunk := StreamChunk{
		Text:  "content",
		Done:  true,
		Error: errors.New("error"),
	}

	assert.Equal(t, "content", chunk.Text)
	assert.True(t, chunk.Done)
	assert.Error(t, chunk.Error)
}
