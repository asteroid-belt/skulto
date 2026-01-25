package scraper

import "strings"

// SeedRepository represents a known source of skills.
type SeedRepository struct {
	Owner     string
	Repo      string
	Priority  int    // 1-10, higher = scrape first
	Type      string // official, curated, community
	SkillPath string // path to skills, default is root
}

// OfficialSeeds are first-party skill repositories.
var OfficialSeeds = []SeedRepository{
	// Primary skills repository (asteroid-belt official)
	{Owner: "asteroid-belt", Repo: "skills", Priority: 10, Type: "official"},

	// Anthropic official
	{Owner: "anthropics", Repo: "skills", Priority: 10, Type: "official"},
	{Owner: "anthropics", Repo: "anthropic-cookbook", Priority: 10, Type: "official"},
}

// PrimarySkillsRepo is the hardcoded primary skills repository.
// This is used for the onboarding flow and auto-sync.
var PrimarySkillsRepo = SeedRepository{
	Owner:    "asteroid-belt",
	Repo:     "skills",
	Priority: 10,
	Type:     "official",
}

// CuratedSeeds are high-quality community repositories.
var CuratedSeeds = []SeedRepository{
	// Awesome-list
	{Owner: "skillcreatorai", Repo: "Awesome-Agent-Skills", Priority: 9, Type: "curated"},
	{Owner: "travisvn", Repo: "awesome-claude-skills", Priority: 9, Type: "curated"},
	{Owner: "alirezarezvani", Repo: "claude-skills", Priority: 9, Type: "curated"},
	{Owner: "abubakarsiddik31", Repo: "claude-skills-collection", Priority: 9, Type: "curated"},
	{Owner: "jqueryscript", Repo: "awesome-claude-code", Priority: 9, Type: "curated"},
	{Owner: "hesreallyhim", Repo: "awesome-claude-code", Priority: 9, Type: "curated"},
	{Owner: "VoltAgent", Repo: "awesome-claude-skills", Priority: 9, Type: "curated"},
	{Owner: "sickn33", Repo: "antigravity-awesome-skills", Priority: 9, Type: "curated"},
	{Owner: "ComposioHQ", Repo: "awesome-claude-skills", Priority: 9, Type: "curated"},
	{Owner: "obra", Repo: "superpowers", Priority: 9, Type: "curated", SkillPath: "skills"},
}

// SearchQueries for discovering new skill repositories.
// Searches for SKILL.md files in common locations and any subdirectory.
var SearchQueries = []string{
	"filename:SKILL.md",

	// Root-level one directory deep
	"filename:SKILL.md path:*/",

	// Nested paths
	"filename:SKILL.md path:.claude/skills",
	"filename:SKILL.md path:.codex/skills",
	"filename:SKILL.md path:.cursor/skills",
}

// AllSeeds returns all seed repositories sorted by priority (highest first).
func AllSeeds() []SeedRepository {
	all := make([]SeedRepository, 0, len(OfficialSeeds)+len(CuratedSeeds))
	all = append(all, OfficialSeeds...)
	all = append(all, CuratedSeeds...)

	// Sort by priority descending (simple insertion sort for small list)
	for i := 1; i < len(all); i++ {
		j := i
		for j > 0 && all[j].Priority > all[j-1].Priority {
			all[j], all[j-1] = all[j-1], all[j]
			j--
		}
	}

	return all
}

// SkillFilePatterns are file patterns that indicate skill files.
// These are the canonical patterns checked (case-insensitive).
var SkillFilePatterns = []string{
	"SKILL.md",
	"skill.md",
	"Skill.md",
	"CLAUDE.md",
	"claude.md",
	"Claude.md",
}

// IsSkillFile checks if a filename matches known skill file patterns.
// Uses case-insensitive matching for consistency with IsSkillFilePath.
func IsSkillFile(filename string) bool {
	lower := strings.ToLower(filename)
	return lower == "skill.md" || lower == "claude.md"
}
