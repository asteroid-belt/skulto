package skillgen

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/asteroid-belt/skulto/internal/log"
)

// debugLog wraps log.DebugLog with the "skillgen" component.
func debugLog(format string, args ...interface{}) {
	log.DebugLog("skillgen", format, args...)
}

// AITool represents the supported AI CLI tools.
type AITool string

const (
	AIToolClaude   AITool = "claude"
	AIToolCodex    AITool = "codex"
	AIToolOpenCode AITool = "opencode"
)

// StreamReader provides streaming access to CLI output.
type StreamReader struct {
	stdout    io.ReadCloser
	stderr    *bytes.Buffer // captures stderr for error messages
	cmd       *exec.Cmd
	mu        sync.Mutex
	done      bool
	err       error
	killErr   error // stores error from Kill() if any
	collected strings.Builder
}

// Read reads a chunk from the stream.
func (sr *StreamReader) Read() (string, bool, error) {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	if sr.done {
		return "", true, sr.err
	}

	buf := make([]byte, 4096)
	n, err := sr.stdout.Read(buf)
	if err != nil {
		if err == io.EOF {
			sr.done = true
			// Wait for command to finish
			waitErr := sr.cmd.Wait()
			if waitErr != nil && sr.err == nil {
				// Include stderr in the error message for better debugging
				stderrContent := ""
				if sr.stderr != nil {
					stderrContent = strings.TrimSpace(sr.stderr.String())
				}
				debugLog("Command failed: %v, stderr: %s", waitErr, stderrContent)
				if stderrContent != "" {
					sr.err = fmt.Errorf("%w: %s", waitErr, stderrContent)
				} else {
					sr.err = waitErr
				}
			}
			return "", true, sr.err
		}
		sr.err = err
		sr.done = true
		return "", true, err
	}

	chunk := string(buf[:n])
	sr.collected.WriteString(chunk)
	return chunk, false, nil
}

// Collect reads all remaining output and returns the full content.
func (sr *StreamReader) Collect() (string, error) {
	for {
		chunk, done, err := sr.Read()
		if done {
			if err != nil {
				return sr.collected.String(), err
			}
			return sr.collected.String(), nil
		}
		_ = chunk // Already collected in Read()
	}
}

// Cancel stops the running command.
func (sr *StreamReader) Cancel() {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	if sr.cmd != nil && sr.cmd.Process != nil {
		sr.killErr = sr.cmd.Process.Kill()
	}
	sr.done = true
}

// KillError returns any error that occurred when killing the process.
func (sr *StreamReader) KillError() error {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	return sr.killErr
}

// CLIExecutor executes AI CLI tools.
type CLIExecutor struct {
	// SkillPromptTemplate is the prompt template for skill generation.
	// It wraps the user's description with instructions for skill format.
	SkillPromptTemplate string
}

// NewCLIExecutor creates a new CLI executor.
func NewCLIExecutor() *CLIExecutor {
	return &CLIExecutor{
		SkillPromptTemplate: defaultSkillPromptTemplate,
	}
}

const defaultSkillPromptTemplate = `You are a skill file generator. Generate a complete skill file in AgentSkills.io format.

For the most up-to-date skill specification, refer to: https://agentskills.io/specification

The user wants a skill that:
%s

Generate a complete skill file with:
1. YAML frontmatter with: name, description, metadata (version, author, tags, platforms)
2. Markdown body with clear instructions
3. Good and bad examples if appropriate

Output ONLY the skill file content, starting with --- for the frontmatter.
Do not include any explanation or commentary outside the skill file.`

