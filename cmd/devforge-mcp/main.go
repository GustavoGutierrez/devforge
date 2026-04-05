// cmd/dev-forge-mcp is the MCP server entry point.
// It exposes design, image, and pattern tools via the MCP stdio transport.
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"dev-forge-mcp/internal/config"
	"dev-forge-mcp/internal/dpf"
	"dev-forge-mcp/internal/tools"
)

// mcpApp holds all server dependencies with hot-reload support.
type mcpApp struct {
	srv        *tools.Server
	mu         sync.RWMutex
	geminiKey  string
	imageModel string
}

func (a *mcpApp) getGeminiKey() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.geminiKey
}

func (a *mcpApp) setGeminiKey(key string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.geminiKey = key
}

func (a *mcpApp) getImageModel() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.imageModel
}

func main() {
	// 1. Load config
	cfg, err := config.Load()
	if err != nil {
		log.Printf("warning: could not load config: %v", err)
		cfg = &config.Config{
			ImageModel: "gemini-2.5-flash-image",
		}
	}

	// Resolve paths relative to the executable so the server works regardless of CWD.
	exeDir, err := executableDir()
	if err != nil {
		log.Fatalf("failed to resolve executable directory: %v", err)
	}

	// 2. Initialize StreamClient for dpf (DevPixelForge)
	var sc *dpf.StreamClient
	dpfPath, err := dpf.ResolveBinaryPath(exeDir)
	if err != nil {
		log.Printf("warning: dpf binary not available: %v", err)
		log.Printf("optimize_images, generate_favicon, markdown_to_pdf, and media tools will return errors")
		sc = nil
	} else {
		sc, err = dpf.NewStreamClient(dpfPath)
	}
	if err != nil {
		log.Printf("warning: dpf binary not available at %s: %v", dpfPath, err)
		log.Printf("optimize_images, generate_favicon, markdown_to_pdf, and media tools will return errors")
		sc = nil
	} else {
		log.Printf("dpf available at %s", dpfPath)
		defer sc.Close()
	}

	// 3. Build app state
	app := &mcpApp{
		srv: &tools.Server{
			DPF: sc,
		},
		geminiKey:  cfg.GeminiAPIKey,
		imageModel: cfg.ImageModel,
	}

	// 4. Build MCP server and register all tools
	s := mcpserver.NewMCPServer("devforge", "2.1.2",
		mcpserver.WithToolCapabilities(true),
	)

	registerTools(s, app)

	// 5. Serve via stdio transport
	if err := mcpserver.ServeStdio(s); err != nil {
		log.Fatalf("mcp server error: %v", err)
	}
}

// executableDir returns the directory that contains the running binary,
// resolving symlinks so the path is always the real location.
func executableDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", err
	}
	return filepath.Dir(resolved), nil
}

