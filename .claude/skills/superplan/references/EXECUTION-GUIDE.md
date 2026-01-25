# Superplan Execution Guide

This guide provides the detailed execution flow, prompts, and output checkpoints for running Superplan.

---

## Execution Sequence

Follow this exact sequence when running Superplan:

### Step 1: Intake

Start with this prompt to gather requirements:

```
I'll help you create a comprehensive implementation plan.

First, please provide the feature requirements in one of these ways:
1. Paste the user story/requirements directly
2. Share a link to the ticket (Jira, Linear, GitHub, etc.)
3. Describe the feature you want to build

Which would you like to do?
```

**After receiving input:**
- Capture raw requirements exactly as provided
- Identify story type (User Story, Technical Task, Bug Fix, Epic)
- Summarize initial understanding in 1-2 sentences

### Step 2: Detect (USE PARALLEL SUB-AGENTS)

Identify the technology stack to inform best practices research:

**Launch parallel Explore agents to detect:**

1. **Languages** - Primary language(s): TypeScript, Python, Go, Rust, etc.
2. **Frameworks** - Major frameworks: React, Next.js, FastAPI, Django, etc.
3. **Quality Tools** - Existing linters, formatters, type checkers:
   - Look for: `.eslintrc`, `.prettierrc`, `tsconfig.json`, `pyproject.toml`, etc.
4. **Testing Tools** - Test frameworks: jest, pytest, vitest, go test, etc.

**Determine Bootstrap Requirements:**
- If quality tools missing → Plan must include Phase 0: Bootstrap
- If testing not set up → Plan must include Phase 0: Bootstrap

### Step 3: Interview

After intake, ask 3-5 clarifying questions based on what's missing:

```
Before I create the plan, I need to understand a few things:

1. **Scope**: [Question about boundaries/MVP]
2. **Technical**: [Question about constraints/requirements]
3. **Data**: [Question about data flow/storage]
4. **Testing**: [Question about validation/sign-off]

Please answer these so I can create an accurate plan.
```

See [INTERVIEW-GUIDE.md](INTERVIEW-GUIDE.md) for comprehensive question templates.

**Wait for answers before proceeding.**

### Step 4: Research (USE PARALLEL WEB SEARCHES)

After clarification, research current best practices **for the DETECTED stack**:

**Launch parallel web searches targeting detected technologies:**

- `[Language] [YEAR] best practices` (e.g., "TypeScript 2025 best practices")
- `[Framework] [YEAR] patterns` (e.g., "React 2025 patterns")
- `[Framework] security guidelines OWASP`
- `[Language] testing best practices [YEAR]`

**Also research:**
- Industry standards (OWASP, WCAG, etc.)
- Recommended linter/formatter configs for detected stack
- Document sources and key findings

### Step 5: Codebase Exploration

Analyze the existing codebase:

- Identify existing patterns for similar features
- Find integration points and files to modify
- Note shared utilities that can be reused
- Document any technical debt that affects the work

### Step 6: Create Plan

Write the plan to `docs/<feature-name>-plan.md`.

**Include in every plan:**
- Technology Stack section (from DETECT phase)
- Poker estimates for all phases and tasks
- Definition of Done (quality gates) for every phase
- Explicit parallel execution instructions for parallelizable phases

**If Bootstrap required** (from DETECT phase):
- Include Phase 0: Bootstrap before implementation phases

**For large plans exceeding ~20,000 tokens**, split into multiple files:
- `docs/<feature-name>-plan-1.md` - Overview, Requirements, Architecture
- `docs/<feature-name>-plan-2.md` - Phase 0 and parallelizable phases
- `docs/<feature-name>-plan-N.md` - Remaining phases, Appendix

See [PLAN-TEMPLATE.md](PLAN-TEMPLATE.md) for the complete plan structure.

### Step 7: Review

Ask the user to review and provide feedback:

```
The implementation plan is ready for review.

Location: docs/<feature>-plan.md
Phases: X total (Y parallelizable)

Please review and let me know:
1. Any requirements I missed or misunderstood?
2. Any technical concerns with the approach?
3. Ready to proceed, or changes needed?
```

---

## Output Checkpoints

Use these status updates throughout execution to show progress:

### After Intake

```
INTAKE COMPLETE
- Source: [Pasted / Link / MCP / Verbal]
- Type: [User Story / Technical Task / Bug Fix / Epic]
- Summary: [1-2 sentence summary]
```

### After Detect

