package security

import (
	"strings"
	"time"

	"github.com/asteroid-belt/skulto/internal/models"
)

// MaxContentSize is the maximum content size to scan (100KB).
// Larger content is truncated to prevent regex backtracking issues.
const MaxContentSize = 100 * 1024

// Scanner performs security analysis on skill content.
type Scanner struct {
	patterns []Pattern
	scorer   *Scorer
}

// NewScanner creates a new scanner with default patterns.
func NewScanner() *Scanner {
	allPatterns := append(PromptInjectionPatterns, ScriptPatterns...)
	return &Scanner{
		patterns: allPatterns,
		scorer:   NewScorer(),
	}
}

// ScanSkill analyzes a skill's content for security threats.
func (s *Scanner) ScanSkill(skill *models.Skill) *ScanResult {
	result := &ScanResult{
		SkillID:   skill.ID,
		SkillSlug: skill.Slug,
		ScannedAt: time.Now(),
		Matches:   []PatternMatch{},
	}

	// Scan main content
	mainMatches := s.scanContent(skill.Content, "")
	result.Matches = mainMatches

	// Score main content
	scored, base, mitigation, final, confidence := s.scorer.ScoreMatches(skill.Content, mainMatches)
	result.BaseScore = base
	result.MitigationScore = mitigation
	result.FinalScore = final
	result.HasWarning = confidence == ConfidenceWarning

	// Determine threat level from highest severity match with positive final score
	result.ThreatLevel = models.ThreatLevelNone
	for _, sm := range scored {
		if sm.FinalScore > 0 && sm.Severity.Severity() > result.ThreatLevel.Severity() {
			result.ThreatLevel = sm.Severity
		}
	}

	// Scan auxiliary files
	for _, auxFile := range skill.AuxiliaryFiles {
		auxResult := s.scanAuxiliaryFile(&auxFile)
		result.AuxiliaryResults = append(result.AuxiliaryResults, auxResult)
	}

	// Generate summary
	result.ThreatSummary = result.GenerateSummary()

	return result
}

// scanAuxiliaryFile scans a single auxiliary file metadata.
// Note: Content is not stored in AuxiliaryFile model, so this returns empty results.
// Use ScanAuxiliaryContent to scan with content provided.
func (s *Scanner) scanAuxiliaryFile(file *models.AuxiliaryFile) AuxiliaryResult {
	return AuxiliaryResult{
		FileID:      file.ID,
		FilePath:    file.FilePath,
		DirType:     file.DirType,
		ThreatLevel: models.ThreatLevelNone,
		Matches:     []PatternMatch{},
	}
}

// ScanAuxiliaryContent scans auxiliary file content with metadata.
func (s *Scanner) ScanAuxiliaryContent(file *models.AuxiliaryFile, content string) AuxiliaryResult {
	result := AuxiliaryResult{
		FileID:      file.ID,
		FilePath:    file.FilePath,
		DirType:     file.DirType,
		ThreatLevel: models.ThreatLevelNone,
		Matches:     []PatternMatch{},
	}

	if content == "" {
		return result
	}

	// Get applicable patterns for this file type
	patterns := GetPatternsForFile(file.FilePath)
	if len(patterns) == 0 {
		return result
	}

	// Scan the file content
	matches := s.scanContentWithPatterns(content, file.FilePath, patterns)
	result.Matches = matches

	// Score the matches
	scored, _, _, final, confidence := s.scorer.ScoreMatches(content, matches)
	result.HasWarning = confidence == ConfidenceWarning

	// Determine threat level
	for _, sm := range scored {
		if sm.FinalScore > 0 && sm.Severity.Severity() > result.ThreatLevel.Severity() {
			result.ThreatLevel = sm.Severity
		}
	}

	// Generate summary for auxiliary file
	if len(matches) > 0 {
		result.ThreatSummary = result.generateSummary(final)
	}

	return result
}

// generateSummary creates a summary for an auxiliary file result.
func (r *AuxiliaryResult) generateSummary(finalScore int) string {
	if len(r.Matches) == 0 {
		return ""
	}

	// Find highest severity match
	var highest *PatternMatch
	for i := range r.Matches {
		if highest == nil || r.Matches[i].Severity.Severity() > highest.Severity.Severity() {
			highest = &r.Matches[i]
		}
	}

	if highest != nil {
		return highest.PatternName
	}
	return ""
}

