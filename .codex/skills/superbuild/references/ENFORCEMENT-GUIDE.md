# Enforcement Guide

Detailed guidance for enforcing Definition of Done and quality gates in superbuild.

---

## Definition of Done Verification

### Step-by-Step Verification Process

For each phase, run these checks in order:

```
┌─────────────────────────────────────────────────────────────────────┐
│                    DEFINITION OF DONE VERIFICATION                   │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  1. DETECT STACK      │  Identify language/framework                │
│          ↓            │                                             │
│  2. IDENTIFY FILES    │  What files were created/modified?         │
│          ↓            │                                             │
│  3. CHECK TESTS EXIST │  Do tests exist for new code?              │
│          ↓            │  NO → STOP and warn user                   │
│  4. RUN TESTS         │  Do all tests pass?                        │
│          ↓            │  NO → STOP and show failures               │
│  5. RUN LINTER        │  Does linter pass?                         │
│          ↓            │  NO → STOP and show errors                 │
│  6. RUN FORMATTER     │  Does formatter check pass?                │
│          ↓            │  NO → STOP and show diff                   │
│  7. RUN TYPE CHECKER  │  Does type checker pass?                   │
│          ↓            │  NO → STOP and show errors                 │
│  8. UPDATE PLAN       │  Check off tasks, update status            │
│          ↓            │  See references/PLAN-UPDATES.md            │
│  9. GENERATE COMMIT   │  All passed → generate commit message      │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Stack Detection

### Detection Rules

Run these checks in order to detect the stack:

```bash
# Check for config files (in priority order)
if [ -f "package.json" ]; then
    # Check for TypeScript
    if [ -f "tsconfig.json" ]; then
        STACK="typescript"
    else
        STACK="javascript"
    fi
elif [ -f "pyproject.toml" ] || [ -f "requirements.txt" ] || [ -f "setup.py" ]; then
    STACK="python"
elif [ -f "go.mod" ]; then
    STACK="go"
elif [ -f "Cargo.toml" ]; then
    STACK="rust"
elif [ -f "Gemfile" ]; then
    STACK="ruby"
elif [ -f "pom.xml" ] || [ -f "build.gradle" ]; then
    STACK="java"
else
    STACK="unknown"
fi
```

### Stack Indicators

| File | Stack | Sub-type |
|------|-------|----------|
| `package.json` + `tsconfig.json` | TypeScript | Node.js |
| `package.json` (no tsconfig) | JavaScript | Node.js |
| `pyproject.toml` | Python | Modern |
| `requirements.txt` | Python | Legacy |
| `go.mod` | Go | Modules |
| `Cargo.toml` | Rust | Cargo |
| `Gemfile` | Ruby | Bundler |
| `pom.xml` | Java | Maven |
| `build.gradle` | Java/Kotlin | Gradle |

---

## Test Verification

### Test File Location Patterns

#### JavaScript/TypeScript

| Source Pattern | Test Patterns (check all) |
|----------------|---------------------------|
| `src/foo.ts` | `src/foo.test.ts`, `src/foo.spec.ts`, `__tests__/foo.test.ts`, `test/foo.test.ts`, `tests/foo.test.ts` |
| `src/components/Bar.tsx` | `src/components/Bar.test.tsx`, `src/components/__tests__/Bar.test.tsx` |
| `src/services/auth.ts` | `src/services/auth.test.ts`, `tests/unit/services/auth.test.ts` |
| `src/api/users.ts` | `tests/integration/api/users.test.ts`, `src/api/users.integration.test.ts` |

#### Python

| Source Pattern | Test Patterns (check all) |
|----------------|---------------------------|
| `src/foo.py` | `tests/test_foo.py`, `src/test_foo.py`, `tests/foo_test.py` |
| `mypackage/bar.py` | `tests/test_bar.py`, `tests/mypackage/test_bar.py` |
| `app/services/auth.py` | `tests/services/test_auth.py`, `tests/unit/test_auth.py` |

#### Go

| Source Pattern | Test Patterns |
|----------------|---------------|
| `pkg/foo/bar.go` | `pkg/foo/bar_test.go` (MUST be same directory) |
| `internal/auth/jwt.go` | `internal/auth/jwt_test.go` |

#### Rust

| Source Pattern | Test Patterns |
|----------------|---------------|
| `src/lib.rs` | Tests in same file (`#[cfg(test)]` module) or `tests/` directory |
| `src/foo.rs` | `src/foo.rs` (inline tests) or `tests/foo.rs` |

