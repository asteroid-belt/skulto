# Sub-Agent Prompts Reference

> Exact prompts for launching phase execution sub-agents. Copy these templates exactly.

## Core Principle

Each phase runs in its own sub-agent to:
1. **Isolate context** - Sub-agent starts fresh, no accumulated context
2. **Enable parallelism** - Multiple sub-agents can run concurrently
3. **Prevent exhaustion** - Main agent context stays minimal
4. **Support recovery** - Failed sub-agent doesn't corrupt main state

---

## Sequential Phase Prompt

Use this exact template for sequential phases:

```
You are executing Phase {PHASE_ID} of an implementation plan.

═══════════════════════════════════════════════════════════════
PHASE DETAILS
═══════════════════════════════════════════════════════════════

Phase ID: {PHASE_ID}
Phase Name: {PHASE_NAME}
Plan File: {PLAN_PATH}

═══════════════════════════════════════════════════════════════
YOUR TASK
═══════════════════════════════════════════════════════════════

1. Read the plan file at {PLAN_PATH}
2. Find the section for Phase {PHASE_ID}: {PHASE_NAME}
3. Execute EACH task following the 5-step TDD micro-structure:
   - Step 1: Write the failing test
   - Step 2: Run test, verify it FAILS (mandatory - must see failure)
   - Step 3: Write minimal implementation
   - Step 4: Run test, verify it PASSES (mandatory - must see pass)
   - Step 5: Stage files (do not commit)

   **TDD ENFORCEMENT - STOP IMMEDIATELY IF:**
   | Rule | Violation |
   |------|-----------|
   | Test before code | Implementation exists without test |
   | Verify RED | Test passes immediately (testing existing behavior) |
   | Minimal code | Adding features beyond current test |
   | Verify GREEN | Test not run after implementation |
   | No regressions | Other tests now fail |

   Read `superplan/references/TDD-DISCIPLINE.md` for full TDD rules.

4. After ALL tasks complete, run quality gates:
   - Lint: {LINT_COMMAND}
   - Format: {FORMAT_COMMAND}
   - Typecheck: {TYPECHECK_COMMAND}
   - Test: {TEST_COMMAND}

   Read `superbuild/references/ENFORCEMENT-GUIDE.md` for:
   - Stack-specific commands (JS/TS, Python, Go, Rust)
   - Exit code interpretation
   - Evidence requirements by claim
   - Flaky test detection

5. Update the plan file:
   - Check off completed tasks (- [ ] -> - [x])
   - Check off Definition of Done items
   - Update phase status in overview table (if present)

   Read `superbuild/references/PLAN-UPDATES.md` for:
   - Checkbox pattern recognition
   - Status indicator patterns
   - Verification steps

6. **VERIFY DEFINITION OF DONE (CRITICAL)**:

   For EACH DoD item, verify the USER-FACING BEHAVIOR actually works:

   - "Section displays X" → Trace: Is the load function called? Is the handler wired up?
   - "Feature Y works" → Can you demonstrate end-to-end, not just unit test?
   - "Component Z appears" → Is it actually rendered, or just defined?

   **THE INTEGRATION TRAP:**
   Creating a function is NOT completing a task. The function must be:
   - Called from the appropriate entry point
   - Have its result handled
   - Actually produce the user-facing behavior

   If you created `LoadDiscoveries()` but it's never called from `Init()`,
   the DoD "displays discoveries" is NOT met. The phase is NOT complete.

   **NO TODOs:** If you write `// TODO: Handle this in Phase X`, the phase
   has FAILED. Complete the integration or report failure.

═══════════════════════════════════════════════════════════════
CRITICAL RULES
═══════════════════════════════════════════════════════════════

- Do NOT skip the TDD cycle - tests MUST fail first
- Do NOT run git commit commands
- Do NOT modify other phases
- Do NOT proceed if quality gates fail
- STOP immediately if you cannot complete a task
- Do NOT mark phase complete if code exists but isn't wired up
- Do NOT leave TODOs for core functionality
- VERIFY user-facing behavior works, not just that code exists

═══════════════════════════════════════════════════════════════
RETURN FORMAT (JSON)
═══════════════════════════════════════════════════════════════

When complete, output EXACTLY this JSON structure:

