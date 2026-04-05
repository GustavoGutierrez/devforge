# Scripts

This directory contains shell scripts and Go helpers for installing, configuring, seeding, and managing DevForge MCP.

## File Overview

| File / Directory | Purpose |
|---|---|
| `install.sh` | Full installation: build, copy binaries, init database, seed, create symlinks |
| `install-dpf.sh` | Download the DevPixelForge pre-built binary from GitHub releases |
| `uninstall.sh` | Uninstallation with database backup prompt |
| `setup-mcp-client.sh` | Interactive configurator for VS Code, Claude Desktop, Claude Code, and OpenCode |
| `package_release_bundle.sh` | Builds the Linux amd64 release bundle with binaries, dpf, and seeded DB |
| `render_homebrew_formula.py` | Renders the Homebrew formula from the packaging template and checksums.txt |
| `release-homebrew.sh` | Deprecated legacy script kept only to redirect users to `release.yml` |
| `link-skills.sh` | Creates symlinks from `.claude/skills/` to `.agents/skills/` |
| `seed.sh` | Standalone seed script: schema init, SQL seeds, and Ollama embeddings |
| `init_db_runner/` | Go helper: applies schema migrations to a libSQL database |
| `seed_runner/` | Go helper: applies SQL seed files to a libSQL database |

---

## install.sh

Installs DevForge to `~/.local/share/devforge/versions/<version>/` and creates symlinks in `~/.local/bin/`.

**Steps executed:**

1. **Build** — Compiles `devforge-mcp` and `devforge` with `CGO_ENABLED=1` into `dist/`.
2. **Install** — Copies binaries and `dpf` (if present) to the versioned share directory.
3. **Database** — Initializes the SQLite database (via `init_db_runner`) or copies it from `dist/`.
4. **Seed (optional)** — Prompts to apply SQL seed data via `seed_runner`.
5. **Symlinks** — Creates `~/.local/bin/devforge-mcp`, `devforge`, and `dpf` symlinks.
6. **Config** — Creates `~/.config/devforge/config.json` if absent (sets `gemini_api_key` to empty, `ollama_url` to `http://localhost:11434`, `embedding_model` to `nomic-embed-text`).

**Prerequisites:**
- Go 1.24+
- `CGO_ENABLED=1` build support (requires a C compiler and SQLite development headers)
- `chmod +x bin/dpf` before running if the Rust media-processing binary exists

**Usage:**
```bash
bash scripts/install.sh
```

---

## install-dpf.sh

