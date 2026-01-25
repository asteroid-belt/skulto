package llm

import (
	"context"
	"errors"
	"testing"

	openai "github.com/sashabaranov/go-openai"
)

// mockOpenAIClient implements OpenAIClientInterface for testing.
type mockOpenAIClient struct {
	completionResponse openai.ChatCompletionResponse
	completionErr      error
	streamCreateErr    error
}

func (m *mockOpenAIClient) CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	if m.completionErr != nil {
		return openai.ChatCompletionResponse{}, m.completionErr
	}
	return m.completionResponse, nil
}

func (m *mockOpenAIClient) CreateChatCompletionStream(ctx context.Context, req openai.ChatCompletionRequest) (*openai.ChatCompletionStream, error) {
	if m.streamCreateErr != nil {
		return nil, m.streamCreateErr
	}
	// We can't easily mock the real stream, so return an error for unit tests
	return nil, errors.New("stream mocking not supported in unit tests")
}

func TestNewOpenAIProvider_ValidAPIKey(t *testing.T) {
	provider, err := newOpenAIProvider("test-api-key", "")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if provider == nil {
		t.Fatal("expected provider to be non-nil")
	}
	if provider.model != OpenAIDefaultModel {
		t.Errorf("expected default model %q, got %q", OpenAIDefaultModel, provider.model)
	}
}

func TestNewOpenAIProvider_EmptyAPIKey(t *testing.T) {
	_, err := newOpenAIProvider("", "")
	if err == nil {
		t.Fatal("expected error for empty API key")
	}
	if err.Error() != "OpenAI API key is required" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestNewOpenAIProvider_CustomModel(t *testing.T) {
	provider, err := newOpenAIProvider("test-api-key", OpenAIModelGPT4o)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if provider.model != OpenAIModelGPT4o {
		t.Errorf("expected model %q, got %q", OpenAIModelGPT4o, provider.model)
	}
}

func TestNewOpenAIProvider_InvalidModel(t *testing.T) {
	_, err := newOpenAIProvider("test-api-key", "invalid-model")
	if err == nil {
		t.Fatal("expected error for invalid model")
	}
}

func TestOpenAIProvider_Name(t *testing.T) {
	provider, _ := newOpenAIProvider("test-api-key", "")
	if provider.Name() != "openai" {
		t.Errorf("expected name 'openai', got %q", provider.Name())
	}
}

func TestOpenAIProvider_Models(t *testing.T) {
	provider, _ := newOpenAIProvider("test-api-key", "")
	models := provider.Models()

	expected := []string{OpenAIModelGPT4oMini, OpenAIModelGPT4o, OpenAIModelGPT4Turbo}
	if len(models) != len(expected) {
		t.Errorf("expected %d models, got %d", len(expected), len(models))
	}

	for i, m := range models {
		if m != expected[i] {
			t.Errorf("expected model[%d] = %q, got %q", i, expected[i], m)
		}
	}
}

func TestOpenAIProvider_DefaultModel(t *testing.T) {
	provider, _ := newOpenAIProvider("test-api-key", "")
	if provider.DefaultModel() != OpenAIDefaultModel {
		t.Errorf("expected default model %q, got %q", OpenAIDefaultModel, provider.DefaultModel())
	}
}

func TestConvertToOpenAIMessages(t *testing.T) {
	messages := []Message{
		NewSystemMessage("You are a helpful assistant."),
		NewUserMessage("Hello!"),
		NewAssistantMessage("Hi there!"),
	}

	openaiMsgs := convertToOpenAIMessages(messages)

	if len(openaiMsgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(openaiMsgs))
	}

	tests := []struct {
		index   int
		role    string
		content string
	}{
		{0, "system", "You are a helpful assistant."},
		{1, "user", "Hello!"},
		{2, "assistant", "Hi there!"},
	}

	for _, tc := range tests {
		if openaiMsgs[tc.index].Role != tc.role {
			t.Errorf("message[%d] role: expected %q, got %q", tc.index, tc.role, openaiMsgs[tc.index].Role)
		}
		if openaiMsgs[tc.index].Content != tc.content {
			t.Errorf("message[%d] content: expected %q, got %q", tc.index, tc.content, openaiMsgs[tc.index].Content)
		}
	}
}

