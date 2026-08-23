# Skulto v2 — The Skill Environment Manager

**Date:** 2026-07-12
**Status:** Design / PRD — approved in brainstorming, pending implementation plan
**Supersedes:** the positioning in `docs/overview.md` ("cross-platform AI skills management")
**Amends:** ADR 0003 (three interfaces, shared services)

---

## 1. Summary

Skulto v1 is a package manager for AI agent skills: it scrapes them from GitHub, indexes
them in SQLite, scans them for prompt injection, and symlinks them into 33 AI coding tools.
The unit of management is **a skill**.

Skulto v2 changes the unit of management to **the environment** — the set of skills a
repository runs on, taken as a whole and held to a standard.

> Skulto proposes a **minimal, budgeted skill environment intended to be sufficient** for what
> you're about to build,
> **proves it fits** only where the agent's context budget is verified, labels fallback estimates
> **budget estimated**, marks unresolved limits **budget unverified**, **reports explicit observed
> skill invocation where a supported adapter provides evidence**, and
> **produces a reproducible scan result against specifically identified signed ruleset content** —
> and reruns that scan as new detection rules land.

Three capabilities, in a lifecycle:

| | | |
|---|---|---|
| **Compose** | Propose a minimal, budgeted environment intended to be sufficient for the work at hand | select → bind → prune → budget-verify |
| **Verify** | Produce reproducible results against a named security ruleset, and keep them current | deterministic scan → signed, patchable rule feed |
| **Evolve** | Feed back what the agent learned by running, and only what it couldn't have read | execution oracle → distil → delta-update |

Here, **minimal** is the composition objective, not a proof of global minimality, and **intended to
be sufficient** is an agent proposal, not a Skulto guarantee. Deterministic validation establishes
artifact identity, policy compliance, and budget status; Phase 2.5 supplies the empirical evidence
for whether composed environments preserve task success.

**Headline value is subtraction, not addition.** Every claim maps to a strongly-evidenced
market pain (§2). No skill is ever generated (§4).

---

## 2. Why: the evidence

### 2.1 What practitioners actually complain about (July 2026)

| Pain | Evidence strength | Independent sources |
|---|---|---|
| Context bloat; people actively deleting MCP servers and skills | **STRONG** | 10+ |
| Skills silently truncated → never fire | **STRONG** | 8+ |
| Trust: 36% of published skills contain injection flaws | **STRONG** | 6+ |
| Agent ignores CLAUDE.md / repo conventions | **STRONG** | 5+ |
| Curation/quality ("everything is findable, most is junk") | MODERATE | 4 |
| Team config drift | WEAK (thought-leadership only) | 4 |
| *Generic skills need forking for my stack* | **WEAK — contradicted** | ~1 |
| *Authoring a skill is hard* | **WEAK — contradicted** (`skill-creator` ships free, ~5 min) | — |

Hard numbers behind the top rows:

