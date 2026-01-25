# Testing Pyramid & TDD Strategy

This guide covers the testing approach for Superplan, emphasizing Test-Driven Development (TDD), the testing pyramid, and writing durable (not brittle) tests.

---

## The Testing Pyramid

```
                        /\
                       /  \         E2E Tests (Few)
                      /    \        ~5% of tests
                     /  E2E \       Critical user journeys only
                    /────────\
                   /          \     Integration Tests (Some)
                  /            \    ~15% of tests
                 / Integration  \   API contracts, database ops
                /────────────────\
               /                  \  Unit Tests (Many)
              /                    \ ~80% of tests
             /       Unit           \ Business logic, utilities
            /──────────────────────── \
```

### Why This Pyramid Shape?

| Layer | Speed | Reliability | Maintenance | Coverage |
|-------|-------|-------------|-------------|----------|
| Unit | Fast (ms) | Very Stable | Low | High |
| Integration | Medium (s) | Stable | Medium | Medium |
| E2E | Slow (min) | Flaky | High | Low |

**Key Principle**: Fast, reliable tests at the bottom; slow, comprehensive tests at the top.

---

## Test-Driven Development (TDD) in Superplan

### The Red-Green-Refactor Cycle

For each feature/phase:

```
┌─────────────────────────────────────────────────────────────────────┐
│                         TDD CYCLE                                   │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   1. RED         2. GREEN         3. REFACTOR                       │
│   ┌───────┐      ┌───────┐        ┌───────┐                        │
│   │ Write │      │ Write │        │ Clean │                        │
│   │Failing│ ───▶ │Minimal│  ───▶  │  Up   │                        │
│   │ Test  │      │ Code  │        │ Code  │                        │
│   └───────┘      └───────┘        └───────┘                        │
│       │              │                │                             │
│       ▼              ▼                ▼                             │
│   Test FAILS     Test PASSES     Tests STILL PASS                   │
│                                                                     │
│   ◀──────────────────────────────────────────────────────────────  │
│                        Repeat for next behavior                     │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### TDD Rules for Superplan

1. **Tests First**: Write tests BEFORE implementation code
2. **Failing First**: Confirm tests fail before writing implementation
3. **Minimal Code**: Write only enough code to make tests pass
4. **All Green**: Ensure ALL tests pass (not just new ones)
5. **Then Refactor**: Clean up code while keeping tests green

### Phase-Level TDD Workflow

For each implementation phase:

```markdown
## Phase X: [Name]

### Step 1: Write Failing Tests
1. Create test file
2. Write test cases for expected behavior
3. Run tests - they should FAIL
4. Document: "Tests written, confirmed failing"

### Step 2: Implement
1. Write implementation code
2. Run tests frequently
3. Stop when all tests pass
4. Document: "Implementation complete, tests passing"