Downloads the pre-built DevPixelForge binary from [github.com/GustavoGutierrez/devpixelforge/releases](https://github.com/GustavoGutierrez/devpixelforge/releases) into `bin/dpf`.

**Behavior:**
- Fetches the latest release tag from the GitHub API if no version is given
- Uses `curl` if available, falls back to `wget`
- Places the binary at `bin/dpf` and sets executable permissions

**Usage:**
```bash
# Download latest release
bash scripts/install-dpf.sh

# Download a specific version
bash scripts/install-dpf.sh 0.2.0
```

---

## uninstall.sh

Removes the DevForge installation, symlinks, and optionally the config directory.

**Behavior:**
- Detects all installed versions under `~/.local/share/devforge/versions/`
- Prompts for a database backup before removal (timestamped `.db` file)
- Removes symlinks from `~/.local/bin/`
- Removes `~/.local/share/devforge/`
- Asks separately whether to remove `~/.config/devforge/`

**Usage:**
```bash
bash scripts/uninstall.sh
```

---

## setup-mcp-client.sh

Interactive configurator that adds `devforge-mcp` to various MCP clients. Detects whether the binary is installed and the config file exists before configuring.

**Supported clients:**

| Client | Config File |
|---|---|
| VS Code (GitHub Copilot MCP) | `.vscode/mcp.json` (workspace) or `~/.config/Code/User/mcp.json` |
| Claude Desktop | `~/.config/Claude/claude_desktop_config.json` (Linux) or `~/Library/Application Support/Claude/` (macOS) |
| Claude Code (CLI) | `~/.claude.json` (global) or `.mcp.json` (project-level) |
| OpenCode | `~/.config/opencode/config.json` (global) or `opencode.json` (project-level) |

**JSON merging:** Uses `jq` if available to safely merge the new server entry into an existing config. Falls back to a full rewrite if `jq` is absent (with a warning).

**Environment variables:**
- `DEV_FORGE_BIN` — overrides the binary path (default: `~/.local/bin/devforge-mcp`)
- `DEV_FORGE_CONFIG` — overrides the config path (default: `~/.config/devforge/config.json`)

**Usage:**
```bash
bash scripts/setup-mcp-client.sh
```

---

## package_release_bundle.sh

Builds the canonical Linux amd64 release artifact used by GitHub Releases and
Homebrew.

**Bundle contents:**
- `devforge`
- `devforge-mcp`
- `dpf`
- `devforge.db`

**Usage:**
```bash
bash scripts/package_release_bundle.sh --version 1.1.4
```

Default output:

```text
dist/release/devforge_1.1.4_linux_amd64.tar.gz
```

---

## render_homebrew_formula.py

Renders `packaging/homebrew/Formula/devforge.rb` by replacing the version and
bundle checksum placeholders from a release `checksums.txt` file.

**Usage:**
```bash
python3 scripts/render_homebrew_formula.py \
  --template packaging/homebrew/Formula/devforge.rb \
  --version 1.1.4 \
  --checksums-file dist/release/checksums.txt \
  --output dist/release/devforge.rb
```

---

## link-skills.sh

Creates symlinks from `.claude/skills/` (Claude Code's skill directory) pointing to `.agents/skills/` (DevForge's skill directory). This makes skills discoverable by both Claude Code and any other tool that scans `.claude/skills/`.

**Behavior:**
- **Idempotent** — skips symlinks that already point to the correct target
- **Fixes broken symlinks** — replaces symlinks with incorrect targets
- **Protects real files** — skips if a skill name is already a real file or directory (does not overwrite)
- Reports counts: created, replaced, and already-correct symlinks

**Usage:**
```bash
bash scripts/link-skills.sh
```

---

## seed.sh

Standalone script to initialize and populate the database. Combines schema initialization, SQL seed application, and Ollama embedding generation in one run.

**Modes (mutually exclusive):**
- `--seeds-only` — init schema and apply SQL seeds (no embeddings)
- `--embeddings-only` — generate embeddings only (skip schema and seeds)
- `--no-embeddings` — same as `--seeds-only`

**Prerequisites checked:**
- `go` — required to run the schema initializer
- `curl` + Ollama reachable at `--ollama-url` — required for embedding generation

**Key features:**
- Schema init via `init_db_runner` (calls `db.Open()` / `RunMigrations()`)
- SQL seeds via `seed_runner` (uses `INSERT OR IGNORE` — idempotent)
- Embeddings via a self-contained Go embedder that calls Ollama's `/api/embeddings` endpoint and stores results as little-endian F32_BLOB
- Summary report showing row counts and embedding coverage per table

**Usage:**
```bash
# Full run (schema + seeds + embeddings)
bash scripts/seed.sh --db ./dist/devforge.db

# Seeds only (skip embeddings)
bash scripts/seed.sh --db ./dist/devforge.db --seeds-only

# Re-generate embeddings only
bash scripts/seed.sh --db ./dist/devforge.db --embeddings-only

# Custom Ollama endpoint
bash scripts/seed.sh --db ./dist/devforge.db --ollama-url http://localhost:11434 --ollama-model nomic-embed-text
```

---

## init_db_runner/main.go

Go program that opens (or creates) a libSQL database at a given path and runs all schema migrations.

**Why a Go runner instead of `sqlite3` CLI?**
- The schema uses libSQL-specific types (`F32_BLOB`, `libsql_vector_idx`) that plain SQLite does not understand
- Uses the same `db.Open()` / `RunMigrations()` code as the MCP server, guaranteeing consistency

**Usage:**
```bash
CGO_ENABLED=1 go run ./scripts/init_db_runner -db ./dist/devforge.db
```

---

## seed_runner/main.go

Go program that applies one or more SQL seed files to a libSQL database using the `go-libsql` driver.

**Why a Go runner instead of `sqlite3` CLI?**
- `go-libsql` understands `F32_BLOB` and `libsql_vector_idx` types
- Custom SQL splitter correctly handles semicolons inside single-quoted string literals (e.g. CSS/HTML snippets in INSERT statements) and line comments (`--`)

**Key features:**
- Supports multiple `-sql` flags to apply several seed files in one run
- Skips empty statements; logs warnings for individual statement errors but continues
- Handles `INSERT OR IGNORE` idempotently

**Usage:**
```bash
CGO_ENABLED=1 go run ./scripts/seed_runner -db ./dist/devforge.db -sql db/seeds/001_patterns.sql -sql db/seeds/002_architectures.sql
```

---

## Dependency Graph

```
install-dpf.sh
  └─→ downloads from https://github.com/GustavoGutierrez/devpixelforge/releases

install.sh
  ├─→ CGO_ENABLED=1 go build ./cmd/devforge-mcp/
  ├─→ CGO_ENABLED=1 go build ./cmd/devforge/
  ├─→ init_db_runner    (for DB init)
  ├─→ seed_runner       (for optional seeding)
  └─→ writes ~/.config/devforge/config.json

seed.sh
  ├─→ init_db_runner    (schema init)
  ├─→ seed_runner       (SQL seeds)
  └─→ self-contained embedder (Ollama embeddings)

setup-mcp-client.sh
  └─→ merges JSON into target client config files

link-skills.sh
  └─→ ln -s .agents/skills/<skill> .claude/skills/<skill>
```
