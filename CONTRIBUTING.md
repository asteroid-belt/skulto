# Contributing to Skulto

Thank you for considering contributing to Skulto! This document outlines how to contribute.

## Getting Started

### Prerequisites

- Go 1.25+
- Make
- (Optional) GitHub token for higher API rate limits

### Development Setup

```bash
# Clone the repository
git clone https://github.com/asteroid-belt/skulto.git
cd skulto

# Install dependencies
make deps

# Build
make build

# Run tests
make test

# Run linter
make lint
```

## How to Contribute

### Reporting Bugs

1. Check existing issues first
2. Use the bug report template
3. Include: Go version, OS, steps to reproduce, expected vs actual behavior

### Suggesting Features

1. Open an issue with the feature request template
2. Describe the use case and proposed solution
3. Be open to discussion

### Submitting Pull Requests

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Ensure tests pass (`make test`)
5. Ensure linter passes (`make lint`)
6. Commit with conventional commit messages
7. Push to your fork
8. Open a Pull Request

### Commit Message Format

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(scraper): add support for new skill format
fix(tui): correct keybinding for search
docs(readme): update installation instructions
refactor(installer): extract symlink helper
test(db): add characterization tests
```

### Code Style

- Follow standard Go conventions (`make format`)
- Write tests for new functionality
- Document exported functions
- Keep functions focused and small

## Code of Conduct

Please read our [Code of Conduct](CODE_OF_CONDUCT.md) before contributing.

## Questions?

Open an issue with the question label or start a discussion.
