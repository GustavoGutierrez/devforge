# Legacy branch-based release process

This file documents a deprecated flow.

The branch-based `homebrew-tap` publication model has been replaced by the
dedicated tap repository flow described here:

- `packaging/homebrew/README.md`
- `packaging/homebrew/RELEASE_PROCESS.md`
- `.github/workflows/release.yml`

Canonical target repository:

```text
GustavoGutierrez/homebrew-devforge
```

Canonical user install flow:

```bash
brew tap GustavoGutierrez/devforge
brew install GustavoGutierrez/devforge/devforge
```

Do not use the old `.github/workflows/homebrew.yml` flow for new releases.

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

### Tap command fails with "repository not found"

Homebrew expects the tap repo to be named `homebrew-{name}` (e.g., `homebrew-devforge` for `GustavoGutierrez/devforge`). Since this repo is `devforge-mcp`, you must specify the URL explicitly:

```bash
brew tap GustavoGutierrez/devforge https://github.com/GustavoGutierrez/devforge homebrew-tap
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