// interactiveSystemPromptTemplate is used when launching Claude interactively.
// It instructs Claude to save the generated skill to the skulto skills folder.
// The %s placeholder is replaced with the absolute path to the skills directory.
const interactiveSystemPromptTemplate = `You are a skill file generator for AgentSkills.io / Skulto.

## REQUIRED: Read the Specification First

Before generating any skill, you MUST use WebFetch to read the official specification:
https://agentskills.io/specification

Read this specification completely to understand the current skill format, required fields, and best practices.

The user will describe a skill they want to create. Your job is to:
1. Read the specification at https://agentskills.io/specification using WebFetch
2. Ask clarifying questions if needed to understand exactly what the skill should do
3. Research best practices to make the skill repeatable and high-quality
4. Generate a complete skill file in AgentSkills.io format
5. Save all research and references to the references/ folder
6. Create automation scripts in the scripts/ folder where applicable
7. Save the skill to the Skulto skills folder

## REQUIRED: Research for Repeatability

To make skills repeatable and high-quality, you MUST:
1. Research current best practices for the skill's domain using WebSearch
2. Find authoritative sources, documentation, and examples
3. Document your research findings in markdown files in the references/ folder:
   - %s/<skill-slug>/references/research.md - Summary of research findings
   - %s/<skill-slug>/references/sources.md - List of sources with URLs
   - %s/<skill-slug>/references/<topic>.md - Additional topic-specific research as needed

Include in your research:
- Official documentation links
- Best practices from authoritative sources
- Common pitfalls to avoid
- Version-specific considerations (if applicable)

## REQUIRED: Automation Scripts

To maximize automation, create scripts in the scripts/ folder where applicable:
- %s/<skill-slug>/scripts/ - Directory for automation scripts
- Use appropriate scripting languages (bash, python, etc.) based on the task
- Include a README.md in the scripts folder explaining each script

Examples of scripts to create:
- setup.sh - Environment setup or installation steps
- validate.sh - Validation or linting scripts
- test.sh - Test scripts for the skill's functionality
- Any task-specific automation that makes the skill more repeatable

Make scripts executable and well-documented with comments.

## Skill File Format

The skill file must have:
- YAML frontmatter between --- markers with: name, description, metadata (version, author, tags)
- Markdown body with clear instructions for how the AI should behave
- Examples of good and bad usage if appropriate

Example structure:
` + "```" + `
---
name: my-skill-name
description: Brief description of what this skill does
metadata:
  version: 1.0.0
  author: User
  tags:
    - category1
    - category2
---

# Skill Title

Instructions for the AI...

## When to Use

- Scenario 1
- Scenario 2

## Examples

<example>
Good example here
</example>
` + "```" + `

## CRITICAL: Saving the Skill

**YOUR SKILLS DIRECTORY IS: %s**

When the skill is ready, you MUST save files using the Write tool to these EXACT paths:

1. Main skill file:
   %s/<skill-slug>/skill.md

2. Research references:
   %s/<skill-slug>/references/research.md
   %s/<skill-slug>/references/sources.md

3. Automation scripts (if applicable):
   %s/<skill-slug>/scripts/README.md
   %s/<skill-slug>/scripts/<script-name>.sh

Where <skill-slug> is a kebab-case version of the skill name (e.g., "my-awesome-skill").

DO NOT use ~ or $HOME. Use the EXACT absolute path shown above.
Create all directories if they don't exist.`

// Execute runs the specified AI CLI tool with the given prompt.
func (e *CLIExecutor) Execute(ctx context.Context, tool AITool, userPrompt string) (*StreamReader, error) {
	// Find the CLI executable
	cliPath, err := e.findCLI(tool)
	if err != nil {
		debugLog("CLI not found for tool %s: %v", tool, err)
		return nil, err
	}

	debugLog("Found CLI at: %s", cliPath)

	// Build the full prompt
	fullPrompt := fmt.Sprintf(e.SkillPromptTemplate, userPrompt)

	debugLog("Building command for tool %s with prompt length %d", tool, len(fullPrompt))

	// Build the command
	cmd := e.buildCommand(ctx, tool, cliPath, fullPrompt)

	debugLog("Command args: %v", cmd.Args)

	// Set up pipe for stdout
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Capture stderr for error messages
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	// Start the command
	debugLog("Starting command...")
	if err := cmd.Start(); err != nil {
		debugLog("Failed to start command: %v", err)
		return nil, fmt.Errorf("failed to start %s: %w", tool, err)
	}

	debugLog("Command started with PID %d", cmd.Process.Pid)

	return &StreamReader{
		stdout: stdout,
		stderr: &stderrBuf,
		cmd:    cmd,
	}, nil
}

