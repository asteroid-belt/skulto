# Verification Reference

> Fresh verification after sub-agent completion and commit handling by mode.

## Trust But Verify Principle

**Sub-agent claims are reports, not evidence.**

After a sub-agent reports success, the main agent MUST independently verify:
1. All quality gates pass
2. Plan file was actually updated
3. Expected files exist

This catches:
- Sub-agent hallucinations
- Partial completions reported as success
- Quality gate regressions from parallel work

---

## Fresh Verification Sequence

```
VERIFICATION SEQUENCE
=====================

1. COLLECT sub-agent result JSON
2. PARSE status and quality gate claims
3. RUN each quality gate command fresh
4. COMPARE fresh results to claims
5. VERIFY plan file updates exist
6. VERIFY files created/modified exist
7. DETERMINE true pass/fail status
```

### Verification Output Format

```
VERIFICATION - Phase {PHASE_ID}
===============================

Sub-agent claimed: {STATUS}

Running fresh verification...

Quality Gates:
┌────────────┬─────────┬─────────────────────────────┐
│ Check      │ Result  │ Output                      │
├────────────┼─────────┼─────────────────────────────┤
│ Lint       │ PASS    │ No errors                   │
│ Format     │ PASS    │ All files formatted         │
│ Typecheck  │ PASS    │ No type errors              │
│ Test       │ PASS    │ 47 passed, 0 failed         │
└────────────┴─────────┴─────────────────────────────┘

Plan Updates:
- Tasks checked: YES (4/4 items)
- DoD checked: YES (6/6 items)
- Status updated: YES (⬜ -> ✅)

Files:
- Created: 2/2 exist
- Modified: 1/1 exist

VERIFICATION: PASSED
```

---

## Running Quality Gates

### Command Execution

Run each quality gate command and capture output:

```python
def run_quality_gate(name, command, timeout=120):
    result = Bash(
        command=command,
        timeout=timeout * 1000,
        description=f"Verify {name}"
    )

    return {
        "name": name,
        "command": command,
        "passed": result.exit_code == 0,
        "output": result.output[:500],  # Truncate for state file
        "exit_code": result.exit_code
    }
```

### Stack-Specific Commands

| Stack | Lint | Format | Typecheck | Test |
|-------|------|--------|-----------|------|
| TypeScript/Node | `npm run lint` | `npm run format:check` | `npm run typecheck` or `npx tsc --noEmit` | `npm test` |
| Python | `ruff check .` or `pylint src` | `black --check .` or `ruff format --check .` | `mypy .` or `pyright` | `pytest` |
| Go | `golangci-lint run` | `test -z "$(gofmt -l .)"` | `go build ./...` | `go test ./...` |
| Rust | `cargo clippy` | `cargo fmt --check` | `cargo check` | `cargo test` |

### Handling Missing Tools

If a quality gate command fails with "command not found":

```
QUALITY GATE UNAVAILABLE
========================

Check: Lint
Command: npm run lint
Error: Script "lint" not found in package.json

Options:
1. Skip this check (record as N/A)
2. Halt and require tool setup

Proceeding with skip...

Note: Skipped checks will be marked in state file.
```

---

## Verification Mismatch Handling

### Sub-Agent Claimed Success, Verification Fails

```
VERIFICATION MISMATCH
=====================

Sub-agent claimed: complete
Fresh verification: FAILED

Discrepancy:
┌────────────┬───────────────┬─────────────────┐
│ Check      │ Sub-agent     │ Fresh Result    │
├────────────┼───────────────┼─────────────────┤
│ Lint       │ passed        │ FAILED (3 err)  │
│ Test       │ 47 passed     │ FAILED (2 fail) │
└────────────┴───────────────┴─────────────────┘

This indicates a sub-agent error. Common causes:
1. Sub-agent ran checks before final changes
2. Sub-agent hallucinated passing results
3. Parallel phase caused regression

Action: Treating as phase FAILURE. Initiating retry...
```

### Sub-Agent Claimed Failure, Verification Passes

Rare but possible (sub-agent was too pessimistic):

