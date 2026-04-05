# Legacy homebrew-tap snapshot

This directory is no longer the source of truth for DevForge Homebrew packaging.

Use these files instead:

- `packaging/homebrew/Formula/devforge.rb`
- `packaging/homebrew/README.md`
- `packaging/homebrew/RELEASE_PROCESS.md`

Canonical publication now targets the dedicated tap repository:

```text
GustavoGutierrez/homebrew-devforge
```

User-facing install command:

```bash
brew tap GustavoGutierrez/devforge
brew install GustavoGutierrez/devforge/devforge
```

This `homebrew-tap/` directory remains only as a historical snapshot for the old
branch-based flow.
