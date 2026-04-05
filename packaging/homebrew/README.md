# DevForge Homebrew Tap Assets

This directory is the source of truth for the dedicated DevForge Homebrew tap.

- Source repository: `GustavoGutierrez/devforge`
- Dedicated tap repository: `GustavoGutierrez/homebrew-devforge`
- User-facing tap command: `brew tap GustavoGutierrez/devforge`

Homebrew maps `brew tap <owner>/<name>` to `<owner>/homebrew-<name>`, so
`brew tap GustavoGutierrez/devforge` resolves to
`GustavoGutierrez/homebrew-devforge`.

## Supported Homebrew targets

- Linux amd64

Planned future work:

- macOS arm64

Windows remains out of scope for Homebrew.

## Install DevForge with Homebrew

```bash
brew tap GustavoGutierrez/devforge
brew install GustavoGutierrez/devforge/devforge
```

After installation:

```bash
brew update
brew upgrade devforge
```

## What gets installed

The Homebrew formula installs a runtime bundle into `libexec` containing:

- `devforge`
- `devforge-mcp`
- `dpf`
- `devforge.db`

The formula exposes wrappers for `devforge` and `devforge-mcp`, plus a `dpf`
symlink, so the binaries can resolve their colocated runtime files reliably.

## Publishing model

On every tagged release (`v*`), `.github/workflows/release.yml`:

1. Builds the Linux runtime bundle from the source repository
2. Uploads `devforge_<version>_linux_amd64.tar.gz` and `checksums.txt`
3. Renders `packaging/homebrew/Formula/devforge.rb`
4. Publishes the formula plus tap docs to `GustavoGutierrez/homebrew-devforge`

## Authentication requirement

Cross-repository publishing requires a repository secret named
`HOMEBREW_TAP_SSH_KEY` in `GustavoGutierrez/devforge`.

That secret must contain the private half of a write-enabled deploy key
registered on `GustavoGutierrez/homebrew-devforge`.
