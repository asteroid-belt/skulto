package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: setupTestDB and setupTestFavorites are defined in server_test.go

func seedTestSkills(t *testing.T, s *Server) {
	t.Helper()

	skills := []models.Skill{
		{
			ID:          "test-skill-1",
			Slug:        "test-react-hooks",
			Title:       "React Hooks Best Practices",
			Description: "Learn React hooks patterns",
			Content:     "# React Hooks\n\nUse hooks for state management.",
			Summary:     "React hooks guide",
			Author:      "testauthor",
			Difficulty:  "intermediate",
		},
		{
			ID:          "test-skill-2",
			Slug:        "test-go-patterns",
			Title:       "Go Design Patterns",
			Description: "Common patterns in Go",
			Content:     "# Go Patterns\n\nInterface patterns.",
			Summary:     "Go patterns guide",
			Author:      "gopher",
			Difficulty:  "advanced",
		},
	}

	for _, skill := range skills {
		require.NoError(t, s.db.CreateSkill(&skill))
	}
}

func TestHandleSearch(t *testing.T) {
	database := setupTestDB(t)
	cfg := &config.Config{}
	favStore := setupTestFavorites(t)
	server := NewServer(database, cfg, favStore)
	seedTestSkills(t, server)

	ctx := context.Background()

	t.Run("search returns matching skills", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{
			"query": "react",
			"limit": float64(10),
		}

		result, err := server.handleSearch(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.False(t, result.IsError)

		var skills []SkillResponse
		textContent, ok := result.Content[0].(mcp.TextContent)
		require.True(t, ok)
		err = json.Unmarshal([]byte(textContent.Text), &skills)
		require.NoError(t, err)

		assert.Len(t, skills, 1)
		assert.Equal(t, "test-react-hooks", skills[0].Slug)
	})

	t.Run("search with no results returns empty array", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{
			"query": "nonexistent",
		}

		result, err := server.handleSearch(ctx, req)
		require.NoError(t, err)
		require.False(t, result.IsError)

		var skills []SkillResponse
		textContent, ok := result.Content[0].(mcp.TextContent)
		require.True(t, ok)
		err = json.Unmarshal([]byte(textContent.Text), &skills)
		require.NoError(t, err)

		assert.Len(t, skills, 0)
	})

	t.Run("search requires query parameter", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{}

		result, err := server.handleSearch(ctx, req)
		require.NoError(t, err)
		assert.True(t, result.IsError)
	})
}

func TestHandleGetSkill(t *testing.T) {
	database := setupTestDB(t)
	cfg := &config.Config{}
	favStore := setupTestFavorites(t)
	server := NewServer(database, cfg, favStore)
	seedTestSkills(t, server)

	ctx := context.Background()

	t.Run("get skill returns full content", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{
			"slug": "test-react-hooks",
		}

		result, err := server.handleGetSkill(ctx, req)
		require.NoError(t, err)
		require.False(t, result.IsError)

		var skill SkillResponse
		textContent, ok := result.Content[0].(mcp.TextContent)
		require.True(t, ok)
		err = json.Unmarshal([]byte(textContent.Text), &skill)
		require.NoError(t, err)

		assert.Equal(t, "test-react-hooks", skill.Slug)
		assert.Contains(t, skill.Content, "# React Hooks")
	})

	t.Run("get skill updates viewed_at", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{
			"slug": "test-go-patterns",
		}

		_, err := server.handleGetSkill(ctx, req)
		require.NoError(t, err)

		// Verify viewed_at was updated
		skill, err := server.db.GetSkillBySlug("test-go-patterns")
		require.NoError(t, err)
		assert.NotNil(t, skill.ViewedAt)
	})

	t.Run("get nonexistent skill returns error", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{
			"slug": "nonexistent",
		}

		result, err := server.handleGetSkill(ctx, req)
		require.NoError(t, err)
		assert.True(t, result.IsError)
	})
}

func TestHandleListSkills(t *testing.T) {
	database := setupTestDB(t)
	cfg := &config.Config{}
	favStore := setupTestFavorites(t)
	server := NewServer(database, cfg, favStore)
	seedTestSkills(t, server)

	ctx := context.Background()

	t.Run("list skills returns paginated results", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{
			"limit":  float64(1),
			"offset": float64(0),
		}

		result, err := server.handleListSkills(ctx, req)
		require.NoError(t, err)
		require.False(t, result.IsError)

		var skills []SkillResponse
		textContent, ok := result.Content[0].(mcp.TextContent)
		require.True(t, ok)
		err = json.Unmarshal([]byte(textContent.Text), &skills)
		require.NoError(t, err)

		assert.Len(t, skills, 1)
	})

	t.Run("list skills with offset skips results", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{
			"limit":  float64(10),
			"offset": float64(1),
		}

		result, err := server.handleListSkills(ctx, req)
		require.NoError(t, err)
		require.False(t, result.IsError)

		var skills []SkillResponse
		textContent, ok := result.Content[0].(mcp.TextContent)
		require.True(t, ok)
		err = json.Unmarshal([]byte(textContent.Text), &skills)
		require.NoError(t, err)

		assert.Len(t, skills, 1)
	})
}