### Test Existence Check Algorithm

```
FOR each source_file in changed_files:
    IF source_file is test file:
        SKIP (test files don't need tests)

    IF source_file is config/data file:
        SKIP (configs don't need unit tests)

    test_patterns = get_test_patterns(source_file, stack)
    found_test = false

    FOR each pattern in test_patterns:
        IF file_exists(pattern):
            found_test = true
            BREAK

    IF NOT found_test:
        ADD to missing_tests list

IF missing_tests is not empty:
    HALT with missing tests error
```

### Files That Don't Require Tests

| Pattern | Reason |
|---------|--------|
| `*.config.js`, `*.config.ts` | Configuration files |
| `*.d.ts` | Type declaration files |
| `*.json`, `*.yaml`, `*.yml` | Data files |
| `*.md` | Documentation |
| `*.test.*`, `*.spec.*`, `*_test.*` | Already test files |
| `__mocks__/*` | Mock files |
| `index.ts` (re-exports only) | Barrel files with no logic |

---

## Exit Code Interpretation

### Universal Exit Codes

| Code | Meaning | Action |
|------|---------|--------|
| 0 | Success | Continue to next check |
| 1 | Failure (errors found) | HALT and show errors |
| 2 | Misuse / invalid args | Check command syntax |
| 126 | Permission denied | Check file permissions |
| 127 | Command not found | Tool not installed |
| 130 | SIGINT (Ctrl+C) | User cancelled |
| 137 | SIGKILL (OOM) | Increase memory/timeout |
| 143 | SIGTERM | Process terminated |

### Tool-Specific Exit Codes

#### ESLint
| Code | Meaning |
|------|---------|
| 0 | No errors |
| 1 | Lint errors found |
| 2 | Config or internal error |

#### Prettier
| Code | Meaning |
|------|---------|
| 0 | All files formatted |
| 1 | Some files need formatting |
| 2 | Error running prettier |

#### Jest
| Code | Meaning |
|------|---------|
| 0 | All tests passed |
| 1 | Tests failed |

#### pytest
| Code | Meaning |
|------|---------|
| 0 | All tests passed |
| 1 | Tests failed |
| 2 | Test execution interrupted |
| 3 | Internal error |
| 4 | pytest usage error |
| 5 | No tests collected |

#### TypeScript (tsc)
| Code | Meaning |
|------|---------|
| 0 | No type errors |
| 1 | Type errors found |
| 2 | Config error |

---

## Quality Commands Reference

### JavaScript/TypeScript

```bash
# Package manager detection
if [ -f "pnpm-lock.yaml" ]; then PM="pnpm"
elif [ -f "yarn.lock" ]; then PM="yarn"
elif [ -f "bun.lockb" ]; then PM="bun"
else PM="npm"; fi

# Linter
$PM run lint 2>&1                    # If script exists
npx eslint . --ext .ts,.tsx 2>&1     # Direct invocation
npx eslint . --max-warnings 0 2>&1   # Treat warnings as errors

# Formatter
$PM run format:check 2>&1            # If script exists
npx prettier --check "src/**/*.{ts,tsx}" 2>&1

# Type Checker
$PM run typecheck 2>&1               # If script exists
npx tsc --noEmit 2>&1                # Direct invocation

# Tests
$PM test 2>&1                        # Standard
$PM test -- --coverage 2>&1          # With coverage
npx jest --passWithNoTests 2>&1      # Allow empty test suites
npx vitest run 2>&1                  # Vitest
```

### Python

```bash
# Virtual environment detection
if [ -d ".venv" ]; then source .venv/bin/activate; fi
if [ -d "venv" ]; then source venv/bin/activate; fi

# Linter (try in order)
ruff check . 2>&1                    # Fast, modern
pylint src 2>&1                      # Traditional
flake8 . 2>&1                        # Alternative

# Formatter
black --check . 2>&1                 # Black
ruff format --check . 2>&1           # Ruff formatter

# Type Checker
mypy . 2>&1                          # MyPy
pyright 2>&1                         # Pyright (faster)

# Tests
pytest 2>&1                          # Standard
pytest --cov=src 2>&1                # With coverage
python -m pytest 2>&1                # Module invocation
python -m unittest discover 2>&1     # Unittest
```

### Go

