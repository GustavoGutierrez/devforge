package tools_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"dev-forge-mcp/internal/tools"
)

// ─── ImageCrop ───────────────────────────────────────────────────────────────

func TestImageCrop_NilDPF(t *testing.T) {
	s := &tools.Server{}
	got := s.ImageCrop(context.Background(), tools.ImageCropInput{
		Input: "in.png", Output: "out.png", Width: 10, Height: 10,
	})
	if !strings.Contains(got, "dpf binary not available") {
		t.Fatalf("expected dpf-not-available error, got: %s", got)
	}
}

func TestImageCrop_Success(t *testing.T) {
	fake := &fakeStreamer{result: successResult("/out/crop.png")}
	s := &tools.Server{DPF: fake}
	got := s.ImageCrop(context.Background(), tools.ImageCropInput{
		Input: "in.png", Output: "/out/crop.png", X: 0, Y: 0, Width: 100, Height: 100,
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("invalid JSON: %s", got)
	}
	if out["success"] != true {
		t.Errorf("expected success=true, got %v", out["success"])
	}
	if out["output_path"] != "/out/crop.png" {
		t.Errorf("unexpected output_path: %v", out["output_path"])
	}
}

// ─── ImageRotate ─────────────────────────────────────────────────────────────

func TestImageRotate_NilDPF(t *testing.T) {
	s := &tools.Server{}
	got := s.ImageRotate(context.Background(), tools.ImageRotateInput{
		Input: "in.png", Output: "out.png", Angle: 90,
	})
	if !strings.Contains(got, "dpf binary not available") {
		t.Fatalf("expected dpf-not-available error, got: %s", got)
	}
}

func TestImageRotate_Success(t *testing.T) {
	fake := &fakeStreamer{result: successResult("/out/rotated.png")}
	s := &tools.Server{DPF: fake}
	got := s.ImageRotate(context.Background(), tools.ImageRotateInput{
		Input: "in.png", Output: "/out/rotated.png", Angle: 90,
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("invalid JSON: %s", got)
	}
	if out["success"] != true {
		t.Errorf("expected success=true, got %v", out["success"])
	}
}

// ─── ImageWatermark ──────────────────────────────────────────────────────────

func TestImageWatermark_NilDPF(t *testing.T) {
	s := &tools.Server{}
	got := s.ImageWatermark(context.Background(), tools.ImageWatermarkInput{
		Input: "in.png", Output: "out.png", Text: "hello",
	})
	if !strings.Contains(got, "dpf binary not available") {
		t.Fatalf("expected dpf-not-available error, got: %s", got)
	}
}

func TestImageWatermark_Success(t *testing.T) {
	fake := &fakeStreamer{result: successResult("/out/wm.png")}
	s := &tools.Server{DPF: fake}
	got := s.ImageWatermark(context.Background(), tools.ImageWatermarkInput{
		Input: "in.png", Output: "/out/wm.png", Text: "© 2024",
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("invalid JSON: %s", got)
	}
	if out["success"] != true {
		t.Errorf("expected success=true, got %v", out["success"])
	}
}

// ─── ImageAdjust ─────────────────────────────────────────────────────────────

func TestImageAdjust_NilDPF(t *testing.T) {
	s := &tools.Server{}
	got := s.ImageAdjust(context.Background(), tools.ImageAdjustInput{
		Input: "in.png", Output: "out.png", Brightness: 10,
	})
	if !strings.Contains(got, "dpf binary not available") {
		t.Fatalf("expected dpf-not-available error, got: %s", got)
	}
}

func TestImageAdjust_Success(t *testing.T) {
	fake := &fakeStreamer{result: successResult("/out/adj.png")}
	s := &tools.Server{DPF: fake}
	got := s.ImageAdjust(context.Background(), tools.ImageAdjustInput{
		Input: "in.png", Output: "/out/adj.png", Brightness: 20,
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("invalid JSON: %s", got)
	}
	if out["success"] != true {
		t.Errorf("expected success=true, got %v", out["success"])
	}
}

// ─── ImageQuality ────────────────────────────────────────────────────────────

func TestImageQuality_NilDPF(t *testing.T) {
	s := &tools.Server{}
	got := s.ImageQuality(context.Background(), tools.ImageQualityInput{
		Input: "in.png", Output: "out.png", TargetSizeKB: 100,
	})
	if !strings.Contains(got, "dpf binary not available") {
		t.Fatalf("expected dpf-not-available error, got: %s", got)
	}
}

func TestImageQuality_Success(t *testing.T) {
	fake := &fakeStreamer{result: successResult("/out/quality.png")}
	s := &tools.Server{DPF: fake}
	got := s.ImageQuality(context.Background(), tools.ImageQualityInput{
		Input: "in.png", Output: "/out/quality.png", TargetSizeKB: 200,
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("invalid JSON: %s", got)
	}
	if out["success"] != true {
		t.Errorf("expected success=true, got %v", out["success"])
	}
}

// ─── ImageSrcset ─────────────────────────────────────────────────────────────

func TestImageSrcset_NilDPF(t *testing.T) {
	s := &tools.Server{}
	got := s.ImageSrcset(context.Background(), tools.ImageSrcsetInput{
		Input: "in.png", OutputDir: "/out",
	})
	if !strings.Contains(got, "dpf binary not available") {
		t.Fatalf("expected dpf-not-available error, got: %s", got)
	}
}

func TestImageSrcset_Success(t *testing.T) {
	fake := &fakeStreamer{result: successResult("/out/img-320.webp")}
	s := &tools.Server{DPF: fake}
	got := s.ImageSrcset(context.Background(), tools.ImageSrcsetInput{
		Input: "in.png", OutputDir: "/out", Widths: []int{320, 640},
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("invalid JSON: %s", got)
	}
	if out["success"] != true {
		t.Errorf("expected success=true, got %v", out["success"])
	}
	if _, ok := out["variants"]; !ok {
		t.Error("expected variants field in output")
	}
}

// ─── ImageExif ───────────────────────────────────────────────────────────────

func TestImageExif_NilDPF(t *testing.T) {
	s := &tools.Server{}
	got := s.ImageExif(context.Background(), tools.ImageExifInput{
		Input: "in.png", ExifOp: "strip",
	})
	if !strings.Contains(got, "dpf binary not available") {
		t.Fatalf("expected dpf-not-available error, got: %s", got)
	}
}

func TestImageExif_Success(t *testing.T) {
	fake := &fakeStreamer{result: successResult("/out/stripped.png")}
	s := &tools.Server{DPF: fake}
	got := s.ImageExif(context.Background(), tools.ImageExifInput{
		Input: "in.png", Output: "/out/stripped.png", ExifOp: "strip",
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("invalid JSON: %s", got)
	}
	if out["success"] != true {
		t.Errorf("expected success=true, got %v", out["success"])
	}
}

// ─── ImageResize ─────────────────────────────────────────────────────────────

func TestImageResize_NilDPF(t *testing.T) {
	s := &tools.Server{}
	got := s.ImageResize(context.Background(), tools.ImageResizeInput{
		Input: "in.png", OutputDir: "/out",
	})
	if !strings.Contains(got, "dpf binary not available") {
		t.Fatalf("expected dpf-not-available error, got: %s", got)
	}
}

func TestImageResize_Success(t *testing.T) {
	fake := &fakeStreamer{result: successResult("/out/img-640.png")}
	s := &tools.Server{DPF: fake}
	got := s.ImageResize(context.Background(), tools.ImageResizeInput{
		Input: "in.png", OutputDir: "/out", Widths: []int{640},
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("invalid JSON: %s", got)
	}
	if out["success"] != true {
		t.Errorf("expected success=true, got %v", out["success"])
	}
}

// ─── ImageConvert ────────────────────────────────────────────────────────────

func TestImageConvert_NilDPF(t *testing.T) {
	s := &tools.Server{}
	got := s.ImageConvert(context.Background(), tools.ImageConvertInput{
		Input: "in.png", Output: "out.webp", Format: "webp",
	})
	if !strings.Contains(got, "dpf binary not available") {
		t.Fatalf("expected dpf-not-available error, got: %s", got)
	}
}

func TestImageConvert_Success(t *testing.T) {
	fake := &fakeStreamer{result: successResult("/out/out.webp")}
	s := &tools.Server{DPF: fake}
	got := s.ImageConvert(context.Background(), tools.ImageConvertInput{
		Input: "in.png", Output: "/out/out.webp", Format: "webp",
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("invalid JSON: %s", got)
	}
	if out["success"] != true {
		t.Errorf("expected success=true, got %v", out["success"])
	}
}

// ─── ImagePlaceholder ────────────────────────────────────────────────────────

func TestImagePlaceholder_NilDPF(t *testing.T) {
	s := &tools.Server{}
	got := s.ImagePlaceholder(context.Background(), tools.ImagePlaceholderInput{
		Input: "in.png",
	})
	if !strings.Contains(got, "dpf binary not available") {
		t.Fatalf("expected dpf-not-available error, got: %s", got)
	}
}

func TestImagePlaceholder_Success(t *testing.T) {
	fake := &fakeStreamer{result: successResult("/out/placeholder.png")}
	s := &tools.Server{DPF: fake}
	got := s.ImagePlaceholder(context.Background(), tools.ImagePlaceholderInput{
		Input: "in.png",
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("invalid JSON: %s", got)
	}
	if out["success"] != true {
		t.Errorf("expected success=true, got %v", out["success"])
	}
}

// ─── ImagePalette ────────────────────────────────────────────────────────────

func TestImagePalette_NilDPF(t *testing.T) {
	s := &tools.Server{}
	got := s.ImagePalette(context.Background(), tools.ImagePaletteInput{
		Input: "in.png", OutputDir: "/out",
	})
	if !strings.Contains(got, "dpf binary not available") {
		t.Fatalf("expected dpf-not-available error, got: %s", got)
	}
}

func TestImagePalette_Success(t *testing.T) {
	fake := &fakeStreamer{result: successResult("/out/palette.png")}
	s := &tools.Server{DPF: fake}
	got := s.ImagePalette(context.Background(), tools.ImagePaletteInput{
		Input: "in.png", OutputDir: "/out",
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("invalid JSON: %s", got)
	}
	if out["success"] != true {
		t.Errorf("expected success=true, got %v", out["success"])
	}
}

// ─── ImageSprite ─────────────────────────────────────────────────────────────

func TestImageSprite_NilDPF(t *testing.T) {
	s := &tools.Server{}
	got := s.ImageSprite(context.Background(), tools.ImageSpriteInput{
		Inputs: []string{"a.png", "b.png"}, Output: "sprite.png",
	})
	if !strings.Contains(got, "dpf binary not available") {
		t.Fatalf("expected dpf-not-available error, got: %s", got)
	}
}

func TestImageSprite_Success(t *testing.T) {
	fake := &fakeStreamer{result: successResult("/out/sprite.png")}
	s := &tools.Server{DPF: fake}
	got := s.ImageSprite(context.Background(), tools.ImageSpriteInput{
		Inputs: []string{"a.png", "b.png"}, Output: "/out/sprite.png",
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("invalid JSON: %s", got)
	}
	if out["success"] != true {
		t.Errorf("expected success=true, got %v", out["success"])
	}
}
