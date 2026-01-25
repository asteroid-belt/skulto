// Package llm provides interfaces and implementations for LLM providers.
package llm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	openai "github.com/sashabaranov/go-openai"
)

const (
	// OpenRouterBaseURL is the base URL for OpenRouter's OpenAI-compatible API.
	OpenRouterBaseURL = "https://openrouter.ai/api/v1"

	// OpenRouterDefaultModel is the default model for OpenRouter.
	OpenRouterDefaultModel = "anthropic/claude-3-haiku"
)

// OpenRouterModels lists the available models via OpenRouter.
var OpenRouterModels = []string{
	"anthropic/claude-3-haiku",
	"openai/gpt-4o-mini",
	"meta-llama/llama-3-70b-instruct",
	"mistralai/mistral-large",
}

// OpenRouterProvider implements the Provider interface for OpenRouter.
type OpenRouterProvider struct {
	client       *openai.Client
	defaultModel string
}

// openRouterTransport is a custom HTTP transport that adds required OpenRouter headers.
type openRouterTransport struct {
	base http.RoundTripper
}

func (t *openRouterTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Add required OpenRouter headers
	req.Header.Set("HTTP-Referer", "https://github.com/asteroid-belt/skulto")
	req.Header.Set("X-Title", "Skulto Skill Builder")
	return t.base.RoundTrip(req)
}

// newOpenRouterProvider creates an OpenRouter provider (internal, for testing with custom client).
func newOpenRouterProvider(apiKey, model string) (*OpenRouterProvider, error) {
	if apiKey == "" {
		return nil, errors.New("OpenRouter API key is required")
	}

	// Configure the OpenAI client with OpenRouter's base URL
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = OpenRouterBaseURL

	// Add custom HTTP client with required headers
	config.HTTPClient = &http.Client{
		Transport: &openRouterTransport{
			base: http.DefaultTransport,
		},
	}

	client := openai.NewClientWithConfig(config)

	// Use default model if none specified
	if model == "" {
		model = OpenRouterDefaultModel
	}

	return &OpenRouterProvider{
		client:       client,
		defaultModel: model,
	}, nil
}

// Chat sends messages and returns a streaming response.
func (p *OpenRouterProvider) Chat(ctx context.Context, messages []Message, opts ChatOptions) (*StreamReader, error) {
	model := opts.Model
	if model == "" {
		model = p.defaultModel
	}

	// Convert messages to OpenAI format
	openaiMessages := make([]openai.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		openaiMessages[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Build request
	req := openai.ChatCompletionRequest{
		Model:    model,
		Messages: openaiMessages,
		Stream:   true,
	}

	if opts.MaxTokens > 0 {
		req.MaxTokens = opts.MaxTokens
	}

	if opts.Temperature > 0 {
		req.Temperature = float32(opts.Temperature)
	}

	// Create streaming request
	stream, err := p.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("openrouter stream error: %w", err)
	}

	// Create stream reader and start goroutine to process chunks
	reader := NewStreamReader()

	go func() {
		defer reader.Close()
		defer func() {
			_ = stream.Close()
		}()

		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				reader.Send(StreamChunk{Done: true})
				return
			}
			if err != nil {
				reader.Send(StreamChunk{Error: fmt.Errorf("openrouter stream recv error: %w", err)})
				return
			}

			// Extract content from response
			if len(response.Choices) > 0 {
				delta := response.Choices[0].Delta
				if delta.Content != "" {
					reader.Send(StreamChunk{Text: delta.Content})
				}

				// Check for finish reason
				if response.Choices[0].FinishReason != "" {
					reader.Send(StreamChunk{Done: true})
					return
				}
			}
		}
	}()

	return reader, nil
}

// ChatSync sends messages and waits for complete response.
func (p *OpenRouterProvider) ChatSync(ctx context.Context, messages []Message, opts ChatOptions) (*Response, error) {
	model := opts.Model
	if model == "" {
		model = p.defaultModel
	}

	// Convert messages to OpenAI format
	openaiMessages := make([]openai.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		openaiMessages[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Build request
	req := openai.ChatCompletionRequest{
		Model:    model,
		Messages: openaiMessages,
	}

	if opts.MaxTokens > 0 {
		req.MaxTokens = opts.MaxTokens
	}

	if opts.Temperature > 0 {
		req.Temperature = float32(opts.Temperature)
	}

	// Make synchronous request
	resp, err := p.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("openrouter chat error: %w", err)
	}

	// Extract response
	if len(resp.Choices) == 0 {
		return nil, errors.New("openrouter returned no choices")
	}

	choice := resp.Choices[0]
	return &Response{
		Content:      choice.Message.Content,
		Model:        resp.Model,
		FinishReason: string(choice.FinishReason),
		Usage: Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

// Name returns the provider name.
func (p *OpenRouterProvider) Name() string {
	return "openrouter"
}

// Models returns available model IDs for this provider.
func (p *OpenRouterProvider) Models() []string {
	return OpenRouterModels
}

// DefaultModel returns the default model for this provider.
func (p *OpenRouterProvider) DefaultModel() string {
	return p.defaultModel
}

// GetClient returns the underlying OpenAI client (for testing).
func (p *OpenRouterProvider) GetClient() *openai.Client {
	return p.client
}

// GetBaseURL returns the configured base URL (for testing).
func (p *OpenRouterProvider) GetBaseURL() string {
	return OpenRouterBaseURL
}
