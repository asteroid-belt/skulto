# Example Deep Mastery Teaching Session

This shows how a teaching session unfolds with the mastery ladder approach.

---

## Scenario

**Document:** Phase 2 Core Infrastructure Plan (SQLite + LanceDB storage, indexing pipeline)

**Learner:** Software developer, knows Python, no experience with vector databases or RAG systems

---

## Session Transcript

### INTAKE Phase

**Teacher:** I've reviewed the Phase 2 Core Infrastructure Plan. This is substantial material covering:

1. SQLite storage for metadata and full-text search (FTS5)
2. LanceDB for vector embeddings
3. An indexing pipeline to parse YAML files and populate both databases
4. Basic retrieval combining trigger matching and semantic search

The complexity level is intermediate—requires understanding some ML concepts but builds on your Python foundation.

Before we dive in — are there other documents I should include? For example, any related architecture docs or the YAML schema?

**Learner:** No, this should be enough.

**Teacher:** Good. Here's our learning path, with the Bloom's level each chunk targets:

| Chunk | Topic | Target Level |
|-------|-------|--------------|
| 1 | Why dual storage (SQLite + vectors)? | Analyze |
| 2 | SQLite fundamentals and FTS5 | Apply |
| 3 | Vector embeddings and LanceDB | Analyze |
| 4 | The indexing pipeline | Apply |
| 5 | Retrieval strategies | Evaluate |

Each chunk will have a mastery ladder of 3-5 questions you'll need to pass before we advance. Ready for chunk 1?

---

### Chunk 1: Why Dual Storage

**Teacher:** Before we discuss why this system uses both SQLite and a vector database, I need to check your foundation.

What do you understand about how traditional database search (like SQL `LIKE` or full-text search) differs from vector/semantic search?

**Learner:** SQL search looks for exact matches or patterns. I'm not sure about vector search.

**Teacher:** Good clarity on SQL. You've got a gap on vector search—that's a prerequisite for understanding the dual storage architecture. Let me explain embeddings from the ground up.

*[BACKFILL: Vector Search Basics]*

Text gets converted to numbers called "embeddings"—think of them as coordinates in a high-dimensional space. The key insight: **similar meanings end up near each other**.

Here's a concrete example:
- "How do I optimize React performance?" → [0.23, 0.87, -0.12, ...]
- "Make my React app faster" → [0.21, 0.89, -0.14, ...]
- "Getting started with Vue" → [-0.45, 0.12, 0.67, ...]

The first two are "close" in this space because they mean similar things, even though they share few words. The Vue query is "far away" because the topic differs.

A second example: searching for "authentication problems" would match documents about "login errors" and "OAuth token issues"—same meaning, different words.

Common misconception: embeddings don't understand language "deeply"—they capture statistical patterns. "Bank" (financial) and "bank" (river) might be confused without context.

**Key insight:** Semantic search finds meaning similarity, not word matches.

*[BACKFILL VERIFICATION]*

**Teacher:** Let's verify this foundation before proceeding.

Q1 (Understand): In your own words, why would a search for "database performance tuning" match a document about "optimizing SQL queries" in semantic search?

**Learner:** Because they're about the same topic—making databases faster—even though the words are different. The embeddings would be close together.

**Teacher:** Exactly.

Q2 (Apply): Given this query: "fixing broken CSS styles". Which document would semantic search rank highest?
(a) "JavaScript debugging techniques"
(b) "Troubleshooting CSS layout issues"
(c) "HTML best practices"

**Learner:** (b) because it's about fixing CSS problems, which is semantically closest.

**Teacher:** Correct. Foundation verified. Now let's discuss why the system needs BOTH types of search.

*[TEACH CHUNK]*

This system uses dual storage because each excels at different things:

| Storage | Strength | Use Case |
|---------|----------|----------|
| SQLite + FTS5 | Exact phrase matching, filtering by metadata | "Show me all units about 'prompt injection'" |
| LanceDB vectors | Semantic similarity | "How do I prevent LLM attacks?" (matches prompt injection content) |

Why not just use semantic search for everything? Three reasons:

1. **Precision queries:** When users want "FTS5 syntax", they mean those exact letters—not documents about "full-text search in general"

2. **Metadata filtering:** "Show me all ERROR-level logs from yesterday"—this is a filter operation, not similarity

3. **Speed:** Keyword search in indexed tables is faster for exact matches

But why not just use keyword search? Because users often describe what they want conceptually: "How do I make my app faster?" could match dozens of relevant topics that don't contain "faster."

*[MASTERY LADDER]*

**Teacher:** Let's verify mastery of this chunk.

Q1 (Understand): Explain in your own words why using ONLY semantic search would be a problem.

**Learner:** Because sometimes you want exact matches. If I search for "FTS5" I want documents with that specific term, not general full-text search documents.

