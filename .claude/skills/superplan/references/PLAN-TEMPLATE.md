# Plan Template Reference

This is the complete template for `docs/<feature-name>-plan.md` files.

Copy and adapt this template when creating new implementation plans.

---

```markdown
# [Feature Name] Implementation Plan

> Generated: [DATE]
> Status: Draft | In Review | Approved | In Progress | Complete
> Author: [Agent/Human name]
> Last Updated: [DATE]

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Technology Stack](#technology-stack)
3. [Requirements](#requirements)
4. [Research & Best Practices](#research--best-practices)
5. [Architecture](#architecture)
6. [Implementation Phases](#implementation-phases)
7. [Testing Strategy](#testing-strategy)
8. [Assumptions & Known Unknowns](#assumptions--known-unknowns)
9. [Appendix](#appendix)

---

## Executive Summary

### One-Line Summary
[Single sentence describing what this plan delivers]

### Goals
- [ ] **Primary Goal**: [Main objective]
- [ ] **Secondary Goal**: [Supporting objective]
- [ ] **Success Metric**: [How we measure success]

### Non-Goals (Explicitly Out of Scope)
- ❌ [Thing we are NOT doing]
- ❌ [Another thing we are NOT doing]

### Key Decisions Made
| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| [Decision 1] | [Why] | [Other options] |
| [Decision 2] | [Why] | [Other options] |

### Phase Overview (with Poker Estimates)

| Phase | Name | Depends On | Parallel With | Estimate | Status |
|-------|------|------------|---------------|----------|--------|
| 0 | Bootstrap (if needed) | - | - | 5 | ⬜/⏭️ |
| 1 | Setup | 0 | - | 3 | ⬜ |
| 2A | [Name] | 1 | 2B, 2C | 8 | ⬜ |
| 2B | [Name] | 1 | 2A, 2C | 5 | ⬜ |
| 2C | [Name] | 1 | 2A, 2B | 3 | ⬜ |
| 3 | Integration | 2A, 2B, 2C | - | 5 | ⬜ |

**Total Estimate**: [Sum] points

**Legend**: ⬜ Not Started | 🔄 In Progress | ✅ Complete | ⏸️ Blocked | ⏭️ Skipped

**PARALLEL EXECUTION**: Phases marked as "Parallel With" MUST be executed using parallel sub-agents.

---

## Technology Stack (Detected)

### Languages
- **Primary**: [Language] v[X.Y.Z]
- **Secondary**: [Language] (if applicable)

### Frameworks
| Framework | Version | Purpose |
|-----------|---------|---------|
| [Framework] | [X.Y.Z] | [Frontend/Backend/Testing/etc.] |

### Build Tools
| Tool | Version | Purpose |
|------|---------|---------|
| [npm/pnpm/yarn] | [X.Y.Z] | Package manager |
| [webpack/vite/etc.] | [X.Y.Z] | Bundler |

### Quality Tools Status

| Tool Type | Status | Config File | Command |
|-----------|--------|-------------|---------|
| Linter | ✅/❌ | [path or "N/A"] | [npm run lint] |
| Formatter | ✅/❌ | [path or "N/A"] | [npm run format] |
| Type Checker | ✅/❌ | [path or "N/A"] | [npm run typecheck] |
| Test Framework | ✅/❌ | [path or "N/A"] | [npm test] |

### Bootstrap Required?

- [ ] **Yes** - Missing tools detected, include Phase 0: Bootstrap
- [ ] **No** - All quality tools present, skip Phase 0

### Best Practices Research Targets

Based on detected stack, research these topics:
- [Language] [YEAR] best practices
- [Framework] [YEAR] patterns and recommendations
- [Framework] security guidelines (OWASP)
- [Language] testing best practices [YEAR]

---

## Requirements

### Original Story/Request

```
[Paste the original story, ticket, or request verbatim]
```

**Source**: [Jira-123 | Linear | GitHub Issue #X | Verbal]

### Acceptance Criteria

High-level criteria for the complete feature:

- [ ] **AC-1**: [User can do X]
- [ ] **AC-2**: [System behaves as Y when Z]
- [ ] **AC-3**: [Data is stored correctly]
- [ ] **AC-4**: [Performance meets requirements]

### Clarifications from Interview

| Question | Answer | Implication |
|----------|--------|-------------|
| [Question asked] | [Answer received] | [How it affects plan] |
| [Question asked] | [Answer received] | [How it affects plan] |

### Constraints

- **Technical**: [Must use X framework, cannot use Y]
- **Performance**: [Must respond in <Xms, handle Y req/sec]
- **Security**: [Must comply with X, cannot store Y]
- **Timeline**: [Must ship by X, blocked by Y until Z]

---

## Research & Best Practices

> Research conducted: [DATE]
> Sources: [List major sources]

### Industry Standards

#### [Topic 1: e.g., Authentication Best Practices]

**Current Recommendation (as of [DATE]):**
[Summary of best practice]

**Why It Applies Here:**
[Relevance to this feature]

**Implementation Note:**
[How we'll apply it]

#### [Topic 2]

[Same structure]

### Technology-Specific Findings

#### [Framework/Library Name]

**Version**: [X.Y.Z]
**Key Findings**:
- [Finding 1]
- [Finding 2]

**Gotchas to Avoid**:
- [Gotcha 1]
- [Gotcha 2]

### Patterns to Apply

| Pattern | Where to Use | Benefit |
|---------|--------------|---------|
| [Pattern 1] | [Location] | [Benefit] |
| [Pattern 2] | [Location] | [Benefit] |

### Anti-Patterns to Avoid

| Anti-Pattern | Why It's Bad | What to Do Instead |
|--------------|--------------|-------------------|
| [Anti-pattern 1] | [Problem] | [Solution] |
| [Anti-pattern 2] | [Problem] | [Solution] |

---

## Architecture

### System Context Diagram

Shows how the feature fits into the larger system:

```
┌─────────────────────────────────────────────────────────────────────┐
│                         SYSTEM CONTEXT                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│    ┌──────────┐                              ┌──────────────┐       │
│    │  User    │                              │  External    │       │
│    │  Browser │                              │  Service     │       │
│    └────┬─────┘                              └──────┬───────┘       │
│         │                                           │               │
│         │ HTTPS                                     │ API           │
│         ▼                                           ▼               │
│    ┌──────────────────────────────────────────────────────┐        │
│    │                  OUR APPLICATION                      │        │
│    │  ┌──────────┐    ┌──────────┐    ┌──────────┐        │        │
│    │  │ Frontend │───▶│   API    │───▶│ Database │        │        │
│    │  └──────────┘    └──────────┘    └──────────┘        │        │
│    │                       │                               │        │
│    │                       ▼                               │        │
│    │              ┌──────────────┐                         │        │
│    │              │    Cache     │                         │        │
│    │              └──────────────┘                         │        │
│    └──────────────────────────────────────────────────────┘        │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Component Diagram

