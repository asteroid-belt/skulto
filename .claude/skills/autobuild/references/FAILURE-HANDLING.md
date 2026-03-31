# Failure Handling Reference

> Retry logic, error recovery, parallel sibling handling, and halt procedures.

## Failure Classification

| Failure Type | Retryable | Description |
|--------------|-----------|-------------|
| `test_failure` | Yes | Tests failed, code may need fixes |
| `lint_error` | Yes | Linting errors, auto-fixable or manual |
| `typecheck_error` | Yes | Type errors in implementation |
| `format_error` | Yes | Usually auto-fixable |
| `task_incomplete` | Yes | Sub-agent didn't finish all tasks |
| `verification_mismatch` | Yes | Claimed success but verification failed |
| `timeout` | Yes | Sub-agent took too long |
| `parse_error` | Yes | Couldn't parse sub-agent result |
| `dependency_missing` | No | Required package/tool not installed |
| `circular_dependency` | No | Plan has circular phase dependencies |
| `file_conflict` | No | Parallel phases modified same file |
| `plan_invalid` | No | Plan format is incorrect |

---

## Retry Logic

### Single Retry Strategy

Each phase gets exactly ONE retry before failure:

```
PHASE FAILURE HANDLING
======================

Attempt 1: Execute phase normally
            |
            v
        Failed?
       /       \
      No        Yes
      |          |
   Success    Attempt 2: Execute with error context
                   |
                   v
               Failed?
              /       \
             No        Yes
             |          |
          Success    HALT
```

### Why Single Retry?

1. **Efficiency** - Multiple retries waste time on unfixable issues
2. **Signal** - Two failures strongly suggest manual intervention needed
3. **Determinism** - Predictable behavior, not indefinite loops
4. **Context** - Retry includes error context for better chance of success

---

## Retry Execution

### Initiating Retry

```
PHASE FAILURE - Initiating Retry
================================

Phase: 2A (Backend API)
Attempt: 1 of 2

Failure details:
  Type: test_failure
  Message: 3 tests failing in auth.test.ts
  Tests: validateToken, refreshToken, handleExpired

Launching retry sub-agent with error context...

[See SUBAGENT-PROMPTS.md for retry prompt template]
```

### Error Context for Retry

Pass previous failure information to retry sub-agent:

```python
def build_retry_context(phase_state):
    return {
        "previous_error_type": phase_state["error"]["type"],
        "previous_error_message": phase_state["error"]["message"],
        "previous_error_details": phase_state["error"].get("details", ""),
        "tasks_completed": phase_state["execution"]["tasks_completed"],
        "tasks_total": phase_state["execution"]["tasks_total"],
        "retry_guidance": get_retry_guidance(phase_state["error"]["type"])
    }
```

### Retry State Update

Before retry:
```json
{
  "phase_id": "2a",
  "status": "running",
  "attempt": 2,
  "max_attempts": 2,
  "error": {
    "attempt_1": {
      "type": "test_failure",
      "message": "3 tests failing"
    }
  }
}
```

After retry success:
```json
{
  "phase_id": "2a",
  "status": "complete",
  "attempt": 2,
  "error": {
    "attempt_1": {
      "type": "test_failure",
      "message": "3 tests failing"
    },
    "resolved_on_retry": true
  }
}
```

After retry failure:
```json
{
  "phase_id": "2a",
  "status": "failed",
  "attempt": 2,
  "error": {
    "attempt_1": {
      "type": "test_failure",
      "message": "3 tests failing"
    },
    "attempt_2": {
      "type": "test_failure",
      "message": "3 tests still failing"
    },
    "final": "Failed after maximum retries"
  }
}
```

---

## Parallel Sibling Handling

When a phase in a parallel batch fails:

### Let Siblings Complete

```
PARALLEL BATCH - PARTIAL FAILURE
================================

Batch: [2A, 2B, 2C]

Status:
  Phase 2A: FAILED (attempt 1)
  Phase 2B: running...
  Phase 2C: complete

Decision: Allow running siblings to complete before retry.

Rationale:
1. Siblings may succeed, preserving their work
2. Retry of 2A won't conflict with running 2B
3. More efficient than cancelling all

Waiting for 2B to complete...
```

### Sibling Completion Then Retry

```
PARALLEL BATCH - RETRY AFTER SIBLINGS
=====================================

All siblings complete:
  Phase 2A: failed (attempt 1)
  Phase 2B: complete
  Phase 2C: complete

Initiating retry for Phase 2A...

Note: 2B and 2C results preserved. Their commits (if --commit=auto)
already created. 2A retry runs in isolation.
```

