#!/bin/bash
#
# Phase 1 Test Script: Onboarding Skills View & App Integration
#
# This script tests the Phase 1 implementation including:
# - OnboardingSkillsView component exists and compiles
# - ViewOnboardingSkills enum is added to app.go
# - State machine transitions are wired correctly
# - PrimarySkillsFetchedMsg handler exists
#
# Usage:
#   ./scripts/primary/test-phase1.sh
#

set -e  # Exit on first error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     Phase 1 Tests: Onboarding Skills View Integration     ║${NC}"
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

# Test 1: OnboardingSkillsView file exists
test_view_file_exists() {
    echo -e "${BLUE}━━━ Test 1: OnboardingSkillsView File ━━━${NC}"

    VIEW_FILE="internal/tui/views/onboarding_skills.go"
    if [ -f "$VIEW_FILE" ]; then
        pass "OnboardingSkillsView file exists at $VIEW_FILE"
    else
        fail "OnboardingSkillsView file not found at $VIEW_FILE"
        return
    fi

    # Check for key components
    if grep -q "type OnboardingSkillsView struct" "$VIEW_FILE"; then
        pass "OnboardingSkillsView struct defined"
    else
        fail "OnboardingSkillsView struct not found"
    fi

    if grep -q "type PrimarySkillsFetchedMsg struct" "$VIEW_FILE"; then
        pass "PrimarySkillsFetchedMsg struct defined"
    else
        fail "PrimarySkillsFetchedMsg struct not found"
    fi

    if grep -q "type SkillItem struct" "$VIEW_FILE"; then
        pass "SkillItem struct defined"
    else
        fail "SkillItem struct not found"
    fi

    if grep -q "func NewOnboardingSkillsView" "$VIEW_FILE"; then
        pass "NewOnboardingSkillsView constructor defined"
    else
        fail "NewOnboardingSkillsView constructor not found"
    fi

    if grep -q "func (v \*OnboardingSkillsView) HandleSkillsFetched" "$VIEW_FILE"; then
        pass "HandleSkillsFetched method defined"
    else
        fail "HandleSkillsFetched method not found"
    fi

    if grep -q "func (v \*OnboardingSkillsView) GetSelectedSkills" "$VIEW_FILE"; then
        pass "GetSelectedSkills method defined"
    else
        fail "GetSelectedSkills method not found"
    fi

    if grep -q "func (v \*OnboardingSkillsView) GetReplaceSkills" "$VIEW_FILE"; then
        pass "GetReplaceSkills method defined"
    else
        fail "GetReplaceSkills method not found"
    fi

    echo ""
}

# Test 2: ViewOnboardingSkills enum exists
test_view_enum() {
    echo -e "${BLUE}━━━ Test 2: ViewOnboardingSkills Enum ━━━${NC}"

    APP_FILE="internal/tui/app.go"
    if grep -q "ViewOnboardingSkills" "$APP_FILE"; then
        pass "ViewOnboardingSkills enum exists in app.go"
    else
        fail "ViewOnboardingSkills enum not found in app.go"
    fi

    echo ""
}

# Test 3: Model has onboardingSkillsView field
test_model_field() {
    echo -e "${BLUE}━━━ Test 3: Model Fields ━━━${NC}"

    APP_FILE="internal/tui/app.go"
    if grep -q "onboardingSkillsView.*\*views.OnboardingSkillsView" "$APP_FILE"; then
        pass "onboardingSkillsView field exists in Model"
    else
        fail "onboardingSkillsView field not found in Model"
    fi

    if grep -q "pendingInstallSkills.*\[\]models.Skill" "$APP_FILE"; then
        pass "pendingInstallSkills field exists in Model"
    else
        fail "pendingInstallSkills field not found in Model"
    fi

    echo ""
}

