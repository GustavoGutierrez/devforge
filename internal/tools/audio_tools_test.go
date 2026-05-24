package tools_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"dev-forge-mcp/internal/tools"
)

// ─── AudioTranscode ──────────────────────────────────────────────────────────

func TestAudioTranscode_NilDPF(t *testing.T) {
	s := &tools.Server{}
	got := s.AudioTranscode(context.Background(), tools.AudioTranscodeInput{
		Input: "in.wav", Output: "out.mp3", Codec: "mp3",
	})
	if !strings.Contains(got, "dpf binary not available") {
		t.Fatalf("expected dpf-not-available error, got: %s", got)
	}
}

func TestAudioTranscode_Success(t *testing.T) {
	fake := &fakeStreamer{result: successResult("/out/out.mp3")}
	s := &tools.Server{DPF: fake}
	got := s.AudioTranscode(context.Background(), tools.AudioTranscodeInput{
		Input: "in.wav", Output: "/out/out.mp3", Codec: "mp3",
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("invalid JSON: %s", got)
	}
	if out["success"] != true {
		t.Errorf("expected success=true, got %v", out["success"])
	}
	if out["output_path"] != "/out/out.mp3" {
		t.Errorf("unexpected output_path: %v", out["output_path"])
	}
}

// ─── AudioTrim ───────────────────────────────────────────────────────────────

func TestAudioTrim_NilDPF(t *testing.T) {
	s := &tools.Server{}
	got := s.AudioTrim(context.Background(), tools.AudioTrimInput{
		Input: "in.mp3", Output: "out.mp3", Start: 0, End: 30,
	})
	if !strings.Contains(got, "dpf binary not available") {
		t.Fatalf("expected dpf-not-available error, got: %s", got)
	}
}

func TestAudioTrim_Success(t *testing.T) {
	fake := &fakeStreamer{result: successResult("/out/trimmed.mp3")}
	s := &tools.Server{DPF: fake}
	got := s.AudioTrim(context.Background(), tools.AudioTrimInput{
		Input: "in.mp3", Output: "/out/trimmed.mp3", Start: 0, End: 30,
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("invalid JSON: %s", got)
	}
	if out["success"] != true {
		t.Errorf("expected success=true, got %v", out["success"])
	}
}

// ─── AudioNormalize ──────────────────────────────────────────────────────────

func TestAudioNormalize_NilDPF(t *testing.T) {
	s := &tools.Server{}
	got := s.AudioNormalize(context.Background(), tools.AudioNormalizeInput{
		Input: "in.mp3", Output: "out.mp3", TargetLUFS: -14,
	})
	if !strings.Contains(got, "dpf binary not available") {
		t.Fatalf("expected dpf-not-available error, got: %s", got)
	}
}

func TestAudioNormalize_Success(t *testing.T) {
	fake := &fakeStreamer{result: successResult("/out/normalized.mp3")}
	s := &tools.Server{DPF: fake}
	got := s.AudioNormalize(context.Background(), tools.AudioNormalizeInput{
		Input: "in.mp3", Output: "/out/normalized.mp3", TargetLUFS: -14,
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("invalid JSON: %s", got)
	}
	if out["success"] != true {
		t.Errorf("expected success=true, got %v", out["success"])
	}
}

// ─── AudioSilenceTrim ────────────────────────────────────────────────────────

func TestAudioSilenceTrim_NilDPF(t *testing.T) {
	s := &tools.Server{}
	got := s.AudioSilenceTrim(context.Background(), tools.AudioSilenceTrimInput{
		Input: "in.mp3", Output: "out.mp3",
	})
	if !strings.Contains(got, "dpf binary not available") {
		t.Fatalf("expected dpf-not-available error, got: %s", got)
	}
}

func TestAudioSilenceTrim_Success(t *testing.T) {
	fake := &fakeStreamer{result: successResult("/out/silence_trimmed.mp3")}
	s := &tools.Server{DPF: fake}
	got := s.AudioSilenceTrim(context.Background(), tools.AudioSilenceTrimInput{
		Input: "in.mp3", Output: "/out/silence_trimmed.mp3", ThresholdDB: -40,
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("invalid JSON: %s", got)
	}
	if out["success"] != true {
		t.Errorf("expected success=true, got %v", out["success"])
	}
}