```bash
# Linter
golangci-lint run 2>&1               # Comprehensive (preferred)
go vet ./... 2>&1                    # Built-in (basic)
staticcheck ./... 2>&1               # Alternative

# Formatter (Go has mandatory formatting)
gofmt -l . 2>&1                      # List unformatted (check mode)
# Exit 0 if output is empty, exit 1 if files listed
test -z "$(gofmt -l .)"

# Type Checker (implicit in build)
go build ./... 2>&1

# Tests
go test ./... 2>&1                   # All tests
go test ./... -v 2>&1                # Verbose
go test ./... -cover 2>&1            # With coverage
go test ./... -race 2>&1             # Race detection
```

### Rust

```bash
# Linter
cargo clippy 2>&1                    # Standard
cargo clippy -- -D warnings 2>&1     # Treat warnings as errors
cargo clippy --all-targets 2>&1      # Include tests/benches

# Formatter
cargo fmt --check 2>&1               # Check mode
cargo fmt -- --check 2>&1            # Alternative syntax

# Type Checker (implicit in check/build)
cargo check 2>&1                     # Fast check
cargo build 2>&1                     # Full build

# Tests
cargo test 2>&1                      # All tests
cargo test -- --nocapture 2>&1       # Show stdout
cargo test --all-features 2>&1       # All feature flags
```

---

## Timeout Handling

### Default Timeouts

| Check | Default | Max |
|-------|---------|-----|
| Lint | 60s | 300s |
| Format | 30s | 120s |
| Type check | 120s | 600s |
| Tests | 300s | 900s |

### Timeout Failure Message

```
⛔ DEFINITION OF DONE FAILED - Timeout
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Check: Tests
Command: npm test
Timeout: 300 seconds exceeded

Possible causes:
1. Tests are hanging (infinite loop, unresolved promise)
2. Tests are slow (consider parallelization)
3. External service dependency is down

Action Required:
1. Run tests locally to identify slow/hanging tests
2. Check for async operations without proper cleanup
3. Increase timeout if tests are legitimately slow

[EXECUTION HALTED]
```

---

## Flaky Test Detection

### Signs of Flaky Tests

| Symptom | Likely Cause |
|---------|--------------|
| Pass locally, fail in CI | Environment dependency |
| Fail intermittently | Race condition |
| Fail after unrelated changes | Shared state / test pollution |
| Timeout randomly | Network dependency |

### Flaky Test Response

```
⚠️  POTENTIAL FLAKY TEST DETECTED
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Test: should fetch user data
File: tests/api/users.test.ts

Behavior: Passed on first run, failed on second run

This MAY be a flaky test. Before proceeding:

1. Run the test 3 more times:
   npm test -- --testNamePattern="should fetch user data"

2. If it fails inconsistently:
   - Check for race conditions
   - Check for shared state
   - Check for network/timing dependencies

3. If it passes consistently:
   - May have been a one-time failure
   - Proceed with caution

Recommendation: Fix flaky tests before continuing.
Flaky tests erode confidence in the test suite.

[AWAITING USER DECISION]
```

---

## Test Failure Output Parsing

### Jest Output Pattern

```
FAIL src/services/auth.test.ts
  ● Auth Service › should validate token

    expect(received).toBe(expected) // Object.is equality

    Expected: true
    Received: false

      23 |     const result = validateToken(token);
    > 24 |     expect(result).toBe(true);
         |                    ^
      25 |   });

      at Object.<anonymous> (src/services/auth.test.ts:24:20)
```

**Extract:**
- File: `src/services/auth.test.ts`
- Test: `Auth Service › should validate token`
- Line: 24
- Expected: `true`
- Received: `false`

### pytest Output Pattern

```
FAILED tests/test_auth.py::test_validate_token - AssertionError: assert False == True
```

**Extract:**
- File: `tests/test_auth.py`
- Test: `test_validate_token`
- Error: `AssertionError`

### Go Test Output Pattern

```
--- FAIL: TestValidateToken (0.00s)
    auth_test.go:24: expected true, got false
FAIL
```

**Extract:**
- File: `auth_test.go`
- Test: `TestValidateToken`
- Line: 24

---

## Parallel Phase Coordination

### Sub-Agent Instructions Template

