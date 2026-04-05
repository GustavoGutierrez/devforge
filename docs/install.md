# Installing DevForge

## Prerequisites

| Dependency | Required | Notes |
|---|---|---|
| Go 1.24+ | Yes | Build from source |
| FFmpeg | For media tools | Video/audio operations |
| Rust toolchain | No | Only if rebuilding `dpf` |

## Homebrew

Supported packaged targets:

- Linux amd64
- macOS arm64

```bash
brew tap GustavoGutierrez/devforge
brew install GustavoGutierrez/devforge/devforge
```

Bundle contents:

- `devforge`
- `devforge-mcp`
- `dpf`

## From source

```bash
git clone https://github.com/GustavoGutierrez/devforge.git
cd devforge
go build ./...
bash scripts/install-dpf.sh
chmod +x bin/dpf
./devforge-mcp
```

## Config file

Path:

```text
~/.config/devforge/config.json
```

Example:

```json
{
  "gemini_api_key": "",
  "image_model": "gemini-2.5-flash-image"
}
```

| Field | Purpose |
|---|---|
| `gemini_api_key` | Required for Gemini-powered image/doc tools |
| `image_model` | Gemini image model override |

Override path with `DEV_FORGE_CONFIG`.

## Notes

- DevForge no longer uses a bundled database.
- No SQLite, libSQL, FTS5, Ollama, or embedding setup is required.
- `dpf` must be executable and colocated with the binaries for bundled installs.
