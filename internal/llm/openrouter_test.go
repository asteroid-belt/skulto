package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOpenRouterProvider_ValidKey(t *testing.T) {
	provider, err := NewOpenRouterProvider("sk-or-test-key", "")
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestNewOpenRouterProvider_EmptyKey(t *testing.T) {
	provider, err := NewOpenRouterProvider("", "")
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "API key is required")
}

func TestNewOpenRouterProvider_DefaultModel(t *testing.T) {
	provider, err := NewOpenRouterProvider("sk-or-test-key", "")
	require.NoError(t, err)
	assert.Equal(t, OpenRouterDefaultModel, provider.DefaultModel())
}

func TestNewOpenRouterProvider_CustomModel(t *testing.T) {
	customModel := "openai/gpt-4o-mini"
	provider, err := NewOpenRouterProvider("sk-or-test-key", customModel)
	require.NoError(t, err)
	assert.Equal(t, customModel, provider.DefaultModel())
}

func TestOpenRouterProvider_Name(t *testing.T) {
	provider, err := NewOpenRouterProvider("sk-or-test-key", "")
	require.NoError(t, err)
	assert.Equal(t, "openrouter", provider.Name())
}

func TestOpenRouterProvider_Models(t *testing.T) {
	provider, err := NewOpenRouterProvider("sk-or-test-key", "")
	require.NoError(t, err)

	models := provider.Models()
	assert.Contains(t, models, "anthropic/claude-3-haiku")
	assert.Contains(t, models, "openai/gpt-4o-mini")
	assert.Contains(t, models, "meta-llama/llama-3-70b-instruct")
	assert.Contains(t, models, "mistralai/mistral-large")
	assert.Len(t, models, 4)
}

func TestOpenRouterProvider_BaseURL(t *testing.T) {
	provider, err := NewOpenRouterProvider("sk-or-test-key", "")
	require.NoError(t, err)

	orProvider := provider.(*OpenRouterProvider)
	assert.Equal(t, OpenRouterBaseURL, orProvider.GetBaseURL())
}

func TestOpenRouterProvider_GetClient(t *testing.T) {
	provider, err := NewOpenRouterProvider("sk-or-test-key", "")
	require.NoError(t, err)

	orProvider := provider.(*OpenRouterProvider)
	assert.NotNil(t, orProvider.GetClient())
}

func TestOpenRouterTransport_Headers(t *testing.T) {
	// Create a test server that captures headers
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()
		// Return a valid OpenAI response
		resp := openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Content: "test response",
					},
					FinishReason: "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create provider with custom base URL pointing to test server
	config := openai.DefaultConfig("test-key")
	config.BaseURL = server.URL
	config.HTTPClient = &http.Client{
		Transport: &openRouterTransport{
			base: http.DefaultTransport,
		},
	}

	client := openai.NewClientWithConfig(config)
	provider := &OpenRouterProvider{
		client:       client,
		defaultModel: OpenRouterDefaultModel,
	}

	// Make a request
	ctx := context.Background()
	_, err := provider.ChatSync(ctx, []Message{
		{Role: "user", Content: "test"},
	}, ChatOptions{})

	require.NoError(t, err)

	// Verify headers were set
	assert.Equal(t, "https://github.com/asteroid-belt/skulto", capturedHeaders.Get("HTTP-Referer"))
	assert.Equal(t, "Skulto Skill Builder", capturedHeaders.Get("X-Title"))
}

func TestOpenRouterProvider_ChatSync_Success(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "chat/completions")

		// Parse request body
		var req openai.ChatCompletionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		assert.Equal(t, "anthropic/claude-3-haiku", req.Model)
		assert.Len(t, req.Messages, 2)
		assert.Equal(t, "system", req.Messages[0].Role)
		assert.Equal(t, "user", req.Messages[1].Role)

		// Return response
		resp := openai.ChatCompletionResponse{
			Model: "anthropic/claude-3-haiku",
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Role:    "assistant",
						Content: "Hello! How can I help you today?",
					},
					FinishReason: "stop",
				},
			},
			Usage: openai.Usage{
				PromptTokens:     10,
				CompletionTokens: 8,
				TotalTokens:      18,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create provider with mock server
	provider := createTestProvider(t, server.URL)

	// Make request
	ctx := context.Background()
	resp, err := provider.ChatSync(ctx, []Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "Hello!"},
	}, ChatOptions{})

	require.NoError(t, err)
	assert.Equal(t, "Hello! How can I help you today?", resp.Content)
	assert.Equal(t, "anthropic/claude-3-haiku", resp.Model)
	assert.Equal(t, "stop", resp.FinishReason)
	assert.Equal(t, 10, resp.Usage.PromptTokens)
	assert.Equal(t, 8, resp.Usage.CompletionTokens)
	assert.Equal(t, 18, resp.Usage.TotalTokens)
}

