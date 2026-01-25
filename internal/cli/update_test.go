package cli

import (
	"testing"

	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestUpdateCmd_Structure(t *testing.T) {
	assert.Equal(t, "update", updateCmd.Use)
	assert.NotEmpty(t, updateCmd.Short)
	assert.NotEmpty(t, updateCmd.Long)
	assert.Contains(t, updateCmd.Long, "Update pulls all registered skill repositories")
	assert.Contains(t, updateCmd.Long, "skulto update")
}

func TestUpdateCmd_ArgsValidation(t *testing.T) {
	// Test the cobra.NoArgs validator
	validator := cobra.NoArgs

	// Should pass with no args
	err := validator(updateCmd, []string{})
	assert.NoError(t, err)

	// Should fail with any args
	err = validator(updateCmd, []string{"unexpected"})
	assert.Error(t, err)
}

func TestUpdateCmd_ScanAllFlag(t *testing.T) {
	flag := updateCmd.Flags().Lookup("scan-all")
	assert.NotNil(t, flag)
	assert.Equal(t, "false", flag.DefValue)
}

func TestUpdateResult_Initialization(t *testing.T) {
	result := &UpdateResult{}

	assert.Equal(t, 0, result.ReposSynced)
	assert.Equal(t, 0, result.ReposErrored)
	assert.Equal(t, 0, result.SkillsNew)
	assert.Equal(t, 0, result.SkillsUpdated)
	assert.Equal(t, 0, result.SkillsScanned)
	assert.Empty(t, result.UpdatedSkills)
	assert.Empty(t, result.Changes)
}

func TestUpdateResult_ThreatCounts(t *testing.T) {
	result := &UpdateResult{
		ThreatsCritical: 1,
		ThreatsHigh:     2,
		ThreatsMedium:   3,
		ThreatsLow:      4,
		SkillsClean:     10,
	}

	totalThreats := result.ThreatsCritical + result.ThreatsHigh +
		result.ThreatsMedium + result.ThreatsLow
	assert.Equal(t, 10, totalThreats)
	assert.Equal(t, 10, result.SkillsClean)
}

func TestSkillChange_ChangeTypes(t *testing.T) {
	change := SkillChange{
		Skill:      models.Skill{Title: "Test Skill"},
		ChangeType: "new",
	}

	assert.Equal(t, "new", change.ChangeType)
	assert.Equal(t, "Test Skill", change.Skill.Title)

	change.ChangeType = "updated"
	assert.Equal(t, "updated", change.ChangeType)
}

func TestGetThreatIndicator(t *testing.T) {
	tests := []struct {
		level    models.ThreatLevel
		expected string
	}{
		{models.ThreatLevelNone, ""},
		{models.ThreatLevelLow, " [LOW]"},
		{models.ThreatLevelMedium, " [MEDIUM]"},
		{models.ThreatLevelHigh, " [HIGH]"},
		{models.ThreatLevelCritical, " [CRITICAL]"},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			result := getThreatIndicator(tt.level)
			// The result includes ANSI escape codes for colors, so we just check
			// that it contains the expected text (or is empty for NONE)
			if tt.expected == "" {
				assert.Empty(t, result)
			} else {
				assert.Contains(t, result, tt.expected[1:len(tt.expected)-1]) // Remove brackets for contains check
			}
		})
	}
}

func TestProgressBar_Initialization(t *testing.T) {
	pb := NewProgressBar(10, 15)
	assert.NotNil(t, pb)
	assert.Equal(t, 10, pb.total)
	assert.Equal(t, 15, pb.width)
	assert.Equal(t, 0, pb.completed)
	assert.Empty(t, pb.label)
}

func TestProgressBar_DefaultWidth(t *testing.T) {
	pb := NewProgressBar(10, 0)
	assert.Equal(t, 15, pb.width)

	pb = NewProgressBar(10, -5)
	assert.Equal(t, 15, pb.width)
}

func TestProgressBar_Update(t *testing.T) {
	pb := NewProgressBar(10, 15)
	pb.Update(5, "test-label")

	assert.Equal(t, 5, pb.completed)
	assert.Equal(t, "test-label", pb.label)
}

func TestProgressBar_RenderEmpty(t *testing.T) {
	pb := NewProgressBar(0, 15)
	result := pb.Render()
	assert.Empty(t, result)
}

func TestProgressBar_Render(t *testing.T) {
	pb := NewProgressBar(10, 10)
	pb.Update(5, "test")

	result := pb.Render()
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "5/10")
	assert.Contains(t, result, "test")
}

func TestProgressBar_RenderScanEmpty(t *testing.T) {
	pb := NewProgressBar(0, 15)
	result := pb.RenderScan()
	assert.Empty(t, result)
}

func TestProgressBar_RenderScan(t *testing.T) {
	pb := NewProgressBar(10, 10)
	pb.Update(8, "scanning")

	result := pb.RenderScan()
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "8/10")
	assert.Contains(t, result, "scanning")
}
