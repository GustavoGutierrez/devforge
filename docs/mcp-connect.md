# Connecting to devforge-mcp

How to attach an MCP client to the server and how to send tool calls from the terminal.

---

## Finding Your devforge-mcp Installation Paths

Before configuring any MCP client, you need to know two paths:

### 1. The Binary Path

Find where `devforge-mcp` is installed:

```bash
# Most reliable method - uses your shell's PATH
which devforge-mcp

# If Homebrew is installed (Linuxbrew on Linux, Homebrew on macOS)
# Common Homebrew locations:
# - Linux:      /home/linuxbrew/.linuxbrew/bin/devforge-mcp
# - macOS ARM:  /opt/homebrew/bin/devforge-mcp
# - macOS Intel: /usr/local/bin/devforge-mcp

# Manual search in common locations
ls -la ~/.local/bin/devforge-mcp 2>/dev/null || \
ls -la /usr/local/bin/devforge-mcp 2>/dev/null || \
ls -la /opt/homebrew/bin/devforge-mcp 2>/dev/null || \
ls -la /home/linuxbrew/.linuxbrew/bin/devforge-mcp 2>/dev/null
```

**Homebrew installation paths by platform:**

| Platform | Path |
|----------|------|
| Linux (Linuxbrew) | `/home/linuxbrew/.linuxbrew/bin/devforge-mcp` |
| macOS Apple Silicon | `/opt/homebrew/bin/devforge-mcp` |
| macOS Intel | `/usr/local/bin/devforge-mcp` |
| Manual install | `~/.local/bin/devforge-mcp` |

### 2. The Config Path

Find where the config file is stored:

```bash
# Check standard locations
ls -la ~/.config/devforge/config.json 2>/dev/null

# Check Homebrew locations (if installed via Homebrew)
ls -la /home/linuxbrew/.linuxbrew/etc/devforge/config.json 2>/dev/null  # Linux
ls -la /opt/homebrew/etc/devforge/config.json 2>/dev/null               # macOS ARM
ls -la /usr/local/etc/devforge/config.json 2>/dev/null                  # macOS Intel

# Or check the DEV_FORGE_CONFIG environment variable
echo $DEV_FORGE_CONFIG
```

**Config file priority:**
1. `DEV_FORGE_CONFIG` environment variable (if set)
2. Existing Homebrew config location (auto-detected)
3. `~/.config/devforge/config.json` (XDG default)

---

## Transport: stdio — no host, no port

`devforge-mcp` uses the **MCP stdio transport**. There is no TCP socket, no
HTTP endpoint, and no port to connect to. The protocol works as follows:

1. The MCP client **spawns** `devforge-mcp` as a child process.
2. JSON-RPC 2.0 messages are exchanged over the process's **stdin / stdout**.
3. The server exits when the client closes the pipe.

This means you cannot point `curl` at a host/port — you write JSON-RPC messages
to stdin and read responses from stdout.

---

## Connecting from an MCP client

### VS Code — GitHub Copilot MCP

Create `.vscode/mcp.json` in any workspace (recommended for project-specific setup):

```json
{
  "servers": {
    "devforge": {
      "type": "stdio",
      "command": "/home/YOUR_USERNAME/.local/bin/devforge-mcp",
      "args": [],
      "env": {
        "DEV_FORGE_CONFIG": "/home/YOUR_USERNAME/.config/devforge/config.json"
      }
    }
  }
}
```

Or for a global setup, edit `~/.config/Code/User/mcp.json` (VS Code user settings).

**Quick setup with the CLI:**

```bash
# Use the interactive setup wizard
./devforge

# Or use the shell script
bash scripts/setup-mcp-client.sh
# Select option 1 for VS Code
```

---

### OpenCode

