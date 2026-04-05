---
name: publish-release
description: Publish a DevForge GitHub release with the Linux-first runtime bundle and dedicated Homebrew tap synchronization.
license: Apache-2.0
metadata:
  author: GustavoGutierrez
  version: "2.0"
---

## When to use this skill

- Publishing a new DevForge version
- Re-running a failed release for an existing tag
- Verifying GitHub release assets or the dedicated Homebrew tap after the tap migration

## Release source of truth

- Workflow: `.github/workflows/release.yml`
- Homebrew formula template: `packaging/homebrew/Formula/devforge.rb`
- Homebrew tap docs: `packaging/homebrew/README.md`, `packaging/homebrew/RELEASE_PROCESS.md`
- Release bundle builder: `scripts/package_release_bundle.sh`
- Homebrew formula renderer: `scripts/render_homebrew_formula.py`

## Packaging model

- DevForge ships a Linux-first runtime bundle, not a single binary.
- The canonical release asset is:

  ```text
  devforge_X.Y.Z_linux_amd64.tar.gz
  ```

- That bundle must contain all runtime-critical artifacts:
  - `devforge`
  - `devforge-mcp`
  - `dpf`
  - `devforge.db`

- The Homebrew formula installs those files into `libexec` and exposes wrappers
  for `devforge` and `devforge-mcp` plus a symlink for `dpf`.

## Dedicated tap decision

- User-facing Homebrew commands are:

  ```bash
  brew tap GustavoGutierrez/devforge
  brew install GustavoGutierrez/devforge/devforge
  ```

- Homebrew resolves that tap command to the dedicated repository
  `GustavoGutierrez/homebrew-devforge`.
- Current Homebrew scope is Linux amd64.
- macOS arm64 is future work.
- Windows is out of scope.

## Versioning rules

- Use Semantic Versioning.
- Tag format must be `vX.Y.Z`.
- Use:
  - `PATCH` for fixes, docs, packaging, release automation, and maintenance.
  - `MINOR` for backward-compatible features.
  - `MAJOR` for breaking CLI, MCP, config, or workflow changes.

## Standard publish flow

1. Validate locally:

   ```bash
   CGO_ENABLED=1 go test ./...
   make release-bundle
   ruby -c packaging/homebrew/Formula/devforge.rb
   ```

2. Ensure the dedicated tap repository and credential exist:

   - Tap repository: `GustavoGutierrez/homebrew-devforge`
   - Required secret in `GustavoGutierrez/devforge`: `HOMEBREW_TAP_SSH_KEY`
   - That secret must be the private half of a write-enabled deploy key on the tap repo

3. Commit release changes if needed.

4. Create and push the release tag:

   ```bash
   git tag vX.Y.Z
   git push origin vX.Y.Z
   ```

5. Let `.github/workflows/release.yml` publish the GitHub release assets and update the tap.

## Manual rerun flow

Use this only when the tag already exists and the workflow must be rebuilt or retried:

```bash
gh workflow run release.yml -f tag=vX.Y.Z --repo GustavoGutierrez/devforge
```

Important:

- `workflow_dispatch` does not create the tag
- The tag must already exist remotely

## What the workflow does

For a tagged release, `.github/workflows/release.yml`:

1. Verifies `VERSION` matches the tag
2. Runs `CGO_ENABLED=1 go test ./...`
3. Builds `devforge`, `devforge-mcp`, `dpf`, and `devforge.db` into a Linux amd64 bundle
4. Uploads the bundle and `checksums.txt` to the GitHub release
5. Renders the Homebrew formula from `packaging/homebrew/Formula/devforge.rb`
6. Pushes the formula and tap docs to `GustavoGutierrez/homebrew-devforge`

## Verification

### Verify GitHub release assets

```bash
gh release view vX.Y.Z --repo GustavoGutierrez/devforge --json assets
```

Expected assets:

- `devforge_X.Y.Z_linux_amd64.tar.gz`
- `checksums.txt`

### Verify the dedicated tap repository

```bash
gh repo view GustavoGutierrez/homebrew-devforge
gh api repos/GustavoGutierrez/homebrew-devforge/contents/Formula/devforge.rb?ref=HEAD
```

### Verify Homebrew installation behavior

```bash
brew tap GustavoGutierrez/devforge
brew install GustavoGutierrez/devforge/devforge
devforge-mcp
brew info GustavoGutierrez/devforge/devforge
```

## Failure handling

- If `go test` fails, fix the code before tagging.
- If `make release-bundle` fails, verify `bin/dpf` exists and is executable.
- If the tag does not exist remotely, create or push it before using the rerun flow.
- If the tap update fails, verify `GustavoGutierrez/homebrew-devforge` exists and `HOMEBREW_TAP_SSH_KEY` is valid.
- Do not use the legacy `.github/workflows/homebrew.yml` flow; `release.yml` is the canonical release automation.
