package tools

import (
	"context"

	"dev-forge-mcp/internal/dpf"
)

// ─── Video Tool Inputs ──────────────────────────────────────────────────────────

// VideoTranscodeInput is the input schema for the video_transcode tool.
type VideoTranscodeInput struct {
	Input   string `json:"input"`
	Output  string `json:"output"`
	Codec   string `json:"codec"`
	Bitrate string `json:"bitrate,omitempty"`
	Preset  string `json:"preset,omitempty"`
}

// VideoTranscodeOutput is the output schema for the video_transcode tool.
type VideoTranscodeOutput struct {
	Success    bool   `json:"success"`
	OutputPath string `json:"output_path,omitempty"`
	ElapsedMs  uint64 `json:"elapsed_ms,omitempty"`
	Error      string `json:"error,omitempty"`
}

// VideoResizeInput is the input schema for the video_resize tool.
type VideoResizeInput struct {
	Input          string `json:"input"`
	Output         string `json:"output"`
	Width          uint32 `json:"width,omitempty"`
	Height         uint32 `json:"height,omitempty"`
	MaintainAspect bool   `json:"maintain_aspect,omitempty"`
}

// VideoResizeOutput is the output schema for the video_resize tool.
type VideoResizeOutput struct {
	Success    bool   `json:"success"`
	OutputPath string `json:"output_path,omitempty"`
	ElapsedMs  uint64 `json:"elapsed_ms,omitempty"`
	Error      string `json:"error,omitempty"`
}

// VideoTrimInput is the input schema for the video_trim tool.
type VideoTrimInput struct {
	Input  string  `json:"input"`
	Output string  `json:"output"`
	Start  float64 `json:"start"`
	End    float64 `json:"end"`
}

// VideoTrimOutput is the output schema for the video_trim tool.
type VideoTrimOutput struct {
	Success    bool   `json:"success"`
	OutputPath string `json:"output_path,omitempty"`
	ElapsedMs  uint64 `json:"elapsed_ms,omitempty"`
	Error      string `json:"error,omitempty"`
}

// VideoThumbnailInput is the input schema for the video_thumbnail tool.
type VideoThumbnailInput struct {
	Input     string `json:"input"`
	Output    string `json:"output"`
	Timestamp string `json:"timestamp"`
	Format    string `json:"format,omitempty"`
	Quality   int    `json:"quality,omitempty"`
}

// VideoThumbnailOutput is the output schema for the video_thumbnail tool.
type VideoThumbnailOutput struct {
	Success    bool   `json:"success"`
	OutputPath string `json:"output_path,omitempty"`
	ElapsedMs  uint64 `json:"elapsed_ms,omitempty"`
	Error      string `json:"error,omitempty"`
}

// VideoProfileInput is the input schema for the video_profile tool.
type VideoProfileInput struct {
	Input   string `json:"input"`
	Output  string `json:"output"`
	Profile string `json:"profile"`
}

// VideoProfileOutput is the output schema for the video_profile tool.
type VideoProfileOutput struct {
	Success    bool   `json:"success"`
	OutputPath string `json:"output_path,omitempty"`
	ElapsedMs  uint64 `json:"elapsed_ms,omitempty"`
	Error      string `json:"error,omitempty"`
}

// ─── Video Tool Handlers ────────────────────────────────────────────────────────

// VideoTranscode implements the video_transcode MCP tool.
func (s *Server) VideoTranscode(ctx context.Context, input VideoTranscodeInput) string {
	if input.Input == "" {
		return errorJSON("input is required")
	}
	if input.Output == "" {
		return errorJSON("output is required")
	}
	if input.Codec == "" {
		return errorJSON("codec is required (h264, h265, vp8, vp9, av1)")
	}
	if s.DPF == nil {
		return errorJSON("dpf binary not available. Ensure bin/dpf is installed and executable.")
	}

	job := &dpf.VideoTranscodeJob{
		Operation: "video_transcode",
		Input:     input.Input,
		Output:    input.Output,
		Codec:     input.Codec,
		Bitrate:   input.Bitrate,
		Preset:    input.Preset,
	}

	result, err := s.DPF.VideoTranscode(job)
	if msg := dpfErrorJSON(result, err); msg != "" {
		return msg
	}

	return mustJSON(VideoTranscodeOutput{
		Success:    result.Success,
		OutputPath: getFirstOutputPath(result),
		ElapsedMs:  result.ElapsedMs,
	})
}

