package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSkillURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		wantSlug string
		wantMeta bool
		wantErr  bool
	}{
		{
			name:     "valid content URI",
			uri:      "skulto://skill/my-skill",
			wantSlug: "my-skill",
			wantMeta: false,
			wantErr:  false,
		},
		{
			name:     "valid metadata URI",
			uri:      "skulto://skill/my-skill/metadata",
			wantSlug: "my-skill",
			wantMeta: true,
			wantErr:  false,
		},
		{
			name:     "slug with dashes",
			uri:      "skulto://skill/react-hooks-best-practices",
			wantSlug: "react-hooks-best-practices",
			wantMeta: false,
			wantErr:  false,
		},
		{
			name:    "invalid scheme",
			uri:     "http://skill/my-skill",
			wantErr: true,
		},
		{
			name:    "empty slug",
			uri:     "skulto://skill/",
			wantErr: true,
		},
		{
			name:    "wrong path prefix",
			uri:     "skulto://tags/my-tag",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slug, isMeta, err := parseSkillURI(tt.uri)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantSlug, slug)
			assert.Equal(t, tt.wantMeta, isMeta)
		})
	}
}

func TestHandleSkillContentResource(t *testing.T) {
	database := setupTestDB(t)
	cfg := &config.Config{}
	favStore := setupTestFavorites(t)
	server := NewServer(database, cfg, favStore, nil)
	seedTestSkills(t, server)

	ctx := context.Background()

	t.Run("returns skill content as markdown", func(t *testing.T) {
		req := mcp.ReadResourceRequest{}
		req.Params.URI = "skulto://skill/test-react-hooks"

		contents, err := server.handleSkillContentResource(ctx, req)
		require.NoError(t, err)
		require.Len(t, contents, 1)

		textContent, ok := contents[0].(mcp.TextResourceContents)
		require.True(t, ok)

		assert.Equal(t, "text/markdown", textContent.MIMEType)
		assert.Contains(t, textContent.Text, "# React Hooks")
	})

	t.Run("returns error for nonexistent skill", func(t *testing.T) {
		req := mcp.ReadResourceRequest{}
		req.Params.URI = "skulto://skill/nonexistent"

		_, err := server.handleSkillContentResource(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "skill not found")
	})

	t.Run("returns error for invalid URI", func(t *testing.T) {
		req := mcp.ReadResourceRequest{}
		req.Params.URI = "http://invalid/uri"

		_, err := server.handleSkillContentResource(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid URI scheme")
	})
}

func TestHandleSkillMetadataResource(t *testing.T) {
	database := setupTestDB(t)
	cfg := &config.Config{}
	favStore := setupTestFavorites(t)
	server := NewServer(database, cfg, favStore, nil)
	seedTestSkills(t, server)

	ctx := context.Background()

	t.Run("returns skill metadata as JSON", func(t *testing.T) {
		req := mcp.ReadResourceRequest{}
		req.Params.URI = "skulto://skill/test-react-hooks/metadata"

		contents, err := server.handleSkillMetadataResource(ctx, req)
		require.NoError(t, err)
		require.Len(t, contents, 1)

		textContent, ok := contents[0].(mcp.TextResourceContents)
		require.True(t, ok)

		assert.Equal(t, "application/json", textContent.MIMEType)

		var skill SkillResponse
		err = json.Unmarshal([]byte(textContent.Text), &skill)
		require.NoError(t, err)

		assert.Equal(t, "test-react-hooks", skill.Slug)
		assert.Equal(t, "React Hooks Best Practices", skill.Title)
		assert.Empty(t, skill.Content) // Content should not be included in metadata
	})

	t.Run("returns error for nonexistent skill", func(t *testing.T) {
		req := mcp.ReadResourceRequest{}
		req.Params.URI = "skulto://skill/nonexistent/metadata"

		_, err := server.handleSkillMetadataResource(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "skill not found")
	})
}
