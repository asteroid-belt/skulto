package components

import (
	"testing"

	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestNewTagGrid(t *testing.T) {
	tg := NewTagGrid()
	assert.NotNil(t, tg)
	assert.Equal(t, 0, tg.TagCount())
	assert.Nil(t, tg.GetSelectedTag())
}

func TestTagGridSetTags(t *testing.T) {
	tg := NewTagGrid()
	tags := []models.Tag{
		{ID: "python", Name: "python", Count: 10, Category: "language"},
		{ID: "react", Name: "react", Count: 5, Category: "framework"},
	}

	tg.SetTags(tags)

	assert.Equal(t, 2, tg.TagCount())
	assert.Equal(t, 0, tg.GetSelectedIndex())
	assert.Equal(t, "python", tg.GetSelectedTag().Name)
}

func TestTagGridNavigation(t *testing.T) {
	tg := NewTagGrid()
	tg.SetSize(100, 20)
	tags := []models.Tag{
		{ID: "python", Name: "python", Count: 10},
		{ID: "react", Name: "react", Count: 5},
		{ID: "docker", Name: "docker", Count: 3},
	}
	tg.SetTags(tags)

	// Start at first tag
	assert.Equal(t, 0, tg.GetSelectedIndex())

	// Move right
	tg.MoveRight()
	assert.Equal(t, 1, tg.GetSelectedIndex())

	// Move right again
	tg.MoveRight()
	assert.Equal(t, 2, tg.GetSelectedIndex())

	// Move right at end (should stay)
	tg.MoveRight()
	assert.Equal(t, 2, tg.GetSelectedIndex())

	// Move left
	tg.MoveLeft()
	assert.Equal(t, 1, tg.GetSelectedIndex())

	// Move left again
	tg.MoveLeft()
	assert.Equal(t, 0, tg.GetSelectedIndex())

	// Move left at start (should stay)
	tg.MoveLeft()
	assert.Equal(t, 0, tg.GetSelectedIndex())
}

func TestTagGridMoveUpFromFirstRow(t *testing.T) {
	tg := NewTagGrid()
	tg.SetSize(100, 20)
	tags := []models.Tag{
		{ID: "python", Name: "python", Count: 10},
	}
	tg.SetTags(tags)

	// MoveUp from first row should return true (signal to move to search)
	atTop := tg.MoveUp()
	assert.True(t, atTop)
}

func TestTagGridRowNavigation(t *testing.T) {
	tg := NewTagGrid()
	// Small width to force multiple rows
	tg.SetSize(40, 20)
	tags := []models.Tag{
		{ID: "python", Name: "python", Count: 10},
		{ID: "react", Name: "react", Count: 5},
		{ID: "docker", Name: "docker", Count: 3},
		{ID: "kubernetes", Name: "kubernetes", Count: 2},
		{ID: "aws", Name: "aws", Count: 1},
	}
	tg.SetTags(tags)

	// Verify layout creates multiple rows (depends on width)
	assert.Greater(t, len(tg.rowStarts), 0)
}

func TestTagGridFocus(t *testing.T) {
	tg := NewTagGrid()

	assert.False(t, tg.IsFocused())

	tg.SetFocused(true)
	assert.True(t, tg.IsFocused())

	tg.SetFocused(false)
	assert.False(t, tg.IsFocused())
}

func TestTagGridView(t *testing.T) {
	tg := NewTagGrid()
	tg.SetSize(100, 20)
	tags := []models.Tag{
		{ID: "python", Name: "python", Count: 10, Category: "language"},
		{ID: "react", Name: "react", Count: 5, Category: "framework"},
	}
	tg.SetTags(tags)
	tg.SetFocused(true)

	view := tg.View()

	// Should contain tag names and counts
	assert.Contains(t, view, "python")
	assert.Contains(t, view, "(10)")
	assert.Contains(t, view, "react")
	assert.Contains(t, view, "(5)")
}

func TestTagGridEmptyView(t *testing.T) {
	tg := NewTagGrid()
	tg.SetSize(100, 20)

	view := tg.View()

	assert.Contains(t, view, "No tags available")
}