Detailed view of components being built/modified:

```
┌─────────────────────────────────────────────────────────────────────┐
│                    [FEATURE NAME] COMPONENTS                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  FRONTEND                          BACKEND                          │
│  ┌────────────────────┐           ┌────────────────────┐           │
│  │  [Component 1]     │           │  [Service 1]       │           │
│  │  ├── SubComponent  │  ──────▶  │  ├── Handler       │           │
│  │  └── SubComponent  │           │  ├── Validator     │           │
│  └────────────────────┘           │  └── Repository    │           │
│                                   └────────────────────┘           │
│  ┌────────────────────┐                     │                       │
│  │  [Component 2]     │                     ▼                       │
│  │  └── SubComponent  │           ┌────────────────────┐           │
│  └────────────────────┘           │  [Database]        │           │
│                                   │  ├── Table 1       │           │
│                                   │  └── Table 2       │           │
│                                   └────────────────────┘           │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Data Flow Diagram

Shows how data moves through the system:

```
┌─────────────────────────────────────────────────────────────────────┐
│                         DATA FLOW: [OPERATION NAME]                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  1. User Action                                                     │
│     │                                                               │
│     ▼                                                               │
│  ┌──────────────────┐                                              │
│  │ Validate Input   │ ──── Invalid ───▶ Show Error                 │
│  └────────┬─────────┘                                              │
│           │ Valid                                                   │
│           ▼                                                         │
│  ┌──────────────────┐                                              │
│  │ Process Request  │                                              │
│  └────────┬─────────┘                                              │
│           │                                                         │
│      ┌────┴────┐                                                   │
│      │         │                                                    │
│   Success    Failure                                                │
│      │         │                                                    │
│      ▼         ▼                                                    │
│  ┌───────┐ ┌───────┐                                               │
│  │ Store │ │ Error │                                               │
│  │ Data  │ │ Log   │                                               │
│  └───┬───┘ └───┬───┘                                               │
│      │         │                                                    │
│      ▼         ▼                                                    │
│  ┌──────────────────┐                                              │
│  │ Return Response  │                                              │
│  └──────────────────┘                                              │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### API Design

