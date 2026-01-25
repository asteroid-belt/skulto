package llm

import (
	"context"
	"errors"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/packages/ssestream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAnthropicClient implements AnthropicClientInterface for testing.
type mockAnthropicClient struct {
	messageResponse *anthropic.Message
	messageErr      error
	capturedParams  anthropic.MessageNewParams
}

func (m *mockAnthropicClient) CreateMessage(ctx context.Context, params anthropic.MessageNewParams) (*anthropic.Message, error) {
	m.capturedParams = params
	if m.messageErr != nil {
		return nil, m.messageErr
	}
	return m.messageResponse, nil
}

func (m *mockAnthropicClient) CreateMessageStream(ctx context.Context, params anthropic.MessageNewParams) *ssestream.Stream[anthropic.MessageStreamEventUnion] {
	// Return nil for stream - streaming tests will need a different approach
	return nil
}

// TestNewAnthropicProviderImpl_ValidAPIKey tests creating a provider with a valid API key.
func TestNewAnthropicProviderImpl_ValidAPIKey(t *testing.T) {
	provider, err := NewAnthropicProviderImpl("test-api-key", "")
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, DefaultAnthropicModel, provider.model)
}

// TestNewAnthropicProviderImpl_EmptyAPIKey tests creating a provider with an empty API key.
func TestNewAnthropicProviderImpl_EmptyAPIKey(t *testing.T) {
	_, err := NewAnthropicProviderImpl("", "")
	require.Error(t, err)
	assert.Equal(t, "API key is required", err.Error())
}

// TestNewAnthropicProviderImpl_CustomModel tests creating a provider with a custom model.
func TestNewAnthropicProviderImpl_CustomModel(t *testing.T) {
	provider, err := NewAnthropicProviderImpl("test-api-key", "claude-3-5-sonnet-20241022")
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, "claude-3-5-sonnet-20241022", provider.model)
}

// TestNewAnthropicProviderImpl_InvalidModel tests creating a provider with an invalid model.
func TestNewAnthropicProviderImpl_InvalidModel(t *testing.T) {
	_, err := NewAnthropicProviderImpl("test-api-key", "invalid-model")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid Anthropic model")
}

// TestAnthropicProvider_Name tests the Name method.
func TestAnthropicProvider_Name(t *testing.T) {
	provider := NewAnthropicProviderWithClient(&mockAnthropicClient{}, "")
	assert.Equal(t, "anthropic", provider.Name())
}

// TestAnthropicProvider_Models tests the Models method.
func TestAnthropicProvider_Models(t *testing.T) {
	provider := NewAnthropicProviderWithClient(&mockAnthropicClient{}, "")
	models := provider.Models()

	assert.Equal(t, AnthropicModels, models)
	assert.Contains(t, models, "claude-3-haiku-20240307")
	assert.Contains(t, models, "claude-3-5-sonnet-20241022")
	assert.Contains(t, models, "claude-3-opus-20240229")
}

// TestAnthropicProvider_DefaultModel tests the DefaultModel method.
func TestAnthropicProvider_DefaultModel(t *testing.T) {
	provider := NewAnthropicProviderWithClient(&mockAnthropicClient{}, "")
	assert.Equal(t, DefaultAnthropicModel, provider.DefaultModel())
}

// TestAnthropicProvider_ConvertMessages tests message conversion.
func TestAnthropicProvider_ConvertMessages(t *testing.T) {
	provider := NewAnthropicProviderWithClient(&mockAnthropicClient{}, "")

	tests := []struct {
		name                 string
		messages             []Message
		expectedCount        int
		expectedSystemPrompt string
	}{
		{
			name: "single user message",
			messages: []Message{
				NewUserMessage("Hello!"),
			},
			expectedCount:        1,
			expectedSystemPrompt: "",
		},
		{
			name: "system and user message",
			messages: []Message{
				NewSystemMessage("You are a helpful assistant."),
				NewUserMessage("Hello!"),
			},
			expectedCount:        1, // System is separate
			expectedSystemPrompt: "You are a helpful assistant.",
		},
		{
			name: "full conversation",
			messages: []Message{
				NewSystemMessage("You are helpful."),
				NewUserMessage("Hello!"),
				NewAssistantMessage("Hi there!"),
				NewUserMessage("How are you?"),
			},
			expectedCount:        3, // User, Assistant, User (system is separate)
			expectedSystemPrompt: "You are helpful.",
		},
		{
			name: "no system message",
			messages: []Message{
				NewUserMessage("Hello!"),
				NewAssistantMessage("Hi!"),
			},
			expectedCount:        2,
			expectedSystemPrompt: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			anthropicMsgs, systemPrompt := provider.convertMessages(tt.messages)
			assert.Len(t, anthropicMsgs, tt.expectedCount)
			assert.Equal(t, tt.expectedSystemPrompt, systemPrompt)
		})
	}
}

