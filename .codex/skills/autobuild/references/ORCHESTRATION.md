# Orchestration Reference

> Main execution loop, phase coordination, and completion handling.

## Orchestration Overview

The main agent orchestrates execution without accumulating context:

```
┌─────────────────────────────────────────────────────────────────────┐
│                    ORCHESTRATION RESPONSIBILITIES                    │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Main Agent (Orchestrator):                                         │
│  ├── Parse arguments and validate                                   │
│  ├── Initialize .autobuild/ directory                               │
│  ├── Detect technology stack                                        │
│  ├── Parse plan and build execution order                           │
│  ├── Load existing state (if --resume)                              │
│  ├── Launch sub-agents for each phase                               │
│  ├── Verify sub-agent results                                       │
│  ├── Handle commits based on --commit mode                          │
│  ├── Handle failures and retries                                    │
│  └── Output completion or halt summary                              │
│                                                                     │
│  Sub-Agents (Executors):                                            │
│  ├── Read their phase section from plan                             │
│  ├── Execute tasks following TDD                                    │
│  ├── Run quality gates                                              │
│  ├── Update plan file                                               │
│  └── Return structured result JSON                                  │
│                                                                     │
│  Filesystem (State):                                                │
│  ├── .autobuild/config.json - execution config                      │
│  ├── .autobuild/phases/*.json - phase state files                   │
│  └── .autobuild/logs/*.log - execution logs                         │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Main Execution Loop

### Pseudocode

```python
def autobuild_main(plan_path, commit_mode, resume, fresh):
    # Step 1: Initialize
    validate_arguments(plan_path, commit_mode)
    if fresh:
        delete_autobuild_dir()
    create_autobuild_dir()
    config = detect_stack_and_commands()
    save_config(config)

    # Step 2: Parse plan
    plan = read_plan(plan_path)
    phases = parse_phases(plan)
    batches = create_execution_batches(phases)

    # Step 3: Load state (if resume)
    if resume and state_exists():
        states = load_phase_states()
        resume_point = find_resume_point(states, batches)
    else:
        states = create_initial_states(phases)
        resume_point = batches[0]

    # Step 4: Execute batches
    for batch in batches:
        if batch_before(batch, resume_point):
            continue  # Already complete

        if batch["type"] == "sequential":
            result = execute_sequential_batch(batch, config, states)
        else:
            result = execute_parallel_batch(batch, config, states)

        if result["status"] == "failed":
            output_halt_summary(states, result["failed_phase"])
            return

    # Step 5: Completion
    output_completion_summary(states, commit_mode)
```

### Batch Execution Functions

```python
def execute_sequential_batch(batch, config, states):
    for phase_id in batch["phases"]:
        result = execute_phase(phase_id, config, states)

        if result["status"] == "failed":
            # Try retry
            retry_result = retry_phase(phase_id, config, states, result["error"])

            if retry_result["status"] == "failed":
                return {"status": "failed", "failed_phase": phase_id}

    return {"status": "complete"}


def execute_parallel_batch(batch, config, states):
    # Launch all phases in parallel
    tasks = {}
    for phase_id in batch["phases"]:
        task_id = launch_phase_background(phase_id, config)
        tasks[phase_id] = task_id

    # Wait for all to complete
    results = {}
    for phase_id, task_id in tasks.items():
        results[phase_id] = wait_for_task(task_id)

    # Check for failures
    failed = [p for p, r in results.items() if r["status"] == "failed"]

    if failed:
        # Retry failed phases (can also be parallel)
        retry_results = retry_phases_parallel(failed, config, states, results)

        still_failed = [p for p, r in retry_results.items() if r["status"] == "failed"]

        if still_failed:
            return {"status": "failed", "failed_phase": still_failed[0]}

    return {"status": "complete"}
```

---

## Phase Execution Flow

### Single Phase Execution

```
PHASE EXECUTION: {PHASE_ID}
===========================

1. CREATE/UPDATE state file
   Status: running
   Attempt: {N}

2. LAUNCH sub-agent
   [See SUBAGENT-PROMPTS.md]

3. WAIT for sub-agent completion
   Timeout: 10 minutes

4. PARSE sub-agent result
   Extract JSON from output

5. VERIFY results
   Run fresh quality gates
   Check files exist
   Check plan updated

6. IF verification passes:
   - Update state: complete
   - Handle commit
   - Log success

7. IF verification fails:
   - Update state: error details
   - Return failure for retry handling
