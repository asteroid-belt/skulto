package models

import (
	"testing"
)

func TestSkill_IsUsable(t *testing.T) {
	tests := []struct {
		name           string
		securityStatus SecurityStatus
		wantUsable     bool
	}{
		{
			name:           "PENDING status blocks usage",
			securityStatus: SecurityStatusPending,
			wantUsable:     false,
		},
		{
			name:           "QUARANTINED status blocks usage",
			securityStatus: SecurityStatusQuarantined,
			wantUsable:     false,
		},
		{
			name:           "CLEAN status allows usage",
			securityStatus: SecurityStatusClean,
			wantUsable:     true,
		},
		{
			name:           "RELEASED status allows usage",
			securityStatus: SecurityStatusReleased,
			wantUsable:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skill := &Skill{
				ID:             "test-skill",
				SecurityStatus: tt.securityStatus,
			}

			got := skill.IsUsable()
			if got != tt.wantUsable {
				t.Errorf("IsUsable() = %v, want %v for status %s", got, tt.wantUsable, tt.securityStatus)
			}
		})
	}
}

func TestSkill_ComputeContentHash(t *testing.T) {
	skill := &Skill{
		ID:      "test-skill",
		Content: "# Test Skill\n\nThis is test content.",
	}

	// Compute hash twice - should be deterministic
	hash1 := skill.ComputeContentHash()
	hash2 := skill.ComputeContentHash()

	if hash1 != hash2 {
		t.Errorf("ComputeContentHash() not deterministic: got %s and %s", hash1, hash2)
	}

	// SHA256 produces 64 hex characters
	if len(hash1) != 64 {
		t.Errorf("ComputeContentHash() length = %d, want 64", len(hash1))
	}

	// Verify it's valid hex
	for _, c := range hash1 {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			t.Errorf("ComputeContentHash() contains invalid hex character: %c", c)
		}
	}
}

func TestSkill_ComputeContentHash_DifferentContent(t *testing.T) {
	skill1 := &Skill{
		ID:      "skill-1",
		Content: "Content A",
	}
	skill2 := &Skill{
		ID:      "skill-2",
		Content: "Content B",
	}

	hash1 := skill1.ComputeContentHash()
	hash2 := skill2.ComputeContentHash()

	if hash1 == hash2 {
		t.Errorf("ComputeContentHash() should return different hashes for different content, got same: %s", hash1)
	}
}

func TestSkill_ComputeContentHash_EmptyContent(t *testing.T) {
	skill := &Skill{
		ID:      "empty-skill",
		Content: "",
	}

	hash := skill.ComputeContentHash()

	// Even empty content should produce a valid hash
	if len(hash) != 64 {
		t.Errorf("ComputeContentHash() with empty content: length = %d, want 64", len(hash))
	}

	// SHA256 of empty string is known: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
	expectedHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if hash != expectedHash {
		t.Errorf("ComputeContentHash() empty content = %s, want %s", hash, expectedHash)
	}
}
