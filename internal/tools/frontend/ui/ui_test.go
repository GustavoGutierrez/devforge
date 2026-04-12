package ui_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dev-forge-mcp/internal/tools/frontend/ui"
)

func parseMap(t *testing.T, raw string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("invalid JSON %q: %v", raw, err)
	}
	return m
}

func TestSVGOptimize(t *testing.T) {
	ctx := context.Background()
	raw := `<?xml version="1.0"?><svg><!--comment--><metadata>foo</metadata><g></g><path d="M0 0"/></svg>`
	res := ui.SVGOptimize(ctx, ui.SVGOptimizeInput{SVG: raw})
	m := parseMap(t, res)
	if _, ok := m["error"]; ok {
		t.Fatalf("unexpected error: %v", m["error"])
	}
	optimized := m["optimized_svg"].(string)
	if strings.Contains(optimized, "comment") || strings.Contains(optimized, "metadata") {
		t.Fatalf("expected metadata/comments removed, got: %s", optimized)
	}
	if strings.Contains(optimized, "<g></g>") {
		t.Fatalf("expected empty tag removed, got: %s", optimized)
	}
}

func TestSVGOptimizeErrors(t *testing.T) {
	ctx := context.Background()
	res := ui.SVGOptimize(ctx, ui.SVGOptimizeInput{SVG: ""})
	m := parseMap(t, res)
	if _, ok := m["error"]; !ok {
		t.Fatalf("expected error for empty svg")
	}
}

func TestImageBase64(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "icon.png")
	if err := os.WriteFile(imgPath, []byte("test-image-content"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	res := ui.ImageBase64(ctx, ui.ImageBase64Input{Path: imgPath, DataURI: true, MimeType: "image/png"})
	m := parseMap(t, res)
	if _, ok := m["error"]; ok {
		t.Fatalf("unexpected error: %v", m["error"])
	}
	if m["mime_type"].(string) != "image/png" {
		t.Fatalf("unexpected mime type: %v", m["mime_type"])
	}
	if !strings.HasPrefix(m["data_uri"].(string), "data:image/png;base64,") {
		t.Fatalf("unexpected data uri: %v", m["data_uri"])
	}
}

func TestImageBase64Errors(t *testing.T) {
	ctx := context.Background()
	res := ui.ImageBase64(ctx, ui.ImageBase64Input{Path: "/does/not/exist.png", DataURI: true})
	m := parseMap(t, res)
	if _, ok := m["error"]; !ok {
		t.Fatalf("expected error for missing file")
	}
}
