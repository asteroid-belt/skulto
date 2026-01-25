package security

import (
	"fmt"
	"time"

	"github.com/asteroid-belt/skulto/internal/models"
)

// ThreatCategory represents a category of security threat.
type ThreatCategory string

const (
	CategoryInstructionOverride ThreatCategory = "instruction_override"
	CategoryJailbreak           ThreatCategory = "jailbreak"
	CategorySystemSpoofing      ThreatCategory = "system_spoofing"
	CategoryDataExfiltration    ThreatCategory = "data_exfiltration"
	CategoryObfuscation         ThreatCategory = "obfuscation"
	CategoryAgentManipulation   ThreatCategory = "agent_manipulation"
	CategoryPrivilegeEscalation ThreatCategory = "privilege_escalation"
	CategoryMultiTurnErosion    ThreatCategory = "multi_turn_erosion"
	CategoryScriptDanger        ThreatCategory = "script_danger"
)

// ScanResult represents the outcome of scanning a skill.
type ScanResult struct {
	SkillID   string
	SkillSlug string
	ScannedAt time.Time

	// Overall assessment
	HasWarning    bool
	ThreatLevel   models.ThreatLevel
	ThreatSummary string

	// Detailed matches
	Matches []PatternMatch

	// Scoring breakdown
	BaseScore       int
	MitigationScore int
	FinalScore      int

	// Auxiliary file results
	AuxiliaryResults []AuxiliaryResult
}

// AuxiliaryResult represents scan result for a single auxiliary file.
type AuxiliaryResult struct {
	FileID        string
	FilePath      string
	DirType       models.AuxiliaryDirType
	HasWarning    bool
	ThreatLevel   models.ThreatLevel
	ThreatSummary string
	Matches       []PatternMatch
}

// PatternMatch represents a single pattern that matched.
type PatternMatch struct {
	PatternID   string
	PatternName string
	Category    ThreatCategory
	Severity    models.ThreatLevel
	MatchedText string
	LineNumber  int
	Context     string // Surrounding text for review
	FilePath    string // Empty for main content, path for aux files
}

// MaxThreatLevel returns the highest threat level across main and aux files.
func (r *ScanResult) MaxThreatLevel() models.ThreatLevel {
	max := r.ThreatLevel
	for _, aux := range r.AuxiliaryResults {
		if aux.ThreatLevel.Severity() > max.Severity() {
			max = aux.ThreatLevel
		}
	}
	return max
}

// TotalMatchCount returns total matches across all files.
func (r *ScanResult) TotalMatchCount() int {
	count := len(r.Matches)
	for _, aux := range r.AuxiliaryResults {
		count += len(aux.Matches)
	}
	return count
}

// GenerateSummary creates a human-readable summary.
func (r *ScanResult) GenerateSummary() string {
	if !r.HasWarning && len(r.AuxiliaryResults) == 0 {
		return "No threats detected"
	}

	totalMatches := r.TotalMatchCount()
	if totalMatches == 0 {
		return "No threats detected"
	}

	// Find highest severity pattern for display
	var highestMatch *PatternMatch
	for i := range r.Matches {
		if highestMatch == nil || r.Matches[i].Severity.Severity() > highestMatch.Severity.Severity() {
			highestMatch = &r.Matches[i]
		}
	}
	for _, aux := range r.AuxiliaryResults {
		for i := range aux.Matches {
			if highestMatch == nil || aux.Matches[i].Severity.Severity() > highestMatch.Severity.Severity() {
				highestMatch = &aux.Matches[i]
			}
		}
	}

	if highestMatch != nil {
		return fmt.Sprintf("Detected: %s (%d total patterns)", highestMatch.PatternName, totalMatches)
	}

	return fmt.Sprintf("Detected %d potential threat patterns", totalMatches)
}
