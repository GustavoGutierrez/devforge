package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"

	"dev-forge-mcp/internal/tools"
	"dev-forge-mcp/internal/tools/colors/conversion"
	"dev-forge-mcp/internal/tools/colors/gradient"
	"dev-forge-mcp/internal/tools/colors/harmony"
	"dev-forge-mcp/internal/tools/frontend"
	"dev-forge-mcp/internal/tools/frontend/micro"
	frontendui "dev-forge-mcp/internal/tools/frontend/ui"
)

var frontendOps = []string{
	"Color Convert / WCAG Contrast",
	"CSS Unit Converter",
	"Breakpoint Lookup",
	"Regex Tester",
	"Locale Format",
	"ICU Message Format",
	"Color Harmony Palette",
	"Color Code Conversion",
	"CSS Gradient Generator",
	"SVG Optimizer",
	"Image Base64 Encoder",
	"Text Diff Checker",
	"Batch CSS Unit Converter",
	"WCAG Contrast Checker",
	"Aspect Ratio Calculator",
	"Batch String Case Converter",
}

type frontendToolsModel struct {
	phase    int
	opIdx    int
	fieldIdx int
	fields   []fieldDef
	values   map[string]string
	inputBuf string
	result   string
	isError  bool
	goHome   bool
	srv      *tools.Server
}

func newFrontendToolsModel(srv *tools.Server) frontendToolsModel {
	return frontendToolsModel{srv: srv, values: make(map[string]string)}
}

func (m frontendToolsModel) Init() tea.Cmd { return nil }

func (m frontendToolsModel) Update(msg tea.Msg) (frontendToolsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.phase {
		case 0:
			return m.updatePhase0(msg)
		case 1:
			return m.updatePhase1(msg)
		case 2:
			return m.updatePhase2(msg)
		}
	}
	return m, nil
}

func (m frontendToolsModel) updatePhase0(msg tea.KeyMsg) (frontendToolsModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.opIdx > 0 {
			m.opIdx--
		}
	case "down", "j":
		if m.opIdx < len(frontendOps)-1 {
			m.opIdx++
		}
	case "enter", " ":
		m.fields = frontendFields(m.opIdx)
		m.fieldIdx = 0
		m.inputBuf = ""
		m.values = make(map[string]string)
		m.phase = 1
	case "esc", "q":
		m.goHome = true
	}
	return m, nil
}

func (m frontendToolsModel) updatePhase1(msg tea.KeyMsg) (frontendToolsModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.fieldIdx == 0 {
			m.phase = 0
			m.inputBuf = ""
		} else {
			m.fieldIdx--
			m.inputBuf = m.values[m.fields[m.fieldIdx].key]
		}
	case "enter":
		m.values[m.fields[m.fieldIdx].key] = m.inputBuf
		if m.fieldIdx == len(m.fields)-1 {
			res := m.execute()
			m.result = res
			m.isError = strings.Contains(res, `"error"`)
			m.phase = 2
			return m, nil
		}
		m.fieldIdx++
		m.inputBuf = m.values[m.fields[m.fieldIdx].key]
	case "backspace", "ctrl+h":
		if len(m.inputBuf) > 0 {
			runes := []rune(m.inputBuf)
			m.inputBuf = string(runes[:len(runes)-1])
		}
	default:
		for _, r := range msg.Runes {
			if unicode.IsPrint(r) {
				m.inputBuf += string(r)
			}
		}
	}
	return m, nil
}

func (m frontendToolsModel) updatePhase2(msg tea.KeyMsg) (frontendToolsModel, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc", "q":
		m.phase = 0
		m.inputBuf = ""
		m.values = make(map[string]string)
	}
	return m, nil
}

