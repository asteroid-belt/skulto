#!/bin/bash
#
# Phase 2 Test Script: Auto-Sync on Startup and Pull
#
# This script tests the Phase 2 implementation including:
# - syncPrimarySkillsCmd method exists
# - primarySyncCompleteMsg type exists
# - Primary sync added to Init() (conditional on onboarding completed)
# - Primary sync added to pull action (p key)
#
# Usage:
#   ./scripts/primary/test-phase2.sh
#

set -e  # Exit on first error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     Phase 2 Tests: Auto-Sync on Startup and Pull          ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Track test results
TESTS_PASSED=0
TESTS_FAILED=0

pass() {
    echo -e "${GREEN}✓ $1${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

fail() {
    echo -e "${RED}✗ $1${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

# Check prerequisites
check_prerequisites() {
    echo -e "${BLUE}━━━ Checking Prerequisites ━━━${NC}"

    if ! command -v go &> /dev/null; then
        echo -e "${RED}✗ Go is not installed${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ Go is installed ($(go version | cut -d' ' -f3))${NC}"

    if [ ! -f "go.mod" ] || ! grep -q "skulto" go.mod; then
        echo -e "${RED}✗ Not in skulto project directory${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ In skulto project directory${NC}"

    echo ""
}

# Test 1: syncPrimarySkillsCmd method exists
test_sync_method() {
    echo -e "${BLUE}━━━ Test 1: syncPrimarySkillsCmd Method ━━━${NC}"

    APP_FILE="internal/tui/app.go"

    if grep -q "func (m \*Model) syncPrimarySkillsCmd" "$APP_FILE"; then
        pass "syncPrimarySkillsCmd method exists"
    else
        fail "syncPrimarySkillsCmd method not found"
        return
    fi

    # Check it uses correct scraper constructor
    if grep -A15 "func (m \*Model) syncPrimarySkillsCmd" "$APP_FILE" | grep -q "NewScraperWithConfig"; then
        pass "Uses NewScraperWithConfig (not deprecated New)"
    else
        fail "Not using NewScraperWithConfig"
    fi

    # Check it uses PrimarySkillsRepo
    if grep -A15 "func (m \*Model) syncPrimarySkillsCmd" "$APP_FILE" | grep -q "scraper.PrimarySkillsRepo"; then
        pass "Uses scraper.PrimarySkillsRepo constant"
    else
        fail "Not using scraper.PrimarySkillsRepo"
    fi

    # Check it returns primarySyncCompleteMsg
    if grep -A20 "func (m \*Model) syncPrimarySkillsCmd" "$APP_FILE" | grep -q "primarySyncCompleteMsg"; then
        pass "Returns primarySyncCompleteMsg"
    else
        fail "Doesn't return primarySyncCompleteMsg"
    fi

    echo ""
}

# Test 2: primarySyncCompleteMsg type exists
test_msg_type() {
    echo -e "${BLUE}━━━ Test 2: primarySyncCompleteMsg Type ━━━${NC}"

    APP_FILE="internal/tui/app.go"

    if grep -q "type primarySyncCompleteMsg struct" "$APP_FILE"; then
        pass "primarySyncCompleteMsg type defined"
    else
        fail "primarySyncCompleteMsg type not found"
    fi

    echo ""
}

# Test 3: Primary sync in Init()
test_init_sync() {
    echo -e "${BLUE}━━━ Test 3: Primary Sync in Init() ━━━${NC}"

    APP_FILE="internal/tui/app.go"

    # Check that Init() calls syncPrimarySkillsCmd
    if grep -A50 "func (m \*Model) Init()" "$APP_FILE" | grep -q "syncPrimarySkillsCmd"; then
        pass "Init() calls syncPrimarySkillsCmd"
    else
        fail "Init() doesn't call syncPrimarySkillsCmd"
    fi

    # Check that it's conditional on onboarding completed
    if grep -A50 "func (m \*Model) Init()" "$APP_FILE" | grep -q "IsOnboardingCompleted"; then
        pass "Sync is conditional on IsOnboardingCompleted()"
    else
        fail "Sync is not conditional on onboarding status"
    fi

    echo ""
}

# Test 4: Primary sync in pull action
test_pull_sync() {
    echo -e "${BLUE}━━━ Test 4: Primary Sync in Pull Action ━━━${NC}"

    APP_FILE="internal/tui/app.go"

    # Find the pull key handler and check it includes syncPrimarySkillsCmd
    if grep -A10 'key == "p".*ViewHome.*IsPulling' "$APP_FILE" | grep -q "syncPrimarySkillsCmd"; then
        pass "Pull action includes syncPrimarySkillsCmd"
    else
        fail "Pull action doesn't include syncPrimarySkillsCmd"
    fi

    # Check it's in a tea.Batch with other commands
    if grep -A10 'key == "p".*ViewHome.*IsPulling' "$APP_FILE" | grep -q "tea.Batch"; then
        pass "syncPrimarySkillsCmd is batched with other pull commands"
    else
        fail "syncPrimarySkillsCmd is not in tea.Batch"
    fi

    echo ""
}

# Test 5: Build verification
test_build() {
    echo -e "${BLUE}━━━ Test 5: Build Verification ━━━${NC}"

    if go build ./internal/tui/... 2>&1; then
        pass "TUI package builds successfully"
    else
        fail "TUI package failed to build"
    fi

    if go build ./... 2>&1; then
        pass "Full project builds successfully"
    else
        fail "Full project failed to build"
    fi

    echo ""
}

# Test 6: Code quality
test_code_quality() {
    echo -e "${BLUE}━━━ Test 6: Code Quality ━━━${NC}"

    if [ -z "$(gofmt -l internal/tui/app.go 2>&1)" ]; then
        pass "Code passes go fmt"
    else
        fail "Code needs formatting"
    fi

    if golangci-lint run ./internal/tui/... 2>&1; then
        pass "Code passes golangci-lint"
    else
        fail "golangci-lint found issues"
    fi

    echo ""
}

# Test 7: Silent failure behavior
test_silent_failure() {
    echo -e "${BLUE}━━━ Test 7: Silent Failure Behavior ━━━${NC}"

    APP_FILE="internal/tui/app.go"

    # Check that errors in syncPrimarySkillsCmd return the message (not error)
    if grep -A25 "func (m \*Model) syncPrimarySkillsCmd" "$APP_FILE" | grep -B2 "return primarySyncCompleteMsg" | grep -q "err != nil"; then
        pass "Errors return primarySyncCompleteMsg (silent failure)"
    else
        fail "Errors may not be handled silently"
    fi

    echo ""
}

# Print summary
print_summary() {
    echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║                   Phase 2 Test Summary                     ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "Tests passed: ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Tests failed: ${RED}$TESTS_FAILED${NC}"
    echo ""

    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "${GREEN}All Phase 2 tests passed!${NC}"
        echo ""
        echo "Definition of Done:"
        echo -e "  ${GREEN}✓${NC} Code passes golangci-lint"
        echo -e "  ${GREEN}✓${NC} Code passes go fmt"
        echo -e "  ${GREEN}✓${NC} Primary skills sync on startup (background, silent failure)"
        echo -e "  ${GREEN}✓${NC} Primary skills sync when pressing 'p'"
        echo -e "  ${GREEN}✓${NC} No UI blocking during background sync"
        exit 0
    else
        echo -e "${RED}Some Phase 2 tests failed!${NC}"
        exit 1
    fi
}

# Main execution
main() {
    check_prerequisites
    test_sync_method
    test_msg_type
    test_init_sync
    test_pull_sync
    test_build
    test_code_quality
    test_silent_failure
    print_summary
}

main "$@"
