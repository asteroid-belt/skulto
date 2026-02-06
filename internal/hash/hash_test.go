package hash

import "testing"

func TestTruncatedSHA256(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int // expected length
	}{
		{
			name:  "empty string",
			input: "",
			want:  IDLength,
		},
		{
			name:  "simple string",
			input: "hello world",
			want:  IDLength,
		},
		{
			name:  "repo:path format",
			input: "owner/repo:path/to/file.md",
			want:  IDLength,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncatedSHA256(tt.input)
			if len(got) != tt.want {
				t.Errorf("TruncatedSHA256(%q) length = %d, want %d", tt.input, len(got), tt.want)
			}
		})
	}
}

func TestTruncatedSHA256Bytes(t *testing.T) {
	input := []byte("test content")
	got := TruncatedSHA256Bytes(input)
	if len(got) != IDLength {
		t.Errorf("TruncatedSHA256Bytes() length = %d, want %d", len(got), IDLength)
	}
}

func TestTruncatedSHA256_Deterministic(t *testing.T) {
	input := "same input"
	first := TruncatedSHA256(input)
	second := TruncatedSHA256(input)
	if first != second {
		t.Errorf("TruncatedSHA256 not deterministic: %q != %q", first, second)
	}
}

func TestTruncatedSHA256_DifferentInputs(t *testing.T) {
	a := TruncatedSHA256("input a")
	b := TruncatedSHA256("input b")
	if a == b {
		t.Error("Different inputs produced same hash")
	}
}