**Teacher:** Good. That's one reason.

Q2 (Apply): A user searches "SELECT statement syntax". Would you route this to keyword search, semantic search, or both? Why?

**Learner:** Keyword search, because "SELECT statement" is a specific technical term they want to find exactly.

**Teacher:** Correct.

Q3 (Analyze): Compare the trade-offs between using a single powerful vector database versus the dual SQLite + LanceDB approach in this design.

**Learner:** Hmm. A single vector database would be simpler—one system to maintain. But... you'd lose fast exact matching? And filtering on metadata might be slower or awkward in a vector database?

**Teacher:** Good start. What else? Think about what SQLite gives you that a vector database might not.

**Learner:** SQLite is embedded—no separate service. And it's battle-tested for relational data. A vector database might need more setup and maintenance.

**Teacher:** Exactly. The trade-off is complexity (two systems) versus capabilities (right tool for each job).

Q4 (Evaluate): A teammate argues: "We should use Postgres with pgvector—one database for everything." What's your response?

**Learner:** That's a reasonable alternative. You'd get relational queries, full-text search, AND vector search in one system. But... is it as optimized for vector search as LanceDB? Maybe there are performance trade-offs. Also, Postgres requires running a server, while SQLite + LanceDB can both be embedded.

**Teacher:** Solid evaluation. You've identified the key considerations—optimization vs simplicity, embedded vs server-based.

**MASTERY CHECK: 4/4 solid answers with reasoning.** Mastery demonstrated. Let's consolidate before moving on.

---

### Consolidation After Chunk 1

**Teacher:** Good. Let's consolidate.

You now understand:
- Semantic search matches meaning; keyword search matches text
- This system uses both because each excels at different tasks
- Trade-offs: complexity vs capabilities, single-system simplicity vs right-tool-for-job

This foundation is critical because everything else—the schemas, the indexing pipeline, the retrieval logic—builds on this dual-storage decision.

Ready for chunk 2?

---

### Chunk 2: SQLite and FTS5

**Teacher:** Before we dig into the SQLite schema and FTS5, what's your experience with SQLite?

**Learner:** I've used it for small projects. Basic CRUD, some JOINs.

**Teacher:** Good foundation. Have you used full-text search in any database?

**Learner:** No.

**Teacher:** I'll cover FTS5 from scratch then.

*[TEACH CHUNK]*

FTS5 (Full-Text Search 5) is SQLite's full-text search engine.

**Virtual table:** FTS5 creates a special table optimized for text search:
```sql
CREATE VIRTUAL TABLE triggers_fts USING fts5(
    trigger_phrase,
    tokenize='porter unicode61'
);
```

**Tokenizer:** Breaks text into searchable pieces.
- `unicode61`: Handles international characters
- `porter`: Adds stemming—"running" matches "run", "optimization" matches "optimize"

Example: searching "optimizing queries" would match documents containing "optimized", "optimization", "query", "queries".

**BM25 ranking:** Results ranked by relevance, not just "contains match." Documents with more occurrences and shorter overall length rank higher.

**External content:** FTS5 can reference another table, avoiding data duplication:
```sql
CREATE VIRTUAL TABLE triggers_fts USING fts5(
    trigger_phrase,
    content='triggers',  -- References the triggers table
    content_rowid='id'
);
```

The plan uses SQL triggers (not the phrase kind) to keep FTS5 synced when data changes.

Edge case: Porter stemming can over-match. "Universal" stems to "univers" which might match "university"—not always desired.

*[MASTERY LADDER]*

**Teacher:** Mastery verification for FTS5.

Q1 (Remember): What does the `tokenize='porter'` option do?

**Learner:** It enables stemming so "running" matches "run."

**Teacher:** Correct.

Q2 (Understand): Why use `content='triggers'` instead of storing text directly in the FTS5 table?

**Learner:** To avoid duplicating the data. The text lives in the triggers table, and FTS5 just references it.

**Teacher:** Exactly—storage efficiency and single source of truth.

Q3 (Apply): Write the SQL to search for documents containing "authentication" or any stemmed variant, ordered by relevance.

**Learner:**
```sql
SELECT * FROM triggers_fts
WHERE triggers_fts MATCH 'authentication'
ORDER BY bm25(triggers_fts);
```

**Teacher:** Close, but bm25() returns negative values where lower is better. You'd want `ORDER BY bm25(triggers_fts)` ascending, or wrap it. But the core syntax is correct.

Q4 (Analyze): A search for "running tests" returns documents about "marathon running." Is this a bug or expected behavior? What's happening?

**Learner:** Expected behavior—porter stemming reduces "running" to "run" and matches any document with "run." The system doesn't understand context, just stems. This is a limitation of keyword search—it can over-match.

**Teacher:** Exactly. And this is precisely why hybrid search exists—semantic search would understand "running tests" is about software testing.