func (m frontendToolsModel) execute() string {
	ctx := context.Background()
	switch m.opIdx {
	case 0: // Color Convert / WCAG Contrast
		alpha := 1.0
		if v := m.values["alpha"]; v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				alpha = f
			}
		}
		return frontend.Color(ctx, frontend.ColorInput{
			Color:   m.values["color"],
			To:      m.values["to"],
			Alpha:   alpha,
			Against: m.values["against"],
		})
	case 1: // CSS Unit Converter
		var value float64
		if v := m.values["value"]; v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				value = f
			}
		}
		return frontend.CSSUnit(ctx, frontend.CSSUnitInput{
			Value: value,
			From:  m.values["from"],
			To:    m.values["to"],
		})
	case 2: // Breakpoint Lookup
		var width int
		if v := m.values["width"]; v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				width = n
			}
		}
		return frontend.Breakpoint(ctx, frontend.BreakpointInput{
			Width:  width,
			System: m.values["system"],
		})
	case 3: // Regex Tester
		return frontend.Regex(ctx, frontend.RegexInput{
			Pattern:     m.values["pattern"],
			Input:       m.values["input"],
			Flags:       m.values["flags"],
			Operation:   m.values["operation"],
			Replacement: m.values["replacement"],
		})
	case 4: // Locale Format
		return frontend.LocaleFormat(ctx, frontend.LocaleFormatInput{
			Value:    m.values["value"],
			Kind:     m.values["kind"],
			Locale:   m.values["locale"],
			Currency: m.values["currency"],
		})
	case 5: // ICU Message Format
		return frontend.ICUFormat(ctx, frontend.ICUFormatInput{
			Template: m.values["template"],
			Locale:   m.values["locale"],
		})
	case 6: // Color Harmony Palette
		var spread *float64
		if v := strings.TrimSpace(m.values["spread"]); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				spread = &f
			}
		}
		return harmony.Generate(ctx, harmony.GenerateInput{
			BaseColor: m.values["base_color"],
			Harmony:   m.values["harmony"],
			Spread:    spread,
		})
	case 7: // Color Code Conversion
		return conversion.Convert(ctx, conversion.ConvertInput{
			Color: m.values["color_code"],
			From:  m.values["from_space"],
			To:    m.values["to_space"],
		})
	case 8: // CSS Gradient Generator
		stops, err := parseGradientStopsInput(m.values["stops"])
		if err != nil {
			return `{"error":"` + err.Error() + `"}`
		}

		var angle *int
		if v := strings.TrimSpace(m.values["angle"]); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				angle = &n
			}
		}

		return gradient.Generate(ctx, gradient.GenerateInput{
			GradientType: m.values["gradient_type"],
			Angle:        angle,
			Shape:        m.values["shape"],
			Stops:        stops,
		})
	case 9: // SVG Optimizer
		return frontendui.SVGOptimize(ctx, frontendui.SVGOptimizeInput{
			SVG: m.values["svg"],
		})
	case 10: // Image Base64 Encoder
		includeURI := true
		if strings.EqualFold(strings.TrimSpace(m.values["data_uri"]), "false") {
			includeURI = false
		}
		return frontendui.ImageBase64(ctx, frontendui.ImageBase64Input{
			Path:     m.values["path"],
			DataURI:  includeURI,
			MimeType: m.values["mime_type"],
		})
	case 11: // Text Diff Checker
		return micro.GenerateTextDiff(ctx, micro.TextDiffInput{
			OriginalText: m.values["original_text"],
			ModifiedText: m.values["modified_text"],
		})
	case 12: // Batch CSS Unit Converter
		vals, err := parseFloatList(m.values["values_px"])
		if err != nil {
			return `{"error":"` + err.Error() + `"}`
		}
		base := 16.0
		if v := strings.TrimSpace(m.values["base_size"]); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				base = f
			}
		}
		return micro.ConvertCSSUnits(ctx, micro.CSSUnitsBatchInput{
			ValuesPX:   vals,
			BaseSize:   base,
			TargetUnit: m.values["target_unit"],
		})
	case 13: // WCAG Contrast Checker
		return micro.CheckWCAGContrast(ctx, micro.WCAGContrastInput{
			ForegroundColor: m.values["foreground_color"],
			BackgroundColor: m.values["background_color"],
		})
	case 14: // Aspect Ratio Calculator
		var width *float64
		var height *float64
		if v := strings.TrimSpace(m.values["known_width"]); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				width = &f
			}
		}
		if v := strings.TrimSpace(m.values["known_height"]); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				height = &f
			}
		}
		return micro.CalculateAspectRatio(ctx, micro.AspectRatioInput{
			AspectRatio: m.values["aspect_ratio"],
			KnownWidth:  width,
			KnownHeight: height,
		})
	case 15: // Batch String Case Converter
		vars := parseStringList(m.values["variables"])
		return micro.ConvertStringCases(ctx, micro.StringCasesInput{
			Variables:  vars,
			TargetCase: m.values["target_case"],
		})
	}
	return `{"error":"unknown operation"}`
}