// TestAnthropicProvider_ChatSync_Success tests successful synchronous chat.
func TestAnthropicProvider_ChatSync_Success(t *testing.T) {
	mockClient := &mockAnthropicClient{
		messageResponse: &anthropic.Message{
			Model:      "claude-3-haiku-20240307",
			StopReason: "end_turn",
			Content: []anthropic.ContentBlockUnion{
				{
					Type: "text",
					Text: "Hello! How can I help you?",
				},
			},
			Usage: anthropic.Usage{
				InputTokens:  10,
				OutputTokens: 8,
			},
		},
	}

	provider := NewAnthropicProviderWithClient(mockClient, "")

	messages := []Message{
		NewUserMessage("Hello!"),
	}

	resp, err := provider.ChatSync(context.Background(), messages, ChatOptions{})
	require.NoError(t, err)
	assert.Equal(t, "Hello! How can I help you?", resp.Content)
	assert.Equal(t, "claude-3-haiku-20240307", resp.Model)
	assert.Equal(t, "end_turn", resp.FinishReason)
	assert.Equal(t, 10, resp.Usage.PromptTokens)
	assert.Equal(t, 8, resp.Usage.CompletionTokens)
	assert.Equal(t, 18, resp.Usage.TotalTokens)
}

// TestAnthropicProvider_ChatSync_Error tests error handling in synchronous chat.
func TestAnthropicProvider_ChatSync_Error(t *testing.T) {
	mockClient := &mockAnthropicClient{
		messageErr: errors.New("API error"),
	}

	provider := NewAnthropicProviderWithClient(mockClient, "")

	messages := []Message{
		NewUserMessage("Hello!"),
	}

	_, err := provider.ChatSync(context.Background(), messages, ChatOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "anthropic chat")
}

// TestAnthropicProvider_ChatSync_WithSystemMessage tests chat with system message.
func TestAnthropicProvider_ChatSync_WithSystemMessage(t *testing.T) {
	mockClient := &mockAnthropicClient{
		messageResponse: &anthropic.Message{
			Model:      "claude-3-haiku-20240307",
			StopReason: "end_turn",
			Content: []anthropic.ContentBlockUnion{
				{
					Type: "text",
					Text: "Response",
				},
			},
			Usage: anthropic.Usage{},
		},
	}

	provider := NewAnthropicProviderWithClient(mockClient, "")

	messages := []Message{
		NewSystemMessage("You are a helpful assistant."),
		NewUserMessage("Hello!"),
	}

	_, err := provider.ChatSync(context.Background(), messages, ChatOptions{})
	require.NoError(t, err)

	// Verify system was set in params
	assert.Len(t, mockClient.capturedParams.System, 1)
	assert.Equal(t, "You are a helpful assistant.", mockClient.capturedParams.System[0].Text)
}

// TestAnthropicProvider_ChatSync_WithModelOverride tests model override in options.
func TestAnthropicProvider_ChatSync_WithModelOverride(t *testing.T) {
	mockClient := &mockAnthropicClient{
		messageResponse: &anthropic.Message{
			Model:      "claude-3-5-sonnet-20241022",
			StopReason: "end_turn",
			Content: []anthropic.ContentBlockUnion{
				{
					Type: "text",
					Text: "Response",
				},
			},
			Usage: anthropic.Usage{},
		},
	}

	provider := NewAnthropicProviderWithClient(mockClient, "")

	messages := []Message{
		NewUserMessage("Hello!"),
	}

	opts := ChatOptions{
		Model: "claude-3-5-sonnet-20241022",
	}

	_, err := provider.ChatSync(context.Background(), messages, opts)
	require.NoError(t, err)

	// Verify model was set correctly
	assert.Equal(t, anthropic.Model("claude-3-5-sonnet-20241022"), mockClient.capturedParams.Model)
}

