package tools_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"dev-forge-mcp/internal/tools"
)

func TestOptimizeImages_NilDPF(t *testing.T) {
	s := &tools.Server{}
	got := s.OptimizeImages(context.Background(), tools.OptimizeImagesInput{
		Inputs: []tools.OptimizeInput{{Path: "photo.jpg"}},
	})
	if !strings.Contains(got, "dpf binary not available") {
		t.Fatalf("expected dpf-not-available error, got: %s", got)
	}
}

func TestOptimizeImages_Success(t *testing.T) {
	fake := &fakeStreamer{result: successResult("/out/photo.webp")}
	s := &tools.Server{DPF: fake}
	got := s.OptimizeImages(context.Background(), tools.OptimizeImagesInput{
		Inputs: []tools.OptimizeInput{
			{Path: "photo.jpg", Quality: 85},
		},
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("invalid JSON: %s", got)
	}
	results, ok := out["results"].([]any)
	if !ok || len(results) == 0 {
		t.Fatalf("expected non-empty results array, got: %v", out["results"])
	}
	first := results[0].(map[string]any)
	if first["source_path"] != "photo.jpg" {
		t.Errorf("unexpected source_path: %v", first["source_path"])
	}
}

func TestOptimizeImages_MultipleInputs(t *testing.T) {
	fake := &fakeStreamer{result: successResult("/out/a.webp")}
	s := &tools.Server{DPF: fake}
	got := s.OptimizeImages(context.Background(), tools.OptimizeImagesInput{
		Inputs: []tools.OptimizeInput{
			{Path: "a.png"},
			{Path: "b.png"},
		},
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("invalid JSON: %s", got)
	}
	results := out["results"].([]any)
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}
