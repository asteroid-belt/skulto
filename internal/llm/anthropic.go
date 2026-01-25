// Package llm provides interfaces and implementations for LLM providers.
package llm

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/ssestream"
)

// AnthropicModels lists available Anthropic models.
var AnthropicModels = []string{
	"claude-3-haiku-20240307",    // Fast and cheap, good for skill building
	"claude-3-5-sonnet-20241022", // Better quality, more expensive
	"claude-3-5-haiku-20241022",  // Newer haiku version
	"claude-3-opus-20240229",     // Highest quality, most expensive
}

// DefaultAnthropicModel is the default model for skill building (fast and cheap).
const DefaultAnthropicModel = "claude-3-haiku-20240307"

// AnthropicClientInterface defines the interface for Anthropic API client.
// This allows for mocking in tests.
type AnthropicClientInterface interface {
	CreateMessage(ctx context.Context, params anthropic.MessageNewParams) (*anthropic.Message, error)
	CreateMessageStream(ctx context.Context, params anthropic.MessageNewParams) *ssestream.Stream[anthropic.MessageStreamEventUnion]
}

// anthropicClientWrapper wraps the real Anthropic client to implement AnthropicClientInterface.
type anthropicClientWrapper struct {
	client anthropic.Client
}

func (w *anthropicClientWrapper) CreateMessage(ctx context.Context, params anthropic.MessageNewParams) (*anthropic.Message, error) {
	return w.client.Messages.New(ctx, params)
}

func (w *anthropicClientWrapper) CreateMessageStream(ctx context.Context, params anthropic.MessageNewParams) *ssestream.Stream[anthropic.MessageStreamEventUnion] {
	return w.client.Messages.NewStreaming(ctx, params)
}

// AnthropicProvider implements Provider using Anthropic's API.
type AnthropicProvider struct {
	client AnthropicClientInterface
	model  string
}

// NewAnthropicProviderImpl creates a new Anthropic provider.
// This is the actual implementation, called by NewAnthropicProvider in provider.go.
func NewAnthropicProviderImpl(apiKey, model string) (*AnthropicProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	if model == "" {
		model = DefaultAnthropicModel
	}

	if !isValidAnthropicModel(model) {
		return nil, fmt.Errorf("invalid Anthropic model: %s", model)
	}

	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	return &AnthropicProvider{
		client: &anthropicClientWrapper{client: client},
		model:  model,
	}, nil
}

// NewAnthropicProviderWithClient creates an Anthropic provider with a custom client.
// This is useful for testing.
func NewAnthropicProviderWithClient(client AnthropicClientInterface, model string) *AnthropicProvider {
	if model == "" {
		model = DefaultAnthropicModel
	}
	return &AnthropicProvider{
		client: client,
		model:  model,
	}
}

// isValidAnthropicModel checks if the given model is a valid Anthropic model.
func isValidAnthropicModel(model string) bool {
	for _, m := range AnthropicModels {
		if m == model {
			return true
		}
	}
	return false
}

// Chat sends messages and returns a streaming response.
func (p *AnthropicProvider) Chat(ctx context.Context, messages []Message, opts ChatOptions) (*StreamReader, error) {
	model := opts.Model
	if model == "" {
		model = p.model
	}

	maxTokens := opts.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	// Convert messages to Anthropic format
	anthropicMessages, systemPrompt := p.convertMessages(messages)

	sr := NewStreamReader()

	go func() {
		defer sr.Close()

		params := anthropic.MessageNewParams{
			Model:     anthropic.Model(model),
			MaxTokens: int64(maxTokens),
			Messages:  anthropicMessages,
		}

		if systemPrompt != "" {
			params.System = []anthropic.TextBlockParam{
				{Text: systemPrompt},
			}
		}

		stream := p.client.CreateMessageStream(ctx, params)

		for stream.Next() {
			event := stream.Current()

			// Extract text from content block delta events
			switch eventVariant := event.AsAny().(type) {
			case anthropic.ContentBlockDeltaEvent:
				switch deltaVariant := eventVariant.Delta.AsAny().(type) {
				case anthropic.TextDelta:
					sr.Send(StreamChunk{Text: deltaVariant.Text})
				}
			case anthropic.MessageStopEvent:
				sr.Send(StreamChunk{Done: true})
			}
		}

		if err := stream.Err(); err != nil {
			sr.Send(StreamChunk{Error: fmt.Errorf("anthropic stream: %w", err)})
		}
	}()

	return sr, nil
}

// ChatSync sends messages and waits for complete response.
func (p *AnthropicProvider) ChatSync(ctx context.Context, messages []Message, opts ChatOptions) (*Response, error) {
	model := opts.Model
	if model == "" {
		model = p.model
	}

	maxTokens := opts.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	// Convert messages to Anthropic format
	anthropicMessages, systemPrompt := p.convertMessages(messages)

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: int64(maxTokens),
		Messages:  anthropicMessages,
	}

	if systemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: systemPrompt},
		}
	}

	msg, err := p.client.CreateMessage(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("anthropic chat: %w", err)
	}

	// Extract text content from the response
	// We check the Type field directly to support both real API responses
	// (where JSON.raw is populated) and mock responses in tests
	var content string
	for _, block := range msg.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return &Response{
		Content:      content,
		Model:        string(msg.Model),
		FinishReason: string(msg.StopReason),
		Usage: Usage{
			PromptTokens:     int(msg.Usage.InputTokens),
			CompletionTokens: int(msg.Usage.OutputTokens),
			TotalTokens:      int(msg.Usage.InputTokens + msg.Usage.OutputTokens),
		},
	}, nil
}

// convertMessages converts generic messages to Anthropic format.
// System messages are extracted and returned separately since Anthropic
// uses a dedicated system parameter.
func (p *AnthropicProvider) convertMessages(messages []Message) ([]anthropic.MessageParam, string) {
	var anthropicMessages []anthropic.MessageParam
	var systemPrompt string

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			// Anthropic uses a separate system parameter
			systemPrompt = msg.Content
		case "user":
			anthropicMessages = append(anthropicMessages, anthropic.NewUserMessage(
				anthropic.NewTextBlock(msg.Content),
			))
		case "assistant":
			anthropicMessages = append(anthropicMessages, anthropic.NewAssistantMessage(
				anthropic.NewTextBlock(msg.Content),
			))
		}
	}

	return anthropicMessages, systemPrompt
}

// Name returns the provider name.
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

// Models returns available models.
func (p *AnthropicProvider) Models() []string {
	return AnthropicModels
}

// DefaultModel returns the default model.
func (p *AnthropicProvider) DefaultModel() string {
	return DefaultAnthropicModel
}
