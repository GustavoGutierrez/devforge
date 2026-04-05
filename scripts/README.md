# Scripts

## Main scripts

| Script | Purpose |
|---|---|
| `install.sh` | Build and install `devforge`, `devforge-mcp`, and `dpf` locally |
| `install-dpf.sh` | Download the matching DevPixelForge binary |
| `package_release_bundle.sh` | Build a release bundle for linux/amd64 or darwin/arm64 |
| `render_homebrew_formula.py` | Render the Homebrew formula from `checksums.txt` |
| `setup-mcp-client.sh` | Configure MCP clients |
| `uninstall.sh` | Remove a local installation |

## Packaging model

Release bundles contain only:

- `devforge`
- `devforge-mcp`
- `dpf`

DevForge no longer ships or initializes any runtime database.

## install.sh

Builds both Go binaries, copies them into `~/.local/share/devforge/versions/<version>/`, installs `dpf` if available, writes a minimal config file, and creates symlinks in `~/.local/bin`.

## package_release_bundle.sh

Usage:

```bash
bash scripts/package_release_bundle.sh --version 2.1.1 --target-os linux --target-arch amd64
bash scripts/package_release_bundle.sh --version 2.1.1 --target-os darwin --target-arch arm64
```

## render_homebrew_formula.py

Consumes `checksums.txt` containing both release archives and renders the Homebrew formula with Linux amd64 and macOS arm64 SHA256 values.