### Multiple Siblings Fail

```
PARALLEL BATCH - MULTIPLE FAILURES
==================================

Batch: [2A, 2B, 2C]

Results:
  Phase 2A: failed
  Phase 2B: failed
  Phase 2C: complete

Multiple phases failed. Retrying each:

Retry 1: Phase 2A (parallel with 2B retry)
Retry 2: Phase 2B (parallel with 2A retry)

[Retries run in parallel since they're independent]

Retry results:
  Phase 2A: complete (retry succeeded)
  Phase 2B: failed (retry failed)

Phase 2B failed after retry.
AUTOBUILD HALTED
```

---

## Halt Procedure

When halt is triggered:

### Halt Sequence

```
AUTOBUILD HALT SEQUENCE
=======================

1. Stop launching new phases
2. Wait for running phases to complete (or timeout)
3. Write final state files
4. Update execution log
5. Output halt summary
6. Preserve state for resume
```

### Halt Summary Output

```
╔═══════════════════════════════════════════════════════════════════╗
║                       AUTOBUILD HALTED                             ║
╠═══════════════════════════════════════════════════════════════════╣
║                                                                   ║
║  Reason: Phase 2A failed after retry                              ║
║                                                                   ║
║  ─────────────────────────────────────────────────────────────   ║
║  COMPLETED PHASES                                                 ║
║  ─────────────────────────────────────────────────────────────   ║
║                                                                   ║
║  Phase 0: Bootstrap .......................... complete           ║
║    Commit: chore(bootstrap): add quality tools                    ║
║                                                                   ║
║  Phase 1: Setup .............................. complete           ║
║    Commit: feat(db): add user schema                              ║
║                                                                   ║
║  Phase 2B: Frontend .......................... complete           ║
║    Commit: feat(ui): add login form                               ║
║                                                                   ║
║  Phase 2C: Tests ............................. complete           ║
║    Commit: test(auth): add edge cases                             ║
║                                                                   ║
║  ─────────────────────────────────────────────────────────────   ║
║  FAILED PHASE                                                     ║
║  ─────────────────────────────────────────────────────────────   ║
║                                                                   ║
║  Phase 2A: Backend ........................... FAILED             ║
║    Attempts: 2/2                                                  ║
║    Error: test_failure                                            ║
║    Details: 3 tests failing - validateToken returns null          ║
║             instead of user object                                ║
║                                                                   ║
║  ─────────────────────────────────────────────────────────────   ║
║  BLOCKED PHASES                                                   ║
║  ─────────────────────────────────────────────────────────────   ║
║                                                                   ║
║  Phase 3: Integration ........................ blocked            ║
║    Blocked by: 2A                                                 ║
║                                                                   ║
║  ─────────────────────────────────────────────────────────────   ║
║  STATE PRESERVED                                                  ║
║  ─────────────────────────────────────────────────────────────   ║
║                                                                   ║
║  Directory: .autobuild/                                           ║
║  State files: 6 phases                                            ║
║  Logs: .autobuild/logs/                                           ║
║                                                                   ║
║  ─────────────────────────────────────────────────────────────   ║
║  RESUME INSTRUCTIONS                                              ║
║  ─────────────────────────────────────────────────────────────   ║
║                                                                   ║
║  1. Fix the failing tests in src/api/auth.test.ts                 ║
║  2. Run: /autobuild docs/feature-plan.md --commit=auto --resume   ║
║                                                                   ║
║  To start fresh: Add --fresh flag                                 ║
║                                                                   ║
╚═══════════════════════════════════════════════════════════════════╝
```

---

## Error Type Handling

### test_failure

```
ERROR: test_failure
==================

Symptoms:
- Test command exited with non-zero
- One or more test cases failed

Information to capture:
- Number of tests failed
- Test file paths
- Test names
- Error messages

Retry guidance:
"Fix the failing tests. The errors are in {FILES}.
Common issues: incorrect assertions, missing mocks, async timing."

User action if retry fails:
- Review test failures manually
- Check if tests are flaky
- Verify test assumptions match implementation
```

### lint_error

```
ERROR: lint_error
=================

Symptoms:
- Linter exited with non-zero
- Linting rules violated

Information to capture:
- Number of errors
- File paths with errors
- Rule names violated
- Line numbers

Retry guidance:
"Fix linting errors. Run `{LINT_COMMAND}` to see current errors.
Most are auto-fixable with `{LINT_FIX_COMMAND}`."

User action if retry fails:
- Review lint rules
- Consider disabling problematic rules
- Check for conflicting lint configurations
```