func TestOpenRouterProvider_ChatSync_WithOptions(t *testing.T) {
	// Create a mock server that verifies options
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openai.ChatCompletionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Verify options were applied
		assert.Equal(t, "openai/gpt-4o-mini", req.Model)
		assert.Equal(t, 100, req.MaxTokens)
		assert.InDelta(t, 0.7, req.Temperature, 0.01)

		resp := openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Content: "response",
					},
					FinishReason: "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := createTestProvider(t, server.URL)

	ctx := context.Background()
	_, err := provider.ChatSync(ctx, []Message{
		{Role: "user", Content: "test"},
	}, ChatOptions{
		Model:       "openai/gpt-4o-mini",
		MaxTokens:   100,
		Temperature: 0.7,
	})

	require.NoError(t, err)
}

func TestOpenRouterProvider_ChatSync_NoChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := createTestProvider(t, server.URL)

	ctx := context.Background()
	_, err := provider.ChatSync(ctx, []Message{
		{Role: "user", Content: "test"},
	}, ChatOptions{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no choices")
}

func TestOpenRouterProvider_ChatSync_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": {"message": "Invalid API key", "type": "invalid_request_error"}}`))
	}))
	defer server.Close()

	provider := createTestProvider(t, server.URL)

	ctx := context.Background()
	_, err := provider.ChatSync(ctx, []Message{
		{Role: "user", Content: "test"},
	}, ChatOptions{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "openrouter chat error")
}

func TestOpenRouterProvider_Chat_Streaming(t *testing.T) {
	// Create a mock SSE server for streaming
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify streaming was requested
		var req openai.ChatCompletionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.True(t, req.Stream)

		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		require.True(t, ok)

		// Send chunks
		chunks := []string{"Hello", " ", "World", "!"}
		for i, chunk := range chunks {
			resp := openai.ChatCompletionStreamResponse{
				Choices: []openai.ChatCompletionStreamChoice{
					{
						Delta: openai.ChatCompletionStreamChoiceDelta{
							Content: chunk,
						},
					},
				},
			}

			// Last chunk has finish reason
			if i == len(chunks)-1 {
				resp.Choices[0].FinishReason = "stop"
			}

			data, _ := json.Marshal(resp)
			_, _ = w.Write([]byte("data: "))
			_, _ = w.Write(data)
			_, _ = w.Write([]byte("\n\n"))
			flusher.Flush()
		}

		// Send done
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	provider := createTestProvider(t, server.URL)

	ctx := context.Background()
	reader, err := provider.Chat(ctx, []Message{
		{Role: "user", Content: "Say hello"},
	}, ChatOptions{})

	require.NoError(t, err)
	require.NotNil(t, reader)

	// Collect the streamed response
	result, err := reader.Collect()
	require.NoError(t, err)
	assert.Equal(t, "Hello World!", result)
}

func TestOpenRouterProvider_Chat_StreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": {"message": "Server error"}}`))
	}))
	defer server.Close()

	provider := createTestProvider(t, server.URL)

	ctx := context.Background()
	_, err := provider.Chat(ctx, []Message{
		{Role: "user", Content: "test"},
	}, ChatOptions{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "openrouter stream error")
}

func TestOpenRouterProvider_Chat_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(5 * time.Second)
	}))
	defer server.Close()

	provider := createTestProvider(t, server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := provider.Chat(ctx, []Message{
		{Role: "user", Content: "test"},
	}, ChatOptions{})

	assert.Error(t, err)
}