```

### State File Updates During Execution

```python
def execute_phase(phase_id, config, states):
    # Mark as running
    update_state(phase_id, {
        "status": "running",
        "attempt": states[phase_id]["attempt"] + 1,
        "timestamps": {"started": now()}
    })

    # Launch sub-agent
    result = launch_subagent(phase_id, config)

    # Verify
    verification = verify_result(result, config)

    if verification["passed"]:
        update_state(phase_id, {
            "status": "complete",
            "timestamps": {"completed": now()},
            "quality_gates": verification["gates"],
            "commit": result["commit_message"],
            "files": result["files"]
        })
        return {"status": "complete", "result": result}
    else:
        update_state(phase_id, {
            "error": {
                f"attempt_{states[phase_id]['attempt'] + 1}": {
                    "type": verification["error_type"],
                    "message": verification["error_message"]
                }
            }
        })
        return {"status": "failed", "error": verification}
```

---

## Parallel Execution Coordination

### Launching Parallel Sub-Agents

```
PARALLEL LAUNCH: [2A, 2B, 2C]
=============================

1. For each phase in batch:
   - Create state file (status: running)
   - Launch Task with run_in_background: true
   - Store task ID

2. All tasks launched
   Task 2A: task-abc-123
   Task 2B: task-def-456
   Task 2C: task-ghi-789

3. Wait for completion
   Using TaskOutput with block: true
   Timeout: 15 minutes per task

4. Collect results as they complete
   2C complete: 5m 45s
   2B complete: 8m 30s
   2A complete: 10m 15s
```

### Parallel Task Tool Calls

Launch all parallel phases in a SINGLE message with multiple Task tool calls:

```
[Message with 3 Task tool calls]

Task 1:
  description: "Execute Phase 2A"
  prompt: [2A prompt]
  subagent_type: "general-purpose"
  run_in_background: true

Task 2:
  description: "Execute Phase 2B"
  prompt: [2B prompt]
  subagent_type: "general-purpose"
  run_in_background: true

Task 3:
  description: "Execute Phase 2C"
  prompt: [2C prompt]
  subagent_type: "general-purpose"
  run_in_background: true
```

### Collecting Parallel Results

```python
def collect_parallel_results(task_ids, timeout_per_task=600000):
    results = {}

    for phase_id, task_id in task_ids.items():
        output = TaskOutput(
            task_id=task_id,
            block=True,
            timeout=timeout_per_task
        )
        results[phase_id] = parse_subagent_result(output)

    return results
```

---

## Commit Orchestration

### --commit=auto Flow

```
COMMIT ORCHESTRATION (auto)
===========================

After each phase verification passes:

1. Stage files
   git add {files_created} {files_modified}

2. Create commit
   git commit -m "{commit_message}"

3. Capture commit SHA
   sha = git rev-parse HEAD

4. Update state file
   commit.committed = true
   commit.commit_sha = sha

5. Continue to next phase
```

### --commit=message-only Flow

```
COMMIT ORCHESTRATION (message-only)
===================================

After each phase verification passes:

1. Save commit message to state file
   .autobuild/phases/phase-{id}.json

2. Append to commits script
   .autobuild/commits.sh

3. Do NOT run git commands

4. Continue to next phase

At completion:
  Output list of pending commits
  Point user to commits.sh
```

### --commit=single Flow

```
COMMIT ORCHESTRATION (single)
=============================

After each phase verification passes:

1. Save commit message to state file
   Mark as "queued for combined commit"

2. Do NOT run git commands

3. Continue to next phase

At completion:
  Combine all commit messages
  Create single combined commit message
  Optionally stage and commit all files
```

---

## Completion Handling

### All Phases Successful

```python
def output_completion_summary(states, commit_mode):
    completed = [s for s in states.values() if s["status"] == "complete"]
    total_files = sum(len(s["files"]["created"]) + len(s["files"]["modified"])
                      for s in completed)

    print("""
╔═══════════════════════════════════════════════════════════════════╗
║                     AUTOBUILD COMPLETE                             ║
╠═══════════════════════════════════════════════════════════════════╣
""")

    # Phase summary table
    for state in completed:
        print(f"║  Phase {state['phase_id']}: {state['phase_name']} ... complete")
        print(f"║    Commit: {state['commit']['message'].split(chr(10))[0]}")

    print(f"""
║  ─────────────────────────────────────────────────────────────
║  Total phases: {len(completed)}
║  Total files changed: {total_files}
║  State preserved: .autobuild/
╚═══════════════════════════════════════════════════════════════════╝
""")

    # Commit mode specific instructions
    if commit_mode == "auto":
        print("All commits created. Review with: git log --oneline")
    elif commit_mode == "message-only":
        print("Commits ready. Run: bash .autobuild/commits.sh")
    elif commit_mode == "single":
        output_combined_commit(states)
