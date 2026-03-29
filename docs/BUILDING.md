# Building devforge-mcp

DevForge MCP is a Go MCP server that exposes UI/design, image processing, and design-system management tools via the MCP stdio transport protocol. It ships alongside a companion CLI/TUI built with Bubble Tea.

This guide explains how to build the full distribution package from source.

---

## Prerequisites

### Required

| Dependency | Minimum version | Notes |
|------------|-----------------|-------|
| Go | 1.24.5 | Specified in `go.mod` |
| CGO | enabled | `CGO_ENABLED=1` — required by `go-libsql` / `go-sqlite3` |
| gcc (or equivalent C compiler) | any recent | Needed by CGO to link libsql native code |
| make | any | Used to run Makefile targets |

On Debian/Ubuntu:

```shell
sudo apt-get install build-essential
```

On macOS (Xcode command-line tools provide gcc/clang):

```shell
xcode-select --install
```

### Optional

| Dependency | Purpose |
|------------|---------|
| Ollama | Generates vector embeddings for semantic search. Without it, the server falls back to FTS5 full-text search. |
| sqlite3 CLI | Inspecting the database file manually. Not needed for building or running. |
| Rust toolchain | Only needed to rebuild `devforge-imgproc` from source. A pre-built binary lives in `bin/dpf`. |

**Ollama setup** (if you want embeddings):

```shell
# Install Ollama: https://ollama.com/download
# Then pull the embedding model used by default:
ollama pull nomic-embed-text
```

---

## Quick start

To produce a fully initialized `dist/` directory (binaries + seeded database) in one command:

```shell
make dist
```

This runs: build + schema init + SQL seeds + embedding generation (if Ollama is available).

---

## Step-by-step

### 1. Build the binaries

```shell
make build
```

This compiles both binaries into `dist/`:

- `dist/devforge-mcp` — the MCP server (stdio transport)
- `dist/devforge` — the CLI/TUI

To build only the MCP server:

```shell
make build-mcp
```

To build only the CLI/TUI:

```shell
make build-tui
```

CGO is enabled automatically by the Makefile (`CGO_ENABLED=1 go build`). Do not run `go build` directly without this flag — it will fail.

### 2. Initialize the database (schema + migrations)

```shell
make db-init
```

This runs `scripts/init_db_runner` via `go run`, which calls the project's own `db.Open()` and `RunMigrations()` functions. The schema is embedded in Go code (`internal/db/schema.go`). The operation is idempotent — safe to run multiple times.

Default database path: `dist/devforge.db`. Override with `DB_PATH`:

```shell
make db-init DB_PATH=/custom/path/devforge.db
```

### 3. Apply seeds

```shell
make db-seed
```

This applies all `db/seeds/*.sql` files in numeric order using `scripts/seed_runner`. Seeds use `INSERT OR IGNORE`, so re-running never duplicates rows. The database must exist before this step (`make db-init` first).

### 4. Generate embeddings (optional — requires Ollama)

```shell
make db-embeddings
```

This calls `scripts/seed.sh --embeddings-only`, which iterates over every `patterns` and `architectures` row where `embedding IS NULL`, calls the Ollama `/api/embeddings` endpoint, and stores the result as a little-endian `F32_BLOB` in the database.

The default model is `nomic-embed-text`. Pull it before running:

```shell
ollama pull nomic-embed-text
```

If Ollama is not reachable, the script prints a warning and exits cleanly. Embeddings can be generated later at any time by re-running `make db-embeddings`.

To use a different model or Ollama URL:

```shell
make db-embeddings OLLAMA_MODEL=mxbai-embed-large OLLAMA_URL=http://myhost:11434
```

### 5. Full dist/ structure

After `make dist` completes, the `dist/` directory contains:

```
dist/
├── devforge-mcp        # MCP server binary (stdio transport)
├── devforge            # CLI/TUI binary
├── devforge.db         # Fully initialized and seeded libSQL/SQLite database
└── dpf                 # Rust image-processing binary (copied from bin/ if present)
```

Note: `devforge-imgproc` is only copied to `dist/` if `bin/dpf` exists. The MCP server looks for the imgproc binary at `./bin/dpf` relative to its working directory when launched, not inside `dist/`. See the runtime notes below.

---

## Makefile targets reference

| Target | Description |
|--------|-------------|
| `build` | Compile both binaries into `dist/` |
| `build-mcp` | Compile the MCP server binary to `dist/devforge-mcp` |
| `build-tui` | Compile the CLI/TUI binary to `dist/devforge` |
| `install` | Build and install both binaries to `~/.local/bin/` |
| `dist` | Build binaries + fully initialize and seed the distribution DB |
| `db-init` | Create/migrate the libSQL DB. Idempotent. |
| `db-seed` | Apply all `db/seeds/*.sql` files in numeric order. Idempotent. |
| `db-embeddings` | Generate Ollama embeddings for rows with `embedding IS NULL` |
| `seed` | Full seed pipeline: `db-init` + `db-seed` + `db-embeddings` |
| `run` | Build and run the MCP server (stdio transport) |
| `tui` | Build and run the CLI/TUI |
| `test` | Run all tests with `CGO_ENABLED=1` |
| `build-rust` | Build the imgproc Rust binary (dynamic, requires Rust toolchain) |
| `build-rust-static` | Build a fully static imgproc binary (musl, no system deps) |
| `clean` | Remove `dist/` and compiled binaries |
| `help` | Show all available targets and variable defaults |

