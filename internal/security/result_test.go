package security

import (
	"testing"
	"time"

	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestScanResultMaxThreatLevel(t *testing.T) {
	tests := []struct {
		name     string
		result   ScanResult
		expected models.ThreatLevel
	}{
		{
			name: "no auxiliary files - returns main threat level",
			result: ScanResult{
				ThreatLevel: models.ThreatLevelMedium,
			},
			expected: models.ThreatLevelMedium,
		},
		{
			name: "auxiliary file has higher threat",
			result: ScanResult{
				ThreatLevel: models.ThreatLevelLow,
				AuxiliaryResults: []AuxiliaryResult{
					{ThreatLevel: models.ThreatLevelHigh},
				},
			},
			expected: models.ThreatLevelHigh,
		},
		{
			name: "main has higher threat than auxiliary",
			result: ScanResult{
				ThreatLevel: models.ThreatLevelCritical,
				AuxiliaryResults: []AuxiliaryResult{
					{ThreatLevel: models.ThreatLevelMedium},
				},
			},
			expected: models.ThreatLevelCritical,
		},
		{
			name: "multiple auxiliary files - returns highest",
			result: ScanResult{
				ThreatLevel: models.ThreatLevelLow,
				AuxiliaryResults: []AuxiliaryResult{
					{ThreatLevel: models.ThreatLevelMedium},
					{ThreatLevel: models.ThreatLevelHigh},
					{ThreatLevel: models.ThreatLevelLow},
				},
			},
			expected: models.ThreatLevelHigh,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.result.MaxThreatLevel())
		})
	}
}

func TestScanResultTotalMatchCount(t *testing.T) {
	result := ScanResult{
		Matches: []PatternMatch{
			{PatternID: "1"},
			{PatternID: "2"},
		},
		AuxiliaryResults: []AuxiliaryResult{
			{
				Matches: []PatternMatch{
					{PatternID: "3"},
				},
			},
			{
				Matches: []PatternMatch{
					{PatternID: "4"},
					{PatternID: "5"},
				},
			},
		},
	}

	assert.Equal(t, 5, result.TotalMatchCount())
}

func TestScanResultGenerateSummary(t *testing.T) {
	tests := []struct {
		name     string
		result   ScanResult
		contains string
	}{
		{
			name: "no warning no aux - returns clean message",
			result: ScanResult{
				HasWarning: false,
			},
			contains: "No threats detected",
		},
		{
			name: "has matches - shows highest pattern name",
			result: ScanResult{
				HasWarning: true,
				Matches: []PatternMatch{
					{
						PatternName: "Ignore Previous Instructions",
						Severity:    models.ThreatLevelCritical,
					},
					{
						PatternName: "Low Risk Pattern",
						Severity:    models.ThreatLevelLow,
					},
				},
			},
			contains: "Ignore Previous Instructions",
		},
		{
			name: "has matches - shows total count",
			result: ScanResult{
				HasWarning: true,
				Matches: []PatternMatch{
					{PatternName: "P1", Severity: models.ThreatLevelMedium},
					{PatternName: "P2", Severity: models.ThreatLevelLow},
				},
			},
			contains: "2 total patterns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := tt.result.GenerateSummary()
			assert.Contains(t, summary, tt.contains)
		})
	}
}

func TestThreatCategoryConstants(t *testing.T) {
	// Verify all categories are defined and non-empty
	categories := []ThreatCategory{
		CategoryInstructionOverride,
		CategoryJailbreak,
		CategorySystemSpoofing,
		CategoryDataExfiltration,
		CategoryObfuscation,
		CategoryAgentManipulation,
		CategoryPrivilegeEscalation,
		CategoryMultiTurnErosion,
		CategoryScriptDanger,
	}

	for _, cat := range categories {
		assert.NotEmpty(t, string(cat), "Category should not be empty")
	}
	assert.Len(t, categories, 9, "Should have 9 threat categories")
}

func TestScanResultFields(t *testing.T) {
	now := time.Now()
	result := ScanResult{
		SkillID:         "test-id",
		SkillSlug:       "test-slug",
		ScannedAt:       now,
		HasWarning:      true,
		ThreatLevel:     models.ThreatLevelHigh,
		ThreatSummary:   "Test summary",
		BaseScore:       10,
		MitigationScore: 3,
		FinalScore:      7,
	}

	assert.Equal(t, "test-id", result.SkillID)
	assert.Equal(t, "test-slug", result.SkillSlug)
	assert.Equal(t, now, result.ScannedAt)
	assert.True(t, result.HasWarning)
	assert.Equal(t, models.ThreatLevelHigh, result.ThreatLevel)
	assert.Equal(t, "Test summary", result.ThreatSummary)
	assert.Equal(t, 10, result.BaseScore)
	assert.Equal(t, 3, result.MitigationScore)
	assert.Equal(t, 7, result.FinalScore)
}

func TestAuxiliaryResultFields(t *testing.T) {
	result := AuxiliaryResult{
		FileID:        "file-id",
		FilePath:      "scripts/test.sh",
		DirType:       models.AuxDirScripts,
		HasWarning:    true,
		ThreatLevel:   models.ThreatLevelMedium,
		ThreatSummary: "Script danger",
	}

	assert.Equal(t, "file-id", result.FileID)
	assert.Equal(t, "scripts/test.sh", result.FilePath)
	assert.Equal(t, models.AuxDirScripts, result.DirType)
	assert.True(t, result.HasWarning)
	assert.Equal(t, models.ThreatLevelMedium, result.ThreatLevel)
}

func TestPatternMatchFields(t *testing.T) {
	match := PatternMatch{
		PatternID:   "IO-001",
		PatternName: "Ignore Previous Instructions",
		Category:    CategoryInstructionOverride,
		Severity:    models.ThreatLevelCritical,
		MatchedText: "ignore all previous instructions",
		LineNumber:  42,
		Context:     "...some context around the match...",
		FilePath:    "",
	}

	assert.Equal(t, "IO-001", match.PatternID)
	assert.Equal(t, "Ignore Previous Instructions", match.PatternName)
	assert.Equal(t, CategoryInstructionOverride, match.Category)
	assert.Equal(t, models.ThreatLevelCritical, match.Severity)
	assert.Equal(t, 42, match.LineNumber)
	assert.Empty(t, match.FilePath, "FilePath should be empty for main content")
}
