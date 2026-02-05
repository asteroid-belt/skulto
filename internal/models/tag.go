package models

// Tag represents a categorization tag for skills.
type Tag struct {
	ID       string  `gorm:"primaryKey;size:100" json:"id"`
	Name     string  `gorm:"size:100;uniqueIndex" json:"name"`
	Slug     string  `gorm:"size:100;uniqueIndex" json:"slug"`
	Category string  `gorm:"size:50;index" json:"category"`   // language, framework, tool, concept, domain
	Color    string  `gorm:"size:20" json:"color"`            // Hex color for UI
	Count    int     `gorm:"default:0" json:"count"`          // Number of skills with this tag
	Priority int     `gorm:"default:0;index" json:"priority"` // Higher = shown first (0 = normal, 100 = mine)
	ParentID *string `gorm:"size:100" json:"parent_id"`

	// Self-referential relationship for hierarchical tags
	Parent   *Tag    `gorm:"foreignKey:ParentID" json:"-"`
	Children []Tag   `gorm:"foreignKey:ParentID" json:"-"`
	Skills   []Skill `gorm:"many2many:skill_tags" json:"-"`
}

// TableName specifies the table name for GORM.
func (Tag) TableName() string {
	return "tags"
}

// TagCategory defines the main tag categories.
type TagCategory string

const (
	TagCategoryLanguage  TagCategory = "language"
	TagCategoryFramework TagCategory = "framework"
	TagCategoryTool      TagCategory = "tool"
	TagCategoryConcept   TagCategory = "concept"
	TagCategoryDomain    TagCategory = "domain"
	TagCategoryMine      TagCategory = "mine" // Special category for user's own skills
)

// TagColors maps categories to default colors.
var TagColors = map[TagCategory]string{
	TagCategoryLanguage:  "#8B5CF6", // Purple
	TagCategoryFramework: "#EC4899", // Pink
	TagCategoryTool:      "#10B981", // Emerald
	TagCategoryConcept:   "#F59E0B", // Amber
	TagCategoryDomain:    "#3B82F6", // Blue
	TagCategoryMine:      "#DC143C", // Crimson (matches Skulto accent)
}

// PredefinedTags is the curated list of tags for auto-tagging.
var PredefinedTags = map[TagCategory][]string{
	TagCategoryLanguage: {
		// Existing
		"python", "javascript", "typescript", "go", "rust", "java",
		"csharp", "cpp", "ruby", "php", "swift", "kotlin", "scala",
		// New - commonly mentioned
		"bash", "sql", "yaml", "markdown", "lua",
	},
	TagCategoryFramework: {
		// Existing
		"react", "vue", "angular", "svelte", "nextjs", "django", "fastapi",
		"flask", "express", "nestjs", "spring", "rails", "laravel",
		// New - AI frameworks
		"langchain", "llamaindex", "crewai", "autogen",
		// New - Modern web
		"tailwind", "prisma", "drizzle", "shadcn", "htmx", "pydantic",
	},
	TagCategoryTool: {
		// Existing
		"docker", "kubernetes", "terraform", "git", "aws", "gcp", "azure",
		"postgresql", "mongodb", "redis", "elasticsearch", "grafana",
		// New - AI tools
		"claude", "openai", "ollama", "gemini",
		// New - Vector DBs
		"pinecone", "chroma", "weaviate",
		// New - Dev tools
		"vscode", "cursor", "bun", "pnpm", "vite",
		// New - Databases
		"mysql", "sqlite", "supabase", "firebase",
		// New - Deployment
		"vercel", "netlify",
	},
	TagCategoryConcept: {
		// Existing
		"testing", "security", "performance", "accessibility", "documentation",
		"code-review", "refactoring", "debugging", "ci-cd", "monitoring",
		// New - AI/LLM concepts
		"prompts", "agents", "rag", "embeddings", "fine-tuning",
		"chain-of-thought", "few-shot", "tool-use", "function-calling",
		"context-window", "system-prompts",
		// New - MCP
		"mcp",
	},
	TagCategoryDomain: {
		// Existing
		"web", "mobile", "backend", "frontend", "devops", "ml", "ai",
		"data", "security", "cloud", "embedded", "game-dev",
		// New - AI domains
		"llm", "nlp", "chatbot", "automation", "workflows",
	},
}

