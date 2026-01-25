package components

import (
	"testing"

	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/search"
	"github.com/stretchr/testify/assert"
)

func TestUnifiedResultList_NewList(t *testing.T) {
	list := NewUnifiedResultList()
	assert.Empty(t, list.Items)
	assert.Equal(t, -1, list.Selected)
}

func TestUnifiedResultList_SetItems(t *testing.T) {
	list := NewUnifiedResultList()

	items := []UnifiedResultItem{
		{Skill: models.Skill{ID: "1", Title: "Skill 1"}, MatchType: MatchTypeName},
		{Skill: models.Skill{ID: "2", Title: "Skill 2"}, MatchType: MatchTypeContent},
	}
	list.SetItems(items)

	assert.Len(t, list.Items, 2)
	assert.Equal(t, 0, list.Selected) // Auto-selects first
}

func TestUnifiedResultList_Navigation(t *testing.T) {
	list := NewUnifiedResultList()
	list.SetItems([]UnifiedResultItem{
		{Skill: models.Skill{ID: "1"}, MatchType: MatchTypeName},
		{Skill: models.Skill{ID: "2"}, MatchType: MatchTypeName},
		{Skill: models.Skill{ID: "3"}, MatchType: MatchTypeContent},
	})

	assert.Equal(t, 0, list.Selected)

	assert.True(t, list.MoveDown())
	assert.Equal(t, 1, list.Selected)

	assert.True(t, list.MoveDown())
	assert.Equal(t, 2, list.Selected)

	assert.False(t, list.MoveDown()) // At end
	assert.Equal(t, 2, list.Selected)

	assert.True(t, list.MoveUp())
	assert.Equal(t, 1, list.Selected)
}

func TestUnifiedResultList_ToggleExpand(t *testing.T) {
	list := NewUnifiedResultList()
	list.SetItems([]UnifiedResultItem{
		{Skill: models.Skill{ID: "1"}, MatchType: MatchTypeName},
		{
			Skill:     models.Skill{ID: "2"},
			MatchType: MatchTypeContent,
			Snippets:  []search.Snippet{{Text: "test"}},
		},
	})

	// Name match - no expansion
	list.Selected = 0
	assert.False(t, list.ToggleExpand())

	// Content match with snippets - can expand
	list.Selected = 1
	assert.False(t, list.Items[1].Expanded)
	assert.True(t, list.ToggleExpand())
	assert.True(t, list.Items[1].Expanded)

	// Toggle again to collapse
	assert.True(t, list.ToggleExpand())
	assert.False(t, list.Items[1].Expanded)
}

func TestUnifiedResultList_View(t *testing.T) {
	list := NewUnifiedResultList()
	list.SetSize(80, 30)
	list.SetItems([]UnifiedResultItem{
		{
			Skill:     models.Skill{ID: "1", Title: "Test Skill", Description: "A test description"},
			MatchType: MatchTypeName,
		},
		{
			Skill:     models.Skill{ID: "2", Title: "Content Match", Description: "Another desc"},
			MatchType: MatchTypeContent,
			Snippets:  []search.Snippet{{Text: "matching text"}},
		},
	})

	view := list.View()

	assert.Contains(t, view, "[name]")
	assert.Contains(t, view, "[content]")
	assert.Contains(t, view, "Test Skill")
	assert.Contains(t, view, "Content Match")
	assert.Contains(t, view, "1 matching snippet")
}

func TestUnifiedResultList_Scrolling(t *testing.T) {
	list := NewUnifiedResultList()
	list.SetSize(80, 9) // 9 lines = 3 items visible

	// Create 10 items
	items := make([]UnifiedResultItem, 10)
	for i := range 10 {
		items[i] = UnifiedResultItem{
			Skill:     models.Skill{ID: string(rune('0' + i)), Title: "Skill"},
			MatchType: MatchTypeName,
		}
	}
	list.SetItems(items)

	// Initially at top
	assert.Equal(t, 0, list.scrollOffset)

	// Move to item 5 (beyond viewport)
	for range 5 {
		list.MoveDown()
	}
	assert.Equal(t, 5, list.Selected)
	assert.True(t, list.scrollOffset > 0, "should have scrolled")

	// View should show scroll indicators
	view := list.View()
	assert.Contains(t, view, "more above")
	assert.Contains(t, view, "more below")
}

func TestUnifiedResultList_Clear(t *testing.T) {
	list := NewUnifiedResultList()
	list.SetItems([]UnifiedResultItem{
		{Skill: models.Skill{ID: "1"}, MatchType: MatchTypeName},
	})

	assert.Equal(t, 1, list.TotalCount())

	list.Clear()

	assert.Equal(t, 0, list.TotalCount())
	assert.Equal(t, -1, list.Selected)
}

func TestUnifiedResultList_GetSelectedItem(t *testing.T) {
	list := NewUnifiedResultList()

	// Empty list
	assert.Nil(t, list.GetSelectedItem())

	list.SetItems([]UnifiedResultItem{
		{Skill: models.Skill{ID: "1", Title: "First"}, MatchType: MatchTypeName},
		{Skill: models.Skill{ID: "2", Title: "Second"}, MatchType: MatchTypeContent},
	})

	// First item selected by default
	item := list.GetSelectedItem()
	assert.NotNil(t, item)
	assert.Equal(t, "First", item.Skill.Title)

	// Move to second
	list.MoveDown()
	item = list.GetSelectedItem()
	assert.NotNil(t, item)
	assert.Equal(t, "Second", item.Skill.Title)
}

func TestUnifiedResultList_ExpandedView(t *testing.T) {
	list := NewUnifiedResultList()
	list.SetSize(80, 30)
	list.SetItems([]UnifiedResultItem{
		{
			Skill:     models.Skill{ID: "1", Title: "Content Match"},
			MatchType: MatchTypeContent,
			Snippets: []search.Snippet{
				{Text: "first snippet"},
				{Text: "second snippet"},
			},
		},
	})

	// Initially collapsed
	view := list.View()
	assert.Contains(t, view, "2 matching snippets")
	assert.NotContains(t, view, "Matching context")

	// Expand
	list.ToggleExpand()
	view = list.View()
	assert.Contains(t, view, "Matching context")
	assert.Contains(t, view, "first snippet")
	assert.Contains(t, view, "second snippet")
}
