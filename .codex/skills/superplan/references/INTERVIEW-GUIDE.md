# Interview Guide for Feature Planning

This guide provides comprehensive question templates for gathering requirements during the Superplan interview phase.

## Interview Philosophy

1. **Ask only relevant questions** - Don't ask about databases if it's a UI-only feature
2. **Batch questions** - Ask 3-5 questions at a time, not 20
3. **Follow up** - Use answers to inform deeper questions
4. **Capture uncertainty** - "I don't know" is a valid answer that becomes a Known Unknown

---

## Question Bank by Category

### 1. Scope & Boundaries

**Essential Questions:**
- What is the minimum viable version of this feature?
- What is explicitly OUT of scope for this work?
- Are there related features we should NOT modify?

**Deeper Questions:**
- Is this a standalone feature or part of a larger epic?
- What future enhancements are anticipated (so we can design for them without building them)?
- Are there feature flags or gradual rollout requirements?

**Red Flags to Probe:**
- Vague scope ("make it good")
- Scope creep signals ("while we're at it...")
- No clear MVP definition

---

### 2. User Experience

**Essential Questions:**
- Who are the primary users of this feature?
- What is the expected user journey/flow?
- Are there existing designs, mockups, or wireframes?

**Deeper Questions:**
- Are there accessibility requirements (WCAG level)?
- What devices/browsers must be supported?
- Are there internationalization (i18n) requirements?
- What happens on error states? Loading states?
- Are there analytics/tracking requirements?

**Red Flags to Probe:**
- No clear user persona
- Missing error state designs
- Assumptions about user technical ability

---

### 3. Technical Constraints

**Essential Questions:**
- Are there specific technologies we must use or avoid?
- Are there performance requirements (response time, throughput)?
- Are there security or compliance requirements?

**Deeper Questions:**
- What's the expected load/scale?
- Are there rate limiting requirements?
- Must this work offline or in degraded network conditions?
- Are there specific hosting/infrastructure constraints?
- What monitoring/alerting is needed?

**Red Flags to Probe:**
- Unstated performance expectations
- Security requirements not documented
- Infrastructure assumptions

---

### 4. Data & State

**Essential Questions:**
- What data does this feature need to read?
- What data does this feature create or modify?
- Where does the data come from and where does it go?

**Deeper Questions:**
- Are there data migration requirements?
- What's the data retention policy?
- Are there privacy/GDPR considerations?
- Is there PII (personally identifiable information)?
- What happens to data when feature is disabled/removed?

**Red Flags to Probe:**
- Unclear data ownership
- Missing data flow documentation
- No consideration for data lifecycle

---

### 5. Integration Points

**Essential Questions:**
- What existing systems does this integrate with?
- Are there external APIs or services involved?
- What are the contracts/interfaces we must maintain?

**Deeper Questions:**
- What happens if an integration is unavailable?
- Are there rate limits on external services?
- Who owns the external systems?
- What's the SLA for integrations?
- Are there webhook/event requirements?

**Red Flags to Probe:**
- Undocumented integrations
- Assumptions about external system behavior
- Missing error handling for integration failures

---

### 6. Testing & Validation

**Essential Questions:**
- How will we know this feature is working correctly?
- Are there specific test scenarios that must pass?
- Who needs to sign off before release?

**Deeper Questions:**
- Are there existing test suites we should align with?
- Is there a staging/QA environment?
- Are there load testing requirements?
- What's the rollback plan if issues are found?
- Are there A/B testing requirements?

**Red Flags to Probe:**
- No acceptance criteria
- Unclear sign-off process
- No regression test strategy

---

### 7. Dependencies & Sequencing

**Essential Questions:**
- Does this depend on other work being completed first?
- Is other work blocked waiting for this?
- Are there team/resource constraints?

**Deeper Questions:**
- Are there external team dependencies?
- Is there a hard deadline? Why?
- What's the release vehicle (feature flag, version, etc.)?
- Are there documentation requirements?

**Red Flags to Probe:**
- Circular dependencies
- Unrealistic timelines
- Missing stakeholder alignment

---

### 8. Quality Gates & Tooling

**Essential Questions:**
- What linter, formatter, and type checker does this project use?
- What test framework is in use, and what's the coverage threshold?
- Are there CI/CD quality gates that must pass?

**Deeper Questions:**
- Are there pre-commit hooks configured?
- What's the required test coverage percentage?
- Are there code review requirements?
- Is there a staging environment for QA?
- Are there performance benchmarks to maintain?

**Red Flags to Probe:**
- No linter or formatter configured
- No type checking enabled
- No test framework set up
- No CI/CD pipeline

**Bootstrap Trigger:**
If any of these are missing, plan must include **Phase 0: Bootstrap** to set them up before implementation.

---

## Question Selection Strategy

### For Small Features (1-2 days)

Ask only from:
- Scope (MVP question)
- Technical (constraints question)
- Testing (how to validate question)
- Quality Gates (existing tooling)

