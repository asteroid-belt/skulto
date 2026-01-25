---
name: superplan
description: Create comprehensive, multi-phase implementation plans for features. Detects technology stack, interviews user, researches current best practices, generates parallelizable phases with poker estimates, TDD-first acceptance criteria, quality gates (linters/formatters/typecheckers), and detailed code deltas. Use when starting significant features, epics, or complex tasks.
metadata:
  version: "2.0"
  generated-at: "2025-01-13"
compatibility: Requires internet access for best practices research. Works with any codebase.
---

# Superplan: Comprehensive Feature Planning

## Overview

Superplan creates detailed, executable implementation plans that enable parallel agent execution. Each plan includes everything needed to implement a feature: requirements, architecture, code changes, tests, and acceptance criteria.

## When to Use Superplan

- Starting a new feature or epic
- Complex tasks requiring multiple phases
- Tasks that could benefit from parallel execution by multiple agents
- When you need comprehensive documentation of implementation decisions
- When the team needs to understand the full scope before committing

## Core Workflow

```
┌─────────────────────────────────────────────────────────────────────┐
│                         SUPERPLAN WORKFLOW                          │
├─────────────────────────────────────────────────────────────────────┤
│  1. INTAKE          →  Gather story/requirements from user          │
│  2. DETECT          →  Identify tech stack, languages, frameworks   │
│  3. INTERVIEW       →  Ask clarifying questions                     │
│  4. RESEARCH        →  Look up best practices for DETECTED STACK    │
│  5. EXPLORE         →  Understand existing codebase patterns        │
│  6. ARCHITECT       →  Design solution with diagrams                │
│  7. PHASE           →  Break into parallelizable phases + ESTIMATES │
│  8. DETAIL          →  Specify code deltas per phase                │
│  9. TEST            →  Define failing tests per phase (TDD)         │
│ 10. DOCUMENT        →  Write plan to docs/<feature>-plan.md         │
└─────────────────────────────────────────────────────────────────────┘
```

---

## CRITICAL: Parallel Execution with Sub-Agents

**YOU MUST USE SUB-AGENTS OR PARALLEL TASKS** for every parallelizable operation:

| Operation | How to Execute |
|-----------|---------------|
| Independent file reads | Launch multiple Read tasks in single message |
| Code searches | Use Task tool with multiple Explore agents in parallel |
| Parallel phases (1A, 1B, 1C) | Execute using parallel sub-agents |
| Independent test suites | Run unit/integration/e2e concurrently |

**Example**: "Launch 3 sub-agents in parallel to implement Phases 1A, 1B, and 1C"

---

## Poker Planning Estimates

All tasks and phases MUST include Fibonacci estimates: **1, 2, 3, 5, 8, 13, 21**

| Size | Meaning | Example |
|------|---------|---------|
| 1 | Trivial | Config value, typo fix |
| 2 | Small | Single file, simple function |
| 3 | Medium | Multi-file, new component |
| 5 | Large | Feature module, API endpoint |
| 8 | X-Large | Complex feature with dependencies |
| 13 | Epic chunk | Major subsystem change |
| 21 | Too big | **Split into smaller tasks** |

---

## Phase 1: INTAKE - Gather Requirements

### What You're Doing
Collecting the feature requirements from the user through story input.

### Actions

1. **Ask the user to provide their story/requirements** using one of these methods:
   - Copy/paste the story text directly
   - Provide a link to a ticket/story (Jira, Linear, GitHub Issue, etc.)
   - Use an MCP tool to fetch the story if available
   - Describe the feature verbally

2. **Capture the raw input** exactly as provided

3. **Identify the story type**:
   - User Story (`As a <role>, I want <capability>, so that <benefit>`)
   - Technical Task (implementation-focused)
   - Bug Fix (problem/solution)
   - Epic (large feature with sub-stories)

### Output
Document: Source, Type, Raw Requirements, Initial Understanding (1-2 sentences).

---

## Phase 2: DETECT - Technology Stack Analysis

### What You're Doing
Identifying the technology, programming language, and major frameworks to inform best practices research and quality gate setup.

### Detection Actions (USE PARALLEL SUB-AGENTS)

Launch **parallel Explore agents** to detect:

1. **Languages** - Primary language(s): TypeScript, Python, Go, Rust, Java, etc.
2. **Frameworks** - Major frameworks: React, Next.js, FastAPI, Django, Express, etc.
3. **Build Tools** - Package managers, bundlers: npm, pnpm, yarn, webpack, vite, etc.
4. **Quality Tools** - Existing linters, formatters, type checkers:
   - Linters: eslint, ruff, golint, pylint
   - Formatters: prettier, black, gofmt, rustfmt
   - Type checkers: tsc (TypeScript), mypy (Python), go vet
5. **Testing Tools** - Test frameworks: jest, pytest, go test, vitest, playwright

### Quality Tools Assessment