func TestHandleBrowseTags(t *testing.T) {
	database := setupTestDB(t)
	cfg := &config.Config{}
	favStore := setupTestFavorites(t)
	server := NewServer(database, cfg, favStore)

	// Seed some tags
	tags := []models.Tag{
		{ID: "python", Name: "Python", Slug: "python", Category: "language", Count: 5},
		{ID: "react", Name: "React", Slug: "react", Category: "framework", Count: 3},
	}
	for _, tag := range tags {
		require.NoError(t, database.CreateTag(&tag))
	}

	ctx := context.Background()

	t.Run("browse tags returns all tags", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{}

		result, err := server.handleBrowseTags(ctx, req)
		require.NoError(t, err)
		require.False(t, result.IsError)

		var tagResults []TagResponse
		textContent, ok := result.Content[0].(mcp.TextContent)
		require.True(t, ok)
		err = json.Unmarshal([]byte(textContent.Text), &tagResults)
		require.NoError(t, err)

		// Should include seeded tags plus the "mine" tag
		assert.GreaterOrEqual(t, len(tagResults), 2)
	})

	t.Run("browse tags filters by category", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{
			"category": "language",
		}

		result, err := server.handleBrowseTags(ctx, req)
		require.NoError(t, err)
		require.False(t, result.IsError)

		var tagResults []TagResponse
		textContent, ok := result.Content[0].(mcp.TextContent)
		require.True(t, ok)
		err = json.Unmarshal([]byte(textContent.Text), &tagResults)
		require.NoError(t, err)

		assert.Len(t, tagResults, 1)
		assert.Equal(t, "python", tagResults[0].ID)
	})
}

func TestHandleGetStats(t *testing.T) {
	database := setupTestDB(t)
	cfg := &config.Config{}
	favStore := setupTestFavorites(t)
	server := NewServer(database, cfg, favStore)
	seedTestSkills(t, server)

	ctx := context.Background()

	t.Run("get stats returns database statistics", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{}

		result, err := server.handleGetStats(ctx, req)
		require.NoError(t, err)
		require.False(t, result.IsError)

		var stats StatsResponse
		textContent, ok := result.Content[0].(mcp.TextContent)
		require.True(t, ok)
		err = json.Unmarshal([]byte(textContent.Text), &stats)
		require.NoError(t, err)

		assert.Equal(t, int64(2), stats.TotalSkills)
	})
}

func TestHandleGetRecent(t *testing.T) {
	database := setupTestDB(t)
	cfg := &config.Config{}
	favStore := setupTestFavorites(t)
	server := NewServer(database, cfg, favStore)
	seedTestSkills(t, server)

	ctx := context.Background()

	// First, view a skill to set its viewed_at
	getReq := mcp.CallToolRequest{}
	getReq.Params.Arguments = map[string]any{"slug": "test-react-hooks"}
	_, _ = server.handleGetSkill(ctx, getReq)

	t.Run("get recent returns recently viewed skills", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{
			"limit": float64(10),
		}

		result, err := server.handleGetRecent(ctx, req)
		require.NoError(t, err)
		require.False(t, result.IsError)

		var skills []SkillResponse
		textContent, ok := result.Content[0].(mcp.TextContent)
		require.True(t, ok)
		err = json.Unmarshal([]byte(textContent.Text), &skills)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(skills), 1)
		assert.Equal(t, "test-react-hooks", skills[0].Slug)
	})
}

func TestHandleBookmark(t *testing.T) {
	database := setupTestDB(t)
	cfg := &config.Config{}
	favStore := setupTestFavorites(t)
	server := NewServer(database, cfg, favStore)
	seedTestSkills(t, server)

	ctx := context.Background()

	t.Run("bookmark add works", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{
			"slug":   "test-react-hooks",
			"action": "add",
		}

		result, err := server.handleBookmark(ctx, req)
		require.NoError(t, err)
		require.False(t, result.IsError)

		// Verify skill is now in favorites
		assert.True(t, favStore.IsFavorite("test-react-hooks"))
		assert.Equal(t, 1, favStore.Count())
	})

	t.Run("bookmark remove works", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{
			"slug":   "test-react-hooks",
			"action": "remove",
		}

		result, err := server.handleBookmark(ctx, req)
		require.NoError(t, err)
		require.False(t, result.IsError)

		// Verify skill is removed from favorites
		assert.False(t, favStore.IsFavorite("test-react-hooks"))
		assert.Equal(t, 0, favStore.Count())
	})

	t.Run("bookmark invalid action returns error", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{
			"slug":   "test-react-hooks",
			"action": "invalid",
		}

		result, err := server.handleBookmark(ctx, req)
		require.NoError(t, err)
		assert.True(t, result.IsError)
	})

	t.Run("bookmark nonexistent skill returns error", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{
			"slug":   "nonexistent-skill",
			"action": "add",
		}

		result, err := server.handleBookmark(ctx, req)
		require.NoError(t, err)
		assert.True(t, result.IsError)
	})
}

