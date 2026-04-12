package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"

	"dev-forge-mcp/internal/tools"
	"dev-forge-mcp/internal/tools/textenc"
)

var textEncOps = []string{
	"Escape / Unescape",
	"Slug",
	"UUID / ULID / NanoID / Token",
	"Base64 Encode/Decode",
	"URL Encode/Decode",
	"Normalize",
	"Case Convert",
}

type textEncModel struct {
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

func newTextEncModel(srv *tools.Server) textEncModel {
	return textEncModel{srv: srv, values: make(map[string]string)}
}

func (m textEncModel) Init() tea.Cmd { return nil }

func (m textEncModel) Update(msg tea.Msg) (textEncModel, tea.Cmd) {
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

func (m textEncModel) updatePhase0(msg tea.KeyMsg) (textEncModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.opIdx > 0 {
			m.opIdx--
		}
	case "down", "j":
		if m.opIdx < len(textEncOps)-1 {
			m.opIdx++
		}
	case "enter", " ":
		m.fields = textEncFields(m.opIdx)
		m.fieldIdx = 0
		m.inputBuf = ""
		m.values = make(map[string]string)
		m.phase = 1
	case "esc", "q":
		m.goHome = true
	}
	return m, nil
}

func (m textEncModel) updatePhase1(msg tea.KeyMsg) (textEncModel, tea.Cmd) {
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

func (m textEncModel) updatePhase2(msg tea.KeyMsg) (textEncModel, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc", "q":
		m.phase = 0
		m.inputBuf = ""
		m.values = make(map[string]string)
	}
	return m, nil
}

func (m textEncModel) execute() string {
	ctx := context.Background()
	switch m.opIdx {
	case 0:
		return textenc.Escape(ctx, textenc.EscapeInput{
			Text:      m.values["text"],
			Target:    m.values["target"],
			Operation: m.values["operation"],
		})
	case 1:
		return textenc.Slug(ctx, textenc.SlugInput{
			Text:      m.values["text"],
			Separator: m.values["separator"],
			Lower:     m.values["lower"] != "false",
		})
	case 2:
		length := 21
		if v := strings.TrimSpace(m.values["length"]); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				length = n
			}
		}
		count := 1
		if v := strings.TrimSpace(m.values["count"]); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				count = n
			}
		}
		return textenc.UUID(ctx, textenc.UUIDInput{
			Kind:   m.values["kind"],
			Length: length,
			Count:  count,
		})
	case 3:
		return textenc.Base64(ctx, textenc.Base64Input{
			Text:      m.values["text"],
			Variant:   m.values["variant"],
			Operation: m.values["operation"],
		})
	case 4:
		return textenc.URLEncode(ctx, textenc.URLEncodeInput{
			Text:      m.values["text"],
			Operation: m.values["operation"],
			Mode:      m.values["mode"],
		})
	case 5:
		ops := strings.Split(m.values["operations"], ",")
		for i, op := range ops {
			ops[i] = strings.TrimSpace(op)
		}
		return textenc.Normalize(ctx, textenc.NormalizeInput{
			Text:       m.values["text"],
			Operations: ops,
		})
	case 6:
		return textenc.Case(ctx, textenc.CaseInput{
			Text:       m.values["text"],
			TargetCase: m.values["target_case"],
		})
	}
	return `{"error":"unknown operation"}`
}

func textEncFields(opIdx int) []fieldDef {
	switch opIdx {
	case 0:
		return []fieldDef{
			{"text", "Text", true, "text to escape/unescape", false},
			{"target", "Target", true, "json | js | html | sql", false},
			{"operation", "Operation", true, "escape | unescape", false},
		}
	case 1:
		return []fieldDef{
			{"text", "Text", true, "text to slugify", false},
			{"separator", "Separator", false, "default: -", false},
		}
	case 2:
		return []fieldDef{
			{"kind", "Kind", true, "uuid4 | ulid | nanoid | token", false},
			{"length", "Length", false, "for nanoid/token (default: 21)", false},
			{"count", "Count", false, "number of IDs to generate (default: 1)", false},
		}
	case 3:
		return []fieldDef{
			{"text", "Text", true, "text to encode/decode", false},
			{"variant", "Variant", false, "standard | urlsafe", false},
			{"operation", "Operation", true, "encode | decode", false},
		}
	case 4:
		return []fieldDef{
			{"text", "Text", true, "text to encode/decode", false},
			{"operation", "Operation", true, "encode | decode", false},
			{"mode", "Mode", false, "query | path", false},
		}
	case 5:
		return []fieldDef{
			{"text", "Text", true, "text to normalize", false},
			{"operations", "Operations", true, "trim_whitespace, normalize_newlines, strip_bom, nfc...", false},
		}
	case 6:
		return []fieldDef{
			{"text", "Text", true, "text to convert", false},
			{"target_case", "Target Case", true, "camel | snake | kebab | pascal | screaming_snake", false},
		}
	}
	return nil
}

func (m textEncModel) View() string {
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

func (m textEncModel) viewPhase0() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("◆ Text & Encoding") + "\n\n")
	b.WriteString(normalStyle.Render("Select operation:") + "\n")
	for i, op := range textEncOps {
		if i == m.opIdx {
			b.WriteString(selectedStyle.Render(fmt.Sprintf("> %s", op)) + "\n")
		} else {
			b.WriteString(normalStyle.Render(fmt.Sprintf("  %s", op)) + "\n")
		}
	}
	b.WriteString("\n" + helpStyle.Render("↑/↓ navigate • Enter select • Esc back"))
	return b.String()
}

func (m textEncModel) viewPhase1() string {
	if m.fieldIdx >= len(m.fields) {
		return ""
	}
	field := m.fields[m.fieldIdx]
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("◆ %s (%d/%d)", textEncOps[m.opIdx], m.fieldIdx+1, len(m.fields))) + "\n\n")
	b.WriteString(normalStyle.Render(field.label) + "\n")
	if field.hint != "" {
		b.WriteString(dimStyle.Render(field.hint) + "\n")
	}
	b.WriteString("\n")
	b.WriteString(boxStyle.Render(m.inputBuf+"_") + "\n")
	b.WriteString("\n" + helpStyle.Render("Esc back • Enter confirm"))
	return b.String()
}

func (m textEncModel) viewPhase2() string {
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
