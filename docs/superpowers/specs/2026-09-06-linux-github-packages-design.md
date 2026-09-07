# Linux GitHub Packages Design

**Date:** 2026-09-06  
**Status:** Approved design; pending independent spec review  
**Scope:** GitHub Release `.deb` and `.rpm` assets for Skulto

## Goal

Distribute Skulto as installable Linux packages attached to each GitHub Release.
The first milestone supports Debian-family and RPM-family systems without
operating an Apt or DNF repository.

## Non-goals

- AUR or Nix packaging (separate future work).
- Apt, DNF, or Zypper repository hosting.
- Package signing, system services, shell completions, or package lifecycle
  scripts.
- Changing the existing binary build, macOS artifacts, or tarball distribution.

## Package Contract

Each release produces four package assets:

| Format | amd64 asset | arm64 asset |
|--------|-------------|-------------|
| Debian | `skulto_<version>_linux_amd64.deb` | `skulto_<version>_linux_arm64.deb` |
| RPM | `skulto-<version>-1.x86_64.rpm` | `skulto-<version>-1.aarch64.rpm` |

The upstream package name is `skulto`. Every package contains precisely:

- `/usr/bin/skulto`
- `/usr/bin/skulto-mcp`

Both files are executable. Packages have no runtime dependencies: Skulto uses
the pure-Go `go-git` library for repository operations and does not invoke a
system `git` executable at runtime.

Package versions derive from the existing version tag after removing an
optional leading `v`. The package release/revision is `1`; prerelease tags
retain their prerelease information using nFPM's SemVer-compatible version
handling.

## Architecture and Data Flow

The current release workflow builds Linux `amd64` and `arm64` binaries. The
release job already downloads and organizes those outputs. Packaging extends
that job only:

1. Download and validate Linux build artifacts as today.
2. Install a pinned nFPM version in the GitHub runner.
3. For each Linux architecture, invoke nFPM once for `deb` and once for `rpm`.
4. Place the resulting four assets in `dist/` with the names in the package
   contract.
5. Inspect each completed package before creating the GitHub Release.
6. Generate one checksum file covering tarballs and packages.
7. Create the GitHub Release and upload all assets.

`packaging/nfpm.yaml` is the single package metadata definition. It receives
the version and architecture through environment variables. nFPM maps Go
architecture names to the appropriate package format names, including
`amd64` to `x86_64` and `arm64` to `aarch64` for RPM.

## Verification and Failure Handling

Package creation happens before `gh release create`. The release must fail
without publishing assets if any of these checks fail:

- an expected Linux binary is absent;
- nFPM cannot create a package;
- `dpkg-deb` inspection fails for a Debian package;
- `rpm` inspection fails for an RPM package;
- either expected executable is absent from a package;
- any of the four package artifacts is absent from `dist/`.

The workflow validates package metadata and confirms both executable paths in
every package. The existing complete Go test and lint jobs remain prerequisites.
Tests added with the change cover the packaging configuration/command inputs
and artifact-validation helpers. The release workflow itself remains the
integration test for all four actual package formats.

## Release Metadata and Documentation

`checksums.txt` is extended to contain SHA-256 hashes of all tarballs and all
four Linux package assets. Packages are unsigned in this milestone; download
integrity is verified with that checksum file.

The generated release notes and user documentation list the package assets and
give local-file installation examples:

```bash
sudo apt install ./skulto_<version>_linux_amd64.deb
sudo dnf install ./skulto-<version>-1.x86_64.rpm
sudo zypper install ./skulto-<version>-1.x86_64.rpm
```

Documentation makes clear that these packages are GitHub Release downloads,
not packages served from an Apt or DNF repository.

## Implementation Boundaries

Expected touched areas:

- `packaging/nfpm.yaml` — package metadata and contents.
- `.github/workflows/release.yml` — package generation, validation, checksums,
  uploads, and release notes.
- `README.md` and `docs/getting-started.md` — Linux package installation.
- Focused tests and/or workflow-validation helpers for packaging inputs and
  expected artifacts.

No application runtime code changes are required.

## Acceptance Criteria

- A tagged release produces installable Debian and RPM packages for amd64 and
  arm64 alongside the current tarballs.
- Each package installs both Skulto binaries in `/usr/bin`.
- Package metadata uses a package-manager-compatible version without the tag's
  leading `v`.
- Invalid/missing artifacts block release creation.
- `checksums.txt` hashes every published binary archive and Linux package.
- README and getting-started documentation cover installation and checksum
  verification.