### Step 3: Verify No Regressions
1. Run full test suite
2. Confirm no existing tests broke
3. Document: "All tests passing, no regressions"
```

---

## Durable vs Brittle Tests

### What Makes a Test Brittle?

Brittle tests break when implementation changes, even if behavior is correct.

```typescript
// ❌ BRITTLE: Tests implementation details
describe('UserService', () => {
  it('should call hashPassword with bcrypt', async () => {
    const bcryptSpy = jest.spyOn(bcrypt, 'hash');
    await userService.createUser({ email: 'test@test.com', password: 'pass' });
    expect(bcryptSpy).toHaveBeenCalledWith('pass', 10);
  });

  it('should store user in users array', async () => {
    await userService.createUser({ email: 'test@test.com', password: 'pass' });
    expect(userService._users).toHaveLength(1); // Testing private state
  });

  it('should call database.insert with correct parameters', async () => {
    const insertSpy = jest.spyOn(database, 'insert');
    await userService.createUser({ email: 'test@test.com', password: 'pass' });
    expect(insertSpy).toHaveBeenCalledWith('users', expect.any(Object));
  });
});
```

**Why these are brittle:**
- Testing that bcrypt is used (what if we switch to argon2?)
- Testing internal data structures
- Testing database method names

### What Makes a Test Durable?

Durable tests verify behavior/outcomes, not implementation.

```typescript
// ✅ DURABLE: Tests behavior and outcomes
describe('UserService', () => {
  it('should create a user that can be retrieved', async () => {
    const created = await userService.createUser({
      email: 'test@test.com',
      password: 'securePassword123'
    });

    const retrieved = await userService.getUser(created.id);
    expect(retrieved.email).toBe('test@test.com');
  });

  it('should not store plain text passwords', async () => {
    const created = await userService.createUser({
      email: 'test@test.com',
      password: 'securePassword123'
    });

    // Password should not be retrievable in plain text
    expect(created.password).toBeUndefined();
    // Or if stored, should not match plain text
    const stored = await database.getById('users', created.id);
    expect(stored.passwordHash).not.toBe('securePassword123');
  });

  it('should authenticate user with correct password', async () => {
    await userService.createUser({
      email: 'test@test.com',
      password: 'securePassword123'
    });

    const result = await userService.authenticate('test@test.com', 'securePassword123');
    expect(result.success).toBe(true);
  });

  it('should reject authentication with wrong password', async () => {
    await userService.createUser({
      email: 'test@test.com',
      password: 'securePassword123'
    });

    const result = await userService.authenticate('test@test.com', 'wrongPassword');
    expect(result.success).toBe(false);
  });
});
```

**Why these are durable:**
- Test that users can be created and retrieved (behavior)
- Test that passwords aren't stored in plain text (security requirement)
- Test authentication works (core business logic)
- Don't care HOW it's implemented

### Durable Test Checklist

When writing tests, ask:

- [ ] **Am I testing behavior or implementation?**
  - ✅ "User can log in" (behavior)
  - ❌ "Login calls bcrypt.compare" (implementation)

- [ ] **Would this test break if I refactored the code without changing behavior?**
  - ✅ No - durable
  - ❌ Yes - brittle

- [ ] **Am I testing public API or internal state?**
  - ✅ Public methods and their outputs
  - ❌ Private variables, internal method calls

- [ ] **Am I testing the contract or the mechanism?**
  - ✅ "Returns user with ID" (contract)
  - ❌ "Calls database.insert then database.select" (mechanism)

- [ ] **Does this test provide confidence the feature works?**
  - ✅ Verifies core user-facing behavior
  - ❌ Verifies internal plumbing

### Common Brittle Test Patterns to Avoid

| Pattern | Why It's Brittle | Durable Alternative |
|---------|------------------|---------------------|
| Mocking every dependency | Breaks when dependencies change | Test through public API |
| Testing method call order | Breaks when order changes | Test final outcome |
| Snapshot testing everything | Breaks on any UI change | Test specific behaviors |
| Testing private methods | Breaks when implementation changes | Test through public methods |
| Asserting exact error messages | Breaks when copy changes | Assert error type/category |
| Testing CSS classes | Breaks when styles change | Test accessibility/behavior |

---

## Unit Tests

### Purpose
Test individual functions, classes, and modules in isolation.

### Characteristics
- Run in milliseconds
- No external dependencies (database, network, filesystem)
- Mock external dependencies
- Test one thing at a time

### What to Test
- Business logic
- Data transformations
- Validation rules
- Edge cases
- Error handling

### Example: Business Logic

```typescript
// Function being tested
function calculateDiscount(cart: Cart): number {
  const subtotal = cart.items.reduce((sum, item) => sum + item.price, 0);

  if (subtotal >= 100) return subtotal * 0.10; // 10% off
  if (subtotal >= 50) return subtotal * 0.05;  // 5% off
  return 0;
}

// Tests
describe('calculateDiscount', () => {
  it('should return 0 for orders under $50', () => {
    const cart = { items: [{ price: 30 }] };
    expect(calculateDiscount(cart)).toBe(0);
  });

  it('should return 5% for orders $50-$99', () => {
    const cart = { items: [{ price: 75 }] };
    expect(calculateDiscount(cart)).toBe(3.75);
  });

  it('should return 10% for orders $100+', () => {
    const cart = { items: [{ price: 150 }] };
    expect(calculateDiscount(cart)).toBe(15);
  });

  it('should handle empty cart', () => {
    const cart = { items: [] };
    expect(calculateDiscount(cart)).toBe(0);
  });

  it('should handle boundary at exactly $50', () => {
    const cart = { items: [{ price: 50 }] };
    expect(calculateDiscount(cart)).toBe(2.50);
  });

  it('should handle boundary at exactly $100', () => {
    const cart = { items: [{ price: 100 }] };
    expect(calculateDiscount(cart)).toBe(10);
  });
});
```

### Mocking Guidelines

**Mock external systems, not internal logic:**

```typescript
// ✅ GOOD: Mock external API
jest.mock('../services/paymentGateway', () => ({
  processPayment: jest.fn().mockResolvedValue({ success: true })
}));

