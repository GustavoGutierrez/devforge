// Package ui implements frontend UI-focused micro-utilities.
package ui

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func errJSON(msg string) string {
	b, _ := json.Marshal(map[string]string{"error": msg})
	return string(b)
}

func resultJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return errJSON("marshal failed: " + err.Error())
	}
	return string(b)
}

// SVGOptimizeInput defines SVG optimization input.
type SVGOptimizeInput struct {
	SVG string `json:"svg"`
}

// SVGOptimizeOutput is the SVG optimization response.
type SVGOptimizeOutput struct {
	OptimizedSVG   string  `json:"optimized_svg"`
	BytesBefore    int     `json:"bytes_before"`
	BytesAfter     int     `json:"bytes_after"`
	ReductionBytes int     `json:"reduction_bytes"`
	ReductionPct   float64 `json:"reduction_pct"`
}

// SVGOptimize removes common SVG noise and minifies markup.
func SVGOptimize(_ context.Context, in SVGOptimizeInput) string {
	svg := strings.TrimSpace(in.SVG)
	if svg == "" {
		return errJSON("svg is required")
	}
	if !strings.Contains(strings.ToLower(svg), "<svg") {
		return errJSON("input must contain an <svg> root element")
	}

	before := len(svg)
	optimized := optimizeSVG(svg)
	after := len(optimized)

	reduction := before - after
	reductionPct := 0.0
	if before > 0 {
		reductionPct = float64(reduction) * 100.0 / float64(before)
	}

	return resultJSON(SVGOptimizeOutput{
		OptimizedSVG:   optimized,
		BytesBefore:    before,
		BytesAfter:     after,
		ReductionBytes: reduction,
		ReductionPct:   reductionPct,
	})
}

var (
	reComment     = regexp.MustCompile(`(?s)<!--.*?-->`)
	reXMLDecl     = regexp.MustCompile(`(?is)<\?xml[^>]*\?>`)
	reDoctype     = regexp.MustCompile(`(?is)<!doctype[^>]*>`)
	reBetweenTags = regexp.MustCompile(`>\s+<`)
	reMultiSpace  = regexp.MustCompile(`[\t\n\r]+`)
	reSpaceAttrs  = regexp.MustCompile(`\s{2,}`)
)

var metadataTagPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?is)<metadata\b[^>]*>.*?</metadata>`),
	regexp.MustCompile(`(?is)<desc\b[^>]*>.*?</desc>`),
	regexp.MustCompile(`(?is)<title\b[^>]*>.*?</title>`),
}

var emptyTagPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?is)<g\b[^>]*>\s*</g>`),
	regexp.MustCompile(`(?is)<defs\b[^>]*>\s*</defs>`),
	regexp.MustCompile(`(?is)<symbol\b[^>]*>\s*</symbol>`),
	regexp.MustCompile(`(?is)<clipPath\b[^>]*>\s*</clipPath>`),
	regexp.MustCompile(`(?is)<mask\b[^>]*>\s*</mask>`),
	regexp.MustCompile(`(?is)<metadata\b[^>]*>\s*</metadata>`),
	regexp.MustCompile(`(?is)<desc\b[^>]*>\s*</desc>`),
	regexp.MustCompile(`(?is)<title\b[^>]*>\s*</title>`),
}

func optimizeSVG(svg string) string {
	out := svg
	out = reXMLDecl.ReplaceAllString(out, "")
	out = reDoctype.ReplaceAllString(out, "")
	out = reComment.ReplaceAllString(out, "")
	for _, re := range metadataTagPatterns {
		out = re.ReplaceAllString(out, "")
	}

	// Remove repeatedly nested empty containers.
	for {
		next := out
		for _, re := range emptyTagPatterns {
			next = re.ReplaceAllString(next, "")
		}
		if next == out {
			break
		}
		out = next
	}

	out = reBetweenTags.ReplaceAllString(out, "><")
	out = reMultiSpace.ReplaceAllString(out, " ")
	out = reSpaceAttrs.ReplaceAllString(out, " ")
	out = strings.TrimSpace(out)
	return out
}

// ImageBase64Input defines image-to-base64 input parameters.
type ImageBase64Input struct {
	Path     string `json:"path"`
	DataURI  bool   `json:"data_uri"`
	MimeType string `json:"mime_type"`
}

// ImageBase64Output is the image base64 encoding response.
type ImageBase64Output struct {
	Path     string `json:"path"`
	MimeType string `json:"mime_type"`
	Bytes    int    `json:"bytes"`
	Base64   string `json:"base64"`
	DataURI  string `json:"data_uri,omitempty"`
}

// ImageBase64 encodes a local image file as Base64 and optional Data URI.
func ImageBase64(_ context.Context, in ImageBase64Input) string {
	path := strings.TrimSpace(in.Path)
	if path == "" {
		return errJSON("path is required")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return errJSON("failed to read file: " + err.Error())
	}
	if len(data) == 0 {
		return errJSON("file is empty")
	}

	mime := strings.TrimSpace(in.MimeType)
	if mime == "" {
		mime = detectMime(path, data)
	}

	b64 := base64.StdEncoding.EncodeToString(data)
	out := ImageBase64Output{
		Path:     path,
		MimeType: mime,
		Bytes:    len(data),
		Base64:   b64,
	}

	if in.DataURI {
		out.DataURI = fmt.Sprintf("data:%s;base64,%s", mime, b64)
	}

	return resultJSON(out)
}

func detectMime(path string, data []byte) string {
	ext := strings.ToLower(filepath.Ext(path))
	byExt := map[string]string{
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".webp": "image/webp",
		".svg":  "image/svg+xml",
		".avif": "image/avif",
	}
	if m, ok := byExt[ext]; ok {
		return m
	}
	return http.DetectContentType(data)
}