[OpenCode](https://opencode.ai) reads MCP server config from `~/.config/opencode/config.json`
(global) or from an `opencode.json` file at the root of any project (project-local).

**Global config** `~/.config/opencode/config.json`:

```json
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "devforge": {
      "type": "local",
      "command": ["/home/YOUR_USERNAME/.local/bin/devforge-mcp"],
      "environment": {
        "DEV_FORGE_CONFIG": "/home/YOUR_USERNAME/.config/devforge/config.json"
      }
    }
  }
}
```

**Project-local config** `opencode.json` (at workspace root):

```json
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "devforge": {
      "type": "local",
      "command": ["/home/YOUR_USERNAME/.local/bin/devforge-mcp"],
      "environment": {
        "DEV_FORGE_CONFIG": "/home/YOUR_USERNAME/.config/devforge/config.json"
      }
    }
  }
}
```

After saving, run `opencode` in the terminal — the `devforge` tools will be available
in the session automatically.

> **Working directory**: OpenCode launches the server from the current project
> directory. Make sure `./bin/dpf` is reachable from there, or set
> an absolute `DEVFORGE_IMGPROC_PATH` in `environment` if you add support for it.
> The simplest approach is to keep a `bin/dpf` symlink in any
> project that uses image tools:
>
> ```bash
> mkdir -p bin
> ln -sf ~/.local/bin/dpf bin/dpf
> ```

**Quick setup with the CLI:**

```bash
# Use the interactive setup wizard
./devforge

# Or use the shell script
bash scripts/setup-mcp-client.sh
# Select option 4 for OpenCode
```

---

### Claude Code

[Claude Code](https://docs.anthropic.com/en/docs/claude-code) supports MCP servers
through a global user config or a project-level file.

**Option A — Use the CLI (recommended):**

```bash
# Interactive setup
./devforge

# Or via shell script
bash scripts/setup-mcp-client.sh
# Select option 3 for Claude Code
```

**Option B — Manual setup with `claude mcp add`:**

```bash
claude mcp add devforge /home/YOUR_USERNAME/.local/bin/devforge-mcp \
  -e DEV_FORGE_CONFIG=/home/YOUR_USERNAME/.config/devforge/config.json
```

This writes the entry to `~/.claude.json` automatically. To verify:

```bash
claude mcp list
```

**Option C — Project-level `.mcp.json`** (checked into the repo, shared with the team):

Create `.mcp.json` at the workspace root:

```json
{
  "mcpServers": {
    "devforge": {
      "command": "/home/YOUR_USERNAME/.local/bin/devforge-mcp",
      "args": [],
      "env": {
        "DEV_FORGE_CONFIG": "/home/YOUR_USERNAME/.config/devforge/config.json"
      }
    }
  }
}
```

**Option D — Manual edit of `~/.claude.json`:**

```json
{
  "mcpServers": {
    "devforge": {
      "command": "/home/YOUR_USERNAME/.local/bin/devforge-mcp",
      "args": [],
      "env": {
        "DEV_FORGE_CONFIG": "/home/YOUR_USERNAME/.config/devforge/config.json"
      }
    }
  }
}
```

After any of the above, start a new `claude` session — the `devforge` tools are
available immediately, no restart needed.

---

### Claude Desktop (macOS / Linux)

**macOS:**

Edit `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "devforge": {
      "command": "/opt/homebrew/bin/devforge-mcp",
      "args": [],
      "env": {
        "DEV_FORGE_CONFIG": "/home/USERNAME/.config/devforge/config.json"
      }
    }
  }
}
```

**Linux:**

Edit `~/.config/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "devforge": {
      "command": "/home/linuxbrew/.linuxbrew/bin/devforge-mcp",
      "args": [],
      "env": {
        "DEV_FORGE_CONFIG": "/home/USERNAME/.config/devforge/config.json"
      }
    }
  }
}
```

**Windows:**

Edit `%APPDATA%\Claude\claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "devforge": {
      "command": "C:\\Users\\USERNAME\\.local\\bin\\devforge-mcp.exe",
      "args": [],
      "env": {
        "DEV_FORGE_CONFIG": "C:\\Users\\USERNAME\\.config\\devforge\\config.json"
      }
    }
  }
}
```

After saving, restart Claude Desktop. The `devforge` tools will appear in the tool palette.

**Quick setup:**

```bash
bash scripts/setup-mcp-client.sh
# Select option 2 for Claude Desktop
```

---

### Cursor

Open **Settings → MCP → Add server**:

| Field | Value |
|-------|-------|
| Name | `devforge` |
| Type | `stdio` |
| Command | `/home/YOUR_USERNAME/.local/bin/devforge-mcp` |

---

## Using the Interactive Setup Wizard

The DevForge CLI includes an interactive TUI for MCP setup:

```bash
# Run the CLI/TUI
./devforge

# Navigate to "Setup MCP Clients" menu
# Select your IDE (OpenCode, Claude Code, or VSCode)
# Choose scope (Global or Project-local)
# Confirm the binary path (auto-detected)
# Done!
```

The wizard auto-detects:
- Homebrew installations (Linuxbrew, macOS ARM, macOS Intel)
- Binary location using `which` command
- Common installation directories

---

## Any MCP-Compatible Client

All MCP clients that support the stdio transport follow the same pattern:

| Setting | Value |
|---------|-------|
| Transport | `stdio` |
| Command | Absolute path to `devforge-mcp` binary |
| Args | _(none)_ |
| Working dir | Directory containing `./bin/dpf` |
| Env | `DEV_FORGE_CONFIG` (optional — only if overriding the default path) |

---

## Testing from the Terminal

Because the transport is stdio, "curl-style" testing means **piping JSON-RPC
messages to the binary**. Each message is a single-line JSON object followed by
a newline.

### Smoke test — initialize

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.1"}}}' \
  | devforge-mcp
```

Expected response:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2024-11-05",
    "capabilities": { "tools": {} },
    "serverInfo": { "name": "devforge", "version": "1.0.0" }
  }
}
```

---

### List available tools

```bash
printf '%s\n%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.1"}}}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' \
  | devforge-mcp
```

---

### Tool call: `suggest_color_palettes`

```bash
printf '%s\n%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.1"}}}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"suggest_color_palettes","arguments":{"mood":"calm and professional","count":3}}}' \
  | devforge-mcp
```

---

### Tool call: `analyze_layout`

```bash
LAYOUT=$(cat testdata/layouts/hero.html | jq -Rs .)

printf '%s\n%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.1"}}}' \
  "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"analyze_layout\",\"arguments\":{\"markup\":$LAYOUT,\"stack\":{\"css_mode\":\"tailwind-v4\",\"framework\":\"next\"},\"page_type\":\"landing\",\"device_focus\":\"responsive\"}}}" \
  | devforge-mcp
```

---

### Tool call: `suggest_layout`

```bash
printf '%s\n%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.1"}}}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"suggest_layout","arguments":{"description":"SaaS landing page with hero, features grid, and pricing table","stack":{"css_mode":"tailwind-v4","framework":"next"},"fidelity":"mid"}}}' \
  | devforge-mcp
```

---

### Tool call: `manage_tokens` (read)

```bash
printf '%s\n%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.1"}}}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"manage_tokens","arguments":{"mode":"read","css_mode":"tailwind-v4","scope":"colors"}}}' \
  | devforge-mcp
```

---

### Tool call: `list_patterns`

```bash
printf '%s\n%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.1"}}}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"list_patterns","arguments":{"query":"hero section","framework":"next","limit":5}}}' \
  | devforge-mcp
```

---

### Tool call: `configure_gemini`

```bash
printf '%s\n%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.1"}}}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"configure_gemini","arguments":{"api_key":"AIzaXXXXXXXXXXX"}}}' \
  | devforge-mcp
```

---

### Tool call: `store_pattern`

```bash
printf '%s\n%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.1"}}}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"store_pattern","arguments":{"name":"Hero Split","description":"Two-column hero: copy on the left, image on the right","category":"layout","framework":"next","css_mode":"tailwind-v4","snippet":"<section class=\"grid grid-cols-2 gap-8\">...</section>"}}}' \
  | devforge-mcp
```

---

## JSON-RPC Message Format Reference

Every message sent to stdin must follow the MCP JSON-RPC 2.0 envelope:

```json
{
  "jsonrpc": "2.0",
  "id": <integer>,
  "method": "<method-name>",
  "params": { ... }
}
```

Key methods:

| Method | Purpose |
|--------|---------|
| `initialize` | Handshake — must be the first message in every session |
| `tools/list` | Returns the full list of registered tools with their schemas |
| `tools/call` | Invokes a tool; `params.name` is the tool name, `params.arguments` is the input |

All tool errors are returned as valid JSON-RPC responses with the error payload
inside `result.content[0].text` as `{"error":"message"}` — the server never
crashes on tool-level errors.

---

## Tips for Multi-Message Sessions

The server keeps state (open DB, imgproc process) for the entire lifetime of the
process. For automated testing, pipe multiple newline-separated JSON-RPC messages
in a single stdin stream:

```bash
cat <<'EOF' | devforge-mcp | jq .
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.1"}}}
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"suggest_color_palettes","arguments":{"mood":"bold and energetic","count":2}}}
EOF
```

---

## Troubleshooting

### Binary not found

If your MCP client reports "command not found":

```bash
# Find the correct path
which devforge-mcp

# Verify it exists and is executable
ls -la $(which devforge-mcp)
```

### Config not found

If you see config-related errors:

```bash
# Check if config exists
ls -la ~/.config/devforge/config.json

# Or check DEV_FORGE_CONFIG
echo $DEV_FORGE_CONFIG

# Create default config if missing
mkdir -p ~/.config/devforge
echo '{"gemini_api_key":"","ollama_url":"http://localhost:11434"}' > ~/.config/devforge/config.json
```

### Image tools not working

The `dpf` (DevPixelForge) binary must be accessible:

```bash
# Check if dpf exists
ls -la bin/dpf

# If using Homebrew, ensure the binary is in PATH
export PATH="$(brew --prefix)/bin:$PATH"
```