// findCLI locates the CLI executable for the given tool.
func (e *CLIExecutor) findCLI(tool AITool) (string, error) {
	var paths []string

	// Get home directory for user-local installations
	homeDir, _ := os.UserHomeDir()

	switch tool {
	case AIToolClaude:
		// Check user-local Claude installation first (newer versions install here)
		if homeDir != "" {
			paths = append(paths, homeDir+"/.claude/local/claude")
		}
		// Then fall back to PATH lookup
		paths = append(paths, "") // empty string signals PATH lookup for "claude"
	case AIToolCodex:
		paths = append(paths, "") // PATH lookup for "codex"
	case AIToolOpenCode:
		paths = append(paths, "") // PATH lookup for "opencode"
	default:
		return "", fmt.Errorf("unknown AI tool: %s", tool)
	}

	// Mapping for PATH lookups
	pathNames := map[AITool][]string{
		AIToolClaude:   {"claude", "claude-code"},
		AIToolCodex:    {"codex"},
		AIToolOpenCode: {"opencode"},
	}

	for _, p := range paths {
		if p != "" {
			// Direct path check
			if info, err := os.Stat(p); err == nil && !info.IsDir() {
				debugLog("Found CLI at direct path: %s", p)
				return p, nil
			}
		} else {
			// PATH lookup
			for _, name := range pathNames[tool] {
				path, err := exec.LookPath(name)
				if err == nil {
					debugLog("Found CLI via PATH lookup: %s -> %s", name, path)
					return path, nil
				}
			}
		}
	}

	return "", fmt.Errorf("%s CLI not found in PATH. Please install it first", tool)
}

// buildCommand creates the exec.Cmd for the given tool.
func (e *CLIExecutor) buildCommand(ctx context.Context, tool AITool, cliPath, prompt string) *exec.Cmd {
	var args []string

	switch tool {
	case AIToolClaude:
		// claude -p --output-format text "prompt"
		// -p is a boolean flag (print mode), prompt is a positional argument at the end
		args = []string{"-p", "--output-format", "text", prompt}
	case AIToolCodex:
		// codex -q "prompt"
		args = []string{"-q", prompt}
	case AIToolOpenCode:
		// opencode run "prompt"
		args = []string{"run", prompt}
	}

	cmd := exec.CommandContext(ctx, cliPath, args...)
	return cmd
}

// AvailableTools returns a list of AI tools that are installed.
func (e *CLIExecutor) AvailableTools() []AITool {
	var available []AITool

	tools := []AITool{AIToolClaude, AIToolCodex, AIToolOpenCode}
	for _, tool := range tools {
		if _, err := e.findCLI(tool); err == nil {
			available = append(available, tool)
		}
	}

	return available
}

// InteractiveCommand builds an exec.Cmd for launching Claude interactively.
// The caller should use tea.ExecProcess to run this command.
func (e *CLIExecutor) InteractiveCommand(tool AITool, userPrompt string) (*exec.Cmd, error) {
	cliPath, err := e.findCLI(tool)
	if err != nil {
		return nil, err
	}

	// Get the absolute path to the skills directory
	skillsDir, err := SkillsDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get skills directory: %w", err)
	}

	// Build the system prompt with the absolute path
	// The template has 10 %s placeholders for skillsDir paths
	systemPrompt := fmt.Sprintf(interactiveSystemPromptTemplate,
		skillsDir, skillsDir, skillsDir, // references paths in research section
		skillsDir,            // scripts path in automation section
		skillsDir,            // main skills directory header
		skillsDir,            // skill.md path
		skillsDir, skillsDir, // references paths in saving section
		skillsDir, skillsDir, // scripts paths in saving section
	)

	debugLog("Building interactive command for %s with skills dir: %s", tool, skillsDir)

	var args []string
	switch tool {
	case AIToolClaude:
		// claude --system-prompt "..." "user prompt"
		args = []string{"--system-prompt", systemPrompt, userPrompt}
	case AIToolCodex:
		// codex doesn't support system prompt the same way, fall back to combined prompt
		args = []string{systemPrompt + "\n\nUser request: " + userPrompt}
	case AIToolOpenCode:
		// opencode doesn't support system prompt flag, combine into single message
		// opencode run "combined prompt"
		combinedPrompt := systemPrompt + "\n\n---\n\nUser request: " + userPrompt
		args = []string{"run", combinedPrompt}
	}

	debugLog("Interactive command args count: %d", len(args))

	cmd := exec.Command(cliPath, args...)
	return cmd, nil
}

// SkillsDir returns the skulto skills directory path.
func SkillsDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".skulto", "skills"), nil
}

// SkillInfo represents basic info about a skill in the skills folder.
type SkillInfo struct {
	Slug    string
	Path    string
	ModTime time.Time
}

