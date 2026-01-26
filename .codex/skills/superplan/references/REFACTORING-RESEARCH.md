# Refactoring & Rewriting Research Reference

This document provides comprehensive research on refactoring best practices, methodologies, and decision frameworks to guide the REFACTOR/REWRITE ASSESSMENT phase of superplan.

---

## Table of Contents

1. [When to Refactor vs Rewrite](#when-to-refactor-vs-rewrite)
2. [Core Refactoring Principles](#core-refactoring-principles)
3. [The Mikado Method](#the-mikado-method)
4. [Strangler Fig Pattern](#strangler-fig-pattern)
5. [Branch by Abstraction](#branch-by-abstraction)
6. [Code Smells Catalog](#code-smells-catalog)
7. [Technology-Specific Patterns](#technology-specific-patterns)
8. [Refactoring Decision Framework](#refactoring-decision-framework)

---

## When to Refactor vs Rewrite

### Decision Framework

| Factor | Favors Refactoring | Favors Rewriting |
|--------|-------------------|------------------|
| **Code Health** | Isolated trouble spots | Riddled with issues throughout |
| **Business Continuity** | Critical to operations, can't have downtime | Can tolerate migration period |
| **Resources** | Limited time/budget/people | Sufficient resources for long project |
| **Tech Stack** | Stack is current and supported | Stack is obsolete/unsupported |
| **Domain Knowledge** | Original developers available | Institutional knowledge lost |
| **Architecture** | Sound foundation, poor implementation | Fundamentally flawed architecture |
| **Technical Debt** | Manageable, can be paid incrementally | So large it impedes all progress |

### Critical Warning

> "Rewriting from scratch is the single worst strategic mistake that any software company can make." — Joel Spolsky

**Best Practice**: Start with refactoring first—it reveals hidden dependencies and business logic buried in code that would be missed in a rewrite.

### Key Questions to Ask

1. Is the codebase mostly sound with isolated trouble spots? → **Refactor**
2. Is the code nearly unmaintainable across the board? → **Consider rewrite**
3. Can the business tolerate the timeline and risk of a rewrite?
4. Do you have the domain knowledge to rebuild correctly?
5. Are you confident you understand ALL the business rules in the current code?

---

## Core Refactoring Principles

### Definition

> A refactoring is a change made to the internal structure of code that doesn't modify its observable behavior. Refactoring preserves bugs too—if you fix a bug, you are not refactoring.

### Fowler's Core Principles

1. **Small Steps**: Apply a series of small behavior-preserving transformations, each "too small to be worth doing"—the cumulative effect is significant
2. **Continuous Process**: Integrate refactoring into daily workflows, not as a separate "cleanup" phase
3. **Test-Driven**: Always have tests verifying behavior before and after each refactoring
4. **One at a Time**: Never combine a refactoring with a behavior change in the same commit

### Key Refactoring Techniques

| Technique | When to Use |
|-----------|-------------|
| **Extract Method** | Long methods, duplicated code blocks |
| **Extract Class** | Class doing too much (SRP violation) |
| **Replace Primitive with Object** | Primitive obsession, data clumps |
| **Replace Conditional with Polymorphism** | Complex switch/if chains on type |
| **Introduce Parameter Object** | Same parameters passed around together |
| **Pull Up Method** | Duplicate methods in sibling classes |
| **Replace Type Code with Subclasses** | Behavior varies by type code |
| **Inline Method** | Method body as clear as its name |
| **Move Method** | Method uses more features of another class |

### Code Quality Benefits

- **Improved Readability**: Cleaner, more organized code
- **Enhanced Maintainability**: Easier to modify and extend
- **Increased Agility**: Teams respond faster to changes
- **Reduced Bugs**: Simpler code has fewer hiding spots for defects

---

## The Mikado Method

### Overview

The Mikado Method is a technique for breaking up large refactoring tasks into smaller ones systematically, keeping the codebase in a working state throughout.

> Named after the Mikado game where you remove one stick without disturbing others.

### Core Process

```
┌─────────────────────────────────────────────────────────────────────┐
│                      MIKADO METHOD WORKFLOW                          │
├─────────────────────────────────────────────────────────────────────┤
│  1. SET GOAL         →  Define the refactoring you want to achieve  │
│  2. NAIVE ATTEMPT    →  Try to implement it directly (10 min max)   │
│  3. OBSERVE ERRORS   →  Note what breaks (tests fail, won't compile)│
│  4. VISUALIZE        →  Add prerequisites as nodes on Mikado graph  │
│  5. REVERT           →  Undo ALL changes, return to working state   │
│  6. REPEAT           →  Pick a leaf node, repeat process            │
│  7. IMPLEMENT        →  When leaf is trivial, implement & commit    │
│  8. WORK UP          →  Move up the graph until goal achieved       │
└─────────────────────────────────────────────────────────────────────┘
```

### The Mikado Graph

```
                    ┌─────────────────────┐
                    │     GOAL            │
                    │ (Main Refactoring)  │
                    └─────────┬───────────┘
                              │
              ┌───────────────┼───────────────┐
              │               │               │
              ▼               ▼               ▼
        ┌──────────┐   ┌──────────┐   ┌──────────┐
        │ Prereq A │   │ Prereq B │   │ Prereq C │
        └────┬─────┘   └────┬─────┘   └──────────┘
             │              │              (leaf)
        ┌────┴────┐    ┌────┴────┐
        │         │    │         │
        ▼         ▼    ▼         ▼
   ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐
   │Leaf A1 │ │Leaf A2 │ │Leaf B1 │ │Leaf B2 │
   │(trivial)│ │(trivial)│ │(trivial)│ │(trivial)│
   └────────┘ └────────┘ └────────┘ └────────┘
```

**Work from leaves up**: Start with trivial tasks, commit each one, and work up to the goal.

### Critical Rules

1. **10-minute timebox**: Keep exploration attempts short to avoid sunk cost fallacy
2. **Always revert**: Discard code changes after discovering prerequisites
3. **Visualize first**: The graph is more valuable than the code during exploration
4. **Commit at leaves**: Only commit when reaching a trivial, completable task
5. **Working state always**: The codebase must pass tests after every commit

### Benefits for Feature Planning

- **Safe in complex codebases**: Never breaks the build for extended periods
- **Discoverable scope**: Reveals hidden dependencies before committing to timeline
- **Parallelizable**: Once graph is built, independent leaves can be worked in parallel
- **Stoppable**: Can pause at any point with code in working state

---

## Strangler Fig Pattern

### Overview

The Strangler Fig Pattern incrementally replaces parts of a system while keeping it running, like the strangler fig tree that grows around a host tree until it replaces it completely.

### When to Use

- Migrating monolith to microservices
- Replacing a legacy system with a modern one
- Major architectural changes that can't be done all at once
- When business continuity is critical

### Implementation Steps

```
┌─────────────────────────────────────────────────────────────────────┐
│                    STRANGLER FIG PATTERN                             │
├─────────────────────────────────────────────────────────────────────┤
│  1. IDENTIFY    →  Select functionality to extract                   │
│  2. FACADE      →  Create proxy layer between old and new           │
│  3. IMPLEMENT   →  Build new implementation behind facade           │
│  4. ROUTE       →  Use feature flags to route traffic               │
│  5. VALIDATE    →  Run old and new in parallel, compare results     │
│  6. CUTOVER     →  Switch all traffic to new implementation         │
│  7. REMOVE      →  Delete old code once new is stable               │
│  8. REPEAT      →  Move to next functionality chunk                 │
└─────────────────────────────────────────────────────────────────────┘
```

### Facade Layer Architecture

```
       ┌──────────────────────┐
       │    Client Code       │
       └──────────┬───────────┘
                  │
                  ▼
       ┌──────────────────────┐
       │      FACADE          │◄─── Feature flags control routing
       │  (Proxy Layer)       │
       └──────────┬───────────┘
                  │
         ┌───────┴───────┐
         │               │
         ▼               ▼
  ┌─────────────┐ ┌─────────────┐
  │   LEGACY    │ │    NEW      │
  │   System    │ │   System    │
  │  (shrinking)│ │  (growing)  │
  └─────────────┘ └─────────────┘
```

### Considerations

| Advantage | Disadvantage |
|-----------|--------------|
| Early value delivery | Longer overall timeline |
| Reduced risk | Dual maintenance burden |
| Business continuity | Requires strong discipline |
| Parallel operation | Bridging code complexity |

---

## Branch by Abstraction

### Overview

Branch by Abstraction allows large-scale changes while releasing regularly, by introducing an abstraction layer between code and the component being replaced.

### Process

```
┌─────────────────────────────────────────────────────────────────────┐
│                    BRANCH BY ABSTRACTION                             │
├─────────────────────────────────────────────────────────────────────┤
│  1. CREATE ABSTRACTION  →  Interface capturing supplier interaction │
│  2. MIGRATE CLIENTS     →  Move clients to use abstraction layer    │
│  3. BUILD NEW SUPPLIER  →  Implement new version behind abstraction │
│  4. SWITCH GRADUALLY    →  Route clients to new implementation      │
│  5. REMOVE OLD          →  Delete legacy supplier when unused       │
│  6. OPTIONAL: REMOVE    →  Delete abstraction if no longer needed   │
└─────────────────────────────────────────────────────────────────────┘
```

### Comparison with Feature Toggles

| Branch by Abstraction | Feature Toggles |
|-----------------------|-----------------|
| Replace existing functionality | Add new functionality |
| Structural change | Behavioral toggle |
| Abstraction layer persists | Toggle removed after launch |

### Parallel Change Variant

For changing interfaces with many clients:

1. **Expand**: Add new API alongside old
2. **Migrate**: Move clients to new API
3. **Contract**: Remove old API

---

## Code Smells Catalog

### Bloaters

| Smell | Description | Refactoring |
|-------|-------------|-------------|
| **Long Method** | Method too long to understand | Extract Method |
| **Large Class** | Class doing too much | Extract Class |
| **Primitive Obsession** | Overuse of primitives | Replace Primitive with Object |
| **Long Parameter List** | Too many parameters | Introduce Parameter Object |
| **Data Clumps** | Same data passed together | Extract Class |

### Object-Orientation Abusers

| Smell | Description | Refactoring |
|-------|-------------|-------------|
| **Switch Statements** | Complex type-based conditionals | Replace Conditional with Polymorphism |
| **Parallel Inheritance** | Subclass in one hierarchy requires another | Move Method, Move Field |
| **Refused Bequest** | Subclass doesn't use inherited members | Replace Inheritance with Delegation |

### Change Preventers

| Smell | Description | Refactoring |
|-------|-------------|-------------|
| **Divergent Change** | One class changed for different reasons | Extract Class |
| **Shotgun Surgery** | One change requires many small changes | Move Method, Move Field |
| **Parallel Inheritance** | Creating subclass requires creating another | Move Method, Move Field |

### Dispensables

| Smell | Description | Refactoring |
|-------|-------------|-------------|
| **Comments** | Excessive comments masking bad code | Rename, Extract Method |
| **Duplicate Code** | Same structure in multiple places | Extract Method, Pull Up |
| **Dead Code** | Unused code | Delete it |
| **Lazy Class** | Class doing too little | Inline Class |
| **Speculative Generality** | "Just in case" abstractions | Collapse Hierarchy |

### Couplers

| Smell | Description | Refactoring |
|-------|-------------|-------------|
| **Feature Envy** | Method uses another class's data more | Move Method |
| **Inappropriate Intimacy** | Classes too intertwined | Move Method, Extract Class |
| **Message Chains** | a.getB().getC().getD() | Hide Delegate |
| **Middle Man** | Class only delegates | Remove Middle Man |

---

## Technology-Specific Patterns

### TypeScript/JavaScript

| Pattern | Use Case |
|---------|----------|
| **Extract Interface** | Reduce coupling, enable testing |
| **Replace any with Generics** | Type safety without duplication |
| **Eliminate Barrel Files** | Reduce circular dependencies |
| **Convert Class to Functions** | When OOP adds no value |
| **Type Narrowing** | Replace type assertions |

### Python

| Pattern | Use Case |
|---------|----------|
| **Replace Dict with TypedDict/dataclass** | Structure untyped data |
| **Extract Protocol** | Duck typing with type hints |
| **Replace Inheritance with Composition** | Reduce coupling |
| **Convert to Async** | I/O-bound operations |

### Go

| Pattern | Use Case |
|---------|----------|
| **Extract Interface** | Enable testing, reduce coupling |
| **Replace Empty Interface** | Type safety with generics |
| **Flatten Error Handling** | Reduce nesting |
| **Channel Consolidation** | Simplify concurrency |
| **Context Propagation** | Proper cancellation handling |

### Common Cross-Language Patterns

| Pattern | Description |
|---------|-------------|
| **Dependency Injection** | Decouple creation from use |
| **Repository Pattern** | Abstract data access |
| **Strategy Pattern** | Encapsulate algorithms |
| **Facade Pattern** | Simplify complex subsystems |

---

## Refactoring Decision Framework

### Assessment Checklist

Use this checklist when evaluating whether a feature would benefit from refactoring BEFORE implementation:

#### 1. Code Health Assessment

- [ ] Is there duplicated code in the area being modified?
- [ ] Are there long methods that need to be touched?
- [ ] Are there classes with too many responsibilities?
- [ ] Is there excessive coupling between components?
- [ ] Are there code smells blocking clean implementation?

#### 2. Architecture Assessment

- [ ] Does the current architecture support the new feature?
- [ ] Will adding this feature increase technical debt?
- [ ] Are there architectural patterns being violated?
- [ ] Would the feature benefit from a different structure?
- [ ] Is there a cleaner abstraction that would help?

#### 3. Future Roadmap Consideration

- [ ] Are there upcoming features that would benefit from refactoring now?
- [ ] Would this refactoring enable multiple future features?
- [ ] Is this area likely to change frequently?
- [ ] Would other teams benefit from this refactoring?

#### 4. Risk Assessment

- [ ] Is this area well-tested?
- [ ] Is the business logic well-understood?
- [ ] Can the refactoring be done incrementally?
- [ ] Is there a rollback strategy?

### Decision Matrix

| Scenario | Recommendation |
|----------|----------------|
| Feature adds code to already-messy area | **Refactor first** |
| Feature touches well-structured code | **Implement directly** |
| Multiple upcoming features need same area | **Refactor first** |
| One-off change to stable code | **Implement directly** |
| Team struggles to understand code | **Refactor first** |
| Tight deadline, low complexity | **Implement, refactor later** |
| Tight deadline, high complexity | **Discuss with stakeholders** |

### Interview Questions for User

When considering refactoring, probe the user with:

1. **What other features are planned for this area in the next 6 months?**
2. **How often does this part of the codebase change?**
3. **Are there pain points in this area that slow down development?**
4. **What's the risk tolerance for this refactoring?**
5. **Would you rather pay down technical debt now or accumulate more?**

---

## Sources

- [Martin Fowler's Refactoring](https://martinfowler.com/books/refactoring.html)
- [Refactoring.guru Code Smells](https://refactoring.guru/refactoring/smells)
- [The Mikado Method](https://mikadomethod.info/)
- [The Mikado Method Book](https://www.manning.com/books/the-mikado-method)
- [Understand Legacy Code: Mikado Method](https://understandlegacycode.com/blog/a-process-to-do-safe-changes-in-a-complex-codebase/)
- [Strangler Fig Pattern - Martin Fowler](https://martinfowler.com/bliki/StranglerFigApplication.html)
- [Strangler Fig Pattern - Shopify Engineering](https://shopify.engineering/refactoring-legacy-code-strangler-fig-pattern)
- [AWS Strangler Fig Pattern](https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-design-patterns/strangler-fig.html)
- [Branch by Abstraction - Martin Fowler](https://martinfowler.com/bliki/BranchByAbstraction.html)
- [Parallel Change - Martin Fowler](https://martinfowler.com/bliki/ParallelChange.html)
- [Refactor vs Rewrite - Test Double](https://testdouble.com/insights/understanding-legacy-application-rewrite-vs-refactor-tradeoffs)
