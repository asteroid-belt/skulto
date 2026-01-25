#!/bin/bash
#
# Phase 4 Test Script: Unit Tests
#
# This script tests the Phase 4 implementation including:
# - OnboardingSkillsView unit tests exist and pass
# - Seeds unit tests exist and pass
# - Test coverage meets requirements
#
# Usage:
#   ./scripts/primary/test-phase4.sh
#

set -e  # Exit on first error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     Phase 4 Tests: Unit Tests                              ║${NC}"
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

# Test 1: OnboardingSkillsView test file exists
test_onboarding_test_file() {
    echo -e "${BLUE}━━━ Test 1: OnboardingSkillsView Test File ━━━${NC}"

    TEST_FILE="internal/tui/views/onboarding_skills_test.go"

    if [ -f "$TEST_FILE" ]; then
        pass "onboarding_skills_test.go exists"
    else
        fail "onboarding_skills_test.go not found"
        return
    fi

    # Check for key test functions
    if grep -q "TestOnboardingSkillsViewInit" "$TEST_FILE"; then
        pass "TestOnboardingSkillsViewInit exists"
    else
        fail "TestOnboardingSkillsViewInit not found"
    fi

    if grep -q "TestOnboardingSkillsViewHandlesFetch" "$TEST_FILE"; then
        pass "TestOnboardingSkillsViewHandlesFetch exists"
    else
        fail "TestOnboardingSkillsViewHandlesFetch not found"
    fi

    if grep -q "TestOnboardingSkillsViewClassifiesExisting" "$TEST_FILE"; then
        pass "TestOnboardingSkillsViewClassifiesExisting exists"
    else
        fail "TestOnboardingSkillsViewClassifiesExisting not found"
    fi

    if grep -q "TestOnboardingSkillsViewNavigation" "$TEST_FILE"; then
        pass "TestOnboardingSkillsViewNavigation exists"
    else
        fail "TestOnboardingSkillsViewNavigation not found"
    fi

    if grep -q "TestOnboardingSkillsViewToggle" "$TEST_FILE"; then
        pass "TestOnboardingSkillsViewToggle exists"
    else
        fail "TestOnboardingSkillsViewToggle not found"
    fi

    if grep -q "TestOnboardingSkillsViewSelectAll" "$TEST_FILE"; then
        pass "TestOnboardingSkillsViewSelectAll exists"
    else
        fail "TestOnboardingSkillsViewSelectAll not found"
    fi

    if grep -q "TestOnboardingSkillsViewSkip" "$TEST_FILE"; then
        pass "TestOnboardingSkillsViewSkip exists"
    else
        fail "TestOnboardingSkillsViewSkip not found"
    fi

    if grep -q "TestOnboardingSkillsViewContinue" "$TEST_FILE"; then
        pass "TestOnboardingSkillsViewContinue exists"
    else
        fail "TestOnboardingSkillsViewContinue not found"
    fi

    if grep -q "TestOnboardingSkillsViewGetSelected" "$TEST_FILE"; then
        pass "TestOnboardingSkillsViewGetSelected exists"
    else
        fail "TestOnboardingSkillsViewGetSelected not found"
    fi

    echo ""
}

# Test 2: Seeds test file exists
test_seeds_test_file() {
    echo -e "${BLUE}━━━ Test 2: Seeds Test File ━━━${NC}"

    TEST_FILE="internal/scraper/seeds_test.go"

    if [ -f "$TEST_FILE" ]; then
        pass "seeds_test.go exists"
    else
        fail "seeds_test.go not found"
        return
    fi

    if grep -q "TestPrimarySkillsRepoExists" "$TEST_FILE"; then
        pass "TestPrimarySkillsRepoExists exists"
    else
        fail "TestPrimarySkillsRepoExists not found"
    fi

    if grep -q "TestAllSeedsIncludesPrimary" "$TEST_FILE"; then
        pass "TestAllSeedsIncludesPrimary exists"
    else
        fail "TestAllSeedsIncludesPrimary not found"
    fi

    echo ""
}