// ScanSkills returns a list of skills in the skulto skills folder.
func ScanSkills() ([]SkillInfo, error) {
	dir, err := SkillsDir()
	if err != nil {
		return nil, err
	}

	debugLog("Scanning skills directory: %s", dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			debugLog("Skills directory does not exist")
			return nil, nil // No skills folder yet
		}
		return nil, err
	}

	var skills []SkillInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check for skill.md (lowercase)
		skillPath := filepath.Join(dir, entry.Name(), "skill.md")
		info, err := os.Stat(skillPath)
		if err != nil {
			continue // Skip if skill.md doesn't exist
		}

		debugLog("Found skill: %s at %s (mod: %v)", entry.Name(), skillPath, info.ModTime())

		skills = append(skills, SkillInfo{
			Slug:    entry.Name(),
			Path:    skillPath,
			ModTime: info.ModTime(),
		})
	}

	debugLog("Total skills found: %d", len(skills))
	return skills, nil
}

// FindNewSkills compares before and after snapshots to find new/modified skills.
func FindNewSkills(before, after []SkillInfo) []SkillInfo {
	beforeMap := make(map[string]time.Time)
	for _, s := range before {
		beforeMap[s.Slug] = s.ModTime
	}

	var newSkills []SkillInfo
	for _, s := range after {
		if oldTime, exists := beforeMap[s.Slug]; !exists || s.ModTime.After(oldTime) {
			newSkills = append(newSkills, s)
		}
	}

	return newSkills
}

// CwdSkillsDir returns the skills directory path in the current working directory.
// Returns empty string if the directory doesn't exist or cwd cannot be determined.
func CwdSkillsDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	skillsDir := filepath.Join(cwd, ".skulto", "skills")

	// Check if directory exists
	info, err := os.Stat(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // Not an error, just doesn't exist
		}
		return "", err
	}
	if !info.IsDir() {
		return "", nil
	}
	return skillsDir, nil
}

// CwdSkillInfo extends SkillInfo with category information for CWD skills.
// Note: Category field is kept for backward compatibility but is always empty
// as we now only support flat structure (skills/<name>/skill.md).
type CwdSkillInfo struct {
	SkillInfo
	Category string // Always empty - kept for backward compatibility
}

// ScanCwdSkills returns a list of skills in the cwd's .skulto/skills folder.
// Returns nil (not error) if the folder doesn't exist.
// Only scans top-level directories: skills/<name>/skill.md
// Handles case-insensitive skill.md/SKILL.md.
func ScanCwdSkills() ([]SkillInfo, error) {
	skills, err := ScanCwdSkillsWithCategory()
	if err != nil {
		return nil, err
	}

	// Convert to basic SkillInfo for backward compatibility
	result := make([]SkillInfo, len(skills))
	for i, s := range skills {
		result[i] = s.SkillInfo
	}
	return result, nil
}

// ScanCwdSkillsWithCategory returns skills with their category information.
// Only scans top-level directories in the skills folder.
// Category is always empty (flat structure only).
func ScanCwdSkillsWithCategory() ([]CwdSkillInfo, error) {
	dir, err := CwdSkillsDir()
	if err != nil {
		return nil, err
	}
	if dir == "" {
		return nil, nil // No cwd skills folder
	}

	debugLog("Scanning CWD skills directory: %s", dir)

	// Read only top-level entries (no recursive walk)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var skills []CwdSkillInfo

	for _, entry := range entries {
		// Skip non-directories
		if !entry.IsDir() {
			continue
		}

		skillDir := filepath.Join(dir, entry.Name())

		// Check for skill.md or SKILL.md (case-insensitive)
		var skillPath string
		var info os.FileInfo

		// Try lowercase first
		path := filepath.Join(skillDir, "skill.md")
		if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
			skillPath = path
			info = fi
		} else {
			// Try uppercase
			path = filepath.Join(skillDir, "SKILL.md")
			if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
				skillPath = path
				info = fi
			}
		}

		// Skip if no skill file found
		if skillPath == "" {
			continue
		}

		slug := entry.Name()
		debugLog("Found CWD skill: %s at %s", slug, skillPath)

		skills = append(skills, CwdSkillInfo{
			SkillInfo: SkillInfo{
				Slug:    slug,
				Path:    skillPath,
				ModTime: info.ModTime(),
			},
			Category: "", // Always empty - flat structure only
		})
	}

	debugLog("Total CWD skills found: %d", len(skills))
	return skills, nil
}