// AllCategories returns all tag categories.
func AllCategories() []TagCategory {
	return []TagCategory{
		TagCategoryLanguage,
		TagCategoryFramework,
		TagCategoryTool,
		TagCategoryConcept,
		TagCategoryDomain,
	}
}

// MineTag returns the special "mine" tag for CWD skills.
func MineTag() Tag {
	return Tag{
		ID:       "mine",
		Name:     "mine",
		Slug:     "mine",
		Category: string(TagCategoryMine),
		Color:    TagColors[TagCategoryMine],
		Priority: 100, // Always first
	}
}

// MaxTagsPerSkill limits the number of tags assigned to a single skill.
const MaxTagsPerSkill = 5

// TagAliases maps common abbreviations/variants to canonical tag names.
var TagAliases = map[string]string{
	// Languages
	"js":     "javascript",
	"ts":     "typescript",
	"py":     "python",
	"golang": "go",
	"c#":     "csharp",
	"c++":    "cpp",
	// Tools
	"k8s":      "kubernetes",
	"postgres": "postgresql",
	"mongo":    "mongodb",
	"node":     "nodejs",
	"node.js":  "nodejs",
	"es":       "elasticsearch",
	// Frameworks
	"next":         "nextjs",
	"next.js":      "nextjs",
	"nuxt":         "vue",
	"nuxt.js":      "vue",
	"react-native": "react",
	"rn":           "react",
	// AI
	"gpt":       "openai",
	"chatgpt":   "openai",
	"gpt-4":     "openai",
	"anthropic": "claude",
	"llama":     "ollama",
}

// ImpliedTags maps tags to other tags they imply (framework → language, etc.).
var ImpliedTags = map[string][]string{
	// JavaScript frameworks
	"react":   {"javascript", "frontend"},
	"nextjs":  {"javascript", "react", "frontend"},
	"vue":     {"javascript", "frontend"},
	"angular": {"typescript", "frontend"},
	"svelte":  {"javascript", "frontend"},
	"express": {"javascript", "nodejs", "backend"},
	"nestjs":  {"typescript", "nodejs", "backend"},
	// Python frameworks
	"django":     {"python", "backend"},
	"fastapi":    {"python", "backend"},
	"flask":      {"python", "backend"},
	"langchain":  {"python", "ai", "llm"},
	"llamaindex": {"python", "ai", "llm"},
	"pydantic":   {"python"},
	// Other frameworks
	"rails":   {"ruby", "backend"},
	"laravel": {"php", "backend"},
	"spring":  {"java", "backend"},
	// Tools → domains
	"terraform":  {"devops", "cloud"},
	"kubernetes": {"devops", "cloud"},
	"docker":     {"devops"},
	"aws":        {"cloud"},
	"gcp":        {"cloud"},
	"azure":      {"cloud"},
	// AI tools
	"claude": {"ai", "llm"},
	"openai": {"ai", "llm"},
	"ollama": {"ai", "llm"},
	"gemini": {"ai", "llm"},
	// Concepts
	"rag":    {"ai", "llm"},
	"agents": {"ai", "llm"},
	"mcp":    {"ai", "agents"},
}

// MinOccurrences specifies minimum mentions required for generic tags.
// Tags not in this map default to 1 occurrence.
var MinOccurrences = map[string]int{
	// Very generic domain tags need multiple mentions
	"web":        2,
	"data":       2,
	"cloud":      2,
	"ai":         2,
	"ml":         2,
	"backend":    2,
	"frontend":   2,
	"devops":     2,
	"automation": 2,
	"workflows":  2,
	// Generic concepts
	"security":    2,
	"testing":     2,
	"performance": 2,
}

// TitleBoostMultiplier is the weight multiplier for words found in the title.
const TitleBoostMultiplier = 3

// DescriptionBoostMultiplier is the weight multiplier for words found in the description.
const DescriptionBoostMultiplier = 2