# Test 3: OnboardingSkillsView tests pass
test_onboarding_tests_pass() {
    echo -e "${BLUE}━━━ Test 3: OnboardingSkillsView Tests Pass ━━━${NC}"

    if go test ./internal/tui/views/... -run "Onboarding" -v 2>&1 | grep -q "PASS"; then
        pass "OnboardingSkillsView tests pass"
    else
        fail "OnboardingSkillsView tests failed"
    fi

    echo ""
}

# Test 4: Seeds tests pass
test_seeds_tests_pass() {
    echo -e "${BLUE}━━━ Test 4: Seeds Tests Pass ━━━${NC}"

    if go test ./internal/scraper/... -run "Primary|Seeds" -v 2>&1 | grep -q "PASS"; then
        pass "Seeds tests pass"
    else
        fail "Seeds tests failed"
    fi

    echo ""
}

# Test 5: Test coverage meets requirements
test_coverage() {
    echo -e "${BLUE}━━━ Test 5: Test Coverage ━━━${NC}"

    # Run tests with coverage
    go test ./internal/tui/views/... -coverprofile=/tmp/coverage.out -covermode=atomic > /dev/null 2>&1

    # Check coverage for onboarding_skills.go
    COVERAGE=$(go tool cover -func=/tmp/coverage.out | grep "onboarding_skills.go" | awk '{sum += $3; count++} END {print sum/count}' | cut -d'.' -f1)

    if [ -n "$COVERAGE" ] && [ "$COVERAGE" -ge 80 ]; then
        pass "OnboardingSkillsView coverage is ${COVERAGE}% (>= 80%)"
    else
        fail "OnboardingSkillsView coverage is ${COVERAGE}% (< 80%)"
    fi

    # Show detailed coverage for key functions
    echo ""
    echo -e "${YELLOW}Detailed coverage for onboarding_skills.go:${NC}"
    go tool cover -func=/tmp/coverage.out | grep "onboarding_skills.go" | head -20

    echo ""
}

# Test 6: Full test suite passes
test_full_suite() {
    echo -e "${BLUE}━━━ Test 6: Full Test Suite ━━━${NC}"

    if go test ./internal/tui/views/... 2>&1 | grep -q "ok"; then
        pass "Views package tests pass"
    else
        fail "Views package tests failed"
    fi

    if go test ./internal/scraper/... 2>&1 | grep -q "ok"; then
        pass "Scraper package tests pass"
    else
        fail "Scraper package tests failed"
    fi

    echo ""
}

# Test 7: Code quality
test_code_quality() {
    echo -e "${BLUE}━━━ Test 7: Code Quality ━━━${NC}"

    if [ -z "$(gofmt -l internal/tui/views/onboarding_skills_test.go 2>&1)" ]; then
        pass "Test code passes go fmt"
    else
        fail "Test code needs formatting"
    fi

    if golangci-lint run ./internal/tui/views/... 2>&1; then
        pass "Code passes golangci-lint"
    else
        fail "golangci-lint found issues"
    fi

    echo ""
}

# Print summary
print_summary() {
    echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║                   Phase 4 Test Summary                     ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "Tests passed: ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Tests failed: ${RED}$TESTS_FAILED${NC}"
    echo ""

    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "${GREEN}All Phase 4 tests passed!${NC}"
        echo ""
        echo "Definition of Done:"
        echo -e "  ${GREEN}✓${NC} All unit tests pass"
        echo -e "  ${GREEN}✓${NC} Test coverage for OnboardingSkillsView >= 80%"
        echo -e "  ${GREEN}✓${NC} Tests cover: init, fetch handling, classification, navigation, toggle, select all, skip, continue"
        echo -e "  ${GREEN}✓${NC} Seeds test verifies primary repo exists"
        exit 0
    else
        echo -e "${RED}Some Phase 4 tests failed!${NC}"
        exit 1
    fi
}

# Main execution
main() {
    check_prerequisites
    test_onboarding_test_file
    test_seeds_test_file
    test_onboarding_tests_pass
    test_seeds_tests_pass
    test_coverage
    test_full_suite
    test_code_quality
    print_summary
}

main "$@"
