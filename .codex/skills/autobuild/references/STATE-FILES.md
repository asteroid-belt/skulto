# State Files Reference

> Complete specification for .autobuild/ directory structure and state file format.

## Directory Structure

```
.autobuild/
  config.json           # Execution configuration (stack, commands, args)
  commits.sh            # Generated commit script (--commit=message-only)
  phases/
    phase-0.json        # State file per phase
    phase-1.json
    phase-2a.json       # Parallel phases use alpha suffix
    phase-2b.json
    phase-2c.json
    phase-3.json
    ...
  logs/
    execution.log       # Overall execution log
    phase-0.log         # Truncated sub-agent output per phase
    phase-1.log
    ...
```

---

## config.json

Created during initialization, stores execution parameters.

```json
{
  "version": "1.0.0",
  "plan_path": "docs/feature-plan.md",
  "commit_mode": "auto",
  "started_at": "2025-01-25T10:00:00Z",
  "last_updated": "2025-01-25T10:30:00Z",
  "stack": {
    "language": "typescript",
    "framework": "express",
    "package_manager": "pnpm",
    "test_framework": "jest",
    "linter": "eslint",
    "formatter": "prettier"
  },
  "commands": {
    "lint": "pnpm run lint",
    "format": "pnpm run format:check",
    "typecheck": "pnpm run typecheck",
    "test": "pnpm test"
  },
  "phases": {
    "total": 6,
    "completed": 3,
    "failed": 1,
    "pending": 2
  }
}
```

### Config Fields

| Field | Type | Description |
|-------|------|-------------|
| `version` | string | Autobuild state format version |
| `plan_path` | string | Path to plan document |
| `commit_mode` | enum | "auto", "message-only", or "single" |
| `started_at` | ISO 8601 | Execution start time |
| `last_updated` | ISO 8601 | Last state change time |
| `stack` | object | Detected technology stack |
| `commands` | object | Quality gate commands |
| `phases` | object | Phase count summary |

---

## Phase State Files

Each phase gets a state file: `.autobuild/phases/phase-{id}.json`

### Phase Naming

| Phase ID | State File |
|----------|------------|
| Phase 0 | `phase-0.json` |
| Phase 1 | `phase-1.json` |
| Phase 2A | `phase-2a.json` |
| Phase 2B | `phase-2b.json` |
| Phase 3 | `phase-3.json` |

**Normalize phase IDs**: Remove spaces, lowercase, remove punctuation.

### Complete Phase State Format

```json
{
  "phase_id": "2a",
  "phase_name": "Backend API",
  "status": "complete",
  "attempt": 1,
  "max_attempts": 2,
  "timestamps": {
    "started": "2025-01-25T10:15:00Z",
    "completed": "2025-01-25T10:25:00Z",
    "duration_seconds": 600
  },
  "dependencies": {
    "depends_on": ["1"],
    "parallel_with": ["2b", "2c"],
    "blocks": ["3"]
  },
  "execution": {
    "subagent_id": "task-abc123",
    "subagent_model": "sonnet",
    "tasks_total": 4,
    "tasks_completed": 4
  },
  "quality_gates": {
    "lint": {
      "passed": true,
      "command": "pnpm run lint",
      "output_summary": "No errors"
    },
    "format": {
      "passed": true,
      "command": "pnpm run format:check",
      "output_summary": "All files formatted"
    },
    "typecheck": {
      "passed": true,
      "command": "pnpm run typecheck",
      "output_summary": "No type errors"
    },
    "test": {
      "passed": true,
      "command": "pnpm test",
      "output_summary": "47 tests passed, 0 failed",
      "coverage": "87%"
    }
  },
  "verification": {
    "subagent_claimed": "complete",
    "fresh_verification": "passed",
    "verified_at": "2025-01-25T10:26:00Z"
  },
  "commit": {
    "message": "feat(api): implement JWT token validation\n\n- Add validateToken function\n- Add token refresh endpoint\n- Add comprehensive test coverage",
    "type": "feat",
    "scope": "api",
    "files_created": [
      "src/api/auth.ts",
      "src/api/auth.test.ts"
    ],
    "files_modified": [
      "src/api/routes.ts"
    ],
    "files_deleted": [],
    "committed": true,
    "commit_sha": "a1b2c3d"
  },
  "plan_updates": {
    "tasks_checked": 4,
    "dod_checked": 6,
    "status_updated": true
  },
  "error": null
}
```

### Status Values

