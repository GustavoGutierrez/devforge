# DevForge Release Process

## Source of truth

- Workflow: `.github/workflows/release.yml`
- Bundle script: `scripts/package_release_bundle.sh`
- Formula template: `packaging/homebrew/Formula/devforge.rb`
- Formula renderer: `scripts/render_homebrew_formula.py`

## Published assets

Each tagged release publishes:

- `devforge_<version>_linux_amd64.tar.gz`
- `devforge_<version>_darwin_arm64.tar.gz`
- `checksums.txt`

Each archive contains:

- `devforge`
- `devforge-mcp`
- `dpf`

## Local validation

```bash
go test ./...
ruby -c packaging/homebrew/Formula/devforge.rb
```

## Release flow

1. Tag `vX.Y.Z`
2. `prepare` validates `VERSION` and test suite
3. Linux amd64 bundle builds on Ubuntu
4. macOS arm64 bundle builds on native `macos-14`
5. GitHub release uploads both archives plus `checksums.txt`
6. Homebrew formula is rendered from aggregated checksums
7. Tap repo `GustavoGutierrez/homebrew-devforge` is updated

## Notes

- No bundled database is part of the runtime anymore.
- Cross-repo tap publishing still requires `HOMEBREW_TAP_SSH_KEY`.