func TestOpenRouterProvider_MessageConversion(t *testing.T) {
	var capturedMessages []openai.ChatCompletionMessage

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openai.ChatCompletionRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		capturedMessages = req.Messages

		resp := openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{
				{
					Message:      openai.ChatCompletionMessage{Content: "ok"},
					FinishReason: "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := createTestProvider(t, server.URL)

	ctx := context.Background()
	_, err := provider.ChatSync(ctx, []Message{
		NewSystemMessage("You are helpful"),
		NewUserMessage("Hello"),
		NewAssistantMessage("Hi there"),
		NewUserMessage("How are you?"),
	}, ChatOptions{})

	require.NoError(t, err)
	require.Len(t, capturedMessages, 4)

	assert.Equal(t, "system", capturedMessages[0].Role)
	assert.Equal(t, "You are helpful", capturedMessages[0].Content)

	assert.Equal(t, "user", capturedMessages[1].Role)
	assert.Equal(t, "Hello", capturedMessages[1].Content)

	assert.Equal(t, "assistant", capturedMessages[2].Role)
	assert.Equal(t, "Hi there", capturedMessages[2].Content)

	assert.Equal(t, "user", capturedMessages[3].Role)
	assert.Equal(t, "How are you?", capturedMessages[3].Content)
}

func TestOpenRouterProvider_ImplementsInterface(t *testing.T) {
	provider, err := NewOpenRouterProvider("sk-or-test-key", "")
	require.NoError(t, err)

	// Verify it implements the Provider interface
	_ = (Provider)(provider)
}

func TestOpenRouterConstants(t *testing.T) {
	assert.Equal(t, "https://openrouter.ai/api/v1", OpenRouterBaseURL)
	assert.Equal(t, "anthropic/claude-3-haiku", OpenRouterDefaultModel)
}

func TestOpenRouterModels_AllExpected(t *testing.T) {
	expected := []string{
		"anthropic/claude-3-haiku",
		"openai/gpt-4o-mini",
		"meta-llama/llama-3-70b-instruct",
		"mistralai/mistral-large",
	}

	assert.Equal(t, expected, OpenRouterModels)
}

func TestOpenRouterProvider_Chat_EmptyContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		// Send chunk with empty content (should be skipped)
		resp := openai.ChatCompletionStreamResponse{
			Choices: []openai.ChatCompletionStreamChoice{
				{
					Delta: openai.ChatCompletionStreamChoiceDelta{
						Content: "",
					},
				},
			},
		}
		data, _ := json.Marshal(resp)
		_, _ = w.Write([]byte("data: "))
		_, _ = w.Write(data)
		_, _ = w.Write([]byte("\n\n"))
		flusher.Flush()

		// Send actual content
		resp.Choices[0].Delta.Content = "Hello"
		resp.Choices[0].FinishReason = "stop"
		data, _ = json.Marshal(resp)
		_, _ = w.Write([]byte("data: "))
		_, _ = w.Write(data)
		_, _ = w.Write([]byte("\n\n"))
		flusher.Flush()

		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	provider := createTestProvider(t, server.URL)

	ctx := context.Background()
	reader, err := provider.Chat(ctx, []Message{
		{Role: "user", Content: "test"},
	}, ChatOptions{})

	require.NoError(t, err)
	result, err := reader.Collect()
	require.NoError(t, err)
	assert.Equal(t, "Hello", result)
}

func TestOpenRouterProvider_ChatSync_DefaultModel(t *testing.T) {
	var capturedModel string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openai.ChatCompletionRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		capturedModel = req.Model

		resp := openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{
				{
					Message:      openai.ChatCompletionMessage{Content: "ok"},
					FinishReason: "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := createTestProvider(t, server.URL)

	ctx := context.Background()
	_, err := provider.ChatSync(ctx, []Message{
		{Role: "user", Content: "test"},
	}, ChatOptions{}) // No model specified

	require.NoError(t, err)
	assert.Equal(t, OpenRouterDefaultModel, capturedModel)
}

// Helper to create a test provider with custom base URL
func createTestProvider(t *testing.T, baseURL string) *OpenRouterProvider {
	config := openai.DefaultConfig("test-api-key")
	config.BaseURL = baseURL
	config.HTTPClient = &http.Client{
		Transport: &openRouterTransport{
			base: http.DefaultTransport,
		},
	}

	client := openai.NewClientWithConfig(config)

	return &OpenRouterProvider{
		client:       client,
		defaultModel: OpenRouterDefaultModel,
	}
}

// Test for proper handling of stream with multiple chunks
func TestOpenRouterProvider_Chat_MultipleChunks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		// Send multiple chunks
		words := strings.Split("The quick brown fox jumps over the lazy dog", " ")
		for i, word := range words {
			resp := openai.ChatCompletionStreamResponse{
				Choices: []openai.ChatCompletionStreamChoice{
					{
						Delta: openai.ChatCompletionStreamChoiceDelta{
							Content: word,
						},
					},
				},
			}

			// Add space after all but last word
			if i < len(words)-1 {
				resp.Choices[0].Delta.Content = word + " "
			}

			// Last chunk has finish reason
			if i == len(words)-1 {
				resp.Choices[0].FinishReason = "stop"
			}

			data, _ := json.Marshal(resp)
			_, _ = w.Write([]byte("data: "))
			_, _ = w.Write(data)
			_, _ = w.Write([]byte("\n\n"))
			flusher.Flush()
		}

		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	provider := createTestProvider(t, server.URL)

	ctx := context.Background()
	reader, err := provider.Chat(ctx, []Message{
		{Role: "user", Content: "test"},
	}, ChatOptions{})

	require.NoError(t, err)

	var chunks []string
	for reader.Next() {
		chunk := reader.Current()
		if chunk.Text != "" {
			chunks = append(chunks, chunk.Text)
		}
		if chunk.Done {
			break
		}
	}

	// Verify we got all chunks
	assert.Len(t, chunks, 9) // 9 words
	assert.Equal(t, "The quick brown fox jumps over the lazy dog", strings.Join(chunks, ""))
}

// Test that the provider properly handles io.EOF during streaming
func TestOpenRouterProvider_Chat_StreamEOF(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		resp := openai.ChatCompletionStreamResponse{
			Choices: []openai.ChatCompletionStreamChoice{
				{
					Delta: openai.ChatCompletionStreamChoiceDelta{
						Content: "Hello",
					},
				},
			},
		}
		data, _ := json.Marshal(resp)
		_, _ = w.Write([]byte("data: "))
		_, _ = w.Write(data)
		_, _ = w.Write([]byte("\n\n"))
		flusher.Flush()

		// Send [DONE] to trigger EOF
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	provider := createTestProvider(t, server.URL)

	ctx := context.Background()
	reader, err := provider.Chat(ctx, []Message{
		{Role: "user", Content: "test"},
	}, ChatOptions{})

	require.NoError(t, err)
	result, err := reader.Collect()
	require.NoError(t, err)
	assert.Equal(t, "Hello", result)
}

// Test provider with server that returns malformed JSON
func TestOpenRouterProvider_ChatSync_MalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	provider := createTestProvider(t, server.URL)

	ctx := context.Background()
	_, err := provider.ChatSync(ctx, []Message{
		{Role: "user", Content: "test"},
	}, ChatOptions{})

	assert.Error(t, err)
}

// Test that streaming properly closes resources
func TestOpenRouterProvider_Chat_ResourceCleanup(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		resp := openai.ChatCompletionStreamResponse{
			Choices: []openai.ChatCompletionStreamChoice{
				{
					Delta: openai.ChatCompletionStreamChoiceDelta{
						Content: "test",
					},
					FinishReason: "stop",
				},
			},
		}
		data, _ := json.Marshal(resp)
		_, _ = w.Write([]byte("data: "))
		_, _ = w.Write(data)
		_, _ = w.Write([]byte("\n\n"))
		flusher.Flush()

		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	provider := createTestProvider(t, server.URL)

	// Make multiple requests
	for i := 0; i < 3; i++ {
		ctx := context.Background()
		reader, err := provider.Chat(ctx, []Message{
			{Role: "user", Content: "test"},
		}, ChatOptions{})

		require.NoError(t, err)
		_, err = reader.Collect()
		require.NoError(t, err)
	}

	// Verify all requests completed
	assert.Equal(t, 3, requestCount)
}

// Verify transport doesn't modify other headers
func TestOpenRouterTransport_PreservesOtherHeaders(t *testing.T) {
	var capturedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()

		resp := openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{
				{
					Message:      openai.ChatCompletionMessage{Content: "ok"},
					FinishReason: "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := createTestProvider(t, server.URL)

	ctx := context.Background()
	_, _ = provider.ChatSync(ctx, []Message{
		{Role: "user", Content: "test"},
	}, ChatOptions{})

	// Verify standard headers are preserved
	assert.NotEmpty(t, capturedHeaders.Get("Authorization"))
	assert.NotEmpty(t, capturedHeaders.Get("Content-Type"))

	// Verify our custom headers are added
	assert.Equal(t, "https://github.com/asteroid-belt/skulto", capturedHeaders.Get("HTTP-Referer"))
	assert.Equal(t, "Skulto Skill Builder", capturedHeaders.Get("X-Title"))
}

func TestOpenRouterProvider_Chat_ReadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Close connection immediately to simulate read error
		hj, ok := w.(http.Hijacker)
		if ok {
			conn, _, _ := hj.Hijack()
			_ = conn.Close()
		}
	}))
	defer server.Close()

	provider := createTestProvider(t, server.URL)

	ctx := context.Background()
	_, err := provider.Chat(ctx, []Message{
		{Role: "user", Content: "test"},
	}, ChatOptions{})

	// Should get an error when trying to create the stream
	assert.Error(t, err)
}

