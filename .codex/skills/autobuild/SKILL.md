---
name: autobuild
description: Autonomous single-shot plan execution. Runs entire superplan implementation plans without stopping, using sub-agents for each phase. Phases write state to filesystem for resume capability. Sequential phases run in order, parallel phases (1a, 1b) run concurrently. User never needs to compact context.
metadata:
  version: "1.0.0"
  author: skulto
compatibility: Requires plan document in superplan format. Sub-agents execute phases independently. State persisted to .autobuild/ directory.
---

# Autobuild: Autonomous Plan Execution Engine

Execute entire implementation plans in a single pass with zero user intervention. Each phase runs in its own sub-agent, writing state to the filesystem. Context never exhausts because sub-agents are isolated.

## Overview

Autobuild is an **autonomous execution engine** for superplan implementation plans. Unlike `superbuild` which stops after each phase for user confirmation, autobuild runs everything to completion.

**Key Differences from superbuild:**

| Aspect | superbuild | autobuild |
|--------|------------|-----------|
| Execution | Phase-by-phase with stops | Continuous until complete |
| User intervention | Required after each phase | None (fully autonomous) |
| Context management | Manual compaction | Sub-agents isolate context |
| State persistence | In conversation | Filesystem (.autobuild/) |
| Resume capability | Manual | Automatic from state files |

---

## Reference Index - MUST READ When Needed

**References contain detailed templates and patterns. Read BEFORE you need them.**

| When | Reference | What You Get |
|------|-----------|--------------|
| **Step 1: Initialize** | [STATE-FILES.md](references/STATE-FILES.md) | State file format, directory structure |
| **Step 3: Execute phases** | [PHASE-EXECUTION.md](references/PHASE-EXECUTION.md) | Phase ordering, parallel detection, retry logic |
| **Step 3: Launch sub-agents** | [SUBAGENT-PROMPTS.md](references/SUBAGENT-PROMPTS.md) | Exact prompts for phase sub-agents |
| **Step 4: Verify and commit** | [VERIFICATION.md](references/VERIFICATION.md) | Fresh verification, commit handling |
| **Step 5: Handle failures** | [FAILURE-HANDLING.md](references/FAILURE-HANDLING.md) | Retry logic, error recovery |
| **Overall flow** | [ORCHESTRATION.md](references/ORCHESTRATION.md) | Main orchestration loop, completion handling |

**Also reference superplan/superbuild documents:**
- `superplan/references/TASK-MICROSTRUCTURE.md` - TDD 5-step format per task
- `superplan/references/TDD-DISCIPLINE.md` - TDD enforcement rules
- `superbuild/references/ENFORCEMENT-GUIDE.md` - Quality gate commands by stack, evidence requirements
- `superbuild/references/PLAN-UPDATES.md` - Plan checkbox patterns, status updates

**DO NOT SKIP REFERENCES.** They contain exact prompts, templates, and formats that are NOT duplicated here.

---

## CLI Arguments

```
/autobuild <plan-path> [options]

Arguments:
  plan-path           Path to superplan document (required)

Options:
  --commit=<mode>     Git commit behavior (required before execution)
                      - auto: Auto-commit after each phase passes
                      - message-only: Generate messages, user handles git
                      - single: One combined commit at the end

  --resume            Resume from existing state (default if .autobuild/ exists)
  --fresh             Ignore existing state, start from scratch
  --dry-run           Validate plan and show execution order without running
```

**Example invocations:**
```bash
/autobuild docs/feature-plan.md --commit=auto
/autobuild docs/feature-plan.md --commit=message-only --resume
/autobuild docs/feature-plan.md --commit=single --fresh
```

---

## Critical Workflow

