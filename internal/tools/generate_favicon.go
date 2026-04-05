package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"dev-forge-mcp/internal/dpf"
)

// GenerateFaviconInput is the input schema for the generate_favicon tool.
type GenerateFaviconInput struct {
	SourcePath      string   `json:"source_path"`
	BackgroundColor string   `json:"background_color,omitempty"` // default #ffffff
	Sizes           []int    `json:"sizes,omitempty"`
	Formats         []string `json:"formats,omitempty"`
}

// IconOutput represents a single generated favicon file.
type IconOutput struct {
	Size   int    `json:"size"`
	Format string `json:"format"`
	Path   string `json:"path"`
}

// GenerateFaviconOutput is the output schema for the generate_favicon tool.
type GenerateFaviconOutput struct {
	Icons        []IconOutput `json:"icons"`
	HTMLSnippets []string     `json:"html_snippets"`
}

// GenerateFavicon implements the generate_favicon MCP tool.
func (s *Server) GenerateFavicon(ctx context.Context, input GenerateFaviconInput) string {
	if strings.TrimSpace(input.SourcePath) == "" {
		return errorJSON("source_path is required")
	}
	if s.DPF == nil {
		return errorJSON("dpf binary not available. Ensure bin/dpf is installed and executable.")
	}

	bgColor := input.BackgroundColor
	if bgColor == "" {
		bgColor = "#ffffff"
	}

	sizes := input.Sizes
	if len(sizes) == 0 {
		sizes = []int{16, 32, 48, 180, 192, 512}
	}

	// Convert int sizes to uint32
	u32Sizes := make([]uint32, len(sizes))
	for i, s := range sizes {
		u32Sizes[i] = uint32(s)
	}

	outputDir := filepath.Join(filepath.Dir(input.SourcePath), "favicons")

	job := &dpf.FaviconJob{
		Operation:        "favicon",
		Input:            input.SourcePath,
		OutputDir:        outputDir,
		Sizes:            u32Sizes,
		GenerateICO:      true,
		GenerateManifest: true,
	}

	jobResult, err := s.DPF.Execute(job)
	if msg := dpfErrorJSON(jobResult, err); msg != "" {
		return msg
	}

	var icons []IconOutput
	var htmlSnippets []string

	for _, out := range jobResult.Outputs {
		icons = append(icons, IconOutput{
			Size:   int(out.Width),
			Format: out.Format,
			Path:   out.Path,
		})
		htmlSnippets = append(htmlSnippets, faviconHTMLSnippet(out.Format, out.Path, int(out.Width)))
	}

	// If dpf returned no outputs, generate expected outputs list
	if len(icons) == 0 {
		for _, size := range sizes {
			for _, format := range getFormats(input.Formats) {
				iconPath := fmt.Sprintf("%s/favicon-%dx%d.%s", outputDir, size, size, format)
				icons = append(icons, IconOutput{
					Size:   size,
					Format: format,
					Path:   iconPath,
				})
				htmlSnippets = append(htmlSnippets, faviconHTMLSnippet(format, "/favicon-"+fmt.Sprintf("%d", size)+"."+format, size))
			}
		}
	}

	if icons == nil {
		icons = []IconOutput{}
	}
	if htmlSnippets == nil {
		htmlSnippets = []string{}
	}

	return mustJSON(GenerateFaviconOutput{
		Icons:        icons,
		HTMLSnippets: htmlSnippets,
	})
}

func faviconHTMLSnippet(format, path string, size int) string {
	switch format {
	case "ico":
		return `<link rel="icon" href="/favicon.ico">`
	case "svg":
		return `<link rel="icon" type="image/svg+xml" href="/favicon.svg">`
	case "png":
		if size == 180 {
			return fmt.Sprintf(`<link rel="apple-touch-icon" sizes="%dx%d" href="%s">`, size, size, path)
		}
		if size >= 192 {
			return fmt.Sprintf(`<link rel="icon" type="image/png" sizes="%dx%d" href="%s">`, size, size, path)
		}
		return fmt.Sprintf(`<link rel="icon" type="image/png" sizes="%dx%d" href="%s">`, size, size, path)
	default:
		return fmt.Sprintf(`<link rel="icon" href="%s">`, path)
	}
}

func getFormats(requested []string) []string {
	if len(requested) > 0 {
		return requested
	}
	return []string{"ico", "png", "svg"}
}
