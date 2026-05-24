package tools_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"dev-forge-mcp/internal/tools"
)

// ─── VideoTranscode ──────────────────────────────────────────────────────────

func TestVideoTranscode_NilDPF(t *testing.T) {
	s := &tools.Server{}
	got := s.VideoTranscode(context.Background(), tools.VideoTranscodeInput{
		Input: "in.mp4", Output: "out.mp4", Codec: "h264",
	})
	if !strings.Contains(got, "dpf binary not available") {
		t.Fatalf("expected dpf-not-available error, got: %s", got)
	}
}

func TestVideoTranscode_Success(t *testing.T) {
	fake := &fakeStreamer{result: successResult("/out/out.mp4")}
	s := &tools.Server{DPF: fake}
	got := s.VideoTranscode(context.Background(), tools.VideoTranscodeInput{
		Input: "in.mp4", Output: "/out/out.mp4", Codec: "h264",
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("invalid JSON: %s", got)
	}
	if out["success"] != true {
		t.Errorf("expected success=true, got %v", out["success"])
	}
	if out["output_path"] != "/out/out.mp4" {
		t.Errorf("unexpected output_path: %v", out["output_path"])
	}
}

// ─── VideoResize ─────────────────────────────────────────────────────────────

func TestVideoResize_NilDPF(t *testing.T) {
	s := &tools.Server{}
	got := s.VideoResize(context.Background(), tools.VideoResizeInput{
		Input: "in.mp4", Output: "out.mp4", Width: 1280,
	})
	if !strings.Contains(got, "dpf binary not available") {
		t.Fatalf("expected dpf-not-available error, got: %s", got)
	}
}

func TestVideoResize_Success(t *testing.T) {
	fake := &fakeStreamer{result: successResult("/out/resized.mp4")}
	s := &tools.Server{DPF: fake}
	got := s.VideoResize(context.Background(), tools.VideoResizeInput{
		Input: "in.mp4", Output: "/out/resized.mp4", Width: 1280,
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("invalid JSON: %s", got)
	}
	if out["success"] != true {
		t.Errorf("expected success=true, got %v", out["success"])
	}
}

// ─── VideoTrim ───────────────────────────────────────────────────────────────

func TestVideoTrim_NilDPF(t *testing.T) {
	s := &tools.Server{}
	got := s.VideoTrim(context.Background(), tools.VideoTrimInput{
		Input: "in.mp4", Output: "out.mp4", Start: 0, End: 10,
	})
	if !strings.Contains(got, "dpf binary not available") {
		t.Fatalf("expected dpf-not-available error, got: %s", got)
	}
}

func TestVideoTrim_Success(t *testing.T) {
	fake := &fakeStreamer{result: successResult("/out/trimmed.mp4")}
	s := &tools.Server{DPF: fake}
	got := s.VideoTrim(context.Background(), tools.VideoTrimInput{
		Input: "in.mp4", Output: "/out/trimmed.mp4", Start: 0, End: 10,
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("invalid JSON: %s", got)
	}
	if out["success"] != true {
		t.Errorf("expected success=true, got %v", out["success"])
	}
}

// ─── VideoThumbnail ──────────────────────────────────────────────────────────

func TestVideoThumbnail_NilDPF(t *testing.T) {
	s := &tools.Server{}
	got := s.VideoThumbnail(context.Background(), tools.VideoThumbnailInput{
		Input: "in.mp4", Output: "thumb.png", Timestamp: "25%",
	})
	if !strings.Contains(got, "dpf binary not available") {
		t.Fatalf("expected dpf-not-available error, got: %s", got)
	}
}

func TestVideoThumbnail_Success(t *testing.T) {
	fake := &fakeStreamer{result: successResult("/out/thumb.png")}
	s := &tools.Server{DPF: fake}
	got := s.VideoThumbnail(context.Background(), tools.VideoThumbnailInput{
		Input: "in.mp4", Output: "/out/thumb.png", Timestamp: "25%",
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("invalid JSON: %s", got)
	}
	if out["success"] != true {
		t.Errorf("expected success=true, got %v", out["success"])
	}
}

// ─── VideoProfile ────────────────────────────────────────────────────────────

func TestVideoProfile_NilDPF(t *testing.T) {
	s := &tools.Server{}
	got := s.VideoProfile(context.Background(), tools.VideoProfileInput{
		Input: "in.mp4", Output: "out.mp4", Profile: "web-low",
	})
	if !strings.Contains(got, "dpf binary not available") {
		t.Fatalf("expected dpf-not-available error, got: %s", got)
	}
}

func TestVideoProfile_Success(t *testing.T) {
	fake := &fakeStreamer{result: successResult("/out/profile.mp4")}
	s := &tools.Server{DPF: fake}
	got := s.VideoProfile(context.Background(), tools.VideoProfileInput{
		Input: "in.mp4", Output: "/out/profile.mp4", Profile: "web-low",
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("invalid JSON: %s", got)
	}
	if out["success"] != true {
		t.Errorf("expected success=true, got %v", out["success"])
	}
}
