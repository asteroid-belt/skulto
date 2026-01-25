// Package testutil provides testing utilities.
package testutil

import (
	"os"
	"testing"
)

// SkipAITests skips the test if RUN_AI_TESTS is not set.
// Use this for tests that require AI API keys (OpenAI, Anthropic, etc.).
//
// Run AI tests with: RUN_AI_TESTS=1 go test ./...
func SkipAITests(t *testing.T) {
	t.Helper()
	if os.Getenv("RUN_AI_TESTS") == "" {
		t.Skip("Skipping AI test (set RUN_AI_TESTS=1 to run)")
	}
}
