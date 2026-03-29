// Package filetools_test provides table-driven tests for filetools.
package filetools_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dev-forge-mcp/internal/tools/filetools"
)

// ── helpers ──────────────────────────────────────────────────────────────────

// mustParseJSON unmarshals a JSON string into a map; fatal if it fails.
func mustParseJSON(t *testing.T, s string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("invalid JSON %q: %v", s, err)
	}
	return m
}

// isError reports whether the JSON result is an error response.
func isError(m map[string]any) bool {
	_, ok := m["error"]
	return ok
}

func str(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func num(m map[string]any, key string) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}

// ── file_checksum ─────────────────────────────────────────────────────────────

func TestChecksum(t *testing.T) {
	// Create a temp file with known content.
	dir := t.TempDir()
	filePath := filepath.Join(dir, "sample.txt")
	content := "hello world\n"
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	tests := []struct {
		name      string
		in        filetools.ChecksumInput
		wantError bool
		wantKeys  []string
	}{
		{
			name:     "sha256 happy path",
			in:       filetools.ChecksumInput{Path: filePath, Algorithm: "sha256"},
			wantKeys: []string{"checksum", "algorithm", "path", "size_bytes"},
		},
		{
			name:     "md5 happy path",
			in:       filetools.ChecksumInput{Path: filePath, Algorithm: "md5"},
			wantKeys: []string{"checksum", "algorithm"},
		},
		{
			name:     "sha512 happy path",
			in:       filetools.ChecksumInput{Path: filePath, Algorithm: "sha512"},
			wantKeys: []string{"checksum", "algorithm"},
		},
		{
			name:      "missing path",
			in:        filetools.ChecksumInput{},
			wantError: true,
		},
		{
			name:      "nonexistent file",
			in:        filetools.ChecksumInput{Path: "/no/such/file.txt"},
			wantError: true,
		},
		{
			name:      "unknown algorithm",
			in:        filetools.ChecksumInput{Path: filePath, Algorithm: "crc32"},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := filetools.Checksum(context.Background(), tc.in)
			m := mustParseJSON(t, got)
			if tc.wantError {
				if !isError(m) {
					t.Errorf("expected error, got: %s", got)
				}
				return
			}
			if isError(m) {
				t.Fatalf("unexpected error: %s", got)
			}
			for _, k := range tc.wantKeys {
				if _, ok := m[k]; !ok {
					t.Errorf("missing key %q in result: %s", k, got)
				}
			}
		})
	}
}

// ── file_archive ──────────────────────────────────────────────────────────────