func frontendFields(opIdx int) []fieldDef {
	switch opIdx {
	case 0:
		return []fieldDef{
			{"color", "Color", true, "#RRGGBB, rgb(r,g,b), or hsl(h,s%,l%)", false},
			{"to", "Convert To", false, "hex | rgb | rgba | hsl | hsla (default: hex)", false},
			{"alpha", "Alpha", false, "0.0–1.0 (default: 1.0)", false},
			{"against", "Against (for contrast)", false, "second color for WCAG contrast ratio (optional)", false},
		}
	case 1:
		return []fieldDef{
			{"value", "Value", true, "numeric value to convert", false},
			{"from", "From Unit", true, "px | rem | em | percent | vw | vh", false},
			{"to", "To Unit", true, "px | rem | em | percent | vw | vh", false},
		}
	case 2:
		return []fieldDef{
			{"width", "Viewport Width (px)", true, "e.g. 1024", false},
			{"system", "System", false, "tailwind | bootstrap | custom (default: tailwind)", false},
		}
	case 3:
		return []fieldDef{
			{"pattern", "Regex Pattern", true, "Go regexp pattern", false},
			{"input", "Input String", true, "string to test against", false},
			{"flags", "Flags", false, "i (case-insensitive), m (multiline), g (global)", false},
			{"operation", "Operation", false, "test | match | replace (default: test)", false},
			{"replacement", "Replacement", false, "replacement string (for replace operation)", false},
		}
	case 4:
		return []fieldDef{
			{"value", "Value", true, "number or date string", false},
			{"kind", "Kind", true, "number | currency | date | time | datetime | percent", false},
			{"locale", "Locale", false, "IETF locale (default: en-US)", false},
			{"currency", "Currency", false, "ISO 4217 code (required for currency kind)", false},
		}
	case 5:
		return []fieldDef{
			{"template", "ICU Template", true, "e.g. Hello {name}!", false},
			{"locale", "Locale", false, "IETF locale (default: en)", false},
		}
	case 6:
		return []fieldDef{
			{"base_color", "Base Color", true, "HEX color (#RRGGBB or #RGB)", false},
			{"harmony", "Harmony", true, "analogous | monochromatic | triad | complementary | split_complementary | square | compound | shades", false},
			{"spread", "Spread (optional)", false, "custom angle in degrees (blank = harmony default)", false},
		}
	case 7:
		return []fieldDef{
			{"color_code", "Color Code", true, "Input color value (example: #3B82F6, rgb(59,130,246), lab(53.2,80.1,67.2))", false},
			{"from_space", "From Space", true, "hex | rgb | linear_rgb | hsl | hsv | hwb | xyz | lab | lch | oklab | oklch", false},
			{"to_space", "To Space", true, "hex | rgb | linear_rgb | hsl | hsv | hwb | xyz | lab | lch | oklab | oklch", false},
		}
	case 8:
		return []fieldDef{
			{"gradient_type", "Gradient Type", true, "linear | radial", false},
			{"stops", "Color Stops", true, "Comma-separated stops. Use color@position, e.g. #22c1c3@0,#fdbb2d@100 or #22c1c3,#fdbb2d", false},
			{"angle", "Angle (linear only)", false, "Degrees, default 0", false},
			{"shape", "Shape (radial only)", false, "circle | ellipse (default circle)", false},
		}
	case 9:
		return []fieldDef{
			{"svg", "SVG", true, "Raw SVG markup to optimize", false},
		}
	case 10:
		return []fieldDef{
			{"path", "Image Path", true, "Local image file path", false},
			{"data_uri", "Include Data URI", false, "true | false (default: true)", false},
			{"mime_type", "MIME Type", false, "Optional override (e.g. image/png)", false},
		}
	case 11:
		return []fieldDef{
			{"original_text", "Original Text", true, "Base/original text", false},
			{"modified_text", "Modified Text", true, "Updated/modified text", false},
		}
	case 12:
		return []fieldDef{
			{"values_px", "Values PX", true, "Comma-separated numbers, e.g. 12,16,24", false},
			{"base_size", "Base Size", false, "Base font size (default: 16)", false},
			{"target_unit", "Target Unit", true, "rem | em", false},
		}
	case 13:
		return []fieldDef{
			{"foreground_color", "Foreground Color", true, "#hex, rgb(), or hsl()", false},
			{"background_color", "Background Color", true, "#hex, rgb(), or hsl()", false},
		}
	case 14:
		return []fieldDef{
			{"aspect_ratio", "Aspect Ratio", false, "W:H format, e.g. 16:9", false},
			{"known_width", "Known Width", false, "Known width value", false},
			{"known_height", "Known Height", false, "Known height value", false},
		}
	case 15:
		return []fieldDef{
			{"variables", "Variables", true, "Comma-separated variable names", false},
			{"target_case", "Target Case", true, "camelCase | snake_case | kebab-case | PascalCase", false},
		}
	}
	return nil
}

