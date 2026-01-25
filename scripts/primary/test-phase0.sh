#!/bin/bash
#
# Phase 0 Test Script: Primary Skills Repository Seeds
#
# This script tests the Phase 0 implementation including:
# - PrimarySkillsRepo constant is defined and exported
# - Primary repo is included in OfficialSeeds
# - Primary repo appears in AllSeeds() output
# - Code passes golangci-lint and go fmt
#
# Usage:
#   ./scripts/primary/test-phase0.sh
#

set -e  # Exit on first error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     Phase 0 Tests: Primary Skills Repository Seeds        ║${NC}"
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

    # Check Go is installed
    if ! command -v go &> /dev/null; then
        echo -e "${RED}✗ Go is not installed${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ Go is installed ($(go version | cut -d' ' -f3))${NC}"

    # Check we're in the right directory
    if [ ! -f "go.mod" ] || ! grep -q "skulto" go.mod; then
        echo -e "${RED}✗ Not in skulto project directory${NC}"
        echo "Please run this script from the project root"
        exit 1
    fi
    echo -e "${GREEN}✓ In skulto project directory${NC}"

    echo ""
}

# Test 1: Code passes go fmt
test_go_fmt() {
    echo -e "${BLUE}━━━ Test 1: go fmt ━━━${NC}"

    OUTPUT=$(gofmt -l internal/scraper/seeds.go)
    if [ -z "$OUTPUT" ]; then
        pass "Code passes go fmt"
    else
        fail "Code needs formatting: $OUTPUT"
    fi

    echo ""
}

# Test 2: Code passes golangci-lint
test_golangci_lint() {
    echo -e "${BLUE}━━━ Test 2: golangci-lint ━━━${NC}"

    if ! command -v golangci-lint &> /dev/null; then
        echo -e "${YELLOW}⚠ golangci-lint not installed, skipping${NC}"
        echo ""
        return
    fi

    if golangci-lint run ./internal/scraper/seeds.go 2>&1; then
        pass "Code passes golangci-lint"
    else
        fail "golangci-lint found issues"
    fi

    echo ""
}

# Test 3: Package builds successfully
test_build() {
    echo -e "${BLUE}━━━ Test 3: Package Build ━━━${NC}"

    if go build ./internal/scraper/... 2>&1; then
        pass "Scraper package builds successfully"
    else
        fail "Scraper package failed to build"
    fi

    echo ""
}

# Test 4: PrimarySkillsRepo constant exists
test_primary_skills_repo_exists() {
    echo -e "${BLUE}━━━ Test 4: PrimarySkillsRepo Constant ━━━${NC}"

    if grep -q "var PrimarySkillsRepo = SeedRepository{" internal/scraper/seeds.go; then
        pass "PrimarySkillsRepo constant is defined"
    else
        fail "PrimarySkillsRepo constant not found"
        return
    fi

    # Check it has correct values
    if grep -A5 "var PrimarySkillsRepo" internal/scraper/seeds.go | grep -q 'Owner:.*"asteroid-belt"'; then
        pass "PrimarySkillsRepo.Owner = asteroid-belt"
    else
        fail "PrimarySkillsRepo.Owner is incorrect"
    fi

    if grep -A5 "var PrimarySkillsRepo" internal/scraper/seeds.go | grep -q 'Repo:.*"skills"'; then
        pass "PrimarySkillsRepo.Repo = skills"
    else
        fail "PrimarySkillsRepo.Repo is incorrect"
    fi

    if grep -A5 "var PrimarySkillsRepo" internal/scraper/seeds.go | grep -q 'Priority:.*10'; then
        pass "PrimarySkillsRepo.Priority = 10"
    else
        fail "PrimarySkillsRepo.Priority is incorrect"
    fi

    if grep -A5 "var PrimarySkillsRepo" internal/scraper/seeds.go | grep -q 'Type:.*"official"'; then
        pass "PrimarySkillsRepo.Type = official"
    else
        fail "PrimarySkillsRepo.Type is incorrect"
    fi

    echo ""
}

# Test 5: Primary repo in OfficialSeeds
test_primary_in_official_seeds() {
    echo -e "${BLUE}━━━ Test 5: Primary Repo in OfficialSeeds ━━━${NC}"

    if grep -A10 "var OfficialSeeds" internal/scraper/seeds.go | grep -q 'asteroid-belt.*skills'; then
        pass "Primary repo (asteroid-belt/skills) is in OfficialSeeds"
    else
        fail "Primary repo not found in OfficialSeeds"
    fi

    # Check it's first in the list
    FIRST_ENTRY=$(grep -A3 "var OfficialSeeds" internal/scraper/seeds.go | grep -m1 "Owner:" | sed 's/.*Owner:[[:space:]]*"\([^"]*\)".*/\1/')
    if [ "$FIRST_ENTRY" = "asteroid-belt" ]; then
        pass "Primary repo is first in OfficialSeeds"
    else
        fail "Primary repo is not first in OfficialSeeds (found: $FIRST_ENTRY)"
    fi

    echo ""
}

# Test 6: Run unit tests
test_unit_tests() {
    echo -e "${BLUE}━━━ Test 6: Unit Tests ━━━${NC}"

    if go test -v ./internal/scraper/... -run "TestAllSeeds" 2>&1 | grep -q "PASS"; then
        pass "TestAllSeeds passes"
    else
        fail "TestAllSeeds failed"
    fi

    echo ""
}

# Print summary
print_summary() {
    echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║                   Phase 0 Test Summary                     ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "Tests passed: ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Tests failed: ${RED}$TESTS_FAILED${NC}"
    echo ""

    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "${GREEN}All Phase 0 tests passed!${NC}"
        echo ""
        echo "Definition of Done:"
        echo -e "  ${GREEN}✓${NC} Code passes golangci-lint"
        echo -e "  ${GREEN}✓${NC} Code passes go fmt"
        echo -e "  ${GREEN}✓${NC} Primary repo appears in AllSeeds() output"
        echo -e "  ${GREEN}✓${NC} PrimarySkillsRepo constant is accessible"
        exit 0
    else
        echo -e "${RED}Some Phase 0 tests failed!${NC}"
        exit 1
    fi
}

# Main execution
main() {
    check_prerequisites
    test_go_fmt
    test_golangci_lint
    test_build
    test_primary_skills_repo_exists
    test_primary_in_official_seeds
    test_unit_tests
    print_summary
}

main "$@"
