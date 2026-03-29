# DevForge Homebrew Tap

Personal Homebrew tap for [DevForge](https://github.com/GustavoGutierrez/devforge-mcp).

## What is DevForge?

DevForge is a transversal intelligence layer for the software development lifecycle. It ships as three components:

| Binary | Type | Description |
|--------|------|-------------|
| **devforge** | CLI/TUI | Interactive design system manager (Bubble Tea) |
| **devforge-mcp** | MCP server | Model Context Protocol server (stdio transport) for AI IDEs |
| **dpf** | Dependency | [DevPixelForge](https://github.com/GustavoGutierrez/devpixelforge) — High-performance multimedia processing engine in Rust |

Together they expose tools for UI/design, image, video, and audio processing through the MCP stdio protocol.

## Installation

### Option 1 — Tap and install (macOS & Linux)

```bash
brew tap --custom-remote https://github.com/GustavoGutierrez/devforge-mcp gustavogutierrez/homebrew-devforge
brew install devforge
```

### Option 2 — Direct install from URL

```bash
brew install https://raw.githubusercontent.com/GustavoGutierrez/devforge-mcp/homebrew-tap/Formula/devforge.rb
```

### Option 3 — Manual clone

```bash
git clone https://github.com/GustavoGutierrez/devforge-mcp --branch homebrew-tap \
  "$(brew --prefix)/Library/Taps/GustavoGutierrez/homebrew-devforge"
brew install devforge
```

### Linux Requirements

On Linux (Ubuntu, Debian, Fedora), install FFmpeg separately:

```bash
# Ubuntu/Debian
sudo apt install ffmpeg

# Fedora
sudo dnf install ffmpeg
```

## After Installation

### 1. CLI/TUI

Run the interactive design system manager:

```bash
devforge
```

### 2. MCP Server (AI IDE integration)

Add `devforge-mcp` to your AI client configuration:

```json
{
  "mcpServers": {
    "devforge": {
      "command": "/opt/homebrew/bin/devforge-mcp",
      "env": {}
    }
  }
}
```

Config file locations by client:

| Client | Config path |
|--------|-------------|
| OpenCode | `~/.config/opencode/config.json` |
| Claude Desktop | `~/.config/claude-desktop.json` |
| VS Code | `~/Library/Application Support/Code/User/settings.json` |
| Cursor | `~/.config/cursor/settings.json` |

### 3. Optional: Gemini API Key

Some tools (`generate_ui_image`, `ui2md`) require a Gemini API key:

```bash
devforge config set gemini_api_key YOUR_KEY
```

Or edit `~/.config/devforge/config.json` directly.

## Uninstall

```bash
brew uninstall devforge
brew untap gustavogutierrez/homebrew-devforge
```

## Available Tools

Once installed and connected to an MCP client, the following tools are available:

### Layout & Design
- `analyze_layout` — Audit existing layouts for UX issues
- `suggest_layout` — Generate new layout variants
- `manage_tokens` — Read/write design tokens
- `store_pattern` / `list_patterns` — Pattern database
- `suggest_color_palettes` — Color palette generation

### Images (requires `dpf`)
- `optimize_images` — Compress PNG/JPEG + WebP variants
- `generate_favicon` — Full favicon pack from SVG/PNG
- `image_crop`, `image_rotate`, `image_watermark`, `image_adjust`
- `image_resize`, `image_convert`, `image_srcset`, `image_quality`
- `image_exif`, `image_placeholder`, `image_palette`, `image_sprite`

### Video (requires `dpf` + FFmpeg)
- `video_transcode`, `video_resize`, `video_trim`
- `video_thumbnail`, `video_profile`

### Audio (requires `dpf` + FFmpeg)
- `audio_transcode`, `audio_trim`, `audio_normalize`, `audio_silence_trim`

### AI-powered (requires `gemini_api_key`)
- `generate_ui_image` — Generate UI images via Gemini Vision
- `ui2md` — Analyze UI screenshot → Markdown design spec

## Updating

```bash
brew update
brew upgrade devforge
```

## Requirements

### macOS
- **macOS** 12+ (Monterey or later)
- **FFmpeg** — auto-installed via Homebrew dependency

### Linux
- **Linux** (Ubuntu, Debian, Fedora, or other distros supported by Homebrew)
- **FFmpeg** — must be installed via system package manager (see above)

## Troubleshooting

### "binary format error" on Apple Silicon
```bash
brew upgrade devforge
```

### dpf not found
The `dpf` binary is bundled automatically. If you see warnings about missing dpf, verify your Homebrew installation is up to date.

### MCP server not connecting
Verify the config entry uses the correct path. Run:
```bash
which devforge-mcp
```
And use that absolute path in your MCP client config.

## License

GPL-3.0 — see [github.com/GustavoGutierrez/devforge-mcp](https://github.com/GustavoGutierrez/devforge-mcp)
