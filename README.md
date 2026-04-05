<p align="center">
  <img src="devforge.png" width="1024" height="340" alt="DevForge MCP" />
</p>

[![Version](https://img.shields.io/badge/version-1.1.6-blue.svg)](https://github.com/GustavoGutierrez/devforge)
[![License](https://img.shields.io/badge/license-GPL--3.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8.svg?logo=go&logoColor=white)](https://golang.org)
[![MCP](https://img.shields.io/badge/MCP-stdio-8B5CF6.svg?logo=modelcontextprotocol&logoColor=white)](https://modelcontextprotocol.io)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS-1e1e2e.svg?logo=linux&logoColor=white)](https://github.com/GustavoGutierrez/devforge)
[![CGO](https://img.shields.io/badge/CGO-required-orange.svg)](https://github.com/GustavoGutierrez/devforge)


# DevForge MCP

DevForge is a Go-powered MCP server and CLI/TUI that brings 70+ stateless developer tools directly into AI agents and terminals. It covers media processing, layout analysis, design tokens, data conversion, cryptography, HTTP, file operations, frontend/backend helpers, and code formatting — all with zero database dependencies.

---

## 📋 Table of Contents

- [Architecture Overview](#-architecture-overview)
- [Tool Surface](#-tool-surface)
  - [Design & Media Tools](#design--media-tools)
  - [Developer Utilities (48 Tools)](#developer-utilities-48-tools)
- [Installation](#-installation)
  - [Homebrew](#homebrew)
  - [From Source](#from-source)
- [MCP Client Setup](#-mcp-client-setup)
- [Configuration](#️-configuration)
- [Release Assets](#-release-assets)
- [Documentation](#-documentation)
- [License](#license)

---

## 🔧 Architecture Overview

DevForge is composed of three standalone binaries that can be used independently or together:

| Binary | Role |
|--------|------|
| `devforge-mcp` | MCP stdio server consumed by AI clients (Claude Desktop, Cursor, and any MCP-compatible client). Exposes all tools over the Model Context Protocol. |
| `devforge` | Interactive CLI/TUI powered by [Bubble Tea](https://github.com/charmbracelet/bubbletea). Provides the same tool surface from the terminal with an ergonomic interface. |
| `dpf` (DevPixelForge) | Media processing runtime for image, video, audio, and document tools. Requires [FFmpeg](https://ffmpeg.org) available in `\$PATH`. |

All three binaries are stateless — no database, no embeddings, no persistent state.

---

## 🛠️ Tool Surface

### Design & Media Tools

| Group | Tools | Description |
|-------|-------|-------------|
| **Design / UI** | `analyze_layout`, `suggest_layout`, `manage_tokens`, `suggest_color_palettes` | Layout analysis, design token planning, color palette generation |
| **Image Processing** | `optimize_images`, `generate_favicon`, `generate_ui_image`, `ui2md`, `image_resize`, `image_convert`, `image_crop`, `image_watermark`, `image_srcset`, `image_quality`, `image_placeholder`, `markdown_to_pdf` | Optimize, resize, convert, crop, watermark, generate favicons and UI images, export Markdown to PDF |
| **Video** | `video_transcode`, `video_trim`, `video_thumbnail` | Transcode, trim, extract thumbnails |
| **Audio** | `audio_normalize`, `audio_transcode`, `audio_trim` | Normalize, transcode, trim audio |

> Media tools require `dpf` and FFmpeg. See [Installation](#-installation) for setup details.

---

### Developer Utilities (48 Tools)

> Stateless, deterministic tools for everyday developer tasks — callable from AI agents via MCP or from the CLI.

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

---

## 📦 Installation

### Homebrew

Supported targets: **Linux amd64** and **macOS arm64**.

```bash
brew tap GustavoGutierrez/devforge
brew install GustavoGutierrez/devforge/devforge
```

This installs all three binaries: `devforge`, `devforge-mcp`, and `dpf`.

### From Source

```bash
git clone https://github.com/GustavoGutierrez/devforge.git
cd devforge
go build ./...
chmod +x bin/dpf
./devforge-mcp
```

> Requires Go 1.24+. For media tools, FFmpeg must be available in `\$PATH`.

Refer to [docs/install.md](docs/install.md) for full setup and MCP client configuration.

---

## ⚡ MCP Client Setup

Add the following to your MCP client configuration (e.g. `claude_desktop_config.json`, Cursor settings, or equivalent):

```json
{
  "mcpServers": {
    "devforge": {
      "command": "/path/to/devforge-mcp",
      "args": [],
      "env": {}
    }
  }
}
```

> Works with Claude Desktop, Cursor, and any MCP-compatible client using the stdio transport.

See [docs/mcp-connect.md](docs/mcp-connect.md) for platform-specific instructions and tips.

---

## ⚙️ Configuration

DevForge reads its configuration from `~/.config/devforge/config.json`. You can override the path by setting the `DEV_FORGE_CONFIG` environment variable.

```json
{
  "gemini_api_key": "",
  "image_model": "gemini-2.5-flash-image"
}
```

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `gemini_api_key` | For AI tools only | — | Required for `generate_ui_image` and `suggest_color_palettes`. All other tools work without a key. |
| `image_model` | No | `gemini-2.5-flash-image` | Gemini model used for image generation. |

> All non-AI tools are fully key-free and work offline.

---

## 🚀 Release Assets

Tagged releases publish the following artifacts:

- `devforge_<version>_linux_amd64.tar.gz`
- `devforge_<version>_darwin_arm64.tar.gz`
- `checksums.txt`

Each archive contains `devforge`, `devforge-mcp`, and `dpf`. The Homebrew formula is generated from the published checksums.

---

## 📚 Documentation

| Document | Description |
|----------|-------------|
| [docs/install.md](docs/install.md) | Installation & platform notes |
| [docs/cli-tui.md](docs/cli-tui.md) | CLI/TUI reference |
| [docs/mcp-connect.md](docs/mcp-connect.md) | MCP client connection guide |
| [docs/overview.md](docs/overview.md) | Architecture overview |
| [docs/tools/](docs/tools/) | Per-tool documentation |
| [packaging/homebrew/README.md](packaging/homebrew/README.md) | Homebrew packaging notes |

---

## License

DevForge MCP is released under the [GPL-3.0 License](LICENSE).
