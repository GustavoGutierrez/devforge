<p align="center">
  <img src="devforge.png" width="300" alt="DevForge MCP" />
</p>

[![Version](https://img.shields.io/badge/version-1.1.0-blue.svg)](https://github.com/GustavoGutierrez/devforge-mcp)
[![License](https://img.shields.io/badge/license-GPL--3.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8.svg?logo=go&logoColor=white)](https://golang.org)
[![MCP](https://img.shields.io/badge/MCP-stdio-8B5CF6.svg?logo=modelcontextprotocol&logoColor=white)](https://modelcontextprotocol.io)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS-1e1e2e.svg?logo=linux&logoColor=white)](https://github.com/GustavoGutierrez/devforge-mcp)
[![CGO](https://img.shields.io/badge/CGO-required-orange.svg)](https://github.com/GustavoGutierrez/devforge-mcp)

# DevForge MCP

**"One forge for every stage of your dev workflow."**

DevForge MCP is a Go-based MCP server that acts as a transversal intelligence layer and utility toolkit across the software development lifecycle. It exposes a rich set of tools — for code, architecture, design, media processing, and documentation — through the MCP stdio transport, making it accessible to any MCP-compatible AI client.

Built around a SQLite-backed pattern store with FTS5 search and optional vector embeddings, it provides specialized skills and sub-agents that work together to reduce friction at every phase: from initial architecture decisions to production-ready interfaces and optimized media assets.

> **Multimedia engine:** All image, video, and audio processing is powered by [DevPixelForge](https://github.com/GustavoGutierrez/devpixelforge), a Rust-based processing engine that ships as a pre-built binary alongside DevForge.

## Key Capabilities

- **Multimedia optimization** — Compress and convert images, video, and audio for the web using the DevPixelForge Rust engine (with FFmpeg).
- **Design system management** — Store, search, and retrieve UI patterns, design tokens, color palettes, and architecture diagrams.
- **Layout analysis & generation** — Audit existing layouts and generate new ones adapted to any supported frontend stack.
- **Developer utilities** — 48 stateless tools covering text encoding, data format conversion, cryptography, HTTP, date/time, file operations, frontend math, backend helpers, and code formatting — all callable by AI agents or from the CLI.
- **MCP tool surface** — Works as an MCP server so any AI client (Claude, OpenCode, Copilot, etc.) can invoke its tools via stdio.
- **CLI/TUI companion** — A Bubble Tea-based terminal interface for browsing patterns, launching audits, and configuring integrations without leaving the terminal.
- **Specialized skills** — Extends capabilities through configurable skills and sub-agents for frontend, backend, architecture, documentation, and QA.
- **Cross-stack** — A common tool for frontend, backend, infrastructure, and automation.

## Current Frontend Stack Support

The UI and design tools adapt their output to the declared stack:

- Vanilla JS/TS + modern CSS SPA (Vite 8).
- Astro, Next.js, SvelteKit, Nuxt.js.
- Tailwind CSS v4+ with the official Vite plugin:
  - Importing `@import "tailwindcss";` in a single CSS file.
  - Design tokens in CSS instead of `tailwind.config.js`.

## Components

- `cmd/devforge-mcp/`
  - Go MCP server with SQLite/FTS5, optionally libSQL.
  - Current tools:
    - **UI/Design**: `analyze_layout`, `suggest_layout`, `manage_tokens`, `store_pattern`, `list_patterns`, `suggest_color_palettes`
    - **Images**: `optimize_images`, `generate_favicon`, `generate_ui_image` (requires Gemini API key)
    - **Video**: `video_transcode`, `video_resize`, `video_trim`, `video_thumbnail`, `video_profile`
    - **Audio**: `audio_transcode`, `audio_trim`, `audio_normalize`, `audio_silence_trim`
    - **Config**: `configure_gemini`, `ui2md`
    - **Text & Encoding**: `text_escape`, `text_slug`, `text_uuid`, `text_base64`, `text_url_encode`, `text_normalize`, `text_case`
    - **Data Format**: `data_json_format`, `data_yaml_convert`, `data_csv_convert`, `data_jsonpath`, `data_schema_validate`, `data_diff`
    - **Security & Cryptography**: `crypto_hash`, `crypto_hmac`, `crypto_jwt`, `crypto_password`, `crypto_keygen`, `crypto_random`, `crypto_mask`
    - **HTTP & Networking**: `http_request`, `http_curl_convert`, `http_webhook_replay`, `http_signed_url`, `http_url_parse`
    - **Date & Time**: `time_convert`, `time_diff`, `time_cron`, `time_date_range`
    - **File & Archive**: `file_checksum`, `file_archive`, `file_diff`, `file_line_endings`, `file_hex_view`
    - **Frontend Utilities**: `frontend_color`, `frontend_css_unit`, `frontend_breakpoint`, `frontend_regex`, `frontend_locale_format`, `frontend_icu_format`
    - **Backend Utilities**: `backend_sql_format`, `backend_conn_string`, `backend_log_parse`, `backend_env_inspect`, `backend_mq_payload`
    - **Code Utilities**: `code_format`, `code_metrics`, `code_template`

- `cmd/devforge/`
  - Go CLI/TUI built with Bubble Tea for:
    - Browsing patterns and architectures.
    - Launching layout audits.
    - Generating layouts, images, and favicons.
    - Processing video and audio.
    - Exploring color palettes.
    - Configuring integrations (Gemini API key, etc.) from the Settings view.

- `db/devforge.db`
  - SQLite with tables for:
    - `patterns`, `architectures`, `tokens`, `audits`, `assets`, `palettes`.
  - FTS5 virtual tables for efficient full-text search.

- `internal/dpf/`
  - Go bridge to the DevPixelForge Rust multimedia processing engine.
  - Binary: `bin/dpf`.
  - Supports: images (resize, optimize, convert, favicon), video (transcode, resize, trim, thumbnail, profile), audio (transcode, trim, normalize, silence_trim).
  - Requires FFmpeg for video/audio operations.
  - See [`internal/dpf/INTEGRATION.md`](internal/dpf/INTEGRATION.md).

## Developer Utilities (48 Tools)

Stateless, deterministic tools for everyday developer tasks — callable from AI agents via MCP or from the CLI.

| Group | Tools | Description |
|-------|-------|-------------|
| **Text & Encoding** | `text_escape`, `text_slug`, `text_uuid`, `text_base64`, `text_url_encode`, `text_normalize`, `text_case` | String transformations and encoding operations |
| **Data Format** | `data_json_format`, `data_yaml_convert`, `data_csv_convert`, `data_jsonpath`, `data_schema_validate`, `data_diff` | Parse, convert, validate, and diff structured data |
| **Security & Cryptography** | `crypto_hash`, `crypto_hmac`, `crypto_jwt`, `crypto_password`, `crypto_keygen`, `crypto_random`, `crypto_mask` | Hashing, signing, JWT, password hashing, key generation, secret redaction |
| **HTTP & Networking** | `http_request`, `http_curl_convert`, `http_webhook_replay`, `http_signed_url`, `http_url_parse` | Execute requests, convert curl, sign URLs, parse URLs |
| **Date & Time** | `time_convert`, `time_diff`, `time_cron`, `time_date_range` | Timestamp conversion, duration math, cron parsing, date ranges |
| **File & Archive** | `file_checksum`, `file_archive`, `file_diff`, `file_line_endings`, `file_hex_view` | Checksums, zip/tar.gz, unified diff, line ending normalization, hex dump |
| **Frontend Utilities** | `frontend_color`, `frontend_css_unit`, `frontend_breakpoint`, `frontend_regex`, `frontend_locale_format`, `frontend_icu_format` | Color conversion, CSS units, breakpoints, regex testing, locale formatting |
| **Backend Utilities** | `backend_sql_format`, `backend_conn_string`, `backend_log_parse`, `backend_env_inspect`, `backend_mq_payload` | SQL formatting, DSN builder, log parsing, .env validation, MQ payloads |
| **Code Utilities** | `code_format`, `code_metrics`, `code_template` | Code formatting (Go/TS/JSON/HTML/CSS), metrics (LOC/complexity), template rendering |

All 48 tools are documented in [`docs/tools/`](docs/tools/).

## Configuration

Shared config file between the MCP server and the CLI:

```
~/.config/devforge/config.json
```

Override with the `DEV_FORGE_CONFIG` environment variable.

## Installation

### Via Homebrew (Linux & macOS)

Install all three components — `devforge` (CLI/TUI), `devforge-mcp` (MCP server), and `dpf` (media engine):

```bash
# Option 1 — Direct URL (recommended, works immediately)
brew install https://raw.githubusercontent.com/GustavoGutierrez/devforge-mcp/homebrew-tap/Formula/devforge.rb

# Option 2 — Clone the tap first, then install
# (The repo is named devforge-mcp, not homebrew-devforge, so --custom-remote is required)
brew tap --custom-remote gustavogutierrez/devforge https://github.com/GustavoGutierrez/devforge-mcp homebrew-tap
brew install devforge
```

> **Note:** The `--custom-remote` flag is needed because Homebrew's standard tap convention expects `homebrew-{name}/` directories, but this repo uses `homebrew-tap/` for the formula branch.

See the [DevForge Homebrew Tap README](homebrew-tap/README.md) for full post-install setup and troubleshooting.

### From Source

Build from source with Go 1.24+:

```bash
# Clone the repository
git clone https://github.com/GustavoGutierrez/devforge-mcp.git
cd devforge-mcp

# Build all components (requires CGO)
CGO_ENABLED=1 go build ./...

# Ensure the media processing binary is executable
chmod +x bin/dpf

# Run the MCP server
./devforge-mcp

# Or run the CLI/TUI
./devforge
```

For detailed setup instructions, see [docs/install.md](docs/install.md).

## Updating

### Via Homebrew

```bash
brew update
brew upgrade devforge
```

### From Source

```bash
git pull origin main
CGO_ENABLED=1 go build ./...
chmod +x bin/dpf
```

## System Requirements

- **Go 1.24+** with CGO enabled (`CGO_ENABLED=1`)
- **FFmpeg 6.0+** (for video/audio operations)
- **Rust toolchain** (only if recompiling the `dpf` binary from [DevPixelForge](https://github.com/GustavoGutierrez/devpixelforge))
- **Linux**: Ubuntu 22.04+ (for Homebrew bottles; building from source works on any glibc 2.17+)
- **macOS**: 12+ (Monterey or later)

## Documentation

| Doc | Description |
|-----|-------------|
| [docs/install.md](docs/install.md) | Full installation guide: build, install, configure, run |
| [docs/mcp-connect.md](docs/mcp-connect.md) | Connect DevForge to VS Code, Claude Desktop, Claude Code, OpenCode |
| [docs/cli-tui.md](docs/cli-tui.md) | CLI/TUI usage guide |
| [docs/overview.md](docs/overview.md) | High-level project overview |
| [docs/schema.md](docs/schema.md) | Database schema reference |
| [internal/dpf/INTEGRATION.md](internal/dpf/INTEGRATION.md) | How to integrate DevPixelForge into any Go project |
| [scripts/README.md](scripts/README.md) | Script reference for install, seed, and setup |
