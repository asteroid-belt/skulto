# ADR-0006: PostHog for Anonymous Telemetry

**Status:** Accepted
**Date:** 2026-01-01

## Context

Skulto needs usage analytics to understand which features are used, identify error patterns, and guide development priorities. Telemetry must be anonymous (no personal data, no IP addresses), opt-out capable, and lightweight enough for a CLI tool.

## Decision

Use PostHog as the telemetry backend with the Go SDK (`posthog-go`). The API key is injected at build time via ldflags to keep it out of source code. Telemetry is enabled by default and can be disabled with `SKULTO_TELEMETRY_TRACKING_ENABLED=false`.

A `Client` interface abstracts telemetry with two implementations:
- `posthogClient` - sends events to PostHog
- `noopClient` - discards all events (when disabled)

A persistent tracking ID (stored in the database) provides session continuity without identifying the user.

## Consequences

### Benefits

- Product analytics with minimal implementation effort
- Generous free tier suitable for an open-source CLI tool
- Dual implementation pattern (real + noop) makes opt-out clean
- Build-time API key injection keeps credentials out of source
- Events are fire-and-forget; telemetry errors never affect functionality

### Trade-offs

- Default-on telemetry may concern privacy-conscious users (mitigated by clear opt-out and no PII collection)
- PostHog SDK adds a dependency
- Every new feature requires adding telemetry calls to all three interfaces (CLI, TUI, MCP)

### Alternatives Considered

| Alternative | Why Not Chosen |
|-------------|---------------|
| No telemetry | No visibility into usage patterns for development prioritization |
| Self-hosted analytics | Operational overhead for an open-source project |
| Mixpanel / Amplitude | Less generous free tiers; PostHog is open-source friendly |

## Sources

> Evidence used to reconstruct this decision.

| Source Type | Reference |
|-------------|-----------|
| Dependency | `go.mod` - `github.com/posthog/posthog-go v1.9.0` |
| Code | `internal/telemetry/client.go` - PostHog client with endpoint `https://us.i.posthog.com` |
| Config file | `Makefile:9` - `POSTHOG_API_KEY` injected via ldflags |
| Code | `internal/telemetry/events.go` - event definitions for all three interfaces |
| Plan file | `plans/telemetry-consistency-plan.md` - plan for consistent telemetry across interfaces |

## Related

- [Architecture](../architecture.md) - Telemetry component
- [Glossary](../glossary.md) - Telemetry definition