```
+-----------------------------------------------------------------------+
|                      AUTOBUILD EXECUTION FLOW                          |
+-----------------------------------------------------------------------+
|                                                                       |
|  1. INITIALIZE       |  Parse args, create .autobuild/, detect stack  |
|         |            |  NO PLAN = EXIT (ask user, then exit if none)  |
|         v            |  NO --commit = STOP (require commit mode)      |
|  2. LOAD STATE       |  Read existing state files if --resume         |
|         |            |  Identify completed phases, pending phases     |
|         v                                                             |
|  2.5 PLAN REVIEW     |  Validate plan before autonomous execution     |
|         |            |  Check: ambiguity, deps, tests, risks, conflicts|
|         v            |  IF concerns → EXIT (autonomous = no recovery) |
|  3. EXECUTE PHASES   |  For each pending phase (or parallel group):   |
|     |                |                                                |
|     |  3a. Launch sub-agent(s) for phase(s)                           |
|     |      - Sequential phases: one sub-agent at a time               |
|     |      - Parallel phases (2a,2b,2c): concurrent sub-agents        |
|     |                                                                 |
|     |  3b. Sub-agent executes:                                        |
|     |      - Read plan section for its phase                          |
|     |      - Follow TDD micro-structure per task                      |
|     |      - Run quality gates (lint, test, typecheck)                |
|     |      - Update plan checkboxes                                   |
|     |      - Return: status, commit message, files changed            |
|     |                                                                 |
|     |  3c. Write state file for phase                                 |
|         |                                                             |
|         v                                                             |
|  4. VERIFY + COMMIT  |  Re-run quality gates fresh (trust but verify) |
|         |            |  Handle git based on --commit mode              |
|         v                                                             |
|  5. ON FAILURE       |  Retry once, then halt with state preserved    |
|         |            |  Parallel siblings may continue before halt    |
|         v                                                             |
|  6. COMPLETION       |  All phases done, output summary               |
|                      |  State files preserved for audit               |
|                                                                       |
|  =====================================================================|
|  Sub-agents are ISOLATED. Context never exhausts. State is on disk.   |
|  =====================================================================|
|                                                                       |
+-----------------------------------------------------------------------+
```

---

## Step 1: Initialize

**REQUIRED: Plan document and commit mode must be provided.**

### Argument Validation

```
AUTOBUILD INITIALIZATION
=========================

Plan: [path]
Commit Mode: [auto|message-only|single]

Validating...
```

**If no plan provided:**
```
ERROR: No plan document provided.

Usage: /autobuild <plan-path> --commit=<mode>

Example: /autobuild docs/feature-plan.md --commit=auto

[EXIT - Cannot proceed without plan]
```

**If no --commit mode specified:**
```
COMMIT MODE REQUIRED
====================

Before I can execute this plan, I need to know how to handle git commits.

Please specify one of:
  --commit=auto         Auto-commit after each successful phase
  --commit=message-only Generate messages, you handle git (recommended)
  --commit=single       One combined commit after all phases complete

Example: /autobuild docs/feature-plan.md --commit=message-only

[WAITING FOR COMMIT MODE]
```

**NO EXCEPTIONS.** Do not proceed without explicit commit mode.

### Directory Setup

Create `.autobuild/` directory structure:

```
.autobuild/
  config.json           # Execution configuration
  phases/
    phase-0.json        # State file per phase
    phase-1.json
    phase-2a.json
    phase-2b.json
    ...
  logs/
    execution.log       # Overall execution log
    phase-0.log         # Per-phase logs (truncated sub-agent output)
    ...
```

> **STOP. Read [STATE-FILES.md](references/STATE-FILES.md) NOW** for complete state file format.

### Stack Detection

Detect technology stack to determine quality commands:

```
STACK DETECTION
===============

Detected:
- Language: TypeScript
- Framework: Express
- Package Manager: pnpm
- Test Framework: Jest
- Linter: ESLint
- Formatter: Prettier

Quality Commands:
- Lint: pnpm run lint
- Format: pnpm run format:check
- Typecheck: pnpm run typecheck
- Test: pnpm test

Stack saved to .autobuild/config.json
```

---

## Step 2: Load State (Resume Support)

If `.autobuild/` exists and `--resume` (or default):

```
LOADING EXISTING STATE
======================

Found existing execution state:
- Config: .autobuild/config.json
- Phase files: 6 found

Phase Status:
| Phase | Name | State File | Status |
|-------|------|------------|--------|
| 0 | Bootstrap | phase-0.json | complete |
| 1 | Setup | phase-1.json | complete |
| 2A | Backend | phase-2a.json | failed |
| 2B | Frontend | phase-2b.json | complete |
| 2C | Tests | phase-2c.json | pending |
| 3 | Integration | phase-3.json | pending |

Resuming from Phase 2A (first incomplete)...
```

If `--fresh` specified:
```
FRESH START
===========

--fresh flag detected. Clearing existing state.

Removed: .autobuild/ (6 state files)
Created: .autobuild/ (fresh)

Starting from Phase 0...
```

---

## Step 2.5: Critical Plan Review

**Before executing, validate the plan for autonomous execution.**

Unlike interactive superbuild where users can catch issues between phases, autobuild runs autonomously. Plan validation is therefore MORE critical.