#### Endpoints

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| POST | `/api/v1/[resource]` | Create new [resource] | Required |
| GET | `/api/v1/[resource]/:id` | Get [resource] by ID | Required |
| PUT | `/api/v1/[resource]/:id` | Update [resource] | Required |
| DELETE | `/api/v1/[resource]/:id` | Delete [resource] | Required |

#### Request/Response Schemas

**POST /api/v1/[resource]**

Request:
```json
{
  "field1": "string (required)",
  "field2": 123,
  "field3": {
    "nested": "value"
  }
}
```

Response (201 Created):
```json
{
  "id": "uuid",
  "field1": "string",
  "field2": 123,
  "createdAt": "2025-01-09T00:00:00Z"
}
```

Error Response (400 Bad Request):
```json
{
  "error": "VALIDATION_ERROR",
  "message": "field1 is required",
  "details": [
    { "field": "field1", "message": "Required" }
  ]
}
```

### Data Model

#### Database Schema Changes

```sql
-- New table
CREATE TABLE [table_name] (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  field1 VARCHAR(255) NOT NULL,
  field2 INTEGER,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_[table]_[field] ON [table_name]([field1]);

-- Foreign keys
ALTER TABLE [table_name]
  ADD CONSTRAINT fk_[table]_[other]
  FOREIGN KEY (other_id) REFERENCES other_table(id);
```

#### Migration Plan

| Step | Description | Rollback |
|------|-------------|----------|
| 1 | Create new table | Drop table |
| 2 | Add column to existing table | Drop column |
| 3 | Backfill data | No action needed |
| 4 | Add constraints | Remove constraints |

---

## Implementation Phases

### Phase Dependency Graph

```
┌─────────────────────────────────────────────────────────────────────┐
│      Phase 0: Bootstrap (CONDITIONAL - only if tools missing)      │
│         (Linter, Formatter, Type Checker, Test Framework)          │
│                          Estimate: 5 pts                            │
└──────────────────────────────┬──────────────────────────────────────┘
                               │ (skip if tools present)
                               ▼
           ┌─────────────────────────────────────────┐
           │           Phase 1: Setup                │
           │     (Database migrations, configs)      │
           │              Estimate: 3 pts            │
           └──────────────────┬──────────────────────┘
                              │
              ┌───────────────┼───────────────┐
              │               │               │  ← USE PARALLEL SUB-AGENTS
              ▼               ▼               ▼
       ┌───────────┐   ┌───────────┐   ┌───────────┐
       │ Phase 2A  │   │ Phase 2B  │   │ Phase 2C  │
       │ [Backend] │   │ [Frontend]│   │ [Tests]   │
       │  8 pts    │   │   5 pts   │   │  3 pts    │
       └─────┬─────┘   └─────┬─────┘   └─────┬─────┘
             │               │               │
             └───────────────┼───────────────┘
                             │
                             ▼
                  ┌─────────────────────┐
                  │  Phase 3: Integrate  │
                  │   (Wire up + E2E)    │
                  │     Estimate: 5 pts  │
                  └─────────────────────┘
```

---

### Phase 0: Bootstrap (CONDITIONAL)

> **Condition**: Execute ONLY if quality tools not detected in codebase
> **Depends On**: Nothing
> **Can Run With**: Nothing
> **Estimate**: 5 points
> **Status**: ⬜ Not Started | ⏭️ Skipped (tools present)

#### Objectives

- [ ] Bootstrap linter (eslint/ruff/golint based on detected language)
- [ ] Bootstrap formatter (prettier/black/gofmt based on detected language)
- [ ] Bootstrap type checker (tsc/mypy based on detected language)
- [ ] Bootstrap test framework (jest/pytest/go test based on detected language)
- [ ] Configure pre-commit hooks (optional)

#### Tasks with Estimates

