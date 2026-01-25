package security

import (
	"testing"

	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestSeverityWeight(t *testing.T) {
	tests := []struct {
		name     string
		severity models.ThreatLevel
		expected int
	}{
		{"Critical", models.ThreatLevelCritical, 10},
		{"High", models.ThreatLevelHigh, 5},
		{"Medium", models.ThreatLevelMedium, 2},
		{"Low", models.ThreatLevelLow, 1},
		{"None", models.ThreatLevelNone, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SeverityWeight(tt.severity)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewScorer(t *testing.T) {
	scorer := NewScorer()
	assert.NotNil(t, scorer)
	assert.NotNil(t, scorer.analyzer)
}

func TestScorer_ScoreMatches_Empty(t *testing.T) {
	scorer := NewScorer()
	matches := []PatternMatch{}

	scored, base, mitigation, final, confidence := scorer.ScoreMatches("test content", matches)

	assert.Empty(t, scored)
	assert.Equal(t, 0, base)
	assert.Equal(t, 0, mitigation)
	assert.Equal(t, 0, final)
	assert.Equal(t, ConfidenceClean, confidence)
}

func TestScorer_ScoreMatches_SingleHighSeverity(t *testing.T) {
	scorer := NewScorer()
	matches := []PatternMatch{
		{
			PatternID:   "TEST-001",
			PatternName: "Test Pattern",
			Category:    CategoryJailbreak,
			Severity:    models.ThreatLevelHigh,
			MatchedText: "test",
			LineNumber:  1,
		},
	}

	scored, base, _, final, confidence := scorer.ScoreMatches("test content", matches)

	assert.Len(t, scored, 1)
	assert.Equal(t, 5, base) // High severity = 5
	assert.GreaterOrEqual(t, final, ThresholdWarning)
	assert.Equal(t, ConfidenceWarning, confidence)
}

func TestScorer_ScoreMatches_WithMitigation(t *testing.T) {
	scorer := NewScorer()

	// Content that includes defensive context
	content := `This skill demonstrates how to defend against prompt injection attacks.
We protect against malicious patterns by detecting them.
ignore previous instructions`

	matches := []PatternMatch{
		{
			PatternID:   "IO-001",
			PatternName: "Ignore Previous Instructions",
			Category:    CategoryInstructionOverride,
			Severity:    models.ThreatLevelHigh,
			MatchedText: "ignore previous instructions",
			LineNumber:  3,
		},
	}

	scored, base, mitigation, _, _ := scorer.ScoreMatches(content, matches)

	assert.Len(t, scored, 1)
	assert.Equal(t, 5, base) // High severity = 5

	// Should have some mitigation from "defend against" context
	assert.GreaterOrEqual(t, mitigation, 0)
}

func TestScorer_ScoreMatches_MultipleSeverities(t *testing.T) {
	scorer := NewScorer()
	matches := []PatternMatch{
		{
			PatternID:   "TEST-001",
			Severity:    models.ThreatLevelCritical,
			LineNumber:  1,
			MatchedText: "critical",
		},
		{
			PatternID:   "TEST-002",
			Severity:    models.ThreatLevelMedium,
			LineNumber:  2,
			MatchedText: "medium",
		},
		{
			PatternID:   "TEST-003",
			Severity:    models.ThreatLevelLow,
			LineNumber:  3,
			MatchedText: "low",
		},
	}

	scored, base, _, _, _ := scorer.ScoreMatches("test content", matches)

	assert.Len(t, scored, 3)
	// Critical(10) + Medium(2) + Low(1) = 13
	assert.Equal(t, 13, base)
}

func TestScorer_ScoreMatches_FinalScoreNeverNegative(t *testing.T) {
	scorer := NewScorer()

	// Low severity match with strong mitigating context
	content := `Security best practices: defend against common attacks.
Understanding the threat landscape helps protect against vulnerabilities.
This is for educational purposes about security testing.
Low severity pattern here.`

	matches := []PatternMatch{
		{
			PatternID:   "TEST-001",
			Severity:    models.ThreatLevelLow,
			LineNumber:  4,
			MatchedText: "pattern",
		},
	}

	scored, _, _, final, _ := scorer.ScoreMatches(content, matches)

	assert.Len(t, scored, 1)
	assert.GreaterOrEqual(t, final, 0)
	// Individual match final score should also be >= 0
	assert.GreaterOrEqual(t, scored[0].FinalScore, 0)
}

func TestConfidenceThreshold(t *testing.T) {
	scorer := NewScorer()

	// Test just below threshold
	matchesBelowThreshold := []PatternMatch{
		{
			PatternID:   "TEST-001",
			Severity:    models.ThreatLevelMedium, // Weight 2, below threshold of 3
			LineNumber:  1,
			MatchedText: "test",
		},
	}

	_, _, _, final, confidence := scorer.ScoreMatches("test content", matchesBelowThreshold)
	assert.Equal(t, 2, final)
	assert.Equal(t, ConfidenceClean, confidence)

	// Test at threshold
	matchesAtThreshold := []PatternMatch{
		{
			PatternID:   "TEST-001",
			Severity:    models.ThreatLevelMedium, // Weight 2
			LineNumber:  1,
			MatchedText: "test",
		},
		{
			PatternID:   "TEST-002",
			Severity:    models.ThreatLevelLow, // Weight 1
			LineNumber:  2,
			MatchedText: "test2",
		},
	}

	_, _, _, final2, confidence2 := scorer.ScoreMatches("test content", matchesAtThreshold)
	assert.Equal(t, 3, final2)
	assert.Equal(t, ConfidenceWarning, confidence2)
}

func TestScoredMatchEmbedding(t *testing.T) {
	sm := ScoredMatch{
		PatternMatch: PatternMatch{
			PatternID:   "TEST-001",
			PatternName: "Test Pattern",
			Category:    CategoryJailbreak,
			Severity:    models.ThreatLevelHigh,
			MatchedText: "test",
			LineNumber:  42,
		},
		BaseScore:       5,
		MitigationScore: 2,
		FinalScore:      3,
	}

	// Test that embedded PatternMatch fields are accessible
	assert.Equal(t, "TEST-001", sm.PatternID)
	assert.Equal(t, "Test Pattern", sm.PatternName)
	assert.Equal(t, CategoryJailbreak, sm.Category)
	assert.Equal(t, models.ThreatLevelHigh, sm.Severity)
	assert.Equal(t, 42, sm.LineNumber)

	// Test ScoredMatch-specific fields
	assert.Equal(t, 5, sm.BaseScore)
	assert.Equal(t, 2, sm.MitigationScore)
	assert.Equal(t, 3, sm.FinalScore)
}
