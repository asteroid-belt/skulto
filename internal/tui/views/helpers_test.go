package views

import (
	"testing"

	"github.com/asteroid-belt/skulto/internal/models"
)

func TestFilterOutMineTag(t *testing.T) {
	tests := []struct {
		name     string
		tags     []models.Tag
		maxTags  int
		wantLen  int
		wantMine bool
	}{
		{
			name:     "empty tags",
			tags:     []models.Tag{},
			maxTags:  10,
			wantLen:  0,
			wantMine: false,
		},
		{
			name: "filters mine by ID",
			tags: []models.Tag{
				{ID: "go", Name: "Go"},
				{ID: "mine", Name: "Mine"},
				{ID: "python", Name: "Python"},
			},
			maxTags:  10,
			wantLen:  2,
			wantMine: false,
		},
		{
			name: "filters mine by Slug",
			tags: []models.Tag{
				{ID: "1", Slug: "go", Name: "Go"},
				{ID: "2", Slug: "mine", Name: "Mine"},
				{ID: "3", Slug: "python", Name: "Python"},
			},
			maxTags:  10,
			wantLen:  2,
			wantMine: false,
		},
		{
			name: "respects maxTags limit",
			tags: []models.Tag{
				{ID: "go", Name: "Go"},
				{ID: "python", Name: "Python"},
				{ID: "rust", Name: "Rust"},
				{ID: "java", Name: "Java"},
			},
			maxTags: 2,
			wantLen: 2,
		},
		{
			name: "no limit when maxTags is 0",
			tags: []models.Tag{
				{ID: "go", Name: "Go"},
				{ID: "python", Name: "Python"},
				{ID: "rust", Name: "Rust"},
			},
			maxTags: 0,
			wantLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterOutMineTag(tt.tags, tt.maxTags)
			if len(got) != tt.wantLen {
				t.Errorf("FilterOutMineTag() len = %d, want %d", len(got), tt.wantLen)
			}

			// Check no "mine" tag remains
			for _, tag := range got {
				if tag.ID == "mine" || tag.Slug == "mine" {
					t.Error("FilterOutMineTag() still contains 'mine' tag")
				}
			}
		})
	}
}