| Task | Estimate | Parallel With |
|------|----------|---------------|
| Add linter config | 2 | Formatter, Type Checker |
| Add formatter config | 1 | Linter, Type Checker |
| Add type checker config | 2 | Linter, Formatter |
| Add test framework | 3 | None (after above) |
| Configure pre-commit | 1 | None (last) |

**Note**: Linter, Formatter, and Type Checker configs CAN be added in parallel using sub-agents.

#### Code Changes (Example: TypeScript/Node.js)

##### File: `.eslintrc.json` (CREATE)
```json
{
  "extends": ["eslint:recommended", "plugin:@typescript-eslint/recommended"],
  "parser": "@typescript-eslint/parser",
  "plugins": ["@typescript-eslint"],
  "root": true
}
```

##### File: `.prettierrc` (CREATE)
```json
{
  "semi": true,
  "singleQuote": true,
  "tabWidth": 2
}
```

##### File: `package.json` (MODIFY)
```diff
 "scripts": {
+  "lint": "eslint src --ext .ts,.tsx",
+  "format": "prettier --check src",
+  "format:fix": "prettier --write src",
+  "typecheck": "tsc --noEmit",
+  "test": "jest"
 }
```

#### Definition of Done (Quality Gate)
- [ ] Linter runs without configuration errors
- [ ] Formatter runs without configuration errors
- [ ] Type checker runs without configuration errors
- [ ] Test framework runs (empty suite is OK)
- [ ] All tools added to package.json scripts
- [ ] CI/CD updated to run quality checks (if applicable)

---

### Phase 1: Setup

> **Depends On**: Phase 0 (or nothing if skipped)
> **Can Run With**: Nothing
> **Estimate**: 3 points
> **Status**: ⬜ Not Started

#### Objectives

- [ ] Create database migrations
- [ ] Add configuration values
- [ ] Set up feature flags (if applicable)

#### Code Changes

##### File: `migrations/YYYYMMDD_create_[table].sql` (CREATE)

```sql
-- Migration: Create [table] table
-- Generated: [DATE]

BEGIN;

CREATE TABLE [table_name] (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  -- Add columns
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

COMMIT;
```

##### File: `migrations/YYYYMMDD_create_[table]_down.sql` (CREATE)

```sql
-- Rollback: Drop [table] table

BEGIN;
DROP TABLE IF EXISTS [table_name];
COMMIT;
```

##### File: `src/config.ts` (MODIFY)

```diff
 export const config = {
   database: process.env.DATABASE_URL,
+  [featureName]: {
+    enabled: process.env.[FEATURE]_ENABLED === 'true',
+    setting1: process.env.[FEATURE]_SETTING1 || 'default',
+  },
 };
```

#### Tests (Write First - Must Fail Initially)

##### File: `tests/migrations/[table].test.ts` (CREATE)

```typescript
describe('[Table] Migration', () => {
  it('should create [table] table with correct schema', async () => {
    const result = await db.query(`
      SELECT column_name, data_type
      FROM information_schema.columns
      WHERE table_name = '[table_name]'
    `);

    expect(result.rows).toContainEqual({
      column_name: 'id',
      data_type: 'uuid'
    });
    // Add more schema assertions
  });
});
```

#### Definition of Done (Quality Gate)

- [ ] Code passes linter
- [ ] Code passes formatter check
- [ ] Code passes type checker
- [ ] Migration runs successfully
- [ ] Rollback works correctly
- [ ] Configuration loads without errors
- [ ] All new tests pass
- [ ] All existing tests still pass
- [ ] No new warnings introduced

#### Manual Testing Instructions

```bash
# Run migration
npm run migrate

# Verify table exists
psql $DATABASE_URL -c "\\d [table_name]"

# Test rollback
npm run migrate:rollback
npm run migrate
```

---

### Phase 2A: [Backend API]

> **Depends On**: Phase 1
> **Can Run With**: Phase 2B, Phase 2C (USE PARALLEL SUB-AGENTS)
> **Estimate**: 8 points
> **Status**: ⬜ Not Started

#### Objectives

- [ ] Implement API endpoint(s)
- [ ] Add request validation
- [ ] Add business logic
- [ ] Add error handling

#### Code Changes

##### File: `src/api/[feature]/index.ts` (CREATE)