// TestAnthropicProvider_ChatSync_WithMaxTokens tests max tokens in options.
func TestAnthropicProvider_ChatSync_WithMaxTokens(t *testing.T) {
	mockClient := &mockAnthropicClient{
		messageResponse: &anthropic.Message{
			Model:      "claude-3-haiku-20240307",
			StopReason: "end_turn",
			Content: []anthropic.ContentBlockUnion{
				{
					Type: "text",
					Text: "Response",
				},
			},
			Usage: anthropic.Usage{},
		},
	}

	provider := NewAnthropicProviderWithClient(mockClient, "")

	messages := []Message{
		NewUserMessage("Hello!"),
	}

	opts := ChatOptions{
		MaxTokens: 1000,
	}

	_, err := provider.ChatSync(context.Background(), messages, opts)
	require.NoError(t, err)

	// Verify max tokens was set
	assert.Equal(t, int64(1000), mockClient.capturedParams.MaxTokens)
}

// TestAnthropicProvider_ChatSync_DefaultMaxTokens tests default max tokens.
func TestAnthropicProvider_ChatSync_DefaultMaxTokens(t *testing.T) {
	mockClient := &mockAnthropicClient{
		messageResponse: &anthropic.Message{
			Model:      "claude-3-haiku-20240307",
			StopReason: "end_turn",
			Content: []anthropic.ContentBlockUnion{
				{
					Type: "text",
					Text: "Response",
				},
			},
			Usage: anthropic.Usage{},
		},
	}

	provider := NewAnthropicProviderWithClient(mockClient, "")

	messages := []Message{
		NewUserMessage("Hello!"),
	}

	_, err := provider.ChatSync(context.Background(), messages, ChatOptions{})
	require.NoError(t, err)

	// Verify default max tokens (4096)
	assert.Equal(t, int64(4096), mockClient.capturedParams.MaxTokens)
}

// TestIsValidAnthropicModel tests model validation.
func TestIsValidAnthropicModel(t *testing.T) {
	tests := []struct {
		model string
		valid bool
	}{
		{"claude-3-haiku-20240307", true},
		{"claude-3-5-sonnet-20241022", true},
		{"claude-3-5-haiku-20241022", true},
		{"claude-3-opus-20240229", true},
		{"invalid-model", false},
		{"gpt-4", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			result := isValidAnthropicModel(tt.model)
			assert.Equal(t, tt.valid, result, "isValidAnthropicModel(%q)", tt.model)
		})
	}
}

// TestNewAnthropicProviderWithClient tests creating provider with custom client.
func TestNewAnthropicProviderWithClient(t *testing.T) {
	mockClient := &mockAnthropicClient{}

	t.Run("with default model", func(t *testing.T) {
		provider := NewAnthropicProviderWithClient(mockClient, "")
		assert.Equal(t, DefaultAnthropicModel, provider.model)
	})

	t.Run("with custom model", func(t *testing.T) {
		provider := NewAnthropicProviderWithClient(mockClient, "claude-3-opus-20240229")
		assert.Equal(t, "claude-3-opus-20240229", provider.model)
	})
}

// TestAnthropicProvider_ChatSync_EmptyResponse tests handling empty response.
func TestAnthropicProvider_ChatSync_EmptyResponse(t *testing.T) {
	mockClient := &mockAnthropicClient{
		messageResponse: &anthropic.Message{
			Model:      "claude-3-haiku-20240307",
			StopReason: "end_turn",
			Content:    []anthropic.ContentBlockUnion{},
			Usage:      anthropic.Usage{},
		},
	}

	provider := NewAnthropicProviderWithClient(mockClient, "")

	messages := []Message{
		NewUserMessage("Hello!"),
	}

	resp, err := provider.ChatSync(context.Background(), messages, ChatOptions{})
	require.NoError(t, err)
	assert.Equal(t, "", resp.Content)
}

// TestAnthropicProvider_ConvertMessages_Empty tests converting empty messages.
func TestAnthropicProvider_ConvertMessages_Empty(t *testing.T) {
	provider := NewAnthropicProviderWithClient(&mockAnthropicClient{}, "")

	anthropicMsgs, systemPrompt := provider.convertMessages([]Message{})
	assert.Len(t, anthropicMsgs, 0)
	assert.Empty(t, systemPrompt)
}

// TestAnthropicProvider_ImplementsInterface verifies AnthropicProvider implements Provider.
func TestAnthropicProvider_ImplementsInterface(t *testing.T) {
	mockClient := &mockAnthropicClient{}
	provider := NewAnthropicProviderWithClient(mockClient, "")

	// This will fail to compile if AnthropicProvider doesn't implement Provider
	var _ Provider = provider
}
