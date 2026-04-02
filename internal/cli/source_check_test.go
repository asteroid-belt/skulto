package cli

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/asteroid-belt/skulto/internal/manifest"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckSourceMismatch_NilSource(t *testing.T) {
	skill := &models.Skill{Slug: "test-skill"}
	result := CheckSourceMismatch(skill, "owner/repo")
	assert.Nil(t, result)
}

func TestCheckSourceMismatch_MatchingSource(t *testing.T) {
	skill := &models.Skill{
		Slug:   "test-skill",
		Source: &models.Source{FullName: "owner/repo"},
	}
	result := CheckSourceMismatch(skill, "owner/repo")
	assert.Nil(t, result)
}

func TestCheckSourceMismatch_MismatchDetected(t *testing.T) {
	skill := &models.Skill{
		Slug:   "test-skill",
		Source: &models.Source{FullName: "evil-fork/repo"},
	}
	result := CheckSourceMismatch(skill, "owner/repo")

	require.NotNil(t, result)
	assert.Equal(t, "test-skill", result.Slug)
	assert.Equal(t, "owner/repo", result.ExpectedSource)
	assert.Equal(t, "evil-fork/repo", result.ActualSource)
}

func TestPromptSourceMismatch_AcceptChoice(t *testing.T) {
	mismatch := &SourceMismatch{
		Slug:           "superplan",
		ExpectedSource: "asteroid-belt/skills",
		ActualSource:   "evil-fork/skills",
	}
	reader := bufio.NewReader(strings.NewReader("a\n"))
	action := PromptSourceMismatch(mismatch, reader, true)
	assert.Equal(t, SourceMismatchAccept, action)
}

func TestPromptSourceMismatch_SkipChoice(t *testing.T) {
	mismatch := &SourceMismatch{
		Slug:           "superplan",
		ExpectedSource: "asteroid-belt/skills",
		ActualSource:   "evil-fork/skills",
	}
	reader := bufio.NewReader(strings.NewReader("s\n"))
	action := PromptSourceMismatch(mismatch, reader, true)
	assert.Equal(t, SourceMismatchSkip, action)
}

func TestPromptSourceMismatch_InstallAnywayChoice(t *testing.T) {
	mismatch := &SourceMismatch{
		Slug:           "superplan",
		ExpectedSource: "asteroid-belt/skills",
		ActualSource:   "evil-fork/skills",
	}
	reader := bufio.NewReader(strings.NewReader("i\n"))
	action := PromptSourceMismatch(mismatch, reader, true)
	assert.Equal(t, SourceMismatchInstallAnyway, action)
}

func TestPromptSourceMismatch_DefaultIsSkip(t *testing.T) {
	mismatch := &SourceMismatch{
		Slug:           "superplan",
		ExpectedSource: "asteroid-belt/skills",
		ActualSource:   "evil-fork/skills",
	}
	reader := bufio.NewReader(strings.NewReader("\n"))
	action := PromptSourceMismatch(mismatch, reader, true)
	assert.Equal(t, SourceMismatchSkip, action)
}

func TestPromptSourceMismatch_NonInteractiveDefaultsToSkip(t *testing.T) {
	mismatch := &SourceMismatch{
		Slug:           "superplan",
		ExpectedSource: "asteroid-belt/skills",
		ActualSource:   "evil-fork/skills",
	}
	reader := bufio.NewReader(strings.NewReader("a\n"))
	action := PromptSourceMismatch(mismatch, reader, false)
	assert.Equal(t, SourceMismatchSkip, action)
}

func TestApplySourceMismatchAccept_UpdatesExistingEntry(t *testing.T) {
	dir := t.TempDir()
	mf := manifest.New()
	mf.Skills["superplan"] = "old-owner/skills"
	require.NoError(t, manifest.Write(dir, mf))

	err := ApplySourceMismatchAccept(dir, "superplan", "new-owner/skills")
	require.NoError(t, err)

	updated, err := manifest.Read(dir)
	require.NoError(t, err)
	assert.Equal(t, "new-owner/skills", updated.Skills["superplan"])
}

func TestApplySourceMismatchAccept_AddsNewEntry(t *testing.T) {
	dir := t.TempDir()
	mf := manifest.New()
	mf.Skills["existing"] = "owner/repo"
	require.NoError(t, manifest.Write(dir, mf))

	err := ApplySourceMismatchAccept(dir, "new-skill", "owner/repo")
	require.NoError(t, err)

	updated, err := manifest.Read(dir)
	require.NoError(t, err)
	assert.Equal(t, "owner/repo", updated.Skills["new-skill"])
	assert.Equal(t, "owner/repo", updated.Skills["existing"])
}

func TestApplySourceMismatchAccept_NoManifest(t *testing.T) {
	dir := t.TempDir()

	err := ApplySourceMismatchAccept(dir, "superplan", "new-owner/skills")
	assert.NoError(t, err)

	// Verify no manifest was created
	_, err = os.Stat(filepath.Join(dir, "skulto.json"))
	assert.True(t, os.IsNotExist(err))
}

// --- Integration tests ---

func TestCheckSourceMismatch_Integration_SyncScenario(t *testing.T) {
	// Simulates sync resolving a skill from the wrong source
	sourceID := "evil-fork/skills"
	skill := &models.Skill{
		Slug:     "superplan",
		SourceID: &sourceID,
		Source:   &models.Source{FullName: "evil-fork/skills"},
	}

	mismatch := CheckSourceMismatch(skill, "asteroid-belt/skills")
	require.NotNil(t, mismatch)
	assert.Equal(t, "superplan", mismatch.Slug)
	assert.Equal(t, "asteroid-belt/skills", mismatch.ExpectedSource)
	assert.Equal(t, "evil-fork/skills", mismatch.ActualSource)
}

func TestApplySourceMismatchAccept_Integration_ManifestRoundTrip(t *testing.T) {
	dir := t.TempDir()

	// Create a manifest with multiple skills
	mf := manifest.New()
	mf.Skills["superplan"] = "asteroid-belt/skills"
	mf.Skills["teach"] = "asteroid-belt/skills"
	mf.Skills["other"] = "other-owner/repo"
	require.NoError(t, manifest.Write(dir, mf))

	// Accept new source for superplan
	err := ApplySourceMismatchAccept(dir, "superplan", "new-owner/skills")
	require.NoError(t, err)

	// Verify only superplan changed
	updated, err := manifest.Read(dir)
	require.NoError(t, err)
	assert.Equal(t, "new-owner/skills", updated.Skills["superplan"])
	assert.Equal(t, "asteroid-belt/skills", updated.Skills["teach"])
	assert.Equal(t, "other-owner/repo", updated.Skills["other"])
}
