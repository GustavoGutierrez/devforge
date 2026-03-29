// Package filetools implements MCP tools for file and archive operations.
// All functions are stateless and safe for concurrent use.
package filetools

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/md5" // MD5 is user-requested, not used for security purposes
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// errResult returns a JSON-encoded {"error": "..."} string.
func errResult(msg string) string {
	b, _ := json.Marshal(map[string]string{"error": msg})
	return string(b)
}

// resultJSON marshals v to JSON or returns an error JSON.
func resultJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return errResult("marshal failed: " + err.Error())
	}
	return string(b)
}

// ── file_checksum ─────────────────────────────────────────────────────────────

// ChecksumInput holds the parameters for the file_checksum tool.
type ChecksumInput struct {
	Path      string // required
	Algorithm string // md5 | sha256 | sha512; default sha256
}

// ChecksumResult is the output of file_checksum.
type ChecksumResult struct {
	Checksum  string `json:"checksum"`
	Algorithm string `json:"algorithm"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
}

// Checksum computes the checksum of a file by streaming it through the
// selected hash. The entire file is never loaded into memory at once.
func Checksum(_ context.Context, in ChecksumInput) string {
	if in.Path == "" {
		return errResult("path is required")
	}
	algo := in.Algorithm
	if algo == "" {
		algo = "sha256"
	}

	var h hash.Hash
	switch algo {
	case "md5":
		h = md5.New() // MD5 is user-requested; not used for security
	case "sha256":
		h = sha256.New()
	case "sha512":
		h = sha512.New()
	default:
		return errResult("unknown algorithm: must be md5, sha256, or sha512")
	}

	f, err := os.Open(in.Path)
	if err != nil {
		return errResult("cannot open file: " + err.Error())
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return errResult("cannot stat file: " + err.Error())
	}

	if _, err := io.Copy(h, f); err != nil {
		return errResult("read error: " + err.Error())
	}

	return resultJSON(ChecksumResult{
		Checksum:  hex.EncodeToString(h.Sum(nil)),
		Algorithm: algo,
		Path:      in.Path,
		SizeBytes: stat.Size(),
	})
}

// ── file_archive ──────────────────────────────────────────────────────────────

// ArchiveInput holds the parameters for the file_archive tool.
type ArchiveInput struct {
	Operation string   // create | extract; required
	Format    string   // zip | tar.gz; default zip
	Source    string   // for create: source directory or file path
	Output    string   // for create: output archive path
	Archive   string   // for extract: archive path
	Dest      string   // for extract: destination directory
	Exclude   []string // glob patterns to exclude
}

// ArchiveCreateResult is returned when creating an archive.
type ArchiveCreateResult struct {
	Archive    string `json:"archive"`
	FilesAdded int    `json:"files_added"`
	SizeBytes  int64  `json:"size_bytes"`
}

// ArchiveExtractResult is returned when extracting an archive.
type ArchiveExtractResult struct {
	Dest           string `json:"dest"`
	FilesExtracted int    `json:"files_extracted"`
}

// Archive creates or extracts a zip or tar.gz archive.
func Archive(_ context.Context, in ArchiveInput) string {
	format := in.Format
	if format == "" {
		format = "zip"
	}
	if format != "zip" && format != "tar.gz" {
		return errResult("unknown format: must be zip or tar.gz")
	}

	switch in.Operation {
	case "create":
		return archiveCreate(in, format)
	case "extract":
		return archiveExtract(in, format)
	default:
		return errResult("unknown operation: must be create or extract")
	}
}

// matchesExclude reports whether the given path matches any of the exclusion patterns.
func matchesExclude(path string, patterns []string) bool {
	base := filepath.Base(path)
	for _, pat := range patterns {
		// Match against both the base name and the full relative path.
		if ok, _ := filepath.Match(pat, base); ok {
			return true
		}
		if ok, _ := filepath.Match(pat, path); ok {
			return true
		}
	}
	return false
}

// archiveCreate builds a new archive from in.Source and writes it to in.Output.
func archiveCreate(in ArchiveInput, format string) string {
	if in.Source == "" {
		return errResult("source is required for create operation")
	}
	if in.Output == "" {
		return errResult("output is required for create operation")
	}

	// Ensure parent directory of output exists.
	if err := os.MkdirAll(filepath.Dir(in.Output), 0o755); err != nil {
		return errResult("cannot create output directory: " + err.Error())
	}

	outFile, err := os.Create(in.Output)
	if err != nil {
		return errResult("cannot create archive file: " + err.Error())
	}
	defer outFile.Close()

	var filesAdded int

	switch format {
	case "zip":
		filesAdded, err = createZip(outFile, in.Source, in.Exclude)
	case "tar.gz":
		filesAdded, err = createTarGz(outFile, in.Source, in.Exclude)
	}
	if err != nil {
		os.Remove(in.Output)
		return errResult("archive creation failed: " + err.Error())
	}
	if err := outFile.Close(); err != nil {
		return errResult("failed to finalise archive: " + err.Error())
	}

	stat, err := os.Stat(in.Output)
	if err != nil {
		return errResult("cannot stat output archive: " + err.Error())
	}

	return resultJSON(ArchiveCreateResult{
		Archive:    in.Output,
		FilesAdded: filesAdded,
		SizeBytes:  stat.Size(),
	})
}

// createZip writes all files under root (or root itself if it's a file)
// to the given zip writer w, applying exclusion patterns.
func createZip(w io.Writer, root string, exclude []string) (int, error) {
	zw := zip.NewWriter(w)
	defer zw.Close()

	var count int

	// Determine the base to strip from paths so that zip entries are relative.
	rootInfo, err := os.Stat(root)
	if err != nil {
		return 0, err
	}

	var baseDir string
	if rootInfo.IsDir() {
		baseDir = root
	} else {
		baseDir = filepath.Dir(root)
	}

	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		if matchesExclude(relPath, exclude) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			// Write a directory entry with trailing slash.
			_, err = zw.Create(relPath + "/")
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = relPath
		header.Method = zip.Deflate

		fw, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(fw, f); err != nil {
			return err
		}
		count++
		return nil
	})

	return count, err
}

// createTarGz writes all files under root to a gzip-compressed tar archive.
func createTarGz(w io.Writer, root string, exclude []string) (int, error) {
	gzw := gzip.NewWriter(w)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	var count int

	rootInfo, err := os.Stat(root)
	if err != nil {
		return 0, err
	}

	var baseDir string
	if rootInfo.IsDir() {
		baseDir = root
	} else {
		baseDir = filepath.Dir(root)
	}

	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		if matchesExclude(relPath, exclude) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(tw, f); err != nil {
			return err
		}
		count++
		return nil
	})

	return count, err
}

// archiveExtract extracts an archive to in.Dest.
func archiveExtract(in ArchiveInput, format string) string {
	if in.Archive == "" {
		return errResult("archive is required for extract operation")
	}
	if in.Dest == "" {
		return errResult("dest is required for extract operation")
	}

	if err := os.MkdirAll(in.Dest, 0o755); err != nil {
		return errResult("cannot create dest directory: " + err.Error())
	}

	var filesExtracted int
	var err error

	switch format {
	case "zip":
		filesExtracted, err = extractZip(in.Archive, in.Dest, in.Exclude)
	case "tar.gz":
		filesExtracted, err = extractTarGz(in.Archive, in.Dest, in.Exclude)
	}
	if err != nil {
		return errResult("extraction failed: " + err.Error())
	}

	return resultJSON(ArchiveExtractResult{
		Dest:           in.Dest,
		FilesExtracted: filesExtracted,
	})
}

// sanitizeExtractPath ensures the extracted path stays inside dest to prevent
// path traversal (zip slip) attacks.
func sanitizeExtractPath(dest, name string) (string, error) {
	target := filepath.Join(dest, filepath.FromSlash(name))
	// Verify the target is inside dest after resolving ".." components.
	destAbs, err := filepath.Abs(dest)
	if err != nil {
		return "", err
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(targetAbs, destAbs+string(os.PathSeparator)) &&
		targetAbs != destAbs {
		return "", fmt.Errorf("illegal file path: %q would escape destination", name)
	}
	return target, nil
}

// extractZip extracts a zip archive into dest.
func extractZip(archive, dest string, exclude []string) (int, error) {
	r, err := zip.OpenReader(archive)
	if err != nil {
		return 0, err
	}
	defer r.Close()

	var count int
	for _, f := range r.File {
		if matchesExclude(f.Name, exclude) {
			continue
		}

		target, err := sanitizeExtractPath(dest, f.Name)
		if err != nil {
			return count, err
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(target, 0o755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return count, err
		}

		rc, err := f.Open()
		if err != nil {
			return count, err
		}

		out, err := os.Create(target)
		if err != nil {
			rc.Close()
			return count, err
		}

		_, copyErr := io.Copy(out, rc)
		out.Close()
		rc.Close()
		if copyErr != nil {
			return count, copyErr
		}
		count++
	}
	return count, nil
}

// extractTarGz extracts a .tar.gz archive into dest.
func extractTarGz(archive, dest string, exclude []string) (int, error) {
	f, err := os.Open(archive)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return 0, err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	var count int

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return count, err
		}

		if matchesExclude(header.Name, exclude) {
			continue
		}

		target, err := sanitizeExtractPath(dest, header.Name)
		if err != nil {
			return count, err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return count, err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return count, err
			}
			out, err := os.Create(target)
			if err != nil {
				return count, err
			}
			_, copyErr := io.Copy(out, tr)
			out.Close()
			if copyErr != nil {
				return count, copyErr
			}
			count++
		}
	}
	return count, nil
}

// ── file_diff ─────────────────────────────────────────────────────────────────

// DiffInput holds the parameters for the file_diff tool.
type DiffInput struct {
	A            string // file path or text content
	B            string // file path or text content
	Mode         string // file | text; default file
	ContextLines int    // default 3
}

// DiffResult is the output of file_diff.
type DiffResult struct {
	Diff      string `json:"diff"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

// Diff generates a unified diff between two texts or files.
func Diff(_ context.Context, in DiffInput) string {
	mode := in.Mode
	if mode == "" {
		mode = "file"
	}
	ctxLines := in.ContextLines
	if ctxLines < 0 {
		ctxLines = 0
	}
	if ctxLines == 0 && in.ContextLines == 0 {
		ctxLines = 3
	}

	var aText, bText, aLabel, bLabel string

	switch mode {
	case "file":
		if in.A == "" || in.B == "" {
			return errResult("a and b file paths are required")
		}
		aBytes, err := os.ReadFile(in.A)
		if err != nil {
			return errResult("cannot read file a: " + err.Error())
		}
		bBytes, err := os.ReadFile(in.B)
		if err != nil {
			return errResult("cannot read file b: " + err.Error())
		}
		aText = string(aBytes)
		bText = string(bBytes)
		aLabel = in.A
		bLabel = in.B

	case "text":
		aText = in.A
		bText = in.B
		aLabel = "a"
		bLabel = "b"

	default:
		return errResult("unknown mode: must be file or text")
	}

	diff, additions, deletions := unifiedDiff(aLabel, bLabel, aText, bText, ctxLines)
	return resultJSON(DiffResult{
		Diff:      diff,
		Additions: additions,
		Deletions: deletions,
	})
}

// unifiedDiff produces a unified diff string between aText and bText.
// It uses an LCS-based diff algorithm to detect changes.
func unifiedDiff(aLabel, bLabel, aText, bText string, contextLines int) (string, int, int) {
	aLines := splitLines(aText)
	bLines := splitLines(bText)

	edits := computeEdits(aLines, bLines)

	var additions, deletions int
	for _, e := range edits {
		switch e.kind {
		case editAdd:
			additions++
		case editDel:
			deletions++
		}
	}

	if additions == 0 && deletions == 0 {
		return "", 0, 0
	}

	hunks := buildHunks(edits, len(aLines), len(bLines), contextLines)
	var sb strings.Builder
	sb.WriteString("--- ")
	sb.WriteString(aLabel)
	sb.WriteString("\n+++ ")
	sb.WriteString(bLabel)
	sb.WriteString("\n")

	for _, h := range hunks {
		sb.WriteString(h)
	}

	return sb.String(), additions, deletions
}

// splitLines splits text into lines, preserving empty final line behaviour.
func splitLines(s string) []string {
	if s == "" {
		return []string{}
	}
	lines := strings.Split(s, "\n")
	// Remove the trailing empty element that Split adds after a final newline.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// editKind classifies a single edit operation.
type editKind int

const (
	editEq  editKind = iota // line is the same in both files
	editAdd                 // line exists only in b (addition)
	editDel                 // line exists only in a (deletion)
)

// edit represents a single line operation.
type edit struct {
	kind  editKind
	aLine int    // 0-indexed line number in a (valid for eq and del)
	bLine int    // 0-indexed line number in b (valid for eq and add)
	text  string // the line content
}

// computeEdits uses the Myers diff algorithm (simplified LCS via DP table)
// to produce a sequence of edits transforming aLines into bLines.
func computeEdits(a, b []string) []edit {
	// Build LCS table.
	la, lb := len(a), len(b)
	// dp[i][j] = length of LCS of a[:i] and b[:j]
	dp := make([][]int, la+1)
	for i := range dp {
		dp[i] = make([]int, lb+1)
	}
	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	// Back-trace to collect edits.
	var edits []edit
	i, j := la, lb
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && a[i-1] == b[j-1] {
			edits = append(edits, edit{kind: editEq, aLine: i - 1, bLine: j - 1, text: a[i-1]})
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			edits = append(edits, edit{kind: editAdd, bLine: j - 1, text: b[j-1]})
			j--
		} else {
			edits = append(edits, edit{kind: editDel, aLine: i - 1, text: a[i-1]})
			i--
		}
	}

	// Reverse to get the correct order (back-trace gives us edits in reverse).
	for lo, hi := 0, len(edits)-1; lo < hi; lo, hi = lo+1, hi-1 {
		edits[lo], edits[hi] = edits[hi], edits[lo]
	}
	return edits
}

// buildHunks assembles unified-diff hunk strings from a list of edits.
func buildHunks(edits []edit, aLen, bLen, ctxLines int) []string {
	type span struct{ start, end int }

	// Find positions of changed edits.
	var changedPositions []int
	for i, e := range edits {
		if e.kind != editEq {
			changedPositions = append(changedPositions, i)
		}
	}
	if len(changedPositions) == 0 {
		return nil
	}

	// Group changed positions into contiguous hunk ranges (with context).
	type hunkRange struct{ start, end int }
	var ranges []hunkRange
	cur := hunkRange{
		start: max(0, changedPositions[0]-ctxLines),
		end:   min(len(edits)-1, changedPositions[0]+ctxLines),
	}
	for _, pos := range changedPositions[1:] {
		newStart := max(0, pos-ctxLines)
		newEnd := min(len(edits)-1, pos+ctxLines)
		if newStart <= cur.end+1 {
			cur.end = newEnd
		} else {
			ranges = append(ranges, cur)
			cur = hunkRange{start: newStart, end: newEnd}
		}
	}
	ranges = append(ranges, cur)

	var hunks []string
	for _, r := range ranges {
		hunkEdits := edits[r.start : r.end+1]

		// Compute old/new start lines and lengths.
		oldStart, oldCount := 0, 0
		newStart, newCount := 0, 0
		firstOld, firstNew := true, true

		for _, e := range hunkEdits {
			switch e.kind {
			case editEq:
				if firstOld {
					oldStart = e.aLine + 1
					firstOld = false
				}
				if firstNew {
					newStart = e.bLine + 1
					firstNew = false
				}
				oldCount++
				newCount++
			case editDel:
				if firstOld {
					oldStart = e.aLine + 1
					firstOld = false
				}
				if firstNew && newStart == 0 {
					// new start aligns with where deletion happens
				}
				oldCount++
			case editAdd:
				if firstNew {
					newStart = e.bLine + 1
					firstNew = false
				}
				newCount++
			}
		}
		if firstOld {
			oldStart = aLen + 1
		}
		if firstNew {
			newStart = bLen + 1
		}

		var hb strings.Builder
		hb.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", oldStart, oldCount, newStart, newCount))
		for _, e := range hunkEdits {
			switch e.kind {
			case editEq:
				hb.WriteString(" ")
				hb.WriteString(e.text)
				hb.WriteString("\n")
			case editDel:
				hb.WriteString("-")
				hb.WriteString(e.text)
				hb.WriteString("\n")
			case editAdd:
				hb.WriteString("+")
				hb.WriteString(e.text)
				hb.WriteString("\n")
			}
		}
		hunks = append(hunks, hb.String())
	}
	return hunks
}

// max returns the larger of two ints.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// min returns the smaller of two ints.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ── file_line_endings ─────────────────────────────────────────────────────────

// LineEndingsInput holds the parameters for the file_line_endings tool.
type LineEndingsInput struct {
	Input     string // file path or raw text
	Mode      string // file | text; default file
	Operation string // normalize | detect | convert; default detect
	Target    string // lf | crlf; default lf
	Output    string // output file path for file mode
}

// LineEndingsDetectResult is the output for the detect operation.
type LineEndingsDetectResult struct {
	LineEnding string `json:"line_ending"` // lf | crlf | mixed
	LFCount    int    `json:"lf_count"`
	CRLFCount  int    `json:"crlf_count"`
}

// LineEndingsTextResult is the output for text mode operations.
type LineEndingsTextResult struct {
	Result string `json:"result"`
}

// LineEndingsFileResult is the output for file mode normalize/convert operations.
type LineEndingsFileResult struct {
	Output         string `json:"output"`
	LinesConverted int    `json:"lines_converted"`
}

// LineEndings detects or converts line endings in a file or text string.
func LineEndings(_ context.Context, in LineEndingsInput) string {
	mode := in.Mode
	if mode == "" {
		mode = "file"
	}
	op := in.Operation
	if op == "" {
		op = "detect"
	}
	target := in.Target
	if target == "" {
		target = "lf"
	}

	if target != "lf" && target != "crlf" {
		return errResult("unknown target: must be lf or crlf")
	}

	var content string
	switch mode {
	case "file":
		if in.Input == "" {
			return errResult("input file path is required")
		}
		b, err := os.ReadFile(in.Input)
		if err != nil {
			return errResult("cannot read file: " + err.Error())
		}
		content = string(b)
	case "text":
		content = in.Input
	default:
		return errResult("unknown mode: must be file or text")
	}

	switch op {
	case "detect":
		return detectLineEndings(content)

	case "normalize", "convert":
		converted, count := convertLineEndings(content, target)
		if mode == "text" {
			return resultJSON(LineEndingsTextResult{Result: converted})
		}
		// File mode: write to output path.
		outPath := in.Output
		if outPath == "" {
			outPath = in.Input // overwrite in place
		}
		if err := os.WriteFile(outPath, []byte(converted), 0o644); err != nil {
			return errResult("cannot write output file: " + err.Error())
		}
		return resultJSON(LineEndingsFileResult{Output: outPath, LinesConverted: count})

	default:
		return errResult("unknown operation: must be normalize, detect, or convert")
	}
}

// detectLineEndings counts LF and CRLF occurrences in content.
func detectLineEndings(content string) string {
	crlfCount := strings.Count(content, "\r\n")
	// LF-only count: total \n minus those that are part of \r\n sequences.
	totalLF := strings.Count(content, "\n")
	lfCount := totalLF - crlfCount

	var ending string
	switch {
	case crlfCount > 0 && lfCount > 0:
		ending = "mixed"
	case crlfCount > 0:
		ending = "crlf"
	default:
		ending = "lf"
	}

	return resultJSON(LineEndingsDetectResult{
		LineEnding: ending,
		LFCount:    lfCount,
		CRLFCount:  crlfCount,
	})
}

// convertLineEndings normalises all line endings in content to the target
// (lf or crlf) and returns the converted text plus the number of converted lines.
func convertLineEndings(content, target string) (string, int) {
	// First normalise all endings to LF.
	normalised := strings.ReplaceAll(content, "\r\n", "\n")
	normalised = strings.ReplaceAll(normalised, "\r", "\n")

	lfCount := strings.Count(normalised, "\n")

	if target == "crlf" {
		result := strings.ReplaceAll(normalised, "\n", "\r\n")
		return result, lfCount
	}
	// target == "lf"
	converted := strings.Count(content, "\r\n") + strings.Count(content, "\r")
	_ = converted // count is the number of lines we changed
	return normalised, strings.Count(content, "\r\n") + strings.Count(content, "\r")
}

// ── file_hex_view ─────────────────────────────────────────────────────────────

// HexViewInput holds the parameters for the file_hex_view tool.
type HexViewInput struct {
	Input  string // file path or base64-encoded bytes
	Mode   string // file | base64; default file
	Offset int    // byte offset to start from; default 0
	Length int    // number of bytes to show; default 256
	Width  int    // bytes per row; default 16
}

// HexViewResult is the output of file_hex_view.
type HexViewResult struct {
	HexView    string `json:"hex_view"`
	Offset     int    `json:"offset"`
	BytesShown int    `json:"bytes_shown"`
	TotalBytes int    `json:"total_bytes"`
}

// HexView returns a formatted hex+ASCII dump of a file or raw bytes.
func HexView(_ context.Context, in HexViewInput) string {
	mode := in.Mode
	if mode == "" {
		mode = "file"
	}
	length := in.Length
	if length <= 0 {
		length = 256
	}
	width := in.Width
	if width <= 0 {
		width = 16
	}

	var data []byte

	switch mode {
	case "file":
		if in.Input == "" {
			return errResult("input file path is required")
		}
		f, err := os.Open(in.Input)
		if err != nil {
			return errResult("cannot open file: " + err.Error())
		}
		defer f.Close()

		stat, err := f.Stat()
		if err != nil {
			return errResult("cannot stat file: " + err.Error())
		}
		total := int(stat.Size())

		offset := in.Offset
		if offset < 0 {
			offset = 0
		}
		if offset >= total && total > 0 {
			return errResult(fmt.Sprintf("offset %d exceeds file size %d", offset, total))
		}

		if _, err := f.Seek(int64(offset), io.SeekStart); err != nil {
			return errResult("seek failed: " + err.Error())
		}
		toRead := length
		if offset+toRead > total {
			toRead = total - offset
		}
		data = make([]byte, toRead)
		n, err := io.ReadFull(f, data)
		if err != nil && err != io.ErrUnexpectedEOF {
			return errResult("read error: " + err.Error())
		}
		data = data[:n]

		return resultJSON(HexViewResult{
			HexView:    formatHexDump(data, in.Offset, width),
			Offset:     in.Offset,
			BytesShown: len(data),
			TotalBytes: total,
		})

	case "base64":
		if in.Input == "" {
			return errResult("input base64 data is required")
		}
		decoded, err := base64.StdEncoding.DecodeString(in.Input)
		if err != nil {
			// Try URL-safe encoding as fallback.
			decoded, err = base64.URLEncoding.DecodeString(in.Input)
			if err != nil {
				return errResult("base64 decode failed: " + err.Error())
			}
		}
		total := len(decoded)
		offset := in.Offset
		if offset < 0 {
			offset = 0
		}
		if offset > total {
			offset = total
		}
		end := offset + length
		if end > total {
			end = total
		}
		data = decoded[offset:end]

		return resultJSON(HexViewResult{
			HexView:    formatHexDump(data, offset, width),
			Offset:     offset,
			BytesShown: len(data),
			TotalBytes: total,
		})

	default:
		return errResult("unknown mode: must be file or base64")
	}
}

// formatHexDump formats data as a standard hex+ASCII dump table.
// startOffset is the absolute byte offset of data[0] in the source.
func formatHexDump(data []byte, startOffset, width int) string {
	if len(data) == 0 {
		return ""
	}

	var sb strings.Builder
	halfWidth := width / 2

	for i := 0; i < len(data); i += width {
		end := i + width
		if end > len(data) {
			end = len(data)
		}
		row := data[i:end]

		// Address column.
		sb.WriteString(fmt.Sprintf("%08x  ", startOffset+i))

		// Hex columns split at halfWidth.
		for j, b := range row {
			if j == halfWidth {
				sb.WriteString(" ")
			}
			sb.WriteString(fmt.Sprintf("%02x ", b))
		}

		// Pad the last row if shorter than width.
		if len(row) < width {
			missing := width - len(row)
			for k := 0; k < missing; k++ {
				if (len(row) + k) == halfWidth {
					sb.WriteString(" ")
				}
				sb.WriteString("   ")
			}
		}

		// ASCII column.
		sb.WriteString(" |")
		for _, b := range row {
			if b >= 0x20 && b < 0x7f {
				sb.WriteByte(b)
			} else {
				sb.WriteByte('.')
			}
		}
		sb.WriteString("|\n")
	}

	return sb.String()
}

// readLines is a helper that reads all lines from a Reader using a bufio.Scanner.
// It is used internally where streaming line-by-line reading is needed.
func readLines(r io.Reader) []string {
	var lines []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

// bytesReader wraps a byte slice as an io.Reader for use with readLines.
func bytesReader(b []byte) io.Reader {
	return bytes.NewReader(b)
}