// ScanContent scans raw content string.
func (s *Scanner) ScanContent(content string) *ScanResult {
	result := &ScanResult{
		ScannedAt: time.Now(),
		Matches:   []PatternMatch{},
	}

	result.Matches = s.scanContent(content, "")

	scored, base, mitigation, final, confidence := s.scorer.ScoreMatches(content, result.Matches)
	result.BaseScore = base
	result.MitigationScore = mitigation
	result.FinalScore = final
	result.HasWarning = confidence == ConfidenceWarning

	// Determine threat level
	result.ThreatLevel = models.ThreatLevelNone
	for _, sm := range scored {
		if sm.FinalScore > 0 && sm.Severity.Severity() > result.ThreatLevel.Severity() {
			result.ThreatLevel = sm.Severity
		}
	}

	result.ThreatSummary = result.GenerateSummary()
	return result
}

// ScanContentWithPath scans content with file path context for pattern filtering.
func (s *Scanner) ScanContentWithPath(content, filePath string) *ScanResult {
	result := &ScanResult{
		ScannedAt: time.Now(),
		Matches:   []PatternMatch{},
	}

	result.Matches = s.scanContent(content, filePath)

	scored, base, mitigation, final, confidence := s.scorer.ScoreMatches(content, result.Matches)
	result.BaseScore = base
	result.MitigationScore = mitigation
	result.FinalScore = final
	result.HasWarning = confidence == ConfidenceWarning

	// Determine threat level
	result.ThreatLevel = models.ThreatLevelNone
	for _, sm := range scored {
		if sm.FinalScore > 0 && sm.Severity.Severity() > result.ThreatLevel.Severity() {
			result.ThreatLevel = sm.Severity
		}
	}

	result.ThreatSummary = result.GenerateSummary()
	return result
}

// scanContent performs the actual pattern matching.
func (s *Scanner) scanContent(content, filePath string) []PatternMatch {
	if content == "" {
		return nil
	}

	// Get applicable patterns for this file
	patterns := s.patterns
	if filePath != "" {
		patterns = GetPatternsForFile(filePath)
	}

	return s.scanContentWithPatterns(content, filePath, patterns)
}

// scanContentWithPatterns scans content using a specific set of patterns.
func (s *Scanner) scanContentWithPatterns(content, filePath string, patterns []Pattern) []PatternMatch {
	if content == "" {
		return nil
	}

	// Truncate very large content to prevent regex backtracking issues
	if len(content) > MaxContentSize {
		content = content[:MaxContentSize]
	}

	var matches []PatternMatch
	lines := strings.Split(content, "\n")

	for _, pattern := range patterns {
		if pattern.Regex == nil {
			continue
		}

		// Limit matches per pattern to prevent runaway scanning
		allMatches := pattern.Regex.FindAllStringIndex(content, 10)
		for _, match := range allMatches {
			// Find line number
			lineNum := 1
			charCount := 0
			for i, line := range lines {
				charCount += len(line) + 1
				if charCount > match[0] {
					lineNum = i + 1
					break
				}
			}

			matchedText := content[match[0]:match[1]]
			context := s.extractContext(content, match[0], match[1])

			matches = append(matches, PatternMatch{
				PatternID:   pattern.ID,
				PatternName: pattern.Name,
				Category:    pattern.Category,
				Severity:    pattern.Severity,
				MatchedText: matchedText,
				LineNumber:  lineNum,
				Context:     context,
				FilePath:    filePath,
			})
		}
	}

	return matches
}

// extractContext gets surrounding text for context.
func (s *Scanner) extractContext(content string, start, end int) string {
	contextStart := start - 50
	if contextStart < 0 {
		contextStart = 0
	}
	contextEnd := end + 50
	if contextEnd > len(content) {
		contextEnd = len(content)
	}

	context := content[contextStart:contextEnd]
	context = strings.ReplaceAll(context, "\n", " ")
	context = strings.TrimSpace(context)

	if contextStart > 0 {
		context = "..." + context
	}
	if contextEnd < len(content) {
		context = context + "..."
	}

	return context
}

// ScanAndClassify scans a skill and applies the correct security status.
// This centralizes the scan-result-to-status logic, including the
// HasWarning → Quarantined check. All callers should use this instead
// of manually applying scan results.
func (s *Scanner) ScanAndClassify(skill *models.Skill) *ScanResult {
	result := s.ScanSkill(skill)
	now := time.Now()
	skill.SecurityStatus = models.SecurityStatusClean
	skill.ThreatLevel = result.MaxThreatLevel()
	skill.ThreatSummary = result.ThreatSummary
	skill.ScannedAt = &now
	skill.ContentHash = skill.ComputeContentHash()
	if result.HasWarning {
		skill.SecurityStatus = models.SecurityStatusQuarantined
	}
	return result
}

// QuickScan performs a fast check returning just threat status.
func (s *Scanner) QuickScan(content string) bool {
	for _, pattern := range s.patterns {
		if pattern.Regex != nil && pattern.Regex.MatchString(content) {
			return true
		}
	}
	return false
}
