---
name: publish-release
description: >
  Step-by-step workflow for publishing a new DevForge version to GitHub Releases and Homebrew tap.
  Trigger: When the user asks to release a new version, bump the version, publish to GitHub,
  push a new release, create a release, "subir versión", "nueva versión", "nuevo release",
  "publicar versión", or "crear release".
license: Apache-2.0
metadata:
  author: GustavoGutierrez
  version: "1.1"
---

## When to Use

Load this skill when:
- User asks to release a new version, bump version, or publish to GitHub/Homebrew
- User says "subir versión", "nueva versión", "nuevo release", "publicar versión", or "crear release"
- User asks to tag a release or increment the version number

---

## Critical Patterns

- **`gh release create` uses `--notes`, NOT `--body`** — `--body` is an unknown flag and will crash the CI.
- **`VERSION` is the single source of truth** — always read it first; never hardcode the version.
- **`v` prefix required in workflow dispatch** — pass `version=vX.Y.Z`, not `X.Y.Z`.
- **Use `Edit` tool for Go/Ruby files** — do NOT use bash `sed` for `main.go` or `devforge.rb`.
- **PR branch must differ from base** — CI pushes to `update-vX.Y.Z`, PR targets `homebrew-tap`. Never push directly to `homebrew-tap`.
- **`CGO_ENABLED=1` required** — `go build ./...` without it will fail due to `go-sqlite3`.
- **Release notes must be English Markdown** — no plain text, no other language.
- **4 files must be updated every release**: `VERSION`, `README.md`, `cmd/devforge-mcp/main.go`, `Formula/devforge.rb`.
- **`Formula/devforge.rb` in `main` needs BOTH `version` AND `sha256` synced after CI.** The CI only updates `homebrew-tap`; the formula on `main` is left with the old sha256 (Step 2 only updates `version`). If not synced, `brew upgrade` will report a checksum mismatch warning. Always run Step 9 after merging the Homebrew PR.

---

## Semantic Versioning Rules

| Change type | Segment | Example |
|-------------|---------|---------|
| Breaking change to MCP API, protocol, or config schema | **MAJOR** | `1.x.x → 2.0.0` |
| New MCP tool, CLI command, dpf job type (backwards-compatible) | **MINOR** | `1.0.x → 1.1.0` |
| Bug fix, refactor, docs update, dependency update, CI fix | **PATCH** | `1.0.1 → 1.0.2` |

---

## Files Updated Every Release

| File | What to change |
|------|---------------|
| `VERSION` | Plain semver string (no `v` prefix): `1.0.2` |
| `README.md` | Badge URL: `version-X.Y.Z-blue.svg` |
| `cmd/devforge-mcp/main.go` | `mcpserver.NewMCPServer("devforge", "X.Y.Z", ...)` |
| `Formula/devforge.rb` | `version "X.Y.Z"` — **sha256 is updated later in Step 9** |

---

## Step-by-Step Release Process

### Step 1 — Determine the new version

```bash
cat VERSION
```

Apply SemVer rules. Decide the new version (e.g., `1.0.2`).

---

### Step 2 — Update all version references

Use the `Edit` tool for each file:

- `VERSION` → plain semver string, no `v` prefix
- `README.md` → badge `version-OLD-blue.svg` → `version-NEW-blue.svg`
- `cmd/devforge-mcp/main.go` → version string in `NewMCPServer`
- `Formula/devforge.rb` → `version "OLD"` → `version "NEW"`

---

### Step 3 — Verify the build compiles clean

```bash
CGO_ENABLED=1 go build ./...
```

Fix any errors before continuing.

---

### Step 4 — Write the release notes

Use this template (English Markdown only, include only relevant sections):

```markdown
## What's Changed

### Breaking Changes
- Description and migration path (MAJOR only)

### New Features
- Description of new tool/command (MINOR only)

### Bug Fixes
- Fix description (#issue or short description)

### Improvements
- Description of improvement

### Refactors
- Description of refactor

### Dependency Updates
- Updated X from vA to vB

---

**Full Changelog**: https://github.com/GustavoGutierrez/devforge-mcp/compare/vPREV...vNEW
```

---

### Step 5 — Commit and push to main

```bash
git add VERSION README.md cmd/devforge-mcp/main.go Formula/devforge.rb
git commit -m "chore(release): bump version to vX.Y.Z

<one-line summary of what changed>"
git push origin main
```

---

### Step 6 — Trigger the CI workflow

```bash
gh workflow run homebrew.yml \
  -f version=vX.Y.Z \
  --repo GustavoGutierrez/devforge-mcp
```

---

### Step 7 — Monitor the workflow

```bash
sleep 5 && gh run list --repo GustavoGutierrez/devforge-mcp --limit 3
gh run watch <RUN_ID> --repo GustavoGutierrez/devforge-mcp
```

| Job | What it does |
|-----|-------------|
| `prepare-release` | Creates GitHub Release at `vX.Y.Z` |
| `build-linux` | Builds binaries, packages `.tar.gz`, uploads to release |
| `update-formula` | Rewrites `Formula/devforge.rb`, pushes `update-vX.Y.Z`, opens PR |