```
VERIFICATION UPGRADE
====================

Sub-agent claimed: failed
Fresh verification: PASSED

All quality gates pass. Possible causes:
1. Sub-agent made fixes after reporting failure
2. Sub-agent misread output
3. Flaky test passed on retry

Action: Accepting as SUCCESS (verification is authoritative)
```

---

## Plan Update Verification

### Checking Task Completion

```python
def verify_plan_updates(plan_path, phase_id, expected_tasks):
    plan_content = Read(plan_path)

    # Find phase section
    phase_section = extract_phase_section(plan_content, phase_id)

    # Count checked items
    checked = phase_section.count("- [x]")
    unchecked = phase_section.count("- [ ]")

    return {
        "tasks_checked": checked,
        "tasks_unchecked": unchecked,
        "expected": expected_tasks,
        "all_complete": unchecked == 0 and checked >= expected_tasks
    }
```

### Plan Update Failure

If plan wasn't updated:

```
PLAN UPDATE VERIFICATION FAILED
===============================

Phase: 2A
Expected: 4 tasks checked
Found: 2 tasks checked, 2 unchecked

Sub-agent did not complete plan updates.

Action: Main agent will update plan file directly.
```

Main agent then updates the plan:

```python
# Update remaining checkboxes
for task in unchecked_tasks:
    Edit(
        file_path=plan_path,
        old_string=f"- [ ] {task}",
        new_string=f"- [x] {task}"
    )
```

---

## File Existence Verification

### Checking Created Files

```python
def verify_files_exist(files_created, files_modified):
    missing = []

    for file_path in files_created + files_modified:
        try:
            Read(file_path, limit=1)  # Just check existence
        except FileNotFoundError:
            missing.append(file_path)

    return {
        "all_exist": len(missing) == 0,
        "missing": missing
    }
```

### Missing Files Response

```
FILE VERIFICATION FAILED
========================

Expected files from sub-agent result:
- src/api/auth.ts (MISSING)
- src/api/auth.test.ts (EXISTS)

Sub-agent claimed to create src/api/auth.ts but file does not exist.

Action: Treating as phase FAILURE. Initiating retry...
```

---

## Commit Handling by Mode

### --commit=auto

Auto-commit after verification passes:

```
VERIFICATION PASSED - AUTO-COMMIT
=================================

Staging files:
  git add src/api/auth.ts src/api/auth.test.ts src/api/routes.ts

Creating commit:
  git commit -m "$(cat <<'EOF'
feat(api): implement JWT token validation

- Add validateToken function
- Add token refresh endpoint
- Add comprehensive test coverage
EOF
)"

Commit created: a1b2c3d

State file updated with commit SHA.
```

### Git Add Strategy

Stage only files reported by sub-agent:

```python
def stage_files(files_created, files_modified):
    all_files = files_created + files_modified

    # Stage specific files, not "git add ."
    if all_files:
        file_list = " ".join(f'"{f}"' for f in all_files)
        Bash(command=f"git add {file_list}")
```

### Commit Message Sanitization

Ensure commit message is shell-safe:

```python
def sanitize_commit_message(message):
    # Remove dangerous characters
    dangerous = ['"', '`', '$', '!', '\\', ';', '&', '|', '>', '<', '*', '?']
    for char in dangerous:
        message = message.replace(char, '')

    # Ensure proper line endings
    message = message.replace('\r\n', '\n')

    return message.strip()
```

### HEREDOC Commit Format

Always use HEREDOC for multi-line commits:

```bash
git commit -m "$(cat <<'EOF'
feat(api): implement JWT token validation

- Add validateToken function
- Add token refresh endpoint
- Add comprehensive test coverage
EOF
)"
```

### --commit=message-only

Save message to state file and accumulate to commits.sh:

```
VERIFICATION PASSED - MESSAGE SAVED
===================================

Commit message for Phase 2A:

feat(api): implement JWT token validation

- Add validateToken function
- Add token refresh endpoint
- Add comprehensive test coverage

