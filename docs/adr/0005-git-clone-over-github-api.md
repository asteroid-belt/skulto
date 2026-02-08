# ADR-0005: Git Clone Over GitHub API for Repository Sync

**Status:** Accepted
**Date:** 2026-01-01

## Context

Skulto needs to fetch skill files from GitHub repositories. Two approaches exist: using the GitHub REST/GraphQL API to download individual files, or using git clone to fetch the entire repository locally.

The GitHub API has rate limits (60 requests/hour unauthenticated, 5000/hour with token). A single repository may contain dozens of skill files, each requiring multiple API calls for content, metadata, and directory traversal.

## Decision

Use git clone (via the go-git library) with shallow clones to fetch repositories locally. Cloned repositories are stored at `~/.skulto/repositories/{owner}/{repo}/`. The `UseGitClone` config flag defaults to `true`.

The scraper retains a `Client` interface that abstracts the data source, allowing fallback to the GitHub API if needed.

## Consequences

### Benefits

- No rate limit issues: git clone is a single operation regardless of repository size
- Offline access: cloned repositories persist locally for offline skill access
- Symlink installation: local clone provides the target directory for symlinks (see [ADR-0001](0001-symlink-based-skill-installation.md))
- Incremental sync: `git fetch` only downloads changes, not the full repository
- No authentication required: public repositories can be cloned without a token

### Trade-offs

- Disk usage: full repository is cloned even if only a few skill files are needed
- Initial sync is slower than targeted API calls for small repositories
- go-git adds a significant dependency (~20+ transitive packages)
- Repository cleanup (cache TTL) must be managed to prevent unbounded disk growth

### Alternatives Considered

| Alternative | Why Not Chosen |
|-------------|---------------|
| GitHub REST API | Rate limits make bulk operations unreliable; requires auth for reasonable limits |
| GitHub GraphQL API | Better batching but still rate-limited; more complex implementation |
| Download tarball | No incremental sync; still requires GitHub API call |

## Sources

> Evidence used to reconstruct this decision.

| Source Type | Reference |
|-------------|-----------|
| Code | `internal/scraper/scraper.go:63` - `ScraperConfig.UseGitClone` defaults to `true` |
| Code | `internal/scraper/git_client.go` - `GitClient` wraps go-git for clone/fetch operations |
| Code comment | `internal/scraper/scraper.go:58` - "DataDir...repositories cloned to DataDir/repositories" |
| Dependency | `go.mod` - `github.com/go-git/go-git/v5 v5.16.4` |

## Related

- [ADR-0001](0001-symlink-based-skill-installation.md) - Symlinks require local clone
- [Architecture](../architecture.md) - Scraper component