func TestHandleGetBookmarks(t *testing.T) {
	database := setupTestDB(t)
	cfg := &config.Config{}
	favStore := setupTestFavorites(t)
	server := NewServer(database, cfg, favStore)
	seedTestSkills(t, server)

	ctx := context.Background()

	// Add a bookmark via handler
	bookmarkReq := mcp.CallToolRequest{}
	bookmarkReq.Params.Arguments = map[string]any{
		"slug":   "test-react-hooks",
		"action": "add",
	}
	_, _ = server.handleBookmark(ctx, bookmarkReq)

	t.Run("get bookmarks returns bookmarked skills", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{
			"limit": float64(50),
		}

		result, err := server.handleGetBookmarks(ctx, req)
		require.NoError(t, err)
		require.False(t, result.IsError)

		var skills []SkillResponse
		textContent, ok := result.Content[0].(mcp.TextContent)
		require.True(t, ok)
		err = json.Unmarshal([]byte(textContent.Text), &skills)
		require.NoError(t, err)

		assert.Len(t, skills, 1)
		assert.Equal(t, "test-react-hooks", skills[0].Slug)
		assert.Equal(t, "React Hooks Best Practices", skills[0].Title)
	})

	t.Run("get bookmarks returns empty array when no favorites", func(t *testing.T) {
		// Create a new server with empty favorites
		emptyFavStore := setupTestFavorites(t)
		emptyServer := NewServer(database, cfg, emptyFavStore)

		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{}

		result, err := emptyServer.handleGetBookmarks(ctx, req)
		require.NoError(t, err)
		require.False(t, result.IsError)

		var skills []SkillResponse
		textContent, ok := result.Content[0].(mcp.TextContent)
		require.True(t, ok)
		err = json.Unmarshal([]byte(textContent.Text), &skills)
		require.NoError(t, err)

		assert.Len(t, skills, 0)
	})

	t.Run("get bookmarks handles deleted skill gracefully", func(t *testing.T) {
		// Add a favorite directly that doesn't exist in DB
		require.NoError(t, favStore.Add("deleted-skill"))

		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{}

		result, err := server.handleGetBookmarks(ctx, req)
		require.NoError(t, err)
		require.False(t, result.IsError)

		var skills []SkillResponse
		textContent, ok := result.Content[0].(mcp.TextContent)
		require.True(t, ok)
		err = json.Unmarshal([]byte(textContent.Text), &skills)
		require.NoError(t, err)

		// Should return both the valid skill and the deleted one with minimal info
		assert.Len(t, skills, 2)

		// Find the deleted skill response
		var deletedSkill *SkillResponse
		for i := range skills {
			if skills[i].Slug == "deleted-skill" {
				deletedSkill = &skills[i]
				break
			}
		}
		require.NotNil(t, deletedSkill)
		assert.Equal(t, "deleted-skill", deletedSkill.Slug)
		assert.Contains(t, deletedSkill.Description, "no longer in database")
	})
}

func TestHandleInstall(t *testing.T) {
	database := setupTestDB(t)
	cfg := &config.Config{}
	favStore := setupTestFavorites(t)
	server := NewServer(database, cfg, favStore)
	seedTestSkills(t, server)

	ctx := context.Background()

	t.Run("install requires slug parameter", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{}

		result, err := server.handleInstall(ctx, req)
		require.NoError(t, err)
		assert.True(t, result.IsError)
	})

	t.Run("install returns error for nonexistent skill", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{
			"slug": "nonexistent-skill",
		}

		result, err := server.handleInstall(ctx, req)
		require.NoError(t, err)
		assert.True(t, result.IsError)
	})

	t.Run("install returns error when skill has no source", func(t *testing.T) {
		// Skills from seedTestSkills have no source
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{
			"slug": "test-react-hooks",
		}

		result, err := server.handleInstall(ctx, req)
		require.NoError(t, err)
		assert.True(t, result.IsError)
		// Should fail because skill has no source repository
		textContent, ok := result.Content[0].(mcp.TextContent)
		require.True(t, ok)
		assert.Contains(t, textContent.Text, "cannot install skill without source")
	})
}

func TestHandleUninstall(t *testing.T) {
	database := setupTestDB(t)
	cfg := &config.Config{}
	favStore := setupTestFavorites(t)
	server := NewServer(database, cfg, favStore)
	seedTestSkills(t, server)

	ctx := context.Background()

	t.Run("uninstall requires slug parameter", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{}

		result, err := server.handleUninstall(ctx, req)
		require.NoError(t, err)
		assert.True(t, result.IsError)
	})

	t.Run("uninstall returns error for nonexistent skill", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{
			"slug": "nonexistent-skill",
		}

		result, err := server.handleUninstall(ctx, req)
		require.NoError(t, err)
		assert.True(t, result.IsError)
	})
}