| Status | Description |
|--------|-------------|
| `pending` | Not yet started |
| `running` | Sub-agent currently executing |
| `complete` | Successfully finished and verified |
| `failed` | Failed after max attempts |
| `blocked` | Dependencies not met |

### Minimal State (Pending Phase)

```json
{
  "phase_id": "3",
  "phase_name": "Integration",
  "status": "pending",
  "attempt": 0,
  "max_attempts": 2,
  "timestamps": {},
  "dependencies": {
    "depends_on": ["2a", "2b", "2c"],
    "parallel_with": [],
    "blocks": []
  },
  "execution": null,
  "quality_gates": null,
  "verification": null,
  "commit": null,
  "plan_updates": null,
  "error": null
}
```

### Failed Phase State

```json
{
  "phase_id": "2a",
  "phase_name": "Backend API",
  "status": "failed",
  "attempt": 2,
  "max_attempts": 2,
  "timestamps": {
    "started": "2025-01-25T10:15:00Z",
    "failed": "2025-01-25T10:35:00Z",
    "duration_seconds": 1200
  },
  "dependencies": {
    "depends_on": ["1"],
    "parallel_with": ["2b", "2c"],
    "blocks": ["3"]
  },
  "execution": {
    "subagent_id": "task-xyz789",
    "subagent_model": "sonnet",
    "tasks_total": 4,
    "tasks_completed": 2
  },
  "quality_gates": {
    "lint": {
      "passed": true,
      "command": "pnpm run lint",
      "output_summary": "No errors"
    },
    "format": {
      "passed": true,
      "command": "pnpm run format:check",
      "output_summary": "All files formatted"
    },
    "typecheck": {
      "passed": true,
      "command": "pnpm run typecheck",
      "output_summary": "No type errors"
    },
    "test": {
      "passed": false,
      "command": "pnpm test",
      "output_summary": "3 tests failed",
      "failures": [
        "src/api/auth.test.ts:45 - should validate token",
        "src/api/auth.test.ts:67 - should refresh token",
        "src/api/auth.test.ts:89 - should handle expired token"
      ]
    }
  },
  "verification": {
    "subagent_claimed": "complete",
    "fresh_verification": "failed",
    "verified_at": "2025-01-25T10:35:00Z"
  },
  "commit": null,
  "plan_updates": null,
  "error": {
    "type": "test_failure",
    "message": "3 tests failed after retry",
    "details": "Tests in src/api/auth.test.ts failing on token validation logic",
    "attempt_1_error": "3 tests failed - token validation incorrect",
    "attempt_2_error": "3 tests failed - same failures after retry"
  }
}
```

---

## Log Files

### execution.log

Overall execution log with timestamps:

```
[2025-01-25T10:00:00Z] AUTOBUILD STARTED
[2025-01-25T10:00:00Z] Plan: docs/feature-plan.md
[2025-01-25T10:00:00Z] Commit mode: auto
[2025-01-25T10:00:01Z] Stack detected: typescript/express
[2025-01-25T10:00:02Z] Phases identified: 6
[2025-01-25T10:00:02Z] Execution order: [0] -> [1] -> [2a,2b,2c] -> [3]

[2025-01-25T10:00:03Z] PHASE 0 STARTED
[2025-01-25T10:05:00Z] PHASE 0 COMPLETE - chore(bootstrap): add quality tools
[2025-01-25T10:05:01Z] Committed: a1b2c3d

[2025-01-25T10:05:02Z] PHASE 1 STARTED
[2025-01-25T10:10:00Z] PHASE 1 COMPLETE - feat(db): add user schema
[2025-01-25T10:10:01Z] Committed: b2c3d4e

[2025-01-25T10:10:02Z] PARALLEL BATCH STARTED: 2a, 2b, 2c
[2025-01-25T10:25:00Z] PHASE 2B COMPLETE - feat(ui): add login form
[2025-01-25T10:27:00Z] PHASE 2C COMPLETE - test(auth): add edge cases
[2025-01-25T10:30:00Z] PHASE 2A FAILED - tests failing (attempt 1)
[2025-01-25T10:30:01Z] PHASE 2A RETRY STARTED
[2025-01-25T10:35:00Z] PHASE 2A FAILED - tests failing (attempt 2)
[2025-01-25T10:35:01Z] PARALLEL BATCH FAILED: 2a failed, 2b+2c complete

[2025-01-25T10:35:02Z] AUTOBUILD HALTED
[2025-01-25T10:35:02Z] Completed: 4 phases
[2025-01-25T10:35:02Z] Failed: 1 phase (2a)
[2025-01-25T10:35:02Z] Remaining: 1 phase (3)
```