---

## Overriding variables

All variables can be overridden on the command line:

```shell
# Use a custom database path
make db-init DB_PATH=/var/lib/devforge/devforge.db

# Use a different Ollama instance and model
make db-embeddings OLLAMA_URL=http://192.168.1.10:11434 OLLAMA_MODEL=mxbai-embed-large

# Install binaries to a custom directory
make install INSTALL_DIR=/usr/local/bin

# Full dist with non-default DB path
make dist DB_PATH=/opt/devforge/devforge.db
```

Variable defaults (from the Makefile):

| Variable | Default |
|----------|---------|
| `DB_PATH` | `dist/devforge.db` |
| `OLLAMA_URL` | `http://localhost:11434` |
| `OLLAMA_MODEL` | `nomic-embed-text` |
| `INSTALL_DIR` | `~/.local/bin` |

---

## Runtime configuration

The MCP server reads `~/.config/devforge/config.json` at startup. The file is optional — missing fields fall back to defaults.

```json
{
  "ollama_url": "http://localhost:11434",
  "embedding_model": "nomic-embed-text",
  "gemini_api_key": ""
}
```

Override the config path with the `DEV_FORGE_CONFIG` environment variable:

```shell
DEV_FORGE_CONFIG=/etc/devforge/config.json ./dist/devforge-mcp
```

The `configure_gemini` MCP tool saves the API key to this file and hot-reloads it without restarting the server.

---

## Running the MCP server

The server uses the stdio transport. It reads JSON-RPC messages from stdin and writes responses to stdout. Do not run it directly in an interactive terminal.

### Attach to Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or the equivalent path on your platform:

```json
{
  "mcpServers": {
    "devforge": {
      "command": "/absolute/path/to/dist/devforge-mcp",
      "args": []
    }
  }
}
```

The server must be launched from the project root (or a directory containing `./bin/dpf`) so that the relative path to the imgproc binary resolves correctly.

### Attach to any MCP-compatible client

Any client that supports the MCP stdio transport can attach the server:

- **Transport**: `stdio`
- **Command**: absolute path to `dist/devforge-mcp`
- **Working directory**: project root (required for `./bin/dpf` to be found)

### Quick smoke test

```shell
make run
```

This builds the MCP server and starts it. Because it speaks stdio, it will appear to hang — that is normal. Send a valid MCP JSON-RPC request on stdin or attach an MCP client to interact with it.

---

## Troubleshooting

### `cgo: C compiler "gcc" not found`

CGO requires a C compiler. Install `build-essential` (Linux) or Xcode command-line tools (macOS). Verify with:

```shell
gcc --version
```

### Build fails with `CGO_ENABLED` error or undefined references to sqlite symbols

Never run `go build ./...` without `CGO_ENABLED=1`. The `go-libsql` driver requires CGO. Always use the Makefile targets, or prefix manually:

```shell
CGO_ENABLED=1 go build ./cmd/devforge-mcp/
```

### `db-seed` fails: "Database not found at dist/devforge.db"

Seeds require an initialized database. Run `make db-init` before `make db-seed`, or use `make seed` which chains all three steps.

### "Database already exists" or duplicate seed rows

The seed pipeline is idempotent. `make db-init` runs `RunMigrations()` which is safe to re-run. SQL seeds use `INSERT OR IGNORE`, so re-running `make db-seed` is harmless.

### Ollama not running — embeddings skipped

The `make db-embeddings` step (and the `seed.sh --embeddings-only` mode) check Ollama availability before starting. If the check fails, the script prints a warning and exits with code 0. Embeddings are not required for the server to start — it falls back to FTS5 full-text search.

To generate embeddings later:

```shell
# Start Ollama, then:
make db-embeddings
```

### `optimize_images` or `generate_favicon` return errors

These tools require `bin/dpf` to exist and be executable. The server does not crash without it — only those two tools will return a structured error. To fix:

```shell
chmod +x bin/dpf
```

If the binary is missing, rebuild it from source:

```shell
make build-rust        # dynamic binary (requires Rust)
make build-rust-static # fully static musl binary
```

### `generate_ui_image` fails with missing API key

Set the Gemini API key via the MCP tool at runtime (no restart needed):

```json
{"tool": "configure_gemini", "api_key": "YOUR_KEY"}
```

Or add it directly to `~/.config/devforge/config.json` before starting the server.

### FTS5 not available

The server exits at startup if the SQLite build lacks FTS5. This is rare with `go-libsql` but can occur with custom CGO builds. Run the tests to verify:

```shell
make test
```
