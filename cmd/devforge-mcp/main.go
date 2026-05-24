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

	// 2. Initialize dpf Pool (DevPixelForge).
	// Pool size defaults to 2; override via DEVFORGE_DPF_POOL_SIZE (1–16).
	var streamer dpf.Streamer
	dpfPath, err := dpf.ResolveBinaryPath(exeDir)
	if err != nil {
		log.Printf("warning: dpf binary not available: %v", err)
		log.Printf("optimize_images, generate_favicon, markdown_to_pdf, and media tools will return errors")
	} else {
		pool, poolErr := dpf.NewPool(dpfPath)
		if poolErr != nil {
			log.Printf("warning: dpf pool init failed at %s: %v", dpfPath, poolErr)
			log.Printf("optimize_images, generate_favicon, markdown_to_pdf, and media tools will return errors")
		} else {
			log.Printf("dpf pool ready (size=%d) at %s", pool.Size(), dpfPath)
			streamer = pool
		}
	}

	// 3. Build app state
	app := &mcpApp{
		srv: &tools.Server{
			DPF: streamer,
		},
		geminiKey:  cfg.GeminiAPIKey,
		imageModel: cfg.ImageModel,
	}

	// 4. Build MCP server and register all tools
	s := mcpserver.NewMCPServer("devforge", "2.5.0",
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
	registerColorTools(s, app)
	registerFrontendTools(s, app)
	registerBackendTools(s, app)
	registerCodeTools(s, app)

	// ── generate_ui_image ────────────────────────────────────────
	s.AddTool(mcp.NewTool("generate_ui_image",
		mcp.WithDescription("Generate a UI image (wireframe, mockup, or illustration) from a text prompt via the Gemini image-generation API. Params: prompt:string (required), style:string (wireframe|mockup|illustration), width:int (pixels, default 1280), height:int (pixels, default 720), output_path:string (required, file path to save). Requires configure_gemini to be called first. Example: generate a login page mockup at 1280x720 saved to /tmp/login.png. Use generate_ui_image to create new AI-generated UI images; use ui2md to analyze an existing UI screenshot into a Markdown spec."),
		mcp.WithString("prompt", mcp.Required(), mcp.Description("Image generation prompt describing the UI to create")),
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
		mcp.WithDescription("Analyze an existing UI screenshot with Gemini Vision and write a structured Markdown design spec covering layout, colors, typography, and components. Params: image_path:string (required, path to PNG/JPEG/WebP/GIF), output_dir:string (optional, directory to save the .md file; defaults to same directory as the image). Requires configure_gemini to be called first. Example: analyze /designs/dashboard.png and save spec to /docs/. Use ui2md to reverse-engineer an existing image into a spec; use generate_ui_image to create a new image from a prompt."),
		mcp.WithString("image_path", mcp.Required(), mcp.Description("Path to the UI screenshot to analyze (PNG, JPEG, WebP, or GIF)")),
		mcp.WithString("output_dir", mcp.Description("Directory to save the generated Markdown spec (default: same directory as image_path)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		input := tools.UI2MDInput{
			ImagePath: mcp.ParseString(req, "image_path", ""),
			OutputDir: mcp.ParseString(req, "output_dir", ""),
		}
		return mcp.NewToolResultText(app.srv.UI2MD(ctx, input, app.getGeminiKey(), app.getImageModel())), nil
	})

	// ── markdown_to_pdf ─────────────────────────────────────────
	s.AddTool(mcp.NewTool("markdown_to_pdf",
		mcp.WithDescription("Convert Markdown content into a styled PDF document suitable for reports, PRPs, specs, invoices, and technical deliverables. Input: one of input:string (file path), markdown_text:string (inline UTF-8 content), or markdown_base64:string (base64-encoded UTF-8). Output: one of output:string (explicit PDF path), output_dir+file_name, or inline:bool=true (returns base64 PDF in tool response). Optional styling: page_size:string (a4|letter|legal), page_width_mm/page_height_mm:number (mm), layout_mode:string (paged|single_page), theme:string (invoice|scientific_article|professional|engineering|informational), theme_override:object (name, body_font_size_pt, code_font_size_pt, heading_scale, margin_mm). Example: convert /docs/report.md to /output/report.pdf using the professional theme."),
		mcp.WithString("input", mcp.Description("Source mode 1: Markdown file path")),
		mcp.WithString("markdown_text", mcp.Description("Source mode 2: inline UTF-8 Markdown content")),
		mcp.WithString("markdown_base64", mcp.Description("Source mode 3: base64-encoded UTF-8 Markdown content")),
		mcp.WithString("output", mcp.Description("Output mode 1: explicit PDF output path")),
		mcp.WithString("output_dir", mcp.Description("Output mode 2: output directory (optionally combine with file_name)")),
		mcp.WithString("file_name", mcp.Description("Optional filename when using output_dir (defaults to derived name)")),
		mcp.WithBoolean("inline", mcp.Description("Output mode 3: return PDF bytes as base64 in tool response")),
		mcp.WithString("page_size", mcp.Description("a4 | letter | legal")),
		mcp.WithNumber("page_width_mm", mcp.Description("Custom page width in millimeters")),
		mcp.WithNumber("page_height_mm", mcp.Description("Custom page height in millimeters")),
		mcp.WithString("layout_mode", mcp.Description("paged | single_page")),
		mcp.WithString("theme", mcp.Description("invoice | scientific_article | professional | engineering | informational")),
		mcp.WithObject("theme_config", mcp.Description("Raw theme overrides forwarded to dpf (advanced/compatibility path; use for full custom theme payloads)")),
		mcp.WithObject("theme_override", mcp.Description("Typed theme override fields: name, body_font_size_pt, code_font_size_pt, heading_scale, margin_mm")),
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
		if rawThemeOverride, ok := args["theme_override"]; ok {
			data, _ := json.Marshal(rawThemeOverride)
			var themeOverride tools.MarkdownThemeOverride
			if json.Unmarshal(data, &themeOverride) == nil {
				input.ThemeOverride = &themeOverride
			}
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
		mcp.WithDescription("Persist a Gemini API key to the DevForge config file and hot-reload it into the running server without a restart. Params: api_key:string (required, your Google Gemini API key). Must be called before generate_ui_image or ui2md will work. Example: configure_gemini with api_key='AIza...' to enable Gemini-powered tools immediately."),
		mcp.WithString("api_key", mcp.Required(), mcp.Description("Google Gemini API key to persist and activate")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		input := tools.ConfigureGeminiInput{
			APIKey: mcp.ParseString(req, "api_key", ""),
		}
		result := app.srv.ConfigureGemini(ctx, input, app.setGeminiKey)
		return mcp.NewToolResultText(result), nil
	})

	// ── optimize_images ─────────────────────────────────────────
	s.AddTool(mcp.NewTool("optimize_images",
		mcp.WithDescription("Batch-optimize multiple images in a single call, applying per-image constraints (max dimensions, format conversion, quality). Params: inputs:array (required, each object: path:string, max_width:int (pixels), max_height:int (pixels), formats:string[] (e.g. [\"webp\",\"png\"]), quality:int (1-100, default 85)), parallelism:int (max concurrent jobs, default 4). Example: optimize [logo.png, hero.jpg] to WebP at max 1200px wide with quality 80. Use optimize_images for batch processing; use image_quality when you need to hit an exact file-size target on a single image."),
		mcp.WithArray("inputs", mcp.Required(), mcp.Description("Array of per-image optimization requests, each with path, optional max_width/max_height (pixels), formats (e.g. [\"webp\"]), and quality (1-100)"),
			mcp.Items(map[string]any{"type": "object"})),
		mcp.WithNumber("parallelism", mcp.Description("Max parallel dpf operations (default 4)")),
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
		mcp.WithDescription("Generate a complete set of favicon variants (ico, png, svg) from a source image and return ready-to-use HTML <link> snippets. Params: source_path:string (required, path to PNG or SVG source), background_color:string (hex, default #ffffff), sizes:int[] (pixel sizes, default [16,32,48,180,192,512]), formats:string[] (ico|png|svg, default all three). Output files are saved to a favicons/ subdirectory next to the source. Example: generate favicons from /assets/logo.png producing ico, png, and svg variants at default sizes."),
		mcp.WithString("source_path", mcp.Required(), mcp.Description("Path to source image (PNG or SVG); output goes to a favicons/ folder beside it")),
		mcp.WithString("background_color", mcp.Description("Hex background color for transparent sources (default #ffffff)")),
		mcp.WithArray("sizes", mcp.Description("Icon pixel sizes to generate (default [16,32,48,180,192,512])"), mcp.WithNumberItems()),
		mcp.WithArray("formats", mcp.Description("Output formats to produce: ico | png | svg (default: all three)"), mcp.WithStringItems()),
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
		mcp.WithDescription("Crop a rectangular region from an image using pixel coordinates. Params: input:string (required), output:string (required), x:int (pixels, top-left X), y:int (pixels, top-left Y), width:int (pixels, crop width), height:int (pixels, crop height). Example: crop input.png from (100,50) to a 800x600 region into cropped.png."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input image path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output image path")),
		mcp.WithNumber("x", mcp.Required(), mcp.Description("X coordinate of the top-left corner of the crop region (pixels)")),
		mcp.WithNumber("y", mcp.Required(), mcp.Description("Y coordinate of the top-left corner of the crop region (pixels)")),
		mcp.WithNumber("width", mcp.Required(), mcp.Description("Width of the crop region in pixels")),
		mcp.WithNumber("height", mcp.Required(), mcp.Description("Height of the crop region in pixels")),
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
		mcp.WithDescription("Rotate an image by 90, 180, or 270 degrees and/or flip it horizontally or vertically. Params: input:string (required), output:string (required), angle:number (degrees: 90|180|270), flip_h:bool (mirror left-right), flip_v:bool (mirror top-bottom). Any combination of angle and flips can be applied in one call. Example: rotate input.png 90 degrees clockwise and save to rotated.png."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input image path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output image path")),
		mcp.WithNumber("angle", mcp.Description("Rotation angle in degrees: 90, 180, or 270")),
		mcp.WithBoolean("flip_h", mcp.Description("Mirror the image horizontally (left-right flip)")),
		mcp.WithBoolean("flip_v", mcp.Description("Mirror the image vertically (top-bottom flip)")),
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
		mcp.WithDescription("Overlay a text or image watermark onto an image at a specified position. Params: input:string (required), output:string (required), text:string (watermark text — required if image_path is absent), image_path:string (watermark image path — required if text is absent), position:string (center|tile|custom), x:int (pixels, for custom position), y:int (pixels, for custom position), opacity:number (0.0–1.0), size:int (font size in points for text watermarks), color:string (hex color for text). Example: add semi-transparent 'CONFIDENTIAL' text at center of photo.jpg with opacity 0.4."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input image path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output image path")),
		mcp.WithString("text", mcp.Description("Text to use as watermark (required when image_path is not provided)")),
		mcp.WithString("image_path", mcp.Description("Path to an image to use as watermark (required when text is not provided)")),
		mcp.WithString("position", mcp.Description("Watermark position: center | tile | custom")),
		mcp.WithNumber("x", mcp.Description("X offset in pixels for custom position")),
		mcp.WithNumber("y", mcp.Description("Y offset in pixels for custom position")),
		mcp.WithNumber("opacity", mcp.Description("Watermark opacity: 0.0 (invisible) to 1.0 (opaque)")),
		mcp.WithNumber("size", mcp.Description("Font size in points for text watermarks")),
		mcp.WithString("color", mcp.Description("Text color as hex string (e.g. #ffffff)")),
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
		mcp.WithDescription("Apply tonal and focus adjustments to an image in a single pass. Params: input:string (required), output:string (required), brightness:number (-100 to 100), contrast:number (-100 to 100), saturation:number (-100 to 100), blur:number (radius in pixels), sharpen:number (amount, higher = sharper). All adjustment params are optional; only the non-zero ones are applied. Example: increase brightness by 20 and sharpen by 1.5 on photo.jpg."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input image path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output image path")),
		mcp.WithNumber("brightness", mcp.Description("Brightness adjustment: -100 (darkest) to 100 (brightest)")),
		mcp.WithNumber("contrast", mcp.Description("Contrast adjustment: -100 to 100")),
		mcp.WithNumber("saturation", mcp.Description("Saturation adjustment: -100 (grayscale) to 100 (vivid)")),
		mcp.WithNumber("blur", mcp.Description("Gaussian blur radius in pixels")),
		mcp.WithNumber("sharpen", mcp.Description("Sharpening amount (higher values increase edge contrast)")),
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
		mcp.WithDescription("Re-encode an image to meet a target file size using binary search over quality levels. Params: input:string (required), output:string (required), target_size_kb:int (required, desired output size in KB), format:string (webp|jpeg|png), max_quality:int (1-100, upper quality bound), min_quality:int (1-100, lower quality bound). Example: compress hero.png to under 150 KB in WebP format. Use image_quality to hit an exact file-size target on a single image; use optimize_images to batch-process many images at once."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input image path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output image path")),
		mcp.WithNumber("target_size_kb", mcp.Required(), mcp.Description("Desired output file size in kilobytes")),
		mcp.WithString("format", mcp.Description("Output format: webp | jpeg | png")),
		mcp.WithNumber("max_quality", mcp.Description("Maximum quality bound (1-100); default 95")),
		mcp.WithNumber("min_quality", mcp.Description("Minimum quality bound (1-100); default 10")),
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
		mcp.WithDescription("Generate multiple width-based image variants ready for use in an HTML srcset attribute. Params: input:string (required), output_dir:string (required, directory where variants are saved), widths:int[] (pixel widths to generate, e.g. [320,640,960,1280]), sizes:string[] (optional HTML sizes attribute values, e.g. ['100vw','(min-width:768px) 50vw']), format:string (webp|jpeg|png). Returns a list of variants with paths, widths, and file sizes. Example: generate srcset variants of hero.jpg at 320, 640, 960, and 1280px wide in WebP format."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input image path")),
		mcp.WithString("output_dir", mcp.Required(), mcp.Description("Directory where width variants will be saved")),
		mcp.WithArray("widths", mcp.Description("Target pixel widths to generate (e.g. [320, 640, 960, 1280])"), mcp.WithNumberItems()),
		mcp.WithArray("sizes", mcp.Description("HTML sizes attribute values to annotate in the response (e.g. ['100vw'])"), mcp.WithStringItems()),
		mcp.WithString("format", mcp.Description("Output format: webp | jpeg | png")),
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
		mcp.WithDescription("Read or modify EXIF metadata on an image. Params: input:string (required), exif_op:string (required: strip|preserve|extract|auto_orient), output:string (required for strip, preserve, and auto_orient; optional for extract). 'extract' returns the metadata map in the response. 'strip' removes all metadata and writes to output. 'auto_orient' rotates the image to match the EXIF orientation tag then strips the tag. Example: strip EXIF from photo.jpg to sanitized.jpg, or extract metadata from an image without writing output."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input image path")),
		mcp.WithString("output", mcp.Description("Output image path (required for strip, preserve, and auto_orient operations)")),
		mcp.WithString("exif_op", mcp.Required(), mcp.Description("EXIF operation: strip | preserve | extract | auto_orient")),
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
		mcp.WithDescription("Resize an image to one or more target widths or by a percentage scale factor. Params: input:string (required), output_dir:string (required for width-based resize; defaults to same directory when using scale_percent), widths:int[] (pixel widths to generate), scale_percent:number (e.g. 50.0 = half size), max_height:int (pixels, optional height cap), format:string (webp|jpeg|png|avif), quality:int (1-100), filter:string (lanczos3|gaussian|bilinear), linear_rgb:bool (use linear color space for better downscale quality). Either widths or scale_percent must be provided. Example: resize product.jpg to 400 and 800px wide in WebP at quality 85. Use image_resize for bulk or percentage-based resizing; use image_srcset when you specifically need srcset-ready HTML attributes returned."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input image path")),
		mcp.WithString("output_dir", mcp.Description("Output directory for resized variants (required for width-based resize; defaults to input directory when using scale_percent)")),
		mcp.WithArray("widths", mcp.Description("Target pixel widths to generate (e.g. [400, 800, 1200])"), mcp.WithNumberItems()),
		mcp.WithNumber("scale_percent", mcp.Description("Scale by percentage: 50.0 = half size, 200.0 = double size")),
		mcp.WithNumber("max_height", mcp.Description("Maximum output height in pixels (aspect-ratio preserved)")),
		mcp.WithString("format", mcp.Description("Output format: webp | jpeg | png | avif")),
		mcp.WithNumber("quality", mcp.Description("Encoding quality: 1 (lowest) to 100 (highest)")),
		mcp.WithString("filter", mcp.Description("Resampling filter: lanczos3 (best quality) | gaussian | bilinear (fastest)")),
		mcp.WithBoolean("linear_rgb", mcp.Description("Process in linear RGB color space for perceptually better downscale results")),
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
		mcp.WithDescription("Convert an image to a different file format, optionally resizing in the same call. Params: input:string (required), output:string (required), format:string (required: webp|jpeg|png|avif|gif), quality:int (1-100), width:int (pixels, optional resize), height:int (pixels, optional resize). Example: convert logo.png to logo.webp at quality 90 without resizing."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input image path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output image path")),
		mcp.WithString("format", mcp.Required(), mcp.Description("Target format: webp | jpeg | png | avif | gif")),
		mcp.WithNumber("quality", mcp.Description("Encoding quality: 1 (lowest) to 100 (highest)")),
		mcp.WithNumber("width", mcp.Description("Optional target width in pixels (aspect ratio preserved if only one dimension is set)")),
		mcp.WithNumber("height", mcp.Description("Optional target height in pixels (aspect ratio preserved if only one dimension is set)")),
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
		mcp.WithDescription("Generate a lightweight image placeholder for use during lazy-loading: a low-quality preview image (LQIP), a dominant-color hex string, or a CSS gradient string. Params: input:string (required), kind:string (lqip|dominant_color|css_gradient), output:string (optional file path to save the placeholder), lqip_width:int (pixels, width of the tiny preview image), inline:bool (return placeholder as base64 in tool response instead of writing to file). Example: generate a 20px-wide LQIP of hero.jpg returned as inline base64. Use image_placeholder for lazy-load placeholders derived from an existing image; use generate_ui_image to generate a brand-new AI image."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input image path to derive the placeholder from")),
		mcp.WithString("output", mcp.Description("Optional file path to save the placeholder (not needed when inline=true)")),
		mcp.WithString("kind", mcp.Description("Placeholder type: lqip (tiny blurred preview) | dominant_color (hex string) | css_gradient")),
		mcp.WithNumber("lqip_width", mcp.Description("Width of the LQIP preview in pixels (default 20)")),
		mcp.WithBoolean("inline", mcp.Description("Return the placeholder as base64 data in the tool response")),
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
		mcp.WithDescription("Quantize an image to a limited color palette (for GIF/indexed-PNG output) and extract the dominant color list as hex values. Params: input:string (required), output_dir:string (required), max_colors:int (palette size, default 16), dithering:number (0.0–1.0, controls dither noise vs. banding), format:string (gif|png). Returns the quantized image and a colors array. Example: reduce banner.png to a 32-color GIF with 0.5 dithering. Use image_palette to quantize or extract dominant colors from an existing image; use image_placeholder to generate a CSS gradient or dominant-color placeholder for lazy loading."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input image path")),
		mcp.WithString("output_dir", mcp.Required(), mcp.Description("Directory where the palette-quantized output file will be saved")),
		mcp.WithNumber("max_colors", mcp.Description("Maximum number of colors in the palette (default 16)")),
		mcp.WithNumber("dithering", mcp.Description("Dithering intensity: 0.0 (none, more banding) to 1.0 (maximum, less banding)")),
		mcp.WithString("format", mcp.Description("Output format: gif | png")),
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
		mcp.WithDescription("Pack multiple images into a single sprite sheet PNG and optionally generate companion CSS with background-position rules for each sprite. Params: inputs:string[] (required, ordered list of image paths), output:string (required, sprite sheet output path), cell_size:int (pixels, forces all cells to this square size), columns:int (grid columns; auto-calculated if omitted), padding:int (pixels between sprites, default 0), generate_css:bool (write a .css file alongside the sprite). Example: pack icons/[a,b,c].png into sprites.png with 4 columns and 2px padding, generating CSS."),
		mcp.WithArray("inputs", mcp.Required(), mcp.Description("Ordered list of input image paths to pack into the sprite sheet"), mcp.WithStringItems()),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output path for the generated sprite sheet PNG")),
		mcp.WithNumber("cell_size", mcp.Description("Force all sprite cells to this square pixel size (width = height)")),
		mcp.WithNumber("columns", mcp.Description("Number of columns in the sprite grid (auto-calculated if omitted)")),
		mcp.WithNumber("padding", mcp.Description("Padding in pixels between sprites (default 0)")),
		mcp.WithBoolean("generate_css", mcp.Description("Also write a CSS file with background-position rules for each sprite")),
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

	// ── Video Tools ────────────────────────────────────────────────

	// video_transcode
	s.AddTool(mcp.NewTool("video_transcode",
		mcp.WithDescription("Re-encode a video file to a different codec with control over bitrate and encoder speed. Params: input:string (required), output:string (required), codec:string (required: h264|h265|vp8|vp9|av1), bitrate:string (e.g. '2M' or '5000k'), preset:string (ultrafast|fast|medium|slow|veryslow — trades speed for compression efficiency). Example: transcode raw.mov to output.mp4 using h264 at 4M bitrate with the fast preset. Use video_transcode for full re-encoding with codec choice; use video_profile for opinionated web-optimized presets without specifying codec details."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input video path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output video path")),
		mcp.WithString("codec", mcp.Required(), mcp.Description("Target video codec: h264 | h265 | vp8 | vp9 | av1")),
		mcp.WithString("bitrate", mcp.Description("Target bitrate (e.g. '2M' for 2 Mbps, '5000k' for 5000 kbps)")),
		mcp.WithString("preset", mcp.Description("Encoder speed/quality preset: ultrafast | fast | medium | slow | veryslow")),
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
		mcp.WithDescription("Scale a video to different dimensions, optionally preserving the original aspect ratio. Params: input:string (required), output:string (required), width:int (pixels, target width), height:int (pixels, target height), maintain_aspect:bool (default true; set false to force exact dimensions and allow distortion). At least one of width or height is required. Example: resize lecture.mp4 to 1280px wide while keeping aspect ratio. Use video_resize to change pixel dimensions; use video_profile to apply a complete web-optimized preset (resolution + bitrate in one step)."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input video path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output video path")),
		mcp.WithNumber("width", mcp.Description("Target width in pixels (required if height is not set)")),
		mcp.WithNumber("height", mcp.Description("Target height in pixels (required if width is not set)")),
		mcp.WithBoolean("maintain_aspect", mcp.Description("Preserve aspect ratio (default true); set false to force exact width × height")),
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
		mcp.WithDescription("Extract a contiguous time segment from a video by specifying start and end times in seconds. Params: input:string (required), output:string (required), start:number (required, seconds from beginning, must be ≥ 0), end:number (required, seconds, must be > start). Example: trim intro.mp4 from 5.0 to 30.0 seconds and save to clip.mp4. Use video_trim to cut a clip by time range; use audio_trim to cut audio-only files."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input video path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output video path")),
		mcp.WithNumber("start", mcp.Required(), mcp.Description("Start time in seconds (must be ≥ 0)")),
		mcp.WithNumber("end", mcp.Required(), mcp.Description("End time in seconds (must be greater than start)")),
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
		mcp.WithDescription("Extract a single frame from a video and save it as an image. Params: input:string (required), output:string (required, image file path), timestamp:string (required: percentage like '25%' or seconds like '30.5'), format:string (jpeg|png|webp, default jpeg), quality:int (1-100, default 85). Example: extract the frame at 5 seconds from interview.mp4 and save as thumb.jpg. Use video_thumbnail to grab one frame; use video_trim to extract a time segment as a new video."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input video path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output image file path")),
		mcp.WithString("timestamp", mcp.Required(), mcp.Description("Frame position: percentage string like '25%' or elapsed seconds like '30.5'")),
		mcp.WithString("format", mcp.Description("Output image format: jpeg | png | webp (default jpeg)")),
		mcp.WithNumber("quality", mcp.Description("Image encoding quality: 1 (lowest) to 100 (highest), default 85")),
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
		mcp.WithDescription("Encode a video using a named web-delivery profile that bundles resolution and bitrate settings. Params: input:string (required), output:string (required), profile:string (required: web-low = 480p/1 Mbps | web-mid = 720p/2.5 Mbps | web-high = 1080p/5 Mbps). Example: produce a web-ready 720p copy of raw-footage.mp4 using web-mid. Use video_profile when you want sensible web defaults with a single parameter; use video_transcode when you need explicit codec, bitrate, or preset control."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input video path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output video path")),
		mcp.WithString("profile", mcp.Required(), mcp.Description("Web delivery profile: web-low (480p, 1 Mbps) | web-mid (720p, 2.5 Mbps) | web-high (1080p, 5 Mbps)")),
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
		mcp.WithDescription("Re-encode an audio file to a different format or codec. Params: input:string (required), output:string (required), codec:string (required: mp3|aac|opus|vorbis|flac|wav), bitrate:string (e.g. '192k' or '320k'). Example: convert podcast.flac to podcast.mp3 at 192k bitrate. Use audio_transcode to change codec or bitrate; use audio_trim to extract a time segment without re-encoding."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input audio path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output audio path")),
		mcp.WithString("codec", mcp.Required(), mcp.Description("Target audio codec: mp3 | aac | opus | vorbis | flac | wav")),
		mcp.WithString("bitrate", mcp.Description("Target bitrate (e.g. '192k' for 192 kbps, '320k' for 320 kbps)")),
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
		mcp.WithDescription("Extract a contiguous segment from an audio file by specifying start and end positions in seconds. Params: input:string (required), output:string (required), start:number (required, seconds from beginning, must be ≥ 0), end:number (required, seconds, must be > start). Example: extract the segment from 5.0 to 30.0 seconds of interview.mp3 into clip.mp3. Use audio_trim to cut audio by time range; use audio_silence_trim to automatically remove quiet sections at the edges."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input audio path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output audio path")),
		mcp.WithNumber("start", mcp.Required(), mcp.Description("Start position in seconds (must be ≥ 0)")),
		mcp.WithNumber("end", mcp.Required(), mcp.Description("End position in seconds (must be greater than start)")),
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
		mcp.WithDescription("Normalize an audio file's integrated loudness to a target LUFS level (ITU-R BS.1770). Params: input:string (required), output:string (required), target_lufs:number (required, negative float: -14 for YouTube, -16 for Spotify, -23 for broadcast/EBU R128). Example: normalize podcast.mp3 to -16 LUFS for Spotify distribution. Use audio_normalize to adjust perceived loudness; use audio_silence_trim to remove quiet padding from the edges."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input audio path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output audio path")),
		mcp.WithNumber("target_lufs", mcp.Required(), mcp.Description("Target integrated loudness in LUFS: -14 (YouTube), -16 (Spotify), -23 (EBU R128 broadcast)")),
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
		mcp.WithDescription("Automatically remove leading and trailing silence from an audio file based on a dB threshold. Params: input:string (required), output:string (required), threshold_db:number (dB level below which audio is considered silent, default -40), min_duration:number (minimum silent segment length in seconds to cut, default 0.5). Example: trim dead air from recording.wav using a -35 dB threshold and 0.3s minimum silence. Use audio_silence_trim to auto-remove edge silence by level detection; use audio_trim to cut specific seconds manually."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input audio path")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output audio path")),
		mcp.WithNumber("threshold_db", mcp.Description("dB level below which audio counts as silence (default -40; e.g. -35 for noisier recordings)")),
		mcp.WithNumber("min_duration", mcp.Description("Minimum contiguous silence duration in seconds to remove (default 0.5)")),
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
