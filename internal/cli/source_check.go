package cli

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/asteroid-belt/skulto/internal/manifest"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/charmbracelet/lipgloss"
)

// SourceMismatchAction is the user's chosen resolution for a source mismatch.
type SourceMismatchAction int

const (
	// SourceMismatchSkip skips the skill (do not install).
	SourceMismatchSkip SourceMismatchAction = iota
	// SourceMismatchAccept updates the manifest to the new source and installs.
	SourceMismatchAccept
	// SourceMismatchInstallAnyway installs without updating the manifest.
	SourceMismatchInstallAnyway
)

// SourceMismatch describes a detected source mismatch.
type SourceMismatch struct {
	Slug           string
	ExpectedSource string
	ActualSource   string
}

// CheckSourceMismatch compares a skill's actual source against an expected source.
// Returns nil if no mismatch (including when skill.Source is nil).
func CheckSourceMismatch(skill *models.Skill, expectedSource string) *SourceMismatch {
	if skill.Source == nil {
		return nil
	}
	if skill.Source.FullName == expectedSource {
		return nil
	}
	return &SourceMismatch{
		Slug:           skill.Slug,
		ExpectedSource: expectedSource,
		ActualSource:   skill.Source.FullName,
	}
}

// PromptSourceMismatch prints a warning and prompts the user for resolution.
// When interactive is false, returns SourceMismatchSkip.
func PromptSourceMismatch(mismatch *SourceMismatch, reader *bufio.Reader, interactive bool) SourceMismatchAction {
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

	fmt.Printf("  %s Skill '%s' found but from different source (%s, expected %s)\n",
		warnStyle.Render("WARN"), mismatch.Slug, mismatch.ActualSource, mismatch.ExpectedSource)

	if !interactive {
		return SourceMismatchSkip
	}

	fmt.Print("  [a]ccept new source  [s]kip  [i]nstall anyway\n")
	fmt.Print("  Choice [s]: ")

	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	switch answer {
	case "a", "accept":
		return SourceMismatchAccept
	case "i", "install":
		return SourceMismatchInstallAnyway
	default:
		return SourceMismatchSkip
	}
}

// ApplySourceMismatchAccept updates skulto.json to point the slug at the new source.
// No-ops if no manifest file exists. If the slug is not in the manifest, it is added.
// Uses manifest.Read and manifest.Write for atomic file operations.
func ApplySourceMismatchAccept(dir string, slug string, newSource string) error {
	mf, err := manifest.Read(dir)
	if err != nil {
		return fmt.Errorf("read manifest for source update: %w", err)
	}
	if mf == nil {
		return nil // No manifest, nothing to update
	}

	mf.Skills[slug] = newSource

	if err := manifest.Write(dir, mf); err != nil {
		return fmt.Errorf("write manifest after source update: %w", err)
	}

	fmt.Printf("  Updated skulto.json: %s -> %s\n", slug, newSource)
	return nil
}
