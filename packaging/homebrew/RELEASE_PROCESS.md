# DevForge Release Process

This document describes the Linux-first DevForge release flow and dedicated
Homebrew tap publication.

## Source of truth

- Workflow: `.github/workflows/release.yml`
- Homebrew template: `packaging/homebrew/Formula/devforge.rb`
- Tap docs: `packaging/homebrew/README.md`, `packaging/homebrew/RELEASE_PROCESS.md`
- Bundle script: `scripts/package_release_bundle.sh`
- Formula renderer: `scripts/render_homebrew_formula.py`

## Release trigger

Preferred trigger:

```bash
git tag vX.Y.Z
git push origin vX.Y.Z
```

Manual rerun for an existing tag:

```bash
gh workflow run release.yml -f tag=vX.Y.Z
```

The manual workflow rebuilds an existing tag. It does not create tags.

## Local validation

```bash
CGO_ENABLED=1 go test ./...
make release-bundle
ruby -c packaging/homebrew/Formula/devforge.rb
```

## Runtime bundle contents

For Linux amd64, the release workflow publishes one canonical bundle:

- `devforge_<version>_linux_amd64.tar.gz`
- `checksums.txt`

The archive contains:

- `devforge`
- `devforge-mcp`
- `dpf`
- `devforge.db`

This ensures Homebrew installs all required runtime artifacts, including the
seeded SQLite database and the bundled media-processing binary.

## Workflow behavior

For tag `vX.Y.Z`, `.github/workflows/release.yml`:

1. Verifies `VERSION` matches the tag
2. Runs `CGO_ENABLED=1 go test ./...`
3. Validates the Homebrew template with Ruby
4. Builds the Linux amd64 runtime bundle
5. Uploads the bundle plus `checksums.txt` to the GitHub release
6. Renders the Homebrew formula with the published checksum
7. Pushes `Formula/devforge.rb`, `README.md`, and `RELEASE_PROCESS.md` to
   `GustavoGutierrez/homebrew-devforge`

## Dedicated tap repository

Users should install DevForge with:

```bash
brew tap GustavoGutierrez/devforge
brew install GustavoGutierrez/devforge/devforge
```

That command only works after the dedicated tap repository
`GustavoGutierrez/homebrew-devforge` exists.

## Required repository secret

Configure this secret in `GustavoGutierrez/devforge`:

- `HOMEBREW_TAP_SSH_KEY`

Recommended setup:

1. Create `GustavoGutierrez/homebrew-devforge`
2. Add a write-enabled deploy key to that repository
3. Store the private key in the source repository as `HOMEBREW_TAP_SSH_KEY`

The default source-repository `GITHUB_TOKEN` is not sufficient for the
cross-repository push.

## Linux-first scope

Current release automation intentionally prioritizes Debian/Linux amd64.

- Linux amd64: supported and published
- macOS arm64: documented future work
- Windows: out of scope

## Verification commands

After the workflow finishes:

```bash
gh release view vX.Y.Z --repo GustavoGutierrez/devforge --json assets
gh repo view GustavoGutierrez/homebrew-devforge
gh api repos/GustavoGutierrez/homebrew-devforge/contents/Formula/devforge.rb?ref=HEAD
```

## Troubleshooting

### Bundle creation fails because `bin/dpf` is missing

Reinstall or refresh the bundled runtime binary:

```bash
bash scripts/install-dpf.sh
chmod +x bin/dpf
```

### The tap publish step cannot authenticate

Confirm `HOMEBREW_TAP_SSH_KEY` exists and matches a write-enabled deploy key on
`GustavoGutierrez/homebrew-devforge`.

### The dedicated tap repo does not exist yet

Create `GustavoGutierrez/homebrew-devforge` before triggering a release. The
workflow publishes into that repository, but it does not create it.
