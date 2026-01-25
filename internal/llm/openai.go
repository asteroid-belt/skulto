// Package llm provides interfaces and implementations for LLM providers.
package llm

import (
	"context"
	"errors"
	"fmt"
	"io"

	openai "github.com/sashabaranov/go-openai"
)

// OpenAI model constants.
const (
	OpenAIModelGPT4oMini   = "gpt-4o-mini"
	OpenAIModelGPT4o       = "gpt-4o"
	OpenAIModelGPT4Turbo   = "gpt-4-turbo"
	OpenAIDefaultModel     = OpenAIModelGPT4oMini
	OpenAIDefaultMaxTokens = 4096
)

// openAIModels lists available OpenAI models.
var openAIModels = []string{
	OpenAIModelGPT4oMini,
	OpenAIModelGPT4o,
	OpenAIModelGPT4Turbo,
}

// OpenAIProvider implements the Provider interface for OpenAI.
type OpenAIProvider struct {
	client       *openai.Client
	model        string
	clientConfig openai.ClientConfig
}

// OpenAIClientFactory allows injecting a mock client for testing.
type OpenAIClientFactory func(config openai.ClientConfig) OpenAIClientInterface

// OpenAIClientInterface abstracts the OpenAI client for testing.
type OpenAIClientInterface interface {
	CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
	CreateChatCompletionStream(ctx context.Context, req openai.ChatCompletionRequest) (*openai.ChatCompletionStream, error)
}

// testableOpenAIProvider is used internally for testing with mock clients.
type testableOpenAIProvider struct {
	client OpenAIClientInterface
	model  string
}

// NewOpenAIProviderWithClient creates a provider with a custom client interface (for testing).
func NewOpenAIProviderWithClient(client OpenAIClientInterface, model string) Provider {
	if model == "" {
		model = OpenAIDefaultModel
	}
	return &testableOpenAIProvider{
		client: client,
		model:  model,
	}
}

// newOpenAIProvider creates a new OpenAI provider with the given API key and model.
func newOpenAIProvider(apiKey, model string) (*OpenAIProvider, error) {
	if apiKey == "" {
		return nil, errors.New("OpenAI API key is required")
	}

	if model == "" {
		model = OpenAIDefaultModel
	}

	// Validate model
	if !isValidOpenAIModel(model) {
		return nil, fmt.Errorf("invalid OpenAI model: %s (available: %v)", model, openAIModels)
	}

	config := openai.DefaultConfig(apiKey)
	client := openai.NewClientWithConfig(config)

	return &OpenAIProvider{
		client:       client,
		model:        model,
		clientConfig: config,
	}, nil
}

// isValidOpenAIModel checks if the model is a valid OpenAI model.
func isValidOpenAIModel(model string) bool {
	for _, m := range openAIModels {
		if m == model {
			return true
		}
	}
	return false
}

// Name returns the provider name.
func (p *OpenAIProvider) Name() string {
	return string(ProviderOpenAI)
}

// Models returns available model IDs.
func (p *OpenAIProvider) Models() []string {
	return openAIModels
}

// DefaultModel returns the default model.
func (p *OpenAIProvider) DefaultModel() string {
	return OpenAIDefaultModel
}

// Chat sends messages and returns a streaming response.
func (p *OpenAIProvider) Chat(ctx context.Context, messages []Message, opts ChatOptions) (*StreamReader, error) {
	model := opts.Model
	if model == "" {
		model = p.model
	}

	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = OpenAIDefaultMaxTokens
	}

	req := openai.ChatCompletionRequest{
		Model:       model,
		Messages:    convertToOpenAIMessages(messages),
		MaxTokens:   maxTokens,
		Temperature: float32(opts.Temperature),
		Stream:      true,
	}

	stream, err := p.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create stream: %w", err)
	}

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
				reader.Send(StreamChunk{Error: err})
				return
			}

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
func (p *OpenAIProvider) ChatSync(ctx context.Context, messages []Message, opts ChatOptions) (*Response, error) {
	model := opts.Model
	if model == "" {
		model = p.model
	}

	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = OpenAIDefaultMaxTokens
	}

	req := openai.ChatCompletionRequest{
		Model:       model,
		Messages:    convertToOpenAIMessages(messages),
		MaxTokens:   maxTokens,
		Temperature: float32(opts.Temperature),
	}

	resp, err := p.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, errors.New("no choices in response")
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

// convertToOpenAIMessages converts internal messages to OpenAI format.
func convertToOpenAIMessages(messages []Message) []openai.ChatCompletionMessage {
	result := make([]openai.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		result[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return result
}

// --- testableOpenAIProvider implementation ---

func (p *testableOpenAIProvider) Name() string {
	return string(ProviderOpenAI)
}

func (p *testableOpenAIProvider) Models() []string {
	return openAIModels
}

func (p *testableOpenAIProvider) DefaultModel() string {
	return OpenAIDefaultModel
}

func (p *testableOpenAIProvider) Chat(ctx context.Context, messages []Message, opts ChatOptions) (*StreamReader, error) {
	model := opts.Model
	if model == "" {
		model = p.model
	}

	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = OpenAIDefaultMaxTokens
	}

	req := openai.ChatCompletionRequest{
		Model:       model,
		Messages:    convertToOpenAIMessages(messages),
		MaxTokens:   maxTokens,
		Temperature: float32(opts.Temperature),
		Stream:      true,
	}

	stream, err := p.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create stream: %w", err)
	}

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
				reader.Send(StreamChunk{Error: err})
				return
			}

			if len(response.Choices) > 0 {
				delta := response.Choices[0].Delta
				if delta.Content != "" {
					reader.Send(StreamChunk{Text: delta.Content})
				}

				if response.Choices[0].FinishReason != "" {
					reader.Send(StreamChunk{Done: true})
					return
				}
			}
		}
	}()

	return reader, nil
}

func (p *testableOpenAIProvider) ChatSync(ctx context.Context, messages []Message, opts ChatOptions) (*Response, error) {
	model := opts.Model
	if model == "" {
		model = p.model
	}

	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = OpenAIDefaultMaxTokens
	}

	req := openai.ChatCompletionRequest{
		Model:       model,
		Messages:    convertToOpenAIMessages(messages),
		MaxTokens:   maxTokens,
		Temperature: float32(opts.Temperature),
	}

	resp, err := p.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, errors.New("no choices in response")
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