### phase-{id}.log

Truncated sub-agent conversation for debugging:

```
=== PHASE 2A SUB-AGENT LOG ===
Started: 2025-01-25T10:15:00Z
Model: sonnet

--- TASK 1: Create auth types ---
[Reading plan section...]
[Writing test: src/api/auth.test.ts]
[Running test - expected FAIL]
[Writing implementation: src/api/auth.ts]
[Running test - PASS]

--- TASK 2: Implement validateToken ---
[Writing test...]
[Running test - expected FAIL]
[Writing implementation...]
[Running test - PASS]

--- QUALITY GATES ---
Lint: pnpm run lint -> PASS
Format: pnpm run format:check -> PASS
Typecheck: pnpm run typecheck -> PASS
Tests: pnpm test -> PASS (47 tests)

--- PLAN UPDATES ---
Checked: 4 tasks
DoD: 6 items

--- RESULT ---
Status: complete
Commit: feat(api): implement JWT token validation

=== END PHASE 2A LOG ===
```

---

## commits.sh (--commit=message-only)

Generated script for manual commit execution:

```bash
#!/bin/bash
# Generated by autobuild
# Plan: docs/feature-plan.md
# Generated: 2025-01-25T10:35:00Z

set -e

echo "Committing Phase 0: Bootstrap"
git add .eslintrc.json .prettierrc tsconfig.json jest.config.js
git commit -m "$(cat <<'EOF'
chore(bootstrap): add quality tools

- Add ESLint configuration
- Add Prettier configuration
- Add TypeScript configuration
- Add Jest configuration
EOF
)"

echo "Committing Phase 1: Setup"
git add prisma/schema.prisma prisma/migrations/
git commit -m "$(cat <<'EOF'
feat(db): add user schema

- Create user model with auth fields
- Add initial migration
EOF
)"

echo "Committing Phase 2A: Backend API"
git add src/api/auth.ts src/api/auth.test.ts src/api/routes.ts
git commit -m "$(cat <<'EOF'
feat(api): implement JWT token validation

- Add validateToken function
- Add token refresh endpoint
- Add comprehensive test coverage
EOF
)"

# ... more phases ...

echo "All commits complete!"
```

---

## State File Operations

### Creating State File

```python
# Pseudocode for state file creation
def create_phase_state(phase_id, phase_name, dependencies):
    return {
        "phase_id": normalize_id(phase_id),
        "phase_name": phase_name,
        "status": "pending",
        "attempt": 0,
        "max_attempts": 2,
        "timestamps": {},
        "dependencies": dependencies,
        "execution": None,
        "quality_gates": None,
        "verification": None,
        "commit": None,
        "plan_updates": None,
        "error": None
    }
```

### Updating State File

```python
# Pseudocode for state update
def update_phase_state(state_file, updates):
    state = read_json(state_file)
    state.update(updates)
    state["timestamps"]["last_updated"] = now_iso()
    write_json(state_file, state)
```

### Reading State for Resume

```python
# Pseudocode for resume logic
def get_resume_point():
    states = read_all_phase_states()

    for phase in topological_order(states):
        if phase["status"] == "pending":
            return phase["phase_id"]
        if phase["status"] == "failed":
            return phase["phase_id"]  # Retry failed phase
        if phase["status"] == "running":
            return phase["phase_id"]  # Resume interrupted

    return None  # All complete
```

---

## Gitignore Recommendation

Add to `.gitignore`:

```gitignore
# Autobuild state (optional - some teams prefer to track)
.autobuild/

# Or keep state but ignore logs
# .autobuild/logs/
```

**Arguments for tracking .autobuild/:**
- Preserves execution history
- Enables team visibility into build progress
- Supports debugging failed builds

**Arguments against tracking:**
- State is ephemeral and regenerable
- Can contain large log files
- May conflict between team members

---

## State File Validation

Before using state files, validate:

```json
{
  "required_fields": [
    "phase_id",
    "phase_name",
    "status",
    "attempt",
    "max_attempts"
  ],
  "valid_statuses": [
    "pending",
    "running",
    "complete",
    "failed",
    "blocked"
  ]
}
```

If validation fails, treat state as corrupted and prompt user:

```
STATE FILE VALIDATION FAILED
============================

File: .autobuild/phases/phase-2a.json
Error: Missing required field 'status'

Options:
1. Delete corrupted state and re-run phase
2. Manually fix state file
3. Start fresh with --fresh flag

[WAITING FOR USER INPUT]
```