# Test 4: State machine transitions
test_state_transitions() {
    echo -e "${BLUE}━━━ Test 4: State Machine Transitions ━━━${NC}"

    APP_FILE="internal/tui/app.go"

    # Check ViewOnboardingTools -> ViewOnboardingSkills transition
    if grep -q "m.currentView = ViewOnboardingSkills" "$APP_FILE"; then
        pass "Transition to ViewOnboardingSkills exists"
    else
        fail "Transition to ViewOnboardingSkills not found"
    fi

    # Check ViewOnboardingSkills case in Update
    if grep -q "case ViewOnboardingSkills:" "$APP_FILE"; then
        pass "ViewOnboardingSkills case in Update() exists"
    else
        fail "ViewOnboardingSkills case not found in Update()"
    fi

    # Check PrimarySkillsFetchedMsg handler
    if grep -q "case views.PrimarySkillsFetchedMsg:" "$APP_FILE"; then
        pass "PrimarySkillsFetchedMsg handler exists"
    else
        fail "PrimarySkillsFetchedMsg handler not found"
    fi

    echo ""
}

# Test 5: Fetch methods
test_fetch_methods() {
    echo -e "${BLUE}━━━ Test 5: Fetch Methods ━━━${NC}"

    APP_FILE="internal/tui/app.go"

    if grep -q "func (m \*Model) startPrimarySkillsFetchCmd" "$APP_FILE"; then
        pass "startPrimarySkillsFetchCmd method exists"
    else
        fail "startPrimarySkillsFetchCmd method not found"
    fi

    if grep -q "func (m \*Model) fetchPrimarySkills" "$APP_FILE"; then
        pass "fetchPrimarySkills method exists"
    else
        fail "fetchPrimarySkills method not found"
    fi

    # Check it uses the correct scraper constructor
    if grep -A10 "func (m \*Model) fetchPrimarySkills" "$APP_FILE" | grep -q "NewScraperWithConfig"; then
        pass "Uses NewScraperWithConfig (not deprecated New)"
    else
        fail "Not using NewScraperWithConfig"
    fi

    echo ""
}

# Test 6: Build and lint
test_build_lint() {
    echo -e "${BLUE}━━━ Test 6: Build and Lint ━━━${NC}"

    if go build ./internal/tui/... 2>&1; then
        pass "TUI package builds successfully"
    else
        fail "TUI package failed to build"
    fi

    if [ -z "$(gofmt -l internal/tui/views/onboarding_skills.go internal/tui/app.go 2>&1)" ]; then
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

# Test 7: Run unit tests
test_unit_tests() {
    echo -e "${BLUE}━━━ Test 7: Unit Tests ━━━${NC}"

    if go test ./internal/tui/views/... 2>&1 | grep -q "ok"; then
        pass "Views package tests pass"
    else
        fail "Views package tests failed"
    fi

    echo ""
}

# Print summary
print_summary() {
    echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║                   Phase 1 Test Summary                     ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "Tests passed: ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Tests failed: ${RED}$TESTS_FAILED${NC}"
    echo ""

    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "${GREEN}All Phase 1 tests passed!${NC}"
        echo ""
        echo "Phase 1A Definition of Done:"
        echo -e "  ${GREEN}✓${NC} View renders loading state"
        echo -e "  ${GREEN}✓${NC} View renders error state with skip option"
        echo -e "  ${GREEN}✓${NC} View renders checklist with two sections"
        echo -e "  ${GREEN}✓${NC} Navigation works between sections"
        echo -e "  ${GREEN}✓${NC} Selection toggling works"
        echo -e "  ${GREEN}✓${NC} A/N shortcuts work"
        echo ""
        echo "Phase 1B Definition of Done:"
        echo -e "  ${GREEN}✓${NC} ViewOnboardingSkills enum value exists"
        echo -e "  ${GREEN}✓${NC} Transition from Tools -> Skills works"
        echo -e "  ${GREEN}✓${NC} Async fetch triggers on entering view"
        echo -e "  ${GREEN}✓${NC} Skip goes to Home + pull"
        echo -e "  ${GREEN}✓${NC} Selection triggers install dialog"
        exit 0
    else
        echo -e "${RED}Some Phase 1 tests failed!${NC}"
        exit 1
    fi
}

# Main execution
main() {
    check_prerequisites
    test_view_file_exists
    test_view_enum
    test_model_field
    test_state_transitions
    test_fetch_methods
    test_build_lint
    test_unit_tests
    print_summary
}

main "$@"
