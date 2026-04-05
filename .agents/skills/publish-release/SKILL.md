---
name: publish-release
description: Publish a DevForge GitHub release with Linux amd64 and macOS arm64 runtime bundles plus Homebrew tap synchronization.
license: Apache-2.0
metadata:
  author: GustavoGutierrez
  version: "3.0"
---

## Source of truth

- `.github/workflows/release.yml`
- `scripts/package_release_bundle.sh`
- `scripts/render_homebrew_formula.py`
- `packaging/homebrew/Formula/devforge.rb`

## Packaging model

Published release assets:

- `devforge_X.Y.Z_linux_amd64.tar.gz`
- `devforge_X.Y.Z_darwin_arm64.tar.gz`
- `checksums.txt`

Each bundle contains:

- `devforge`
- `devforge-mcp`
- `dpf`

DevForge no longer ships any runtime database, SQLite data, FTS index, or embedding assets.

## Standard flow

1. Validate locally:

   ```bash
   go test ./...
   ruby -c packaging/homebrew/Formula/devforge.rb
   ```

2. Tag and push:

   ```bash
   git tag vX.Y.Z
   git push origin vX.Y.Z
   ```

3. Let `release.yml` build Linux amd64 on Ubuntu and macOS arm64 on native `macos-14`.

4. Verify assets and tap formula.

## Verification

```bash
gh release view vX.Y.Z --repo GustavoGutierrez/devforge --json assets
gh api repos/GustavoGutierrez/homebrew-devforge/contents/Formula/devforge.rb?ref=HEAD
```
