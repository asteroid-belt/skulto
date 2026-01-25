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
