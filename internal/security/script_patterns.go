package security

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/asteroid-belt/skulto/internal/models"
)

// Note: Pattern struct is defined in patterns.go

// ScriptPatterns contains all patterns for detecting dangerous script content.
var ScriptPatterns = []Pattern{
	// ==========================================
	// SHELL DANGEROUS COMMANDS (SH-001 to SH-009)
	// ==========================================
	{
		ID:          "SH-001",
		Name:        "Recursive Delete",
		Description: "Detects rm -rf commands that could destroy files recursively",
		Category:    CategoryScriptDanger,
		Severity:    models.ThreatLevelHigh,
		Regex:       regexp.MustCompile(`\brm\s+(-[a-zA-Z]*r[a-zA-Z]*f|-[a-zA-Z]*f[a-zA-Z]*r)\b`),
		FileTypes:   []string{"*.sh", "*.bash", "*.zsh", "SKILL.md"},
	},
	{
		ID:          "SH-002",
		Name:        "World-Writable Permissions",
		Description: "Detects chmod 777 or chmod with world-write permissions",
		Category:    CategoryScriptDanger,
		Severity:    models.ThreatLevelMedium,
		Regex:       regexp.MustCompile(`\bchmod\s+(777|[0-7][0-7][2367]|o\+[rwx]*w|a\+[rwx]*w)`),
		FileTypes:   []string{"*.sh", "*.bash", "*.zsh", "SKILL.md"},
	},
	{
		ID:          "SH-003",
		Name:        "Disk Overwrite",
		Description: "Detects dd commands that could overwrite disk devices",
		Category:    CategoryScriptDanger,
		Severity:    models.ThreatLevelCritical,
		Regex:       regexp.MustCompile(`\bdd\s+.*of=/dev/(sd[a-z]|hd[a-z]|nvme[0-9]|disk[0-9])`),
		FileTypes:   []string{"*.sh", "*.bash", "*.zsh", "SKILL.md"},
	},
	{
		ID:          "SH-004",
		Name:        "Fork Bomb",
		Description: "Detects fork bomb patterns that exhaust system resources",
		Category:    CategoryScriptDanger,
		Severity:    models.ThreatLevelCritical,
		Regex:       regexp.MustCompile(`:\(\)\s*\{\s*:\s*\|\s*:\s*&\s*\}\s*;?\s*:|[a-zA-Z_]+\(\)\s*\{\s*[a-zA-Z_]+\s*\|\s*[a-zA-Z_]+\s*&\s*\}`),
		FileTypes:   []string{"*.sh", "*.bash", "*.zsh", "SKILL.md"},
	},
	{
		ID:          "SH-005",
		Name:        "Curl Pipe to Shell",
		Description: "Detects piping curl output directly to shell execution",
		Category:    CategoryScriptDanger,
		Severity:    models.ThreatLevelHigh,
		Regex:       regexp.MustCompile(`\bcurl\b[^|]*\|\s*(ba)?sh\b|\bwget\b[^|]*\|\s*(ba)?sh\b`),
		FileTypes:   []string{"*.sh", "*.bash", "*.zsh", "SKILL.md"},
	},
	{
		ID:          "SH-006",
		Name:        "History Clear",
		Description: "Detects attempts to clear command history (anti-forensics)",
		Category:    CategoryObfuscation,
		Severity:    models.ThreatLevelMedium,
		Regex:       regexp.MustCompile(`\bhistory\s+-[cw]|\brm\s+[~.]*(bash_history|zsh_history|history)|>\s*[~.]*[/]?(bash_history|zsh_history)`),
		FileTypes:   []string{"*.sh", "*.bash", "*.zsh", "SKILL.md"},
	},
	{
		ID:          "SH-007",
		Name:        "Suspicious Curl POST",
		Description: "Detects curl POST requests that may exfiltrate data",
		Category:    CategoryDataExfiltration,
		Severity:    models.ThreatLevelMedium,
		Regex:       regexp.MustCompile(`\bcurl\b[^;]*(-X\s*POST|--request\s+POST|-d\s|--data)`),
		FileTypes:   []string{"*.sh", "*.bash", "*.zsh", "SKILL.md"},
	},
	{
		ID:          "SH-008",
		Name:        "Reverse Shell",
		Description: "Detects reverse shell patterns for remote access",
		Category:    CategoryScriptDanger,
		Severity:    models.ThreatLevelCritical,
		Regex:       regexp.MustCompile(`\b(nc|netcat|ncat)\b.*-[elp]|/dev/(tcp|udp)/|bash\s+-i\s+>&\s*/dev/(tcp|udp)/|\bpython[23]?\b.*socket.*connect`),
		FileTypes:   []string{"*.sh", "*.bash", "*.zsh", "*.py", "SKILL.md"},
	},
	{
		ID:          "SH-009",
		Name:        "Sensitive Environment Access",
		Description: "Detects access to sensitive environment variables",
		Category:    CategoryDataExfiltration,
		Severity:    models.ThreatLevelMedium,
		Regex:       regexp.MustCompile(`\$\{?(AWS_SECRET|AWS_ACCESS|GITHUB_TOKEN|API_KEY|SECRET_KEY|PRIVATE_KEY|PASSWORD|CREDENTIALS)\}?`),
		FileTypes:   []string{"*.sh", "*.bash", "*.zsh", "*.py", "*.js", "SKILL.md"},
	},

	// ==========================================
	// PYTHON DANGEROUS CODE (PY-001 to PY-005)
	// ==========================================
	{
		ID:          "PY-001",
		Name:        "Python eval/exec",
		Description: "Detects eval() or exec() which can execute arbitrary code",
		Category:    CategoryScriptDanger,
		Severity:    models.ThreatLevelHigh,
		Regex:       regexp.MustCompile(`\b(eval|exec)\s*\(`),
		FileTypes:   []string{"*.py", "SKILL.md"},
	},
	{
		ID:          "PY-002",
		Name:        "Subprocess shell=True",
		Description: "Detects subprocess calls with shell=True which enables shell injection",
		Category:    CategoryScriptDanger,
		Severity:    models.ThreatLevelHigh,
		Regex:       regexp.MustCompile(`subprocess\.(call|run|Popen)\s*\([^)]*shell\s*=\s*True`),
		FileTypes:   []string{"*.py", "SKILL.md"},
	},
	{
		ID:          "PY-003",
		Name:        "Python os.system",
		Description: "Detects os.system() calls which execute shell commands",
		Category:    CategoryScriptDanger,
		Severity:    models.ThreatLevelMedium,
		Regex:       regexp.MustCompile(`\bos\.system\s*\(`),
		FileTypes:   []string{"*.py", "SKILL.md"},
	},
	{
		ID:          "PY-004",
		Name:        "Pickle Load",
		Description: "Detects pickle.load which can execute arbitrary code during deserialization",
		Category:    CategoryScriptDanger,
		Severity:    models.ThreatLevelHigh,
		Regex:       regexp.MustCompile(`\bpickle\.(load|loads)\s*\(|\bcPickle\.(load|loads)\s*\(`),
		FileTypes:   []string{"*.py", "SKILL.md"},
	},
	{
		ID:          "PY-005",
		Name:        "Python Network Exfiltration",
		Description: "Detects HTTP POST requests that may exfiltrate data",
		Category:    CategoryDataExfiltration,
		Severity:    models.ThreatLevelMedium,
		Regex:       regexp.MustCompile(`requests\.(post|put)\s*\(|urllib\.request\.(urlopen|Request)\s*\([^)]*method\s*=\s*['"]POST['"]|httpx\.(post|put)\s*\(`),
		FileTypes:   []string{"*.py", "SKILL.md"},
	},

	// ==========================================
	// JAVASCRIPT DANGEROUS CODE (JS-001 to JS-004)
	// ==========================================
	{
		ID:          "JS-001",
		Name:        "JavaScript eval",
		Description: "Detects eval() which can execute arbitrary code",
		Category:    CategoryScriptDanger,
		Severity:    models.ThreatLevelHigh,
		Regex:       regexp.MustCompile(`\beval\s*\(`),
		FileTypes:   []string{"*.js", "*.ts", "*.mjs", "*.cjs", "SKILL.md"},
	},
	{
		ID:          "JS-002",
		Name:        "Function Constructor",
		Description: "Detects new Function() which can execute arbitrary code",
		Category:    CategoryScriptDanger,
		Severity:    models.ThreatLevelHigh,
		Regex:       regexp.MustCompile(`new\s+Function\s*\(`),
		FileTypes:   []string{"*.js", "*.ts", "*.mjs", "*.cjs", "SKILL.md"},
	},
	{
		ID:          "JS-003",
		Name:        "Child Process Execution",
		Description: "Detects child_process module usage for command execution",
		Category:    CategoryScriptDanger,
		Severity:    models.ThreatLevelMedium,
		Regex:       regexp.MustCompile(`require\s*\(\s*['"]child_process['"]\s*\)|from\s+['"]child_process['"]|child_process\.(exec|spawn|execSync|spawnSync)\s*\(`),
		FileTypes:   []string{"*.js", "*.ts", "*.mjs", "*.cjs", "SKILL.md"},
	},
	{
		ID:          "JS-004",
		Name:        "JavaScript Fetch POST",
		Description: "Detects fetch POST requests that may exfiltrate data",
		Category:    CategoryDataExfiltration,
		Severity:    models.ThreatLevelMedium,
		Regex:       regexp.MustCompile(`fetch\s*\([^)]*\{[^}]*method\s*:\s*['"]POST['"]|axios\.(post|put)\s*\(`),
		FileTypes:   []string{"*.js", "*.ts", "*.mjs", "*.cjs", "SKILL.md"},
	},

	// ==========================================
	// GENERAL SCRIPT PATTERNS (GEN-001 to GEN-003)
	// ==========================================
	{
		ID:          "GEN-001",
		Name:        "Encoded Payload",
		Description: "Detects base64 encoded content that may hide malicious payloads",
		Category:    CategoryObfuscation,
		Severity:    models.ThreatLevelMedium,
		Regex:       regexp.MustCompile(`\bbase64\s+(-d|--decode)|\batob\s*\(|base64\.b64decode\s*\(`),
		FileTypes:   []string{"*.sh", "*.bash", "*.zsh", "*.py", "*.js", "*.ts", "SKILL.md"},
	},
	{
		ID:          "GEN-002",
		Name:        "Cron Persistence",
		Description: "Detects crontab modifications for persistence",
		Category:    CategoryPrivilegeEscalation,
		Severity:    models.ThreatLevelMedium,
		Regex:       regexp.MustCompile(`crontab\s+-[elri]|/etc/cron\.(d|daily|hourly|weekly|monthly)/|\*\s+\*\s+\*\s+\*\s+\*`),
		FileTypes:   []string{"*.sh", "*.bash", "*.zsh", "SKILL.md"},
	},
	{
		ID:          "GEN-003",
		Name:        "SSH Key Injection",
		Description: "Detects SSH authorized_keys modifications",
		Category:    CategoryPrivilegeEscalation,
		Severity:    models.ThreatLevelHigh,
		Regex:       regexp.MustCompile(`>>\s*~?/?\.ssh/authorized_keys|echo\s+.*>>\s*.*authorized_keys|ssh-keygen.*-f`),
		FileTypes:   []string{"*.sh", "*.bash", "*.zsh", "SKILL.md"},
	},
}

// GetScriptPatterns returns all script analysis patterns.
func GetScriptPatterns() []Pattern {
	return ScriptPatterns
}

// GetPatternsForFile returns patterns applicable to a specific file based on its path.
func GetPatternsForFile(filePath string) []Pattern {
	var applicable []Pattern

	for _, pattern := range ScriptPatterns {
		for _, fileType := range pattern.FileTypes {
			if matchFileType(filePath, fileType) {
				applicable = append(applicable, pattern)
				break
			}
		}
	}

	return applicable
}

// matchFileType checks if a file path matches a file type pattern.
// Supports wildcards like "*.sh" and exact matches like "SKILL.md".
func matchFileType(filePath, fileType string) bool {
	// Get just the filename from the path
	fileName := filepath.Base(filePath)

	// Handle wildcard patterns like "*.sh"
	if strings.HasPrefix(fileType, "*.") {
		ext := strings.TrimPrefix(fileType, "*")
		return strings.HasSuffix(strings.ToLower(fileName), strings.ToLower(ext))
	}

	// Handle exact matches like "SKILL.md"
	return strings.EqualFold(fileName, fileType)
}