- 7 MCP servers ≈ **67,300 tokens ≈ 33.7%** of a 200k window, before the user types anything.
- Tool-selection accuracy fell **95% → 71%** with a single large MCP server loaded.
- Claude Code injects skill descriptions under a **~15,000-char budget**. Exceed it and skills
  remain installed but become **invisible**, silently. (`anthropics/claude-code` #14549 reports
  *"Showing 60 of 3218 skills"* when only 169 existed. Also #40121, #47627, #9926, #27888.)
- Passive skill trigger rate from descriptions alone: **30–50%**.
- Snyk "ToxicSkills" audited 3,984 skills: **36% contain prompt injection or security flaws,
  13.4% critical, 76 confirmed malicious payloads, 8 still live at publication.** The barrier
  to publishing is *"a SKILL.md and a GitHub account that's one week old. No code signing.
  No security review. No sandbox by default."*

**The market's loudest complaint is that it already has too many artifacts.** A product that
adds artifacts is fighting the current. A product that *removes* them, makes the remainder
reproducibly auditable, and reports supported execution evidence is riding it.

### 2.2 The competitive gap is real — and empty for a reason

No shipped product specialises a curated skill for a specific project. The near-misses:

- **Vercel `skills`** (25,947★, six months) — seven commands, none touch skill content.
- **SkillKit** — `recommend` reads your repo, detects your stack, ranks skills… and stops at ranking.
- **Cursor** — `/Generate Cursor Rules` generates rules from your codebase, but has no registry to pull from.
- **AWS Kiro** — "Generate Steering Docs" generates guidance **from the codebase**, not from the spec.
- **Agent OS** — `/discover-standards` generates standards **from the codebase**.
- **BMAD v6** — generates skills **from installed modules**.
- **Tessl** — installs skill+rule "Tiles" **from a registry**.
- **Anthropic** — `/run` and `/verify` bootstrap repo-local skills **from execution traces**.

Every one derives the skill layer from what **already exists** or from **human-authored
governance**. Nobody derives it from **what you are about to build**.

**But the demand research found zero organic requests for that capability, and zero
willingness to pay.** People do not fork generic skills — *they delete them*. An empty market
slot with no demand signal is evidence *against*, not *for*. This PRD treats the anticipatory
arrow as an unproven hypothesis (§9), not as the product.

### 2.3 The experiment that already ran, and what it actually proved

**"Evaluating AGENTS.md: Are Repository-Level Context Files Helpful for Coding Agents?"**
(ETH Zurich SRI Lab + LogicStar, Feb 2026, arXiv 2602.11988). 138 tasks from 12 real repos
mined from 5,694 PRs, plus SWE-bench Lite. Four agents.

- **LLM-generated context files reduced task success** (~-2% on AGENTbench), worse in **5 of 8 settings**.
- **Cost rose regardless: +20–23% inference cost**, +2.45–3.92 steps/task.
- Human-written files: **only +4pp**.
- **The ablation is the finding.** Delete the repo's other documentation, *then* generate a
  context file, and the generated file **improves 2.7% and beats the human-written one**.

**Mechanism: generated context is a lossy re-encoding of documentation the agent could read
itself.** Its value is *negative* when the source exists and *positive* only when it doesn't.

The corollary is skulto v2's central discipline: **the only content worth writing down is
content the agent cannot discover.** Human-written files earned their +4pp on exactly this —
*"run tests with `--no-cache` or fixtures fail"*, *"use `uv` not `pip`"*, *"this deprecated
module is load-bearing."*

### 2.4 Every system that works has an execution oracle

Voyager (15.3× faster progression), SkillWeaver (+31.8% / +39.8% relative), ACE (+17pp
AppWorld), ReasoningBank (+8.3pp WebArena), Alita-G (83.03% GAIA), GEPA (ICLR 2026 oral) —
**every one generates its artifacts from verified trajectories with a success/failure signal,
and retrieves them rather than preloading them.** ACE loses 3.4 points when run without ground
truth.

ACE also documents **context collapse**: a single wholesale regeneration took an accumulated
context from 18,282 tokens @ 66.7% accuracy to **122 tokens @ 57.1% — below the
no-adaptation baseline of 63.7%.** Delta updates, never rewrites.

**Consequence for this PRD:** skulto does not author capability claims, because it has no
oracle for them (§4, §9).

### 2.5 A worked example, from this repository

Applying the discoverability test to skulto's own `AGENTS.md`:

**Discoverable** (net-negative by the ETH criterion — the agent reads it itself in one or two
tool calls): the Quick Facts table (all of it is in `go.mod` and `Makefile`), the Repository
Tour (that is `tree`; the paper found codebase overviews *"did not reduce steps-to-relevant-file
at all"*), Build/Run/Test/Lint/Clean (a paraphrase of the Makefile), the documentation tables
(that is `ls docs/`), and "follow standard Go conventions."

**Non-discoverable** (the +4pp content class — no linter catches these, no generic skill
contains them):

1. **"Three interfaces, one codebase."** Every user-facing feature must land in CLI *and* TUI
   *and* MCP (ADR 0003). **The compiler will never tell you that you shipped a CLI command with
   no MCP handler.** Invisible to `go vet`, invisible to tests. An LLM reviewer, told the rule,
   catches it every time.
2. **The telemetry four-step ritual.** Define in `events.go` → add to `posthogClient` **and**
   `noopClient` → add to the `Client` interface → call from **all three** interfaces. Steps 1–3
   are compiler-enforced. **Step 4 is not**, and it is the one that silently breaks the product.
3. **`make dev` and `make test-race` require `CGO_ENABLED=1`**, while the DB layer was
   deliberately chosen pure-Go to avoid CGO (ADR 0002). An agent hits a confusing race-detector
   failure with no way to know it is expected.
4. **"Every symlink points to a directory under `~/.agents/skulto/`"** — the invariant from the
   accepted `2026-04-19-unify-local-skill-storage` spec. Go's type system cannot express it.

Roughly **85% of skulto's own `AGENTS.md` is content the paper says costs you success rate.**
That ratio is the product demo.

---

## 3. Non-goals

Explicitly out of scope for v2. Each was considered and rejected with a reason.

| Not doing | Why |
|---|---|
| **Generating or forking skills** ("skill compilation") | No execution oracle. §2.3, §2.4, §9. |
| Template engine, `text/template`, parameter substitution in `SKILL.md` | Follows from the above. |
| A hosted backend, accounts, or a service tier | The rule feed is a git repo; embedded and cached exact rules keep audit offline-capable. |
| Hooks into the agent's session | Transcripts already contain the data. §7. |
| A proxy or wrapper (`skulto run claude …`) | Puts skulto in the critical path of the user's session. |
| Writing prose into `AGENTS.md` / `CLAUDE.md` | §4, and the spec-kit tombstone (#365, #596, #309). |
| Instructing the agent to report telemetry | Instructions are a terrible substrate for measurement. §7.1. |
| MCP feature parity with the CLI | Capability-gated, not parity-gated. §6. |

---

## 4. Invariants

These are the constitution. Violating one of them is a bug, not a trade-off.

1. **Skulto never authors a capability claim.** It selects, binds, prunes, and orchestrates.
   Generation is confined to non-discoverable facts elicited from a human or extracted from a
   verified execution trace — never invented from reading a repo.
2. **Skulto never writes a file it does not own.** Two narrow, opt-in exceptions, both
   additive and idempotent: one delimited reference line in `AGENTS.md` pointing at
   `.skulto/context.md`, and a minimal set of read-only `skulto` command patterns appended to the
   Bash allowlist array in the platform's settings JSON. Nothing else. Ever.
3. **`.skulto/context.md` changes by entry-level delta only, never by whole-file regeneration.**
   Entries may be added, amended, deprecated, or removed; history is retained separately. (ACE
   context collapse, §2.4.)
4. **Facts and routing are distinct.** Every fact in `.skulto/context.md` must fail the
   repository-wide discoverability test. Routing entries may point to authoritative repository
   documents but must not paraphrase them. Enforced by deterministic lint where possible, then
   agent classification and human approval (§5.2).
5. **Every write to an artifact passes a human approval gate.** Skulto proposes; the human disposes.
   Approval is either Skulto's native interactive flow or an explicit `accept` response from a
   capability-negotiated MCP client elicitation showing the exact diff. Noninteractive mutation
   without one of those approvals bound to the exact proposal digest is a hard rejection.
6. **Determinism where skulto acts.** Budget counts are arithmetic. Scans are rule-driven and
   reproducible. Manifests are serialisation. Skulto never embeds an LLM client or adds a new
   hosted inference dependency; it may orchestrate a supported, locally installed agent CLI
   (§6.1). The existing opt-in, `OPENAI_API_KEY`-gated TUI embedding path is grandfathered for
   compatibility only and may not become a dependency of Compose, CLI, or MCP.
7. **Skulto's own always-on context footprint is a published, enforced budget** (~120 tokens).
   `skulto doctor` fails if it grows.
8. **Transcript content never leaves the machine.** No transcript text, correction span, file
   path, skill body, or search query from any surface appears in any telemetry event. Queries may
   not be hashed, truncated, tokenised, sampled, or placed in error text; bounded aggregate counts
   only. Repository and skill identifiers may be emitted only when their source is verified public;
   private, local, and unknown provenance is reduced to coarse origin and aggregates. Raw CLI
   arguments and free-form errors are forbidden. Enforced by test.
9. **Flag, never delete or disable automatically.** A skill that newly fails audit remains active
   but is marked and reported. New installation or upgrade is blocked until the content is fixed or
   a human explicitly accepts the exact findings for the exact content digest.
10. **Observability fails soft.** A transcript parse error degrades a report. It never crashes,
    never blocks compose, never blocks audit.
11. **Budget claims never exceed their evidence.** Verified limits are hard constraints. Estimated
    limits are always labeled `budget estimated` and permit an explicit human overrun override.
    Unknown limits never produce a fit claim; Compose may apply after specific human confirmation,
    but the manifest, diff, doctor output, and reports retain `budget unverified` until a supported
    source establishes a limit. Live probes report drift without mutating the approved lockfile
    snapshot; only a human-approved recomposition may replace it.
12. **A pinned skill has immutable identity, not magically immutable local bytes.** Every
    environment-managed installation resolves to a content-addressed cache object whose full
    directory digest matches the manifest at the operation's trust boundary. The cache is
    write-protected and tamper-evident, not trusted merely because its path contains a digest. A
    source commit without a freshly verified object is not an installed pin.
13. **Platform-native tracked skills are mandatory baseline.** If a Git-tracked skill exists in a
    selected platform's project discovery directory, Skulto cannot pretend it is absent: Compose
    audits it, locks it as a workspace source, and includes its description in that platform's
    budget. Removal requires a human repository change and a new composition.
14. **Agent capability probes are inert and fail closed.** Only adapter-owned, allowlisted,
    non-interactive version and machine-readable capability commands may establish dynamic facts.
    Probes run with closed stdin, bounded execution, and schema-validated output; they never start
    an agent session, load repository instructions, run project hooks, or construct a command from
    untrusted input. Unsupported discovery becomes `estimated` or `unknown`, never guessed.
15. **Artifact identity has one portable byte-level definition.** Only directories and regular
    files enter an artifact; symlinks and special files are rejected. Canonical path, executable
    bit, length, and raw content bytes determine the digest, with no host-dependent normalization.
16. **Untrusted structured input is parsed strictly and within bounds.** Manifests, proposals,
    rulesets, agent results, MCP arguments, and history events reject duplicate keys, unknown
    schema versions, invalid Unicode, trailing data, oversized inputs, and ambiguous encodings.
    Untrusted strings are never interpreted as shell syntax or terminal control sequences.
17. **Apply is serialized, recoverable, and compare-and-swap.** A mutation-scope lock prevents
    concurrent changes to the same repository or local installation target. Apply rechecks every
    bound preimage immediately before writing, journals its intent, uses durable atomic replacement
    where the filesystem permits, and either completes or recovers idempotently after interruption.
    Partial success is never reported as success.

### 4.1 Threat model and limits

Skulto is designed to resist mutable upstream refs, accidental or opportunistic corruption of its
local cache, unsigned or rolled-back ruleset updates, stale budget observations, untrusted agent
proposals, path traversal, ambiguous structured data, accidental concurrent writes, and telemetry
leakage. It makes those failures detectable and fails closed at the trust boundary.

Skulto does **not** claim to protect against:

- an attacker with the same OS-user privileges—or root—who can alter Skulto's executable, local
  state, trust keys, agent binary, or bytes after verification and before the agent reads them;
- a compromised supported agent CLI, the agent vendor, the user's account, or network service;
- an authorized signing key or rules maintainer publishing a bad but correctly signed rule;
- semantic malicious behavior that the named deterministic ruleset does not detect; or
- a compromised kernel, filesystem, Git client, or cryptographic implementation.

Write protection, atomic rename, and re-verification narrow local races but do not eliminate the
same-user time-of-check/time-of-use window. Closing that window requires an OS-enforced sandbox,
read-only mount, or stronger process isolation and is outside v2. A repository-read-only agent
session prevents repository mutation where the adapter can enforce it; it does not make the agent
trustworthy or prevent the user's chosen agent service from receiving repository content under its
normal product terms. Before launching an external agent, Skulto identifies the executable,
adapter, version, account/network boundary where detectable, and enforced restrictions. The user
may decline without changing project state.

A passing audit means only that the exact content produced the recorded result under the exact
scanner and signed ruleset. It is not a malware-free, safe, or functional certification. An
accepted risk remains an explicit policy exception, not a changed scan result.

---

## 5. Artifacts

Four. No skill is ever generated.

| Artifact | Owner | Purpose |
|---|---|---|
| `skulto.json` (schema 2) | skulto | Current environment lockfile: pinned skills, budget cost, and scan state |
| `.skulto/context.md` | skulto | Pointer-only routing plus genuinely non-discoverable project facts |
| `.skulto/history.jsonl` | skulto | Metadata-only, append-only audit history for approved artifact mutations |
| `rules/*.yaml` (git feed) | skulto (remote repo) | Versioned, ID'd injection-detection rules as data |

### 5.1 `skulto.json` schema 2

```json
{
  "schema": 2,
  "revision": 7,
  "generated_by": { "name": "skulto", "version": "2.0.0" },
  "minimum_skulto_version": "2.0.0",
  "history_head": "sha256:...",
  "environment": {
    "composed_at": "2026-07-12T18:22:04Z",
    "composed_from": "docs/prd/skulto-v2.md",
    "platforms": ["claude", "codex"],
    "scope": "project"
  },
  "skills": [
    {
      "slug": "golang-cli-review",
      "source": { "kind": "git", "repository": "github.com/asteroid-belt/skills" },
      "path": "skills/golang-cli-review",
      "ref": "sha1:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      "content_digest": "sha256:...",
      "description": { "digest": "sha256:...", "unicode_scalar_values": 142 },
      "scanned": {
        "ruleset_sequence": 42,
        "ruleset_version": "2026-07-01",
        "ruleset_digest": "sha256:...",
        "ruleset_commit": "sha1:ffffffffffffffffffffffffffffffffffffffff",
        "ruleset_key_id": "release-2026-a",
        "scanner_version": "2.0.0",
        "scope": "artifact-v1-all-files",
        "coverage": "complete",
        "files_total": 4,
        "files_scanned": 4,
        "bytes_scanned": 18240,
        "verdict": "passed",
        "finding_ids": []
      }
    },
    {
      "slug": "team-release-checks",
      "source": { "kind": "workspace", "path": ".claude/skills/team-release-checks" },
      "content_digest": "sha256:...",
      "description": { "digest": "sha256:...", "unicode_scalar_values": 96 },
      "scanned": {
        "ruleset_sequence": 42,
        "ruleset_version": "2026-07-01",
        "ruleset_digest": "sha256:...",
        "ruleset_commit": "sha1:ffffffffffffffffffffffffffffffffffffffff",
        "ruleset_key_id": "release-2026-a",
        "scanner_version": "2.0.0",
        "scope": "artifact-v1-all-files",
        "coverage": "complete",
        "files_total": 3,
        "files_scanned": 3,
        "bytes_scanned": 9400,
        "verdict": "audit_failed",
        "finding_ids": ["SKL-0012"]
      },
      "risk_acceptance": {
        "content_digest": "sha256:...",
        "findings": [
          { "rule_id": "SKL-0012", "rule_digest": "sha256:...", "severity": "high" }
        ],
        "reason": "Quoted adversarial example used by the security training skill.",
        "approved_at": "2026-07-12T18:22:04Z",
        "expires_at": null
      }
    }
  ],
  "budget": {
    "claude": {
      "status": "verified",
      "limit": 15000,
      "used": 4210,
      "unit": "unicode_scalar_values",
      "calculation": "claude-skill-metadata-v1",
      "skills": 6,
      "source": {
        "kind": "dynamic_cli",
        "agent_version": "1.2.3",
        "method": "capabilities"
      }
    },
    "opencode": {
      "status": "unknown",
      "used": 4210,
      "unit": "unicode_scalar_values",
      "calculation": "opencode-skill-metadata-v1",
      "skills": 6,
      "source": { "kind": "none", "agent_version": "0.9.1" }
    }
  },
  "ignored": []
}
```

The core additions are load-bearing:

- **`schema`** — a real schema version. Today's `version` field is a *revision counter* that
  `internal/cli/save.go:178` increments on content change; the name is a lie. It becomes
  `revision`. Migration keys off the absence of `schema`. `generated_by` and
  `minimum_skulto_version` make compatibility and unsupported downgrade explicit; a binary that
  does not understand the schema or minimum version preserves the bytes and fails closed.
- **`history_head`** — the digest of the last metadata-only history event. It makes deletion,
  insertion, and reordering of local history detectable when the manifest and history are checked
  together; Git remains the durable team record.
- **`path` + `ref` + `content_digest`** — immutable content identity. v1 has **none**: `skulto sync`
  resolves a slug against whatever HEAD happens to be, and installed symlinks point into a mutable
  checkout. Schema 2 resolves the exact repository path at the full commit, hashes the complete
  skill directory, materialises it under `~/.agents/skulto/objects/sha256/<digest>/`, and points
  environment-managed platform symlinks at that object. **The digest is the immutable identity;
  the local object is a write-protected, tamper-evident cache and must be reverified.**
  Commit IDs are full, algorithm-qualified Git object IDs (`sha1:<40hex>` or `sha256:<64hex>`),
  never abbreviations, branches, or tags; the independent SHA-256 content digest binds the artifact
  bytes regardless of the repository's Git object format.

#### Canonical artifact digest v1

`content_digest` is `sha256:<lowercase-hex>` over this stream:

1. the domain-separation bytes `skulto-artifact-v1\0`;
2. one record per regular file, ordered lexicographically by its canonical artifact-root-relative
   UTF-8 path bytes; and
3. for each record, `uint32be(path_byte_length)`, the path bytes, one byte (`0x00` non-executable or
   `0x01` executable), `uint64be(content_byte_length)`, and the raw file bytes.

Canonical paths use `/`, are valid UTF-8 in NFC form, and contain no absolute prefix, empty
segment, `.`, `..`, NUL, or backslash. Paths that collide after Unicode normalization or
case-folding are rejected for cross-platform portability. Directories are implicit—Git does not
track empty directories—and file mode is reduced to the Git executable bit (`0644` or `0755`),
read from the resolved tree or workspace index rather than host umask. Content receives no newline,
encoding, timestamp, ownership, extended-attribute, or platform normalization.

The walker never follows links. A symlink, socket, FIFO, device, or other non-regular entry is a
hard artifact-resolution error. Git submodules/gitlinks and unresolved Git LFS pointer files are
also rejected; artifact resolution never performs an implicit nested or LFS network fetch.

Materialization uses no-follow filesystem operations beneath a `0700` Skulto-owned cache root and
creates its private staging directory on the same filesystem as the final object. It writes regular
files, verifies the digest, durably flushes file and directory metadata, removes write permission
(`0444` data files, `0555` executable files and directories), atomically renames the result to
`objects/sha256/<lowercase-hex>/`, and durably flushes the parent directory. These permissions are
defense in depth, not the trust root. An existing object is accepted only after re-verification.
Install, apply, audit, `doctor`, repair paths, and any operation that treats cached bytes as the
pinned artifact recompute the canonical digest first; a mismatch fails closed and requires
rematerialization from the locked source.

Resolution has centrally defined, versioned upper bounds for path length, file count, individual
file size, total bytes, nesting depth, and time. Exceeding a bound is an explicit artifact error,
never truncation or partial hashing. The implementation plan selects measured defaults and records
them in golden and boundary tests. Cross-platform golden vectors define the encoding and digest
before any installer uses it.

- **`source.kind`** — `git` entries resolve a repository, full commit, and path. `workspace`
  entries resolve a Git-tracked skill directory in the same repository as `skulto.json`, such as
  `.claude/skills/<slug>`, plus its full-directory digest. An untracked, modified, global-only, or
  out-of-worktree local skill is unresolved and cannot enter a reproducible environment.
  Repository identity includes the canonical provider host and case-normalized provider owner/name;
  a branch, tag, redirect, or shorthand is never the installed identity. Retrieval may use the
  user's existing Git credentials but never records or emits them, and any provider redirect must
  resolve to the approved canonical repository identity. The content digest remains authoritative
  for bytes even when a provider later transfers or renames a repository.
- **`description`** — the exact tier-1 description digest and Unicode-scalar-value count. It is
  source evidence, not a universal platform cost. Each selected platform's budget snapshot records
  its own `unit` and versioned `calculation`; proposals provide the per-skill breakdown used to
  reach the aggregate. Character, byte, and tokenizer-specific units are never silently converted
  or compared. Skill metadata is parsed strictly with duplicate-key rejection. The description
  digest is SHA-256 over `skulto-description-v1\0`, its `uint64be` UTF-8 byte length, and the exact
  parsed UTF-8 value without Unicode or newline normalization.
- **`scanned` evidence** — which ruleset version cleared or flagged this skill. Enables *"this was
  scanned against 2026-07-01; 14 rules have landed since."* The manifest also records ruleset
  sequence, digest, source commit, signing-key ID, scanner version, verdict, and finding IDs so CI
  can reproduce the evidence and distinguish display labels from anti-rollback order. Scope,
  coverage, and file/byte counts prevent a nominal pass from hiding skipped artifact content.
- **`risk_acceptance`** — reviewed desired policy, not local observation state. It binds a required
  reason and approval time to the exact skill digest and exact finding fingerprints (`rule_id` +
  rule-definition digest + severity), with optional expiry. The same unchanged findings remain
  accepted across ruleset-version updates; changed content, a new or changed finding, increased
  severity, or expiry invalidates acceptance. Exact accepted findings pass CI with warnings. The
  approval UI warns that `reason` is checked-in team-visible text, applies length and control-
  character bounds, and tells the approver not to include secrets; no reason is copied to
  telemetry. Skulto does not claim that heuristic secret detection can make arbitrary prose safe.

Invocation observations are deliberately absent from `skulto.json`. They are volatile,
machine-local, and adapter-specific, so `internal/observe` recomputes them from local transcripts or
caches bounded derived events in local SQLite. The cache never enters Git, manifest equality,
history, telemetry, or artifact approval. Absence means `not observed`, never `did not influence
the session`. Observations inform pruning proposals; they never perform them.

Budget status is platform-specific: `verified` when the installed agent exposes a stable,
machine-readable limit or a supported adapter confirms it empirically; `estimated` when a
versioned Skulto registry supplies a reverse-engineered limit with named evidence; and `unknown`
when neither source is defensible. The fallback registry is embedded in the Skulto binary, versioned
with the release, tied to tested adapter and agent-version ranges, and covered by adapter contract
tests. Each entry records its evidence citation and measurement semantics. It has no independent
network refresh path and is not part of the remote security-rules feed; changing it requires a
Skulto release. Resolution always probes the installed agent first, then uses the embedded registry
only as fallback. An installed version outside an entry's tested range does not inherit the nearest
estimate; it becomes `unknown`. Skulto never scrapes human-oriented help text as a verified source.

Dynamic discovery is an adapter-owned, fail-closed operation. Each supported adapter defines an
exact allowlist of non-interactive version and machine-readable capability invocations, their
expected output schema, timeout, and supported version range. Probes run with stdin closed and must
not launch an agent session, load repository instructions, execute project hooks, accept arbitrary
arguments, or construct commands from manifest or agent output. Unexpected output, timeout,
unsupported versions, or inability to suppress project startup behavior makes dynamic discovery
unavailable; resolution falls back to the registry estimate or `unknown`. Human-oriented `help`
text and free-form terminal output are never parsed into a `verified` budget.

Adapters resolve an absolute executable once, invoke it directly without a shell, and run
capability probes from a private empty directory with a minimal allowlisted environment. Output and
execution time are bounded. If the agent cannot report capabilities without loading user or
project startup configuration, that dynamic method is unsupported rather than weakened.

The budget block is the human-approved composition snapshot, not a user-editable authority or a
live machine-state cache. It records the installed agent version, discovery kind and method,
resolved limit when known, and measured use at approval time so the decision is explainable and
reproducible. `compose`, `compose validate`, `compose apply`, and `doctor` re-probe the installed
agent, but probing and comparison are read-only: they never rewrite `skulto.json`, increment its
revision, or silently bless new observations. An agent-version change forces a fresh probe but does
not by itself invalidate the environment. A changed effective limit or measurement method, or
inability to verify the prior result, marks the approved snapshot stale and requires a new
composition proposal before apply; a lockfile value never overrides the current agent. Only human
approval of that proposal may replace the budget snapshot in `skulto.json`.

Enforcement follows evidence strength:

- **`verified`** — a hard constraint. Overflow rejects the composition; approval cannot convert a
  known truncation into a fit claim.
- **`estimated`** — a planning constraint, always labeled **`budget estimated`**. A composition
  within the estimate may proceed through the ordinary approval flow but is never reported as
  verified. Overflow requires a specific human override in the exact diff and remains an estimated
  overrun in the manifest and `doctor` output.
- **`unknown`** — no fit decision is possible. Installation may proceed only after specific human
  confirmation of **`budget unverified`**; the status remains in the manifest and `doctor` output.

No aggregate environment is described as budget-verified unless every selected platform has a
verified, non-overflowing budget observation.

Migration: schema-1 manifests (`{version, skills: {slug: repo}, ignored: []}`) never silently pin
whatever upstream `HEAD` happens to contain. Migration first produces a read-only preview and
backup. For each installed skill, it may propose an exact ref only when the currently installed
bytes, clean source tree, and resolved commit produce the same canonical digest; otherwise the
entry becomes unresolved and names the required human action. The schema-2 file is written only
through the normal approval protocol. Unknown schema versions fail closed without rewriting the
file; downgrade after approval requires explicit restoration of the backup and is never automatic.

### 5.2 `.skulto/context.md`

```markdown
<!-- skulto:context v1 — managed by skulto. Routing points; facts must not be repo-derivable. -->

## Routing
- Before changing telemetry, read `context/telemetry.md`.
  <!-- skulto id=route-001 kind=routing target=context/telemetry.md source=manual added=2026-07-12 -->

## Undocumented constraints
- The staging maintenance window ends at 15:00 UTC; start deploys early enough to preserve the
  rollback window. <!-- illustrative format only; not a claim about this repository -->
  <!-- skulto id=fact-001 kind=fact source=interview evidence="operational policy is not recorded in the repository" added=2026-07-12 -->
```

Every entry carries `id`, `kind`, `source` (`interview` | `correction` | `manual`), and dates.
Routing entries carry a repository-relative `target` and may not restate its claim. Fact entries
carry evidence explaining why the fact is not repository-derivable. Human-approved entry-level
deltas may add, amend, deprecate, or remove entries; whole-file regeneration is forbidden.

The managed-comment grammar is versioned and strict. IDs, enums, dates, paths, and evidence have
documented length and character bounds; metadata is canonically escaped, and comment terminators,
control characters, duplicate IDs, unknown fields, and malformed entries are rejected. Routing
targets are canonical paths contained in the same worktree, resolve to existing regular files
without crossing symlinks, and are rechecked by lint and apply. Facts carry `reviewed_at` and may
carry `review_after`; overdue review is a warning and proposal input, never automatic deletion.

**`skulto context lint` is partly deterministic, and that is the point.** For each fact, grep the
repo for its claim: if the fact says *"run tests with `make test`"* and `Makefile` contains a
`test:` target, **the fact is discoverable and is flagged for deletion.** Cheap, exact, and it
catches the 85% case. The fuzzy residue goes to the agent to judge and the human to approve.

Lint reports evidence, method, and uncertainty. Exact duplicate/broken-path checks are
deterministic; fuzzy or agent classification is labeled advisory and never represented as proof
that a fact is undiscoverable.

**`skulto context add --kind fact` requires `--evidence`**, a stated reason the agent cannot derive
the fact. Empty evidence is a hard rejection. `--kind routing` instead requires a valid
repository-relative `--target` and rejects paraphrased claims. **The gate is the product.**

### 5.3 `.skulto/history.jsonl`

`skulto.json` remains a clean description of current desired state. Approved project-artifact
mutations append one canonical JSON event to `.skulto/history.jsonl`:

```json
{"event_id":"evt-01","timestamp":"2026-07-12T18:22:04Z","operation":"context.amend","entry_id":"fact-001","before_digest":"sha256:...","after_digest":"sha256:...","proposal_digest":"sha256:...","manifest_revision":7,"approval":"tty","previous_event_digest":"sha256:...","event_digest":"sha256:..."}
```

The log contains metadata only: event and entry IDs, operation enum, before/after digests, proposal
digest, manifest revision, timestamp, and approval channel (`tty` | `mcp_elicitation`). It never
contains previous fact text, transcript content, queries, free-form errors, user identity, or local
paths. It is append-only, checked into Git, never loaded into agent context, and therefore costs no
context budget. Apply does not report success unless both the intended mutation and its history
event are durable; crash recovery completes or rolls back an interrupted pair.

Events form a domain-separated hash chain. `event_digest` is SHA-256 over
`skulto-history-event-v1\0` plus the RFC 8785 canonical JSON bytes of the event with
`event_digest` omitted; `previous_event_digest` links the prior event and `skulto.json.history_head`
links the approved state to the tail. The first event uses `null`. Timestamps are informational
wall-clock claims, not trusted ordering; revision and the hash chain establish order. A mismatch is
a loud integrity failure and is never repaired by silently rewriting history.

### 5.4 Canonical structured data

Manifests, proposals, approval payloads, history-event hashing, signed ruleset manifests, and
machine-readable agent results use strict I-JSON and RFC 8785 JSON Canonicalization Scheme bytes
for hashing or signing. Parsers reject duplicate property names, invalid Unicode, non-integer or
out-of-range values where integers are required, unknown schema versions or fields, trailing
tokens, and data beyond declared size/depth limits. Schema-2 security objects reject all unknown
fields; forward compatibility requires a new schema rather than silent
interpretation differences. Producers emit one canonical representation; consumers never hash a
reinterpreted or partially parsed document.

Every cryptographic digest or signature has a format-specific domain-separation prefix and names
its algorithm. Proposal and event digests are lowercase `sha256:<hex>`; Ed25519 signatures and key
IDs use one specified base64url-without-padding encoding. Golden vectors cover each format.

### 5.5 The security ruleset

```yaml
version: 2026-07-01
rules:
  - id: SKL-0012
    category: instruction_override
    severity: high
    pattern: '(?i)ignore\s+(all\s+)?previous\s+instructions'
    mitigations: [educational_context, quoted_example]
    added: 2026-03-11
    references: ["https://snyk.io/blog/toxicskills-malicious-ai-agent-skills-clawhub/"]
```

Rules stop being regex compiled into the binary and become **data**.

- **A baseline ruleset and its signed manifest ship embedded** (`go:embed`). Build and runtime
  verification use the same trust store and digest path as remote releases. Offline audit works on
  day one without silently weakening provenance.
- **The feed is a git repo** (`asteroid-belt/skulto-rules`), pulled through `internal/scraper` —
  the clone/pull machinery that already exists and is already tested. **No backend or accounts.**
- **Every accepted remote ruleset has an Ed25519-signed release manifest.** The offline release
  private key stays outside ordinary GitHub write access; its public verification key ships in the
  binary. RFC 8785 canonical signed metadata contains a schema, strictly increasing unsigned
  sequence, display version, SHA-256 content digest, creation and expiry times, source commit,
  algorithm, and key ID. The signature covers the domain-separated metadata bytes, not a mutable
  branch or tag name. The ruleset content digest uses canonical artifact digest v1 over the release
  payload with the signed manifest excluded, so the manifest does not recursively hash itself.
- **Rollback, freeze, and mix-and-match are explicit states.** Skulto persists the highest accepted
  sequence and its digest, rejects any lower sequence or same-sequence content change, verifies all
  referenced bytes as one release, and labels expired last-known-good metadata `ruleset stale`.
  Offline audit remains available with the embedded baseline or verified last-known-good content,
  but never calls stale content current. Deleting or modifying local anti-rollback state is within
  the same-user limitation in §4.1. Anti-rollback metadata is atomically stored `0600` beneath the
  `0700` data root; missing or corrupt state disables remote update acceptance until explicit
  repair rather than resetting the highest sequence to zero. Wall-clock rollback can affect expiry
  warnings and is covered by the same local-host limitation.
- **Key rotation is release-mediated in v2.** The binary trust store carries identified current and
  next public keys; accepting a new signing key or revoking a compromised one requires a Skulto
  release. The design explicitly accepts the residual single-release-key compromise risk; full TUF
  threshold roles and Sigstore transparency remain deferred. Any signature, schema, expiry,
  sequence, digest, key, or update failure leaves the previous verified ruleset untouched.
  Protected `main`, required review, CODEOWNERS, no force-pushes, and maintainer 2FA are defense in
  depth, not the trust root.
- **Rules are constrained data, not executable policy.** YAML is parsed in strict mode with
  duplicate-key and unknown-field rejection. Rule IDs are unique and immutable; categories,
  severities, regex length, rule count, and total file size are bounded. Patterns compile through
  Go's RE2-class `regexp` engine; rules cannot run code, read files, fetch references, or invoke
  network services. The entire candidate release validates and compiles before it can replace the
  last-known-good ruleset.
- **Coverage is part of the verdict.** The scanner inventories every canonical artifact file before
  applying rules and records its declared scope, file and byte counts, and complete/incomplete
  coverage. Unsupported binary/archive content, scanner size limits, read errors, or skipped files
  create stable synthetic findings bound to the affected file digest; they can never yield
  `passed`. New install or upgrade fails closed unless the exact incomplete-coverage findings are
  explicitly risk-accepted. Existing active skills remain installed but are flagged under
  invariant 9.
- **Determinism is a tested property:** same skill digest + same ruleset digest + same scanner
  version ⇒ byte-identical verdict and score. Golden tests enforce it. Today's scanner has
  context-mitigation scoring that is effectively unfalsifiable; v2 makes every deduction an
  explicit, named, tested rule.

---

## 6. Architecture

### 6.1 The spine: skulto ships deterministic tools; the user's agent supplies the intelligence

Compose needs an LLM — to turn a PRD into search queries, to run the discoverability test, to
classify a correction. `internal/llm/` exists, fully built (Anthropic/OpenAI/OpenRouter, streaming,
tests) and **completely unimported**.

**Leave it dead. Delete it.** Skulto exposes deterministic primitives and orchestrates a supported,
locally installed agent CLI when intelligence is required. If the user is already in an agent
session, the shipped skill teaches that agent to drive the primitives directly. The agent proposes;
Skulto probes, counts, scans, searches, validates, records approval, and writes.

This buys:

- **No new Skulto-managed model credential.** Deterministic search, pinning, audit, budget
  arithmetic, and cached exact-artifact use remain offline-capable. Compose availability follows
  the selected agent CLI and may require that vendor's account, credentials, and network. Compose
  uses deterministic FTS5 plus iterative query reformulation by the installed agent; it does not
  depend on the existing `OPENAI_API_KEY`-gated semantic-search path. That path remains an
  explicitly configured, grandfathered TUI-only compatibility feature: it is never enabled
  implicitly, expanded to a new surface, or used as a fallback by Compose.
- **No Skulto-operated inference service or bill.** Agent CLI usage remains visible to and paid by
  the user under that agent's own account or subscription.
- **The interview happens where the user already lives**, conversationally, with the repo loaded.
- **No agent assertion becomes a Skulto guarantee.** Proposals and rationales may be wrong; budget,
  digest, ruleset, manifest, and apply results are deterministic and independently validated.

Bare `skulto compose` is the terminal-first entry point. It launches a tested Claude, Codex, or
OpenCode adapter as an interactive TTY session in repository-read-only mode. The adapter may read
the repo and run only allowlisted non-mutating Skulto commands; it returns a versioned, tagged,
schema-constrained proposal through a controlled result channel. Skulto regains the terminal,
validates the proposal, and runs its native approval flow. Unsupported tools or versions fail
closed if read-only execution and a reliable result channel cannot be enforced. The process and
cancellation plumbing in `internal/skillgen/executor.go` survives; its prompt and permissive command
builders do not.

Each adapter resolves and displays the exact executable path and version, invokes it directly
without a shell, passes only an adapter-owned environment allowlist plus credentials the selected
CLI requires, and enables the vendor's strongest tested read-only/sandbox controls. Prompts,
repository text, and the prompt tag are not security boundaries. Agent results are untrusted,
strictly parsed from the dedicated bounded channel, schema- and size-checked, and then
deterministically recomputed where possible. Ordinary terminal chatter is never treated as a
proposal. Timeout, cancellation, terminal loss, multiple results, malformed output, output-limit
overflow, or a surviving child process fails the composition and cleans up the process group and
temporary state without changing project artifacts. Skulto does not copy agent output into logs or
telemetry.

When the user is already inside a coding agent, the skill uses `skulto compose prepare --json`,
read-only search/probe/budget/audit commands, `skulto compose validate <proposal>`, and the native
approval/apply protocol. It never invokes bare `skulto compose`, so it never launches a nested
agent. In this path, Skulto can enforce that its own commands are non-mutating but cannot attest
that the pre-existing agent session is repository-read-only; that session retains whatever shell
and filesystem authority the user already granted it. The proposal/approval protocol protects
Skulto-managed mutations, not unrelated writes by the host agent.

### 6.2 Surfaces: capability-gated, not parity-gated

Claude Desktop and Codex Desktop have **no project cwd, repo shell, or coding-session transcripts.**
They *cannot* compose (it probes a working tree), *cannot* audit a project environment, and
*cannot* report local coding-agent observations. A locally running MCP server can still browse,
read, and install globally after MCP elicitation approval.

| Job | Needs repo? | Needs shell? | Desktop? | Surface |
|---|---|---|---|---|
| search, read a skill, install it | no | no | **yes** | **MCP** + CLI |
| compose an environment | yes | yes | no | CLI + skill |
| audit against ruleset | yes | yes | no | CLI + skill + CI |
| invocation / correction reports | yes | yes | no | CLI + skill |
| `context.md` entry mutation | yes | yes | no | CLI + skill |
| favourites, recents, stats, tags | no | — | — | **TUI only** — browsing affordances, not agent capabilities |

**MCP v2: three tools.** `skulto_search`, `skulto_get_skill`, `skulto_install`. **~300 tokens,
down from ~1,200.** Nine tools removed. `skulto_install` is mutation-capability-gated: it applies
only after the client accepts a server-initiated elicitation containing the exact proposal diff.
Without elicitation support it returns `installation unavailable: client cannot obtain explicit
human approval` and performs no write.

MCP install arguments never accept an arbitrary destination, local path, shell command, or
unvalidated URL. The server resolves a typed source identity to an exact artifact, scans it, and
may target only registered platform roots owned by the installer. Canonical path containment and
no-follow checks run both when proposing and immediately before applying. Search/get results and
skill metadata are untrusted data, schema- and size-bounded, and escaped before display. MCP client
identity or prior consent never grants ambient filesystem authority.

**The membership rule, which goes in the ADR:** *a tool earns permanent MCP residency only if
(1) the environment cannot reach the CLI, and (2) the need arises spontaneously mid-conversation.*
Search passes both. `skulto_get_stats` passes neither, and never did.

**You never pay for both.** In a coding agent: CLI + one skill, **~120 tokens**, MCP not connected.
In Desktop: MCP + the same skill, **~420 tokens**, and nothing else is competing for that budget.
Connecting both is a misconfiguration and `skulto doctor` says so out loud.

### 6.3 One shipped skill, two bodies

**Exactly one skill: `skulto`.** ~120 tokens of tier-1 residency. Auto-installed on
`skulto init` into `.claude/skills/skulto/` — the specific entry skulto owns. Skulto does not own
the parent skills directory or neighboring workspace skills. Skills self-register by existing on
disk; **no config file is touched for this.**

Shipping five skills to teach one CLI would recreate the 1,200-token MCP server in a different
file format, and the product would die of hypocrisy.

The body — loaded on demand only — branches by environment:

```
## If you have a shell
If already inside an agent session, run `skulto compose prepare --json`, the read-only analysis
commands, and `skulto compose validate <proposal>`. Stop for native human approval before apply.

## If you have skulto MCP tools (Desktop)
Call skulto_search / skulto_get_skill / skulto_install.
Compose, audit, and reports are unavailable here — they need a repo.
```

Tier-1 cost is unchanged, because tier 1 is only the name and description. The branching lives in
tier 2, where it is free until it fires.

### 6.4 `skulto init`, and the only two files skulto touches that it does not own

`skulto init` does three things and asks before each:

1. **Detects the surface.** Coding agent with a shell → CLI + skill. Desktop → MCP + skill.
   Both connected → warn; that is a misconfiguration.
2. **Installs the `skulto` skill** as the platform-specific `skulto` entry. Skulto owns that entry,
   not the platform's parent skills directory or neighboring workspace skills. Skills self-register
   by existing on disk. **No config file is touched for this.**
3. **Offers to add the platform's minimal read-only `skulto` command patterns to its settings
   allowlist.** Mutating commands are never allowlisted by `skulto init`.

Step 3 is the one real concession, and it is necessary: without narrow allowlist entries, **every
read-only `skulto` invocation prompts the user**, which is fatal to the agent-driven analysis loop.
The rules are strict, and they are the operational form of invariants 2 and 5:

- **Opt-in.** `skulto init` prints the exact diff and asks. `--dry-run` shows it without writing.
- **Read-only commands only.** Probe, search, audit reporting, budget checks, reports, proposal
  generation, and proposal validation may be allowlisted. Install, apply, context mutation, risk
  acceptance, migration, and any other write may not.
- **Additive JSON key merge.** Only the minimum platform-specific patterns are appended to the
  allowlist array. **Never a rewrite. Never prose.**
- **Idempotent and transactional.** The original bytes and mode live only in the private `0700`
  recovery journal while the operation is in flight; successful completion removes that snapshot
  rather than leaving a second long-lived copy of a potentially sensitive settings file.
- If the settings file is malformed or unparseable, skulto **declines to write** and prints the
  entries for the user to paste. It does not guess.

Settings and `AGENTS.md` mutations accept only expected regular files beneath their resolved
project or platform root, reject symlinks and path races, preserve unrelated bytes, permissions,
and existing newline style, and use no-follow atomic replacement. User-controlled content is
escaped before terminal display. If safe additive editing or rollback cannot be guaranteed,
Skulto prints the proposed snippet and performs no write.

The second file is `AGENTS.md`, which receives **one delimited, idempotent reference line**
pointing at `.skulto/context.md` — also opt-in, also `--dry-run`able. Skulto writes a pointer.
**Skulto never writes prose into a file it does not own.**

### 6.5 The proposal and approval protocol

Every potentially mutating operation first produces a schema-versioned canonical proposal. It
binds a random nonce, creation and expiry times, an exact mutation-scope identity (repository when
present, otherwise the local installation scope), current Git HEAD and worktree state where
relevant, current manifest revision and history head, every target's canonical path, type, mode,
and preimage digest, intended writes, source content digests, budget snapshot, scanner and ruleset
identity, and the complete logical diff. Proposal identity is SHA-256 over
`skulto-proposal-v1\0` plus its RFC 8785 canonical JSON bytes. Read-only agent commands may emit
and validate this proposal, but they cannot apply it.

Proposal generation and validation reject paths outside declared roots, path aliases, symlink
targets, duplicate destinations, unsupported file types, unknown fields, oversized output, and
state that changed during inspection. Agent-provided strings never become command arguments or
filesystem paths except through typed allowlists and canonical validation. Validated proposals are
stored only in bounded `0600` local state beneath the `0700` data root, expire with their declared
TTL, and never enter the repository, telemetry, or general logs.

`skulto approve <proposal-digest>` runs through Skulto's native interactive flow, renders the exact
diff, and requires explicit confirmation from the controlling terminal rather than stdin. All
untrusted text is rendered as escaped data with terminal control sequences disabled; security-
relevant fields and writes are never truncated. If the complete review cannot be rendered safely,
approval fails rather than substituting a summary.

Approval creates a cryptographically random, maximum-five-minute, single-use receipt bound to the
exact proposal, mutation-scope identity, and approving channel. It lives only in `0700` local
state with `0600` permissions, never in the repository, logs, history, or telemetry. `apply`
revalidates all bound state, rejects any digest mismatch or expired/consumed receipt, and consumes
the receipt immediately before the first mutation. Redirected stdin, a generic `--yes`, a platform
allowlist, or an agent's claim that the user approved never substitutes for this receipt. The
same-user attacker limitation in §4.1 applies to local approval state.

Apply acquires a mutation-scope exclusive lock before final validation. It writes a durable
local transaction journal, stages project-file replacements beside their destinations, fsyncs
content, uses no-follow compare-and-swap replacement, updates the history chain and manifest head,
where applicable, and records local installation effects idempotently. Cross-filesystem effects cannot be one atomic
rename, so the journal defines deterministic roll-forward or rollback and remains until recovery
finishes. A concurrent revision, Git state, file preimage, platform target, ruleset, budget, or
artifact change invalidates approval. Recovery may complete the already-approved transaction but
may not broaden it or overwrite a post-crash external edit whose preimage no longer matches. Such a
conflict stops in a recoverable manual-review state. Partial application is reported as
`recovery required`, never success.

Interactive terminal Compose performs approval directly after its read-only agent subprocess
returns. An already-running agent stops at a validated proposal and asks the human to run the
native approval flow. CI and other noninteractive environments may audit and report, but cannot
mutate project artifacts.

MCP Desktop is the deliberate exception to the native-terminal mechanism, not to human approval.
For `skulto_install`, the server first builds the canonical proposal and then uses MCP elicitation
to show its exact source, pin, digest, scan result, target platform, paths, and writes. An explicit
client `accept` response authorizes only that in-flight proposal; the server immediately revalidates
and applies it in the same request and never persists a reusable MCP approval. `decline`, `cancel`,
disconnect, timeout, or changed state are no-ops. Elicitation fields are structured and bounded,
and untrusted strings are escaped so skill metadata cannot spoof approval UI. Tool annotations are
also set conservatively for client UX, but annotations alone never count as approval because the
MCP specification defines them as untrusted hints rather than enforcement. Clients without
elicitation retain `skulto_search` and `skulto_get_skill`; installation is unavailable.
Global MCP installs have no project history file; their exact artifact digest, target, approval
channel, and transaction outcome are retained only in the permissioned local installation record.

### 6.6 New packages

| Package | Responsibility | Depends on |
|---|---|---|
| `internal/probe/` | Deterministic stack fingerprint of cwd (`package.json`, `go.mod`, lockfiles, framework markers) | — |
| `internal/budget/` | Dynamic agent-budget discovery, release-embedded evidence-backed fallback registry, and description-cost accounting. Records provenance, detects stale observations, costs a candidate set, and flags overflow. | `installer`, agent adapters |
| `internal/artifact/` | Locked-artifact identity; exact-ref resolution; canonical directory hashing; atomic write-protected cache materialisation and trust-boundary verification | `scraper` |
| `internal/environment/` | Manifest schema 2: model, compose/diff/apply, pinning, and scan state | `db`, `installer`, `artifact`, `budget`, `security` |
| `internal/observe/` | `Adapter` interface over agent transcripts → `InvocationEvent`, `CorrectionSpan`; bounded local SQLite cache. One impl (`claudecode`); 32 stubs. | `db` |
| `internal/security/ruleset/` | Rules as data: versioned, ID'd, severity-tagged. Loader, version pinning, golden tests. | `scraper` |

### 6.7 New CLI verbs

Each with `--json`. This is the agent-facing contract.

`skulto init` · `skulto compose` · `skulto approve` · `skulto apply` · `skulto audit` ·
`skulto context add|lint|list` · `skulto report invocations|corrections` ·
`skulto data purge --observations` · `skulto doctor`

JSON output has a command-specific schema version, stable enumerated error code, and deterministic
stdout contract; human diagnostics go to stderr. Secret, transcript, query, agent-output, and
free-form error content is excluded from both channels unless an explicit local export command is
being reviewed by the user.

### 6.8 Component flow

```
                    ┌─────────── the user's agent (the LLM) ───────────┐
                    │  drives everything below via the skulto skill    │
                    └──┬──────────────┬──────────────┬────────────────┬┘
                       │              │              │                │
                  ┌────▼────┐   ┌─────▼─────┐  ┌─────▼─────┐   ┌──────▼──────┐
   COMPOSE        │  probe  │   │  search   │  │  budget   │   │ environment │
                  └─────────┘   └───────────┘  └───────────┘   └──────┬──────┘
                   stack           corpus        does it fit?          │
                   fingerprint     (FTS5)        evidence status       │
                                                                       │
   VERIFY         ┌──────────────┐                              writes │
                  │   ruleset    │──── scan ──────────────────────────►│
                  │  (git feed)  │     pins ruleset version            │
                  └──────────────┘                                     │
                                                                       ▼
   EVOLVE         ┌──────────────┐──── invocations ─────► local SQLite (derived cache)
                  │   observe    │──── corrections ─────► .skulto/context.md
                  │ (transcripts)│
                  └──────────────┘   approved project mutations ─────► .skulto/history.jsonl
```

Every arrow into an artifact passes a human approval gate.

---

## 7. The observe pipeline

```go
type Adapter interface {
    Name() string
    Available() bool
    Sessions(projectPath string) ([]Session, error)
}

type Session struct {
    ID               string
    SkillInvocations []SkillInvocation // slug, timestamp
    Corrections      []CorrectionSpan  // user turn immediately following an assistant tool-use turn
    ToolFailures     []ToolFailure     // command, exit code, and the later command that succeeded
}
```

MVP ships one implementation: **`claudecode`**, reading `~/.claude/projects/<slug>/*.jsonl`.
Thirty-two stubs return `Available() == false`.

Transcript access is on-demand only: an explicit report command or approved correction-analysis
step names the adapter and paths it will read. There is no watcher, background upload, startup
scan, or implicit cross-project crawl. Parsers open expected regular files read-only without
following links and enforce file, record, nesting, and time bounds. A malformed or oversized
record degrades that source rather than causing partial trusted output.

- **`skulto report invocations`** — per installed skill: supported sessions observed, explicit
  invocations found, and last observed invocation. **A skill with no explicit invocation across the
  last 10 supported sessions is surfaced as unobserved** (threshold configurable via
  `--min-sessions`; 10 is a default, not a finding). Unobserved skills are *reported*, never
  auto-pruned — invariant 5. Skulto does not infer that the skill had no influence or was useless.
- **`skulto report corrections`** — skulto **deterministically extracts** candidate spans (a user
  turn following a tool-use turn containing correction markers; a failed command followed by a
  successful variant). **The agent classifies** whether each is a durable, non-discoverable fact.
  **The human approves.** Only then does it become a `context.md` delta. Three gates; skulto's own
  contribution is the boring deterministic one.

Raw transcript text and correction spans exist only in memory for the explicit operation and are
released on completion; they are never persisted in Skulto's SQLite cache. The cache stores only
bounded derived invocation facts, coarse counters, adapter version, and opaque local session
fingerprints needed for deduplication. It is rebuildable, permissioned `0600` beneath a `0700` data
directory, retained for at most 30 days or 100 sessions per project (whichever is smaller), and
removable with `skulto data purge --observations`. Purge is local-state deletion, never a telemetry
event containing affected identifiers.

### 7.1 Why the agent is never instructed to report anything

Supported Claude Code versions already write the events used by the adapter to local session logs.
**For those versions, the data is there whether the agent follows a Skulto instruction or not.**
Putting *"always run `skulto track`"* into `AGENTS.md` would instead give us:

- **A permanent context tax** — the bloat this product sells a cure for.
- **Unreliability by construction** — passive instruction adherence runs 30–50%; IFScale shows
  adherence decaying with instruction count; the loudest complaint in the demand research is
  *"Claude regularly ignores explicit instructions in CLAUDE.md."*
- **Selection bias in our own data** — we would observe only the sessions where the agent complied.
  The trigger-rate report, the product's central execution measurement, would be measuring its own
  instruction-following rather than the skills'.
- **The spec-kit tombstone** — `update-agent-context.sh` overwrote and destroyed users' CLAUDE.md
  files (github/spec-kit #365, #596, #309).

**Instructions are a terrible substrate for telemetry. Existing local logs are a useful,
adapter-specific observation source when their format is supported.**

### 7.2 Privacy

Transcript content **never leaves the machine**. `internal/telemetry/` (PostHog) must never emit
transcript text, correction spans, file paths, skill bodies, or search query content from CLI, TUI,
MCP, Compose, or any future surface. Aggregate counts only. **Enforced by test.** Search telemetry
may include bounded aggregates such as query count, query-length bucket, retrieval type, latency
bucket, result count, zero-result status, reformulation count, selected-result rank, and conversion.
It may not emit query text, hashes, fragments, tokens, samples, or free-form errors containing a
query. Exact failed-query diagnostics remain local and may leave the machine only through an
explicit, user-reviewed export. One leaked correction span or query ends the trust product.

Identifiers follow a public-only rule. A repository or skill identifier may be emitted only when
GitHub visibility metadata positively establishes that its source is public. `unknown` fails
closed and is treated like `private`; local skills never emit names or slugs. Private, local, and
unknown items emit only a coarse `origin` enum plus bounded counts and categories. Raw `os.Args`,
agent output, stderr, and free-form error messages have no trustworthy public provenance and never
enter telemetry; enumerated command names, flag-presence booleans, and error types replace them.
Non-GitHub providers remain `unknown` until a provider-specific visibility verifier exists and is
covered by the same fail-closed tests.

Telemetry uses per-event property allowlists rather than generic map forwarding. Visibility must
be positively established from a bounded-fresh GitHub observation at emission time; stale cache,
rate limit, network failure, repository transfer, deletion, or any ambiguity degrades to the coarse
private/unknown shape. Crash reporting, logs, and metrics do not bypass these rules. Canary tests
exercise nested structures, errors, cancellation, malformed agent output, and every interface.

Local logs are `0600`, rotate within a documented bound, and omit transcript bodies, skill bodies,
queries, credentials, agent output, raw arguments, and environment values by default. Exact local
diagnostics require an explicit debug/export action that previews included files and redactions;
they are never attached or transmitted automatically.

### 7.3 Fragility

The JSONL format is undocumented and will change without notice. The adapter **fails soft**: a
parse error degrades to *"observability unavailable for this Claude Code version"* — a loud but
harmless warning. It never crashes, never blocks compose, never blocks audit. Golden fixtures per
known format version. Observability is a bonus layer, never a dependency.

---

## 8. Phasing

**Phase 0 — Required prerequisite: unified local skill storage.** Phase 0 is an exit gate, not a
parallel workstream. **No Phase 1–4 implementation begins until the accepted
[`2026-04-19-unify-local-skill-storage`](2026-04-19-unify-local-skill-storage-design.md) spec is
fully landed and its certification passes.** It is a showstopper because v2 pinning, auditing, and
environment application cannot be trustworthy while a local skill may resolve to a file or to an
arbitrary project directory.

Phase 0 has two sequential hard gates. Gate 0A (unified local storage) is complete only when:

- every `IsLocal=true` skill stores a directory path under `~/.agents/skulto/`;
- every symlink produced by `skulto install` resolves to a directory under
  `~/.agents/skulto/`;
- legacy `cwd-*` rows and unsafe symlinks have been removed by the idempotent one-shot migration;
- project-scoped `local-*` skills have been moved into canonical storage, with collision and crash
  recovery behavior covered by tests;
- the cwd skill scanner and project-scoped ingest path have been removed; and
- the local-storage acceptance test and release certification pass.

After Gate 0A passes, Gate 0B lands manifest schema 2 (`schema`, `revision`, immutable identities,
migration) and the locked-artifact boundary. Exact source ref and path are
resolved into a canonical digest over the full skill directory; the verified bytes are atomically
materialised under `~/.agents/skulto/objects/sha256/<digest>/`; environment-managed platform
symlinks point only to freshly verified, write-protected cache objects. The installer receives a
materialised artifact rather than deriving a path into a mutable repository checkout. Interrupted
materialisation is recoverable and digest mismatches fail closed. A freshly verified cached object
is sufficient offline; an unavailable upstream ref fails loudly only when the exact object is
absent or corrupt. Garbage collection of unreferenced objects is deferred until after v2, while
`doctor` reports object count and disk usage so growth is visible.

Gate 0B passes only when exact-ref retrieval, canonical whole-directory hashing, atomic
materialisation, digest verification, object-backed platform symlinks, installation-record
identity, schema-1 migration, interruption recovery, unavailable-upstream failure behavior,
cross-platform digest golden vectors, path-collision rejection, and symlink/special-file rejection
are covered by unit and acceptance tests. Phase 0 then deletes `internal/llm/` (dead code and a
standing temptation) and retains only the agent-CLI execution plumbing from `internal/skillgen/`.
**No Phase 1–4 implementation begins until both Phase 0 gates pass.**

Workspace skills are the deliberate exception to remote commit resolution, not to reproducibility.
If Compose encounters a global or otherwise local-only skill, it reports the skill as unresolved
and asks the human to copy its actual directory into the same Git worktree as `skulto.json`—for
example `.claude/skills/<slug>`—commit it, and rerun Compose. Skulto does not perform that copy or
commit. It accepts the skill only when the directory is tracked and clean at `HEAD`, then records
its repository-relative path and canonical digest. The workspace directory remains the checked-in
source; Skulto materialises the same digest into the object store for other platform installations.

Any Git-tracked skill already living in a selected platform's native project discovery directory
is a mandatory, non-prunable baseline for that platform—for example every tracked directory under
`.claude/skills/` when Claude is selected. Compose discovers and audits all of them, records them as
workspace lock entries, and includes their tier-1 descriptions in budget arithmetic whether or not
the user originally named them. A recognized Skulto-managed platform symlink is validated through
its installation record and object digest rather than mistaken for a workspace source; any other
symlink is rejected. Skulto's own shipped resident skill is counted separately as fixed platform
overhead. If the mandatory baseline alone exceeds a verified budget or contains an unaccepted
audit-failed finding, Compose cannot repair that state automatically: it names the exact entries
and asks the human to remove, move, fix, or explicitly accept the eligible findings in the
repository, then rerun Compose. An estimated overrun follows the explicit override rule in §5.1;
an unknown limit retains `budget unverified`.

**Phase 1 — VERIFY. Ships first.** Ruleset-as-data, signed git feed, `skulto audit`, the CI action,
audit-failed flagging, explicit risk acceptance, and determinism golden tests.

CI verifies the manifest's exact artifact and signed ruleset digests; it never silently substitutes
`latest`, an embedded baseline, or a different last-known-good version for a pinned result. If exact
verified bytes are unavailable, the result is `verification unavailable` with a distinct nonzero
exit code. Unaccepted findings fail, exact unexpired risk acceptances pass with warnings, and stale
or estimated evidence is labeled in both human and schema-versioned JSON output. CI remains
read-only and cannot create approval receipts, risk acceptances, lockfile changes, or repairs.

Compose is the exciting capability. Verify ships first anyway:

- It attacks the **#3 strongest evidenced pain**. Compose attacks a pain with **zero organic demand
  evidence**.
- It is **half-built** and depends on nothing else.
- It ships standalone as a v1.5 capability that does not require Compose.
- **It is the credibility wedge.** "The skill manager that audits your supply chain" earns the right
  to then say "…and here is how it composes your environment." The reverse order asks strangers to
  trust an unevidenced thesis.

**Phase 2 — COMPOSE.** `probe`, `budget`, the compose flow, the one shipped skill, `context.md` and
the discoverability lint.

**Phase 2.5 — EVAL. A hard gate, implemented outside the product.** §9. **No public claim about
Compose is made before this passes.** The gate is a repository-owned, developer-only harness under
`eval/`, not a `skulto eval` command and not a runtime dependency of either shipped binary. There
is no evidenced customer demand for self-evaluation, so productising the harness would be premature.

The harness shells supported, locally installed agent CLIs to execute tasks in isolated disposable
checkouts. Agents may write and run commands only inside those checkouts. The harness—not the
agent—assigns outcomes using each benchmark task's deterministic test oracle. It records the
dataset revision, Skulto revision, agent CLI and model version, environment condition, prompt,
OS/toolchain image, dependency locks, random seed where exposed, network policy, limits, run order,
token use, tool calls, timing, exit state, and oracle result. Conditions use the same agent settings
and limits; their order is randomised within task; malformed, timed-out, refused, and failed runs
remain in the intention-to-treat record rather than being retried until they pass. Every run
requires explicit cost disclosure and hard token, time, output, and iteration caps.

Development tasks and retrieval tuning data are disjoint from the held-out gate set. The Compose
implementation, condition prompts, adapters, tokenizer/accounting code, exclusions, and analysis
script are frozen under a signed preregistration tag before gate outcomes are inspected. The power
analysis determines the number of paired tasks and independent repetitions; early stopping,
optional exclusions, and swapping failed repositories are forbidden unless preregistered. All
attempts and negative results remain in the report.

Definition-token reduction is measured from the actual skill and tool definitions presented to
the agent, not installed-file counts or manifest character estimates. Each adapter pins the exact
tokenizer or vendor-reported accounting method and records definition-payload digests and lengths,
inclusion rules, and method version. Exact payload bytes are retained or published only when source
license and privacy permit; otherwise pinned public sources reproduce them. If actual exposure
cannot be observed or reproduced for an adapter, that adapter cannot support the token claim.
Results are reported per agent/model; any pooled or hierarchical estimate and weighting must be
preregistered and may not hide an inferior adapter.

The harness, benchmark licenses and integrity digests, pinned run instructions, environment
description, analysis code, and complete redacted result records are committed so an independent
developer can reproduce the result. Public claims use only data that can be lawfully disclosed;
private repository runs may inform development but cannot substitute for the reproducible gate. If
a stable customer use case for repository self-evaluation later emerges, a separate design may
promote it into the product; Phase 2.5 does not.

Phase 2.5 also contains a retrieval ablation for Compose. A pre-registered labeled query set
compares FTS5 plus agent query reformulation with dense+sparse local hybrid retrieval on candidate
recall at fixed `k`, latency, and distribution size. A local embedding model may enter the product
only if it produces a material, pre-registered recall improvement and a separate ADR accepts the
model runtime, licensing, versioning, cross-compilation, cache, and reproducibility costs. Until
then, Compose is FTS5-only.

What the gate blocks, precisely: **further investment in Compose**, and the correction-mining half
of Phase 3 (which exists only to feed `.skulto/context.md`, a Compose artifact). It does **not**
block invocation reporting — unobserved-skill detection has standalone value for pruning any
environment, composed or hand-rolled, that a supported adapter can observe, and it survives even
if Compose dies.

**Phase 3 — OBSERVE.** The `claudecode` transcript adapter, `report invocations` (unconditional),
`report corrections` (gated on Phase 2.5).

**Phase 4 — MCP diet.** 12 → 3 tools, deprecation warning in 2.0, removal in 2.1. ADR 0003 superseded.

---

## 9. The falsifiable claim

The ETH result (§2.3) is the standard skulto will be measured against. **We measure ourselves
against it before someone else does.**

**Method.** The developer-only `eval/` harness runs a primary controlled pair plus two secondary
reference conditions on real repositories with real tasks. AGENTbench is published; SWE-bench Lite
is free. It shells the same supported agent CLI for all conditions in isolated disposable checkouts
and scores success only through the benchmark's deterministic test oracle; the agent never judges
its own result.

Primary controlled pair:

1. **v1-agent-managed** — the agent receives Skulto's current v1 search, get, and install tools and
   chooses skills through the realistic current workflow; it receives no Compose orchestration,
   context routing, pruning, or automatic budget optimisation
2. **composed** — the same agent and tools, plus the budgeted and pruned Compose workflow and
   `.skulto/context.md`

Secondary reference conditions, reported separately and never substituted for the primary control:

- **no-skills** — no skills and no context file; a lower anchor, not the current-product baseline
- **install-everything** — install every corpus skill judged relevant; a stress test, not a credible
  representation of an agent using v1 normally

The primary pair uses the same agent and model version, task/user prompt, starting repository,
candidate-corpus snapshot, signed ruleset, run-order randomisation scheme, and token, time,
iteration, and tool-call limits. Condition-specific orchestration instructions are the treatment,
are preregistered, and are the only intended prompt difference.

**Primary claim.** Deliberately a *subtraction* claim, which is the easier one to win and the one the
evidence supports:

> A skulto-composed environment is **non-inferior to the v1-agent-managed environment on task
> success, within a pre-registered margin of 2 percentage points**, while spending **≥20% fewer
> tokens** on skill and tool definitions.

The analysis is paired at the task level: v1-agent-managed and composed run the same tasks under
identical agent, model, prompt, and limit settings. Before any outcome is inspected, the harness
commits its task set, power analysis, exclusion rules, and statistical procedure. The procedure
must account for task- and repository-level variation and report effect sizes with confidence
intervals; a simple "won 2 of 3 repositories" vote is not valid evidence.

The gate passes only when both conditions hold:

1. the one-sided 95% lower confidence bound for
   `success(composed) - success(v1-agent-managed)` is greater
   than `-2pp`; and
2. the one-sided 95% lower confidence bound for the paired mean reduction in skill-and-tool
   definition tokens is at least `20%`.

**Steps to first relevant file is a secondary diagnostic**, reported with uncertainty but excluded
from the conjunctive pass gate. The available task set must be capable of resolving the
pre-registered margins; an underpowered benchmark cannot pass merely because its point estimate
looks favorable.

**Secondary measurement.** Phase 3 reports explicit observed invocation rate on supported agents.
The cited 30–50% passive-trigger baseline is not used as a target because it is not definitionally
equivalent to transcript-observed invocation. A comparative claim requires a baseline measured by
the same adapter and event definition; until then, invocation rate is descriptive and not a Phase
2.5 gate.

**Decision rule — pre-registered, before we are attached to the answer:**

- **Pass:** both primary conditions above pass. Compose may advance and the claim may be made with
  the measured effect sizes and intervals.
- **Inferior:** the confidence interval establishes a success loss greater than `2pp`, or establishes
  that the token reduction is below `20%`. **Compose is dead.** Skulto becomes the trust layer
  (Verify) and observability layer (Observe), and says so publicly.
- **Inconclusive:** either interval crosses its decision boundary. The Phase 2.5 gate remains closed;
  no Compose claim is made and no gated downstream investment begins. This is uncertainty, not
  evidence of equality and not evidence of inferiority. Any larger follow-up evaluation must be
  specified and budgeted before inspecting additional outcomes.

---

## 10. Risks

Ranked by what actually kills the product.

**1. There is no demand.** The research found zero people asking for project-specific skill
composition and zero willing to pay. *Mitigation:* lead with Verify, which has a screaming,
evidenced pain. Compose rides in behind it. **If Verify does not get adoption, Compose never will.**

**2. The ETH result generalises and `.skulto/context.md` is net-negative too.** *Mitigation:* we
generate almost nothing — one small file containing pointer-only routing entries and genuinely
undocumented facts. It changes through human-approved, entry-level deltas rather than wholesale
regeneration; superseded entries may be amended, deprecated, or removed while history is retained
separately. Lint rejects duplicated claims, broken pointers, stale metadata, and budget overflow;
the agent and human review semantic discoverability. §9 settles whether the result is net-positive.

**3. The budget numbers are folklore.** 15,000 chars (Claude Code) and 2% of window (Codex) are
reverse-engineered from blog posts and GitHub issues, **not documented by the vendors**. They can
change silently and break the central metric. *Mitigation:* each supported adapter first discovers
the effective limit dynamically through a stable machine-readable agent capability or supported
empirical signal. Transcript evidence such as *"Showing 60 of 3218 skills"* can confirm truncation
and invalidate an earlier observation, but Skulto does not depend on brittle scraping of arbitrary
human-readable output. When dynamic discovery is unavailable, an evidence-backed registry embedded
and tested with the installed Skulto release supplies an `estimated` fallback; it does not update
independently or share the security-rules feed. When neither source is defensible, status is
`unknown`.

Dynamic probes are fixed, allowlisted, non-interactive adapter commands with closed stdin, bounded
execution, and schema-validated output; they may not start sessions or run project hooks.
Platforms without a defensible limit remain installable only with an explicit **`budget unverified`**
warning recorded in the manifest; Skulto makes no fit claim for them. Every compose, validate,
apply, and doctor path re-probes the installed agent. A version change alone is harmless when the
new probe confirms equivalent budget semantics; a changed or unverifiable limit requires
recomposition, so an update cannot silently inherit a stale lockfile value.

**4. Kiro or Agent OS closes the gap with one feature.** They can, and quickly — Kiro already has
`requirements.md` and a "Generate Steering Docs" button. *Mitigation:* the idea is not the moat.
**The corpus, the scanner, the ruleset feed, and the trigger data are.** The July 2026 research
snapshot found no competitor combining 33-platform install, deterministic scanning, and transcript
observability. Recheck that claim before publishing it.

**5. Skulto becomes the bloat it sells a cure for.** *Mitigation:* the ~120-token footprint is a
published, enforced constraint. `skulto doctor` fails if it grows.

**6. Telemetry touches transcript content.** PostHog is already in this codebase. *Mitigation:*
invariant 8, enforced by test.

**7. Someone unpauses skill generation.** A contributor adds forking because it is a fun afternoon.
*Mitigation:* the ADR forbids authoring capability claims without an execution oracle, in writing,
with the reasoning. **Generation is not "later" — it is "not without an oracle."**

---

## 11. Blast radius in the existing codebase

- **Unified local storage is a prerequisite, not incidental cleanup.** The accepted
  `2026-04-19-unify-local-skill-storage` design must land and pass its release certification before
  any v2 phase proceeds. Its path invariant becomes an assumed precondition for manifest pinning,
  immutable artifact identities, audit, and environment application.
- **Installation is artifact-driven, not checkout-driven.** A new `internal/artifact/` boundary
  resolves `source + path + full ref`, computes a canonical whole-directory digest, and atomically
  materialises write-protected, tamper-evident objects under `~/.agents/skulto/objects/sha256/`.
  Installer entry points accept a verified object path instead of reconstructing mutable repository
  paths from `models.Skill + models.Source`; installation records retain the object digest. Garbage
  collection is explicitly deferred.
- **`skulto.json` schema changes.** `manifest.Version` is a revision counter despite the name
  (`internal/cli/save.go:178`). Becomes `revision`; a real `schema` field is added; migration keys
  off its absence.
- **`internal/skillgen/` is gutted.** Its cwd scanner is already condemned. Its CLI-shelling executor
  survives as the CLI's path to the user's agent.
- **`installer.Platform` gains `SkillBudget`.** Thirty-three entries begin as `unknown` unless a
  tested dynamic method or cited fallback supports a stronger status. Unknown coverage is reported
  as a product limitation, not generalized into a claim about every platform vendor.
- **`internal/llm/` is deleted.** Unused, and a temptation toward violating invariant 1.
- **ADR 0003 is superseded.** The three interfaces are **no longer at parity**: CLI is complete, TUI
  is human-facing, MCP is a deliberately narrow discovery window. Written down explicitly, or the
  next contributor will helpfully restore the nine deleted tools.
- **MCP installation requires elicitation.** The current `mcp-go` v0.27 dependency exposes tool-risk
  annotations but not server-initiated elicitation. Phase 4 must upgrade or replace that SDK path;
  annotations alone do not satisfy the human-approval invariant.
- **Compose does not depend on semantic search.** It uses FTS5 plus iterative query reformulation by
  the installed agent. A bundled local embedding model is evidence-gated by the Phase 2.5 retrieval
  ablation and requires a separate ADR; hosted embeddings are not a Compose fallback.
- **The existing hosted embedding path is grandfathered, not endorsed for expansion.**
  `internal/embedding/` and `internal/vector/` may continue serving the explicitly configured,
  `OPENAI_API_KEY`-gated TUI feature. No new v2 package or interface may import that path; removing
  or replacing it later requires a separate compatibility decision.
- **Raw search-query telemetry is removed across every surface.** `TrackSearchPerformed` no longer
  accepts or emits query content; it records bounded operational and funnel aggregates only.
  Privacy regression tests seed distinctive canary queries through CLI, TUI, MCP, and Compose and
  assert that neither the canaries nor derived hashes/fragments appear in emitted properties or
  free-form errors. `context/telemetry.md` is corrected in the same change. Exact failed-query
  diagnostics remain local behind explicit user-reviewed export.
- **Telemetry identifiers become public-only.** `models.Source` gains persisted GitHub visibility
  provenance, refreshed during scrape; only a positively verified public source permits repository
  or skill identifiers in an event. Private, local, and unknown sources emit coarse origin and
  aggregates only. `TrackCLIHelpViewed` drops raw arguments, and free-form failure telemetry is
  replaced with enumerated error types. Tests cover public, private, local, stale, and unknown
  provenance, with unknown failing closed. `context/telemetry.md` and ADR 0006 are updated in the
  same implementation change.

---

## 12. Sources

- ETH Zurich SRI Lab + LogicStar, *Evaluating AGENTS.md* — https://arxiv.org/abs/2602.11988
- ACE, *Agentic Context Engineering* — https://arxiv.org/abs/2510.04618
- SkillWeaver — https://arxiv.org/abs/2504.07079 · Voyager — https://arxiv.org/abs/2305.16291 ·
  ReasoningBank — https://arxiv.org/abs/2509.25140 · Alita-G — https://arxiv.org/abs/2510.23601 ·
  GEPA — https://arxiv.org/abs/2507.19457
- IFScale (instruction-density decay) — https://arxiv.org/abs/2507.11538
- RFC 8785, *JSON Canonicalization Scheme (JCS)* —
  https://www.rfc-editor.org/rfc/rfc8785.html
- The Update Framework, *Specification* —
  https://theupdateframework.github.io/specification/
- Google, *Managing ML projects: experiments* —
  https://developers.google.com/machine-learning/managing-ml-projects/experiments
- NIST AI 800-2, *Practices for Automated Benchmark Evaluations of Language Models* —
  https://nvlpubs.nist.gov/nistpubs/ai/NIST.AI.800-2.ipd.pdf
- NIST AI 800-3, *Expanding the AI Evaluation Toolbox with Statistical Models* —
  https://doi.org/10.6028/NIST.AI.800-3
- FDA, *Non-Inferiority Clinical Trials* —
  https://www.fda.gov/regulatory-information/search-fda-guidance-documents/non-inferiority-clinical-trials
- Model Context Protocol, *Elicitation* —
  https://modelcontextprotocol.io/specification/2025-06-18/client/elicitation
- Model Context Protocol, *Tool Annotations as Risk Vocabulary* —
  https://blog.modelcontextprotocol.io/posts/2026-03-16-tool-annotations/
- SkillReducer (55,315 skills analysed) — https://arxiv.org/html/2603.29919v1
- Snyk, *ToxicSkills* — https://snyk.io/blog/toxicskills-malicious-ai-agent-skills-clawhub/
- Anthropic, *Equipping agents with Agent Skills* —
  https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills
- Anthropic, *Advanced tool use / Tool Search Tool* —
  https://www.anthropic.com/engineering/advanced-tool-use
- Claude Code skill-budget truncation — https://github.com/anthropics/claude-code/issues/14549
- Codex skill metadata budget — https://github.com/openai/codex/issues/19679
- spec-kit CLAUDE.md destruction — https://github.com/github/spec-kit/issues/365
- Kiro steering docs — https://kiro.dev/docs/steering/ · Agent OS —
  https://buildermethods.com/agent-os/workflow · Tessl — https://tessl.io/
