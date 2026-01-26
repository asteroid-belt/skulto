# Verification Examples: The Mastery Ladder

This reference shows how to construct mastery ladders for different content types, climbing through Bloom's cognitive levels.

---

## The Mastery Ladder Principle

Each chunk requires **3-5 questions** that climb cognitive levels:

```
Bloom's Levels (ascending difficulty):
1. Remember  → Recall facts
2. Understand → Explain in own words
3. Apply     → Use in new situations
4. Analyze   → Break down, compare, contrast
5. Evaluate  → Judge, critique, defend
6. Create    → Design new solutions
```

**Minimum ladder:** Understand → Apply → Analyze
**Full ladder:** Understand → Apply → Analyze → Evaluate

Never verify with only Remember-level questions—that tests recognition, not understanding.

---

## Example Ladders by Content Type

### Conceptual Content (Theory, Mental Models)

**Topic:** Why RAG systems use retrieval + generation

```
Q1 (Understand): "In your own words, why can't we just give an LLM
    a million documents in its context window?"

PASSING SIGNALS:
- Mentions context window limits
- Notes cost/latency of large contexts
- Understands that retrieval is selective

Q2 (Apply): "A user asks about a company policy document from last
    month. Your LLM's training data is 6 months old. How would a
    RAG system handle this?"

PASSING SIGNALS:
- Retrieve the policy document first
- Include retrieved content in prompt
- LLM generates answer using fresh data

Q3 (Analyze): "Compare RAG with fine-tuning. When would you choose
    each approach?"

PASSING SIGNALS:
- RAG: When data changes frequently, when you need citations
- Fine-tuning: When you want behavioral changes, style adaptation
- Understands trade-offs of each

Q4 (Evaluate): "A colleague says: 'Just use a bigger context window
    instead of building RAG—it's simpler.' What's your response?"

PASSING SIGNALS:
- Acknowledges the simplicity argument
- Counters with: cost, latency, retrieval precision, update flexibility
- Makes nuanced recommendation based on use case
```

**THRESHOLD:** 3/4 with solid reasoning

---

### Procedural Content (How-to, Workflows)

**Topic:** Setting up a Python project with uv

```
Q1 (Understand): "What's the difference between `uv init` and
    `uv init --package`? When would you use each?"

PASSING SIGNALS:
- Knows --package creates src/ layout
- Understands package vs script distinction
- Can articulate when each matters

Q2 (Apply): "You're starting a new CLI tool project called 'myapp'.
    Walk me through the first 5 commands you'd run."

PASSING SIGNALS:
- `uv init --package myapp`
- `cd myapp`
- `uv add typer rich` (or similar deps)
- `uv sync`
- `uv run pytest` (or manual verification)

Q3 (Apply): "A test requires a dev-only dependency (pytest-cov).
    How do you add it without including in production deps?"

PASSING SIGNALS:
- `uv add --dev pytest-cov`
- Knows about dependency groups
- Understands prod vs dev separation

Q4 (Analyze): "Your CI pipeline runs `pip install .` but your local
    dev uses `uv sync`. What problems might this cause?"

PASSING SIGNALS:
- Dependency version differences
- Lock file not used by pip
- Potential for "works on my machine" bugs
- Solution: use uv in CI too, or ensure pyproject.toml is precise
```

**THRESHOLD:** 3/4 with correct commands and reasoning

---

### Factual Content (Definitions, Specs)

**Topic:** Embedding model dimensions

```
Q1 (Remember): "What's the embedding dimension of all-MiniLM-L6-v2?"
    (a) 128  (b) 384  (c) 768  (d) 1536

ANSWER: (b) 384

Q2 (Understand): "If embeddings from model A are 384-dimensional and
    model B are 1536-dimensional, can you directly compare them?
    Why or why not?"

PASSING SIGNALS:
- No, different spaces
- The numbers mean different things
- Would need same model or mapping between them

Q3 (Apply): "You have 500,000 documents to embed. Calculate the
    storage for 384-dim vs 1536-dim embeddings. (Assume float32)"

PASSING SIGNALS:
- 384: 500K × 384 × 4 bytes = 768MB
- 1536: 500K × 1536 × 4 bytes = 3GB
- 4x storage difference

Q4 (Analyze): "Given the storage difference, when would the larger
    model be worth it?"

PASSING SIGNALS:
- When semantic precision is critical
- When query complexity is high
- When you've benchmarked and smaller model fails
- NOT when: standard search, cost-sensitive, speed-critical
```

**THRESHOLD:** 3/4 with calculations correct

---

### Architectural Content (Systems, Trade-offs)

**Topic:** Choosing between monolith and microservices

