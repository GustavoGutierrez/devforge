# Release & Update Process

This document describes how to publish new versions of DevForge to Homebrew.

## Overview

Publishing a new version requires **three steps**:

1. **Tag & release on GitHub** — create the version tag and GitHub release
2. **Build & upload bottles** — CI builds macOS binaries and uploads them as release assets
3. **Update the formula** — commit updated sha256 checksums to the tap

Steps 2 and 3 are **automated** via GitHub Actions. You only need to do step 1.

---

## Step 1 — Create the GitHub Release

### Prerequisites

- All changes are committed and tested locally
- VERSION file is updated with the new version
- No uncommitted changes

### Process

```bash
# Make sure everything is committed
git status

# Pull latest
git pull origin main

# Tag the release
git tag vX.Y.Z
git push origin vX.Y.Z

# Create the GitHub release with release notes
gh release create vX.Y.Z \
  --title "DevForge vX.Y.Z" \
  --notes "$(cat RELEASE_NOTES.md)"
```

Or use the existing GitHub CLI workflow:

```bash
# Run the release workflow manually
gh workflow run release.yml
```

This will:
- Build both binaries (`devforge`, `devforge-mcp`)
- Download `dpf` binary
- Create the tarballs
- Upload them to the GitHub release
- Trigger the Homebrew workflow automatically

---

## Step 2 — Homebrew Bottles (Automated)

When the GitHub release is **published**, the `homebrew.yml` workflow runs automatically:

```
.github/workflows/homebrew.yml
```

It performs:

1. **Build macOS bottles** (arm64 + intel) via matrix strategy
2. **Build Linux bottle** (amd64) on ubuntu-latest
3. **Upload bottles** to the GitHub release as assets
4. **Open a PR** against the `homebrew-tap` branch with updated sha256 checksums

### Manual trigger

If the automated run fails or you need to retry:

```bash
# Via GitHub CLI
gh workflow run homebrew.yml

# Via GitHub web UI
# Actions → Homebrew Bottles → Run workflow
```

### Verify bottles were uploaded

```bash
gh release view vX.Y.Z --json assets
```

You should see three assets:
- `devforge-X.Y.Z.macos-arm64.tar.gz`
- `devforge-X.Y.Z.macos-intel.tar.gz`
- `devforge-X.Y.Z.linux-amd64.tar.gz`

---

## Step 3 — Merge the Formula PR (Automated)

The workflow automatically opens a PR on the `homebrew-tap` branch with the updated `Formula/devforge.rb` containing the correct sha256 checksums.

**Merge the PR** via GitHub web UI or:

```bash
gh pr merge --admin --delete-branch
```

---

## Manual Update (Without CI)

If you need to update the formula without the automated workflow:

### Option A — Use the release script

```bash
GITHUB_TOKEN=your_token \
DPF_VERSION=0.2.0 \
  bash scripts/release-homebrew.sh
```

The script will:
- Build bottles for arm64 and intel
- Upload to GitHub release
- Print the updated formula snippet

### Option B — Manual formula update

```bash
# 1. Download the macOS release tarballs
curl -sSL \
  "https://github.com/GustavoGutierrez/devforge-mcp/releases/download/vX.Y.Z/devforge-X.Y.Z.macos-arm64.tar.gz" \
  -o /tmp/devforge-arm64.tar.gz

# Compute sha256
shasum -a 256 /tmp/devforge-arm64.tar.gz
# → abc123... (arm64)

curl -sSL \
  "https://github.com/GustavoGutierrez/devforge-mcp/releases/download/vX.Y.Z/devforge-X.Y.Z.macos-intel.tar.gz" \
  -o /tmp/devforge-intel.tar.gz

shasum -a 256 /tmp/devforge-intel.tar.gz
# → def456... (intel)

# 2. Download the Linux release tarball
curl -sSL \
  "https://github.com/GustavoGutierrez/devforge-mcp/releases/download/vX.Y.Z/devforge-X.Y.Z.linux-amd64.tar.gz" \
  -o /tmp/devforge-linux.tar.gz

sha256sum /tmp/devforge-linux.tar.gz
# → ghi789... (linux)

# 3. Update Formula/devforge.rb
# Replace the sha256 values in the bottle blocks
```

---

## Updating dpf Version

When DevPixelForge releases a new version:

1. **Update `DPF_VERSION`** in:
   - `.github/workflows/homebrew.yml`
   - `Formula/devforge.rb` (the `dpf_version` helper method)

2. **Test locally**:
   ```bash
   brew install --build-from-source ./homebrew-tap/Formula/devforge.rb
   devforge-mcp  # verify it starts
   ```

3. **Commit** the formula changes before the next release.

---

## Homebrew Tap Maintenance

### Branch structure

```
main (source repo)
└── homebrew-tap (branch)
    └── Formula/
        └── devforge.rb
```

The `homebrew-tap` branch is used as the tap source. Users install via:

```bash
brew tap GustavoGutierrez/devforge
brew install devforge
```

### Verify the tap is working

```bash
# Tap the repo
brew tap GustavoGutierrez/devforge

# Check the formula is visible
brew info devforge

# Install from source (no bottles yet)
brew install --build-from-source devforge
```

### After merging the formula PR

Users can install pre-built bottles:

```bash
brew update
brew upgrade devforge
```

---

## Troubleshooting

### Bottle build fails with CGO error

Ensure the macOS runners have Xcode Command Line Tools:
```bash
xcode-select --install
```

For Linux builds, ensure the ubuntu-latest runner has the necessary build tools.

### dpf download fails in formula

Check that the DevPixelForge release exists:
```
https://github.com/GustavoGutierrez/devpixelforge/releases
```

Update `DPF_VERSION` if the version tag changed.

### sha256 mismatch after brew install

Delete the bottle cache and retry:
```bash
# macOS
rm -rf ~/Library/Caches/Homebrew/downloads/devforge-*

# Linux
rm -rf ~/.cache/Homebrew/downloads/devforge-*

brew install --build-from-source devforge
```

### Formula not found after git push

Make sure the `homebrew-tap` branch is pushed:
```bash
git push origin homebrew-tap
```

### Linux: FFmpeg not found

Install FFmpeg via your system package manager:
```bash
# Ubuntu/Debian
sudo apt install ffmpeg

# Fedora
sudo dnf install ffmpeg
```

---

## Summary Checklist

For each new release:

- [ ] Update `VERSION` file
- [ ] Commit all changes
- [ ] Run tests: `CGO_ENABLED=1 go test ./...`
- [ ] Tag: `git tag vX.Y.Z && git push origin vX.Y.Z`
- [ ] Create GitHub release (or use `gh workflow run release.yml`)
- [ ] Wait for `homebrew.yml` to complete
- [ ] Merge the auto-opened PR on `homebrew-tap`
- [ ] Verify: `brew info devforge`