```json
{
  "phase_id": "{PHASE_ID}",
  "status": "complete" | "failed",
  "tasks": {
    "total": <number>,
    "completed": <number>
  },
  "quality_gates": {
    "lint": {"passed": true|false, "output": "<summary>"},
    "format": {"passed": true|false, "output": "<summary>"},
    "typecheck": {"passed": true|false, "output": "<summary>"},
    "test": {"passed": true|false, "output": "<summary>", "count": <number>}
  },
  "commit_message": "<conventional commit message>",
  "files": {
    "created": ["path/to/file.ts", ...],
    "modified": ["path/to/file.ts", ...],
    "deleted": []
  },
  "plan_updated": true|false,
  "error": null | {"type": "<type>", "message": "<details>"}
}
```

Begin execution now.
```

---

## Parallel Phase Prompt

Use this for phases in a parallel batch. Identical to sequential but emphasizes isolation:

```
You are executing Phase {PHASE_ID} of an implementation plan.

THIS IS A PARALLEL PHASE - Other phases ({SIBLING_PHASES}) are running concurrently.
Do NOT modify files that belong to sibling phases.

═══════════════════════════════════════════════════════════════
PHASE DETAILS
═══════════════════════════════════════════════════════════════

Phase ID: {PHASE_ID}
Phase Name: {PHASE_NAME}
Plan File: {PLAN_PATH}
Parallel With: {SIBLING_PHASES}

═══════════════════════════════════════════════════════════════
YOUR TASK
═══════════════════════════════════════════════════════════════

1. Read the plan file at {PLAN_PATH}
2. Find the section for Phase {PHASE_ID}: {PHASE_NAME}
3. Execute EACH task following the 5-step TDD micro-structure:
   - Step 1: Write the failing test
   - Step 2: Run test, verify it FAILS (mandatory - must see failure)
   - Step 3: Write minimal implementation
   - Step 4: Run test, verify it PASSES (mandatory - must see pass)
   - Step 5: Stage files (do not commit)

   **TDD ENFORCEMENT - STOP IMMEDIATELY IF:**
   | Rule | Violation |
   |------|-----------|
   | Test before code | Implementation exists without test |
   | Verify RED | Test passes immediately (testing existing behavior) |
   | Minimal code | Adding features beyond current test |
   | Verify GREEN | Test not run after implementation |
   | No regressions | Other tests now fail |

   Read `superplan/references/TDD-DISCIPLINE.md` for full TDD rules.

4. After ALL tasks complete, run quality gates:
   - Lint: {LINT_COMMAND}
   - Format: {FORMAT_COMMAND}
   - Typecheck: {TYPECHECK_COMMAND}
   - Test: {TEST_COMMAND}

   Read `superbuild/references/ENFORCEMENT-GUIDE.md` for:
   - Stack-specific commands (JS/TS, Python, Go, Rust)
   - Exit code interpretation
   - Evidence requirements by claim
   - Flaky test detection

5. Update the plan file:
   - Check off completed tasks for YOUR PHASE ONLY
   - Check off Definition of Done items for YOUR PHASE ONLY
   - Update YOUR phase status in overview table

   Read `superbuild/references/PLAN-UPDATES.md` for:
   - Checkbox pattern recognition
   - Status indicator patterns
   - Verification steps

6. **VERIFY DEFINITION OF DONE (CRITICAL)**:

   For EACH DoD item, verify the USER-FACING BEHAVIOR actually works:

   - "Section displays X" → Trace: Is the load function called? Is the handler wired up?
   - "Feature Y works" → Can you demonstrate end-to-end, not just unit test?
   - "Component Z appears" → Is it actually rendered, or just defined?

   **THE INTEGRATION TRAP:**
   Creating a function is NOT completing a task. The function must be:
   - Called from the appropriate entry point
   - Have its result handled
   - Actually produce the user-facing behavior

   **NO TODOs:** If you write `// TODO: Handle this in Phase X`, the phase
   has FAILED. Complete the integration or report failure.

═══════════════════════════════════════════════════════════════
PARALLEL EXECUTION RULES
═══════════════════════════════════════════════════════════════

- ONLY modify files listed in YOUR phase section
- Do NOT touch files that sibling phases might be editing
- If you need a shared file, coordinate via imports only
- Siblings phases are: {SIBLING_PHASES}

═══════════════════════════════════════════════════════════════
CRITICAL RULES
═══════════════════════════════════════════════════════════════

- Do NOT skip the TDD cycle - tests MUST fail first
- Do NOT run git commit commands
- Do NOT modify other phases
- Do NOT proceed if quality gates fail
- STOP immediately if you cannot complete a task
- Do NOT mark phase complete if code exists but isn't wired up
- Do NOT leave TODOs for core functionality
- VERIFY user-facing behavior works, not just that code exists

