package dpf

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveBinaryPath(t *testing.T) {
	t.Setenv("DEVFORGE_DPF_PATH", "")

	root := t.TempDir()
	exeDir := filepath.Join(root, "dist")
	if err := os.MkdirAll(exeDir, 0o755); err != nil {
		t.Fatalf("mkdir exeDir: %v", err)
	}

	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir binDir: %v", err)
	}

	binaryPath := filepath.Join(binDir, "dpf")
	if err := os.WriteFile(binaryPath, []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() {
		if chdirErr := os.Chdir(oldWD); chdirErr != nil {
			t.Fatalf("restore wd: %v", chdirErr)
		}
	}()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir temp root: %v", err)
	}

	resolved, err := ResolveBinaryPath(exeDir)
	if err != nil {
		t.Fatalf("ResolveBinaryPath returned error: %v", err)
	}
	if resolved != binaryPath {
		t.Fatalf("ResolveBinaryPath = %q, want %q", resolved, binaryPath)
	}
}

func TestClientAndStreamClientHandleDPFProtocol(t *testing.T) {
	binaryPath := writeFakeDPF(t)

	client := NewClient(binaryPath)
	result, err := client.MarkdownToPDF(context.Background(), &MarkdownToPDFJob{Inline: true})
	if err != nil {
		t.Fatalf("MarkdownToPDF returned error: %v", err)
	}
	if result.Success {
		t.Fatalf("expected success=false, got true")
	}
	if result.Error != "unsupported theme: bad_theme" {
		t.Fatalf("result.Error = %q", result.Error)
	}

	caps, err := client.Caps(context.Background())
	if err != nil {
		t.Fatalf("Caps returned error: %v", err)
	}
	if caps.Version != "0.4.2" {
		t.Fatalf("caps.Version = %q", caps.Version)
	}
	if !caps.Features["streaming_mode"] {
		t.Fatalf("expected streaming_mode=true in caps")
	}

	sc, err := NewStreamClient(binaryPath)
	if err != nil {
		t.Fatalf("NewStreamClient returned error: %v", err)
	}
	defer func() {
		if closeErr := sc.Close(); closeErr != nil {
			t.Fatalf("Close returned error: %v", closeErr)
		}
	}()

	streamResult, err := sc.MarkdownToPDF(&MarkdownToPDFJob{Inline: true})
	if err != nil {
		t.Fatalf("stream MarkdownToPDF returned error: %v", err)
	}
	if !streamResult.Success {
		t.Fatalf("expected stream success=true, got false")
	}
	if streamResult.Operation != "markdown_to_pdf" {
		t.Fatalf("streamResult.Operation = %q", streamResult.Operation)
	}
}

func writeFakeDPF(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "dpf")
	script := `#!/usr/bin/env bash
set -euo pipefail

case "${1:-}" in
  caps)
    printf '%s\n' '{"version":"0.4.2","features":{"streaming_mode":true,"markdown_to_pdf":true}}'
    ;;
  process)
    printf '%s\n' '{"success":false,"operation":"markdown_to_pdf","error":"unsupported theme: bad_theme","elapsed_ms":1}'
    ;;
  --stream)
    while IFS= read -r line; do
      printf '%s\n' '{"success":true,"operation":"markdown_to_pdf","outputs":[],"elapsed_ms":1}'
    done
    ;;
  *)
    exit 1
    ;;
esac
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake dpf: %v", err)
	}
	return path
}
