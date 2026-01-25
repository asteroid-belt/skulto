package search

import (
	"time"

	"github.com/asteroid-belt/skulto/internal/models"
)

// MatchType indicates how a skill matched the query.
type MatchType int

const (
	// MatchTypeTitle indicates the query matched in the skill title.
	MatchTypeTitle MatchType = iota
	// MatchTypeContent indicates the query matched in the skill content.
	MatchTypeContent
)

// Snippet represents a matching text snippet with optional highlights.
type Snippet struct {
	Text       string
	Highlights []Highlight
}

// Highlight marks a highlighted region in a snippet.
type Highlight struct {
	Start int
	End   int
}

// SkillMatch represents a search result with metadata.
type SkillMatch struct {
	Skill     models.Skill
	Score     float32
	MatchType MatchType
	Snippets  []Snippet
}

// SearchResults contains categorized search results.
type SearchResults struct {
	TitleMatches   []SkillMatch
	ContentMatches []SkillMatch
	Query          string
	Duration       time.Duration
	TotalHits      int
}

// SearchOptions configures search behavior.
type SearchOptions struct {
	Limit           int
	Threshold       float32
	IncludeFTS      bool
	IncludeSemantic bool
}

// DefaultSearchOptions returns sensible defaults.
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		Limit:           50,
		Threshold:       0.6,
		IncludeFTS:      true,
		IncludeSemantic: true,
	}
}

// IndexStats contains statistics about the search index.
type IndexStats struct {
	TotalSkills      int64
	IndexedSkills    int64
	PendingSkills    int64
	LastIndexedAt    time.Time
	VectorStoreReady bool
}