### typecheck_error

```
ERROR: typecheck_error
======================

Symptoms:
- Type checker exited with non-zero
- Type mismatches or missing types

Information to capture:
- Number of errors
- File paths with errors
- Error messages (type expected vs actual)
- Line numbers

Retry guidance:
"Fix type errors. Run `{TYPECHECK_COMMAND}` to see current errors.
Check that function signatures match usage."

User action if retry fails:
- Review type definitions
- Check for missing @types packages
- Verify tsconfig/mypy configuration
```

### timeout

```
ERROR: timeout
==============

Symptoms:
- Sub-agent didn't respond within time limit
- May have been stuck on a task

Information to capture:
- How long before timeout
- Last known activity (if any)

Retry guidance:
"Previous attempt timed out. Focus on completing tasks efficiently.
If tests are slow, consider running targeted tests only."

User action if retry fails:
- Increase timeout (if justified)
- Break phase into smaller phases
- Check for infinite loops or hanging operations
```

### verification_mismatch

```
ERROR: verification_mismatch
============================

Symptoms:
- Sub-agent claimed success
- Fresh verification in main agent failed

Information to capture:
- What sub-agent claimed
- What fresh verification found
- Discrepancy details

Retry guidance:
"Previous sub-agent reported success but verification failed.
Run ALL quality gates after EVERY change. Ensure tests pass
at the end, not just during development."

User action if retry fails:
- Review sub-agent logs
- Check for race conditions
- Verify no external state changes
```

---

## Non-Retryable Errors

### dependency_missing

```
ERROR: dependency_missing (NON-RETRYABLE)
=========================================

Required dependency not installed:
  Package: @types/jest
  Required by: Type checking

AUTOBUILD CANNOT PROCEED

Resolution:
  Install missing dependency: npm install -D @types/jest
  Then resume: /autobuild docs/feature-plan.md --commit=auto --resume

[HALTED - Manual intervention required]
```

### circular_dependency

```
ERROR: circular_dependency (NON-RETRYABLE)
==========================================

Plan contains circular phase dependencies:
  Phase 2A depends on Phase 3
  Phase 3 depends on Phase 2A

AUTOBUILD CANNOT PROCEED

Resolution:
  Fix the plan's "Depends On" column to remove the cycle.
  Ensure dependencies form a directed acyclic graph (DAG).

[HALTED - Invalid plan]
```

### file_conflict

```
ERROR: file_conflict (NON-RETRYABLE)
====================================

Parallel phases modified the same file:
  File: src/api/routes.ts
  Phase 2A modified lines 10-25
  Phase 2B modified lines 15-30

AUTOBUILD CANNOT PROCEED

Resolution options:
1. Manually merge the changes
2. Modify plan to make these phases sequential
3. Split the shared file to avoid conflicts

[HALTED - Parallel conflict]
```

---

## Recovery Scenarios

### Resume After Fix

User fixes the issue externally, then resumes:

```
RESUME AFTER EXTERNAL FIX
=========================

Previous state:
  Phase 2A: failed (test_failure)

User action: Fixed tests in src/api/auth.test.ts

Resume command: /autobuild docs/feature-plan.md --commit=auto --resume

Resume behavior:
1. Read state files
2. Find Phase 2A in failed status
3. Reset attempt count to 0
4. Re-execute Phase 2A
5. Continue with remaining phases if successful
```

### Fresh Start After Corruption

If state files are corrupted or need reset:

```
FRESH START
===========

User command: /autobuild docs/feature-plan.md --commit=auto --fresh

Behavior:
1. Delete .autobuild/ directory
2. Create fresh state files
3. Start from Phase 0
4. All previous commits remain (if --commit=auto was used)

Warning: Does not undo previous commits. Use git reset if needed.
```

### Partial Recovery

User wants to keep some completed work:

```
PARTIAL RECOVERY
================

Scenario: Phases 0-2B complete, 2C and later need re-run

Option 1: Edit state files manually
  - Set phase-2c.json status to "pending"
  - Delete phase-3.json (or set to "pending")
  - Run with --resume

Option 2: Use fresh and re-commit
  - Run with --fresh
  - All phases re-execute
  - Creates duplicate commits (may need squash)

Recommendation: Option 1 for surgical recovery
```