// Test empty messages array
func TestOpenRouterProvider_ChatSync_EmptyMessages(t *testing.T) {
	requestMade := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMade = true
		var req openai.ChatCompletionRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		// Empty messages should still be sent
		resp := openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{
				{
					Message:      openai.ChatCompletionMessage{Content: "ok"},
					FinishReason: "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := createTestProvider(t, server.URL)

	ctx := context.Background()
	_, err := provider.ChatSync(ctx, []Message{}, ChatOptions{})

	require.NoError(t, err)
	assert.True(t, requestMade)
}

// Benchmark streaming performance
func BenchmarkOpenRouterProvider_Chat_Streaming(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		for i := 0; i < 100; i++ {
			resp := openai.ChatCompletionStreamResponse{
				Choices: []openai.ChatCompletionStreamChoice{
					{
						Delta: openai.ChatCompletionStreamChoiceDelta{
							Content: "chunk",
						},
					},
				},
			}
			data, _ := json.Marshal(resp)
			_, _ = w.Write([]byte("data: "))
			_, _ = w.Write(data)
			_, _ = w.Write([]byte("\n\n"))
			flusher.Flush()
		}

		// Final chunk
		resp := openai.ChatCompletionStreamResponse{
			Choices: []openai.ChatCompletionStreamChoice{
				{
					Delta:        openai.ChatCompletionStreamChoiceDelta{},
					FinishReason: "stop",
				},
			},
		}
		data, _ := json.Marshal(resp)
		_, _ = w.Write([]byte("data: "))
		_, _ = w.Write(data)
		_, _ = w.Write([]byte("\n\n"))
		flusher.Flush()

		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	config := openai.DefaultConfig("test-api-key")
	config.BaseURL = server.URL
	config.HTTPClient = &http.Client{
		Transport: &openRouterTransport{
			base: http.DefaultTransport,
		},
	}

	provider := &OpenRouterProvider{
		client:       openai.NewClientWithConfig(config),
		defaultModel: OpenRouterDefaultModel,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		reader, _ := provider.Chat(ctx, []Message{
			{Role: "user", Content: "test"},
		}, ChatOptions{})
		_, _ = io.Copy(io.Discard, &streamReaderWrapper{reader})
	}
}

// Helper wrapper to use StreamReader as io.Reader
type streamReaderWrapper struct {
	sr *StreamReader
}

func (w *streamReaderWrapper) Read(p []byte) (n int, err error) {
	if w.sr.Next() {
		chunk := w.sr.Current()
		if chunk.Error != nil {
			return 0, chunk.Error
		}
		if chunk.Done {
			return 0, io.EOF
		}
		copy(p, []byte(chunk.Text))
		return len(chunk.Text), nil
	}
	return 0, io.EOF
}
