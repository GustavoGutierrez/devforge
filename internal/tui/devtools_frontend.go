package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"

	"dev-forge-mcp/internal/tools"
	"dev-forge-mcp/internal/tools/frontend"
)

var frontendOps = []string{
	"Color Convert / WCAG Contrast",
	"CSS Unit Converter",
	"Breakpoint Lookup",
	"Regex Tester",
	"Locale Format",
	"ICU Message Format",
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
	}
	return nil
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