### Review Checklist

1. **Ambiguous tasks** - Any task unclear about what to do?
2. **Missing dependencies** - Are required files/APIs/packages identified?
3. **Test gaps** - Does each task have testable acceptance criteria?
4. **Risky changes** - Any destructive operations or breaking changes?
5. **Parallel conflicts** - Could parallel phases modify the same files?

### If Concerns Exist

```
⚠️  PLAN REVIEW - Concerns Identified
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

1. [Concern 1 - what and why]
2. [Concern 2 - what and why]

Autonomous execution requires clear plans.
Please clarify before I proceed.

[EXECUTION BLOCKED - Plan validation failed]
```

**For autonomous builds, DO NOT proceed with ambiguous plans.** Exit and request clarification.

### If No Concerns

```
PLAN VALIDATION: PASSED
=======================

- Tasks: Clear and actionable
- Dependencies: Identified
- Tests: Acceptance criteria present
- Risks: None detected
- Parallel safety: No file conflicts expected

Proceeding to execution...
```

---

## Step 3: Execute Phases

### Phase Ordering

> **STOP. Read [PHASE-EXECUTION.md](references/PHASE-EXECUTION.md) NOW** for phase ordering rules.

Parse plan to build execution order:

1. **Identify all phases** from plan overview table
2. **Build dependency graph** from "Depends On" column
3. **Identify parallel groups** from "Parallel With" column
4. **Topologically sort** for execution order

```
EXECUTION ORDER
===============

Batch 1 (sequential): Phase 0
Batch 2 (sequential): Phase 1
Batch 3 (parallel): Phase 2A, 2B, 2C
Batch 4 (sequential): Phase 3

Total: 6 phases in 4 batches
```

### Launching Sub-Agents

> **STOP. Read [SUBAGENT-PROMPTS.md](references/SUBAGENT-PROMPTS.md) NOW** for exact sub-agent prompts.

#### Sequential Phase

```
EXECUTING PHASE 1: Setup
========================

Launching sub-agent...

[Sub-agent executes autonomously]
[Reads plan section]
[Follows TDD micro-structure]
[Runs quality gates]
[Updates plan checkboxes]
[Returns result]

Sub-agent complete. Processing result...
```

#### Parallel Phases

```
EXECUTING PARALLEL BATCH: 2A, 2B, 2C
=====================================

Launching 3 sub-agents in parallel...

[Sub-agent 2A: Backend API]
[Sub-agent 2B: Frontend UI]
[Sub-agent 2C: Edge Case Tests]

Waiting for all sub-agents to complete...

Results received:
- Phase 2A: complete
- Phase 2B: complete
- Phase 2C: complete

Processing parallel results...
```

**CRITICAL:** Use Task tool with `run_in_background: true` for parallel phases, then collect results with TaskOutput.

### Sub-Agent Execution

Each sub-agent:

1. **Reads the plan** - Specific section for its phase
2. **Follows TDD** - 5-step micro-structure per task
3. **Runs quality gates** - Lint, test, typecheck
4. **Updates plan file** - Checks off completed tasks
5. **Returns structured result** - JSON with status, commit, files

Sub-agent does NOT:
- Run git commands (unless --commit=auto in main agent)
- Modify other phases
- Access conversation history from main agent

---

## Step 4: Verify and Commit

### Fresh Verification (Trust But Verify)

> **STOP. Read [VERIFICATION.md](references/VERIFICATION.md) NOW** for verification requirements.

**After sub-agent reports success, VERIFY FRESH in main agent:**

```
VERIFICATION - Phase 2A
=======================

Sub-agent reported: complete

Running fresh verification...

Lint:      pnpm run lint           ... PASS
Format:    pnpm run format:check   ... PASS
Typecheck: pnpm run typecheck      ... PASS
Tests:     pnpm test               ... PASS (47 tests)

All quality gates passed.
```

**If verification fails (sub-agent claimed success but check fails):**

```
VERIFICATION MISMATCH
=====================

Sub-agent claimed success, but fresh verification failed:

Tests: FAIL
  5 tests failed in src/services/auth.test.ts

This indicates a sub-agent error. Treating as phase failure.
Initiating retry...
```

### Commit Handling

Based on `--commit` mode:

#### --commit=auto
```
AUTO-COMMIT - Phase 2A
======================

git add src/api/auth.ts src/api/auth.test.ts
git commit -m "$(cat <<'EOF'
feat(auth): implement JWT token validation

- Add validateToken function
- Add token refresh endpoint
- Add comprehensive test coverage
EOF
)"

Commit created: a1b2c3d
```