// ❌ BAD: Mock internal helper
jest.mock('../utils/calculateTax', () => ({
  calculateTax: jest.fn().mockReturnValue(10)
}));
// Instead, just let calculateTax run - it's fast and deterministic
```

---

## Integration Tests

### Purpose
Test how components work together (API + database, service + service).

### Characteristics
- Run in seconds
- Use real database (test database)
- Test API contracts
- Test data persistence

### What to Test
- API endpoint behavior
- Database operations (CRUD)
- Service interactions
- Authentication/authorization
- Error responses

### Example: API Integration Test

```typescript
import request from 'supertest';
import { app } from '../src/app';
import { db } from '../src/database';

describe('POST /api/orders', () => {
  beforeEach(async () => {
    await db.query('DELETE FROM orders');
    await db.query('DELETE FROM users');
    // Seed test user
    await db.query(`INSERT INTO users (id, email) VALUES ('user-1', 'test@test.com')`);
  });

  afterAll(async () => {
    await db.end();
  });

  it('should create order and persist to database', async () => {
    const response = await request(app)
      .post('/api/orders')
      .set('Authorization', 'Bearer valid-token')
      .send({
        items: [{ productId: 'prod-1', quantity: 2 }]
      });

    expect(response.status).toBe(201);
    expect(response.body.id).toBeDefined();

    // Verify persisted
    const result = await db.query('SELECT * FROM orders WHERE id = $1', [response.body.id]);
    expect(result.rows).toHaveLength(1);
  });

  it('should return 400 for invalid order data', async () => {
    const response = await request(app)
      .post('/api/orders')
      .set('Authorization', 'Bearer valid-token')
      .send({
        items: [] // Invalid: empty
      });

    expect(response.status).toBe(400);
    expect(response.body.error).toBe('VALIDATION_ERROR');
  });

  it('should return 401 without authentication', async () => {
    const response = await request(app)
      .post('/api/orders')
      .send({ items: [{ productId: 'prod-1', quantity: 1 }] });

    expect(response.status).toBe(401);
  });
});
```

### Database Testing Best Practices

1. **Use a test database**: Never test against production
2. **Clean state**: Reset database before each test
3. **Transactions**: Wrap tests in transactions, rollback after
4. **Seed data**: Create minimal required data for each test
5. **Parallel safety**: Tests shouldn't interfere with each other

```typescript
// Transaction-based cleanup
describe('OrderService', () => {
  let transaction: Transaction;

  beforeEach(async () => {
    transaction = await db.beginTransaction();
  });

  afterEach(async () => {
    await transaction.rollback();
  });

  it('should create order', async () => {
    // Test runs within transaction
    // Rollback cleans up automatically
  });
});
```

---

## E2E Tests

### Purpose
Test complete user journeys through the real application.

### Characteristics
- Run in minutes
- Use real browser (Playwright, Cypress)
- Test user-facing behavior
- Most realistic but slowest

### What to Test (Be Selective!)

Only test **critical user journeys**:
- Sign up and login
- Core feature happy paths
- Payment/checkout flows
- Data export/import

**Don't test:**
- Every edge case (use unit tests)
- Every form validation (use integration tests)
- Every UI variation

### Example: E2E Test

```typescript
import { test, expect } from '@playwright/test';

