<p align="center">
  <img src="devforge.png" width="300" alt="DevForge MCP" />
</p>

[![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)](https://github.com/GustavoGutierrez/devforge-mcp)
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

## Configuration

Shared config file between the MCP server and the CLI:

```
~/.config/devforge/config.json
```

Override with the `DEV_FORGE_CONFIG` environment variable.

## System Requirements

- **Go 1.24+** with CGO enabled (`CGO_ENABLED=1`)
- **FFmpeg 6.0+** (for video/audio operations)
- **Rust toolchain** (only if recompiling the `dpf` binary from [DevPixelForge](https://github.com/GustavoGutierrez/devpixelforge))

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