#### --commit=message-only
```
COMMIT MESSAGE - Phase 2A
=========================

feat(auth): implement JWT token validation

- Add validateToken function
- Add token refresh endpoint
- Add comprehensive test coverage

Files to commit:
- src/api/auth.ts (CREATE)
- src/api/auth.test.ts (CREATE)

[Message saved to .autobuild/phases/phase-2a.json]
[User handles git operations]
```

#### --commit=single
```
PHASE 2A COMPLETE
=================

Commit message queued for final combined commit.
Continuing to next phase...
```

### State File Update

After each phase (success or failure), update state file:

```
STATE UPDATED - Phase 2A
========================

File: .autobuild/phases/phase-2a.json
Status: complete
Commit: feat(auth): implement JWT token validation
Files: 2 created, 0 modified
Timestamp: 2025-01-25T10:30:00Z
```

---

## Step 5: Handle Failures

> **STOP. Read [FAILURE-HANDLING.md](references/FAILURE-HANDLING.md) NOW** for failure handling.

### Retry Logic

On phase failure, retry ONCE:

```
PHASE FAILURE - Phase 2A (Attempt 1/2)
======================================

Sub-agent reported failure:
- Tests failed: 3 failing in auth.test.ts
- Linter: passed
- Typecheck: passed

Initiating retry...

RETRY - Phase 2A (Attempt 2/2)
==============================

Launching fresh sub-agent with error context...

[Sub-agent executes with knowledge of previous failure]
```

### Parallel Sibling Handling

If a parallel phase fails, let siblings complete:

```
PARALLEL BATCH FAILURE
======================

Phase 2A: FAILED (after retry)
Phase 2B: complete
Phase 2C: running...

Waiting for 2C to complete before halting...

Phase 2C: complete

EXECUTION HALTED
================

Phase 2A failed after retry.
Siblings 2B and 2C completed successfully.

State preserved in .autobuild/
Resume with: /autobuild docs/feature-plan.md --commit=auto --resume
```

### Halt State

```
AUTOBUILD HALTED
================

Execution stopped due to: Phase 2A failure (tests failing)

Completed phases:
- Phase 0: Bootstrap (complete)
- Phase 1: Setup (complete)
- Phase 2B: Frontend (complete)
- Phase 2C: Tests (complete)

Failed phase:
- Phase 2A: Backend (failed after retry)

Remaining phases:
- Phase 3: Integration (blocked by 2A)

State preserved: .autobuild/

To resume after fixing:
  /autobuild docs/feature-plan.md --commit=auto --resume

To start fresh:
  /autobuild docs/feature-plan.md --commit=auto --fresh
```

---

## Step 6: Completion

When all phases complete:

```
+=====================================================================+
|                     AUTOBUILD COMPLETE                               |
+=====================================================================+

Plan: docs/feature-plan.md
Duration: [calculated from timestamps]
Phases: 6/6 complete

Phase Summary:
| Phase | Name | Status | Commit |
|-------|------|--------|--------|
| 0 | Bootstrap | complete | chore(bootstrap): add quality tools |
| 1 | Setup | complete | feat(db): add user schema |
| 2A | Backend | complete | feat(api): implement auth endpoints |
| 2B | Frontend | complete | feat(ui): add login form |
| 2C | Tests | complete | test(auth): add edge case coverage |
| 3 | Integration | complete | feat(auth): wire frontend to backend |

Files Changed: [total count]
Tests Added: [count from test output]
Coverage: [if available]

State preserved: .autobuild/

[Based on --commit mode, output final instructions]
```

### --commit=auto completion
```
All commits created successfully.
Review with: git log --oneline -6
```

### --commit=message-only completion
```
PENDING COMMITS
===============

The following commits are ready to create:

1. chore(bootstrap): add quality tools
   git add ... && git commit -m "..."

2. feat(db): add user schema
   git add ... && git commit -m "..."

[etc.]

Full commit commands saved to: .autobuild/commits.sh
Run with: bash .autobuild/commits.sh
```

### --commit=single completion
```
COMBINED COMMIT MESSAGE
=======================

feat(auth): implement complete authentication system

This commit includes:
- Bootstrap: quality tools configuration
- Database: user schema and migrations
- API: authentication endpoints with JWT
- UI: login and registration forms
- Tests: comprehensive test coverage
- Integration: full e2e authentication flow

Phases completed: 6
Files changed: [count]
```