test.describe('Order Creation Journey', () => {
  test('user can create and view order', async ({ page }) => {
    // 1. Login
    await page.goto('/login');
    await page.fill('[data-testid="email"]', 'test@test.com');
    await page.fill('[data-testid="password"]', 'password123');
    await page.click('[data-testid="login-button"]');
    await page.waitForURL('/dashboard');

    // 2. Add item to cart
    await page.goto('/products');
    await page.click('[data-testid="add-to-cart-prod-1"]');
    await expect(page.locator('[data-testid="cart-count"]')).toHaveText('1');

    // 3. Checkout
    await page.click('[data-testid="checkout-button"]');
    await page.waitForURL('/checkout');
    await page.fill('[data-testid="address"]', '123 Test St');
    await page.click('[data-testid="place-order"]');

    // 4. Verify order created
    await page.waitForURL(/\/orders\/.+/);
    await expect(page.locator('[data-testid="order-status"]')).toHaveText('Confirmed');

    // 5. Verify in order history
    await page.goto('/orders');
    await expect(page.locator('[data-testid="order-list"]')).toContainText('Confirmed');
  });
});
```

### E2E Best Practices

1. **Use data-testid**: More stable than CSS selectors
2. **Wait properly**: Use `waitForURL`, `waitForSelector`, not arbitrary delays
3. **Independent tests**: Each test should be runnable in isolation
4. **Clean data**: Reset test data before each test
5. **Retry flaky tests**: Configure automatic retries for network issues

---

## Test Strategy by Feature Type

### API Feature

```
Tests to write:
├── Unit (80%)
│   ├── Request validation
│   ├── Business logic
│   ├── Response formatting
│   └── Error handling
├── Integration (15%)
│   ├── API contract tests
│   ├── Database operations
│   └── Authentication
└── E2E (5%)
    └── Critical API journey (if UI exists)
```

### UI Feature

```
Tests to write:
├── Unit (80%)
│   ├── Component rendering
│   ├── Event handlers
│   ├── State management
│   └── Utility functions
├── Integration (15%)
│   ├── API integration
│   ├── Form submission
│   └── Navigation
└── E2E (5%)
    └── Critical user journey
```

### Database Migration

```
Tests to write:
├── Unit (70%)
│   └── Migration logic (if complex)
├── Integration (30%)
│   ├── Migration runs successfully
│   ├── Rollback works
│   └── Data integrity maintained
└── E2E (0%)
    └── Not applicable
```

---

## Acceptance Criteria Template

For each phase, define acceptance criteria tied to tests:

```markdown
## Phase X Acceptance Criteria

### Automated Tests
- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] No regressions in existing tests
- [ ] Code coverage meets minimum (e.g., 80%)

### Manual Verification
- [ ] [Specific manual test 1]
- [ ] [Specific manual test 2]

### Performance (if applicable)
- [ ] Response time < Xms
- [ ] Can handle Y concurrent users

### Definition of Done
This phase is complete when:
1. All automated tests pass
2. All manual verification complete
3. Code reviewed and approved
4. No known bugs
```

---

## Running Tests in Superplan Phases

### Before Starting Implementation

```bash
# Write tests first
# Then run to confirm they fail
npm test -- --testNamePattern="[FeatureName]"
# Expected: Tests should FAIL (red phase)
```

### During Implementation

```bash
# Run frequently to check progress
npm test -- --watch
# Watch mode reruns on file changes
```

### After Implementation

```bash
# Run full suite to check for regressions
npm test

# Run with coverage
npm test -- --coverage

# Verify no regressions
git diff --name-only | grep test # Should only show new test files
```

### Before Phase Completion

```bash
# Full verification
npm test                    # All tests pass
npm run lint               # No lint errors
npm run type-check         # No type errors
npm run test:integration   # Integration tests pass
npm run test:e2e           # E2E tests pass (if applicable)
```

---

## Test File Organization

```
project/
├── src/
│   ├── services/
│   │   └── orders.ts
│   ├── api/
│   │   └── orders.ts
│   └── components/
│       └── OrderForm.tsx
└── tests/
    ├── unit/
    │   ├── services/
    │   │   └── orders.test.ts
    │   └── components/
    │       └── OrderForm.test.tsx
    ├── integration/
    │   └── api/
    │       └── orders.test.ts
    └── e2e/
        └── orders.spec.ts
```

**Alternative (collocated):**

```
project/
└── src/
    ├── services/
    │   ├── orders.ts
    │   └── orders.test.ts      # Unit test next to source
    ├── api/
    │   ├── orders.ts
    │   └── orders.integration.test.ts
    └── components/
        ├── OrderForm.tsx
        └── OrderForm.test.tsx
```

Choose based on existing project conventions.

---

## Summary: TDD in Superplan

1. **Write tests first** - Every phase starts with failing tests
2. **Follow the pyramid** - Many unit, some integration, few E2E
3. **Make tests durable** - Test behavior, not implementation
4. **All green before done** - Phase isn't complete until all tests pass
5. **No regressions** - Existing tests must still pass
6. **Document test commands** - Include how to run tests in each phase
