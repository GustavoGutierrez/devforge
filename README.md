<p align="center">
  <img src="devforge.png" width="1024" height="340" alt="DevForge MCP" />
</p>

[![Version](https://img.shields.io/badge/version-2.4.7-blue.svg)](https://github.com/GustavoGutierrez/devforge)
[![License](https://img.shields.io/badge/license-GPL--3.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8.svg?logo=go&logoColor=white)](https://golang.org)
[![MCP](https://img.shields.io/badge/MCP-stdio-8B5CF6.svg?logo=modelcontextprotocol&logoColor=white)](https://modelcontextprotocol.io)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS-1e1e2e.svg?logo=linux&logoColor=white)](https://github.com/GustavoGutierrez/devforge)
[![CGO](https://img.shields.io/badge/CGO-disabled-green.svg)](https://github.com/GustavoGutierrez/devforge)

# DevForge MCP

DevForge is a Go-powered MCP server and CLI/TUI with **90 stateless tools**. It covers media processing, standards-based color code conversion and harmony generation, data conversion, cryptography, HTTP, file operations, frontend/backend helpers, and code formatting — all with zero database dependencies.

---

## 📋 Table of Contents

- [Architecture Overview](#-architecture-overview)
- [Tool Surface](#-tool-surface)
  - [Media & AI Tools](#media--ai-tools)
  - [Developer Utilities (60 Tools)](#developer-utilities-60-tools)
- [Installation](#-installation)
  - [Homebrew](#homebrew)
  - [From Source](#from-source)
- [MCP Client Setup](#-mcp-client-setup)
- [Configuration](#️-configuration)
- [Release Assets](#-release-assets)
- [Documentation](#-documentation)
- [Powered By DevPixelForge](#-powered-by-devpixelforge)
- [Contributing](#-contributing)
- [License](#license)

---

## 🔧 Architecture Overview

DevForge is composed of three standalone binaries that can be used independently or together:

| Binary | Role |
|--------|------|
| `devforge-mcp` | MCP stdio server consumed by AI clients (Claude Desktop, Cursor, and any MCP-compatible client). Exposes all tools over the Model Context Protocol. |
| `devforge` | Interactive CLI/TUI powered by [Bubble Tea](https://github.com/charmbracelet/bubbletea). Provides the same tool surface from the terminal with an ergonomic interface. |
| [`dpf` (DevPixelForge)](https://github.com/GustavoGutierrez/devpixelforge) | Media processing runtime for image, video, audio, and document tools. Video/audio tools call [FFmpeg](https://ffmpeg.org) as a subprocess — install it separately only if you need those tools. |

All three binaries are stateless — no database, no embeddings, no persistent state.

---

## 🛠️ Tool Surface

### Media & AI Tools

| Group | Tools | Description |
|-------|-------|-------------|
| **Gemini AI** | `generate_ui_image`, `ui2md`, `configure_gemini` | Generate UI images, convert screenshots to Markdown, configure Gemini API key at runtime |
| **Image Processing** | `optimize_images`, `generate_favicon`, `image_resize`, `image_convert`, `image_crop`, `image_rotate`, `image_watermark`, `image_adjust`, `image_quality`, `image_srcset`, `image_exif`, `image_placeholder`, `image_palette`, `image_sprite` | Optimize, resize, convert, crop, rotate, watermark, adjust, generate favicons, extract EXIF, generate sprites and srcsets |
| **Video** | `video_transcode`, `video_resize`, `video_trim`, `video_thumbnail`, `video_profile` | Transcode, resize, trim, extract thumbnails, inspect video profiles |
| **Audio** | `audio_normalize`, `audio_transcode`, `audio_trim`, `audio_silence_trim` | Normalize, transcode, trim, remove silence from audio |
| **Document** | `markdown_to_pdf` | Export Markdown to PDF |
| **Color Utilities** | `color_code_convert`, `color_harmony_palette`, `css_gradient_generate` | Convert color codes across HEX/RGB/HSL/HSV/HWB/XYZ/LAB/LCH/OKLAB/OKLCH, generate harmony-based palettes, and build CSS linear/radial gradients |

> Media tools require `dpf` (bundled). Video and audio tools additionally require FFmpeg in `$PATH` — install with `brew install ffmpeg` if needed.

---

### Developer Utilities (60 Tools)

> Stateless, deterministic tools for everyday developer tasks — callable from AI agents via MCP or from the CLI.

| Group | Tools | Description |
|-------|-------|-------------|
| **Text & Encoding** | `text_escape`, `text_slug`, `text_uuid`, `text_base64`, `text_url_encode`, `text_normalize`, `text_case`, `text_stats` | String transformations, UUID generation, Base64, URL encoding, text normalization, case conversion, word/character counting |
| **Data Format** | `json_format`, `data_yaml_convert`, `data_csv_convert`, `data_jsonpath`, `data_schema_validate`, `data_diff`, `fake_data` | Parse, convert, validate, diff structured data, generate fake data from JSON Schema |
| **Security & Cryptography** | `crypto_hash`, `crypto_hmac`, `jwt`, `crypto_password`, `crypto_keygen`, `crypto_random`, `crypto_mask`, `password_generate` | Hashing (SHA256/512, MD5, SHA1), HMAC, JWT encode/decode/verify, password hashing (bcrypt/argon2), key generation, secure random values, secret redaction, secure password generation |
| **HTTP & Networking** | `http_request`, `http_curl_convert`, `http_webhook_replay`, `http_signed_url`, `http_url_parse` | Execute requests, convert curl, sign URLs, parse URLs |
| **Date & Time** | `time_convert`, `time_diff`, `time_cron`, `time_date_range`, `current_date`, `current_week`, `week_number`, `calendar` | Timestamp conversion, duration math, cron parsing, date ranges, current date info, week days list, week number calculation, monthly calendar |
| **File & Archive** | `file_checksum`, `file_archive`, `file_diff`, `file_line_endings`, `file_hex_view` | Checksums, zip/tar.gz, unified diff, line ending normalization, hex dump |
| **Frontend Utilities** | `generate_text_diff`, `convert_css_units`, `check_wcag_contrast`, `calculate_aspect_ratio`, `convert_string_cases`, `frontend_svg_optimize`, `frontend_image_base64`, `frontend_color`, `frontend_css_unit`, `frontend_breakpoint`, `regex_test`, `frontend_locale_format`, `frontend_icu_format` | Diff generation, batch CSS unit conversion, WCAG checks, aspect ratio helpers, string case conversion, SVG optimization, image-to-Base64 encoding, color conversion, CSS units, breakpoints, regex testing, locale/ICU formatting |
| **Backend Utilities** | `sql_format`, `backend_conn_string`, `backend_log_parse`, `backend_env_inspect`, `backend_mq_payload`, `backend_cidr_subnet` | SQL formatting, DSN builder, log parsing, .env validation, MQ payloads, and CIDR subnet calculations |
| **Code Utilities** | `code_json_to_types`, `code_ast_explorer`, `code_format`, `code_metrics`, `code_template` | JSON-to-code type generation, JS/TS AST outline, code formatting, metrics (LOC/complexity), template rendering |

All **60 developer utilities** (and **90 total tools** including media/AI) are documented in [`docs/tools/`](docs/tools/).

---

## 📦 Installation

### Homebrew

Supported targets: **Linux amd64** and **macOS arm64**.

```bash
brew install GustavoGutierrez/devforge/devforge
```

This installs all three binaries (`devforge`, `devforge-mcp`, `dpf`) and creates a starter config at `~/.config/devforge/config.json`.

To upgrade to a newer version:

```bash
brew update && brew upgrade gustavogutierrez/devforge/devforge
```

> For video/audio tools, FFmpeg is an optional dependency — install separately with `brew install ffmpeg`.

### From Source

```bash
git clone https://github.com/GustavoGutierrez/devforge.git
cd devforge
go build ./...
```

> Requires Go 1.24+. CGO is not required. For media tools, `dpf` and FFmpeg must be available in `$PATH`.

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
| `gemini_api_key` | For AI tools only | — | Required for `generate_ui_image` (and any future Gemini-backed tool). All other tools work without a key. |
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

## ⚡ Powered By DevPixelForge

DevForge's media pipeline — image, video, audio, and document processing — is powered by **[DevPixelForge (dpf)](https://github.com/GustavoGutierrez/devpixelforge)**, a high-performance multimedia processing engine written in Rust with a Go client for seamless integration.

> **"Transform pixels at the speed of Rust."**

[![DevPixelForge](https://img.shields.io/badge/DevPixelForge-dpf-orange.svg?logo=rust&logoColor=white)](https://github.com/GustavoGutierrez/devpixelforge)

`dpf` is bundled with every DevForge release — no separate installation required. It handles image optimization, format conversion, PDF generation, and more through a streaming subprocess protocol that keeps the Go server stateless and dependency-free at the OS level.

---

## 🤝 Contributing

DevForge is an open project and contributions are welcome.

The aim is to give AI coding agents and professional developers a **token-efficient, deterministic utility belt** — tools that return precise, structured results instead of forcing agents to write ad-hoc code for common tasks. Every new tool reduces hallucination surface, cuts token usage, and makes agent-generated code more reliable.

**We are particularly interested in contributions that:**

- Add new stateless tools useful for everyday frontend and backend web development
- Improve token efficiency in AI agent workflows — less prompt engineering, more reliable outputs
- Replace non-deterministic LLM reasoning with precise, auditable utility functions (hashing, formatting, validation, transformation)
- Cover common patterns that professional developers repeat daily: SQL formatting, JWT handling, log parsing, image optimization, environment validation, and similar
- Improve the CLI/TUI experience for developers who prefer working outside an AI client

If you use DevForge in your agent setup and notice a gap — a tool you keep writing by hand — open an issue or submit a PR. The best tools come from real workflows.

### Principal Contributor

<a href="https://github.com/GustavoGutierrez">
  <img src="https://avatars.githubusercontent.com/u/3159203?v=4" width="64" alt="Gustavo Gutierrez" />
</a>

**[Gustavo Gutierrez](https://github.com/GustavoGutierrez)** — Author and principal maintainer of DevForge MCP and [DevPixelForge](https://github.com/GustavoGutierrez/devpixelforge).

---

## License

DevForge MCP is released under the [GPL-3.0 License](LICENSE).