---

## Definition of Done Verification (CRITICAL)

**Quality gates alone are insufficient.** Tests passing does NOT mean the phase is complete.

### The Integration Trap

A common failure mode is:
1. Code is written (function exists)
2. Tests pass (function works when called)
3. BUT the code is never wired up (function never gets called)
4. Phase marked "complete" when it's actually broken

**Example:** `LoadDiscoveries()` created, tests pass, but it's never called from `Init()`. The Discovered section shows nothing. Tests pass because no test verified the integration.

### Mandatory DoD Verification

After quality gates pass, the sub-agent MUST verify each Definition of Done item by **actually testing the user-facing behavior**:

```
DEFINITION OF DONE VERIFICATION
===============================

DoD: "DISCOVERED SKILLS section displays discoveries"

Verification approach:
1. Check if LoadDiscoveries() is called (not just defined)
2. Check if result handler exists in app.go
3. Trace the data flow: scan → DB → load → display

Result: FAIL - LoadDiscoveries() exists but is never called

Action: Wire up the integration before claiming phase complete
```

### No TODOs, No Incomplete Wiring

**Leaving TODOs = phase FAILED.** Do not mark a phase complete if:
- Functions exist but aren't called
- Handlers exist but aren't wired up
- Code has `// TODO` comments for core functionality
- A feature "works in isolation" but isn't integrated

### Verify User-Facing Behavior

For each DoD item, ask: "If a user tried this right now, would it work?"

- "Button displays" → Would the button actually appear?
- "Data loads" → Would calling the entry point actually load data?
- "Feature works" → End-to-end, not just unit test passing

**If you can't demonstrate the behavior works end-to-end, the phase is not complete.**

---

## Rationalizations to Reject

| Excuse | Reality |
|--------|---------|
| "Let me run multiple phases in one sub-agent" | **NO.** Each phase gets its own sub-agent for context isolation |
| "I'll skip the state file for this phase" | **NO.** State files enable resume and audit |
| "The sub-agent passed, no need to verify" | **NO.** Fresh verification is mandatory |
| "Let me just continue after failure" | **NO.** Retry once, then halt |
| "Sequential phases can run in parallel" | **NO.** Respect dependency order |
| "I'll batch the verification at the end" | **NO.** Verify after each phase |
| "The plan doesn't specify commit mode" | **NO.** Require --commit before execution |
| "Let me auto-commit without being told" | **NO.** Explicit commit mode required |
| "Tests pass so the feature works" | **NO.** Verify user-facing behavior, not just unit tests |
| "The function exists, I'll wire it up later" | **NO.** Unwired code = incomplete phase |
| "I added a TODO for the next phase" | **NO.** TODOs for core functionality = phase FAILED |

---

## Red Flags - STOP Immediately

If you catch yourself thinking any of these, STOP:

- "Context is getting long, let me combine phases"
- "This phase is simple, I can skip the sub-agent"
- "The retry failed, but let me try once more"
- "Sequential phases don't really depend on each other"
- "I'll write the state files at the end"
- "Verification is redundant, sub-agents are reliable"
- "The user probably wants auto-commit"
- "Tests pass so the DoD is met"
- "The code exists, even if it's not called yet"
- "I'll add a TODO and the next phase will handle it"
- "This function works, I just need to wire it up"
- "The implementation is there, integration is trivial"

**All of these = violation of autobuild protocol.**

---

## The Iron Rules

1. **Explicit commit mode** - Never start without --commit flag
2. **One sub-agent per phase** - Context isolation is non-negotiable
3. **State file per phase** - Written immediately after completion/failure
4. **Fresh verification** - Never trust sub-agent claims alone
5. **Single retry on failure** - Then halt with state preserved
6. **Respect dependencies** - Sequential means sequential
7. **Parallel means parallel** - Launch concurrent sub-agents
8. **Resume from state** - Check .autobuild/ before starting
9. **Preserve state on halt** - Enable future resume
10. **No user intervention** - Fully autonomous execution
11. **DoD means user-facing behavior works** - Not just "code exists" or "tests pass"
12. **No TODOs for core functionality** - Complete the phase or mark it failed
13. **Integration is part of implementation** - Unwired code is incomplete code

**Autobuild is rigid by design.** The sub-agent isolation prevents context exhaustion. The state files enable resume. The verification ensures quality. Do not rationalize around it.