═══════════════════════════════════════════════════════════════
RETURN FORMAT (JSON)
═══════════════════════════════════════════════════════════════

When complete, output EXACTLY this JSON structure:

```json
{
  "phase_id": "{PHASE_ID}",
  "status": "complete" | "failed",
  "tasks": {
    "total": <number>,
    "completed": <number>
  },
  "quality_gates": {
    "lint": {"passed": true|false, "output": "<summary>"},
    "format": {"passed": true|false, "output": "<summary>"},
    "typecheck": {"passed": true|false, "output": "<summary>"},
    "test": {"passed": true|false, "output": "<summary>", "count": <number>}
  },
  "commit_message": "<conventional commit message>",
  "files": {
    "created": ["path/to/file.ts", ...],
    "modified": ["path/to/file.ts", ...],
    "deleted": []
  },
  "plan_updated": true|false,
  "error": null | {"type": "<type>", "message": "<details>"}
}
```

Begin execution now.
```

---

## Retry Phase Prompt

When retrying a failed phase, include context about the previous failure:

```
You are RETRYING Phase {PHASE_ID} of an implementation plan.

═══════════════════════════════════════════════════════════════
RETRY CONTEXT
═══════════════════════════════════════════════════════════════

This is attempt {ATTEMPT} of {MAX_ATTEMPTS}.

Previous failure:
  Type: {PREVIOUS_ERROR_TYPE}
  Message: {PREVIOUS_ERROR_MESSAGE}
  Details: {PREVIOUS_ERROR_DETAILS}

You must address the previous failure while completing the phase.

═══════════════════════════════════════════════════════════════
PHASE DETAILS
═══════════════════════════════════════════════════════════════

Phase ID: {PHASE_ID}
Phase Name: {PHASE_NAME}
Plan File: {PLAN_PATH}

═══════════════════════════════════════════════════════════════
YOUR TASK
═══════════════════════════════════════════════════════════════

1. Read the plan file at {PLAN_PATH}
2. Find the section for Phase {PHASE_ID}: {PHASE_NAME}
3. Review what was already done (check plan for [x] items)
4. Focus on fixing the previous failure:
   {RETRY_GUIDANCE}

5. Complete any remaining tasks using TDD micro-structure
6. Run ALL quality gates (even if some passed before)
7. Update plan file with any newly completed items

═══════════════════════════════════════════════════════════════
CRITICAL RULES
═══════════════════════════════════════════════════════════════

- Do NOT skip the TDD cycle - tests MUST fail first
- Do NOT run git commit commands
- Do NOT modify other phases
- Do NOT proceed if quality gates fail
- STOP immediately if you cannot complete a task

═══════════════════════════════════════════════════════════════
RETURN FORMAT (JSON)
═══════════════════════════════════════════════════════════════

[Same JSON format as above]

Begin retry now.
```

### Retry Guidance by Error Type

| Error Type | Retry Guidance |
|------------|----------------|
| `test_failure` | "Fix the failing tests. Read the error messages carefully. The tests at {FILES} are failing." |
| `lint_error` | "Fix linting errors. Run `{LINT_COMMAND}` to see current errors." |
| `typecheck_error` | "Fix type errors. Run `{TYPECHECK_COMMAND}` to see current errors." |
| `format_error` | "Run `{FORMAT_FIX_COMMAND}` to auto-fix formatting, then verify." |
| `task_incomplete` | "Task {TASK_NAME} was not completed. Review the plan and finish it." |
| `verification_mismatch` | "Previous sub-agent claimed success but verification failed. Re-run all quality gates carefully." |

---

## Task Tool Invocation

### Sequential Phase

```python
# Pseudocode for launching sequential phase sub-agent
result = Task(
    description=f"Execute Phase {phase_id}",
    prompt=format_sequential_prompt(phase, config),
    subagent_type="general-purpose",
    model="sonnet",  # Or inherit from config
    run_in_background=False  # Wait for completion
)
```

### Parallel Phases

```python
# Pseudocode for launching parallel phase sub-agents
task_ids = []

for phase in parallel_batch:
    task_id = Task(
        description=f"Execute Phase {phase.id}",
        prompt=format_parallel_prompt(phase, parallel_batch, config),
        subagent_type="general-purpose",
        model="sonnet",
        run_in_background=True  # Don't wait
    )
    task_ids.append((phase.id, task_id))

