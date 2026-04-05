package tools_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"dev-forge-mcp/internal/dpf"
	"dev-forge-mcp/internal/tools"
)

func TestMarkdownToPDF_ValidatesSingleInputSource(t *testing.T) {
	srv := &tools.Server{}

	result := srv.MarkdownToPDF(context.Background(), tools.MarkdownToPDFInput{
		Input:        "doc.md",
		MarkdownText: "# inline",
		Output:       "doc.pdf",
	})

	var out map[string]string
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if out["error"] == "" {
		t.Fatalf("expected validation error, got: %s", result)
	}
}

func TestMarkdownToPDF_RequiresOutputMode(t *testing.T) {
	srv := &tools.Server{}

	result := srv.MarkdownToPDF(context.Background(), tools.MarkdownToPDFInput{
		MarkdownText: "# inline",
	})

	var out map[string]string
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if out["error"] == "" {
		t.Fatalf("expected validation error, got: %s", result)
	}
}

func TestMarkdownToPDF_StreamSuccess(t *testing.T) {
	srv := newMarkdownToPDFServer(t)

	result := srv.MarkdownToPDF(context.Background(), tools.MarkdownToPDFInput{
		MarkdownText: "# Hola\n\nPDF inline",
		Inline:       true,
		Theme:        "engineering",
	})

	var out tools.MarkdownToPDFOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v -- got: %s", err, result)
	}
	if !out.Success {
		t.Fatalf("expected success=true, got: %s", result)
	}
	if len(out.Outputs) != 1 {
		t.Fatalf("expected one output, got %d", len(out.Outputs))
	}
	if out.Outputs[0].Format != "pdf" {
		t.Fatalf("expected pdf format, got %q", out.Outputs[0].Format)
	}
	if out.Outputs[0].DataBase64 == "" {
		t.Fatalf("expected inline base64 payload, got empty output")
	}
	if out.Metadata["backend"] != "typst" {
		t.Fatalf("expected typst backend, got %#v", out.Metadata["backend"])
	}
}

func TestMarkdownToPDF_StreamFailureReturnsErrorJSON(t *testing.T) {
	srv := newMarkdownToPDFServer(t)

	result := srv.MarkdownToPDF(context.Background(), tools.MarkdownToPDFInput{
		MarkdownText: "# Hola\n\nPDF inline",
		Inline:       true,
		Theme:        "bad_theme",
	})

	var out map[string]string
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v -- got: %s", err, result)
	}
	if got := out["error"]; got != "dpf error: unsupported theme: bad_theme" {
		t.Fatalf("unexpected error: %q", got)
	}
}

func newMarkdownToPDFServer(t *testing.T) *tools.Server {
	t.Helper()

	binaryPath := writeFakeMarkdownDPF(t)
	sc, err := dpf.NewStreamClient(binaryPath)
	if err != nil {
		t.Fatalf("NewStreamClient returned error: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := sc.Close(); closeErr != nil {
			t.Fatalf("Close returned error: %v", closeErr)
		}
	})

	return &tools.Server{DPF: sc}
}

func writeFakeMarkdownDPF(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "dpf")
	script := `#!/usr/bin/env bash
set -euo pipefail

case "${1:-}" in
  caps)
    printf '%s\n' '{"version":"0.4.2","features":{"streaming_mode":true,"markdown_to_pdf":true}}'
    ;;
  --stream)
    while IFS= read -r line; do
      case "$line" in
        *'"theme":"bad_theme"'*)
          printf '%s\n' '{"success":false,"operation":"markdown_to_pdf","error":"unsupported theme: bad_theme","elapsed_ms":1}'
          ;;
        *)
          printf '%s\n' '{"success":true,"operation":"markdown_to_pdf","outputs":[{"path":"inline://test.pdf","format":"pdf","size_bytes":123,"data_base64":"UEZERkFLRQ=="}],"elapsed_ms":2,"metadata":{"backend":"typst","theme":"engineering","inline":true}}'
          ;;
      esac
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
