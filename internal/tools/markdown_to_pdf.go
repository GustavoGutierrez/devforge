package tools

import (
	"context"
	"encoding/json"
	"strings"

	"dev-forge-mcp/internal/dpf"
)

// MarkdownToPDFInput is the input schema for the markdown_to_pdf tool.
type MarkdownToPDFInput struct {
	Input          string                 `json:"input,omitempty"`
	MarkdownText   string                 `json:"markdown_text,omitempty"`
	MarkdownBase64 string                 `json:"markdown_base64,omitempty"`
	Output         string                 `json:"output,omitempty"`
	OutputDir      string                 `json:"output_dir,omitempty"`
	FileName       string                 `json:"file_name,omitempty"`
	Inline         bool                   `json:"inline,omitempty"`
	PageSize       string                 `json:"page_size,omitempty"`
	PageWidthMM    *float64               `json:"page_width_mm,omitempty"`
	PageHeightMM   *float64               `json:"page_height_mm,omitempty"`
	LayoutMode     string                 `json:"layout_mode,omitempty"`
	Theme          string                 `json:"theme,omitempty"`
	ThemeConfig    map[string]interface{} `json:"theme_config,omitempty"`
	ResourceFiles  map[string]string      `json:"resource_files,omitempty"`
}

// MarkdownToPDFOutputFile represents one produced PDF artifact.
type MarkdownToPDFOutputFile struct {
	Path       string `json:"path,omitempty"`
	Format     string `json:"format,omitempty"`
	SizeBytes  uint64 `json:"size_bytes,omitempty"`
	DataBase64 string `json:"data_base64,omitempty"`
}

// MarkdownToPDFOutput is the output schema for the markdown_to_pdf tool.
type MarkdownToPDFOutput struct {
	Success   bool                      `json:"success"`
	Operation string                    `json:"operation,omitempty"`
	Outputs   []MarkdownToPDFOutputFile `json:"outputs,omitempty"`
	ElapsedMs uint64                    `json:"elapsed_ms,omitempty"`
	Metadata  map[string]interface{}    `json:"metadata,omitempty"`
}

// MarkdownToPDF implements the markdown_to_pdf MCP tool.
func (s *Server) MarkdownToPDF(ctx context.Context, input MarkdownToPDFInput) string {
	input.Input = strings.TrimSpace(input.Input)
	input.MarkdownText = strings.TrimSpace(input.MarkdownText)
	input.MarkdownBase64 = strings.TrimSpace(input.MarkdownBase64)
	input.Output = strings.TrimSpace(input.Output)
	input.OutputDir = strings.TrimSpace(input.OutputDir)
	input.FileName = strings.TrimSpace(input.FileName)
	input.PageSize = strings.TrimSpace(input.PageSize)
	input.LayoutMode = strings.TrimSpace(input.LayoutMode)
	input.Theme = strings.TrimSpace(input.Theme)

	providedSources := 0
	if input.Input != "" {
		providedSources++
	}
	if input.MarkdownText != "" {
		providedSources++
	}
	if input.MarkdownBase64 != "" {
		providedSources++
	}
	if providedSources != 1 {
		return errorJSON("exactly one input source is required: input, markdown_text, or markdown_base64")
	}

	if input.Output == "" && input.OutputDir == "" && !input.Inline {
		return errorJSON("at least one output mode is required: output, output_dir, or inline=true")
	}

	if input.OutputDir != "" && input.Input == "" && input.FileName == "" {
		return errorJSON("file_name is required when using output_dir with inline markdown input")
	}

	if (input.PageWidthMM == nil) != (input.PageHeightMM == nil) {
		return errorJSON("custom page size requires both page_width_mm and page_height_mm")
	}
	if input.PageWidthMM != nil && (*input.PageWidthMM <= 0 || *input.PageHeightMM <= 0) {
		return errorJSON("custom page size dimensions must be positive")
	}

	if s.DPF == nil {
		return errorJSON("dpf binary not available. Ensure bin/dpf is installed and executable.")
	}

	job := &dpf.MarkdownToPDFJob{
		Input:         input.Input,
		Output:        input.Output,
		OutputDir:     input.OutputDir,
		FileName:      input.FileName,
		Inline:        input.Inline,
		PageWidthMM:   input.PageWidthMM,
		PageHeightMM:  input.PageHeightMM,
		ResourceFiles: input.ResourceFiles,
	}
	if input.MarkdownText != "" {
		job.MarkdownText = &input.MarkdownText
	}
	if input.MarkdownBase64 != "" {
		job.MarkdownBase64 = &input.MarkdownBase64
	}
	if input.PageSize != "" {
		job.PageSize = &input.PageSize
	}
	if input.LayoutMode != "" {
		job.LayoutMode = &input.LayoutMode
	}
	if input.Theme != "" {
		job.Theme = &input.Theme
	}
	if len(input.ThemeConfig) > 0 {
		data, err := json.Marshal(input.ThemeConfig)
		if err != nil {
			return errorJSON("invalid theme_config: " + err.Error())
		}
		job.ThemeConfig = data
	}

	result, err := s.DPF.MarkdownToPDF(job)
	if msg := dpfErrorJSON(result, err); msg != "" {
		return msg
	}

	var outputs []MarkdownToPDFOutputFile
	for _, out := range result.Outputs {
		item := MarkdownToPDFOutputFile{
			Path:      out.Path,
			Format:    out.Format,
			SizeBytes: out.SizeBytes,
		}
		if out.DataBase64 != nil {
			item.DataBase64 = *out.DataBase64
		}
		outputs = append(outputs, item)
	}
	if outputs == nil {
		outputs = []MarkdownToPDFOutputFile{}
	}

	var metadata map[string]interface{}
	if result.Metadata != nil {
		_ = json.Unmarshal(*result.Metadata, &metadata)
	}

	return mustJSON(MarkdownToPDFOutput{
		Success:   result.Success,
		Operation: result.Operation,
		Outputs:   outputs,
		ElapsedMs: result.ElapsedMs,
		Metadata:  metadata,
	})
}
