package tools_test

import (
	"context"
	"strings"
	"testing"

	"dev-forge-mcp/internal/tools"
)

// UI2MD does not use DPF — it uses the Gemini API.
// The analogous "missing dependency" error path is geminiAPIKey == "".

func TestUI2MD_MissingGeminiKey(t *testing.T) {
	s := &tools.Server{}
	got := s.UI2MD(context.Background(), tools.UI2MDInput{
		ImagePath: "/some/screenshot.png",
	}, "" /* geminiAPIKey */, "" /* imageModel */)
	if !strings.Contains(got, "Gemini API key not configured") {
		t.Fatalf("expected Gemini-key error, got: %s", got)
	}
}

func TestUI2MD_MissingImagePath(t *testing.T) {
	s := &tools.Server{}
	// With a key but no image_path — should get image_path required error.
	got := s.UI2MD(context.Background(), tools.UI2MDInput{
		ImagePath: "",
	}, "fake-key", "")
	if !strings.Contains(got, "image_path is required") {
		t.Fatalf("expected image_path error, got: %s", got)
	}
}

func TestUI2MD_ImageFileNotFound(t *testing.T) {
	s := &tools.Server{}
	// Valid key + valid image_path but the file does not exist on disk.
	// The handler tries to read the file before calling Gemini, so this
	// exercises the os.ReadFile error branch without any network call.
	got := s.UI2MD(context.Background(), tools.UI2MDInput{
		ImagePath: "/nonexistent/path/image.png",
	}, "fake-key", "gemini-2.5-flash-preview-05-20")
	if !strings.Contains(got, "failed to read image file") {
		t.Fatalf("expected file-read error, got: %s", got)
	}
}
