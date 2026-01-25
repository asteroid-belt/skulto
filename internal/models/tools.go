package models

// AITool represents an available AI tool that developers can select.
type AITool string

const (
	// AIToolClaudeCode represents the Claude Code editor extension.
	AIToolClaudeCode AITool = "Claude Code"

	// AIToolCursor represents the Cursor editor.
	AIToolCursor AITool = "Cursor"

	// AIToolGitHubCopilot represents GitHub Copilot.
	AIToolGitHubCopilot AITool = "GitHub Copilot"

	// AIToolOpenAICodex represents OpenAI Codex.
	AIToolOpenAICodex AITool = "OpenAI Codex"

	// AIToolOpenCode represents OpenCode editor.
	AIToolOpenCode AITool = "OpenCode"

	// AIToolWindsurf represents Windsurf editor.
	AIToolWindsurf AITool = "Windsurf"
)

// AllAITools returns a list of all available AI tools.
func AllAITools() []AITool {
	return []AITool{
		AIToolClaudeCode,
		AIToolCursor,
		AIToolGitHubCopilot,
		AIToolOpenAICodex,
		AIToolOpenCode,
		AIToolWindsurf,
	}
}

// String returns the string representation of the AI tool.
func (t AITool) String() string {
	return string(t)
}

// IsValid checks if the AI tool is valid.
func (t AITool) IsValid() bool {
	switch t {
	case AIToolClaudeCode, AIToolCursor, AIToolGitHubCopilot, AIToolOpenAICodex, AIToolOpenCode, AIToolWindsurf:
		return true
	default:
		return false
	}
}
