# Git CLI Safe Commit Messages

**CRITICAL: Commit messages must be safe for direct use with `git commit -m`.**

Based on [Conventional Commits v1.0.0](https://www.conventionalcommits.org/en/v1.0.0/).

---

## Canonical Format

```
<type>(<scope>): <description>

<body>

<footer>
```

**Regex validation pattern:**
```regex
^(feat|fix|docs|style|refactor|perf|test|build|ci|chore)(\([a-z0-9-]+\))?(!)?: .{1,72}$
```

---

## Shell-Safe Characters

### Characters to NEVER USE

| Character | Why Dangerous | Shell Behavior |
|-----------|---------------|----------------|
| `"` | Breaks quoting | Interpreted as string delimiter |
| backtick | Command substitution | Executes as subshell command |
| `$` | Variable expansion | Replaced with variable value (often empty) |
| exclamation mark | History expansion | Replaced with previous command in bash |
| `\` | Escape sequences | May escape following character unexpectedly |
| `#` at line start | Comment marker | Everything after ignored |
| `;` | Command separator | Splits into multiple commands |
| `&` | Background operator | May background the commit |
| `|` | Pipe operator | May pipe output unexpectedly |
| `>` `<` | Redirects | May redirect to/from files |
| `*` `?` | Glob patterns | May expand to filenames |

### ALWAYS SAFE Characters

```
a-z A-Z 0-9
- _ . , : ( ) / ' space
```

### Transformation Rules

When you encounter unsafe characters, apply these transformations:

| Original | Transform To |
|----------|--------------|
| `"word"` | `'word'` or just `word` |
| `it's` | `it is` (avoid apostrophe in contractions) |
| `$variable` | `variable` or `the variable` |
| fix followed by exclamation | `fix.` or `fix -` |
| `foo & bar` | `foo and bar` |
| `a > b` | `a to b` or `a over b` |
| `100%` | `100 percent` |
| `C++` | `cpp` or `C plus plus` |
| `#123` | `issue 123` or in footer only |

---

## Commit Types (Required)

| Type | Semantic Version | When to Use |
|------|------------------|-------------|
| `feat` | MINOR bump | New feature for the user |
| `fix` | PATCH bump | Bug fix for the user |
| `docs` | No bump | Documentation only |
| `style` | No bump | Formatting, whitespace (no logic change) |
| `refactor` | No bump | Code restructure (no behavior change) |
| `perf` | PATCH bump | Performance improvement |
| `test` | No bump | Adding/updating tests |
| `build` | No bump | Build system, dependencies |
| `ci` | No bump | CI/CD configuration |
| `chore` | No bump | Other maintenance |

**Decision tree:**
```
Is it user-facing?
├── Yes → Does it add capability?
│         ├── Yes → feat
│         └── No → fix
└── No → Is it code change?
         ├── Yes → refactor/perf/style
         └── No → docs/test/build/ci/chore
```

---

## Scope (Optional but Recommended)

The scope provides context about which part of the codebase changed.

### Scope Naming Rules

1. **Lowercase only**: `auth` not `Auth`
2. **Hyphenated**: `user-service` not `userService`
3. **Short**: 1-2 words max
4. **Consistent**: Use same scope for related changes

### Common Scope Patterns

| Pattern | Examples |
|---------|----------|
| Feature area | `auth`, `api`, `ui`, `db` |
| Component | `button`, `modal`, `navbar` |
| Layer | `service`, `controller`, `model` |
| Module | `users`, `orders`, `payments` |

### Deriving Scope from File Path

```
src/services/auth.ts      → scope: auth
src/components/Button.tsx → scope: button
src/api/users/index.ts    → scope: users-api
lib/utils/validation.ts   → scope: validation
```

---

## Description (Required)

### Rules

1. **Imperative mood**: "add feature" not "added feature" or "adds feature"
2. **No period at end**: "add login" not "add login."
3. **Lowercase first letter**: "add login" not "Add login"
4. **Max 50 characters** (hard limit: 72)
5. **Complete thought**: Should complete "This commit will..."

### Imperative Mood Reference

| Wrong | Correct |
|-------|---------|
| added | add |
| adding | add |
| adds | add |
| fixed | fix |
| fixing | fix |
| updated | update |
| removed | remove |
| changed | change |

### Description Templates

```
feat(scope): add [feature] for [purpose]
fix(scope): resolve [issue] when [condition]
refactor(scope): extract [thing] to [location]
test(scope): add tests for [component/function]
docs(scope): update [document] with [content]
chore(scope): update [dependency] to [version]
```

---

## Body (Optional)

### When to Include Body

- Change is not obvious from description
- Multiple related changes in one commit
- Context needed for reviewers
- Workarounds or non-obvious decisions

### Body Format Rules

1. **Blank line** between description and body
2. **Wrap at 72 characters**
3. **Use bullet points** for multiple items (hyphen `-`)
4. **Explain WHY**, not just what

### Body Template

```
<description>

<why this change was needed>

Changes:
- <change 1>
- <change 2>
- <change 3>

<any caveats or notes>
```

---

## Footer (Optional)

### Breaking Changes

Two valid formats:

```
# Format 1: Footer (preferred for detailed explanation)
feat(api): change authentication flow

BREAKING CHANGE: JWT tokens now expire after 1 hour instead of 24 hours.
Clients must implement token refresh logic.

# Format 2: Bang notation (for simple breaks)
feat(api)!: change authentication endpoint path
```

### Issue References

```
# Closes issue when merged
Closes #123
Fixes #456

# References without closing
Refs #789
See #101

# Multiple issues
Closes #123, #456
```

### Co-authors

```
Co-authored-by: Name <email@example.com>
```

---

## Complete Examples

### Simple Fix
```
fix(auth): resolve token refresh race condition
```

### Feature with Body
```
feat(api): add rate limiting to public endpoints

Implement token bucket algorithm with configurable limits per endpoint.
Default: 100 requests per minute for unauthenticated users.

- Add rate limit middleware
- Create redis-backed token bucket
- Add rate limit headers to responses

Closes #234
```

### Breaking Change
```
feat(db)!: migrate from MySQL to PostgreSQL

BREAKING CHANGE: Database connection strings must be updated.
All date fields now use ISO 8601 format.

Migration guide: docs/migrations/mysql-to-postgres.md

Closes #567
```

### Multi-line with HEREDOC

**Output this format for the user:**

```bash
git commit -m "$(cat <<'EOF'
feat(auth): add multi-factor authentication

Implement TOTP-based 2FA with backup codes.

- Add TOTP secret generation
- Create QR code display component
- Implement backup code system
- Add 2FA verification middleware

Closes #890
EOF
)"
```

---

## Validation Checklist

Before outputting a commit message, verify:

- [ ] Type is one of: feat, fix, docs, style, refactor, perf, test, build, ci, chore
- [ ] Scope (if present) is lowercase with hyphens only
- [ ] Description starts lowercase, no period, imperative mood
- [ ] Description under 50 chars (max 72)
- [ ] No shell-unsafe characters: double quote, backtick, dollar sign, exclamation mark, backslash, semicolon, ampersand, pipe, redirects, glob wildcards
- [ ] Body (if present) has blank line after description
- [ ] Footer (if present) has blank line after body
- [ ] BREAKING CHANGE is uppercase if used
- [ ] Issue refs use correct format: Closes #N, Fixes #N, Refs #N

---

## Sources

- [Conventional Commits v1.0.0](https://www.conventionalcommits.org/en/v1.0.0/)
- [Angular Commit Guidelines](https://github.com/angular/angular/blob/main/CONTRIBUTING.md#commit)
- [Git Commit Best Practices](https://cbea.ms/git-commit/)
