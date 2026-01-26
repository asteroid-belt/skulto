# Plan Document Updates (Living Memory)

**After ALL quality gates pass, BEFORE generating commit message:**

The plan document is the living memory of execution. Update it to reflect completed work.

---

## Update Sequence

```
┌─────────────────────────────────────────────────────────────────────┐
│                    PLAN UPDATE SEQUENCE                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  1. READ current plan file                                          │
│          ↓                                                          │
│  2. LOCATE phase section being completed                            │
│          ↓                                                          │
│  3. UPDATE task checkboxes (- [ ] → - [x])                          │
│          ↓                                                          │
│  4. UPDATE DoD checkboxes (- [ ] → - [x])                           │
│          ↓                                                          │
│  5. UPDATE phase status in overview table (⬜ → ✅)                  │
│          ↓                                                          │
│  6. VERIFY all edits applied correctly                              │
│          ↓                                                          │
│  7. OUTPUT confirmation message                                     │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Pattern Recognition

### Task Checkbox Patterns

**Recognize these as unchecked tasks:**
```markdown
- [ ] Task description
* [ ] Task description
- [] Task description (no space - still valid)
  - [ ] Nested task
```

**Transform to checked:**
```markdown
- [x] Task description
* [x] Task description
- [x] Task description
  - [x] Nested task
```

### Status Indicator Patterns

**Recognize these status indicators:**

| Pattern | Meaning | Replace With |
|---------|---------|--------------|
| `⬜` | Not started | `✅` |
| `🔲` | Not started | `✅` |
| `[ ]` | Not started | `[x]` |
| `TODO` | Not started | `DONE` |
| `Not Started` | Not started | `Complete` |
| `Pending` | Not started | `Complete` |
| `🔄` | In progress | `✅` |
| `⏳` | In progress | `✅` |
| `In Progress` | In progress | `Complete` |

### Table Status Patterns

**Phase overview table formats:**

```markdown
# Format 1: Emoji status
| Phase | Name | Est. | Status |
|-------|------|------|--------|
| 1 | Setup | 3 | ⬜ |        ← Change to ✅

# Format 2: Text status
| Phase | Name | Est. | Status |
|-------|------|------|--------|
| 1 | Setup | 3 | Not Started | ← Change to Complete

# Format 3: Checkbox in table
| Phase | Name | Est. | Done |
|-------|------|------|------|
| 1 | Setup | 3 | [ ] |      ← Change to [x]
```

---

## Required Updates (In Order)

### 1. Phase Task Checkboxes

Locate the phase section and check off ALL completed tasks:

```markdown
### Phase 2A: Backend API

#### Tasks

# BEFORE
- [ ] Create API route handler
- [ ] Implement request validation
- [ ] Add authentication middleware
- [ ] Write unit tests

# AFTER
- [x] Create API route handler
- [x] Implement request validation
- [x] Add authentication middleware
- [x] Write unit tests
```

**Edit command pattern:**
```
old_string: "- [ ] Create API route handler"
new_string: "- [x] Create API route handler"
```

### 2. Definition of Done Checkboxes

Locate the DoD section within the phase and check off ALL items:

```markdown
#### Definition of Done

# BEFORE
- [ ] Code passes linter
- [ ] Code passes formatter
- [ ] Code passes type checker
- [ ] All tests pass
- [ ] No new warnings

# AFTER
- [x] Code passes linter
- [x] Code passes formatter
- [x] Code passes type checker
- [x] All tests pass
- [x] No new warnings
```

### 3. Phase Status in Overview Table

Locate the phase overview table and update status:

```markdown
## Phase Overview

# BEFORE
| Phase | Name | Depends On | Status |
|-------|------|------------|--------|
| 1 | Setup | - | ✅ |
| 2A | Backend | 1 | ⬜ |
| 2B | Frontend | 1 | ⬜ |