```typescript
import { Router } from 'express';
import { createHandler } from './handlers/create';
import { getHandler } from './handlers/get';
import { authenticate } from '../../middleware/auth';
import { validate } from '../../middleware/validate';
import { createSchema, getSchema } from './schemas';

export const [feature]Router = Router();

[feature]Router.post(
  '/',
  authenticate,
  validate(createSchema),
  createHandler
);

[feature]Router.get(
  '/:id',
  authenticate,
  validate(getSchema),
  getHandler
);
```

##### File: `src/api/[feature]/handlers/create.ts` (CREATE)

```typescript
import { Request, Response } from 'express';
import { [Feature]Service } from '../../../services/[feature]';

export async function createHandler(req: Request, res: Response) {
  try {
    const result = await [Feature]Service.create(req.body);
    res.status(201).json(result);
  } catch (error) {
    if (error instanceof ValidationError) {
      return res.status(400).json({
        error: 'VALIDATION_ERROR',
        message: error.message,
      });
    }
    throw error;
  }
}
```

##### File: `src/api/routes.ts` (MODIFY)

```diff
 import { userRouter } from './users';
+import { [feature]Router } from './[feature]';

 export function registerRoutes(app: Express) {
   app.use('/api/users', userRouter);
+  app.use('/api/[feature]', [feature]Router);
 }
```

#### Tests (Write First - Must Fail Initially)

##### File: `tests/unit/services/[feature].test.ts` (CREATE)

```typescript
import { [Feature]Service } from '../../../src/services/[feature]';

describe('[Feature]Service', () => {
  describe('create', () => {
    it('should create a new [feature] with valid data', async () => {
      const input = {
        field1: 'value1',
        field2: 123,
      };

      const result = await [Feature]Service.create(input);

      expect(result.id).toBeDefined();
      expect(result.field1).toBe('value1');
      expect(result.field2).toBe(123);
    });

    it('should throw ValidationError for invalid data', async () => {
      const input = {
        field1: '', // Invalid: empty string
      };

      await expect([Feature]Service.create(input))
        .rejects.toThrow('field1 is required');
    });
  });
});
```

##### File: `tests/integration/api/[feature].test.ts` (CREATE)

```typescript
import request from 'supertest';
import { app } from '../../../src/app';
import { createTestUser, getAuthToken } from '../../helpers/auth';

describe('POST /api/[feature]', () => {
  let authToken: string;

  beforeAll(async () => {
    const user = await createTestUser();
    authToken = await getAuthToken(user);
  });

  it('should return 201 when creating with valid data', async () => {
    const response = await request(app)
      .post('/api/[feature]')
      .set('Authorization', `Bearer ${authToken}`)
      .send({
        field1: 'value1',
        field2: 123,
      });

    expect(response.status).toBe(201);
    expect(response.body.id).toBeDefined();
  });

  it('should return 400 for invalid data', async () => {
    const response = await request(app)
      .post('/api/[feature]')
      .set('Authorization', `Bearer ${authToken}`)
      .send({});

    expect(response.status).toBe(400);
    expect(response.body.error).toBe('VALIDATION_ERROR');
  });

  it('should return 401 without authentication', async () => {
    const response = await request(app)
      .post('/api/[feature]')
      .send({ field1: 'value' });

    expect(response.status).toBe(401);
  });
});
```

#### Definition of Done (Quality Gate)

- [ ] Code passes linter (eslint/ruff/golint)
- [ ] Code passes formatter check (prettier/black/gofmt)
- [ ] Code passes type checker (tsc/mypy)
- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] Test coverage >= 80% for new code
- [ ] API responds correctly to valid requests
- [ ] API returns proper error codes for invalid requests
- [ ] Authentication is enforced
- [ ] All existing tests still pass
- [ ] No new warnings introduced

#### Manual Testing Instructions

```bash
# Start the server
npm run dev

# Test create endpoint
curl -X POST http://localhost:3000/api/[feature] \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"field1": "test", "field2": 123}'

# Test get endpoint
curl http://localhost:3000/api/[feature]/$ID \
  -H "Authorization: Bearer $TOKEN"

# Test error handling
curl -X POST http://localhost:3000/api/[feature] \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{}'
```

---

### Phase 2B: [Frontend UI]

> **Depends On**: Phase 1
> **Can Run With**: Phase 2A, Phase 2C (USE PARALLEL SUB-AGENTS)
> **Estimate**: 5 points
> **Status**: ⬜ Not Started