func TestOpenAIProvider_ChatSync_WithMock(t *testing.T) {
	mockClient := &mockOpenAIClient{
		completionResponse: openai.ChatCompletionResponse{
			Model: OpenAIModelGPT4oMini,
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Role:    "assistant",
						Content: "Hello! How can I help you?",
					},
					FinishReason: openai.FinishReasonStop,
				},
			},
			Usage: openai.Usage{
				PromptTokens:     10,
				CompletionTokens: 8,
				TotalTokens:      18,
			},
		},
	}

	provider := NewOpenAIProviderWithClient(mockClient, "")

	messages := []Message{
		NewUserMessage("Hello!"),
	}

	resp, err := provider.ChatSync(context.Background(), messages, ChatOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "Hello! How can I help you?" {
		t.Errorf("unexpected content: %q", resp.Content)
	}
	if resp.Model != OpenAIModelGPT4oMini {
		t.Errorf("unexpected model: %q", resp.Model)
	}
	if resp.FinishReason != string(openai.FinishReasonStop) {
		t.Errorf("unexpected finish reason: %q", resp.FinishReason)
	}
	if resp.Usage.TotalTokens != 18 {
		t.Errorf("unexpected total tokens: %d", resp.Usage.TotalTokens)
	}
}

func TestOpenAIProvider_ChatSync_Error(t *testing.T) {
	mockClient := &mockOpenAIClient{
		completionErr: errors.New("API error"),
	}

	provider := NewOpenAIProviderWithClient(mockClient, "")

	messages := []Message{
		NewUserMessage("Hello!"),
	}

	_, err := provider.ChatSync(context.Background(), messages, ChatOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOpenAIProvider_ChatSync_NoChoices(t *testing.T) {
	mockClient := &mockOpenAIClient{
		completionResponse: openai.ChatCompletionResponse{
			Model:   OpenAIModelGPT4oMini,
			Choices: []openai.ChatCompletionChoice{},
		},
	}

	provider := NewOpenAIProviderWithClient(mockClient, "")

	messages := []Message{
		NewUserMessage("Hello!"),
	}

	_, err := provider.ChatSync(context.Background(), messages, ChatOptions{})
	if err == nil {
		t.Fatal("expected error for no choices")
	}
	if err.Error() != "no choices in response" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestOpenAIProvider_ChatSync_WithModelOverride(t *testing.T) {
	var capturedRequest openai.ChatCompletionRequest

	mockClient := &customMockClient{
		onCompletion: func(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
			capturedRequest = req
			return openai.ChatCompletionResponse{
				Model: req.Model,
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "Response",
						},
						FinishReason: openai.FinishReasonStop,
					},
				},
			}, nil
		},
	}

	provider := NewOpenAIProviderWithClient(mockClient, "")

	messages := []Message{
		NewUserMessage("Hello!"),
	}

	opts := ChatOptions{
		Model:       OpenAIModelGPT4o,
		MaxTokens:   1000,
		Temperature: 0.5,
	}

	_, err := provider.ChatSync(context.Background(), messages, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedRequest.Model != OpenAIModelGPT4o {
		t.Errorf("expected model %q, got %q", OpenAIModelGPT4o, capturedRequest.Model)
	}
	if capturedRequest.MaxTokens != 1000 {
		t.Errorf("expected max tokens 1000, got %d", capturedRequest.MaxTokens)
	}
	if capturedRequest.Temperature != 0.5 {
		t.Errorf("expected temperature 0.5, got %f", capturedRequest.Temperature)
	}
}

// customMockClient allows custom behavior for testing.
type customMockClient struct {
	onCompletion func(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
	onStream     func(ctx context.Context, req openai.ChatCompletionRequest) (*openai.ChatCompletionStream, error)
}

func (m *customMockClient) CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	if m.onCompletion != nil {
		return m.onCompletion(ctx, req)
	}
	return openai.ChatCompletionResponse{}, nil
}

func (m *customMockClient) CreateChatCompletionStream(ctx context.Context, req openai.ChatCompletionRequest) (*openai.ChatCompletionStream, error) {
	if m.onStream != nil {
		return m.onStream(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func TestIsValidOpenAIModel(t *testing.T) {
	tests := []struct {
		model string
		valid bool
	}{
		{OpenAIModelGPT4oMini, true},
		{OpenAIModelGPT4o, true},
		{OpenAIModelGPT4Turbo, true},
		{"gpt-3.5-turbo", false},
		{"invalid", false},
		{"", false},
	}

	for _, tc := range tests {
		result := isValidOpenAIModel(tc.model)
		if result != tc.valid {
			t.Errorf("isValidOpenAIModel(%q) = %v, want %v", tc.model, result, tc.valid)
		}
	}
}

func TestTestableOpenAIProvider_Name(t *testing.T) {
	provider := NewOpenAIProviderWithClient(&mockOpenAIClient{}, "")
	if provider.Name() != "openai" {
		t.Errorf("expected name 'openai', got %q", provider.Name())
	}
}

func TestTestableOpenAIProvider_Models(t *testing.T) {
	provider := NewOpenAIProviderWithClient(&mockOpenAIClient{}, "")
	models := provider.Models()
	if len(models) != 3 {
		t.Errorf("expected 3 models, got %d", len(models))
	}
}

func TestTestableOpenAIProvider_DefaultModel(t *testing.T) {
	provider := NewOpenAIProviderWithClient(&mockOpenAIClient{}, "")
	if provider.DefaultModel() != OpenAIDefaultModel {
		t.Errorf("expected default model %q, got %q", OpenAIDefaultModel, provider.DefaultModel())
	}
}

func TestTestableOpenAIProvider_CustomModel(t *testing.T) {
	// The testable provider stores the model internally - verify it through a request
	mockClient := &customMockClient{
		onCompletion: func(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
			if req.Model != OpenAIModelGPT4Turbo {
				t.Errorf("expected model %q, got %q", OpenAIModelGPT4Turbo, req.Model)
			}
			return openai.ChatCompletionResponse{
				Model: req.Model,
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "Test",
						},
						FinishReason: openai.FinishReasonStop,
					},
				},
			}, nil
		},
	}

	provider2 := NewOpenAIProviderWithClient(mockClient, OpenAIModelGPT4Turbo)
	_, err := provider2.ChatSync(context.Background(), []Message{NewUserMessage("test")}, ChatOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenAIProvider_Chat_StreamError(t *testing.T) {
	mockClient := &customMockClient{
		onStream: func(ctx context.Context, req openai.ChatCompletionRequest) (*openai.ChatCompletionStream, error) {
			return nil, errors.New("stream creation failed")
		},
	}

	provider := NewOpenAIProviderWithClient(mockClient, "")

	messages := []Message{
		NewUserMessage("Hello!"),
	}

	_, err := provider.Chat(context.Background(), messages, ChatOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOpenAIProvider_DefaultMaxTokens(t *testing.T) {
	var capturedRequest openai.ChatCompletionRequest

	mockClient := &customMockClient{
		onCompletion: func(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
			capturedRequest = req
			return openai.ChatCompletionResponse{
				Model: req.Model,
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "Response",
						},
						FinishReason: openai.FinishReasonStop,
					},
				},
			}, nil
		},
	}

	provider := NewOpenAIProviderWithClient(mockClient, "")

	_, err := provider.ChatSync(context.Background(), []Message{NewUserMessage("test")}, ChatOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedRequest.MaxTokens != OpenAIDefaultMaxTokens {
		t.Errorf("expected default max tokens %d, got %d", OpenAIDefaultMaxTokens, capturedRequest.MaxTokens)
	}
}
