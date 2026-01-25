package models

import (
	"testing"
)

// --- SecurityStatus Tests ---

func TestSecurityStatus_IsValid(t *testing.T) {
	tests := []struct {
		status   SecurityStatus
		expected bool
	}{
		{SecurityStatusPending, true},
		{SecurityStatusClean, true},
		{SecurityStatusQuarantined, true},
		{SecurityStatusReleased, true},
		{SecurityStatus("INVALID"), false},
		{SecurityStatus(""), false},
		{SecurityStatus("pending"), false}, // Case sensitive
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := tt.status.IsValid()
			if result != tt.expected {
				t.Errorf("SecurityStatus(%q).IsValid() = %v, want %v", tt.status, result, tt.expected)
			}
		})
	}
}

func TestSecurityStatus_IsBlocked(t *testing.T) {
	tests := []struct {
		status   SecurityStatus
		expected bool
	}{
		{SecurityStatusPending, true},     // Pending blocks usage
		{SecurityStatusQuarantined, true}, // Quarantined blocks usage
		{SecurityStatusClean, false},      // Clean allows usage
		{SecurityStatusReleased, false},   // Released allows usage
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := tt.status.IsBlocked()
			if result != tt.expected {
				t.Errorf("SecurityStatus(%q).IsBlocked() = %v, want %v", tt.status, result, tt.expected)
			}
		})
	}
}

func TestAllSecurityStatuses(t *testing.T) {
	statuses := AllSecurityStatuses()

	// Should return exactly 4 statuses
	if len(statuses) != 4 {
		t.Errorf("AllSecurityStatuses() returned %d statuses, want 4", len(statuses))
	}

	// All returned statuses should be valid
	for _, s := range statuses {
		if !s.IsValid() {
			t.Errorf("AllSecurityStatuses() returned invalid status: %q", s)
		}
	}

	// Should include all expected values
	expected := map[SecurityStatus]bool{
		SecurityStatusPending:     false,
		SecurityStatusClean:       false,
		SecurityStatusQuarantined: false,
		SecurityStatusReleased:    false,
	}
	for _, s := range statuses {
		expected[s] = true
	}
	for status, found := range expected {
		if !found {
			t.Errorf("AllSecurityStatuses() missing status: %q", status)
		}
	}
}

// --- ThreatLevel Tests ---

func TestThreatLevel_IsValid(t *testing.T) {
	tests := []struct {
		level    ThreatLevel
		expected bool
	}{
		{ThreatLevelNone, true},
		{ThreatLevelLow, true},
		{ThreatLevelMedium, true},
		{ThreatLevelHigh, true},
		{ThreatLevelCritical, true},
		{ThreatLevel("INVALID"), false},
		{ThreatLevel(""), false},
		{ThreatLevel("high"), false}, // Case sensitive
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			result := tt.level.IsValid()
			if result != tt.expected {
				t.Errorf("ThreatLevel(%q).IsValid() = %v, want %v", tt.level, result, tt.expected)
			}
		})
	}
}

func TestThreatLevel_Severity(t *testing.T) {
	tests := []struct {
		level    ThreatLevel
		expected int
	}{
		{ThreatLevelNone, 0},
		{ThreatLevelLow, 1},
		{ThreatLevelMedium, 2},
		{ThreatLevelHigh, 3},
		{ThreatLevelCritical, 4},
		{ThreatLevel("INVALID"), 0}, // Unknown defaults to 0
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			result := tt.level.Severity()
			if result != tt.expected {
				t.Errorf("ThreatLevel(%q).Severity() = %d, want %d", tt.level, result, tt.expected)
			}
		})
	}
}

func TestThreatLevel_Severity_Ordering(t *testing.T) {
	// Verify severity increases with threat level
	levels := []ThreatLevel{
		ThreatLevelNone,
		ThreatLevelLow,
		ThreatLevelMedium,
		ThreatLevelHigh,
		ThreatLevelCritical,
	}

	for i := 0; i < len(levels)-1; i++ {
		if levels[i].Severity() >= levels[i+1].Severity() {
			t.Errorf("Severity ordering broken: %q (%d) should be less than %q (%d)",
				levels[i], levels[i].Severity(), levels[i+1], levels[i+1].Severity())
		}
	}
}

func TestAllThreatLevels(t *testing.T) {
	levels := AllThreatLevels()

	// Should return exactly 5 levels
	if len(levels) != 5 {
		t.Errorf("AllThreatLevels() returned %d levels, want 5", len(levels))
	}

	// All returned levels should be valid
	for _, l := range levels {
		if !l.IsValid() {
			t.Errorf("AllThreatLevels() returned invalid level: %q", l)
		}
	}

	// Should include all expected values
	expected := map[ThreatLevel]bool{
		ThreatLevelNone:     false,
		ThreatLevelLow:      false,
		ThreatLevelMedium:   false,
		ThreatLevelHigh:     false,
		ThreatLevelCritical: false,
	}
	for _, l := range levels {
		expected[l] = true
	}
	for level, found := range expected {
		if !found {
			t.Errorf("AllThreatLevels() missing level: %q", level)
		}
	}
}