#### Objectives

- [ ] Create UI components
- [ ] Implement form handling
- [ ] Add loading/error states
- [ ] Connect to API (mock for now if 1A not done)

#### Code Changes

##### File: `src/components/[Feature]/index.tsx` (CREATE)

```tsx
import React, { useState } from 'react';
import { [Feature]Form } from './[Feature]Form';
import { [Feature]List } from './[Feature]List';
import { use[Feature] } from '../../hooks/use[Feature]';

export function [Feature]Page() {
  const { items, isLoading, error, create } = use[Feature]();

  if (isLoading) return <LoadingSpinner />;
  if (error) return <ErrorMessage error={error} />;

  return (
    <div className="[feature]-page">
      <h1>[Feature]</h1>
      <[Feature]Form onSubmit={create} />
      <[Feature]List items={items} />
    </div>
  );
}
```

##### File: `src/components/[Feature]/[Feature]Form.tsx` (CREATE)

```tsx
import React, { useState } from 'react';

interface [Feature]FormProps {
  onSubmit: (data: [Feature]Input) => Promise<void>;
}

export function [Feature]Form({ onSubmit }: [Feature]FormProps) {
  const [field1, setField1] = useState('');
  const [field2, setField2] = useState(0);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    setError(null);

    try {
      await onSubmit({ field1, field2 });
      setField1('');
      setField2(0);
    } catch (err) {
      setError(err.message);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} data-testid="[feature]-form">
      {error && <div className="error">{error}</div>}

      <label>
        Field 1
        <input
          type="text"
          value={field1}
          onChange={(e) => setField1(e.target.value)}
          data-testid="field1-input"
          required
        />
      </label>

      <label>
        Field 2
        <input
          type="number"
          value={field2}
          onChange={(e) => setField2(Number(e.target.value))}
          data-testid="field2-input"
        />
      </label>

      <button type="submit" disabled={isSubmitting}>
        {isSubmitting ? 'Saving...' : 'Save'}
      </button>
    </form>
  );
}
```

#### Tests (Write First - Must Fail Initially)

##### File: `tests/components/[Feature]/[Feature]Form.test.tsx` (CREATE)

```tsx
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { [Feature]Form } from '../../../src/components/[Feature]/[Feature]Form';

describe('[Feature]Form', () => {
  it('should render form fields', () => {
    render(<[Feature]Form onSubmit={jest.fn()} />);

    expect(screen.getByTestId('field1-input')).toBeInTheDocument();
    expect(screen.getByTestId('field2-input')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /save/i })).toBeInTheDocument();
  });

  it('should call onSubmit with form data', async () => {
    const onSubmit = jest.fn().mockResolvedValue(undefined);
    render(<[Feature]Form onSubmit={onSubmit} />);

    await userEvent.type(screen.getByTestId('field1-input'), 'test value');
    await userEvent.type(screen.getByTestId('field2-input'), '42');
    await userEvent.click(screen.getByRole('button', { name: /save/i }));

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalledWith({
        field1: 'test value',
        field2: 42,
      });
    });
  });

  it('should show loading state while submitting', async () => {
    const onSubmit = jest.fn().mockImplementation(
      () => new Promise(resolve => setTimeout(resolve, 100))
    );
    render(<[Feature]Form onSubmit={onSubmit} />);

    await userEvent.type(screen.getByTestId('field1-input'), 'test');
    await userEvent.click(screen.getByRole('button', { name: /save/i }));

    expect(screen.getByRole('button')).toHaveTextContent('Saving...');
    expect(screen.getByRole('button')).toBeDisabled();
  });

  it('should display error message on failure', async () => {
    const onSubmit = jest.fn().mockRejectedValue(new Error('Save failed'));
    render(<[Feature]Form onSubmit={onSubmit} />);

    await userEvent.type(screen.getByTestId('field1-input'), 'test');
    await userEvent.click(screen.getByRole('button', { name: /save/i }));

    await waitFor(() => {
      expect(screen.getByText('Save failed')).toBeInTheDocument();
    });
  });
});
```

#### Definition of Done (Quality Gate)

