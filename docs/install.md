# Installing DevForge

## Prerequisites

| Dependency | Required | Notes |
|---|---|---|
| Go 1.24+ | For source builds only | Not needed for Homebrew installs |
| FFmpeg | Optional | Only required for video/audio tools via `dpf` |
| Rust toolchain | No | Only if rebuilding `dpf` from source |

## Homebrew (recommended)

Supported packaged targets:

- Linux amd64
- macOS arm64

### First install

```bash
brew install GustavoGutierrez/devforge/devforge
```

This installs all three binaries (`devforge`, `devforge-mcp`, `dpf`) into your Homebrew prefix and automatically creates a starter config at `~/.config/devforge/config.json` if one does not already exist.

### Upgrading

```bash
brew update && brew upgrade gustavogutierrez/devforge/devforge
```

Homebrew will download the new bundle, swap the Cellar entry, and clean up the old version automatically.

### Optional: FFmpeg

DevForge does not depend on FFmpeg at install time. Video and audio tools (`video_transcode`, `audio_normalize`, etc.) call `ffmpeg` as a subprocess at runtime. Install it only if you need those tools:

```bash
brew install ffmpeg
```

## From source

```bash
git clone https://github.com/GustavoGutierrez/devforge.git
cd devforge
go build ./...
bash scripts/install-dpf.sh
```

This places `devforge` and `devforge-mcp` in the project root and `dpf` at `bin/dpf`. Add both to your `$PATH` or copy them to a directory already on it.

> Requires Go 1.24+. CGO is not required.

## Config file

The config file is created automatically on first Homebrew install. For source builds, create it manually:

```text
~/.config/devforge/config.json
```

```json
{
  "gemini_api_key": "",
  "image_model": "gemini-2.5-flash-image"
}
```

| Field | Purpose |
|---|---|
| `gemini_api_key` | Required only for Gemini-powered tools (`generate_ui_image`) |
| `image_model` | Gemini image model override (default: `gemini-2.5-flash-image`) |

Override the config path with the `DEV_FORGE_CONFIG` environment variable.

## Notes

- DevForge is fully stateless — no SQLite, libSQL, FTS5, Ollama, or embedding setup required.
- `dpf` must be executable and on `$PATH` (or colocated with `devforge-mcp`) for media tools to work.
- All non-AI tools work offline and without any API key.
