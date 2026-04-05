package tools

import (
	"context"
	"fmt"

	"dev-forge-mcp/internal/dpf"
)

// OptimizeInput represents a single image optimization request.
type OptimizeInput struct {
	Path      string   `json:"path"`
	MaxWidth  int      `json:"max_width,omitempty"`
	MaxHeight int      `json:"max_height,omitempty"`
	Formats   []string `json:"formats,omitempty"`
	Quality   int      `json:"quality,omitempty"`
}

// OptimizeImagesInput is the input schema for the optimize_images tool.
type OptimizeImagesInput struct {
	Inputs      []OptimizeInput `json:"inputs"`
	Parallelism int             `json:"parallelism,omitempty"`
}

// OptimizeOutput represents the output for a single image.
type OptimizeOutput struct {
	Format       string `json:"format"`
	Path         string `json:"path"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	ApproxSizeKB int    `json:"approx_size_kb"`
}

// OptimizeResult is the per-source-image result.
type OptimizeResult struct {
	SourcePath string           `json:"source_path"`
	Outputs    []OptimizeOutput `json:"outputs"`
}

// OptimizeImagesOutput is the output schema for the optimize_images tool.
type OptimizeImagesOutput struct {
	Results []OptimizeResult `json:"results"`
}

// OptimizeImages implements the optimize_images MCP tool.
func (s *Server) OptimizeImages(ctx context.Context, input OptimizeImagesInput) string {
	if len(input.Inputs) == 0 {
		return errorJSON("inputs is required and must not be empty")
	}
	if s.DPF == nil {
		return errorJSON("dpf binary not available. Ensure bin/dpf is installed and executable.")
	}

	var results []OptimizeResult

	for _, item := range input.Inputs {
		if item.Path == "" {
			continue
		}

		quality := item.Quality
		if quality <= 0 {
			quality = 85
		}

		formats := item.Formats
		if len(formats) == 0 {
			formats = []string{"webp"}
		}

		job := &dpf.OptimizeJob{
			Operation: "optimize",
			Inputs:    []string{item.Path},
			AlsoWebp:  containsFormat(formats, "webp"),
		}

		if quality > 0 {
			q := uint8(quality)
			job.Quality = &q
		}

		jobResult, err := s.DPF.Execute(job)
		if callErr := dpfCallError(jobResult, err); callErr != nil {
			results = append(results, OptimizeResult{
				SourcePath: item.Path,
				Outputs:    []OptimizeOutput{{Format: "error", Path: fmt.Sprintf("error: %s", callErr.Error())}},
			})
			continue
		}

		var outputs []OptimizeOutput
		for _, out := range jobResult.Outputs {
			outputs = append(outputs, OptimizeOutput{
				Format:       out.Format,
				Path:         out.Path,
				Width:        int(out.Width),
				Height:       int(out.Height),
				ApproxSizeKB: int(out.SizeBytes / 1024),
			})
		}

		if outputs == nil {
			outputs = []OptimizeOutput{}
		}

		results = append(results, OptimizeResult{
			SourcePath: item.Path,
			Outputs:    outputs,
		})
	}

	if results == nil {
		results = []OptimizeResult{}
	}

	return mustJSON(OptimizeImagesOutput{Results: results})
}

func containsFormat(formats []string, target string) bool {
	for _, f := range formats {
		if f == target {
			return true
		}
	}
	return false
}
