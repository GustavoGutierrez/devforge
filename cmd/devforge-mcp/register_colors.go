// Package main — color tool registration for the DevForge MCP server.
package main

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"dev-forge-mcp/internal/tools/colors/conversion"
	"dev-forge-mcp/internal/tools/colors/gradient"
	"dev-forge-mcp/internal/tools/colors/harmony"
)

// registerColorTools registers color-related MCP tools.
func registerColorTools(s *mcpserver.MCPServer, _ *mcpApp) {
	// ── css_gradient_generate ───────────────────────────────────
	s.AddTool(mcp.NewTool("css_gradient_generate",
		mcp.WithDescription("Generate CSS linear or radial gradients with two or more color stops, including optional stop positions and browser-safe fallback color."),
		mcp.WithString("gradient_type", mcp.Required(), mcp.Description("Gradient type: linear | radial")),
		mcp.WithArray("stops", mcp.Required(), mcp.Description("Color stops array. Each item: { color: string, position?: number (0-100) }")),
		mcp.WithNumber("angle", mcp.Description("Linear gradient angle in degrees (default: 0)")),
		mcp.WithString("shape", mcp.Description("Radial gradient shape: circle | ellipse (default: circle)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)

		var stops []gradient.ColorStopInput
		if rawStops, ok := args["stops"]; ok {
			data, _ := json.Marshal(rawStops)
			_ = json.Unmarshal(data, &stops)
		}

		var angle *int
		if _, ok := args["angle"]; ok {
			v := int(numVal(args, "angle", 0))
			angle = &v
		}

		in := gradient.GenerateInput{
			GradientType: mcp.ParseString(req, "gradient_type", ""),
			Angle:        angle,
			Shape:        mcp.ParseString(req, "shape", ""),
			Stops:        stops,
		}
		return mcp.NewToolResultText(gradient.Generate(ctx, in)), nil
	})

	// ── color_code_convert ───────────────────────────────────────
	s.AddTool(mcp.NewTool("color_code_convert",
		mcp.WithDescription("Convert a color code between standards-based color spaces using a linear-sRGB/XYZ hub pipeline. Supports hex, rgb, linear_rgb, hsl, hsv, hwb, xyz, lab, lch, oklab, oklch."),
		mcp.WithString("color", mcp.Required(), mcp.Description("Input color code string. Examples: #3B82F6, rgb(59,130,246), lab(53.2,80.1,67.2), oklch(0.63,0.25,29.2)")),
		mcp.WithString("from", mcp.Required(), mcp.Description("Source space: hex | rgb | linear_rgb | hsl | hsv | hwb | xyz | lab | lch | oklab | oklch")),
		mcp.WithString("to", mcp.Required(), mcp.Description("Destination space: hex | rgb | linear_rgb | hsl | hsv | hwb | xyz | lab | lch | oklab | oklch")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		in := conversion.ConvertInput{
			Color: mcp.ParseString(req, "color", ""),
			From:  mcp.ParseString(req, "from", ""),
			To:    mcp.ParseString(req, "to", ""),
		}
		return mcp.NewToolResultText(conversion.Convert(ctx, in)), nil
	})

	// ── color_harmony_palette ────────────────────────────────────
	s.AddTool(mcp.NewTool("color_harmony_palette",
		mcp.WithDescription("Generate a 5-color harmony palette from a base HEX color using classic harmony rules (analogous, monochromatic, triad, complementary, split_complementary, square, compound, shades)."),
		mcp.WithString("base_color", mcp.Required(), mcp.Description("Base color in HEX format (#RRGGBB or #RGB).")),
		mcp.WithString("harmony", mcp.Required(), mcp.Description("Harmony type: analogous | monochromatic | triad | complementary | split_complementary | square | compound | shades")),
		mcp.WithNumber("spread", mcp.Description("Optional angular spread in degrees. If omitted, harmony-specific defaults are used.")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		var spread *float64
		if _, ok := args["spread"]; ok {
			v := numVal(args, "spread", 0)
			spread = &v
		}

		in := harmony.GenerateInput{
			BaseColor: mcp.ParseString(req, "base_color", ""),
			Harmony:   mcp.ParseString(req, "harmony", ""),
			Spread:    spread,
		}
		return mcp.NewToolResultText(harmony.Generate(ctx, in)), nil
	})
}