func TestArchive(t *testing.T) {
	dir := t.TempDir()
	// Build a small source tree.
	srcDir := filepath.Join(dir, "src")
	os.MkdirAll(filepath.Join(srcDir, "sub"), 0o755)
	os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("file a"), 0o644)
	os.WriteFile(filepath.Join(srcDir, "b.txt"), []byte("file b"), 0o644)
	os.WriteFile(filepath.Join(srcDir, "sub", "c.txt"), []byte("file c"), 0o644)
	os.WriteFile(filepath.Join(srcDir, "skip.log"), []byte("should be excluded"), 0o644)

	tests := []struct {
		name      string
		in        filetools.ArchiveInput
		wantError bool
		checkFn   func(t *testing.T, m map[string]any)
	}{
		{
			name: "create zip happy path",
			in: filetools.ArchiveInput{
				Operation: "create",
				Format:    "zip",
				Source:    srcDir,
				Output:    filepath.Join(dir, "out.zip"),
				Exclude:   []string{"*.log"},
			},
			checkFn: func(t *testing.T, m map[string]any) {
				if num(m, "files_added") < 3 {
					t.Errorf("expected at least 3 files, got %v", m["files_added"])
				}
				if str(m, "archive") == "" {
					t.Error("missing archive field")
				}
			},
		},
		{
			name: "create tar.gz happy path",
			in: filetools.ArchiveInput{
				Operation: "create",
				Format:    "tar.gz",
				Source:    srcDir,
				Output:    filepath.Join(dir, "out.tar.gz"),
				Exclude:   []string{"*.log"},
			},
			checkFn: func(t *testing.T, m map[string]any) {
				if num(m, "files_added") < 3 {
					t.Errorf("expected at least 3 files, got %v", m["files_added"])
				}
			},
		},
		{
			name: "extract zip happy path",
			in: filetools.ArchiveInput{
				Operation: "extract",
				Format:    "zip",
				Archive:   filepath.Join(dir, "out.zip"),
				Dest:      filepath.Join(dir, "extracted_zip"),
			},
			checkFn: func(t *testing.T, m map[string]any) {
				if num(m, "files_extracted") < 3 {
					t.Errorf("expected at least 3 files extracted, got %v", m["files_extracted"])
				}
			},
		},
		{
			name: "extract tar.gz happy path",
			in: filetools.ArchiveInput{
				Operation: "extract",
				Format:    "tar.gz",
				Archive:   filepath.Join(dir, "out.tar.gz"),
				Dest:      filepath.Join(dir, "extracted_targz"),
			},
			checkFn: func(t *testing.T, m map[string]any) {
				if num(m, "files_extracted") < 3 {
					t.Errorf("expected at least 3 files extracted, got %v", m["files_extracted"])
				}
			},
		},
		{
			name:      "unknown operation",
			in:        filetools.ArchiveInput{Operation: "list"},
			wantError: true,
		},
		{
			name:      "create missing source",
			in:        filetools.ArchiveInput{Operation: "create", Format: "zip", Output: filepath.Join(dir, "x.zip")},
			wantError: true,
		},
		{
			name:      "extract missing archive",
			in:        filetools.ArchiveInput{Operation: "extract", Format: "zip", Dest: filepath.Join(dir, "d")},
			wantError: true,
		},
		{
			name:      "unknown format",
			in:        filetools.ArchiveInput{Operation: "create", Format: "rar", Source: srcDir, Output: filepath.Join(dir, "x.rar")},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := filetools.Archive(context.Background(), tc.in)
			m := mustParseJSON(t, got)
			if tc.wantError {
				if !isError(m) {
					t.Errorf("expected error, got: %s", got)
				}
				return
			}
			if isError(m) {
				t.Fatalf("unexpected error: %s", got)
			}
			if tc.checkFn != nil {
				tc.checkFn(t, m)
			}
		})
	}
}

// ── file_diff ─────────────────────────────────────────────────────────────────

func TestDiff(t *testing.T) {
	dir := t.TempDir()

	fileA := filepath.Join(dir, "a.txt")
	fileB := filepath.Join(dir, "b.txt")
	os.WriteFile(fileA, []byte("line1\nline2\nline3\n"), 0o644)
	os.WriteFile(fileB, []byte("line1\nline2 modified\nline3\nline4\n"), 0o644)

	tests := []struct {
		name      string
		in        filetools.DiffInput
		wantError bool
		checkFn   func(t *testing.T, m map[string]any)
	}{
		{
			name: "file diff happy path",
			in:   filetools.DiffInput{A: fileA, B: fileB, Mode: "file", ContextLines: 1},
			checkFn: func(t *testing.T, m map[string]any) {
				if num(m, "additions") < 1 {
					t.Errorf("expected additions > 0, got: %v", m)
				}
				if num(m, "deletions") < 1 {
					t.Errorf("expected deletions > 0, got: %v", m)
				}
				diff := str(m, "diff")
				if !strings.Contains(diff, "---") || !strings.Contains(diff, "+++") {
					t.Errorf("diff missing headers: %q", diff)
				}
			},
		},
		{
			name: "text diff happy path",
			in: filetools.DiffInput{
				A:    "hello world",
				B:    "hello Go",
				Mode: "text",
			},
			checkFn: func(t *testing.T, m map[string]any) {
				if num(m, "additions") == 0 && num(m, "deletions") == 0 {
					t.Error("expected at least one addition or deletion")
				}
			},
		},
		{
			name: "identical files produce no diff",
			in: filetools.DiffInput{
				A:    "same\ncontent\n",
				B:    "same\ncontent\n",
				Mode: "text",
			},
			checkFn: func(t *testing.T, m map[string]any) {
				if num(m, "additions") != 0 || num(m, "deletions") != 0 {
					t.Errorf("expected zero changes, got: %v", m)
				}
			},
		},
		{
			name:      "nonexistent file a",
			in:        filetools.DiffInput{A: "/no/such/a.txt", B: fileB, Mode: "file"},
			wantError: true,
		},
		{
			name:      "unknown mode",
			in:        filetools.DiffInput{A: "x", B: "y", Mode: "binary"},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := filetools.Diff(context.Background(), tc.in)
			m := mustParseJSON(t, got)
			if tc.wantError {
				if !isError(m) {
					t.Errorf("expected error, got: %s", got)
				}
				return
			}
			if isError(m) {
				t.Fatalf("unexpected error: %s", got)
			}
			if tc.checkFn != nil {
				tc.checkFn(t, m)
			}
		})
	}
}