**Example:**
```
I have a few quick questions before planning:

1. **Scope**: Is this the complete feature, or is it part of something larger?
2. **Technical**: Any specific libraries or patterns I should use/avoid?
3. **Validation**: How should we verify this is working correctly?
4. **Quality**: What quality tools (linter/formatter/tests) are already set up?
```

### For Medium Features (1-2 weeks)

Ask from:
- Scope (MVP + out of scope)
- UX (user journey + designs)
- Technical (constraints + performance)
- Data (what data + where from)
- Testing (validation + sign-off)
- Quality Gates (tooling + coverage requirements)

**Example:**
```
Before I create the implementation plan, I need to understand:

1. **Scope**: What's the MVP, and what's explicitly out of scope?
2. **UX**: Do you have designs, or should I propose the user flow?
3. **Data**: What data does this feature need, and where does it come from?
4. **Performance**: Are there specific performance requirements?
5. **Validation**: Who needs to sign off, and what are the key test scenarios?
6. **Quality**: What's the test coverage threshold and CI/CD requirements?
```

### For Large Features (Epics, Multi-Week)

Full interview across all categories, done in 2-3 rounds:

**Round 1 - High Level:**
```
Let's start with the big picture:

1. **Vision**: What does success look like for this feature?
2. **Scope**: What's the MVP vs. future phases?
3. **Users**: Who are the primary users, and what's their journey?
4. **Dependencies**: What must be done before this, and what's waiting on this?
```

**Round 2 - Technical:**
```
Now let's get into technical details:

1. **Architecture**: Are there preferred patterns or technologies?
2. **Data**: Walk me through the data model - what's created, read, updated?
3. **Integrations**: What external systems are involved?
4. **Security**: Any compliance or security requirements?
```

**Round 3 - Execution:**
```
Final questions about execution:

1. **Testing**: What are the critical test scenarios?
2. **Rollout**: Feature flags? Gradual rollout? Hard launch?
3. **Monitoring**: How will we know it's working in production?
4. **Fallback**: What's the plan if something goes wrong?
```

---

## Handling Common Responses

### "I don't know"

This is valuable information! Record as a Known Unknown:

```markdown
### Known Unknowns
| Unknown | Impact | Resolution |
|---------|--------|------------|
| Expected peak load | Could affect caching strategy | Need to check with ops team |
```

### "Let's figure it out as we go"

Push back gently:

```
I understand we'll discover things as we build, but having a rough idea now
will help us design something flexible. Even a guess is helpful - we can
note it as an assumption that needs validation.
```

### "Just make it work like [competitor]"

Clarify:

```
I can look at [competitor] for inspiration. To make sure we're aligned:
1. Which specific aspects should we emulate?
2. What should we do differently or better?
3. Are there legal/IP concerns with copying too closely?
```

### Conflicting Requirements

Surface the conflict:

```
I'm seeing a potential conflict:
- Requirement A says [X]
- Requirement B says [Y]
- These seem to conflict because [reason]

Which takes priority, or how should we reconcile these?
```

---

## Interview Output Template

After completing the interview, document findings:

```markdown
## Interview Summary

### Participants
- [Names and roles]

### Date
[Date of interview]

### Key Decisions
1. [Decision 1 and rationale]
2. [Decision 2 and rationale]

### Confirmed Requirements
- [Requirement 1]
- [Requirement 2]

### Out of Scope (Confirmed)
- [Item 1]
- [Item 2]

### Quality Gates Status
| Tool | Status | Config | Notes |
|------|--------|--------|-------|
| Linter | ✅/❌ | [path] | [notes] |
| Formatter | ✅/❌ | [path] | [notes] |
| Type Checker | ✅/❌ | [path] | [notes] |
| Test Framework | ✅/❌ | [path] | [notes] |
| CI/CD | ✅/❌ | [path] | [notes] |

**Bootstrap Required**: [Yes / No]
**Coverage Threshold**: [X%]

### Assumptions Made
| # | Assumption | Risk if Wrong | Agreed By |
|---|------------|---------------|-----------|
| 1 | [Assumption] | [Risk] | [Name] |

### Known Unknowns
| # | Unknown | Impact | Resolution Plan |
|---|---------|--------|-----------------|
| 1 | [Unknown] | [Impact] | [Plan] |

### Open Questions (Pending)
- [ ] [Question] - Assigned to: [Name], Due: [Date]

### Next Steps
1. [Next step 1]
2. [Next step 2]
```

---

## Anti-Patterns to Avoid

### 1. The Interrogation
**Wrong**: Asking 20 questions upfront
**Right**: Ask 3-5, wait for answers, then follow up

### 2. The Assumption
**Wrong**: Assuming you know what they meant
**Right**: Confirm understanding with "So what you're saying is..."

### 3. The Technical Dump
**Wrong**: Asking deep technical questions to non-technical stakeholders
**Right**: Tailor question depth to the audience

### 4. The Yes/No Trap
**Wrong**: "Is performance important?" (always yes)
**Right**: "What's an acceptable response time for this operation?"

### 5. The Future Creep
**Wrong**: Planning for every possible future requirement
**Right**: Acknowledge future needs, design flexibly, but build only what's needed now