| Tool Type | If Present | If Missing |
|-----------|------------|------------|
| Linter | Note config path | Add to Phase 0 Bootstrap |
| Formatter | Note config path | Add to Phase 0 Bootstrap |
| Type Checker | Note config path | Add to Phase 0 Bootstrap |
| Test Framework | Note config path | Add to Phase 0 Bootstrap |

### Output
Document: Languages, frameworks, build tools, quality tools (present/missing), testing setup, Bootstrap requirements.

---

## Phase 3: INTERVIEW - Clarifying Questions

### What You're Doing
Asking targeted questions to fill gaps in requirements before planning.

### Question Categories

Ask questions in these categories (select relevant ones):

1. **Scope & Boundaries** - MVP, out of scope, related features
2. **User Experience** - Primary users, user flow, mockups/designs
3. **Technical Constraints** - Performance, security, compliance, integrations
4. **Data & State** - Data sources, storage, migrations
5. **Testing & Validation** - Success criteria, test scenarios, sign-off
6. **Dependencies** - Blockers, external services, sequencing
7. **Quality Gates** - Existing CI/CD, required checks, coverage thresholds

### How to Interview

Ask 3-5 most relevant questions, then **wait for answers before proceeding**.

See [Interview Guide](references/INTERVIEW-GUIDE.md) for comprehensive question templates and strategies by feature size.

---

## Phase 4: RESEARCH - Best Practices Lookup

### What You're Doing
Researching current best practices as of TODAY'S DATE **for the DETECTED technology stack**.

### Research Actions (USE PARALLEL WEB SEARCHES)

Launch **parallel web searches** targeting the detected stack:

1. **[Language] [YEAR] best practices** - e.g., "TypeScript 2025 best practices"
2. **[Framework] [YEAR] patterns** - e.g., "React 2025 patterns"
3. **[Framework] security guidelines** - e.g., "Next.js security OWASP"
4. **[Language] testing best practices** - e.g., "Python pytest 2025"

### Research Areas

1. **Industry Standards** - Current best practices, security (OWASP), accessibility (WCAG)
2. **Technology-Specific** - Framework patterns, library versions, known gotchas
3. **Architecture Patterns** - Recommended patterns, anti-patterns, scalability
4. **Quality Tool Configs** - Recommended linter/formatter configs for detected stack

### Output
Document: Sources consulted, key findings by topic, specific recommendations for this feature.

---

## Phase 5: EXPLORE - Codebase Analysis

### What You're Doing
Understanding existing patterns, conventions, and integration points in the codebase.

### Exploration Targets

1. **Existing Patterns** - How similar features are implemented, conventions, testing patterns
2. **Integration Points** - Files/modules to touch, API contracts, shared utilities
3. **Technical Debt** - Areas needing refactoring, known issues affecting this work

### Output
Document: Relevant files table, patterns to follow, integration points, technical debt notes.

---

## Phase 6: ARCHITECT - Solution Design

### What You're Doing
Designing the technical solution with diagrams.

### Architecture Components

1. **High-Level Architecture** - System/component diagram, data flow
2. **API Design** (if applicable) - Endpoints, request/response schemas, error handling
3. **Data Model** (if applicable) - Schema changes, migrations
4. **Component Design** (if applicable) - Component hierarchy, state management, interfaces

### Diagram Format

Use ASCII/text diagrams for portability. See [Plan Template](references/PLAN-TEMPLATE.md) for diagram examples.

---

## Phase 7: PHASE - Parallel Work Breakdown with Estimates

### What You're Doing
Breaking work into phases with **poker estimates** that can be executed in parallel where possible.

### Phase Design Principles

1. **Independence**: Phases executable without waiting for others when possible
2. **Testability**: Each phase independently testable
3. **Clear Boundaries**: Well-defined inputs and outputs
4. **Estimated Size**: Each phase with poker estimate (target: 3-8 points)
5. **Quality Gated**: Each phase includes Definition of Done

### Parallelization Rules (CRITICAL: USE SUB-AGENTS)

- **CAN parallelize**: Independent components, separate API endpoints, unrelated tests
  - **Execute with parallel sub-agents**
- **CANNOT parallelize**: Sequential dependencies, shared state setup, migrations before data access

### Output

Create a phase dependency diagram showing:
- Phase 0 (Bootstrap/Setup) at top - **CONDITIONAL: only if quality tools missing**
- Parallel phases branching out
- Integration phase at bottom

Include dependency table with estimates:

| Phase | Name | Depends On | Parallel With | Estimate | Status |
|-------|------|------------|---------------|----------|--------|

---

## Phase 8: DETAIL - Code Deltas Per Phase

### What You're Doing
Specifying exact code changes for each phase.

### Code Delta Format

For each file, specify:
- **File path** and action: `(CREATE)` or `(MODIFY)`
- **Full file contents** for new files
- **Diff format** for modifications (show context lines with +/- changes)

See [Plan Template](references/PLAN-TEMPLATE.md) for full code delta examples.

---

## Phase 9: TEST - TDD Acceptance Criteria

### What You're Doing
Defining failing tests for each phase BEFORE implementation.

### Testing Pyramid

