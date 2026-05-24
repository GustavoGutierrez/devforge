package tools_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"dev-forge-mcp/internal/dpf"
	"dev-forge-mcp/internal/tools"
)

func TestGenerateFavicon_NilDPF(t *testing.T) {
	s := &tools.Server{}
	got := s.GenerateFavicon(context.Background(), tools.GenerateFaviconInput{
		SourcePath: "/assets/logo.png",
	})
	if !strings.Contains(got, "dpf binary not available") {
		t.Fatalf("expected dpf-not-available error, got: %s", got)
	}
}

func TestGenerateFavicon_Success_WithOutputs(t *testing.T) {
	// When dpf returns outputs, the handler maps them to icons.
	fake := &fakeStreamer{
		result: &dpf.JobResult{
			Success:   true,
			Operation: "favicon",
			ElapsedMs: 2,
			Outputs: []dpf.OutputFile{
				{Path: "/assets/favicons/favicon-32x32.png", Format: "png", Width: 32, Height: 32},
				{Path: "/assets/favicons/favicon-192x192.png", Format: "png", Width: 192, Height: 192},
			},
		},
	}
	s := &tools.Server{DPF: fake}
	got := s.GenerateFavicon(context.Background(), tools.GenerateFaviconInput{
		SourcePath: "/assets/logo.png",
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("invalid JSON: %s", got)
	}
	icons, ok := out["icons"].([]any)
	if !ok || len(icons) == 0 {
		t.Fatalf("expected non-empty icons array, got: %v", out["icons"])
	}
	snippets, ok := out["html_snippets"].([]any)
	if !ok || len(snippets) == 0 {
		t.Fatal("expected non-empty html_snippets")
	}
}

func TestGenerateFavicon_Success_NoOutputs(t *testing.T) {
	// When dpf returns no outputs, the handler falls back to generating expected paths.
	fake := &fakeStreamer{
		result: &dpf.JobResult{
			Success:   true,
			Operation: "favicon",
			ElapsedMs: 1,
			Outputs:   nil,
		},
	}
	s := &tools.Server{DPF: fake}
	got := s.GenerateFavicon(context.Background(), tools.GenerateFaviconInput{
		SourcePath: "/assets/logo.png",
		Sizes:      []int{32, 180},
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("invalid JSON: %s", got)
	}
	icons := out["icons"].([]any)
	if len(icons) == 0 {
		t.Error("expected fallback icons to be generated")
	}
}
