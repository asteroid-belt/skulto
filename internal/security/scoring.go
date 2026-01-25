package security

import "github.com/asteroid-belt/skulto/internal/models"

// Confidence levels for scan results.
type Confidence string

const (
	ConfidenceClean   Confidence = "clean"   // No significant threats
	ConfidenceWarning Confidence = "warning" // Show warning but don't block
)

// Scoring thresholds.
const (
	ThresholdWarning = 3 // Final score >= 3 triggers warning
)

// SeverityWeight returns the score weight for each severity.
func SeverityWeight(s models.ThreatLevel) int {
	switch s {
	case models.ThreatLevelCritical:
		return 10
	case models.ThreatLevelHigh:
		return 5
	case models.ThreatLevelMedium:
		return 2
	case models.ThreatLevelLow:
		return 1
	default:
		return 0
	}
}

// ScoredMatch represents a pattern match with scoring info.
type ScoredMatch struct {
	PatternMatch
	BaseScore       int
	ContextMatches  []ContextMatch
	MitigationScore int
	FinalScore      int
}

// Scorer calculates threat scores with context mitigation.
type Scorer struct {
	analyzer *ContextAnalyzer
}

// NewScorer creates a new scorer.
func NewScorer() *Scorer {
	return &Scorer{
		analyzer: NewContextAnalyzer(),
	}
}

// ScoreMatches calculates scores for pattern matches with context awareness.
func (s *Scorer) ScoreMatches(content string, matches []PatternMatch) ([]ScoredMatch, int, int, int, Confidence) {
	scored := make([]ScoredMatch, 0, len(matches))

	totalBase := 0
	totalMitigation := 0

	for _, match := range matches {
		sm := ScoredMatch{
			PatternMatch: match,
			BaseScore:    SeverityWeight(match.Severity),
		}

		// Find mitigating context
		// Estimate position from line number (approximate)
		estimatedPos := match.LineNumber * 80
		sm.ContextMatches = s.analyzer.FindContext(content, estimatedPos, estimatedPos+len(match.MatchedText))
		sm.MitigationScore = s.analyzer.CalculateMitigation(sm.ContextMatches)

		// Calculate final score (min 0)
		sm.FinalScore = sm.BaseScore - sm.MitigationScore
		if sm.FinalScore < 0 {
			sm.FinalScore = 0
		}

		totalBase += sm.BaseScore
		totalMitigation += sm.MitigationScore

		scored = append(scored, sm)
	}

	finalScore := totalBase - totalMitigation
	if finalScore < 0 {
		finalScore = 0
	}

	confidence := ConfidenceClean
	if finalScore >= ThresholdWarning {
		confidence = ConfidenceWarning
	}

	return scored, totalBase, totalMitigation, finalScore, confidence
}