Follow the pyramid - more unit tests, fewer integration, even fewer E2E:
- **Unit Tests (80%)**: Business logic, utilities - fast, many
- **Integration Tests (15%)**: API contracts, database - medium
- **E2E Tests (5%)**: Critical user journeys only - slow, few

### Test-First Workflow

For each phase:
1. Write tests FIRST (they will fail)
2. Run tests to confirm they fail
3. Implement the code
4. Run tests to confirm they pass
5. All existing tests must still pass

### Durable vs Brittle Tests

Write **durable tests** that test behavior/outcomes, not implementation details:
- Test what the function DOES, not HOW it does it
- Test public API, not internal state
- Tests should survive refactoring without changes

See [Testing Pyramid](references/TESTING-PYRAMID.md) for comprehensive test examples and strategies.

---

## Definition of Done (Quality Gate) - REQUIRED PER PHASE

Every phase MUST include this Definition of Done checklist:

```markdown
### Definition of Done
- [ ] Code passes linter ([detected linter]: eslint/ruff/golint)
- [ ] Code passes formatter check ([detected formatter]: prettier/black/gofmt)
- [ ] Code passes type checker ([detected checker]: tsc/mypy/go vet)
- [ ] All new code has test coverage (target: 80%+)
- [ ] All new tests pass
- [ ] All existing tests still pass
- [ ] No new warnings introduced
```

**IF QUALITY TOOLS MISSING**: Plan must include **Phase 0: Quality Bootstrap** before any implementation phases.

---

## Phase 10: DOCUMENT - Write the Plan

### What You're Doing
Writing the complete plan to `docs/<feature>-plan.md`.

### Multi-File Strategy

Large plans may exceed file size limits (~25,000 tokens). When this happens:

1. **Split by logical boundaries**:
   - **File 1 (`-plan-1.md`)**: Executive Summary, Requirements, Research, Architecture
   - **File 2 (`-plan-2.md`)**: Phase 0 and Phase 1 (parallelizable phases)
   - **File 3+ (`-plan-N.md`)**: Remaining phases, Testing Strategy, Assumptions, Appendix

2. **Each file must be self-contained**:
   - Include header referencing the plan set
   - Include navigation links to other parts
   - Include Table of Contents for that file

3. **Size thresholds** (~4 tokens per word):
   - Under ~4,000 lines / ~20,000 tokens: Single file
   - Over ~4,000 lines / ~20,000 tokens: Split into multiple files

### Multi-File Header

```markdown
# [Feature Name] Implementation Plan - Part [N] of [Total]

> **Plan Set**: `docs/<feature>-plan-*.md`
> **This File**: Part [N] - [Section Names]
> **Navigation**: [Part 1](feature-plan-1.md) | [Part 2](feature-plan-2.md) | ...
```

### Plan Structure

See [Plan Template](references/PLAN-TEMPLATE.md) for the complete plan file structure including all sections, formats, and examples.

### Chunked Writing

For large plans, write in chunks to prevent context loss:
1. Write Overview + Requirements → Save
2. Write Architecture → Save
3. Write each Phase separately → Save after each
4. Write Assumptions + Appendix → Save

---

## Execution Flow

### Quick Reference

1. **Intake** → Ask for requirements, capture raw input, identify type
2. **Detect** → Identify tech stack, quality tools (USE PARALLEL AGENTS)
3. **Interview** → Ask 3-5 clarifying questions, wait for answers
4. **Research** → Web search best practices FOR DETECTED STACK (USE PARALLEL SEARCHES)
5. **Explore** → Analyze codebase patterns and integration points
6. **Architect** → Design solution with diagrams
7. **Phase** → Break into parallelizable phases WITH POKER ESTIMATES
8. **Detail** → Specify code deltas per phase WITH DEFINITION OF DONE
9. **Test** → Define failing tests (TDD) per phase
10. **Document** → Write to `docs/<feature>-plan.md`

### Status Updates

Use checkpoint summaries after each major phase to show progress. Examples:

```
DETECT COMPLETE
- Language: TypeScript
- Framework: Next.js 14
- Quality Tools: eslint ✅, prettier ✅, tsc ✅, jest ✅
- Bootstrap Required: No
```

```
PHASES DEFINED (with estimates)
- Total phases: 5
- Total estimate: 26 points
- Parallelizable: 1A (8pts), 1B (5pts), 1C (3pts) - USE SUB-AGENTS
- Sequential: Phase 0 (5pts) → Phase 1s → Phase 2 (5pts)
```

See [Execution Guide](references/EXECUTION-GUIDE.md) for full execution instructions, prompts, and checkpoint templates.

---

## References

For detailed guidance on specific topics:

- [Interview Guide](references/INTERVIEW-GUIDE.md) - Comprehensive question templates by category and feature size
- [Plan Template](references/PLAN-TEMPLATE.md) - Full plan file structure with all sections and examples
- [Testing Pyramid](references/TESTING-PYRAMID.md) - TDD workflow, test examples, durable vs brittle tests
- [Execution Guide](references/EXECUTION-GUIDE.md) - Step-by-step execution flow and checkpoint templates