**MASTERY CHECK: 3.5/4 (the ORDER BY detail was minor).** Threshold passed. One thing to note: bm25 returns negative scores. Let's proceed.

---

### Natural Break Point

**Teacher:** Good stopping point after two chunks.

**MASTERY STATUS:**
✓ Dual storage architecture (4/4, solid with evaluation-level reasoning)
✓ SQLite FTS5 (3.5/4, minor syntax detail to review)

**COVERED:** Why dual storage, how FTS5 provides full-text search
**NEXT:** Vector embeddings and LanceDB, indexing pipeline, retrieval strategies

Continue, or save progress for later?

**Learner:** Let's continue.

---

### Chunk 3: Vector Embeddings and LanceDB

**Teacher:** We covered vector search conceptually. Now let's get concrete.

Do you know what "embedding dimension" means? For example, "384-dimensional embeddings"?

**Learner:** Not really. I know vectors have dimensions but not what it means for embeddings.

**Teacher:** *[BACKFILL]*

An embedding is a list of numbers representing text. The "dimension" is how many numbers.

- all-MiniLM-L6-v2: 384 numbers per text
- OpenAI text-embedding-3-small: 1536 numbers per text

Think of it like coordinates. 2D: (x, y). 3D: (x, y, z). 384D: (x₁, x₂, ..., x₃₈₄).

More dimensions can capture more nuance, but:
- Use more memory (384 floats × 4 bytes = 1.5KB per embedding)
- Slower to search (more numbers to compare)
- Diminishing returns after a point

384 dimensions is a sweet spot—fast, small, good quality for most use cases.

*[BACKFILL VERIFICATION]*

Q1: If you have 100,000 documents with 384-dimensional embeddings, roughly how much memory just for the vectors?

**Learner:** 100,000 × 384 × 4 bytes... about 150MB?

**Teacher:** Exactly. Foundation verified.

*[TEACH CHUNK]*

LanceDB is a vector database designed for similarity search. Key concepts:

**Storage format:** Columnar (like Parquet) for efficient scans. Embeddings stored contiguously for fast SIMD operations.

**Similarity metrics:** Cosine similarity (angle between vectors) is most common for text. Embeddings are typically normalized so cosine similarity equals dot product.

**Indexing:** LanceDB can build indexes for approximate nearest neighbor search. Trades perfect accuracy for speed—might return the 2nd closest instead of 1st closest, but 100x faster.

**Schema in the plan:**
```python
embeddings_table = db.create_table("embeddings", [
    {"unit_id": "...",
     "chunk_text": "...",
     "vector": [...384 floats...]}
])
```

Second example: When you search "how to debug React", LanceDB:
1. Converts your query to a 384-dim vector using the same model
2. Finds the N vectors closest to your query vector
3. Returns the associated document IDs

Edge case: If the embedding model wasn't trained on your domain (e.g., medical text with general model), similarity might be poor.

*[MASTERY LADDER]*

**Teacher:** Let's verify.

Q1 (Understand): In your own words, why must the query and documents use the same embedding model?

**Learner:** Because the "space" is defined by the model. If documents are embedded with model A and the query with model B, they're in different spaces—the positions wouldn't mean the same thing.

**Teacher:** Exactly.

Q2 (Apply): A user complains semantic search is slow on their 10 million document collection. The search takes 5 seconds. What would you check first?

**Learner:** Whether there's an index built. Without an index, it has to compare the query against all 10 million vectors. With an index, it can do approximate search much faster.

**Teacher:** Good. What's the trade-off of adding that index?

**Learner:** Some accuracy—you might not get the absolute closest matches, just very close ones. But probably acceptable for most uses.

**Teacher:** Correct.

Q3 (Analyze): Compare storing embeddings in SQLite (as BLOBs) versus LanceDB. What would you gain and lose with each approach?

**Learner:** SQLite BLOB: Single database, simpler deployment. But no native vector search—you'd have to load all embeddings into memory and compute distances in Python. Slow and memory-heavy at scale.

LanceDB: Purpose-built for vectors, efficient storage and search, can use indexes. But now you have two databases to manage.

The plan chose LanceDB because the vector operations are the bottleneck—you want them optimized.

**Teacher:** Solid analysis.

Q4 (Evaluate): The plan uses all-MiniLM-L6-v2 (384 dimensions). A colleague suggests switching to OpenAI's text-embedding-3-large (3072 dimensions) for "better quality." Make the case for and against.

**Learner:** For OpenAI large: Likely better semantic understanding, trained on more data, might capture subtle meanings that MiniLM misses.

Against: 8x more dimensions means 8x more storage and slower search. Requires API calls with latency and cost. Creates external dependency—if OpenAI is down, your system breaks. Also, for typical search queries, the quality improvement might not matter.

I'd need benchmarks on the actual use case to decide, but MiniLM is a reasonable default unless you prove you need more.

