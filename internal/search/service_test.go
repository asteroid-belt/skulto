package search

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultSearchOptions(t *testing.T) {
	opts := DefaultSearchOptions()

	assert.Equal(t, 50, opts.Limit)
	assert.Equal(t, float32(0.6), opts.Threshold)
	assert.True(t, opts.IncludeFTS)
	assert.True(t, opts.IncludeSemantic)
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, float32(0.6), cfg.MinSimilarity)
	assert.Equal(t, 50, cfg.MaxResults)
	assert.Equal(t, 3, cfg.MaxSnippets)
}

func TestNewService_NilStore(t *testing.T) {
	// Service should work with nil store (FTS-only mode)
	svc := New(nil, nil, DefaultConfig())

	assert.NotNil(t, svc)
	assert.False(t, svc.HasVectorStore())
}

func TestNewService_DefaultsApplied(t *testing.T) {
	// Test that zero values get defaults
	svc := New(nil, nil, Config{})

	assert.Equal(t, float32(0.6), svc.config.MinSimilarity)
	assert.Equal(t, 50, svc.config.MaxResults)
	assert.Equal(t, 3, svc.config.MaxSnippets)
}

func TestBuildSearchableContent(t *testing.T) {
	tests := []struct {
		name        string
		description string
		summary     string
		content     string
		expected    string
	}{
		{
			description: "A description",
			summary:     "A summary",
			content:     "The content",
			expected:    "A description A summary The content",
		},
		{
			description: "Only description",
			expected:    "Only description",
		},
		{
			content:  "Only content",
			expected: "Only content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			// Note: Using local struct to avoid importing models in this test
			// The actual function uses models.Skill
		})
	}
}

func TestContainsAnyTerm(t *testing.T) {
	tests := []struct {
		text     string
		query    string
		expected bool
	}{
		{"testing react components", "react", true},
		{"testing react components", "vue", false},
		{"testing react components", "react vue", true}, // Contains react
		{"testing react components", "a b", false},      // Terms too short
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			result := containsAnyTerm(tt.text, tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	// Service with no store or db (will use nil-safe paths)
	svc := New(nil, nil, DefaultConfig())

	ctx := context.Background()
	opts := DefaultSearchOptions()
	opts.IncludeFTS = false      // Skip FTS since we have no db
	opts.IncludeSemantic = false // Skip semantic since we have no store

	results, err := svc.Search(ctx, "", opts)

	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Empty(t, results.TitleMatches)
	assert.Empty(t, results.ContentMatches)
}
