package tools

import (
	"context"
	"encoding/json"
	"path/filepath"

	"dev-forge-mcp/internal/dpf"
)

// ─── Image Tool Inputs ───────────────────────────────────────────────────────────

// ImageCropInput is the input schema for the image_crop tool.
type ImageCropInput struct {
	Input  string `json:"input"`
	Output string `json:"output"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// ImageCropOutput is the output schema for the image_crop tool.
type ImageCropOutput struct {
	Success    bool   `json:"success"`
	OutputPath string `json:"output_path,omitempty"`
	Width      uint32 `json:"width,omitempty"`
	Height     uint32 `json:"height,omitempty"`
	ElapsedMs  uint64 `json:"elapsed_ms,omitempty"`
	Error      string `json:"error,omitempty"`
}

// ImageRotateInput is the input schema for the image_rotate tool.
type ImageRotateInput struct {
	Input  string  `json:"input"`
	Output string  `json:"output"`
	Angle  float64 `json:"angle"` // degrees (90, 180, 270)
	FlipH  bool    `json:"flip_h,omitempty"`
	FlipV  bool    `json:"flip_v,omitempty"`
}

// ImageRotateOutput is the output schema for the image_rotate tool.
type ImageRotateOutput struct {
	Success    bool   `json:"success"`
	OutputPath string `json:"output_path,omitempty"`
	Width      uint32 `json:"width,omitempty"`
	Height     uint32 `json:"height,omitempty"`
	ElapsedMs  uint64 `json:"elapsed_ms,omitempty"`
	Error      string `json:"error,omitempty"`
}

// ImageWatermarkInput is the input schema for the image_watermark tool.
type ImageWatermarkInput struct {
	Input     string  `json:"input"`
	Output    string  `json:"output"`
	Text      string  `json:"text,omitempty"`
	ImagePath string  `json:"image_path,omitempty"`
	Position  string  `json:"position,omitempty"` // center, tile, custom
	X         *int    `json:"x,omitempty"`
	Y         *int    `json:"y,omitempty"`
	Opacity   float64 `json:"opacity,omitempty"` // 0.0-1.0
	Size      *int    `json:"size,omitempty"`
	Color     string  `json:"color,omitempty"`
}

// ImageWatermarkOutput is the output schema for the image_watermark tool.
type ImageWatermarkOutput struct {
	Success    bool   `json:"success"`
	OutputPath string `json:"output_path,omitempty"`
	ElapsedMs  uint64 `json:"elapsed_ms,omitempty"`
	Error      string `json:"error,omitempty"`
}

// ImageAdjustInput is the input schema for the image_adjust tool.
type ImageAdjustInput struct {
	Input      string  `json:"input"`
	Output     string  `json:"output"`
	Brightness float64 `json:"brightness,omitempty"` // -100 to 100
	Contrast   float64 `json:"contrast,omitempty"`   // -100 to 100
	Saturation float64 `json:"saturation,omitempty"` // -100 to 100
	Blur       float64 `json:"blur,omitempty"`       // radius in pixels
	Sharpen    float64 `json:"sharpen,omitempty"`    // amount
}

// ImageAdjustOutput is the output schema for the image_adjust tool.
type ImageAdjustOutput struct {
	Success    bool   `json:"success"`
	OutputPath string `json:"output_path,omitempty"`
	ElapsedMs  uint64 `json:"elapsed_ms,omitempty"`
	Error      string `json:"error,omitempty"`
}

// ImageQualityInput is the input schema for the image_quality tool.
type ImageQualityInput struct {
	Input        string `json:"input"`
	Output       string `json:"output"`
	TargetSizeKB int    `json:"target_size_kb"`
	Format       string `json:"format,omitempty"` // webp, jpeg, png
	MaxQuality   *int   `json:"max_quality,omitempty"`
	MinQuality   *int   `json:"min_quality,omitempty"`
}

// ImageQualityOutput is the output schema for the image_quality tool.
type ImageQualityOutput struct {
	Success      bool   `json:"success"`
	OutputPath   string `json:"output_path,omitempty"`
	ActualSizeKB int    `json:"actual_size_kb,omitempty"`
	QualityUsed  int    `json:"quality_used,omitempty"`
	ElapsedMs    uint64 `json:"elapsed_ms,omitempty"`
	Error        string `json:"error,omitempty"`
}

// ImageSrcsetInput is the input schema for the image_srcset tool.
type ImageSrcsetInput struct {
	Input     string   `json:"input"`
	OutputDir string   `json:"output_dir"`
	Widths    []int    `json:"widths,omitempty"` // e.g., [320, 640, 960, 1280]
	Sizes     []string `json:"sizes,omitempty"`  // e.g., ["100vw", "(min-width: 768px) 50vw"]
	Format    string   `json:"format,omitempty"` // webp, jpeg, png
}

// ImageSrcsetOutput is the output schema for the image_srcset tool.
type ImageSrcsetOutput struct {
	Success   bool            `json:"success"`
	Variants  []SrcsetVariant `json:"variants,omitempty"`
	ElapsedMs uint64          `json:"elapsed_ms,omitempty"`
	Error     string          `json:"error,omitempty"`
}

// SrcsetVariant represents a single srcset variant.
type SrcsetVariant struct {
	Path   string `json:"path"`
	Width  uint32 `json:"width"`
	Format string `json:"format"`
	SizeKB int    `json:"size_kb"`
}

// ImageExifInput is the input schema for the image_exif tool.
type ImageExifInput struct {
	Input  string `json:"input"`
	Output string `json:"output,omitempty"`
	ExifOp string `json:"exif_op"` // strip, preserve, extract, auto_orient
}

// ImageExifOutput is the output schema for the image_exif tool.
type ImageExifOutput struct {
	Success    bool                   `json:"success"`
	OutputPath string                 `json:"output_path,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty"` // for extract operation
	ElapsedMs  uint64                 `json:"elapsed_ms,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

// ImageResizeInput is the input schema for the image_resize tool.
type ImageResizeInput struct {
	Input        string   `json:"input"`
	OutputDir    string   `json:"output_dir"`
	Widths       []int    `json:"widths,omitempty"`        // target widths
	ScalePercent *float64 `json:"scale_percent,omitempty"` // e.g., 50.0 for half
	MaxHeight    *int     `json:"max_height,omitempty"`
	Format       string   `json:"format,omitempty"` // webp, jpeg, png, avif
	Quality      *int     `json:"quality,omitempty"`
	Filter       string   `json:"filter,omitempty"`     // lanczos3, gaussian, bilinear, etc.
	LinearRGB    bool     `json:"linear_rgb,omitempty"` // use linear RGB for better quality
}

// ImageResizeOutput is the output schema for the image_resize tool.
type ImageResizeOutput struct {
	Success   bool            `json:"success"`
	Variants  []ResizeVariant `json:"variants,omitempty"`
	ElapsedMs uint64          `json:"elapsed_ms,omitempty"`
	Error     string          `json:"error,omitempty"`
}

// ResizeVariant represents a single resized variant.
type ResizeVariant struct {
	Path   string `json:"path"`
	Width  uint32 `json:"width"`
	Height uint32 `json:"height"`
	Format string `json:"format"`
	SizeKB int    `json:"size_kb"`
}

// ImageConvertInput is the input schema for the image_convert tool.
type ImageConvertInput struct {
	Input   string `json:"input"`
	Output  string `json:"output"`
	Format  string `json:"format"` // webp, jpeg, png, avif, gif
	Quality *int   `json:"quality,omitempty"`
	Width   *int   `json:"width,omitempty"`
	Height  *int   `json:"height,omitempty"`
}

// ImageConvertOutput is the output schema for the image_convert tool.
type ImageConvertOutput struct {
	Success    bool   `json:"success"`
	OutputPath string `json:"output_path,omitempty"`
	Width      uint32 `json:"width,omitempty"`
	Height     uint32 `json:"height,omitempty"`
	Format     string `json:"format,omitempty"`
	SizeKB     int    `json:"size_kb,omitempty"`
	ElapsedMs  uint64 `json:"elapsed_ms,omitempty"`
	Error      string `json:"error,omitempty"`
}

// ImagePlaceholderInput is the input schema for the image_placeholder tool.
type ImagePlaceholderInput struct {
	Input     string `json:"input"`
	Output    string `json:"output,omitempty"` // optional output path
	Kind      string `json:"kind,omitempty"`   // lqip, dominant_color, css_gradient
	LQIPWidth *int   `json:"lqip_width,omitempty"`
	Inline    bool   `json:"inline"` // return base64 data inline
}

// ImagePlaceholderOutput is the output schema for the image_placeholder tool.
type ImagePlaceholderOutput struct {
	Success        bool   `json:"success"`
	OutputPath     string `json:"output_path,omitempty"`
	DataBase64     string `json:"data_base64,omitempty"`
	DominantColor  string `json:"dominant_color,omitempty"` // hex color
	CSSGradient    string `json:"css_gradient,omitempty"`
	LQIPDimensions string `json:"lqip_dimensions,omitempty"` // WxH
	ElapsedMs      uint64 `json:"elapsed_ms,omitempty"`
	Error          string `json:"error,omitempty"`
}

// ImagePaletteInput is the input schema for the image_palette tool.
type ImagePaletteInput struct {
	Input     string   `json:"input"`
	OutputDir string   `json:"output_dir"`
	MaxColors *int     `json:"max_colors,omitempty"` // default 16
	Dithering *float32 `json:"dithering,omitempty"`  // 0.0 to 1.0
	Format    string   `json:"format,omitempty"`     // gif, png
}

// ImagePaletteOutput is the output schema for the image_palette tool.
type ImagePaletteOutput struct {
	Success    bool     `json:"success"`
	OutputPath string   `json:"output_path,omitempty"`
	Colors     []string `json:"colors,omitempty"` // extracted hex colors
	ElapsedMs  uint64   `json:"elapsed_ms,omitempty"`
	Error      string   `json:"error,omitempty"`
}

// ImageSpriteInput is the input schema for the image_sprite tool.
type ImageSpriteInput struct {
	Inputs      []string `json:"inputs"`
	Output      string   `json:"output"`
	CellSize    *int     `json:"cell_size,omitempty"` // width,height per cell
	Columns     *int     `json:"columns,omitempty"`
	Padding     *int     `json:"padding,omitempty"` // pixels between sprites
	GenerateCSS bool     `json:"generate_css,omitempty"`
}

// ImageSpriteOutput is the output schema for the image_sprite tool.
type ImageSpriteOutput struct {
	Success     bool   `json:"success"`
	SpritePath  string `json:"sprite_path,omitempty"`
	CSSPath     string `json:"css_path,omitempty"` // path to generated CSS if requested
	Columns     uint32 `json:"columns,omitempty"`
	CellWidth   uint32 `json:"cell_width,omitempty"`
	CellHeight  uint32 `json:"cell_height,omitempty"`
	TotalImages int    `json:"total_images,omitempty"`
	ElapsedMs   uint64 `json:"elapsed_ms,omitempty"`
	Error       string `json:"error,omitempty"`
}

// ─── Image Tool Handlers ─────────────────────────────────────────────────────────

// ImageCrop implements the image_crop MCP tool.
func (s *Server) ImageCrop(ctx context.Context, input ImageCropInput) string {
	if input.Input == "" {
		return errorJSON("input is required")
	}
	if input.Output == "" {
		return errorJSON("output is required")
	}
	if s.DPF == nil {
		return errorJSON("dpf binary not available. Ensure bin/dpf is installed and executable.")
	}

	w, h := uint32(input.Width), uint32(input.Height)
	job := &dpf.CropJob{
		Operation: "crop",
		Input:     input.Input,
		Output:    input.Output,
		Rect: &dpf.CropRect{
			X:      uint32(input.X),
			Y:      uint32(input.Y),
			Width:  uint32(input.Width),
			Height: uint32(input.Height),
		},
		Width:  &w,
		Height: &h,
	}

	result, err := s.DPF.Crop(job)
	if err != nil {
		return errorJSON("dpf error: " + err.Error())
	}

	return mustJSON(ImageCropOutput{
		Success:    result.Success,
		OutputPath: getFirstOutputPath(result),
		ElapsedMs:  result.ElapsedMs,
	})
}

// ImageRotate implements the image_rotate MCP tool.
func (s *Server) ImageRotate(ctx context.Context, input ImageRotateInput) string {
	if input.Input == "" {
		return errorJSON("input is required")
	}
	if input.Output == "" {
		return errorJSON("output is required")
	}
	if s.DPF == nil {
		return errorJSON("dpf binary not available. Ensure bin/dpf is installed and executable.")
	}

	job := &dpf.RotateJob{
		Operation: "rotate",
		Input:     input.Input,
		Output:    input.Output,
	}
	if input.Angle != 0 {
		job.AngleF = &input.Angle
	}
	if input.FlipH && input.FlipV {
		job.Flip = "both"
	} else if input.FlipH {
		job.Flip = "horizontal"
	} else if input.FlipV {
		job.Flip = "vertical"
	}

	result, err := s.DPF.Rotate(job)
	if err != nil {
		return errorJSON("dpf error: " + err.Error())
	}

	return mustJSON(ImageRotateOutput{
		Success:    result.Success,
		OutputPath: getFirstOutputPath(result),
		ElapsedMs:  result.ElapsedMs,
	})
}

// ImageWatermark implements the image_watermark MCP tool.
func (s *Server) ImageWatermark(ctx context.Context, input ImageWatermarkInput) string {
	if input.Input == "" {
		return errorJSON("input is required")
	}
	if input.Output == "" {
		return errorJSON("output is required")
	}
	if input.Text == "" && input.ImagePath == "" {
		return errorJSON("either text or image_path is required for watermark")
	}
	if s.DPF == nil {
		return errorJSON("dpf binary not available. Ensure bin/dpf is installed and executable.")
	}

	job := &dpf.WatermarkJob{
		Operation: "watermark",
		Input:     input.Input,
		Output:    input.Output,
		Text:      input.Text,
		Image:     input.ImagePath,
		Position:  input.Position,
		Color:     input.Color,
	}
	if input.Opacity != 0 {
		job.Opacity = &input.Opacity
	}
	if input.X != nil {
		v := uint32(*input.X)
		job.OffsetX = &v
	}
	if input.Y != nil {
		v := uint32(*input.Y)
		job.OffsetY = &v
	}
	if input.Size != nil {
		v := uint32(*input.Size)
		job.FontSize = &v
	}

	result, err := s.DPF.Watermark(job)
	if err != nil {
		return errorJSON("dpf error: " + err.Error())
	}

	return mustJSON(ImageWatermarkOutput{
		Success:    result.Success,
		OutputPath: getFirstOutputPath(result),
		ElapsedMs:  result.ElapsedMs,
	})
}

// ImageAdjust implements the image_adjust MCP tool.
func (s *Server) ImageAdjust(ctx context.Context, input ImageAdjustInput) string {
	if input.Input == "" {
		return errorJSON("input is required")
	}
	if input.Output == "" {
		return errorJSON("output is required")
	}
	if s.DPF == nil {
		return errorJSON("dpf binary not available. Ensure bin/dpf is installed and executable.")
	}

	job := &dpf.AdjustJob{
		Operation: "adjust",
		Input:     input.Input,
		Output:    input.Output,
	}
	if input.Brightness != 0 {
		job.Brightness = &input.Brightness
	}
	if input.Contrast != 0 {
		job.Contrast = &input.Contrast
	}
	if input.Saturation != 0 {
		job.Saturation = &input.Saturation
	}
	if input.Blur != 0 {
		job.Blur = &input.Blur
	}
	if input.Sharpen != 0 {
		job.Sharpen = &input.Sharpen
	}

	result, err := s.DPF.Adjust(job)
	if err != nil {
		return errorJSON("dpf error: " + err.Error())
	}

	return mustJSON(ImageAdjustOutput{
		Success:    result.Success,
		OutputPath: getFirstOutputPath(result),
		ElapsedMs:  result.ElapsedMs,
	})
}

// ImageQuality implements the image_quality MCP tool.
func (s *Server) ImageQuality(ctx context.Context, input ImageQualityInput) string {
	if input.Input == "" {
		return errorJSON("input is required")
	}
	if input.Output == "" {
		return errorJSON("output is required")
	}
	if input.TargetSizeKB <= 0 {
		return errorJSON("target_size_kb must be a positive number")
	}
	if s.DPF == nil {
		return errorJSON("dpf binary not available. Ensure bin/dpf is installed and executable.")
	}

	job := &dpf.QualityJob{
		Operation:  "quality",
		Input:      input.Input,
		Output:     input.Output,
		TargetSize: uint64(input.TargetSizeKB) * 1024,
		Format:     input.Format,
	}

	if input.MaxQuality != nil {
		mq := uint8(*input.MaxQuality)
		job.MaxQuality = &mq
	}
	if input.MinQuality != nil {
		mq := uint8(*input.MinQuality)
		job.MinQuality = &mq
	}

	result, err := s.DPF.AutoQuality(job)
	if err != nil {
		return errorJSON("dpf error: " + err.Error())
	}

	return mustJSON(ImageQualityOutput{
		Success:    result.Success,
		OutputPath: getFirstOutputPath(result),
		ElapsedMs:  result.ElapsedMs,
	})
}

// ImageSrcset implements the image_srcset MCP tool.
func (s *Server) ImageSrcset(ctx context.Context, input ImageSrcsetInput) string {
	if input.Input == "" {
		return errorJSON("input is required")
	}
	if input.OutputDir == "" {
		return errorJSON("output_dir is required")
	}
	if s.DPF == nil {
		return errorJSON("dpf binary not available. Ensure bin/dpf is installed and executable.")
	}

	// Convert int widths to uint32
	var u32Widths []uint32
	for _, w := range input.Widths {
		u32Widths = append(u32Widths, uint32(w))
	}

	job := &dpf.SrcsetJob{
		Operation: "srcset",
		Input:     input.Input,
		OutputDir: input.OutputDir,
		Widths:    u32Widths,
		Format:    input.Format,
	}

	result, err := s.DPF.Srcset(job)
	if err != nil {
		return errorJSON("dpf error: " + err.Error())
	}

	var variants []SrcsetVariant
	for _, out := range result.Outputs {
		variants = append(variants, SrcsetVariant{
			Path:   out.Path,
			Width:  out.Width,
			Format: out.Format,
			SizeKB: int(out.SizeBytes / 1024),
		})
	}

	if variants == nil {
		variants = []SrcsetVariant{}
	}

	return mustJSON(ImageSrcsetOutput{
		Success:   result.Success,
		Variants:  variants,
		ElapsedMs: result.ElapsedMs,
	})
}

// ImageExif implements the image_exif MCP tool.
func (s *Server) ImageExif(ctx context.Context, input ImageExifInput) string {
	if input.Input == "" {
		return errorJSON("input is required")
	}
	if input.ExifOp == "" {
		return errorJSON("exif_op is required (strip, preserve, extract, auto_orient)")
	}
	if s.DPF == nil {
		return errorJSON("dpf binary not available. Ensure bin/dpf is installed and executable.")
	}

	job := &dpf.ExifJob{
		Operation: "exif",
		Input:     input.Input,
		Output:    input.Output,
		ExifOp:    input.ExifOp,
	}

	result, err := s.DPF.Exif(job)
	if err != nil {
		return errorJSON("dpf error: " + err.Error())
	}

	// Parse metadata if available (for extract operation)
	var data map[string]interface{}
	if result.Metadata != nil {
		json.Unmarshal(*result.Metadata, &data)
	}

	return mustJSON(ImageExifOutput{
		Success:    result.Success,
		OutputPath: getFirstOutputPath(result),
		Data:       data,
		ElapsedMs:  result.ElapsedMs,
	})
}

// ImageResize implements the image_resize MCP tool.
func (s *Server) ImageResize(ctx context.Context, input ImageResizeInput) string {
	if input.Input == "" {
		return errorJSON("input is required")
	}
	if input.OutputDir == "" && input.ScalePercent == nil {
		return errorJSON("either output_dir or scale_percent is required")
	}
	if s.DPF == nil {
		return errorJSON("dpf binary not available. Ensure bin/dpf is installed and executable.")
	}

	// Convert int widths to uint32
	var u32Widths []uint32
	for _, w := range input.Widths {
		u32Widths = append(u32Widths, uint32(w))
	}

	outputDir := input.OutputDir
	if outputDir == "" {
		// If no output dir but using scale_percent, use same directory
		outputDir = filepath.Dir(input.Input)
	}

	job := &dpf.ResizeJob{
		Operation: "resize",
		Input:     input.Input,
		OutputDir: outputDir,
		Widths:    u32Widths,
		Format:    strPtr(input.Format),
		LinearRGB: input.LinearRGB,
	}

	if input.ScalePercent != nil {
		job.ScalePercent = floatPtr(*input.ScalePercent)
	}
	if input.MaxHeight != nil {
		mh := uint32(*input.MaxHeight)
		job.MaxHeight = &mh
	}
	if input.Quality != nil {
		q := uint8(*input.Quality)
		job.Quality = &q
	}
	if input.Filter != "" {
		job.Filter = &input.Filter
	}

	result, err := s.DPF.Execute(job)
	if err != nil {
		return errorJSON("dpf error: " + err.Error())
	}

	var variants []ResizeVariant
	for _, out := range result.Outputs {
		variants = append(variants, ResizeVariant{
			Path:   out.Path,
			Width:  out.Width,
			Height: out.Height,
			Format: out.Format,
			SizeKB: int(out.SizeBytes / 1024),
		})
	}

	if variants == nil {
		variants = []ResizeVariant{}
	}

	return mustJSON(ImageResizeOutput{
		Success:   result.Success,
		Variants:  variants,
		ElapsedMs: result.ElapsedMs,
	})
}

// ImageConvert implements the image_convert MCP tool.
func (s *Server) ImageConvert(ctx context.Context, input ImageConvertInput) string {
	if input.Input == "" {
		return errorJSON("input is required")
	}
	if input.Output == "" {
		return errorJSON("output is required")
	}
	if input.Format == "" {
		return errorJSON("format is required (webp, jpeg, png, avif, gif)")
	}
	if s.DPF == nil {
		return errorJSON("dpf binary not available. Ensure bin/dpf is installed and executable.")
	}

	job := &dpf.ConvertJob{
		Operation: "convert",
		Input:     input.Input,
		Output:    input.Output,
		Format:    input.Format,
	}

	if input.Quality != nil {
		q := uint8(*input.Quality)
		job.Quality = &q
	}
	if input.Width != nil {
		w := uint32(*input.Width)
		job.Width = &w
	}
	if input.Height != nil {
		h := uint32(*input.Height)
		job.Height = &h
	}

	result, err := s.DPF.Execute(job)
	if err != nil {
		return errorJSON("dpf error: " + err.Error())
	}

	var sizeKB int
	if len(result.Outputs) > 0 {
		sizeKB = int(result.Outputs[0].SizeBytes / 1024)
	}

	return mustJSON(ImageConvertOutput{
		Success:    result.Success,
		OutputPath: getFirstOutputPath(result),
		ElapsedMs:  result.ElapsedMs,
		SizeKB:     sizeKB,
	})
}

// ImagePlaceholder implements the image_placeholder MCP tool.
func (s *Server) ImagePlaceholder(ctx context.Context, input ImagePlaceholderInput) string {
	if input.Input == "" {
		return errorJSON("input is required")
	}
	if s.DPF == nil {
		return errorJSON("dpf binary not available. Ensure bin/dpf is installed and executable.")
	}

	output := input.Output
	job := &dpf.PlaceholderJob{
		Operation: "placeholder",
		Input:     input.Input,
		Inline:    input.Inline,
	}

	if output != "" {
		job.Output = &output
	}
	if input.Kind != "" {
		job.Kind = &input.Kind
	}
	if input.LQIPWidth != nil {
		w := uint32(*input.LQIPWidth)
		job.LQIPWidth = &w
	}

	result, err := s.DPF.Execute(job)
	if err != nil {
		return errorJSON("dpf error: " + err.Error())
	}

	var dataBase64, dominantColor, cssGradient, lqipDimensions string
	if len(result.Outputs) > 0 {
		if result.Outputs[0].DataBase64 != nil {
			dataBase64 = *result.Outputs[0].DataBase64
		}
	}

	// Parse metadata for additional info
	var meta map[string]interface{}
	if result.Metadata != nil {
		json.Unmarshal(*result.Metadata, &meta)
		if color, ok := meta["dominant_color"].(string); ok {
			dominantColor = color
		}
		if gradient, ok := meta["css_gradient"].(string); ok {
			cssGradient = gradient
		}
		if w, ok := meta["lqip_width"].(float64); ok {
			if h, ok := meta["lqip_height"].(float64); ok {
				lqipDimensions = formatDimensions(int(w), int(h))
			}
		}
	}

	return mustJSON(ImagePlaceholderOutput{
		Success:        result.Success,
		OutputPath:     getFirstOutputPath(result),
		DataBase64:     dataBase64,
		DominantColor:  dominantColor,
		CSSGradient:    cssGradient,
		LQIPDimensions: lqipDimensions,
		ElapsedMs:      result.ElapsedMs,
	})
}

// ImagePalette implements the image_palette MCP tool.
func (s *Server) ImagePalette(ctx context.Context, input ImagePaletteInput) string {
	if input.Input == "" {
		return errorJSON("input is required")
	}
	if input.OutputDir == "" {
		return errorJSON("output_dir is required")
	}
	if s.DPF == nil {
		return errorJSON("dpf binary not available. Ensure bin/dpf is installed and executable.")
	}

	job := &dpf.PaletteJob{
		Operation: "palette",
		Input:     input.Input,
		OutputDir: input.OutputDir,
		Format:    strPtr(input.Format),
	}

	if input.MaxColors != nil {
		mc := uint32(*input.MaxColors)
		job.MaxColors = &mc
	}
	if input.Dithering != nil {
		job.Dithering = input.Dithering
	}

	result, err := s.DPF.Execute(job)
	if err != nil {
		return errorJSON("dpf error: " + err.Error())
	}

	// Extract colors from metadata if available
	var colors []string
	var meta map[string]interface{}
	if result.Metadata != nil {
		json.Unmarshal(*result.Metadata, &meta)
		if c, ok := meta["colors"].([]interface{}); ok {
			for _, color := range c {
				if cStr, ok := color.(string); ok {
					colors = append(colors, cStr)
				}
			}
		}
	}

	return mustJSON(ImagePaletteOutput{
		Success:    result.Success,
		OutputPath: getFirstOutputPath(result),
		Colors:     colors,
		ElapsedMs:  result.ElapsedMs,
	})
}

// ImageSprite implements the image_sprite MCP tool.
func (s *Server) ImageSprite(ctx context.Context, input ImageSpriteInput) string {
	if len(input.Inputs) == 0 {
		return errorJSON("inputs is required and must not be empty")
	}
	if input.Output == "" {
		return errorJSON("output is required")
	}
	if s.DPF == nil {
		return errorJSON("dpf binary not available. Ensure bin/dpf is installed and executable.")
	}

	job := &dpf.SpriteJob{
		Operation:   "sprite",
		Inputs:      input.Inputs,
		Output:      input.Output,
		GenerateCSS: input.GenerateCSS,
	}

	if input.CellSize != nil {
		cs := uint32(*input.CellSize)
		job.CellSize = &cs
	}
	if input.Columns != nil {
		col := uint32(*input.Columns)
		job.Columns = &col
	}
	if input.Padding != nil {
		p := uint32(*input.Padding)
		job.Padding = &p
	}

	result, err := s.DPF.Execute(job)
	if err != nil {
		return errorJSON("dpf error: " + err.Error())
	}

	return mustJSON(ImageSpriteOutput{
		Success:     result.Success,
		SpritePath:  getFirstOutputPath(result),
		TotalImages: len(input.Inputs),
		ElapsedMs:   result.ElapsedMs,
	})
}

// ─── Helper Functions ───────────────────────────────────────────────────────────

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func floatPtr(f float64) *float32 {
	ff := float32(f)
	return &ff
}

func formatDimensions(width, height int) string {
	return string(rune('0'+width/1000%10)) + string(rune('0'+width/100%10)) + "x" +
		string(rune('0'+height/1000%10)) + string(rune('0'+height/100%10))
}
