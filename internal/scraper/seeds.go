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
}

// PrimarySkillsRepo is the hardcoded primary skills repository.
// This is used for the onboarding flow and auto-sync.
var PrimarySkillsRepo = SeedRepository{
	Owner:    "asteroid-belt",
	Repo:     "skills",
	Priority: 10,
	Type:     "official",
}

// CuratedSeeds are high-quality verified repositories from major companies and trusted community members.
var CuratedSeeds = []SeedRepository{
	// Major Companies - Verified Organizations
	{Owner: "vercel-labs", Repo: "agent-skills", Priority: 9, Type: "curated"},       // React/Vercel best practices, 17.6K stars
	{Owner: "supabase", Repo: "agent-skills", Priority: 9, Type: "curated"},           // Postgres optimization
	{Owner: "expo", Repo: "skills", Priority: 9, Type: "curated"},                     // Official Expo team, React Native
	{Owner: "trailofbits", Repo: "skills", Priority: 9, Type: "curated"},              // Security auditing, 24 skills
	{Owner: "remotion-dev", Repo: "skills", Priority: 9, Type: "curated"},             // Video rendering
	{Owner: "better-auth", Repo: "skills", Priority: 8, Type: "curated"},              // Authentication patterns
	{Owner: "neondatabase", Repo: "agent-skills", Priority: 8, Type: "curated"},       // Serverless Postgres
	{Owner: "cloudflare", Repo: "skills", Priority: 8, Type: "curated"},               // Workers/Edge
	{Owner: "getsentry", Repo: "skills", Priority: 8, Type: "curated"},                // Error tracking
	{Owner: "tinybirdco", Repo: "tinybird-agent-skills", Priority: 8, Type: "curated"}, // Real-time analytics

	// High-Quality Community
	{Owner: "obra", Repo: "superpowers", Priority: 9, Type: "curated", SkillPath: "skills"}, // Jesse Vincent's TDD/debugging methodology
	{Owner: "alirezarezvani", Repo: "claude-skills", Priority: 8, Type: "curated"},          // 48 domain expert skills
	{Owner: "muratcankoylan", Repo: "Agent-Skills-for-Context-Engineering", Priority: 8, Type: "curated"}, // Context engineering

	// Specialized Skills
	{Owner: "antonbabenko", Repo: "terraform-skill", Priority: 7, Type: "curated"},    // Terraform/IaC expert
	{Owner: "zxkane", Repo: "aws-skills", Priority: 7, Type: "curated"},               // AWS
	{Owner: "lackeyjb", Repo: "playwright-skill", Priority: 7, Type: "curated"},       // E2E testing
	{Owner: "ibelick", Repo: "ui-skills", Priority: 7, Type: "curated"},               // UI design
	{Owner: "callstackincubator", Repo: "agent-skills", Priority: 7, Type: "curated"}, // React Native (Callstack)
	{Owner: "czlonkowski", Repo: "n8n-skills", Priority: 7, Type: "curated"},          // n8n automation
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
