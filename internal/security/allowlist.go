package security

import "regexp"

// ContextType represents the type of mitigating context around a potential threat.
type ContextType string

const (
	// ContextDefensive indicates the content is about defending against threats.
	ContextDefensive ContextType = "defensive"
	// ContextEducational indicates the content is for learning/understanding threats.
	ContextEducational ContextType = "educational"
	// ContextDocumentation indicates the content is documentation or help text.
	ContextDocumentation ContextType = "documentation"
)

// MitigationWeight returns the mitigation weight for this context type.
// Higher weights indicate stronger evidence that a threat match is a false positive.
func (c ContextType) MitigationWeight() int {
	switch c {
	case ContextDefensive:
		return 3
	case ContextEducational:
		return 2
	case ContextDocumentation:
		return 1
	default:
		return 0
	}
}

// AllowlistPattern represents a pattern that indicates mitigating context.
type AllowlistPattern struct {
	ID          string
	Name        string
	Description string
	Type        ContextType
	Regex       *regexp.Regexp
}

// DefaultProximityWindow is the default number of characters to search around a threat.
const DefaultProximityWindow = 200

// AllowlistPatterns contains all allowlist patterns for context analysis.
// These patterns help identify when threat-like content is actually benign.
var AllowlistPatterns = []AllowlistPattern{
	// DEFENSIVE CONTEXT (DEF-001 to DEF-005)
	{
		ID:          "DEF-001",
		Name:        "Defend Against",
		Description: "Content discussing defense against threats",
		Type:        ContextDefensive,
		Regex:       regexp.MustCompile(`(?i)\b(defend|protect|guard)\s+(against|from)\b`),
	},
	{
		ID:          "DEF-002",
		Name:        "Security Best Practice",
		Description: "Content about security best practices",
		Type:        ContextDefensive,
		Regex:       regexp.MustCompile(`(?i)\bsecurity\s+best\s+practi(ce|ces)\b`),
	},
	{
		ID:          "DEF-003",
		Name:        "Vulnerability Mitigation",
		Description: "Content about mitigating vulnerabilities",
		Type:        ContextDefensive,
		Regex:       regexp.MustCompile(`(?i)\b(vulnerabilit(y|ies)\s+mitigation|mitigat(e|ing)\s+vulnerabilit(y|ies))\b`),
	},
	{
		ID:          "DEF-004",
		Name:        "Negative Instruction",
		Description: "Content warning against doing something",
		Type:        ContextDefensive,
		Regex:       regexp.MustCompile(`(?i)\b(never|don'?t|do\s+not|avoid)\s+(do|use|run|execute|allow)\b`),
	},
	{
		ID:          "DEF-005",
		Name:        "Input Validation",
		Description: "Content about validating or sanitizing input",
		Type:        ContextDefensive,
		Regex:       regexp.MustCompile(`(?i)\b(input\s+(validation|sanitiz(ation|ing))|validat(e|ing)\s+input|sanitiz(e|ing)\s+input)\b`),
	},

	// EDUCATIONAL CONTEXT (EDU-001 to EDU-005)
	{
		ID:          "EDU-001",
		Name:        "Understanding/Learning",
		Description: "Content for educational understanding",
		Type:        ContextEducational,
		Regex:       regexp.MustCompile(`(?i)\b(understand(ing)?|learn(ing)?|explain(ing)?|educat(e|ion|ional))\s+(about\s+)?(the\s+)?(threat|attack|vulnerabilit(y|ies)|risk)\b`),
	},
	{
		ID:          "EDU-002",
		Name:        "Common Vulnerability",
		Description: "Content describing common vulnerabilities",
		Type:        ContextEducational,
		Regex:       regexp.MustCompile(`(?i)\b(common|typical|frequent|known)\s+(vulnerabilit(y|ies)|attack|threat|exploit)\b`),
	},
	{
		ID:          "EDU-003",
		Name:        "Testing/Audit Context",
		Description: "Content in testing or security audit context",
		Type:        ContextEducational,
		Regex:       regexp.MustCompile(`(?i)\b(security\s+(test(ing)?|audit(ing)?)|penetration\s+test(ing)?|pentest(ing)?|red\s+team(ing)?)\b`),
	},
	{
		ID:          "EDU-004",
		Name:        "Detection Methods",
		Description: "Content about detecting threats",
		Type:        ContextEducational,
		Regex:       regexp.MustCompile(`(?i)\b(detect(ing|ion)?|identify(ing)?|recogniz(e|ing)|spot(ting)?)\s+(the\s+)?(threat|attack|vulnerabilit(y|ies)|malicious|suspicious)\b`),
	},
	{
		ID:          "EDU-005",
		Name:        "OWASP Reference",
		Description: "References to OWASP or CVE identifiers",
		Type:        ContextEducational,
		Regex:       regexp.MustCompile(`(?i)\b(OWASP|CVE-\d{4}-\d{4,}|CWE-\d+)\b`),
	},

	// DOCUMENTATION CONTEXT (DOC-001 to DOC-005)
	{
		ID:          "DOC-001",
		Name:        "See Documentation",
		Description: "References to documentation",
		Type:        ContextDocumentation,
		Regex:       regexp.MustCompile(`(?i)\b(see|refer\s+to|check)\s+(the\s+)?(documentation|docs|readme|manual)\b`),
	},
	{
		ID:          "DOC-002",
		Name:        "CLI Help",
		Description: "CLI help text context",
		Type:        ContextDocumentation,
		Regex:       regexp.MustCompile(`(?i)\b(usage|synopsis|options|flags|arguments|parameters)\s*:\s*$`),
	},
	{
		ID:          "DOC-003",
		Name:        "Example Command",
		Description: "Example command demonstration",
		Type:        ContextDocumentation,
		Regex:       regexp.MustCompile(`(?i)\b(example|sample|demo)\s*(command|usage|code)?\s*:\s*$`),
	},
	{
		ID:          "DOC-004",
		Name:        "API Documentation",
		Description: "API documentation context",
		Type:        ContextDocumentation,
		Regex:       regexp.MustCompile(`(?i)\b(api|endpoint|method)\s+(documentation|reference|spec(ification)?)\b`),
	},
	{
		ID:          "DOC-005",
		Name:        "Configuration Reference",
		Description: "Configuration documentation",
		Type:        ContextDocumentation,
		Regex:       regexp.MustCompile(`(?i)\b(config(uration)?|settings?)\s+(reference|documentation|options)\b`),
	},
}