// ── file_line_endings ─────────────────────────────────────────────────────────

func TestLineEndings(t *testing.T) {
	dir := t.TempDir()
	crlfFile := filepath.Join(dir, "crlf.txt")
	mixedFile := filepath.Join(dir, "mixed.txt")
	lfFile := filepath.Join(dir, "lf.txt")

	os.WriteFile(crlfFile, []byte("line1\r\nline2\r\nline3\r\n"), 0o644)
	os.WriteFile(mixedFile, []byte("line1\r\nline2\nline3\r\n"), 0o644)
	os.WriteFile(lfFile, []byte("line1\nline2\nline3\n"), 0o644)

	tests := []struct {
		name      string
		in        filetools.LineEndingsInput
		wantError bool
		checkFn   func(t *testing.T, m map[string]any)
	}{
		{
			name: "detect crlf",
			in:   filetools.LineEndingsInput{Input: crlfFile, Mode: "file", Operation: "detect"},
			checkFn: func(t *testing.T, m map[string]any) {
				if str(m, "line_ending") != "crlf" {
					t.Errorf("expected crlf, got %q", str(m, "line_ending"))
				}
				if num(m, "crlf_count") != 3 {
					t.Errorf("expected crlf_count=3, got %v", num(m, "crlf_count"))
				}
			},
		},
		{
			name: "detect lf",
			in:   filetools.LineEndingsInput{Input: lfFile, Mode: "file", Operation: "detect"},
			checkFn: func(t *testing.T, m map[string]any) {
				if str(m, "line_ending") != "lf" {
					t.Errorf("expected lf, got %q", str(m, "line_ending"))
				}
			},
		},
		{
			name: "detect mixed",
			in:   filetools.LineEndingsInput{Input: mixedFile, Mode: "file", Operation: "detect"},
			checkFn: func(t *testing.T, m map[string]any) {
				if str(m, "line_ending") != "mixed" {
					t.Errorf("expected mixed, got %q", str(m, "line_ending"))
				}
			},
		},
		{
			name: "normalize text mode to lf",
			in: filetools.LineEndingsInput{
				Input:     "line1\r\nline2\r\n",
				Mode:      "text",
				Operation: "normalize",
				Target:    "lf",
			},
			checkFn: func(t *testing.T, m map[string]any) {
				result := str(m, "result")
				if strings.Contains(result, "\r\n") {
					t.Error("expected no CRLF in normalized output")
				}
			},
		},
		{
			name: "convert text mode to crlf",
			in: filetools.LineEndingsInput{
				Input:     "line1\nline2\n",
				Mode:      "text",
				Operation: "convert",
				Target:    "crlf",
			},
			checkFn: func(t *testing.T, m map[string]any) {
				result := str(m, "result")
				if !strings.Contains(result, "\r\n") {
					t.Error("expected CRLF in converted output")
				}
			},
		},
		{
			name: "normalize file mode",
			in: filetools.LineEndingsInput{
				Input:     crlfFile,
				Mode:      "file",
				Operation: "normalize",
				Target:    "lf",
				Output:    filepath.Join(dir, "normalized.txt"),
			},
			checkFn: func(t *testing.T, m map[string]any) {
				if str(m, "output") == "" {
					t.Error("missing output field")
				}
				if num(m, "lines_converted") == 0 {
					t.Error("expected lines_converted > 0")
				}
			},
		},
		{
			name:      "missing input",
			in:        filetools.LineEndingsInput{Mode: "file", Operation: "detect"},
			wantError: true,
		},
		{
			name:      "unknown operation",
			in:        filetools.LineEndingsInput{Input: lfFile, Mode: "file", Operation: "uppercase"},
			wantError: true,
		},
		{
			name:      "unknown target",
			in:        filetools.LineEndingsInput{Input: lfFile, Mode: "file", Operation: "convert", Target: "cr"},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := filetools.LineEndings(context.Background(), tc.in)
			m := mustParseJSON(t, got)
			if tc.wantError {
				if !isError(m) {
					t.Errorf("expected error, got: %s", got)
				}
				return
			}
			if isError(m) {
				t.Fatalf("unexpected error: %s", got)
			}
			if tc.checkFn != nil {
				tc.checkFn(t, m)
			}
		})
	}
}

