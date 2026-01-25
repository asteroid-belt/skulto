// Package scraper contains GitHub scraping and skill parsing logic.
package scraper

import (
	"regexp"
	"strings"
)

// LicenseInfo contains detected license information and metadata.
type LicenseInfo struct {
	Type     string // SPDX identifier (e.g., "MIT", "Apache-2.0")
	FileName string // The LICENSE file name found
	URL      string // Direct link to the LICENSE file in GitHub
	RawURL   string // Raw content URL for viewing
}

// licensePattern is a single license pattern with SPDX identifier.
type licensePattern struct {
	spdxID string
	regex  *regexp.Regexp
}

// licensePatterns is an ordered list of license patterns to check.
// Order matters because we return the first match!
// Using RE2-compatible syntax (no lookahead/lookbehind).
var licensePatterns = []licensePattern{
	// Check more specific patterns first, then more general ones
	{"Unlicense", regexp.MustCompile(`(?i)unlicense|This is free and unencumbered`)},
	{"CC0-1.0", regexp.MustCompile(`(?i)Creative\s+Commons.*Zero|CC0`)},
	{"MIT", regexp.MustCompile(`(?i)MIT\s+License|Permission is hereby granted, free of charge`)},
	{"Apache-2.0", regexp.MustCompile(`(?i)Apache\s+License.*2\.0|Licensed under the Apache License`)},
	{"GPL-3.0", regexp.MustCompile(`(?i)GNU\s+General\s+Public\s+License.*version\s+3|GPL-3\.0|GPLv3`)},
	{"GPL-2.0", regexp.MustCompile(`(?i)GNU\s+General\s+Public\s+License.*version\s+2|GPL-2\.0|GPLv2`)},
	{"AGPL-3.0", regexp.MustCompile(`(?i)GNU\s+Affero.*License.*version\s+3|AGPL-3\.0|AGPLv3`)},
	{"LGPL-3.0", regexp.MustCompile(`(?i)GNU\s+Lesser.*License.*version\s+3|LGPL-3\.0|LGPLv3`)},
	{"LGPL-2.1", regexp.MustCompile(`(?i)GNU\s+Lesser.*License.*version\s+2\.1|LGPL-2\.1|LGPLv2\.1`)},
	{"BSD-3-Clause", regexp.MustCompile(`(?i)three\s+clauses|BSD.*3.*Clause|New BSD|Modified BSD`)},
	{"BSD-2-Clause", regexp.MustCompile(`(?i)two\s+clauses|BSD.*2.*Clause|Simplified BSD`)},
	{"EPL-2.0", regexp.MustCompile(`(?i)Eclipse\s+Public\s+License.*2\.0|EPL-2\.0`)},
	{"EPL-1.0", regexp.MustCompile(`(?i)Eclipse\s+Public\s+License.*1\.0|EPL-1\.0`)},
	{"MPL-2.0", regexp.MustCompile(`(?i)Mozilla\s+Public\s+License.*2\.0|MPL-2\.0|MPL\s+2`)},
	{"ISC", regexp.MustCompile(`(?i)ISC\s+License|Permission to use, copy, modify.*ISC`)},
	{"Zlib", regexp.MustCompile(`(?i)zlib\s+License`)},
}

// DetectLicenseType analyzes license content and returns the detected SPDX identifier.
// Returns "Unknown" if no license pattern is matched.
func DetectLicenseType(content string) string {
	if content == "" {
		return "Unknown"
	}

	// Limit content for performance (first 2000 chars should be enough)
	if len(content) > 2000 {
		content = content[:2000]
	}

	// Check each license pattern in order (first match wins)
	for _, pattern := range licensePatterns {
		if pattern.regex.MatchString(content) {
			return pattern.spdxID
		}
	}

	// Check for "or later" variants (e.g., "GPL-2.0+")
	lowerContent := strings.ToLower(content)
	if strings.Contains(lowerContent, "or any later") || strings.Contains(lowerContent, "or (at your option) any later") {
		if strings.Contains(lowerContent, "version 2") {
			return "GPL-2.0+"
		}
		if strings.Contains(lowerContent, "version 3") {
			return "GPL-3.0+"
		}
	}

	return "Unknown"
}

// LicenseFileNames returns the list of license file names to search for, in order.
func LicenseFileNames() []string {
	return []string{
		"LICENSE",
		"LICENSE.md",
		"LICENSE.txt",
		"COPYING",
		"COPYING.md",
		"COPYING.txt",
		"LICENSE.rst",
	}
}