Files to commit:
- src/api/auth.ts (CREATE)
- src/api/auth.test.ts (CREATE)
- src/api/routes.ts (MODIFY)

Saved to: .autobuild/phases/phase-2a.json
Appended to: .autobuild/commits.sh

User handles git operations.
```

### Commits.sh Accumulation

Append each phase's commit to the script:

```python
def append_to_commits_script(phase_id, commit_message, files):
    script_path = ".autobuild/commits.sh"

    commit_block = f'''
echo "Committing Phase {phase_id}..."
git add {" ".join(files)}
git commit -m "$(cat <<'EOF'
{commit_message}
EOF
)"
'''

    # Append to existing script
    with open(script_path, "a") as f:
        f.write(commit_block)
```

### --commit=single

Queue commit for final combined message:

```
VERIFICATION PASSED - COMMIT QUEUED
===================================

Phase 2A complete.
Commit message queued for final combined commit.

Queued commits: 3 of 6 phases

Continuing to next phase...
```

### Combined Commit at Completion

When all phases complete with --commit=single:

```
COMBINED COMMIT
===============

All 6 phases complete. Creating combined commit...

Files to commit:
- .eslintrc.json (CREATE)
- .prettierrc (CREATE)
- prisma/schema.prisma (CREATE)
- src/api/auth.ts (CREATE)
- src/api/auth.test.ts (CREATE)
- src/components/Login.tsx (CREATE)
- [... more files ...]

Combined commit message:

feat(auth): implement complete authentication system

Phases completed:
- Phase 0: Bootstrap - quality tools configuration
- Phase 1: Setup - database schema and migrations
- Phase 2A: Backend - authentication API endpoints
- Phase 2B: Frontend - login and registration UI
- Phase 2C: Tests - comprehensive test coverage
- Phase 3: Integration - end-to-end wiring

Files changed: 24
Tests added: 47

git add .
git commit -m "$(cat <<'EOF'
feat(auth): implement complete authentication system

Phases completed:
- Phase 0: Bootstrap - quality tools configuration
- Phase 1: Setup - database schema and migrations
- Phase 2A: Backend - authentication API endpoints
- Phase 2B: Frontend - login and registration UI
- Phase 2C: Tests - comprehensive test coverage
- Phase 3: Integration - end-to-end wiring

Files changed: 24
Tests added: 47
EOF
)"
```

---

## Verification Timeout Handling

If verification commands hang:

```
VERIFICATION TIMEOUT
====================

Check: Test
Command: npm test
Timeout: 5 minutes exceeded

Possible causes:
1. Tests are hanging (infinite loop, unresolved promise)
2. Tests are very slow
3. External dependency is down

Action: Treating as verification FAILURE.

Note: The sub-agent may have completed successfully, but we cannot
verify without passing quality gates. Initiating retry...
```

---

## Parallel Phase Verification

For parallel phases, verify each independently then check for conflicts:

```
PARALLEL VERIFICATION
=====================

Phase 2A:
- Quality gates: PASS
- Files: 2 created, 1 modified

Phase 2B:
- Quality gates: PASS
- Files: 3 created, 0 modified

Phase 2C:
- Quality gates: PASS
- Files: 2 created, 0 modified

Conflict Check:
- No overlapping file modifications
- All tests still pass together

PARALLEL BATCH: VERIFIED
```

### Detecting Parallel Conflicts

```python
def check_parallel_conflicts(results):
    all_files = {}

    for phase_id, result in results.items():
        modified = result["files"]["modified"]
        for file in modified:
            if file in all_files:
                return {
                    "conflict": True,
                    "file": file,
                    "phases": [all_files[file], phase_id]
                }
            all_files[file] = phase_id

    return {"conflict": False}
```

### Conflict Response

```
PARALLEL CONFLICT DETECTED
==========================

File: src/api/routes.ts
Modified by: Phase 2A AND Phase 2B

Both phases modified the same file. This may cause merge conflicts.

Resolution options:
1. Manual merge required
2. Re-run phases sequentially
3. Abort and fix plan

AUTOBUILD HALTED - Parallel conflict
```
