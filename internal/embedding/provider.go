package embedding

import "context"

// Provider defines the interface for generating text embeddings.
type Provider interface {
	// Embed generates an embedding for a single text string.
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch generates embeddings for multiple text strings.
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}
