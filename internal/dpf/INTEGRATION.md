# DevPixelForge (dpf) Integration Guide

This guide explains how to integrate the DevPixelForge Rust processing engine into any Go project. It covers the required files, setup steps, and usage patterns.

> **DevPixelForge** is the underlying Rust engine that powers all image, video, and audio operations in DevForge MCP. Source: [github.com/GustavoGutierrez/devpixelforge](https://github.com/GustavoGutierrez/devpixelforge)

---

## 1. What You Need

### Rust Binary (required)

```
devpixelforge/target/release/dpf
```

This is the processing engine. It must be:
- Compiled for the target platform (`make build-rust` in DevPixelForge).
- Accessible by the Go process at runtime.

> For distribution without system dependencies, use the static binary:
> `make build-rust-static` → `devpixelforge/target/x86_64-unknown-linux-musl/release/dpf`

### Go Client (required)

The Go client lives in `internal/dpf/` of devforge-mcp. It contains:
- All job types (`ResizeJob`, `OptimizeJob`, `VideoTranscodeJob`, `AudioNormalizeJob`, etc.)
- `Client` — one-shot client (one process per operation)
- `StreamClient` — streaming client (persistent Rust process, recommended for servers)
- Convenience methods: `Resize`, `Optimize`, `Convert`, `Favicon`, `Placeholder`
- **Video**: `VideoTranscode`, `VideoResize`, `VideoTrim`, `VideoThumbnail`, `VideoProfile`
- **Audio**: `AudioTranscode`, `AudioTrim`, `AudioNormalize`, `AudioSilenceTrim`

---

## 2. How to Integrate into Your Go Project

### Step 1 — Copy the Files

```bash
# In your Go project
cp /path/to/devpixelforge/target/release/dpf ./bin/
cp -r /path/to/devforge-mcp/internal/dpf ./internal/dpf
```

Recommended project structure:

```
my-project/
├── bin/
│   └── dpf                     # Rust binary
├── internal/
│   └── dpf/
│       ├── dpf.go              # main Go client
│       ├── audio_job.go        # audio job types
│       ├── video_job.go        # video job types
│       └── image_suite_job.go  # image job types
└── ...
```

### Step 2 — Use in Your Code

#### Option A: One-shot Client (simple, for few operations)

```go
import "my-project/internal/dpf"

client := dpf.NewClient("./bin/dpf")
client.SetTimeout(60 * time.Second)

// Responsive resize
result, err := client.Resize(ctx, "uploads/photo.jpg", "public/img", []uint32{320, 640, 1280})

// Video transcode
result, err = client.VideoTranscode(ctx, &dpf.VideoTranscodeJob{
    Operation: "video_transcode",
    Input:     "video.mp4",
    Output:    "video.webm",
    Codec:     "vp9",
})

// Audio normalize
result, err = client.AudioNormalize(ctx, &dpf.AudioNormalizeJob{
    Operation:  "audio_normalize",
    Input:      "audio.mp3",
    Output:     "audio_normalized.mp3",
    TargetLUFS: -14.0,
})
```

#### Option B: StreamClient (recommended for MCP servers or high load)

`StreamClient` starts the Rust process **once** and reuses stdin/stdout for all operations, saving ~5 ms overhead per operation.

```go
import "my-project/internal/dpf"

// Initialize once (e.g. at server startup)
sc, err := dpf.NewStreamClient("./bin/dpf")
if err != nil {
    log.Fatal(err)
}
defer sc.Close()

// Send jobs concurrently (StreamClient is thread-safe)
result, err := sc.Execute(&dpf.ResizeJob{
    Operation: "resize",
    Input:     "uploads/photo.jpg",
    OutputDir: "public/img",
    Widths:    []uint32{320, 640, 1280},
})

result, err = sc.Execute(&dpf.VideoTranscodeJob{
    Operation: "video_transcode",
    Input:     "video.mp4",
    Output:    "video.webm",
    Codec:     "vp9",
})
```

---

## 3. Integration Pattern for a Go MCP Server

```go
type MCPServer struct {
    dpf *dpf.StreamClient
    // ... other fields
}

func NewMCPServer(binaryPath string) (*MCPServer, error) {
    sc, err := dpf.NewStreamClient(binaryPath)
    if err != nil {
        return nil, fmt.Errorf("failed to start dpf: %w", err)
    }
    return &MCPServer{dpf: sc}, nil
}

func (s *MCPServer) Shutdown() {
    s.dpf.Close()
}

// Handler for "optimize_images" tool
func (s *MCPServer) handleOptimizeImages(ctx context.Context, params json.RawMessage) (any, error) {
    var req struct {
        Paths     []string `json:"paths"`
        OutputDir string   `json:"output_dir"`
    }
    if err := json.Unmarshal(params, &req); err != nil {
        return nil, err
    }

    result, err := s.dpf.Execute(&dpf.OptimizeJob{
        Operation: "optimize",
        Inputs:    req.Paths,
        OutputDir: &req.OutputDir,
        AlsoWebp:  true,
    })
    if err != nil {
        return nil, fmt.Errorf("optimization failed: %w", err)
    }

    return result, nil
}

// Handler for "video_transcode" tool
func (s *MCPServer) handleVideoTranscode(ctx context.Context, params json.RawMessage) (any, error) {
    var req struct {
        Input   string `json:"input"`
        Output  string `json:"output"`
        Codec   string `json:"codec"`
        Bitrate string `json:"bitrate,omitempty"`
    }
    if err := json.Unmarshal(params, &req); err != nil {
        return nil, err
    }

    return s.dpf.VideoTranscode(&dpf.VideoTranscodeJob{
        Operation: "video_transcode",
        Input:     req.Input,
        Output:    req.Output,
        Codec:     req.Codec,
        Bitrate:   req.Bitrate,
    })
}

// Handler for "audio_normalize" tool
func (s *MCPServer) handleAudioNormalize(ctx context.Context, params json.RawMessage) (any, error) {
    var req struct {
        Input      string  `json:"input"`
        Output     string  `json:"output"`
        TargetLUFS float64 `json:"target_lufs"`
    }
    if err := json.Unmarshal(params, &req); err != nil {
        return nil, err
    }

    return s.dpf.AudioNormalize(&dpf.AudioNormalizeJob{
        Operation:  "audio_normalize",
        Input:      req.Input,
        Output:     req.Output,
        TargetLUFS: req.TargetLUFS,
    })
}
```

---

## 4. Integration Checklist

- [ ] `dpf` binary copied and executable (`chmod +x`)
- [ ] `dpf` package copied to your project's `internal/dpf/`
- [ ] Binary path configured correctly (absolute or relative to process CWD)
- [ ] `StreamClient` initialized at startup and closed on shutdown (`defer sc.Close()`)
- [ ] Adequate timeout set for heavy operations (`client.SetTimeout(120 * time.Second)`)
- [ ] FFmpeg installed for video/audio operations

---

## 5. Available Job Types

| Type | `operation` value | Use case |
|------|------------------|----------|
| **Images** |||
| `ResizeJob` | `"resize"` | Generate responsive variants |
| `OptimizeJob` | `"optimize"` | Compress PNG/JPEG + generate WebP |
| `ConvertJob` | `"convert"` | Format conversion (SVG→PNG, PNG→WebP, etc.) |
| `FaviconJob` | `"favicon"` | Generate favicon pack from SVG/PNG |
| `SpriteJob` | `"sprite"` | Create sprite sheet + CSS |
| `PlaceholderJob` | `"placeholder"` | LQIP, dominant color, CSS gradient |
| `BatchJob` | `"batch"` | Run multiple operations in parallel |
| **Video** |||
| `VideoTranscodeJob` | `"video_transcode"` | Transcode video to a different codec |
| `VideoResizeJob` | `"video_resize"` | Resize video dimensions |
| `VideoTrimJob` | `"video_trim"` | Trim video by time range |
| `VideoThumbnailJob` | `"video_thumbnail"` | Extract frame as image |
| `VideoProfileJob` | `"video_profile"` | Apply a web-optimized encoding profile |
| **Audio** |||
| `AudioTranscodeJob` | `"audio_transcode"` | Convert between audio formats |
| `AudioTrimJob` | `"audio_trim"` | Trim audio by time range |
| `AudioNormalizeJob` | `"audio_normalize"` | Normalize loudness to target LUFS |
| `AudioSilenceTrimJob` | `"audio_silence_trim"` | Remove leading/trailing silence |

---

## 6. System Requirements

### FFmpeg (required for video/audio)

dpf uses the FFmpeg CLI for video and audio processing. Make sure FFmpeg is installed:

```bash
# Linux
sudo apt install ffmpeg

# macOS
brew install ffmpeg

# Verify
ffmpeg -version
```

Minimum recommended version: **FFmpeg 6.0+**