func registerTools(s *mcpserver.MCPServer, app *mcpApp) {
	// ── Utility groups ───────────────────────────────────────────
	registerTextEncTools(s, app)
	registerDataFmtTools(s, app)
	registerCryptoUtilTools(s, app)
	registerHTTPTools(s, app)
	registerDateTimeTools(s, app)
	registerFileTools(s, app)
	registerFrontendTools(s, app)
	registerBackendTools(s, app)
	registerCodeTools(s, app)

	// ── analyze_layout ──────────────────────────────────────────
	s.AddTool(mcp.NewTool("analyze_layout",
		mcp.WithDescription("Audit HTML/JSX markup for layout issues including accessibility, spacing, typography, and framework-specific conventions."),
		mcp.WithString("markup", mcp.Required(), mcp.Description("HTML or JSX string to analyze")),
		mcp.WithObject("stack", mcp.Required(), mcp.Description("CSS stack metadata")),
		mcp.WithString("page_type", mcp.Description("Page type: landing | dashboard | form")),
		mcp.WithString("device_focus", mcp.Description("Device focus: mobile | desktop | both")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		input := tools.AnalyzeLayoutInput{
			Markup:      mcp.ParseString(req, "markup", ""),
			PageType:    mcp.ParseString(req, "page_type", ""),
			DeviceFocus: mcp.ParseString(req, "device_focus", ""),
		}
		if stackMap, ok := args["stack"].(map[string]interface{}); ok {
			input.Stack.CSSMode = strVal(stackMap, "css_mode")
			input.Stack.Framework = strVal(stackMap, "framework")
		}
		return mcp.NewToolResultText(app.srv.AnalyzeLayout(ctx, input)), nil
	})

	// ── suggest_layout ──────────────────────────────────────────
	s.AddTool(mcp.NewTool("suggest_layout",
		mcp.WithDescription("Generate a layout scaffold based on a description and stack metadata."),
		mcp.WithString("description", mcp.Required(), mcp.Description("Layout description")),
		mcp.WithObject("stack", mcp.Required(), mcp.Description("CSS stack metadata")),
		mcp.WithString("fidelity", mcp.Required(), mcp.Description("wireframe | mid | production")),
		mcp.WithObject("tokens_profile", mcp.Description("Existing token values to incorporate")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		input := tools.SuggestLayoutInput{
			Description: mcp.ParseString(req, "description", ""),
			Fidelity:    mcp.ParseString(req, "fidelity", "mid"),
		}
		if stackMap, ok := args["stack"].(map[string]interface{}); ok {
			input.Stack.CSSMode = strVal(stackMap, "css_mode")
			input.Stack.Framework = strVal(stackMap, "framework")
		}
		return mcp.NewToolResultText(app.srv.SuggestLayout(ctx, input)), nil
	})

	// ── manage_tokens ───────────────────────────────────────────
	s.AddTool(mcp.NewTool("manage_tokens",
		mcp.WithDescription("Read or update design tokens (colors, spacing, typography)."),
		mcp.WithString("mode", mcp.Required(), mcp.Description("read | plan-update | apply-update")),
		mcp.WithString("css_mode", mcp.Required(), mcp.Description("tailwind-v4 | plain-css")),
		mcp.WithString("scope", mcp.Required(), mcp.Description("colors | spacing | typography | all")),
		mcp.WithObject("proposal", mcp.Description("Token key/value pairs to apply (required for apply-update)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		input := tools.ManageTokensInput{
			Mode:    mcp.ParseString(req, "mode", ""),
			CSSMode: mcp.ParseString(req, "css_mode", ""),
			Scope:   mcp.ParseString(req, "scope", "all"),
		}
		if proposal, ok := args["proposal"].(map[string]interface{}); ok {
			input.Proposal = make(map[string]interface{})
			for k, v := range proposal {
				input.Proposal[k] = v
			}
		}
		return mcp.NewToolResultText(app.srv.ManageTokens(ctx, input)), nil
	})

	// ── generate_ui_image ────────────────────────────────────────
	s.AddTool(mcp.NewTool("generate_ui_image",
		mcp.WithDescription("Generate a UI image via Gemini API. Requires gemini_api_key to be configured."),
		mcp.WithString("prompt", mcp.Required(), mcp.Description("Image generation prompt")),
		mcp.WithString("style", mcp.Required(), mcp.Description("wireframe | mockup | illustration")),
		mcp.WithNumber("width", mcp.Description("Image width in pixels (default 1280)")),
		mcp.WithNumber("height", mcp.Description("Image height in pixels (default 720)")),
		mcp.WithString("output_path", mcp.Required(), mcp.Description("File path to save the generated image")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		input := tools.GenerateUIImageInput{
			Prompt:     mcp.ParseString(req, "prompt", ""),
			Style:      mcp.ParseString(req, "style", "mockup"),
			Width:      mcp.ParseInt(req, "width", 1280),
			Height:     mcp.ParseInt(req, "height", 720),
			OutputPath: mcp.ParseString(req, "output_path", ""),
		}
		return mcp.NewToolResultText(app.srv.GenerateUIImage(ctx, input, app.getGeminiKey(), app.getImageModel())), nil
	})

	// ── ui2md ────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool("ui2md",
		mcp.WithDescription("Analyze a UI screenshot and generate a Markdown design spec using Gemini vision."),
		mcp.WithString("image_path", mcp.Required(), mcp.Description("Path to the UI image to analyze")),
		mcp.WithString("output_dir", mcp.Description("Directory to save the generated markdown (default: same as image)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		input := tools.UI2MDInput{
			ImagePath: mcp.ParseString(req, "image_path", ""),
			OutputDir: mcp.ParseString(req, "output_dir", ""),
		}
		return mcp.NewToolResultText(app.srv.UI2MD(ctx, input, app.getGeminiKey(), app.getImageModel())), nil
	})

	// ── markdown_to_pdf ─────────────────────────────────────────
	s.AddTool(mcp.NewTool("markdown_to_pdf",
		mcp.WithDescription("Convert Markdown to PDF via dpf 0.4.2 using file, inline text, or base64 input."),
		mcp.WithString("input", mcp.Description("Markdown file path")),
		mcp.WithString("markdown_text", mcp.Description("Inline UTF-8 Markdown source")),
		mcp.WithString("markdown_base64", mcp.Description("Base64-encoded UTF-8 Markdown source")),
		mcp.WithString("output", mcp.Description("Explicit output PDF path")),
		mcp.WithString("output_dir", mcp.Description("Directory output mode")),
		mcp.WithString("file_name", mcp.Description("Optional output filename when using output_dir")),
		mcp.WithBoolean("inline", mcp.Description("Return base64 PDF data inline")),
		mcp.WithString("page_size", mcp.Description("a4 | letter | legal")),
		mcp.WithNumber("page_width_mm", mcp.Description("Custom page width in millimeters")),
		mcp.WithNumber("page_height_mm", mcp.Description("Custom page height in millimeters")),
		mcp.WithString("layout_mode", mcp.Description("paged | single_page")),
		mcp.WithString("theme", mcp.Description("invoice | scientific_article | professional | engineering | informational")),
		mcp.WithObject("theme_config", mcp.Description("Theme overrides forwarded to dpf")),
		mcp.WithObject("resource_files", mcp.Description("Optional href-to-file mapping for inline assets")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		input := tools.MarkdownToPDFInput{
			Input:          mcp.ParseString(req, "input", ""),
			MarkdownText:   mcp.ParseString(req, "markdown_text", ""),
			MarkdownBase64: mcp.ParseString(req, "markdown_base64", ""),
			Output:         mcp.ParseString(req, "output", ""),
			OutputDir:      mcp.ParseString(req, "output_dir", ""),
			FileName:       mcp.ParseString(req, "file_name", ""),
			Inline:         mcp.ParseBoolean(req, "inline", false),
			PageSize:       mcp.ParseString(req, "page_size", ""),
			LayoutMode:     mcp.ParseString(req, "layout_mode", ""),
			Theme:          mcp.ParseString(req, "theme", ""),
		}
		if v, ok := args["page_width_mm"].(float64); ok {
			input.PageWidthMM = &v
		}
		if v, ok := args["page_height_mm"].(float64); ok {
			input.PageHeightMM = &v
		}
		if themeConfig, ok := args["theme_config"].(map[string]interface{}); ok {
			input.ThemeConfig = themeConfig
		}
		if resourceFiles, ok := args["resource_files"].(map[string]interface{}); ok {
			input.ResourceFiles = make(map[string]string, len(resourceFiles))
			for k, v := range resourceFiles {
				if s, ok := v.(string); ok {
					input.ResourceFiles[k] = s
				}
			}
		}
		return mcp.NewToolResultText(app.srv.MarkdownToPDF(ctx, input)), nil
	})

	// ── configure_gemini ────────────────────────────────────────
	s.AddTool(mcp.NewTool("configure_gemini",
		mcp.WithDescription("Save Gemini API key to config file and hot-reload without restart."),
		mcp.WithString("api_key", mcp.Required(), mcp.Description("Gemini API key")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		input := tools.ConfigureGeminiInput{
			APIKey: mcp.ParseString(req, "api_key", ""),
		}
		result := app.srv.ConfigureGemini(ctx, input, app.setGeminiKey)
		return mcp.NewToolResultText(result), nil
	})

	// ── optimize_images ─────────────────────────────────────────
	s.AddTool(mcp.NewTool("optimize_images",
		mcp.WithDescription("Optimize and convert images using the Rust dpf (DevPixelForge) engine."),
		mcp.WithArray("inputs", mcp.Required(), mcp.Description("Array of image optimization requests"),
			mcp.Items(map[string]any{"type": "object"})),
		mcp.WithNumber("parallelism", mcp.Description("Max parallel operations (default 4)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		var optInputs []tools.OptimizeInput
		if inputsRaw, ok := args["inputs"]; ok {
			data, _ := json.Marshal(inputsRaw)
			json.Unmarshal(data, &optInputs)
		}
		input := tools.OptimizeImagesInput{
			Inputs:      optInputs,
			Parallelism: mcp.ParseInt(req, "parallelism", 4),
		}
		return mcp.NewToolResultText(app.srv.OptimizeImages(ctx, input)), nil
	})

	// ── generate_favicon ────────────────────────────────────────
	s.AddTool(mcp.NewTool("generate_favicon",
		mcp.WithDescription("Generate favicon variants (ico, png, svg) from a source image."),
		mcp.WithString("source_path", mcp.Required(), mcp.Description("Path to source image (PNG or SVG)")),
		mcp.WithString("background_color", mcp.Description("Hex background color (default #ffffff)")),
		mcp.WithArray("sizes", mcp.Description("Icon sizes (default [16,32,48,180,192,512])"), mcp.WithNumberItems()),
		mcp.WithArray("formats", mcp.Description("Output formats: ico | png | svg"), mcp.WithStringItems()),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		input := tools.GenerateFaviconInput{
			SourcePath:      mcp.ParseString(req, "source_path", ""),
			BackgroundColor: mcp.ParseString(req, "background_color", "#ffffff"),
		}
		if sizesRaw, ok := args["sizes"]; ok {
			data, _ := json.Marshal(sizesRaw)
			var sizesF []float64
			if json.Unmarshal(data, &sizesF) == nil {
				for _, f := range sizesF {
					input.Sizes = append(input.Sizes, int(f))
				}
			}
		}
		if formatsRaw, ok := args["formats"]; ok {
			data, _ := json.Marshal(formatsRaw)
			json.Unmarshal(data, &input.Formats)
		}
		return mcp.NewToolResultText(app.srv.GenerateFavicon(ctx, input)), nil
	})

	// ── Image Suite Tools ────────────────────────────────────────

	// image_crop
	s.AddTool(mcp.NewTool("image_crop",
		mcp.WithDescription("Crop an image to specific dimensions."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input image path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output image path")),
		mcp.WithNumber("x", mcp.Required(), mcp.Description("X coordinate of top-left corner")),
		mcp.WithNumber("y", mcp.Required(), mcp.Description("Y coordinate of top-left corner")),
		mcp.WithNumber("width", mcp.Required(), mcp.Description("Crop width in pixels")),
		mcp.WithNumber("height", mcp.Required(), mcp.Description("Crop height in pixels")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		input := tools.ImageCropInput{
			Input:  mcp.ParseString(req, "input", ""),
			Output: mcp.ParseString(req, "output", ""),
			X:      int(numVal(args, "x", 0)),
			Y:      int(numVal(args, "y", 0)),
			Width:  int(numVal(args, "width", 0)),
			Height: int(numVal(args, "height", 0)),
		}
		return mcp.NewToolResultText(app.srv.ImageCrop(ctx, input)), nil
	})

	// image_rotate
	s.AddTool(mcp.NewTool("image_rotate",
		mcp.WithDescription("Rotate and/or flip an image."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input image path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output image path")),
		mcp.WithNumber("angle", mcp.Description("Rotation angle in degrees (90, 180, 270)")),
		mcp.WithBoolean("flip_h", mcp.Description("Horizontal flip")),
		mcp.WithBoolean("flip_v", mcp.Description("Vertical flip")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		input := tools.ImageRotateInput{
			Input:  mcp.ParseString(req, "input", ""),
			Output: mcp.ParseString(req, "output", ""),
			Angle:  numVal(argsMap(req), "angle", 0),
			FlipH:  mcp.ParseBoolean(req, "flip_h", false),
			FlipV:  mcp.ParseBoolean(req, "flip_v", false),
		}
		return mcp.NewToolResultText(app.srv.ImageRotate(ctx, input)), nil
	})

	// image_watermark
	s.AddTool(mcp.NewTool("image_watermark",
		mcp.WithDescription("Add a watermark (text or image) to an image."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input image path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output image path")),
		mcp.WithString("text", mcp.Description("Text watermark")),
		mcp.WithString("image_path", mcp.Description("Image watermark path")),
		mcp.WithString("position", mcp.Description("Position: center, tile, custom")),
		mcp.WithNumber("x", mcp.Description("X coordinate for custom position")),
		mcp.WithNumber("y", mcp.Description("Y coordinate for custom position")),
		mcp.WithNumber("opacity", mcp.Description("Opacity 0.0-1.0")),
		mcp.WithNumber("size", mcp.Description("Font size for text watermark")),
		mcp.WithString("color", mcp.Description("Text color (hex)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		input := tools.ImageWatermarkInput{
			Input:     mcp.ParseString(req, "input", ""),
			Output:    mcp.ParseString(req, "output", ""),
			Text:      mcp.ParseString(req, "text", ""),
			ImagePath: mcp.ParseString(req, "image_path", ""),
			Position:  mcp.ParseString(req, "position", ""),
			Opacity:   numVal(args, "opacity", 1.0),
			Color:     mcp.ParseString(req, "color", ""),
		}
		if x, ok := args["x"].(float64); ok {
			xi := int(x)
			input.X = &xi
		}
		if y, ok := args["y"].(float64); ok {
			yi := int(y)
			input.Y = &yi
		}
		if size, ok := args["size"].(float64); ok {
			si := int(size)
			input.Size = &si
		}
		return mcp.NewToolResultText(app.srv.ImageWatermark(ctx, input)), nil
	})

	// image_adjust
	s.AddTool(mcp.NewTool("image_adjust",
		mcp.WithDescription("Adjust image properties (brightness, contrast, saturation, blur, sharpen)."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input image path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output image path")),
		mcp.WithNumber("brightness", mcp.Description("Brightness adjustment -100 to 100")),
		mcp.WithNumber("contrast", mcp.Description("Contrast adjustment -100 to 100")),
		mcp.WithNumber("saturation", mcp.Description("Saturation adjustment -100 to 100")),
		mcp.WithNumber("blur", mcp.Description("Blur radius in pixels")),
		mcp.WithNumber("sharpen", mcp.Description("Sharpen amount")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		input := tools.ImageAdjustInput{
			Input:      mcp.ParseString(req, "input", ""),
			Output:     mcp.ParseString(req, "output", ""),
			Brightness: numVal(args, "brightness", 0),
			Contrast:   numVal(args, "contrast", 0),
			Saturation: numVal(args, "saturation", 0),
			Blur:       numVal(args, "blur", 0),
			Sharpen:    numVal(args, "sharpen", 0),
		}
		return mcp.NewToolResultText(app.srv.ImageAdjust(ctx, input)), nil
	})

	// image_quality
	s.AddTool(mcp.NewTool("image_quality",
		mcp.WithDescription("Optimize image quality to target file size using binary search."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input image path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output image path")),
		mcp.WithNumber("target_size_kb", mcp.Required(), mcp.Description("Target file size in KB")),
		mcp.WithString("format", mcp.Description("Output format: webp, jpeg, png")),
		mcp.WithNumber("max_quality", mcp.Description("Maximum quality 1-100")),
		mcp.WithNumber("min_quality", mcp.Description("Minimum quality 1-100")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		input := tools.ImageQualityInput{
			Input:        mcp.ParseString(req, "input", ""),
			Output:       mcp.ParseString(req, "output", ""),
			TargetSizeKB: int(numVal(args, "target_size_kb", 100)),
			Format:       mcp.ParseString(req, "format", ""),
		}
		if maxQ, ok := args["max_quality"].(float64); ok {
			mqi := int(maxQ)
			input.MaxQuality = &mqi
		}
		if minQ, ok := args["min_quality"].(float64); ok {
			mqi := int(minQ)
			input.MinQuality = &mqi
		}
		return mcp.NewToolResultText(app.srv.ImageQuality(ctx, input)), nil
	})

	// image_srcset
	s.AddTool(mcp.NewTool("image_srcset",
		mcp.WithDescription("Generate responsive image variants for srcset attribute."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input image path")),
		mcp.WithString("output_dir", mcp.Required(), mcp.Description("Output directory")),
		mcp.WithArray("widths", mcp.Description("Target widths (e.g., [320, 640, 960, 1280])"), mcp.WithNumberItems()),
		mcp.WithArray("sizes", mcp.Description("Sizes attribute values"), mcp.WithStringItems()),
		mcp.WithString("format", mcp.Description("Output format: webp, jpeg, png")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		input := tools.ImageSrcsetInput{
			Input:     mcp.ParseString(req, "input", ""),
			OutputDir: mcp.ParseString(req, "output_dir", ""),
			Format:    mcp.ParseString(req, "format", ""),
		}
		if widthsRaw, ok := args["widths"]; ok {
			data, _ := json.Marshal(widthsRaw)
			json.Unmarshal(data, &input.Widths)
		}
		if sizesRaw, ok := args["sizes"]; ok {
			data, _ := json.Marshal(sizesRaw)
			json.Unmarshal(data, &input.Sizes)
		}
		return mcp.NewToolResultText(app.srv.ImageSrcset(ctx, input)), nil
	})

	// image_exif
	s.AddTool(mcp.NewTool("image_exif",
		mcp.WithDescription("EXIF operations: strip, preserve, extract, or auto-orient."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input image path")),
		mcp.WithString("output", mcp.Description("Output image path (required for strip/preserve)")),
		mcp.WithString("exif_op", mcp.Required(), mcp.Description("Operation: strip, preserve, extract, auto_orient")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		input := tools.ImageExifInput{
			Input:  mcp.ParseString(req, "input", ""),
			Output: mcp.ParseString(req, "output", ""),
			ExifOp: mcp.ParseString(req, "exif_op", ""),
		}
		return mcp.NewToolResultText(app.srv.ImageExif(ctx, input)), nil
	})

	// image_resize
	s.AddTool(mcp.NewTool("image_resize",
		mcp.WithDescription("Resize an image to multiple widths or by percentage."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input image path")),
		mcp.WithString("output_dir", mcp.Description("Output directory (for width-based resize)")),
		mcp.WithArray("widths", mcp.Description("Target widths"), mcp.WithNumberItems()),
		mcp.WithNumber("scale_percent", mcp.Description("Scale by percentage (e.g., 50.0 for half)")),
		mcp.WithNumber("max_height", mcp.Description("Maximum height constraint")),
		mcp.WithString("format", mcp.Description("Output format: webp, jpeg, png, avif")),
		mcp.WithNumber("quality", mcp.Description("Quality 1-100")),
		mcp.WithString("filter", mcp.Description("Resampling filter: lanczos3, gaussian, bilinear")),
		mcp.WithBoolean("linear_rgb", mcp.Description("Use linear RGB for better quality")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		input := tools.ImageResizeInput{
			Input:     mcp.ParseString(req, "input", ""),
			OutputDir: mcp.ParseString(req, "output_dir", ""),
			Format:    mcp.ParseString(req, "format", ""),
			Filter:    mcp.ParseString(req, "filter", ""),
			LinearRGB: mcp.ParseBoolean(req, "linear_rgb", false),
		}
		if widthsRaw, ok := args["widths"]; ok {
			data, _ := json.Marshal(widthsRaw)
			json.Unmarshal(data, &input.Widths)
		}
		if sp, ok := args["scale_percent"].(float64); ok {
			input.ScalePercent = &sp
		}
		if mh, ok := args["max_height"].(float64); ok {
			mhi := int(mh)
			input.MaxHeight = &mhi
		}
		if q, ok := args["quality"].(float64); ok {
			qi := int(q)
			input.Quality = &qi
		}
		return mcp.NewToolResultText(app.srv.ImageResize(ctx, input)), nil
	})

	// image_convert
	s.AddTool(mcp.NewTool("image_convert",
		mcp.WithDescription("Convert an image to a different format."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input image path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output image path")),
		mcp.WithString("format", mcp.Required(), mcp.Description("Target format: webp, jpeg, png, avif, gif")),
		mcp.WithNumber("quality", mcp.Description("Quality 1-100")),
		mcp.WithNumber("width", mcp.Description("Target width")),
		mcp.WithNumber("height", mcp.Description("Target height")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		input := tools.ImageConvertInput{
			Input:  mcp.ParseString(req, "input", ""),
			Output: mcp.ParseString(req, "output", ""),
			Format: mcp.ParseString(req, "format", ""),
		}
		if q, ok := args["quality"].(float64); ok {
			qi := int(q)
			input.Quality = &qi
		}
		if w, ok := args["width"].(float64); ok {
			wi := int(w)
			input.Width = &wi
		}
		if h, ok := args["height"].(float64); ok {
			hi := int(h)
			input.Height = &hi
		}
		return mcp.NewToolResultText(app.srv.ImageConvert(ctx, input)), nil
	})

	// image_placeholder
	s.AddTool(mcp.NewTool("image_placeholder",
		mcp.WithDescription("Generate image placeholders (LQIP, dominant color, CSS gradient)."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input image path")),
		mcp.WithString("output", mcp.Description("Optional output path for placeholder")),
		mcp.WithString("kind", mcp.Description("Placeholder kind: lqip, dominant_color, css_gradient")),
		mcp.WithNumber("lqip_width", mcp.Description("LQIP width in pixels")),
		mcp.WithBoolean("inline", mcp.Description("Return base64 data inline")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		input := tools.ImagePlaceholderInput{
			Input:  mcp.ParseString(req, "input", ""),
			Output: mcp.ParseString(req, "output", ""),
			Kind:   mcp.ParseString(req, "kind", ""),
			Inline: mcp.ParseBoolean(req, "inline", false),
		}
		if lw, ok := args["lqip_width"].(float64); ok {
			lwi := int(lw)
			input.LQIPWidth = &lwi
		}
		return mcp.NewToolResultText(app.srv.ImagePlaceholder(ctx, input)), nil
	})

	// image_palette
	s.AddTool(mcp.NewTool("image_palette",
		mcp.WithDescription("Reduce image to a limited color palette or extract dominant colors."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input image path")),
		mcp.WithString("output_dir", mcp.Required(), mcp.Description("Output directory")),
		mcp.WithNumber("max_colors", mcp.Description("Maximum colors (default 16)")),
		mcp.WithNumber("dithering", mcp.Description("Dithering amount 0.0-1.0")),
		mcp.WithString("format", mcp.Description("Output format: gif, png")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		input := tools.ImagePaletteInput{
			Input:     mcp.ParseString(req, "input", ""),
			OutputDir: mcp.ParseString(req, "output_dir", ""),
			Format:    mcp.ParseString(req, "format", ""),
		}
		if mc, ok := args["max_colors"].(float64); ok {
			mci := int(mc)
			input.MaxColors = &mci
		}
		if d, ok := args["dithering"].(float64); ok {
			dithering := float32(d)
			input.Dithering = &dithering
		}
		return mcp.NewToolResultText(app.srv.ImagePalette(ctx, input)), nil
	})

	// image_sprite
	s.AddTool(mcp.NewTool("image_sprite",
		mcp.WithDescription("Generate a sprite sheet from multiple images with optional CSS."),
		mcp.WithArray("inputs", mcp.Required(), mcp.Description("Input image paths"), mcp.WithStringItems()),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output sprite path")),
		mcp.WithNumber("cell_size", mcp.Description("Cell size (width=height)")),
		mcp.WithNumber("columns", mcp.Description("Number of columns")),
		mcp.WithNumber("padding", mcp.Description("Padding between sprites in pixels")),
		mcp.WithBoolean("generate_css", mcp.Description("Generate CSS for sprite positioning")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		input := tools.ImageSpriteInput{
			Output:      mcp.ParseString(req, "output", ""),
			GenerateCSS: mcp.ParseBoolean(req, "generate_css", false),
		}
		if inputsRaw, ok := args["inputs"]; ok {
			data, _ := json.Marshal(inputsRaw)
			json.Unmarshal(data, &input.Inputs)
		}
		if cs, ok := args["cell_size"].(float64); ok {
			csi := int(cs)
			input.CellSize = &csi
		}
		if col, ok := args["columns"].(float64); ok {
			coli := int(col)
			input.Columns = &coli
		}
		if p, ok := args["padding"].(float64); ok {
			pi := int(p)
			input.Padding = &pi
		}
		return mcp.NewToolResultText(app.srv.ImageSprite(ctx, input)), nil
	})

	// ── suggest_color_palettes ──────────────────────────────────
	s.AddTool(mcp.NewTool("suggest_color_palettes",
		mcp.WithDescription("Generate named color palette proposals for a given use case and mood."),
		mcp.WithString("use_case", mcp.Required(), mcp.Description("e.g. 'SaaS dashboard', 'marketing site'")),
		mcp.WithArray("brand_keywords", mcp.Description("Brand keyword list"), mcp.WithStringItems()),
		mcp.WithString("mood", mcp.Description("e.g. calm | bold | minimal | professional")),
		mcp.WithNumber("count", mcp.Description("Number of palettes to generate (default 3)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		input := tools.SuggestColorPalettesInput{
			UseCase: mcp.ParseString(req, "use_case", ""),
			Mood:    mcp.ParseString(req, "mood", ""),
			Count:   mcp.ParseInt(req, "count", 3),
		}
		if kwRaw, ok := args["brand_keywords"]; ok {
			data, _ := json.Marshal(kwRaw)
			json.Unmarshal(data, &input.BrandKeywords)
		}
		return mcp.NewToolResultText(app.srv.SuggestColorPalettes(ctx, input)), nil
	})

	// ── Video Tools ────────────────────────────────────────────────

	// video_transcode
	s.AddTool(mcp.NewTool("video_transcode",
		mcp.WithDescription("Transcode video to different codec (h264, h265, vp8, vp9, av1)."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input video path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output video path")),
		mcp.WithString("codec", mcp.Required(), mcp.Description("Target codec: h264, h265, vp8, vp9, av1")),
		mcp.WithString("bitrate", mcp.Description("Target bitrate (e.g., '2M', '5000k')")),
		mcp.WithString("preset", mcp.Description("Encoding preset: ultrafast, fast, medium, slow, veryslow")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		input := tools.VideoTranscodeInput{
			Input:   mcp.ParseString(req, "input", ""),
			Output:  mcp.ParseString(req, "output", ""),
			Codec:   mcp.ParseString(req, "codec", ""),
			Bitrate: mcp.ParseString(req, "bitrate", ""),
			Preset:  mcp.ParseString(req, "preset", ""),
		}
		return mcp.NewToolResultText(app.srv.VideoTranscode(ctx, input)), nil
	})

	// video_resize
	s.AddTool(mcp.NewTool("video_resize",
		mcp.WithDescription("Resize video while maintaining aspect ratio."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input video path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output video path")),
		mcp.WithNumber("width", mcp.Description("Target width")),
		mcp.WithNumber("height", mcp.Description("Target height")),
		mcp.WithBoolean("maintain_aspect", mcp.Description("Maintain aspect ratio (default true)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		input := tools.VideoResizeInput{
			Input:          mcp.ParseString(req, "input", ""),
			Output:         mcp.ParseString(req, "output", ""),
			Width:          uint32(mcp.ParseInt(req, "width", 0)),
			Height:         uint32(mcp.ParseInt(req, "height", 0)),
			MaintainAspect: mcp.ParseBoolean(req, "maintain_aspect", true),
		}
		return mcp.NewToolResultText(app.srv.VideoResize(ctx, input)), nil
	})

	// video_trim
	s.AddTool(mcp.NewTool("video_trim",
		mcp.WithDescription("Extract a time range from video."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input video path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output video path")),
		mcp.WithNumber("start", mcp.Required(), mcp.Description("Start time in seconds")),
		mcp.WithNumber("end", mcp.Required(), mcp.Description("End time in seconds")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		input := tools.VideoTrimInput{
			Input:  mcp.ParseString(req, "input", ""),
			Output: mcp.ParseString(req, "output", ""),
			Start:  numVal(args, "start", 0),
			End:    numVal(args, "end", 0),
		}
		return mcp.NewToolResultText(app.srv.VideoTrim(ctx, input)), nil
	})

	// video_thumbnail
	s.AddTool(mcp.NewTool("video_thumbnail",
		mcp.WithDescription("Extract a frame as image from video."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input video path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output image path")),
		mcp.WithString("timestamp", mcp.Required(), mcp.Description("Timestamp: '25%' or seconds like '30.5'")),
		mcp.WithString("format", mcp.Description("Output format: jpeg, png, webp (default jpeg)")),
		mcp.WithNumber("quality", mcp.Description("Image quality 1-100 (default 85)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		input := tools.VideoThumbnailInput{
			Input:     mcp.ParseString(req, "input", ""),
			Output:    mcp.ParseString(req, "output", ""),
			Timestamp: mcp.ParseString(req, "timestamp", ""),
			Format:    mcp.ParseString(req, "format", "jpeg"),
			Quality:   mcp.ParseInt(req, "quality", 85),
		}
		return mcp.NewToolResultText(app.srv.VideoThumbnail(ctx, input)), nil
	})

	// video_profile
	s.AddTool(mcp.NewTool("video_profile",
		mcp.WithDescription("Apply web-optimized encoding profile."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input video path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output video path")),
		mcp.WithString("profile", mcp.Required(), mcp.Description("Profile: web-low (480p/1M), web-mid (720p/2.5M), web-high (1080p/5M)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		input := tools.VideoProfileInput{
			Input:   mcp.ParseString(req, "input", ""),
			Output:  mcp.ParseString(req, "output", ""),
			Profile: mcp.ParseString(req, "profile", ""),
		}
		return mcp.NewToolResultText(app.srv.VideoProfile(ctx, input)), nil
	})

	// ── Audio Tools ────────────────────────────────────────────────

	// audio_transcode
	s.AddTool(mcp.NewTool("audio_transcode",
		mcp.WithDescription("Convert between audio formats (mp3, aac, opus, vorbis, flac, wav)."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input audio path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output audio path")),
		mcp.WithString("codec", mcp.Required(), mcp.Description("Target codec: mp3, aac, opus, vorbis, flac, wav")),
		mcp.WithString("bitrate", mcp.Description("Target bitrate (e.g., '192k')")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		input := tools.AudioTranscodeInput{
			Input:   mcp.ParseString(req, "input", ""),
			Output:  mcp.ParseString(req, "output", ""),
			Codec:   mcp.ParseString(req, "codec", ""),
			Bitrate: mcp.ParseString(req, "bitrate", ""),
		}
		return mcp.NewToolResultText(app.srv.AudioTranscode(ctx, input)), nil
	})

	// audio_trim
	s.AddTool(mcp.NewTool("audio_trim",
		mcp.WithDescription("Trim audio by timestamps."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input audio path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output audio path")),
		mcp.WithNumber("start", mcp.Required(), mcp.Description("Start time in seconds")),
		mcp.WithNumber("end", mcp.Required(), mcp.Description("End time in seconds")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		input := tools.AudioTrimInput{
			Input:  mcp.ParseString(req, "input", ""),
			Output: mcp.ParseString(req, "output", ""),
			Start:  numVal(args, "start", 0),
			End:    numVal(args, "end", 0),
		}
		return mcp.NewToolResultText(app.srv.AudioTrim(ctx, input)), nil
	})

	// audio_normalize
	s.AddTool(mcp.NewTool("audio_normalize",
		mcp.WithDescription("Normalize loudness to target LUFS."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input audio path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output audio path")),
		mcp.WithNumber("target_lufs", mcp.Required(), mcp.Description("Target LUFS (-14 for YouTube, -16 for Spotify, -23 for EBU R128)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		input := tools.AudioNormalizeInput{
			Input:      mcp.ParseString(req, "input", ""),
			Output:     mcp.ParseString(req, "output", ""),
			TargetLUFS: numVal(args, "target_lufs", -14),
		}
		return mcp.NewToolResultText(app.srv.AudioNormalize(ctx, input)), nil
	})

	// audio_silence_trim
	s.AddTool(mcp.NewTool("audio_silence_trim",
		mcp.WithDescription("Remove leading/trailing silence from audio."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input audio path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output audio path")),
		mcp.WithNumber("threshold_db", mcp.Description("Silence threshold in dB (default -40)")),
		mcp.WithNumber("min_duration", mcp.Description("Minimum silence duration in seconds (default 0.5)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		input := tools.AudioSilenceTrimInput{
			Input:       mcp.ParseString(req, "input", ""),
			Output:      mcp.ParseString(req, "output", ""),
			ThresholdDB: numVal(args, "threshold_db", -40),
			MinDuration: numVal(args, "min_duration", 0.5),
		}
		return mcp.NewToolResultText(app.srv.AudioSilenceTrim(ctx, input)), nil
	})
}

// argsMap safely extracts the arguments map from a CallToolRequest.
func argsMap(req mcp.CallToolRequest) map[string]interface{} {
	if m, ok := req.Params.Arguments.(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{}
}

// strVal extracts a string from a map[string]any.
func strVal(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// numVal extracts a number as float64 from a map[string]any.
func numVal(m map[string]interface{}, key string, fallback float64) float64 {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case float32:
			return float64(n)
		case int:
			return float64(n)
		case int64:
			return float64(n)
		}
	}
	return fallback
}