// --- AuxiliaryDirType Tests ---

func TestAuxiliaryDirType_IsValid(t *testing.T) {
	tests := []struct {
		dirType  AuxiliaryDirType
		expected bool
	}{
		{AuxDirScripts, true},
		{AuxDirReferences, true},
		{AuxDirAssets, true},
		{AuxiliaryDirType("INVALID"), false},
		{AuxiliaryDirType(""), false},
		{AuxiliaryDirType("SCRIPTS"), false}, // Case sensitive (lowercase expected)
	}

	for _, tt := range tests {
		t.Run(string(tt.dirType), func(t *testing.T) {
			result := tt.dirType.IsValid()
			if result != tt.expected {
				t.Errorf("AuxiliaryDirType(%q).IsValid() = %v, want %v", tt.dirType, result, tt.expected)
			}
		})
	}
}

func TestAllAuxiliaryDirTypes(t *testing.T) {
	dirTypes := AllAuxiliaryDirTypes()

	// Should return exactly 3 types
	if len(dirTypes) != 3 {
		t.Errorf("AllAuxiliaryDirTypes() returned %d types, want 3", len(dirTypes))
	}

	// All returned types should be valid
	for _, d := range dirTypes {
		if !d.IsValid() {
			t.Errorf("AllAuxiliaryDirTypes() returned invalid type: %q", d)
		}
	}

	// Should include all expected values
	expected := map[AuxiliaryDirType]bool{
		AuxDirScripts:    false,
		AuxDirReferences: false,
		AuxDirAssets:     false,
	}
	for _, d := range dirTypes {
		expected[d] = true
	}
	for dirType, found := range expected {
		if !found {
			t.Errorf("AllAuxiliaryDirTypes() missing type: %q", dirType)
		}
	}
}

// --- AuxiliaryFile Tests ---

func TestAuxiliaryFile_GenerateID(t *testing.T) {
	af := &AuxiliaryFile{
		SkillID:  "skill-123",
		DirType:  AuxDirScripts,
		FilePath: "utils/helper.sh",
	}

	id1 := af.GenerateID()
	id2 := af.GenerateID()

	// Should be deterministic
	if id1 != id2 {
		t.Errorf("GenerateID() not deterministic: %q != %q", id1, id2)
	}

	// Should be 32 characters (hex string from first 16 bytes of SHA256)
	if len(id1) != 32 {
		t.Errorf("GenerateID() length = %d, want 32", len(id1))
	}

	// Should be hex string
	for _, c := range id1 {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			t.Errorf("GenerateID() contains non-hex character: %q", c)
			break
		}
	}
}

func TestAuxiliaryFile_GenerateID_DifferentInputs(t *testing.T) {
	af1 := &AuxiliaryFile{
		SkillID:  "skill-123",
		DirType:  AuxDirScripts,
		FilePath: "utils/helper.sh",
	}
	af2 := &AuxiliaryFile{
		SkillID:  "skill-456", // Different skill
		DirType:  AuxDirScripts,
		FilePath: "utils/helper.sh",
	}
	af3 := &AuxiliaryFile{
		SkillID:  "skill-123",
		DirType:  AuxDirReferences, // Different dir type
		FilePath: "utils/helper.sh",
	}
	af4 := &AuxiliaryFile{
		SkillID:  "skill-123",
		DirType:  AuxDirScripts,
		FilePath: "other/file.sh", // Different path
	}

	id1 := af1.GenerateID()
	id2 := af2.GenerateID()
	id3 := af3.GenerateID()
	id4 := af4.GenerateID()

	// All IDs should be different
	ids := []string{id1, id2, id3, id4}
	seen := make(map[string]bool)
	for _, id := range ids {
		if seen[id] {
			t.Error("GenerateID() produced duplicate IDs for different inputs")
		}
		seen[id] = true
	}
}

func TestAuxiliaryFile_TableName(t *testing.T) {
	af := AuxiliaryFile{}
	if af.TableName() != "auxiliary_files" {
		t.Errorf("TableName() = %q, want %q", af.TableName(), "auxiliary_files")
	}
}

// --- SecurityScan Tests ---

func TestSecurityScan_TableName(t *testing.T) {
	ss := SecurityScan{}
	if ss.TableName() != "security_scans" {
		t.Errorf("TableName() = %q, want %q", ss.TableName(), "security_scans")
	}
}

func TestScanTypeConstants(t *testing.T) {
	// Verify scan type constants are defined
	if ScanTypeFull != "full" {
		t.Errorf("ScanTypeFull = %q, want %q", ScanTypeFull, "full")
	}
	if ScanTypeDelta != "delta" {
		t.Errorf("ScanTypeDelta = %q, want %q", ScanTypeDelta, "delta")
	}
	if ScanTypeSingle != "single" {
		t.Errorf("ScanTypeSingle = %q, want %q", ScanTypeSingle, "single")
	}
}
