# Agent Skills Specification Reference

> Source: https://agentskills.io/specification

## Directory Structure

A skill is a directory containing at minimum a `SKILL.md` file:

```
skill-name/
└── SKILL.md          # Required
```

Optional directories: `scripts/`, `references/`, `assets/`

## SKILL.md Format

YAML frontmatter followed by Markdown content.

### Required Fields

| Field | Constraints |
|-------|-------------|
| `name` | 1-64 chars, lowercase alphanumeric + hyphens, no leading/trailing/consecutive hyphens, must match directory name |
| `description` | Max 1024 chars, non-empty, describes what and when |

### Optional Fields

| Field | Constraints |
|-------|-------------|
| `license` | License name or file reference |
| `compatibility` | Max 500 chars, environment requirements |
| `metadata` | Key-value mapping for additional properties |
| `allowed-tools` | Space-delimited pre-approved tools (experimental) |

## Example Frontmatter

```yaml
---
name: pdf-processing
description: Extract text and tables from PDF files, fill forms, merge documents.
license: Apache-2.0
compatibility: Designed for Claude Code (or similar products)
metadata:
  author: example-org
  version: "1.0"
allowed-tools: Bash(git:*) Bash(jq:*) Read
---
```

## Progressive Disclosure

1. **Metadata** (~100 tokens): name/description loaded at startup
2. **Instructions** (< 5000 tokens recommended): Full body when activated
3. **Resources** (as needed): Files loaded only when required

Keep main `SKILL.md` under 500 lines.

## Optional Directories

### `scripts/`
Executable code agents can run. Should be:
- Self-contained with documented dependencies
- Include helpful error messages
- Handle edge cases gracefully

### `references/`
Additional documentation for on-demand loading:
- REFERENCE.md - Detailed technical reference
- Domain-specific files

### `assets/`
Static resources: templates, images, data files

## Validation

```bash
skills-ref validate ./my-skill
```