func parseFloatList(raw string) ([]float64, error) {
	parts := parseStringList(raw)
	out := make([]float64, 0, len(parts))
	for _, p := range parts {
		f, err := strconv.ParseFloat(p, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid numeric value %q", p)
		}
		out = append(out, f)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("at least one numeric value is required")
	}
	return out, nil
}

func parseStringList(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func parseGradientStopsInput(raw string) ([]gradient.ColorStopInput, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("stops is required")
	}

	parts := strings.Split(raw, ",")
	stops := make([]gradient.ColorStopInput, 0, len(parts))

	for _, part := range parts {
		token := strings.TrimSpace(part)
		if token == "" {
			continue
		}

		segments := strings.SplitN(token, "@", 2)
		color := strings.TrimSpace(segments[0])
		if color == "" {
			return nil, fmt.Errorf("invalid stop %q", token)
		}

		stop := gradient.ColorStopInput{Color: color}
		if len(segments) == 2 {
			pStr := strings.TrimSpace(segments[1])
			p, err := strconv.Atoi(pStr)
			if err != nil {
				return nil, fmt.Errorf("invalid stop position %q", pStr)
			}
			stop.Position = &p
		}

		stops = append(stops, stop)
	}

	if len(stops) < 2 {
		return nil, fmt.Errorf("at least 2 stops are required")
	}

	return stops, nil
}

func (m frontendToolsModel) View() string {
	switch m.phase {
	case 0:
		return m.viewPhase0()
	case 1:
		return m.viewPhase1()
	case 2:
		return m.viewPhase2()
	}
	return ""
}

func (m frontendToolsModel) viewPhase0() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("◆ Frontend Utilities") + "\n\n")
	b.WriteString(normalStyle.Render("Select operation:") + "\n")
	for i, op := range frontendOps {
		if i == m.opIdx {
			b.WriteString(selectedStyle.Render(fmt.Sprintf("> %s", op)) + "\n")
		} else {
			b.WriteString(normalStyle.Render(fmt.Sprintf("  %s", op)) + "\n")
		}
	}
	b.WriteString("\n" + helpStyle.Render("↑/↓ navigate • Enter select • Esc back"))
	return b.String()
}

func (m frontendToolsModel) viewPhase1() string {
	if m.fieldIdx >= len(m.fields) {
		return ""
	}
	field := m.fields[m.fieldIdx]
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("◆ %s (%d/%d)", frontendOps[m.opIdx], m.fieldIdx+1, len(m.fields))) + "\n\n")
	b.WriteString(normalStyle.Render(field.label) + "\n")
	if field.hint != "" {
		b.WriteString(dimStyle.Render(field.hint) + "\n")
	}
	b.WriteString("\n")
	b.WriteString(boxStyle.Render(m.inputBuf+"_") + "\n")
	b.WriteString("\n" + helpStyle.Render("Esc back • Enter confirm"))
	return b.String()
}

func (m frontendToolsModel) viewPhase2() string {
	var b strings.Builder
	if m.isError {
		b.WriteString(errorStyle.Render("✗ Error") + "\n\n")
		b.WriteString(errorStyle.Render(m.result) + "\n")
	} else {
		b.WriteString(successStyle.Render("✓ Result") + "\n\n")
		b.WriteString(normalStyle.Render(m.result) + "\n")
	}
	b.WriteString("\n" + helpStyle.Render("Enter/Esc → back to operations"))
	return b.String()
}