# AFTER
| Phase | Name | Depends On | Status |
|-------|------|------------|--------|
| 1 | Setup | - | ✅ |
| 2A | Backend | 1 | ✅ |      ← Updated
| 2B | Frontend | 1 | ⬜ |
```

**Edit command pattern:**
```
old_string: "| 2A | Backend | 1 | ⬜ |"
new_string: "| 2A | Backend | 1 | ✅ |"
```

---

## Locating Sections

### Finding Phase Section

Search patterns (in order of specificity):

1. `### Phase 2A:` or `## Phase 2A:`
2. `### Phase 2A -` or `## Phase 2A -`
3. `### 2A:` or `### 2A.`
4. Section containing phase identifier

### Finding DoD Section

Search patterns within phase section:

1. `#### Definition of Done`
2. `### Definition of Done`
3. `**Definition of Done**`
4. `DoD:` or `DOD:`
5. Section with quality gate checkboxes

### Finding Overview Table

Search patterns:

1. `## Phase Overview`
2. `### Phase Overview`
3. `## Implementation Phases`
4. Table containing `| Phase |` header

---

## Verification Steps

After making edits, verify:

### 1. Count Verification

```
Expected: X tasks checked in phase
Actual: Count - [x] occurrences in phase section
Match: Yes/No
```

### 2. Status Consistency

```
Phase 2A tasks: All [x] ✓
Phase 2A DoD: All [x] ✓
Phase 2A table status: ✅ ✓
```

### 3. No Partial Updates

If ANY edit fails, report failure and retry. Do not leave plan in inconsistent state.

---

## Output Format

After successful updates:

```
PLAN DOCUMENT UPDATED
━━━━━━━━━━━━━━━━━━━━━

File: docs/feature-plan.md
Phase: 2A - Backend API

Updates applied:
- Tasks: 4/4 items checked [x]
- Definition of Done: 6/6 items checked [x]
- Overview table: ⬜ → ✅

Verification:
- All edits confirmed in file
- No inconsistencies detected

The plan now reflects completed work.
Proceeding to generate commit message...
```

---

## Error Handling

### Plan File Not Found

```
⛔ PLAN UPDATE FAILED
━━━━━━━━━━━━━━━━━━━━━

Error: Plan file not found at specified path

Expected: docs/feature-plan.md
Actual: File does not exist

Action Required:
1. Verify plan file path
2. Provide correct path to plan document
```

### Section Not Found

```
⛔ PLAN UPDATE FAILED
━━━━━━━━━━━━━━━━━━━━━

Error: Could not locate phase section

Searched for:
- "### Phase 2A"
- "## Phase 2A"
- Phase overview table row

Action Required:
1. Verify plan document format
2. Ensure phase section exists
3. Check for non-standard formatting
```

### Edit Conflict

```
⛔ PLAN UPDATE FAILED
━━━━━━━━━━━━━━━━━━━━━

Error: Edit string not found (may have changed)

Attempted: old_string: "- [ ] Create API handler"
Reason: String not unique or already modified

Action Required:
1. Re-read the plan file
2. Locate current state of checkbox
3. Retry with correct string
```

---

## Why This Matters

| Benefit | Explanation |
|---------|-------------|
| **Survives compaction** | Plan file persists; conversation context may not |
| **Visual progress** | Anyone can see status without reading logs |
| **Session handoff** | New session reads plan to understand state |
| **Audit trail** | Git history shows when each phase completed |
| **Prevents re-work** | Clear indication of what's done |

---

## Edge Cases

### Parallel Phases

When multiple phases complete together (2A, 2B, 2C):

1. Update ALL phase task checkboxes
2. Update ALL phase DoD checkboxes
3. Update ALL phase statuses in overview table
4. Verify ALL updates before confirming

### Partial Phase Completion

If a phase is partially complete (some tasks done, some pending):

- Do NOT update overview table status
- Only check off completed tasks
- Leave incomplete tasks unchecked
- Note partial completion in output

### Plan Split Across Files

If plan is in multiple files (e.g., `plan-part-1.md`, `plan-part-2.md`):

1. Identify which file contains current phase
2. Update only that file
3. If overview table is in different file, update both files