```
Q1 (Understand): "Explain the fundamental difference in how a
    monolith and microservices handle a change to the user auth
    system."

PASSING SIGNALS:
- Monolith: Change in one codebase, single deployment
- Microservices: Change auth service, deploy independently
- Understands coupling implications

Q2 (Apply): "A startup has 3 developers and wants to launch in 2
    months. They're debating architecture. Which do you recommend
    and why?"

PASSING SIGNALS:
- Monolith for speed and simplicity
- Small team can't support microservices overhead
- Can extract services later if needed
- Ship fast, refactor when necessary

Q3 (Analyze): "Your monolith is struggling with the checkout flow
    during peak traffic while other pages are fine. What are your
    options and trade-offs?"

PASSING SIGNALS:
- Scale the whole thing (wasteful but simple)
- Extract checkout as a service (targeted but complex)
- Optimize the hot path (might not be enough)
- Add caching/queue (can help but limited)

Q4 (Evaluate): "A VP says: 'Netflix uses microservices, so should
    we.' Construct your counter-argument."

PASSING SIGNALS:
- Netflix scale ≠ our scale
- Netflix has thousands of engineers
- Our team can't maintain dozens of services
- Architecture should match organization (Conway's Law)
- What problem are we solving?

Q5 (Create): "Design a migration path: you have a successful
    monolith that's hitting scaling limits. How would you gradually
    extract services?"

PASSING SIGNALS:
- Strangler fig pattern
- Start with highest-pain/most-independent
- Add API gateway for routing
- Keep monolith running during transition
- Move data ownership gradually
```

**THRESHOLD:** 4/5 with nuanced reasoning

---

### Code-Heavy Content

**Topic:** Pydantic model validation

```
Q1 (Understand): "What happens when you create a Pydantic model
    with a field that doesn't match the type annotation?"

PASSING SIGNALS:
- ValidationError raised
- Pydantic attempts coercion first (e.g., "123" → 123)
- Strict mode can disable coercion

Q2 (Apply): "Write a Pydantic model for a BlogPost with:
    - title (required, 5-100 chars)
    - content (required)
    - tags (optional list of strings)
    - published_at (optional datetime)"

PASSING CODE:
```python
from pydantic import BaseModel, Field
from typing import Optional
from datetime import datetime

class BlogPost(BaseModel):
    title: str = Field(..., min_length=5, max_length=100)
    content: str
    tags: Optional[list[str]] = None
    published_at: Optional[datetime] = None
```

Q3 (Analyze): "Compare Field validation vs @validator decorator.
    When would you use each?"

PASSING SIGNALS:
- Field: Simple constraints (min/max, regex)
- @validator: Complex logic, cross-field validation
- Field is declarative, @validator is imperative
- Use Field when possible, @validator when necessary

Q4 (Evaluate): "A teammate puts all validation in @validator
    methods instead of using Field constraints. Is this a problem?"

PASSING SIGNALS:
- Harder to see constraints at a glance
- More verbose than necessary
- Field constraints are optimized by Pydantic
- But: sometimes complex logic requires validators
- Recommendation: Field for simple, validator for complex

Q5 (Create): "Design a Pydantic model for an API request that
    creates a user with password confirmation (password must match
    password_confirm)."

PASSING CODE:
```python
from pydantic import BaseModel, model_validator

class CreateUser(BaseModel):
    email: str
    password: str
    password_confirm: str

    @model_validator(mode='after')
    def passwords_match(self):
        if self.password != self.password_confirm:
            raise ValueError('Passwords do not match')
        return self
```

**THRESHOLD:** 4/5 with working code

---

## Mastery Signals vs Warning Signs

### Signals of Understanding

| Signal | Meaning |
|--------|---------|
| Explains in own words | Not just parroting |
| Offers concrete example | Can instantiate abstract |
| Identifies edge cases | Deep comprehension |
| Asks clarifying questions | Engaged thinking |
| Makes connections to prior chunks | Building knowledge graph |
| Correctly applies to novel scenario | Transfer achieved |
| Critiques with nuance | Evaluation-level thinking |

### Warning Signs (Need More Work)

| Warning | Meaning |
|---------|---------|
| Repeats definition verbatim | Recognition, not understanding |
| Right answer, can't explain why | Guessed or memorized |
| "I think maybe..." | Not confident, incomplete model |
| Fails when question is rephrased | Shallow pattern matching |
| Can't apply to new context | No transfer |
| Gives incomplete trade-off analysis | Missing depth |
| Asks to see material again | Didn't encode first time |

---

## Question Design Anti-Patterns

**Avoid These:**

| Anti-Pattern | Problem | Fix |
|--------------|---------|-----|
| "Does that make sense?" | Invites false "yes" | "Explain back to me..." |
| "Any questions?" | Burden on confused learner | Verify actively |
| Asking exact words from content | Tests recognition only | Rephrase or apply |
| Long multi-part questions | Cognitive overload | One question at a time |
| Gotcha/trick questions | Damages trust | Fair assessment |
| Only Remember-level | Surface only | Climb the ladder |
| Accept "I think so" | No verification | Require demonstration |

---

## Building a Mastery Ladder: Step by Step

1. **Identify the core concept(s)** in the chunk
2. **Start at Understand**: Can they explain it in their own words?
3. **Move to Apply**: Can they use it in a scenario?
4. **Require Analyze**: Can they compare, contrast, break down?
5. **Add Evaluate if appropriate**: Can they critique or defend?
6. **Score 80%+**: Must get 3/4 or 4/5 with solid reasoning
7. **If below threshold**: Reteach with different angle, then retry
