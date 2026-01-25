#!/bin/bash
#
# Primary Skills Onboarding - All Phases Test Script
#
# Runs all phase tests in sequence.
#
# Usage:
#   ./scripts/primary/test-all.sh
#

set -e  # Exit on first error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   Primary Skills Onboarding - Complete Test Suite         ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""

PHASES_PASSED=0
PHASES_FAILED=0

run_phase() {
    local phase=$1
    local script=$2

    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}Running Phase $phase Tests${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""

    if "$SCRIPT_DIR/$script"; then
        echo ""
        echo -e "${GREEN}Phase $phase: PASSED${NC}"
        PHASES_PASSED=$((PHASES_PASSED + 1))
    else
        echo ""
        echo -e "${RED}Phase $phase: FAILED${NC}"
        PHASES_FAILED=$((PHASES_FAILED + 1))
    fi
    echo ""
}

# Run all phase tests
run_phase "0 (Seeds)" "test-phase0.sh"
run_phase "1 (Onboarding View)" "test-phase1.sh"
run_phase "2 (Auto-Sync)" "test-phase2.sh"
run_phase "3 (Install Flow)" "test-phase3.sh"
run_phase "4 (Unit Tests)" "test-phase4.sh"

# Final summary
echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                    Complete Test Summary                   ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "Phases passed: ${GREEN}$PHASES_PASSED${NC}"
echo -e "Phases failed: ${RED}$PHASES_FAILED${NC}"
echo ""

if [ $PHASES_FAILED -eq 0 ]; then
    echo -e "${GREEN}All phases passed!${NC}"
    echo ""
    echo "Implementation Complete:"
    echo -e "  ${GREEN}✓${NC} Phase 0: Primary repo added to seeds"
    echo -e "  ${GREEN}✓${NC} Phase 1A: OnboardingSkillsView created"
    echo -e "  ${GREEN}✓${NC} Phase 1B: App state machine integrated"
    echo -e "  ${GREEN}✓${NC} Phase 2: Auto-sync on startup and pull"
    echo -e "  ${GREEN}✓${NC} Phase 3: Install flow integration"
    echo -e "  ${GREEN}✓${NC} Phase 4: Unit tests with >= 80% coverage"
    exit 0
else
    echo -e "${RED}Some phases failed!${NC}"
    exit 1
fi
