package components

import (
	"fmt"

	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// SkillItemStyle defines the rendering style for skill items
type SkillItemStyle int

const (
	// SimpleStyle renders title and description only
	SimpleStyle SkillItemStyle = iota
	// DetailedStyle renders title, description, author, and category
	DetailedStyle
)

// RenderSkillItem renders a skill item in normal (non-selected) state
func RenderSkillItem(skill models.Skill, style SkillItemStyle) string {
	switch style {
	case DetailedStyle:
		return renderSkillItemDetailed(skill)
	default:
		return renderSkillItemSimple(skill)
	}
}

// RenderSelectedSkill renders a skill item in selected state
func RenderSelectedSkill(skill models.Skill, style SkillItemStyle) string {
	switch style {
	case DetailedStyle:
		return renderSelectedSkillDetailed(skill)
	default:
		return renderSelectedSkillSimple(skill)
	}
}

// renderSkillItemSimple renders a skill item with title and description only
func renderSkillItemSimple(skill models.Skill) string {
	itemStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Text).
		MarginTop(1).
		MarginLeft(1).
		MarginRight(2)

	title := lipgloss.NewStyle().Bold(true).Render(skill.Title)
	desc := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		Italic(true).
		Render(truncate(skill.Description, 140))

	return itemStyle.Render(title + "\n" + desc)
}

// renderSelectedSkillSimple renders a selected skill item with title and description only
func renderSelectedSkillSimple(skill models.Skill) string {
	boxStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextHighlight).
		MarginTop(1).
		MarginLeft(1).
		MarginRight(2).
		Bold(true)

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current.Accent).
		Render(skill.Title)
	desc := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		Italic(true).
		Render(truncate(skill.Description, 140))

	return boxStyle.Render(title + "\n" + desc)
}

// renderSkillItemDetailed renders a skill item with metadata (title, description, author, category)
func renderSkillItemDetailed(skill models.Skill) string {
	titleStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextHighlight).
		Bold(true)

	sourceStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted)

	descStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Text)

	metaStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		Italic(true)

	itemStyle := lipgloss.NewStyle().
		MarginLeft(1).
		MarginRight(2).
		MarginBottom(1)

	// Build title line with source info
	titleLine := titleStyle.Render(skill.Title)
	if skill.Source != nil && skill.Source.Owner != "" && skill.Source.Repo != "" {
		sourceInfo := sourceStyle.Render(fmt.Sprintf(" (%s/%s)", skill.Source.Owner, skill.Source.Repo))
		titleLine = titleLine + sourceInfo
	}

	desc := descStyle.Render(truncate(skill.Description, 140))
	meta := metaStyle.Render(fmt.Sprintf("↑ by %s • %s",
		skill.Author, skill.Category))

	return itemStyle.Render(titleLine + "\n" + desc + "\n" + meta)
}

// renderSelectedSkillDetailed renders a selected skill item with metadata (title, description, author, category)
func renderSelectedSkillDetailed(skill models.Skill) string {
	titleStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Accent).
		Bold(true)

	sourceStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted)

	descStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Text)

	metaStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		Italic(true)

	boxStyle := lipgloss.NewStyle().
		MarginLeft(1).
		MarginRight(2).
		MarginBottom(1)

	// Build title line with source info
	titleLine := titleStyle.Render(skill.Title)
	if skill.Source != nil && skill.Source.Owner != "" && skill.Source.Repo != "" {
		sourceInfo := sourceStyle.Render(fmt.Sprintf(" (%s/%s)", skill.Source.Owner, skill.Source.Repo))
		titleLine = titleLine + sourceInfo
	}

	desc := descStyle.Render(truncate(skill.Description, 140))
	meta := metaStyle.Render(fmt.Sprintf("↑ by %s • %s",
		skill.Author, skill.Category))

	content := titleLine + "\n" + desc + "\n" + meta
	return boxStyle.Render(content)
}

// truncate truncates a string to a maximum length.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