// VideoResize implements the video_resize MCP tool.
func (s *Server) VideoResize(ctx context.Context, input VideoResizeInput) string {
	if input.Input == "" {
		return errorJSON("input is required")
	}
	if input.Output == "" {
		return errorJSON("output is required")
	}
	if input.Width == 0 && input.Height == 0 {
		return errorJSON("at least one of width or height is required")
	}
	if s.DPF == nil {
		return errorJSON("dpf binary not available. Ensure bin/dpf is installed and executable.")
	}

	job := &dpf.VideoResizeJob{
		Operation:      "video_resize",
		Input:          input.Input,
		Output:         input.Output,
		Width:          input.Width,
		Height:         input.Height,
		MaintainAspect: input.MaintainAspect,
	}

	result, err := s.DPF.VideoResize(job)
	if msg := dpfErrorJSON(result, err); msg != "" {
		return msg
	}

	return mustJSON(VideoResizeOutput{
		Success:    result.Success,
		OutputPath: getFirstOutputPath(result),
		ElapsedMs:  result.ElapsedMs,
	})
}

// VideoTrim implements the video_trim MCP tool.
func (s *Server) VideoTrim(ctx context.Context, input VideoTrimInput) string {
	if input.Input == "" {
		return errorJSON("input is required")
	}
	if input.Output == "" {
		return errorJSON("output is required")
	}
	if input.Start < 0 {
		return errorJSON("start must be a non-negative number")
	}
	if input.End <= input.Start {
		return errorJSON("end must be greater than start")
	}
	if s.DPF == nil {
		return errorJSON("dpf binary not available. Ensure bin/dpf is installed and executable.")
	}

	job := &dpf.VideoTrimJob{
		Operation: "video_trim",
		Input:     input.Input,
		Output:    input.Output,
		Start:     input.Start,
		End:       input.End,
	}

	result, err := s.DPF.VideoTrim(job)
	if msg := dpfErrorJSON(result, err); msg != "" {
		return msg
	}

	return mustJSON(VideoTrimOutput{
		Success:    result.Success,
		OutputPath: getFirstOutputPath(result),
		ElapsedMs:  result.ElapsedMs,
	})
}

// VideoThumbnail implements the video_thumbnail MCP tool.
func (s *Server) VideoThumbnail(ctx context.Context, input VideoThumbnailInput) string {
	if input.Input == "" {
		return errorJSON("input is required")
	}
	if input.Output == "" {
		return errorJSON("output is required")
	}
	if input.Timestamp == "" {
		return errorJSON("timestamp is required (e.g., '25%' or '30.5')")
	}
	if s.DPF == nil {
		return errorJSON("dpf binary not available. Ensure bin/dpf is installed and executable.")
	}

	job := &dpf.VideoThumbnailJob{
		Operation: "video_thumbnail",
		Input:     input.Input,
		Output:    input.Output,
		Timestamp: input.Timestamp,
		Format:    input.Format,
	}

	if input.Quality > 0 {
		q := uint8(input.Quality)
		job.Quality = &q
	}

	result, err := s.DPF.VideoThumbnail(job)
	if msg := dpfErrorJSON(result, err); msg != "" {
		return msg
	}

	return mustJSON(VideoThumbnailOutput{
		Success:    result.Success,
		OutputPath: getFirstOutputPath(result),
		ElapsedMs:  result.ElapsedMs,
	})
}

// VideoProfile implements the video_profile MCP tool.
func (s *Server) VideoProfile(ctx context.Context, input VideoProfileInput) string {
	if input.Input == "" {
		return errorJSON("input is required")
	}
	if input.Output == "" {
		return errorJSON("output is required")
	}
	if input.Profile == "" {
		return errorJSON("profile is required (web-low, web-mid, web-high)")
	}
	if s.DPF == nil {
		return errorJSON("dpf binary not available. Ensure bin/dpf is installed and executable.")
	}

	job := &dpf.VideoProfileJob{
		Operation: "video_profile",
		Input:     input.Input,
		Output:    input.Output,
		Profile:   input.Profile,
	}

	result, err := s.DPF.VideoProfile(job)
	if msg := dpfErrorJSON(result, err); msg != "" {
		return msg
	}

	return mustJSON(VideoProfileOutput{
		Success:    result.Success,
		OutputPath: getFirstOutputPath(result),
		ElapsedMs:  result.ElapsedMs,
	})
}
