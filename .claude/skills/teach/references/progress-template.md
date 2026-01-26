# Progress File Template

Use this template when creating a new progress file at `~/.skulto/teach/{topic-slug}/progress.md`.

---

## Template

```markdown
# Learning Progress: {Topic Name}

**Source:** {list of source document paths}
**Started:** {YYYY-MM-DD}
**Last session:** {YYYY-MM-DD}
**Overall status:** In Progress

## Learning Path

| # | Chunk | Target Level | Status | Score | Last Attempt |
|---|-------|--------------|--------|-------|--------------|
| 1 | {chunk name} | {Bloom level} | ○ Not Started | - | - |
| 2 | {chunk name} | {Bloom level} | ○ Not Started | - | - |
| 3 | {chunk name} | {Bloom level} | ○ Not Started | - | - |

## Session History

### Session 1 - {YYYY-MM-DD}
- Started learning journey
- Chunks attempted: {list}
- Notes: {observations}

## Gaps & Notes

{Track specific concepts that need review or caused difficulty}

## Synthesis Status

○ Not attempted (requires all chunks mastered first)
```

---

## Field Definitions

### Status Values

| Symbol | Status | Meaning |
|--------|--------|---------|
| ✓ | Mastered | Passed mastery ladder (80%+), can proceed |
| ⚠ | In Progress | Currently working on, or failed and needs reteach |
| ○ | Not Started | Haven't attempted yet |

### Target Level (Bloom's)

The highest cognitive level this chunk should be tested at:
- **Remember** - Rarely used as target (too shallow)
- **Understand** - For foundational concepts
- **Apply** - For procedural/how-to content
- **Analyze** - For architectural/systems content
- **Evaluate** - For trade-off heavy content
- **Create** - For synthesis/design content

### Score Format

- `4/4` - Questions correct out of total
- `3.5/4` - Partial credit (e.g., mostly right with minor gap)
- `-` - Not attempted

---

## Example: Filled Progress File

```markdown
# Learning Progress: RAG System Architecture

**Source:** @docs/phase2-infrastructure.md, @docs/rag-overview.md
**Started:** 2024-01-20
**Last session:** 2024-01-23
**Overall status:** In Progress

## Learning Path

| # | Chunk | Target Level | Status | Score | Last Attempt |
|---|-------|--------------|--------|-------|--------------|
| 1 | Why dual storage (SQLite + vectors) | Analyze | ✓ Mastered | 4/4 | 2024-01-20 |
| 2 | SQLite fundamentals and FTS5 | Apply | ✓ Mastered | 3.5/4 | 2024-01-20 |
| 3 | Vector embeddings and LanceDB | Analyze | ⚠ In Progress | 2/4 | 2024-01-23 |
| 4 | The indexing pipeline | Apply | ○ Not Started | - | - |
| 5 | Retrieval strategies | Evaluate | ○ Not Started | - | - |

## Session History

### Session 1 - 2024-01-20
- Completed chunks 1-2
- Backfilled: basic SQL concepts (minor gap)
- Notes: Strong analytical thinking, good at trade-off discussions

### Session 2 - 2024-01-23
- Recall quiz: 3/4 (solid on prior material)
- Attempted chunk 3, scored 2/4
- Identified gap: embedding dimensions concept unclear
- Notes: Needs concrete memory calculation examples

## Gaps & Notes

- **bm25 ordering:** Returns negative scores, lower is better - revisit
- **Embedding dimensions:** Struggled with 384 vs 1536 trade-offs
- **Strength:** Excellent at architectural reasoning and defending choices

## Synthesis Status

○ Not attempted (requires all chunks mastered first)
```

---

## Updating the File

### After Chunk Mastery

1. Change status from `⚠ In Progress` to `✓ Mastered`
2. Update score (e.g., `4/5`)
3. Update last attempt date

### After Failed Attempt

1. Keep status as `⚠ In Progress`
2. Update score to show current attempt
3. Add gap note about what needs work

### At Session End

Add a Session History entry:
```markdown
### Session N - YYYY-MM-DD
- Chunks completed: {list}
- Chunks attempted but not passed: {list}
- Backfills performed: {list if any}
- Key observations: {strengths, struggles, insights}
```

### When All Chunks Mastered

1. Update Synthesis Status to show attempts
2. After passing synthesis: Change overall status to "Completed"

```markdown
## Synthesis Status

✓ Passed - 2024-01-25
- Cross-chunk integration: Solid
- Novel problem: Good diagnosis, practical solution
- Design defense: Excellent, considered multiple perspectives
```
