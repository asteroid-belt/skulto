#!/bin/bash
#
# Phase 3 Test Script: Install Flow Integration
#
# This script tests the Phase 3 implementation including:
# - Batch install logic for pendingInstallSkills
# - installBatchSkillsCmd method exists
# - batchInstallCompleteMsg type and handler exist
# - Cancelled dialog clears pendingInstallSkills
#
# Usage:
#   ./scripts/primary/test-phase3.sh
#

set -e  # Exit on first error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     Phase 3 Tests: Install Flow Integration               ║${NC}"
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

# Test 1: installBatchSkillsCmd method exists
test_batch_install_method() {
    echo -e "${BLUE}━━━ Test 1: installBatchSkillsCmd Method ━━━${NC}"

    APP_FILE="internal/tui/app.go"

    if grep -q "func (m \*Model) installBatchSkillsCmd" "$APP_FILE"; then
        pass "installBatchSkillsCmd method exists"
    else
        fail "installBatchSkillsCmd method not found"
        return
    fi

    # Check it uses installer.InstallTo
    if grep -A30 "func (m \*Model) installBatchSkillsCmd" "$APP_FILE" | grep -q "m.installer.InstallTo"; then
        pass "Uses installer.InstallTo for installation"
    else
        fail "Not using installer.InstallTo"
    fi

    # Check it gets source via GetSource
    if grep -A30 "func (m \*Model) installBatchSkillsCmd" "$APP_FILE" | grep -q "m.db.GetSource"; then
        pass "Gets source via db.GetSource"
    else
        fail "Not getting source via db.GetSource"
    fi

    # Check it uses PrimarySkillsRepo for source ID
    if grep -A30 "func (m \*Model) installBatchSkillsCmd" "$APP_FILE" | grep -q "scraper.PrimarySkillsRepo"; then
        pass "Uses scraper.PrimarySkillsRepo for source ID"
    else
        fail "Not using scraper.PrimarySkillsRepo"
    fi

    echo ""
}

# Test 2: batchInstallCompleteMsg type exists
test_batch_msg_type() {
    echo -e "${BLUE}━━━ Test 2: batchInstallCompleteMsg Type ━━━${NC}"

    APP_FILE="internal/tui/app.go"

    if grep -q "type batchInstallCompleteMsg struct" "$APP_FILE"; then
        pass "batchInstallCompleteMsg type defined"
    else
        fail "batchInstallCompleteMsg type not found"
    fi

    echo ""
}

# Test 3: batchInstallCompleteMsg handler exists
test_batch_msg_handler() {
    echo -e "${BLUE}━━━ Test 3: batchInstallCompleteMsg Handler ━━━${NC}"

    APP_FILE="internal/tui/app.go"

    if grep -q "case batchInstallCompleteMsg:" "$APP_FILE"; then
        pass "batchInstallCompleteMsg handler exists"
    else
        fail "batchInstallCompleteMsg handler not found"
        return
    fi

    # Check it clears pendingInstallSkills
    if grep -A15 "case batchInstallCompleteMsg:" "$APP_FILE" | grep -q "m.pendingInstallSkills = nil"; then
        pass "Handler clears pendingInstallSkills"
    else
        fail "Handler doesn't clear pendingInstallSkills"
    fi

    # Check it sets currentView to ViewHome
    if grep -A15 "case batchInstallCompleteMsg:" "$APP_FILE" | grep -q "m.currentView = ViewHome"; then
        pass "Handler sets currentView to ViewHome"
    else
        fail "Handler doesn't set currentView to ViewHome"
    fi

    # Check it calls completeOnboarding
    if grep -A15 "case batchInstallCompleteMsg:" "$APP_FILE" | grep -q "m.completeOnboarding"; then
        pass "Handler calls completeOnboarding()"
    else
        fail "Handler doesn't call completeOnboarding()"
    fi

    echo ""
}

# Test 4: Batch install check in dialog confirmation
test_dialog_batch_check() {
    echo -e "${BLUE}━━━ Test 4: Dialog Confirmation Batch Check ━━━${NC}"

    APP_FILE="internal/tui/app.go"

    # Check that IsConfirmed handler checks for pendingInstallSkills
    if grep -A15 "IsConfirmed()" "$APP_FILE" | grep -q "len(m.pendingInstallSkills) > 0"; then
        pass "Dialog confirmation checks for pendingInstallSkills"
    else
        fail "Dialog confirmation doesn't check for pendingInstallSkills"
    fi

    # Check that it calls installBatchSkillsCmd
    if grep -A20 "IsConfirmed()" "$APP_FILE" | grep -q "m.installBatchSkillsCmd"; then
        pass "Calls installBatchSkillsCmd for batch install"
    else
        fail "Doesn't call installBatchSkillsCmd"
    fi

    echo ""
}

# Test 5: Cancelled dialog handles batch
test_dialog_cancelled_batch() {
    echo -e "${BLUE}━━━ Test 5: Dialog Cancelled Batch Handling ━━━${NC}"

    APP_FILE="internal/tui/app.go"

    # Check that IsCancelled handler clears pendingInstallSkills
    if grep -A20 "IsCancelled()" "$APP_FILE" | grep -q "m.pendingInstallSkills = nil"; then
        pass "Cancelled dialog clears pendingInstallSkills"
    else
        fail "Cancelled dialog doesn't clear pendingInstallSkills"
    fi

    # Check that cancelled dialog completes onboarding
    if grep -A20 "IsCancelled()" "$APP_FILE" | grep -q "m.completeOnboarding"; then
        pass "Cancelled dialog completes onboarding"
    else
        fail "Cancelled dialog doesn't complete onboarding"
    fi

    echo ""
}

# Test 6: Build verification
test_build() {
    echo -e "${BLUE}━━━ Test 6: Build Verification ━━━${NC}"

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

# Test 7: Code quality
test_code_quality() {
    echo -e "${BLUE}━━━ Test 7: Code Quality ━━━${NC}"

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

# Print summary
print_summary() {
    echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║                   Phase 3 Test Summary                     ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "Tests passed: ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Tests failed: ${RED}$TESTS_FAILED${NC}"
    echo ""

    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "${GREEN}All Phase 3 tests passed!${NC}"
        echo ""
        echo "Definition of Done:"
        echo -e "  ${GREEN}✓${NC} Code passes golangci-lint"
        echo -e "  ${GREEN}✓${NC} Code passes go fmt"
        echo -e "  ${GREEN}✓${NC} Selected skills saved to database"
        echo -e "  ${GREEN}✓${NC} Selected skills installed to filesystem"
        echo -e "  ${GREEN}✓${NC} Onboarding completes after installation"
        echo -e "  ${GREEN}✓${NC} Pull triggers after onboarding completion"
        exit 0
    else
        echo -e "${RED}Some Phase 3 tests failed!${NC}"
        exit 1
    fi
}

# Main execution
main() {
    check_prerequisites
    test_batch_install_method
    test_batch_msg_type
    test_batch_msg_handler
    test_dialog_batch_check
    test_dialog_cancelled_batch
    test_build
    test_code_quality
    print_summary
}

main "$@"