- [ ] Code passes linter (eslint)
- [ ] Code passes formatter check (prettier)
- [ ] Code passes type checker (tsc)
- [ ] All component tests pass
- [ ] Test coverage >= 80% for new code
- [ ] Form renders correctly
- [ ] Form validation works
- [ ] Loading states display correctly
- [ ] Error states display correctly
- [ ] Form resets after successful submission
- [ ] All existing tests still pass
- [ ] No new warnings introduced

#### Manual Testing Instructions

1. Navigate to `http://localhost:3000/[feature]`
2. Verify form displays correctly
3. Submit form with valid data - should succeed
4. Submit form with invalid data - should show errors
5. Verify loading state during submission
6. Verify form resets after success

---

### Phase 2C: [Additional Tests / Edge Cases]

> **Depends On**: Phase 1
> **Can Run With**: Phase 2A, Phase 2B (USE PARALLEL SUB-AGENTS)
> **Estimate**: 3 points
> **Status**: ⬜ Not Started

#### Objectives

- [ ] Write edge case tests
- [ ] Write performance tests (if applicable)
- [ ] Write security tests (if applicable)
- [ ] Create test fixtures/factories

#### Definition of Done (Quality Gate)

- [ ] Code passes linter
- [ ] Code passes formatter check
- [ ] Code passes type checker
- [ ] All edge case tests pass
- [ ] Test coverage >= 80% for new code
- [ ] All existing tests still pass
- [ ] No new warnings introduced

---

### Phase 3: Integration

> **Depends On**: Phase 2A, Phase 2B, Phase 2C
> **Can Run With**: Nothing
> **Estimate**: 5 points
> **Status**: ⬜ Not Started

#### Objectives

- [ ] Wire frontend to real backend
- [ ] Add E2E tests
- [ ] Performance testing
- [ ] Final integration verification

#### E2E Tests (Write First - Must Fail Initially)

##### File: `tests/e2e/[feature].spec.ts` (CREATE)

```typescript
import { test, expect } from '@playwright/test';

test.describe('[Feature] E2E', () => {
  test.beforeEach(async ({ page }) => {
    // Login and navigate
    await page.goto('/login');
    await page.fill('[data-testid="email"]', 'test@example.com');
    await page.fill('[data-testid="password"]', 'password123');
    await page.click('button[type="submit"]');
    await page.waitForURL('/dashboard');
  });

  test('should create new [feature]', async ({ page }) => {
    await page.goto('/[feature]');

    await page.fill('[data-testid="field1-input"]', 'E2E Test');
    await page.fill('[data-testid="field2-input"]', '99');
    await page.click('button:has-text("Save")');

    // Verify success
    await expect(page.locator('.success-message')).toBeVisible();
    await expect(page.locator('[data-testid="[feature]-list"]'))
      .toContainText('E2E Test');
  });

  test('should handle errors gracefully', async ({ page }) => {
    await page.goto('/[feature]');

    // Submit empty form
    await page.click('button:has-text("Save")');

    // Verify error
    await expect(page.locator('.error')).toBeVisible();
  });
});
```

#### Definition of Done (Quality Gate)

- [ ] Code passes all linters
- [ ] Code passes all formatter checks
- [ ] Code passes all type checkers
- [ ] All E2E tests pass
- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] Test coverage >= 80% overall for new code
- [ ] Frontend successfully communicates with backend
- [ ] Feature works as expected in realistic scenarios
- [ ] Performance meets requirements
- [ ] All previous phase tests still pass
- [ ] No regressions in existing functionality
- [ ] No new warnings introduced

#### Manual Testing Instructions

Complete user journey test:

1. Start with clean database
2. Create user account
3. Login
4. Navigate to [feature]
5. Create new [item]
6. Verify [item] appears in list
7. Edit [item]
8. Delete [item]
9. Verify [item] removed
10. Logout

---

## Testing Strategy

### Testing Pyramid

```
                    /\
                   /  \         E2E Tests
                  /    \        - 2-3 critical journeys
                 /──────\       - Run before deploy
                /        \
               /──────────\     Integration Tests
              /            \    - API contracts
             /              \   - Database operations
            /────────────────\  - Run on every PR
           /                  \
          /────────────────────\ Unit Tests
         /                      \ - Business logic
        /                        \ - Utilities
       /                          \- Run on every commit
      /────────────────────────────\
```

### Test Commands