```
DETECT COMPLETE
- Language: [TypeScript / Python / Go / etc.]
- Framework: [React / Next.js / FastAPI / etc.]
- Quality Tools:
  - Linter: ✅ [eslint] / ❌ Missing
  - Formatter: ✅ [prettier] / ❌ Missing
  - Type Checker: ✅ [tsc] / ❌ Missing
  - Test Framework: ✅ [jest] / ❌ Missing
- Bootstrap Required: [Yes / No]
- Research targets: [List of search queries for detected stack]
```

### After Interview

```
INTERVIEW COMPLETE
- Questions asked: [N]
- Key clarifications:
  - [Clarification 1]
  - [Clarification 2]
- Known unknowns identified: [N]
```

### After Research

```
RESEARCH COMPLETE
- Sources consulted: [N]
- Key findings:
  - [Finding 1]
  - [Finding 2]
- Recommendations: [N]
```

### After Codebase Analysis

```
CODEBASE ANALYSIS COMPLETE
- Files analyzed: [N]
- Patterns identified:
  - [Pattern 1]
  - [Pattern 2]
- Integration points: [N]
```

### After Architecture

```
ARCHITECTURE COMPLETE
- Components designed: [N]
- API endpoints: [N] (if applicable)
- Data model changes: [Y/N]
- Key design decisions:
  - [Decision 1]
  - [Decision 2]
```

### After Phase Definition

```
PHASES DEFINED (with estimates)
- Total phases: [N]
- Total estimate: [Sum] points
- Bootstrap Required: [Yes (5 pts) / No (skipped)]
- Parallelizable: [List phases] - USE SUB-AGENTS
- Sequential: [List phases that must run in order]

Phase Overview with Estimates:
| Phase | Name | Estimate | Parallel With |
|-------|------|----------|---------------|
| 0 | Bootstrap | 5 pts | - (conditional) |
| 1 | Setup | 3 pts | - |
| 2A | [Name] | 8 pts | 2B, 2C |
| 2B | [Name] | 5 pts | 2A, 2C |
| 2C | [Name] | 3 pts | 2A, 2B |
| 3 | Integration | 5 pts | - |

Each phase includes Definition of Done (quality gates).
```

### After Plan Written

```
PLAN WRITTEN
- Location: docs/<feature>-plan.md (or docs/<feature>-plan-*.md if split)
- Files: [1 | N files if split]
- Total phases: [N]
- Total estimate: [Sum] points
- Bootstrap included: [Yes / No (skipped)]
- Parallelizable phases: [N] - USE SUB-AGENTS
- Quality gates: Included per phase
- Lines: ~[N]

Ready for review.
```

---

## Checkpoint Usage

Checkpoints serve multiple purposes:

1. **Progress visibility** - User knows where you are in the process
2. **Context anchors** - Summarize decisions before moving forward
3. **Recovery points** - If interrupted, know where to resume
4. **Quality gates** - Confirm understanding before proceeding

### When to Use Checkpoints

- **Always** use after each major phase (Intake, Interview, Research, etc.)
- **Optionally** use mid-phase for very large features
- **Especially** use when there are many clarifications or complex decisions

### Checkpoint Best Practices

1. Keep summaries concise (3-5 bullet points max)
2. Highlight decisions and their rationale
3. Note any assumptions or unknowns discovered
4. Include counts (files, endpoints, phases) for quick reference

---

## Handling Interruptions

If the conversation is interrupted mid-plan:

### To Resume

1. Read the existing plan file(s)
2. Identify the last completed section
3. Summarize what's done vs. remaining
4. Continue from the next section

### Resume Prompt

```
I see we were working on the [Feature] plan. Let me check the current state.

CURRENT STATE:
- Completed: [Sections done]
- In progress: [Current section]
- Remaining: [Sections left]

[Continue from where we left off]
```

---

## Common Issues & Solutions

### Issue: Scope keeps expanding

**Solution**: Reference the documented Non-Goals and gently redirect:

```
That's a great idea for a future enhancement. For this plan, we agreed to
keep [X] out of scope. I've noted it in the Appendix as a future consideration.
```

### Issue: Missing technical information

**Solution**: Document as a Known Unknown and propose a default:

```
I don't have information about [X]. I'll assume [Y] for now and note this
as a Known Unknown. If this assumption is wrong, we can adjust the plan.
```

### Issue: Plan is getting too large

**Solution**: Proactively split into multiple files:

```
This plan is getting comprehensive. To keep files manageable, I'll split it:
- Part 1: Overview, Requirements, Architecture
- Part 2: Implementation Phases
- Part 3: Testing Strategy, Appendix
```

### Issue: Conflicting requirements

**Solution**: Surface the conflict explicitly:

```
I noticed a potential conflict:
- Requirement A: [X]
- Requirement B: [Y]
- Conflict: [Why they conflict]

Which takes priority, or how should we reconcile this?
```
