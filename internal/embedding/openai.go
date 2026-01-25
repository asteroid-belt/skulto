package embedding

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

// OpenAIProvider implements Provider using OpenAI API.
type OpenAIProvider struct {
	client *openai.Client
	model  openai.EmbeddingModel
}

// NewOpenAI creates a new OpenAI embedding provider.
func NewOpenAI(apiKey string, model string) *OpenAIProvider {
	if model == "" {
		model = string(openai.SmallEmbedding3)
	}
	return &OpenAIProvider{
		client: openai.NewClient(apiKey),
		model:  openai.EmbeddingModel(model),
	}
}

// Embed generates an embedding for a single text string.
func (p *OpenAIProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	resp, err := p.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: p.model,
	})
	if err != nil {
		return nil, fmt.Errorf("create embedding: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data returned")
	}

	return resp.Data[0].Embedding, nil
}

// EmbedBatch generates embeddings for multiple text strings.
func (p *OpenAIProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	resp, err := p.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: texts,
		Model: p.model,
	})
	if err != nil {
		return nil, fmt.Errorf("create embeddings: %w", err)
	}

	result := make([][]float32, len(texts))
	for i, data := range resp.Data {
		if i < len(result) {
			result[i] = data.Embedding
		}
	}

	return result, nil
}