**Teacher:** Excellent evaluation. You considered technical, operational, AND pragmatic factors.

**MASTERY CHECK: 4/4 with depth.** Strong mastery demonstrated.

---

### Consolidation After Chunk 3

**Teacher:** Let's consolidate across all three chunks.

You now understand:
- **Why dual storage:** Keyword (SQLite) for exact matches and metadata, semantic (LanceDB) for meaning similarity
- **SQLite + FTS5:** Full-text search with stemming and BM25 ranking
- **Embeddings + LanceDB:** Text as vectors, similarity search, indexing trade-offs

Notice how the pieces interconnect:
- When a query comes in, the system decides: exact match query → SQLite FTS5; conceptual query → LanceDB; sometimes both
- The indexing pipeline we'll cover next has to populate BOTH databases
- Retrieval logic will merge results from both sources

This is the architecture pattern: **hybrid search**. You now understand all the components. The remaining chunks are about implementation details.

---

### Synthesis Test (End of Session)

After completing all 5 chunks, the session ends with synthesis:

**Teacher:** Final synthesis test. These questions integrate everything.

**Q1 (Cross-chunk integration):** Walk me through what happens when a user searches "how to handle API rate limiting"—from query to results. Touch on both storage systems.

**Learner:**
1. Query comes in: "how to handle API rate limiting"
2. System might check if it's an exact-match query (like "FTS5") or conceptual
3. This seems conceptual, so: convert query to embedding using MiniLM
4. Search LanceDB for similar vectors, get top N document IDs
5. Could also run FTS5 search for "rate limiting" in parallel for exact matches
6. Combine results, maybe rank by some hybrid score
7. Return matched documents to user

**Teacher:** Good flow. One detail: step 2 is often skipped in hybrid—you just run both searches and merge.

**Q2 (Novel problem):** Users report that searching "auth" returns nothing, but "authentication" works. Both should match the same documents. Using what you learned, diagnose and propose a fix.

**Learner:** FTS5 with porter stemming wouldn't help here—"auth" doesn't stem to "authentication." And in embedding space, "auth" might be far from "authentication" if the model wasn't trained on that abbreviation pattern.

Fix options:
1. Query expansion: When user searches "auth", also search "authentication", "authorization"
2. Synonym injection: Add synonyms to the FTS5 index
3. Fine-tune or choose a model that understands domain abbreviations

I'd probably start with query expansion—simplest to implement.

**Teacher:** Excellent diagnosis and practical solution.

**Q3 (Design defense):** Someone proposes: "Let's just use PostgreSQL with pg_trgm for fuzzy text search and pgvector for embeddings. One database, simpler." Argue both for and against, then give your recommendation.

**Learner:**
For Postgres:
- Single database, simpler ops
- ACID transactions across everything
- Mature ecosystem
- Can still do hybrid search

Against:
- pg_trgm is trigram-based, different from porter stemming—might behave unexpectedly
- pgvector, while improving, may not be as optimized as purpose-built vector DBs
- Postgres requires a running server; SQLite + LanceDB can be embedded
- More setup complexity initially

My recommendation: Depends on context. For a production service with existing Postgres infrastructure and an ops team? Postgres combo makes sense—simpler to manage one thing.

For a CLI tool or embedded use case? The current SQLite + LanceDB is better—no server needed.

**Teacher:** That's exactly the right framing—context-dependent recommendation with clear reasoning.

**SYNTHESIS MASTERY: PASSED**

---

**FINAL SESSION STATUS:**

✓ Dual storage architecture (4/4, evaluation-level)
✓ SQLite FTS5 (3.5/4, minor detail)
✓ Vector embeddings and LanceDB (4/4, with trade-off analysis)
✓ Indexing pipeline (verified in session)
✓ Retrieval strategies (verified in session)
✓ Synthesis test (3/3, cross-chunk integration demonstrated)

**MASTERY LEVEL: Solid**

You can now:
- Explain the architecture to others
- Make informed trade-off decisions
- Debug issues in the system
- Evaluate alternative approaches

**Next steps if continuing:**
- Hands-on: Implement a simple hybrid search
- Depth: Study specific embedding models and their trade-offs
- Breadth: Explore other vector databases (Pinecone, Weaviate, Qdrant)

---

## Key Patterns Demonstrated

1. **Mastery ladder per chunk** — 3-4 questions climbing Bloom's levels
2. **80% threshold enforced** — Didn't advance until demonstrated
3. **Backfill with verification** — Didn't assume foundation, tested it
4. **Consolidation** — Connected chunks, built integrated picture
5. **Synthesis test** — Final integration verification
6. **Mastery status tracking** — Clear scores at breaks
7. **Multiple examples per concept** — Not one-and-done
8. **Higher-order questions** — Analyze, Evaluate, not just Remember
