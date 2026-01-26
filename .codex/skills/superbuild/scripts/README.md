# Superbuild Scripts

Automation scripts for superbuild phase validation.

## validate-phase.sh

Validates Definition of Done for a phase by running all quality checks.

### Usage

```bash
# Auto-detect stack and run all checks
./validate-phase.sh

# Specify stack explicitly
./validate-phase.sh --stack js
./validate-phase.sh --stack py
./validate-phase.sh --stack go
./validate-phase.sh --stack rust

# Skip tests (for phases without code changes)
./validate-phase.sh --skip-tests
```

### Exit Codes

- `0` - All checks passed
- `1` - One or more checks failed

### Checks Performed

| Check | JS/TS | Python | Go | Rust |
|-------|-------|--------|-------|------|
| Linter | eslint | ruff/pylint | golangci-lint/go vet | cargo clippy |
| Formatter | prettier | black | gofmt | cargo fmt |
| Type Checker | tsc | mypy/pyright | go build | cargo check |
| Tests | jest/npm test | pytest | go test | cargo test |

### Example Output

```
==============================================
DEFINITION OF DONE VALIDATION
Stack: js
==============================================

Running: Linter
Command: npm run lint 2>&1
---
✓ Linter: PASSED

Running: Formatter
Command: npm run format:check 2>&1
---
✓ Formatter: PASSED

Running: Type Checker
Command: npm run typecheck 2>&1
---
✓ Type Checker: PASSED

Running: Tests
Command: npm test 2>&1
---
✓ Tests: PASSED

==============================================
VALIDATION SUMMARY
==============================================
  ✓ Linter: PASS
  ✓ Formatter: PASS
  ✓ Type Checker: PASS
  ✓ Tests: PASS

══════════════════════════════════════════════
DEFINITION OF DONE: PASSED
══════════════════════════════════════════════
```

## Integration with Superbuild

The agent can invoke this script during phase execution:

```bash
# Run validation
./scripts/validate-phase.sh --stack js

# Check exit code
if [ $? -eq 0 ]; then
    echo "DoD passed - generate commit message"
else
    echo "DoD failed - halt execution"
fi
```