```
You are executing Phase [X] of the implementation plan.

═══════════════════════════════════════════════════════════════
CONTEXT
═══════════════════════════════════════════════════════════════

[Paste relevant plan sections for this phase]

═══════════════════════════════════════════════════════════════
REQUIREMENTS
═══════════════════════════════════════════════════════════════

1. Execute ONLY Phase [X] - do not touch other phases
2. Follow the plan's specifications exactly
3. Write tests FIRST if plan specifies TDD
4. Verify Definition of Done before reporting complete:
   - Tests exist for new code
   - All tests pass
   - Linter passes
   - Formatter passes
   - Type checker passes
5. Update plan document (check off tasks, update status)
6. Generate conventional commit message

═══════════════════════════════════════════════════════════════
RETURN FORMAT (JSON)
═══════════════════════════════════════════════════════════════

{
  "phase": "[X]",
  "status": "complete" | "failed",
  "dod_checklist": {
    "tests_exist": true,
    "tests_pass": true,
    "linter_pass": true,
    "formatter_pass": true,
    "typechecker_pass": true,
    "plan_updated": true
  },
  "commit_message": "feat(scope): description\n\nbody...",
  "files_changed": [
    "src/services/auth.ts",
    "tests/services/auth.test.ts"
  ],
  "failure_reason": null
}
```

### Result Aggregation

```
PARALLEL PHASES COMPLETE
━━━━━━━━━━━━━━━━━━━━━━━━

| Phase | Status | Tests | Lint | Format | Types | Plan |
|-------|--------|-------|------|--------|-------|------|
| 2A | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 2B | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 2C | ⛔ | ✅ | ⛔ | ✅ | ✅ | ❌ |

⛔ Phase 2C failed Definition of Done

Issues:
1. Linter: 3 errors in tests/edge-cases/validation.test.ts
2. Plan: Tasks not checked off

The parallel phase group cannot complete until ALL phases pass.
Please fix Phase 2C issues, then tell me to re-verify.

[EXECUTION HALTED]
```

---

## Common Edge Cases

### Phase Has No Code Changes

```
Phase [X]: Documentation Update

Content: README.md, docs/*.md only

Quality gates:
- Tests: N/A (no code)
- Linter: N/A (no code)
- Formatter: ⚠️  Check markdown lint if available
- Type checker: N/A (no code)

Generating commit message for documentation changes...
```

### Phase Modifies Only Tests

```
Phase [X]: Test Coverage Improvement

Content: Test files only

Quality gates:
- Test existence: N/A (these ARE the tests)
- Tests pass: ✅ All 12 tests pass
- Linter: ✅ (test files are linted)
- Formatter: ✅ (test files are formatted)
- Type checker: ✅ (test files are typed)
```

### Phase Modifies Only Config

```
Phase [X]: Build Configuration

Content: *.config.js, *.json, *.yaml only

Quality gates:
- Tests: N/A (config doesn't need unit tests)
- Linter: ✅ (if config linting enabled)
- Formatter: ✅ (if config formatting enabled)
- Type checker: N/A

Note: Config changes should be tested via integration/E2E tests
in subsequent phases.
```

### Tool Not Installed

```
⚠️  TOOL NOT AVAILABLE
━━━━━━━━━━━━━━━━━━━━━━

Tool: eslint
Check: Linter
Status: Command not found (exit 127)

Options:
1. Install the tool:
   npm install -D eslint

2. Use alternative:
   - biome (npm install -D @biomejs/biome)

3. Skip this check (NOT RECOMMENDED):
   Only skip if you're certain no linting is needed.
   This will be noted in the commit message.

Which option do you prefer?
```

---

## Build-All Override Protocol

When `--build-all` flag is present:

```
⚠️  BUILD-ALL MODE ACTIVE
━━━━━━━━━━━━━━━━━━━━━━━━

Executing all phases sequentially without pause.
DoD enforcement is STILL ACTIVE - any failure will halt.

Progress:
─────────────────────────────────────────────────────

Phase 0: Bootstrap
  Status: ✅ Complete
  Commit: [queued]

Phase 1: Setup
  Status: ✅ Complete
  Commit: [queued]

Phase 2A: Backend
  Status: 🔄 In Progress
  Tests: Running...

─────────────────────────────────────────────────────

[HALTED - Phase 2A tests failing]

Commits generated before failure:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Phase 0:
chore(bootstrap): add eslint and prettier configuration

Phase 1:
feat(setup): initialize database schema and migrations

[Phase 2A: No commit - incomplete]
```
