# Installing devforge-mcp

Step-by-step guide to build, install, and configure DevForge MCP on your system so that both the MCP server and the CLI/TUI are ready to use.

---

## Prerequisites

| Dependency | Version | Required |
|------------|---------|----------|
| Go | ≥ 1.24.5 | Yes |
| gcc / clang | any recent | Yes (CGO) |
| make | any | Yes |
| Ollama | any | No (semantic search) |
| Rust toolchain | stable | No (only to recompile dpf) |

Install the C toolchain on **Debian / Ubuntu**:

```bash
sudo apt-get install build-essential
```

Install the C toolchain on **macOS**:

```bash
xcode-select --install
```

---

## 1. Clone the repository

```bash
git clone https://github.com/GustavoGutierrez/devforge-mcp.git
cd devforge-mcp
```

---

## 2. Ensure the dpf binary is executable

A pre-built Rust binary ships in `bin/dpf`. Make it executable:

```bash
chmod +x bin/dpf
```

> If you need to recompile from source (requires Rust toolchain and the [DevPixelForge](https://github.com/GustavoGutierrez/devpixelforge) source):
>
> ```bash
> # Clone DevPixelForge alongside devforge-mcp
> git clone https://github.com/GustavoGutierrez/devpixelforge.git
> cd devpixelforge
> make build-rust         # dynamic binary
> # or for a fully static binary:
> make build-rust-static  # output: target/x86_64-unknown-linux-musl/release/dpf
> cp target/release/dpf ../devforge-mcp/bin/dpf
> chmod +x ../devforge-mcp/bin/dpf
> ```

---

## 3. Build and install the binaries

The recommended path is `~/.local/bin`. The installer script builds both binaries, copies them alongside the dpf binary, and prints the required `PATH` setup.

```bash
bash scripts/install.sh
```

Or use the Makefile (equivalent):

```bash
make install
```

To install to a custom directory:

```bash
make install INSTALL_DIR=/usr/local/bin
```

After installation, the following files are present in `INSTALL_DIR`:

```
~/.local/bin/
├── devforge-mcp        # MCP server (stdio transport)
├── devforge            # CLI/TUI
└── dpf     # Rust image-processing engine
```

---

## 4. Add `~/.local/bin` to your PATH

Add the following line to your shell profile (`~/.bashrc`, `~/.zshrc`, etc.) if it is not already there:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

Reload the shell:

```bash
source ~/.bashrc   # or source ~/.zshrc
```

Verify:

```bash
devforge-mcp --help 2>&1 || echo "binary is ready (stdio server — no --help flag)"
devforge --help
```

---

## 5. Create the configuration file

The MCP server and CLI share a single config file:

```
~/.config/devforge/config.json
```

Create it with the default structure:

```bash
mkdir -p ~/.config/devforge
cat > ~/.config/devforge/config.json <<'EOF'
{
  "gemini_api_key": "",
  "ollama_url": "http://localhost:11434",
  "embedding_model": "nomic-embed-text"
}
EOF
chmod 600 ~/.config/devforge/config.json
```

| Field | Default | Purpose |
|-------|---------|---------|
| `gemini_api_key` | `""` | Required to use `generate_ui_image` |
| `ollama_url` | `http://localhost:11434` | Ollama instance for vector embeddings |
| `embedding_model` | `nomic-embed-text` | Embedding model used by Ollama |

> Override the config path for any session with the `DEV_FORGE_CONFIG` environment variable:
>
> ```bash
> DEV_FORGE_CONFIG=/etc/devforge/config.json devforge-mcp
> ```

---

## 6. Initialize the database

The MCP server and CLI need a seeded SQLite database. From the project root:

```bash
make db-init   # creates schema + migrations
make db-seed   # applies db/seeds/*.sql (patterns, palettes, architectures)
```

Or in one step:

```bash
make dist      # build + db-init + db-seed
```

Default database path: `dist/devforge.db`. When running from `~/.local/bin`:

```bash
make db-init DB_PATH=~/.local/share/devforge/devforge.db
make db-seed  DB_PATH=~/.local/share/devforge/devforge.db
```

---

## 7. (Optional) Set up Ollama for semantic search

Without Ollama the server falls back to FTS5 full-text search. To enable vector search:

```bash
# Install Ollama: https://ollama.com/download
ollama pull nomic-embed-text

# Generate embeddings for the seeded rows
make db-embeddings
```

---

## 8. (Optional) Set up the Gemini API key

Required only for the `generate_ui_image` tool. Add the key to config:

```bash
# Edit directly
nano ~/.config/devforge/config.json
# Set "gemini_api_key": "AIzaXXXX..."
```

Or use the MCP tool `configure_gemini` from any connected MCP client (hot-reload, no restart needed).

---

## 9. (Optional) Run as a system service

### Linux — systemd user service

A ready-made unit file ships at `deploy/devforge-mcp.service`. Edit
`WorkingDirectory` to point to the directory that contains `bin/dpf`
(needed for image tools), then install:

```bash
# Edit the unit file if needed
nano deploy/devforge-mcp.service

mkdir -p ~/.config/systemd/user
cp deploy/devforge-mcp.service ~/.config/systemd/user/

systemctl --user daemon-reload
systemctl --user enable --now devforge-mcp.service

# Check status
systemctl --user status devforge-mcp.service
```

> **Important:** MCP is a stdio protocol. Running the server as a persistent
> daemon and attaching it via socket is not the standard pattern. The service
> file is provided for setups where a process manager is required (e.g. remote
> systemd socket activation). For most users, the MCP client launches the
> binary directly — see [docs/mcp-connect.md](mcp-connect.md).

### macOS — launchd user agent

Edit `deploy/com.devforge.mcp.plist` and replace `YOUR_USERNAME` with your
macOS username, then:

```bash
cp deploy/com.devforge.mcp.plist ~/Library/LaunchAgents/
launchctl load ~/Library/LaunchAgents/com.devforge.mcp.plist
launchctl start com.devforge.mcp
```

---

## Verifying the installation

Run a quick smoke test by sending a JSON-RPC `initialize` message directly on
stdin:

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"smoke-test","version":"0.1"}}}' \
  | devforge-mcp
```

A valid JSON-RPC response with `"result"` and `"serverInfo"` confirms the server
is working correctly.

For a full walkthrough of how to connect an MCP client and send tool calls, see
[docs/mcp-connect.md](mcp-connect.md).

---

## Troubleshooting

### `cgo: C compiler "gcc" not found`

Install `build-essential` (Linux) or Xcode CLI tools (macOS). Verify with `gcc --version`.

### `go build` fails with undefined sqlite symbols

Always build with `CGO_ENABLED=1`. Never run plain `go build ./...`. Use the Makefile targets or prepend the flag manually:

```bash
CGO_ENABLED=1 go build ./cmd/devforge-mcp/
```

### `db-seed` fails: database not found

Run `make db-init` (or `make db-init DB_PATH=...`) before `make db-seed`.

### `optimize_images` / `generate_favicon` return an error about the dpf binary

Ensure `bin/dpf` exists and is executable (`chmod +x bin/dpf`). The MCP server looks for it at `./bin/dpf` relative to its **working directory** — not relative to `PATH`.

### `generate_ui_image` returns "gemini_api_key not configured"

Set the key in `~/.config/devforge/config.json` or call the `configure_gemini` MCP tool.
