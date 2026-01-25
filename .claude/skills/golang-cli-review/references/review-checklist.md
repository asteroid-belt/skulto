# Go CLI Review Checklist Template

Use this as the output template for code reviews. Copy and fill in based on findings.

---

## Review Summary

**Project:** [name]
**Files Reviewed:** [count]
**Review Date:** [date]

| Severity | Count |
|----------|-------|
| Critical | 0 |
| Warning  | 0 |
| Suggestion | 0 |
| Positive | 0 |

---

## Critical Issues
*Must fix before merge/release. Security vulnerabilities, data loss risks, crashes.*

- [ ] **[C1]** `file.go:123` - [Description]
  - **Impact:** [What could go wrong]
  - **Fix:** [How to resolve]

---

## Warnings
*Should fix. Bugs, poor patterns, maintenance concerns.*

- [ ] **[W1]** `file.go:45` - [Description]
  - **Why:** [Explanation]
  - **Fix:** [Suggested change]

---

## Suggestions
*Nice to have. Style, optimization, minor improvements.*

- [ ] **[S1]** `file.go:78` - [Description]
  - **Consider:** [Alternative approach]

---

## Positive Patterns
*Good practices worth highlighting.*

- **[P1]** `file.go:90` - [What was done well]

---

## Category Breakdown

### CLI Structure & Framework
- [ ] Uses established framework (Cobra/urfave) appropriately
- [ ] Command hierarchy is logical and discoverable
- [ ] Help text is clear and includes examples
- [ ] Version command present and follows semver

### Error Handling
- [ ] Uses RunE over Run (returns errors properly)
- [ ] Errors wrapped with context using %w
- [ ] User-facing errors are actionable
- [ ] Exit codes are meaningful and documented
- [ ] No panic in normal code paths

### Flag & Argument Design
- [ ] Flag names are lowercase-kebab-case
- [ ] Short flags for common options (-v, -o, -q)
- [ ] Required flags marked and validated
- [ ] Sensible defaults provided
- [ ] Flag descriptions are helpful

### Input/Output
- [ ] Errors go to stderr, data to stdout
- [ ] Supports machine-readable output (--json, --format)
- [ ] Respects --quiet and --verbose
- [ ] Interactive prompts disabled in non-TTY mode
- [ ] Progress indicators on long operations

### Security
- [ ] No secrets accepted via command-line flags
- [ ] File paths validated against directory traversal
- [ ] No shell command injection vulnerabilities
- [ ] External URLs validated (scheme, host)
- [ ] Sensitive data not logged

### Testing
- [ ] Commands have unit tests
- [ ] Error cases tested
- [ ] Integration tests for critical paths
- [ ] Test coverage adequate (>70%)

### Performance & Resources
- [ ] Context used for cancellation
- [ ] SIGINT/SIGTERM handled gracefully
- [ ] Resources cleaned up properly (defer)
- [ ] No goroutine leaks
- [ ] Large inputs handled efficiently

### Go Idioms
- [ ] Follows effective Go style
- [ ] Errors handled immediately after calls
- [ ] No naked returns in non-trivial functions
- [ ] Package names are lowercase, singular
- [ ] Interfaces accepted, structs returned