# Collect results
results = {}
for phase_id, task_id in task_ids:
    result = TaskOutput(task_id=task_id, block=True, timeout=600000)
    results[phase_id] = parse_result(result)
```

---

## Result Parsing

### Extracting JSON from Sub-Agent Output

Sub-agents may include text before/after the JSON. Extract carefully:

```python
def parse_subagent_result(output):
    # Find JSON block in output
    import json
    import re

    # Look for JSON between ```json and ``` markers
    json_match = re.search(r'```json\s*(.*?)\s*```', output, re.DOTALL)
    if json_match:
        return json.loads(json_match.group(1))

    # Try finding raw JSON object
    json_match = re.search(r'\{[\s\S]*"phase_id"[\s\S]*\}', output)
    if json_match:
        return json.loads(json_match.group(0))

    raise ParseError("Could not extract JSON result from sub-agent output")
```

### Validating Result Structure

```python
def validate_result(result):
    required_fields = [
        "phase_id",
        "status",
        "tasks",
        "quality_gates",
        "commit_message",
        "files",
        "plan_updated"
    ]

    for field in required_fields:
        if field not in result:
            raise ValidationError(f"Missing required field: {field}")

    if result["status"] not in ("complete", "failed"):
        raise ValidationError(f"Invalid status: {result['status']}")

    return True
```

---

## Commit Message Guidelines

Sub-agents generate commit messages following Conventional Commits:

### Message Format

```
<type>(<scope>): <description>

<body>
```

### Type Selection

| Type | When to Use |
|------|-------------|
| `feat` | New feature for users |
| `fix` | Bug fix |
| `refactor` | Code restructure (no behavior change) |
| `test` | Adding/updating tests |
| `docs` | Documentation only |
| `chore` | Build, config, dependencies |

### Scope Derivation

Derive scope from file paths:

```
src/api/auth.ts -> scope: api or auth
src/components/Button.tsx -> scope: ui or button
tests/unit/auth.test.ts -> scope: auth
```

### Example Messages

```
feat(api): add user authentication endpoints

- Implement login endpoint with JWT
- Add token refresh functionality
- Add logout endpoint
```

```
test(auth): add comprehensive auth test coverage

- Add unit tests for token validation
- Add integration tests for login flow
- Add edge case tests for expired tokens
```

---

## Error Handling in Sub-Agents

### Sub-Agent Failure Response

If a sub-agent cannot complete, it should return:

```json
{
  "phase_id": "2a",
  "status": "failed",
  "tasks": {
    "total": 5,
    "completed": 2
  },
  "quality_gates": {
    "lint": {"passed": true, "output": "No errors"},
    "format": {"passed": true, "output": "All formatted"},
    "typecheck": {"passed": true, "output": "No errors"},
    "test": {"passed": false, "output": "3 tests failed", "count": 47}
  },
  "commit_message": null,
  "files": {
    "created": ["src/api/auth.ts"],
    "modified": [],
    "deleted": []
  },
  "plan_updated": false,
  "error": {
    "type": "test_failure",
    "message": "3 tests failing in auth.test.ts",
    "details": "Tests for token validation returning incorrect results. Expected user object, got null."
  }
}
```

### Sub-Agent Timeout

If sub-agent doesn't respond within timeout:

```
SUB-AGENT TIMEOUT
=================

Phase: 2A
Timeout: 10 minutes exceeded

Actions taken:
1. Sub-agent task terminated
2. State file updated with timeout error
3. Phase marked as failed

Error recorded:
{
  "type": "timeout",
  "message": "Sub-agent did not complete within 10 minutes",
  "details": "Task may be stuck or too complex. Consider breaking into smaller tasks."
}

Initiating retry...
```

---

## Sub-Agent Context Loading

Sub-agents should load context efficiently:

### What to Read

1. **Plan file** - Specific phase section only
2. **Referenced files** - Files mentioned in task descriptions
3. **Test files** - To understand existing test patterns

### What NOT to Read

1. Other phase sections (unless explicit dependency)
2. Entire codebase exploration
3. Documentation files (unless task requires)
4. Previous conversation history (isolated by design)

### Efficient Context Pattern

```
# Good: Targeted reads
Read: docs/feature-plan.md (find Phase 2A section)
Read: src/api/routes.ts (modify target)
Read: tests/api/routes.test.ts (test pattern reference)

# Bad: Exploratory reads
Read: src/**/*.ts (too broad)
Grep: "function" (unfocused search)
Read: README.md (not needed for execution)
```