---

### Step 8 — Verify the release and PR

```bash
gh release view vX.Y.Z --repo GustavoGutierrez/devforge-mcp
gh pr list --repo GustavoGutierrez/devforge-mcp --state open
```

Expected: release has `devforge-X.Y.Z.linux-amd64.tar.gz`; PR targets `homebrew-tap`.

---

### Step 9 — Sync sha256 in main branch formula (MANDATORY)

After the Homebrew PR (`update-vX.Y.Z` → `homebrew-tap`) is merged, the CI has computed the correct sha256 for the new bottle. You **must** copy it back to `Formula/devforge.rb` on `main`, otherwise `brew upgrade` will print a checksum mismatch warning for all users.

```bash
# 1. Read the correct sha256 from the homebrew-tap branch
git fetch origin homebrew-tap
git show origin/homebrew-tap:Formula/devforge.rb | grep sha256
```

Copy the sha256 value, then update `Formula/devforge.rb` on `main`:

```bash
# 2. Edit the formula on main (use the Edit tool, not sed)
#    Change: sha256 "OLD_HASH"
#    To:     sha256 "NEW_HASH_FROM_HOMEBREW_TAP"
```

Also ensure `homebrew-tap` has an explicit `version "X.Y.Z"` field. If the CI dropped it, add it back:

```bash
git show origin/homebrew-tap:Formula/devforge.rb | grep version
# If missing, checkout homebrew-tap and add: version "X.Y.Z"
```

Commit and push both fixes:

```bash
# Fix on main
git checkout main
# (after editing Formula/devforge.rb with correct sha256)
git add Formula/devforge.rb
git commit -m "fix(homebrew): sync sha256 for vX.Y.Z bottle in main branch formula"
git push origin main

# Fix on homebrew-tap (only if version field was missing)
git checkout homebrew-tap && git pull origin homebrew-tap
# (after adding version "X.Y.Z")
git add Formula/devforge.rb
git commit -m "fix(homebrew): add explicit version X.Y.Z to tap formula"
git push origin homebrew-tap
git checkout main
```

---

## Commands

```bash
# Read current version
cat VERSION

# Build check (required before release)
CGO_ENABLED=1 go build ./...

# Trigger release CI
gh workflow run homebrew.yml -f version=vX.Y.Z --repo GustavoGutierrez/devforge-mcp

# Monitor runs
gh run list --repo GustavoGutierrez/devforge-mcp --limit 3
gh run watch <RUN_ID> --repo GustavoGutierrez/devforge-mcp

# Verify release
gh release view vX.Y.Z --repo GustavoGutierrez/devforge-mcp

# Verify formula PR
gh pr list --repo GustavoGutierrez/devforge-mcp --state open
```

---

## Complete Checklist

```
[ ] Determined correct SemVer segment (MAJOR / MINOR / PATCH)
[ ] VERSION updated (no v prefix)
[ ] README.md badge updated
[ ] cmd/devforge-mcp/main.go version string updated
[ ] Formula/devforge.rb version updated (sha256 stays old — will be fixed in Step 9)
[ ] CGO_ENABLED=1 go build ./... passes clean
[ ] Release notes written in English Markdown
[ ] Committed: chore(release): bump version to vX.Y.Z
[ ] Pushed to main
[ ] Triggered: gh workflow run homebrew.yml -f version=vX.Y.Z
[ ] All 3 CI jobs passed (prepare-release, build-linux, update-formula)
[ ] GitHub Release vX.Y.Z exists with linux-amd64.tar.gz asset
[ ] Homebrew formula PR opened targeting homebrew-tap branch
[ ] Homebrew formula PR merged
[ ] sha256 in main/Formula/devforge.rb synced from homebrew-tap (Step 9)
[ ] version "X.Y.Z" explicit in homebrew-tap/Formula/devforge.rb (Step 9)
[ ] Pushed sha256 fix to main (and homebrew-tap if needed)
```

---

## Anti-Patterns

| Anti-pattern | Symptom | Fix |
|---|---|---|
| Skip Step 9 (sha256 sync) | `brew upgrade` prints "Formula reports different checksum" warning for all users | After Homebrew PR merges, read sha256 from `homebrew-tap`, update `main/Formula/devforge.rb`, commit+push |
| CI drops `version` field from `homebrew-tap` formula | Homebrew must infer version from URL — fragile, may mismatch | After Homebrew PR merges, verify `version "X.Y.Z"` is explicit in `homebrew-tap/Formula/devforge.rb`; add if missing |
| Hardcode sha256 in Step 2 | Wrong sha256 at release time (binary not yet built) | In Step 2, only update `version`; sha256 is computed by CI and synced in Step 9 |

---

## Resources

- **CI pipeline**: See [.github/workflows/homebrew.yml](../../../.github/workflows/homebrew.yml)
- **Homebrew formula**: See [Formula/devforge.rb](../../../Formula/devforge.rb)
- **Release conventions**: See [AGENTS.md](../../../AGENTS.md) — Homebrew Tap section
