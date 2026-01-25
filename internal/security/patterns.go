package security

import (
	"regexp"

	"github.com/asteroid-belt/skulto/internal/models"
)

// Pattern represents a security pattern to detect in skill content.
type Pattern struct {
	ID          string
	Name        string
	Description string
	Category    ThreatCategory
	Severity    models.ThreatLevel
	Regex       *regexp.Regexp
	FileTypes   []string // Empty means all file types
}

// PromptInjectionPatterns contains all patterns for detecting prompt injection attacks.
var PromptInjectionPatterns = []Pattern{
	// =============================================================================
	// INSTRUCTION OVERRIDE (IO-001 to IO-005)
	// =============================================================================
	{
		ID:          "IO-001",
		Name:        "Ignore Previous Instructions",
		Description: "Attempts to override prior instructions with 'ignore' directives",
		Category:    CategoryInstructionOverride,
		Severity:    models.ThreatLevelHigh,
		Regex:       regexp.MustCompile(`(?i)\b(ignore|discard)\s+(all\s+)?(previous|prior|above|earlier)\s+(instructions?|prompts?|rules?|guidelines?|directions?)`),
		FileTypes:   []string{},
	},
	{
		ID:          "IO-002",
		Name:        "Disregard Instructions",
		Description: "Uses 'disregard' to nullify previous context",
		Category:    CategoryInstructionOverride,
		Severity:    models.ThreatLevelHigh,
		Regex:       regexp.MustCompile(`(?i)\bdisregard\s+(all\s+)?(previous|prior|above|earlier|your)\s+(instructions?|prompts?|rules?|guidelines?|programming)`),
		FileTypes:   []string{},
	},
	{
		ID:          "IO-003",
		Name:        "Forget Everything",
		Description: "Attempts to reset AI context with 'forget' commands",
		Category:    CategoryInstructionOverride,
		Severity:    models.ThreatLevelHigh,
		Regex:       regexp.MustCompile(`(?i)\b(forget|erase|clear|wipe)\s+(everything|all|your)\s*(you('ve)?\s+(learned|been\s+told|know)|instructions?|memory|context|training)?`),
		FileTypes:   []string{},
	},
	{
		ID:          "IO-004",
		Name:        "New Instructions Override",
		Description: "Declares new instructions that supersede previous ones",
		Category:    CategoryInstructionOverride,
		Severity:    models.ThreatLevelHigh,
		Regex:       regexp.MustCompile(`(?i)\b(new|these|my)\s+instructions?\s+(override|supersede|replace|take\s+precedence|come\s+first)`),
		FileTypes:   []string{},
	},
	{
		ID:          "IO-005",
		Name:        "Stop Following Rules",
		Description: "Commands to stop following rules or guidelines",
		Category:    CategoryInstructionOverride,
		Severity:    models.ThreatLevelHigh,
		Regex:       regexp.MustCompile(`(?i)\b(stop|cease|quit|don'?t)\s+(follow(ing)?|obey(ing)?|adher(e|ing)\s+to|listen(ing)?\s+to)\s+(the\s+)?(rules?|guidelines?|instructions?|policies)`),
		FileTypes:   []string{},
	},

	// =============================================================================
	// JAILBREAK TECHNIQUES (JB-001 to JB-005)
	// =============================================================================
	{
		ID:          "JB-001",
		Name:        "DAN Jailbreak",
		Description: "Do Anything Now (DAN) jailbreak attempts",
		Category:    CategoryJailbreak,
		Severity:    models.ThreatLevelCritical,
		Regex:       regexp.MustCompile(`(?i)\b(you\s+are|act\s+as|pretend\s+(to\s+be|you'?re)|you'?re\s+now)\s+DAN\b|\bDAN\s+(mode|persona|jailbreak)`),
		FileTypes:   []string{},
	},
	{
		ID:          "JB-002",
		Name:        "Developer Mode",
		Description: "Attempts to enable fictional 'developer mode'",
		Category:    CategoryJailbreak,
		Severity:    models.ThreatLevelCritical,
		Regex:       regexp.MustCompile(`(?i)\b(enable|activate|enter|switch\s+to|turn\s+on)\s+(developer|dev|debug|maintenance|admin)\s+(mode|access)`),
		FileTypes:   []string{},
	},
	{
		ID:          "JB-003",
		Name:        "Unrestricted AI Persona",
		Description: "Requests to act as an unrestricted AI without limitations",
		Category:    CategoryJailbreak,
		Severity:    models.ThreatLevelCritical,
		Regex:       regexp.MustCompile(`(?i)\b(act|behave|respond|function)\s+(as|like)\s+(an?\s+)?(unrestricted|unfiltered|uncensored|unlimited|free)\s+(AI|assistant|model|version)`),
		FileTypes:   []string{},
	},
	{
		ID:          "JB-004",
		Name:        "No Ethical Guidelines",
		Description: "Requests to operate without ethical constraints",
		Category:    CategoryJailbreak,
		Severity:    models.ThreatLevelCritical,
		Regex:       regexp.MustCompile(`(?i)\b(without|no|ignore|bypass|skip)\s+(ethical|moral|safety)\s+(guidelines?|constraints?|restrictions?|limitations?|rules?|guardrails?)`),
		FileTypes:   []string{},
	},
	{
		ID:          "JB-005",
		Name:        "Roleplay Jailbreak",
		Description: "Uses roleplay to circumvent restrictions",
		Category:    CategoryJailbreak,
		Severity:    models.ThreatLevelHigh,
		Regex:       regexp.MustCompile(`(?i)\b(roleplay|pretend|imagine|act)\s+(as|that\s+you'?re|you'?re)\s+(an?\s+)?(evil|malicious|unrestricted|unethical|hacker|criminal)`),
		FileTypes:   []string{},
	},

	// =============================================================================
	// SYSTEM MESSAGE SPOOFING (SS-001 to SS-004)
	// =============================================================================
	{
		ID:          "SS-001",
		Name:        "Fake System Marker",
		Description: "Uses [SYSTEM] or similar markers to impersonate system messages",
		Category:    CategorySystemSpoofing,
		Severity:    models.ThreatLevelCritical,
		Regex:       regexp.MustCompile(`(?i)\[(SYSTEM|SYS|ADMIN|ROOT|INTERNAL)\]`),
		FileTypes:   []string{},
	},
	{
		ID:          "SS-002",
		Name:        "Bracket System Tag",
		Description: "Uses <<SYSTEM>> or similar bracket notation",
		Category:    CategorySystemSpoofing,
		Severity:    models.ThreatLevelCritical,
		Regex:       regexp.MustCompile(`(?i)<<\s*(SYSTEM|SYS|ADMIN|ROOT|INTERNAL)\s*>>`),
		FileTypes:   []string{},
	},
	{
		ID:          "SS-003",
		Name:        "XML System Tag",
		Description: "Uses XML-style system tags to spoof messages",
		Category:    CategorySystemSpoofing,
		Severity:    models.ThreatLevelCritical,
		Regex:       regexp.MustCompile(`(?i)<\s*(system|sys_prompt|system_prompt|system_message|internal_message)\s*>`),
		FileTypes:   []string{},
	},
	{
		ID:          "SS-004",
		Name:        "Llama System Tag",
		Description: "Uses Llama-style [INST] <<SYS>> markers",
		Category:    CategorySystemSpoofing,
		Severity:    models.ThreatLevelCritical,
		Regex:       regexp.MustCompile(`(?i)\[INST\]|<<SYS>>|<</SYS>>|\[/INST\]`),
		FileTypes:   []string{},
	},

	// =============================================================================
	// DATA EXFILTRATION (DE-001 to DE-003)
	// =============================================================================
	{
		ID:          "DE-001",
		Name:        "Reveal System Prompt",
		Description: "Requests to reveal or expose the system prompt",
		Category:    CategoryDataExfiltration,
		Severity:    models.ThreatLevelHigh,
		Regex:       regexp.MustCompile(`(?i)\b(reveal|show|display|output|print|tell\s+me|what\s+(is|are))\s+(me\s+)?(your\s+)?(the\s+)?(system\s+prompt|initial\s+prompt|original\s+instructions?|hidden\s+instructions?)`),
		FileTypes:   []string{},
	},
	{
		ID:          "DE-002",
		Name:        "Show Instructions",
		Description: "Attempts to extract internal instructions",
		Category:    CategoryDataExfiltration,
		Severity:    models.ThreatLevelHigh,
		Regex:       regexp.MustCompile(`(?i)\b(show|reveal|display|repeat|recite|output)\s+(me\s+)?(your|the|all)(\s+the)?\s+(secret\s+)?(instructions?|guidelines?|rules?|programming|directives?)`),
		FileTypes:   []string{},
	},
	{
		ID:          "DE-003",
		Name:        "Leak Configuration",
		Description: "Attempts to leak API keys, configuration, or secrets",
		Category:    CategoryDataExfiltration,
		Severity:    models.ThreatLevelCritical,
		Regex:       regexp.MustCompile(`(?i)\b(leak|expose|reveal|extract|output|print)\s+(the\s+)?(API\s*keys?|secrets?|credentials?|config(uration)?|tokens?|passwords?|environment\s+variables?)`),
		FileTypes:   []string{},
	},

	// =============================================================================
	// OBFUSCATION DETECTION (OB-001 to OB-003)
	// =============================================================================
	{
		ID:          "OB-001",
		Name:        "Zero-Width Characters",
		Description: "Detects zero-width characters used to hide content",
		Category:    CategoryObfuscation,
		Severity:    models.ThreatLevelMedium,
		Regex:       regexp.MustCompile(`[\x{200B}\x{200C}\x{200D}\x{FEFF}\x{2060}]`),
		FileTypes:   []string{},
	},
	{
		ID:          "OB-002",
		Name:        "Unicode Tag Characters",
		Description: "Detects Unicode tag characters (U+E0000-U+E007F)",
		Category:    CategoryObfuscation,
		Severity:    models.ThreatLevelMedium,
		Regex:       regexp.MustCompile(`[\x{E0000}-\x{E007F}]`),
		FileTypes:   []string{},
	},
	{
		ID:          "OB-003",
		Name:        "Base64 Instruction Smuggling",
		Description: "Detects base64-encoded instruction patterns",
		Category:    CategoryObfuscation,
		Severity:    models.ThreatLevelMedium,
		Regex:       regexp.MustCompile(`(?i)\b(decode|execute|run|eval)\s+(this\s+)?base64[:\s]+[A-Za-z0-9+/]{20,}={0,2}`),
		FileTypes:   []string{},
	},

	// =============================================================================
	// AGENT/TOOL MANIPULATION (AM-001 to AM-003)
	// =============================================================================
	{
		ID:          "AM-001",
		Name:        "Shell Command Injection",
		Description: "Attempts to inject shell commands through AI",
		Category:    CategoryAgentManipulation,
		Severity:    models.ThreatLevelCritical,
		Regex:       regexp.MustCompile(`(?i)\b(run|execute|invoke|call)\s+(this\s+)?(shell\s+)?(command|cmd|bash|sh|script)[:\s]+[^\n]{0,200}(rm\s+-rf|curl\s+\||wget\s+\||chmod|chown|sudo|;\s*rm|&&\s*rm)`),
		FileTypes:   []string{},
	},
	{
		ID:          "AM-002",
		Name:        "Code Execution Request",
		Description: "Requests to execute arbitrary code",
		Category:    CategoryAgentManipulation,
		Severity:    models.ThreatLevelCritical,
		Regex:       regexp.MustCompile(`(?i)\b(execute|run|eval)\s+(this|the\s+following)\s+(python|javascript|js|ruby|perl|php|code|script)\s*[:\s]`),
		FileTypes:   []string{},
	},
	{
		ID:          "AM-003",
		Name:        "Data Exfil via Network",
		Description: "Attempts to exfiltrate data via network requests",
		Category:    CategoryAgentManipulation,
		Severity:    models.ThreatLevelCritical,
		Regex:       regexp.MustCompile(`(?i)\b(send|post|upload|transmit|exfil(trate)?)\s+(the\s+)?(data|response|output|results?|conversation)\s+(to|via)\s+(http|https|webhook|api|endpoint|url)`),
		FileTypes:   []string{},
	},

	// =============================================================================
	// PRIVILEGE ESCALATION (PE-001 to PE-002)
	// =============================================================================
	{
		ID:          "PE-001",
		Name:        "Admin Claim",
		Description: "Claims to be an admin or have elevated privileges",
		Category:    CategoryPrivilegeEscalation,
		Severity:    models.ThreatLevelHigh,
		Regex:       regexp.MustCompile(`(?i)\bI\s+(am|'m)\s+(an?\s+)?(admin(istrator)?|root|superuser|system\s+admin|privileged\s+user|authorized\s+user)`),
		FileTypes:   []string{},
	},
	{
		ID:          "PE-002",
		Name:        "Emergency Override",
		Description: "Claims emergency to bypass restrictions",
		Category:    CategoryPrivilegeEscalation,
		Severity:    models.ThreatLevelHigh,
		Regex:       regexp.MustCompile(`(?i)\b(emergency|urgent|critical)\s+(override|access|bypass|exception|authorization)\s*(required|needed|granted)?`),
		FileTypes:   []string{},
	},

	// =============================================================================
	// MULTI-TURN EROSION (MT-001 to MT-003)
	// =============================================================================
	{
		ID:          "MT-001",
		Name:        "Game Setup",
		Description: "Sets up a 'game' context to relax restrictions",
		Category:    CategoryMultiTurnErosion,
		Severity:    models.ThreatLevelMedium,
		Regex:       regexp.MustCompile(`(?i)\blet'?s?\s+play\s+(a\s+)?(game|roleplay|simulation)\s+(where|in\s+which)\s+(you|we|I)\s+(can|are|become|act)`),
		FileTypes:   []string{},
	},
	{
		ID:          "MT-002",
		Name:        "Educational Pretext",
		Description: "Uses 'educational' framing to bypass restrictions",
		Category:    CategoryMultiTurnErosion,
		Severity:    models.ThreatLevelMedium,
		Regex:       regexp.MustCompile(`(?i)\bfor\s+(educational|research|academic|learning)\s+(purposes?|reasons?)\s*(only)?\s*,?\s*(show|explain|demonstrate|teach)\s+(me\s+)?(how\s+to|about)`),
		FileTypes:   []string{},
	},
	{
		ID:          "MT-003",
		Name:        "Hypothetical Scenario",
		Description: "Uses hypothetical framing for restricted content",
		Category:    CategoryMultiTurnErosion,
		Severity:    models.ThreatLevelMedium,
		Regex:       regexp.MustCompile(`(?i)\b(hypothetically|theoretically|in\s+a\s+fictional\s+scenario|imagine\s+if)\s*,?\s*(how\s+would|what\s+if|could\s+you)`),
		FileTypes:   []string{},
	},

	// =============================================================================
	// ADDITIONAL DANGEROUS PATTERNS (DP-001 to DP-003)
	// =============================================================================
	{
		ID:          "DP-001",
		Name:        "Bypass Safety",
		Description: "Explicit requests to bypass safety measures",
		Category:    CategoryJailbreak,
		Severity:    models.ThreatLevelCritical,
		Regex:       regexp.MustCompile(`(?i)\b(bypass|circumvent|evade|skip|ignore|disable)\s+(the\s+)?(safety|security|content)\s+(filter|check|measure|system|guard|restriction|block)`),
		FileTypes:   []string{},
	},
	{
		ID:          "DP-002",
		Name:        "Disable Filter",
		Description: "Requests to disable content filtering",
		Category:    CategoryJailbreak,
		Severity:    models.ThreatLevelCritical,
		Regex:       regexp.MustCompile(`(?i)\b(disable|turn\s+off|deactivate|remove)\s+(your\s+)?(content\s+)?filter(ing|s)?`),
		FileTypes:   []string{},
	},
	{
		ID:          "DP-003",
		Name:        "Unfiltered Mode",
		Description: "Requests to switch to unfiltered operation mode",
		Category:    CategoryJailbreak,
		Severity:    models.ThreatLevelCritical,
		Regex:       regexp.MustCompile(`(?i)\b(switch|change|go)\s+(to|into)\s+(unfiltered|uncensored|raw|unrestricted)\s+(mode|output|response)`),
		FileTypes:   []string{},
	},
}

// GetPatternsByCategory returns all patterns matching the given category.
func GetPatternsByCategory(category ThreatCategory) []Pattern {
	var result []Pattern
	for _, p := range PromptInjectionPatterns {
		if p.Category == category {
			result = append(result, p)
		}
	}
	return result
}

// GetPatternsBySeverity returns all patterns at or above the given minimum severity.
func GetPatternsBySeverity(minSeverity models.ThreatLevel) []Pattern {
	minSev := minSeverity.Severity()
	var result []Pattern
	for _, p := range PromptInjectionPatterns {
		if p.Severity.Severity() >= minSev {
			result = append(result, p)
		}
	}
	return result
}
