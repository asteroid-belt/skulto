package cli

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// CHARACTERIZATION TESTS FOR CLI
// =============================================================================
// These tests capture CURRENT behavior, not desired behavior.
// If these tests fail after refactoring, behavior changed (possibly incorrectly).
// DO NOT MODIFY these tests without understanding why existing behavior changed.
// =============================================================================

// TestCharacterization_classifyError_ConfigError captures config error classification.
func TestCharacterization_classifyError_ConfigError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected string
	}{
		{"config lowercase", "config file not found", "config_error"},
		{"configuration word", "configuration error occurred", "config_error"},
		{"Config mixed case", "Config is invalid", "config_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.errMsg)
			result := classifyError(err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCharacterization_classifyError_DatabaseError captures database error classification.
func TestCharacterization_classifyError_DatabaseError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected string
	}{
		{"database word", "database connection failed", "database_error"},
		{"db abbreviation", "db error occurred", "database_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.errMsg)
			result := classifyError(err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCharacterization_classifyError_NetworkError captures network error classification.
func TestCharacterization_classifyError_NetworkError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected string
	}{
		{"network word", "network unreachable", "network_error"},
		{"timeout word", "connection timeout", "network_error"},
		{"connection word", "connection refused", "network_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.errMsg)
			result := classifyError(err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCharacterization_classifyError_PermissionError captures permission error classification.
func TestCharacterization_classifyError_PermissionError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected string
	}{
		{"permission word", "permission denied", "permission_error"},
		{"access denied", "access denied to file", "permission_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.errMsg)
			result := classifyError(err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCharacterization_classifyError_NotFoundError captures not found error classification.
func TestCharacterization_classifyError_NotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected string
	}{
		{"not found phrase", "skill not found", "not_found_error"},
		{"does not exist", "file does not exist", "not_found_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.errMsg)
			result := classifyError(err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCharacterization_classifyError_ValidationError captures validation error classification.
func TestCharacterization_classifyError_ValidationError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected string
	}{
		{"invalid word", "invalid argument provided", "validation_error"},
		{"parse word", "failed to parse JSON", "validation_error"},
		{"format word", "wrong format for input", "validation_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.errMsg)
			result := classifyError(err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCharacterization_classifyError_UnknownError captures unknown error classification.
func TestCharacterization_classifyError_UnknownError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected string
	}{
		{"random error", "something weird happened", "unknown_error"},
		{"generic error", "an error occurred", "unknown_error"},
		{"unusual message", "the sky is falling", "unknown_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.errMsg)
			result := classifyError(err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCharacterization_classifyError_CaseInsensitive captures case-insensitive matching.
func TestCharacterization_classifyError_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected string
	}{
		{"CONFIG uppercase", "CONFIG file missing", "config_error"},
		{"Database caps", "Database connection lost", "database_error"},
		{"NETWORK caps", "NETWORK timeout", "network_error"},
		{"Permission caps", "Permission denied", "permission_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.errMsg)
			result := classifyError(err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCharacterization_classifyError_PriorityOrder captures priority when multiple keywords match.
func TestCharacterization_classifyError_PriorityOrder(t *testing.T) {
	// Current behavior: first matching category wins (in order of switch cases)
	tests := []struct {
		name     string
		errMsg   string
		expected string
	}{
		// config is checked first, so it wins over database
		{"config and db", "config database error", "config_error"},
		// database is checked before network
		{"db and network", "database network failure", "database_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.errMsg)
			result := classifyError(err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCharacterization_containsAny_EmptySubstrings captures behavior with empty substrings.
func TestCharacterization_containsAny_EmptySubstrings(t *testing.T) {
	// Current behavior: empty substring matches everything (strings.Contains behavior)
	result := containsAny("hello", "")
	assert.True(t, result)
}

// TestCharacterization_containsAny_NoMatch captures behavior when no substring matches.
func TestCharacterization_containsAny_NoMatch(t *testing.T) {
	result := containsAny("hello world", "foo", "bar", "baz")
	assert.False(t, result)
}

// TestCharacterization_containsAny_SingleMatch captures behavior when one substring matches.
func TestCharacterization_containsAny_SingleMatch(t *testing.T) {
	result := containsAny("hello world", "foo", "world", "bar")
	assert.True(t, result)
}

// TestCharacterization_containsAny_CaseInsensitive captures case-insensitive matching.
func TestCharacterization_containsAny_CaseInsensitive(t *testing.T) {
	// Current behavior: only the input string is lowercased, not the substrings
	// So "HELLO" won't match because it looks for "HELLO" in "hello world"
	result := containsAny("Hello World", "HELLO")
	assert.False(t, result) // substring is not lowercased

	// But lowercase substrings will match
	result = containsAny("Hello World", "hello")
	assert.True(t, result)
}

// TestCharacterization_trackCLIError_NilError captures behavior with nil error.
func TestCharacterization_trackCLIError_NilError(t *testing.T) {
	// Current behavior: returns nil for nil error (no tracking happens)
	result := trackCLIError("test-cmd", nil)
	assert.Nil(t, result)
}

// TestCharacterization_trackCLIError_RequiresTelemetryClient documents that
// trackCLIError requires telemetryClient to be initialized.
// This test is intentionally skipped because it would panic if telemetryClient is nil.
func TestCharacterization_trackCLIError_RequiresTelemetryClient(t *testing.T) {
	// Current behavior: If telemetryClient is nil and err is not nil,
	// calling trackCLIError will panic with nil pointer dereference.
	// This is a known issue that should be fixed (defensive nil check).
	t.Skip("Skipped: trackCLIError panics when telemetryClient is nil - documents current fragile behavior")
}

// TestCharacterization_RootCmd_Structure captures the root command structure.
func TestCharacterization_RootCmd_Structure(t *testing.T) {
	assert.Equal(t, "skulto", rootCmd.Use)
	assert.NotEmpty(t, rootCmd.Short)
	assert.NotEmpty(t, rootCmd.Long)
	assert.True(t, rootCmd.SilenceUsage)
}

// TestCharacterization_RootCmd_HasSubcommands captures registered subcommands.
func TestCharacterization_RootCmd_HasSubcommands(t *testing.T) {
	subcommands := rootCmd.Commands()

	// Current expected subcommands
	expectedNames := []string{"add", "info", "list", "pull", "remove", "scan", "update"}

	actualNames := make([]string, 0, len(subcommands))
	for _, cmd := range subcommands {
		actualNames = append(actualNames, cmd.Name())
	}

	for _, expected := range expectedNames {
		assert.Contains(t, actualNames, expected, "Missing subcommand: %s", expected)
	}
}