```bash
# Run all tests
npm test

# Run unit tests only
npm run test:unit

# Run integration tests only
npm run test:integration

# Run E2E tests
npm run test:e2e

# Run tests in watch mode
npm run test:watch

# Run tests with coverage
npm run test:coverage
```

### Coverage Requirements

| Type | Minimum | Target |
|------|---------|--------|
| Unit | 80% | 90% |
| Integration | 70% | 80% |
| E2E | Critical paths | Critical paths |

---

## Assumptions & Known Unknowns

### Assumptions

| # | Assumption | Risk if Wrong | Mitigation | Validated |
|---|------------|---------------|------------|-----------|
| 1 | [Assumption 1] | [Risk] | [Mitigation] | ⬜ |
| 2 | [Assumption 2] | [Risk] | [Mitigation] | ⬜ |
| 3 | [Assumption 3] | [Risk] | [Mitigation] | ⬜ |

### Known Unknowns

| # | Unknown | Impact | How to Resolve | Resolved |
|---|---------|--------|----------------|----------|
| 1 | [Unknown 1] | [Impact] | [Resolution plan] | ⬜ |
| 2 | [Unknown 2] | [Impact] | [Resolution plan] | ⬜ |

### Open Questions

- [ ] [Question 1] - Owner: @[name], Due: [date]
- [ ] [Question 2] - Owner: @[name], Due: [date]

### Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| [Risk 1] | Low/Med/High | Low/Med/High | [Mitigation] |
| [Risk 2] | Low/Med/High | Low/Med/High | [Mitigation] |

---

## Appendix

### A. Research Sources

1. [Source 1](url) - [Brief description]
2. [Source 2](url) - [Brief description]
3. [Source 3](url) - [Brief description]

### B. Codebase Analysis

#### Existing Patterns Identified

| Pattern | Location | Notes |
|---------|----------|-------|
| [Pattern 1] | `src/path/file.ts` | [Notes] |
| [Pattern 2] | `src/path/file.ts` | [Notes] |

#### Files That Will Be Modified

| File | Changes |
|------|---------|
| `src/path/file1.ts` | Add [feature] import |
| `src/path/file2.ts` | Register [feature] route |

### C. Glossary

| Term | Definition |
|------|------------|
| [Term 1] | [Definition] |
| [Term 2] | [Definition] |

### D. Version History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | [DATE] | [Author] | Initial plan |
| 1.1 | [DATE] | [Author] | [Changes] |

---

## Sign-Off

| Role | Name | Status | Date |
|------|------|--------|------|
| Author | [Name] | ✅ Complete | [Date] |
| Tech Lead | [Name] | ⬜ Pending | |
| Product | [Name] | ⬜ Pending | |
```

---

## Chunked Writing Guide

For large plans, write in chunks to avoid context issues.

### When to Use Multiple Files

Plans exceeding ~20,000 tokens (~4,000 lines) should be split into multiple files to avoid read errors:
- `<feature>-plan-1.md`: Overview, Requirements, Research, Architecture
- `<feature>-plan-2.md`: Phase 0 and Phase 1 (parallelizable phases)
- `<feature>-plan-N.md`: Remaining phases, Testing, Assumptions, Appendix

### Multi-File Header

Each file in a split plan should include:

```markdown
# [Feature Name] Implementation Plan - Part [N] of [Total]

> **Plan Set**: `docs/<feature>-plan-*.md`
> **This File**: Part [N] - [Section Names]
> **Navigation**: [Part 1](feature-plan-1.md) | [Part 2](feature-plan-2.md) | ...
>
> Generated: [DATE]
> Status: Draft | In Review | Approved | In Progress | Complete

## Contents in This File
- [Section 1](#section-1)
- [Section 2](#section-2)
```

### Single File: Chunk 1 (First Message)
- Executive Summary
- Requirements
- Save file

### Single File: Chunk 2 (Second Message)
- Research & Best Practices
- Architecture diagrams
- Save file

### Single File: Chunk 3-N (One Per Message)
- Write one phase completely
- Save file after each phase

### Single File: Final Chunk
- Testing Strategy
- Assumptions & Unknowns
- Appendix
- Save file

### Resuming from Checkpoint

When resuming:
1. Read the existing plan file(s)
2. Find the last completed section
3. Continue from there
4. Reference the existing content to maintain consistency
5. For multi-file plans, ensure navigation links are updated