```

### Creating commits.sh

```python
def create_commits_script(states):
    script = """#!/bin/bash
# Generated by autobuild
# Run with: bash .autobuild/commits.sh

set -e

"""
    for state in sorted(states.values(), key=lambda s: s["timestamps"]["completed"]):
        if state["status"] != "complete":
            continue

        files = state["files"]["created"] + state["files"]["modified"]
        files_str = " ".join(f'"{f}"' for f in files)
        message = state["commit"]["message"]

        script += f'''
echo "Committing Phase {state['phase_id']}: {state['phase_name']}..."
git add {files_str}
git commit -m "$(cat <<'EOF'
{message}
EOF
)"

'''

    script += 'echo "All commits complete!"\n'

    write_file(".autobuild/commits.sh", script)
    Bash(command="chmod +x .autobuild/commits.sh")
```

---

## Logging

### Execution Log Entries

```python
def log_event(event_type, message, details=None):
    timestamp = datetime.now().isoformat()
    entry = f"[{timestamp}] {event_type}: {message}"
    if details:
        entry += f"\n  Details: {details}"

    append_file(".autobuild/logs/execution.log", entry + "\n")
```

### Log Event Types

| Event | Example |
|-------|---------|
| `START` | `AUTOBUILD STARTED - plan: docs/feature-plan.md` |
| `CONFIG` | `Stack detected: typescript/express` |
| `BATCH` | `Executing batch 3 (parallel): [2A, 2B, 2C]` |
| `PHASE_START` | `Phase 2A started (attempt 1)` |
| `PHASE_COMPLETE` | `Phase 2A complete - feat(api): add auth endpoints` |
| `PHASE_FAIL` | `Phase 2A failed - test_failure: 3 tests failing` |
| `RETRY` | `Phase 2A retry initiated (attempt 2)` |
| `VERIFY` | `Verification passed for Phase 2A` |
| `COMMIT` | `Commit created: a1b2c3d` |
| `HALT` | `AUTOBUILD HALTED - Phase 2A failed after retry` |
| `COMPLETE` | `AUTOBUILD COMPLETE - 6 phases, 24 files` |

### Phase Log Files

Each phase gets a truncated log of its sub-agent execution:

```python
def save_phase_log(phase_id, subagent_output):
    # Truncate to reasonable size
    max_lines = 500
    lines = subagent_output.split("\n")
    if len(lines) > max_lines:
        truncated = lines[:max_lines//2] + ["...[truncated]..."] + lines[-max_lines//2:]
        output = "\n".join(truncated)
    else:
        output = subagent_output

    log_content = f"""=== PHASE {phase_id} SUB-AGENT LOG ===
Started: {now()}

{output}

=== END PHASE {phase_id} LOG ===
"""

    write_file(f".autobuild/logs/phase-{phase_id}.log", log_content)
```

---

## Error Recovery

### Graceful Shutdown

If interrupted (Ctrl+C or system shutdown):

```
GRACEFUL SHUTDOWN
=================

Interrupt detected.

Actions:
1. Signal running sub-agents to stop (if possible)
2. Update state files with current status
3. Mark interrupted phases as "running" (will resume)
4. Save execution log

State preserved. Resume with --resume flag.
```

### State Corruption Detection

```python
def validate_state_consistency():
    config = load_config()
    states = load_all_states()

    issues = []

    # Check all expected phases have state files
    for phase_id in config["phases"]:
        if phase_id not in states:
            issues.append(f"Missing state file for phase {phase_id}")

    # Check no phase claims complete without commit
    for state in states.values():
        if state["status"] == "complete" and not state["commit"]:
            issues.append(f"Phase {state['phase_id']} complete but no commit")

    # Check dependency consistency
    for state in states.values():
        if state["status"] == "complete":
            for dep in state["dependencies"]["depends_on"]:
                if states[dep]["status"] != "complete":
                    issues.append(f"Phase {state['phase_id']} complete but dependency {dep} not complete")

    return issues
```

### Recovery from Corruption

```
STATE VALIDATION FAILED
=======================

Issues detected:
1. Phase 2a complete but dependency 1 not complete
2. Missing state file for phase 2c

Options:
1. Attempt automatic repair
2. Start fresh with --fresh
3. Manual state file editing

Recommendation: Start fresh to ensure consistency.

Command: /autobuild docs/feature-plan.md --commit=auto --fresh
```

---

## Progress Tracking

### Progress Output During Execution

```
AUTOBUILD PROGRESS
==================

Plan: docs/feature-plan.md
Mode: --commit=auto

[■■■■■■■■■■░░░░░░░░░░] 50% (3/6 phases)

Completed:
  ✓ Phase 0: Bootstrap (2m 15s)
  ✓ Phase 1: Setup (3m 45s)
  ✓ Phase 2C: Tests (5m 30s)

In Progress:
  ⟳ Phase 2A: Backend (running for 4m 20s)
  ⟳ Phase 2B: Frontend (running for 4m 20s)

Pending:
  ○ Phase 3: Integration

Estimated remaining: ~15 minutes
```

### Progress Update Frequency

- Update after each phase completes
- Update every 60 seconds during long phases
- Immediate update on failure or halt