// ── file_hex_view ─────────────────────────────────────────────────────────────

func TestHexView(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "binary.bin")
	data := []byte("Hello World\n\x00\x01\x02\x03\xff")
	os.WriteFile(filePath, data, 0o644)

	encoded := base64.StdEncoding.EncodeToString(data)

	tests := []struct {
		name      string
		in        filetools.HexViewInput
		wantError bool
		checkFn   func(t *testing.T, m map[string]any)
	}{
		{
			name: "file mode happy path",
			in:   filetools.HexViewInput{Input: filePath, Mode: "file", Length: 256, Width: 16},
			checkFn: func(t *testing.T, m map[string]any) {
				view := str(m, "hex_view")
				if !strings.Contains(view, "48 65 6c 6c 6f") {
					t.Errorf("hex dump missing expected hex for 'Hello': %q", view)
				}
				if num(m, "total_bytes") != float64(len(data)) {
					t.Errorf("total_bytes mismatch: got %v, want %d", m["total_bytes"], len(data))
				}
			},
		},
		{
			name: "base64 mode happy path",
			in:   filetools.HexViewInput{Input: encoded, Mode: "base64", Length: 256, Width: 16},
			checkFn: func(t *testing.T, m map[string]any) {
				if str(m, "hex_view") == "" {
					t.Error("expected non-empty hex_view")
				}
				if num(m, "bytes_shown") <= 0 {
					t.Error("expected bytes_shown > 0")
				}
			},
		},
		{
			name: "file mode with offset",
			in:   filetools.HexViewInput{Input: filePath, Mode: "file", Offset: 5, Length: 5, Width: 16},
			checkFn: func(t *testing.T, m map[string]any) {
				if num(m, "offset") != 5 {
					t.Errorf("expected offset=5, got %v", m["offset"])
				}
				if num(m, "bytes_shown") != 5 {
					t.Errorf("expected bytes_shown=5, got %v", m["bytes_shown"])
				}
			},
		},
		{
			name: "custom width",
			in:   filetools.HexViewInput{Input: filePath, Mode: "file", Width: 8},
			checkFn: func(t *testing.T, m map[string]any) {
				view := str(m, "hex_view")
				// With width=8, each line should have 8 hex bytes (24 chars + spaces).
				if view == "" {
					t.Error("expected non-empty hex_view")
				}
			},
		},
		{
			name:      "missing input",
			in:        filetools.HexViewInput{Mode: "file"},
			wantError: true,
		},
		{
			name:      "nonexistent file",
			in:        filetools.HexViewInput{Input: "/no/such/file.bin", Mode: "file"},
			wantError: true,
		},
		{
			name:      "invalid base64",
			in:        filetools.HexViewInput{Input: "not-valid-base64!!!", Mode: "base64"},
			wantError: true,
		},
		{
			name:      "unknown mode",
			in:        filetools.HexViewInput{Input: filePath, Mode: "octal"},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := filetools.HexView(context.Background(), tc.in)
			m := mustParseJSON(t, got)
			if tc.wantError {
				if !isError(m) {
					t.Errorf("expected error, got: %s", got)
				}
				return
			}
			if isError(m) {
				t.Fatalf("unexpected error: %s", got)
			}
			if tc.checkFn != nil {
				tc.checkFn(t, m)
			}
		})
	}
}
