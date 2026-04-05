package tools

import (
	"context"

	"dev-forge-mcp/internal/dpf"
)

// ─── Audio Tool Inputs ─────────────────────────────────────────────────────────

// AudioTranscodeInput is the input schema for the audio_transcode tool.
type AudioTranscodeInput struct {
	Input      string `json:"input"`
	Output     string `json:"output"`
	Codec      string `json:"codec"`
	Bitrate    string `json:"bitrate,omitempty"`
	SampleRate uint32 `json:"sample_rate,omitempty"`
	Channels   uint32 `json:"channels,omitempty"`
}

// AudioTranscodeOutput is the output schema for the audio_transcode tool.
type AudioTranscodeOutput struct {
	Success    bool   `json:"success"`
	OutputPath string `json:"output_path,omitempty"`
	ElapsedMs  uint64 `json:"elapsed_ms,omitempty"`
	Error      string `json:"error,omitempty"`
}

// AudioTrimInput is the input schema for the audio_trim tool.
type AudioTrimInput struct {
	Input  string  `json:"input"`
	Output string  `json:"output"`
	Start  float64 `json:"start"`
	End    float64 `json:"end"`
}

// AudioTrimOutput is the output schema for the audio_trim tool.
type AudioTrimOutput struct {
	Success    bool   `json:"success"`
	OutputPath string `json:"output_path,omitempty"`
	ElapsedMs  uint64 `json:"elapsed_ms,omitempty"`
	Error      string `json:"error,omitempty"`
}

// AudioNormalizeInput is the input schema for the audio_normalize tool.
type AudioNormalizeInput struct {
	Input      string  `json:"input"`
	Output     string  `json:"output"`
	TargetLUFS float64 `json:"target_lufs"`
}

// AudioNormalizeOutput is the output schema for the audio_normalize tool.
type AudioNormalizeOutput struct {
	Success    bool   `json:"success"`
	OutputPath string `json:"output_path,omitempty"`
	ElapsedMs  uint64 `json:"elapsed_ms,omitempty"`
	Error      string `json:"error,omitempty"`
}

// AudioSilenceTrimInput is the input schema for the audio_silence_trim tool.
type AudioSilenceTrimInput struct {
	Input       string  `json:"input"`
	Output      string  `json:"output"`
	ThresholdDB float64 `json:"threshold_db,omitempty"`
	MinDuration float64 `json:"min_duration,omitempty"`
}

// AudioSilenceTrimOutput is the output schema for the audio_silence_trim tool.
type AudioSilenceTrimOutput struct {
	Success    bool   `json:"success"`
	OutputPath string `json:"output_path,omitempty"`
	ElapsedMs  uint64 `json:"elapsed_ms,omitempty"`
	Error      string `json:"error,omitempty"`
}

// ─── Audio Tool Handlers ───────────────────────────────────────────────────────

// AudioTranscode implements the audio_transcode MCP tool.
func (s *Server) AudioTranscode(ctx context.Context, input AudioTranscodeInput) string {
	if input.Input == "" {
		return errorJSON("input is required")
	}
	if input.Output == "" {
		return errorJSON("output is required")
	}
	if input.Codec == "" {
		return errorJSON("codec is required (mp3, aac, opus, vorbis, flac, wav)")
	}
	if s.DPF == nil {
		return errorJSON("dpf binary not available. Ensure bin/dpf is installed and executable.")
	}

	job := &dpf.AudioTranscodeJob{
		Operation: "audio_transcode",
		Input:     input.Input,
		Output:    input.Output,
		Codec:     input.Codec,
		Bitrate:   input.Bitrate,
	}

	if input.SampleRate > 0 {
		job.SampleRate = &input.SampleRate
	}
	if input.Channels > 0 {
		job.Channels = &input.Channels
	}

	result, err := s.DPF.AudioTranscode(job)
	if msg := dpfErrorJSON(result, err); msg != "" {
		return msg
	}

	return mustJSON(AudioTranscodeOutput{
		Success:    result.Success,
		OutputPath: getFirstOutputPath(result),
		ElapsedMs:  result.ElapsedMs,
	})
}

// AudioTrim implements the audio_trim MCP tool.
func (s *Server) AudioTrim(ctx context.Context, input AudioTrimInput) string {
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

	job := &dpf.AudioTrimJob{
		Operation: "audio_trim",
		Input:     input.Input,
		Output:    input.Output,
		Start:     input.Start,
		End:       input.End,
	}

	result, err := s.DPF.AudioTrim(job)
	if msg := dpfErrorJSON(result, err); msg != "" {
		return msg
	}

	return mustJSON(AudioTrimOutput{
		Success:    result.Success,
		OutputPath: getFirstOutputPath(result),
		ElapsedMs:  result.ElapsedMs,
	})
}

// AudioNormalize implements the audio_normalize MCP tool.
func (s *Server) AudioNormalize(ctx context.Context, input AudioNormalizeInput) string {
	if input.Input == "" {
		return errorJSON("input is required")
	}
	if input.Output == "" {
		return errorJSON("output is required")
	}
	if s.DPF == nil {
		return errorJSON("dpf binary not available. Ensure bin/dpf is installed and executable.")
	}

	job := &dpf.AudioNormalizeJob{
		Operation:  "audio_normalize",
		Input:      input.Input,
		Output:     input.Output,
		TargetLUFS: input.TargetLUFS,
	}

	result, err := s.DPF.AudioNormalize(job)
	if msg := dpfErrorJSON(result, err); msg != "" {
		return msg
	}

	return mustJSON(AudioNormalizeOutput{
		Success:    result.Success,
		OutputPath: getFirstOutputPath(result),
		ElapsedMs:  result.ElapsedMs,
	})
}

// AudioSilenceTrim implements the audio_silence_trim MCP tool.
func (s *Server) AudioSilenceTrim(ctx context.Context, input AudioSilenceTrimInput) string {
	if input.Input == "" {
		return errorJSON("input is required")
	}
	if input.Output == "" {
		return errorJSON("output is required")
	}
	if s.DPF == nil {
		return errorJSON("dpf binary not available. Ensure bin/dpf is installed and executable.")
	}

	job := &dpf.AudioSilenceTrimJob{
		Operation: "audio_silence_trim",
		Input:     input.Input,
		Output:    input.Output,
	}

	if input.ThresholdDB != 0 {
		job.ThresholdDB = &input.ThresholdDB
	}
	if input.MinDuration != 0 {
		job.MinDuration = &input.MinDuration
	}

	result, err := s.DPF.AudioSilenceTrim(job)
	if msg := dpfErrorJSON(result, err); msg != "" {
		return msg
	}

	return mustJSON(AudioSilenceTrimOutput{
		Success:    result.Success,
		OutputPath: getFirstOutputPath(result),
		ElapsedMs:  result.ElapsedMs,
	})
}

// ─── Helper Functions ──────────────────────────────────────────────────────────

// getFirstOutputPath extracts the first output path from a dpf result.
func getFirstOutputPath(result *dpf.JobResult) string {
	if result == nil || len(result.Outputs) == 0 {
		return ""
	}
	return result.Outputs[0].Path
}
