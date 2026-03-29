package tui

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"

	"dev-forge-mcp/internal/tools"
	"dev-forge-mcp/internal/tools/datafmt"
)

var dataFmtOps = []string{
	"Format JSON",
	"YAML Convert (JSON↔YAML)",
	"CSV Convert (CSV↔JSON)",
	"JSONPath Extract",
	"Schema Validate",
	"Diff (JSON/YAML)",
}

type dataFmtModel struct {
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

func newDataFmtModel(srv *tools.Server) dataFmtModel {
	return dataFmtModel{srv: srv, values: make(map[string]string)}
}

func (m dataFmtModel) Init() tea.Cmd { return nil }

func (m dataFmtModel) Update(msg tea.Msg) (dataFmtModel, tea.Cmd) {
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

func (m dataFmtModel) updatePhase0(msg tea.KeyMsg) (dataFmtModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.opIdx > 0 {
			m.opIdx--
		}
	case "down", "j":
		if m.opIdx < len(dataFmtOps)-1 {
			m.opIdx++
		}
	case "enter", " ":
		m.fields = dataFmtFields(m.opIdx)
		m.fieldIdx = 0
		m.inputBuf = ""
		m.values = make(map[string]string)
		m.phase = 1
	case "esc", "q":
		m.goHome = true
	}
	return m, nil
}

func (m dataFmtModel) updatePhase1(msg tea.KeyMsg) (dataFmtModel, tea.Cmd) {
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
	case "ctrl+j":
		m.inputBuf += "\n"
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

func (m dataFmtModel) updatePhase2(msg tea.KeyMsg) (dataFmtModel, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc", "q":
		m.phase = 0
		m.inputBuf = ""
		m.values = make(map[string]string)
	}
	return m, nil
}

func (m dataFmtModel) execute() string {
	ctx := context.Background()
	switch m.opIdx {
	case 0:
		return datafmt.FormatJSON(ctx, datafmt.FormatJSONInput{
			JSON:   m.values["json"],
			Indent: m.values["indent"],
		})
	case 1:
		return datafmt.YAMLConvert(ctx, datafmt.YAMLConvertInput{
			Input: m.values["input"],
			From:  m.values["from"],
			To:    m.values["to"],
		})
	case 2:
		return datafmt.CSVConvert(ctx, datafmt.CSVConvertInput{
			Input:     m.values["input"],
			From:      m.values["from"],
			To:        m.values["to"],
			Separator: m.values["separator"],
			HasHeader: m.values["has_header"] == "true",
		})
	case 3:
		return datafmt.JSONPath(ctx, datafmt.JSONPathInput{
			JSON: m.values["json"],
			Path: m.values["path"],
		})
	case 4:
		return datafmt.SchemaValidate(ctx, datafmt.SchemaValidateInput{
			JSON:   m.values["json"],
			Schema: m.values["schema"],
		})
	case 5:
		return datafmt.Diff(ctx, datafmt.DiffInput{
			A:      m.values["a"],
			B:      m.values["b"],
			Format: m.values["format"],
		})
	}
	return `{"error":"unknown operation"}`
}

func dataFmtFields(opIdx int) []fieldDef {
	switch opIdx {
	case 0:
		return []fieldDef{
			{"json", "JSON", true, "JSON string to format", true},
			{"indent", "Indent", false, "indent string (default: 2 spaces)", false},
		}
	case 1:
		return []fieldDef{
			{"input", "Input", true, "JSON or YAML content", true},
			{"from", "From", true, "json | yaml", false},
			{"to", "To", true, "json | yaml", false},
		}
	case 2:
		return []fieldDef{
			{"input", "Input", true, "CSV or JSON content", true},
			{"from", "From", true, "csv | json", false},
			{"to", "To", true, "csv | json", false},
			{"separator", "Separator", false, "default: ,", false},
			{"has_header", "Has Header", false, "true | false (for CSV input)", false},
		}
	case 3:
		return []fieldDef{
			{"json", "JSON", true, "JSON document", true},
			{"path", "JSONPath", true, "e.g. $.store.book[0].title", false},
		}
	case 4:
		return []fieldDef{
			{"json", "JSON", true, "JSON document to validate", true},
			{"schema", "Schema", true, "JSON Schema", true},
		}
	case 5:
		return []fieldDef{
			{"a", "Document A", true, "first JSON/YAML document", true},
			{"b", "Document B", true, "second JSON/YAML document", true},
			{"format", "Format", false, "json | yaml", false},
		}
	}
	return nil
}

func (m dataFmtModel) View() string {
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

func (m dataFmtModel) viewPhase0() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("◆ Data Format") + "\n\n")
	b.WriteString(normalStyle.Render("Select operation:") + "\n")
	for i, op := range dataFmtOps {
		if i == m.opIdx {
			b.WriteString(selectedStyle.Render(fmt.Sprintf("> %s", op)) + "\n")
		} else {
			b.WriteString(normalStyle.Render(fmt.Sprintf("  %s", op)) + "\n")
		}
	}
	b.WriteString("\n" + helpStyle.Render("↑/↓ navigate • Enter select • Esc back"))
	return b.String()
}

func (m dataFmtModel) viewPhase1() string {
	if m.fieldIdx >= len(m.fields) {
		return ""
	}
	field := m.fields[m.fieldIdx]
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("◆ %s (%d/%d)", dataFmtOps[m.opIdx], m.fieldIdx+1, len(m.fields))) + "\n\n")
	b.WriteString(normalStyle.Render(field.label) + "\n")
	if field.hint != "" {
		b.WriteString(dimStyle.Render(field.hint) + "\n")
	}
	b.WriteString("\n")
	if field.multiline {
		lines := strings.Split(m.inputBuf, "\n")
		start := 0
		if len(lines) > 5 {
			start = len(lines) - 5
		}
		for _, line := range lines[start:] {
			b.WriteString(boxStyle.Render(line) + "\n")
		}
		if len(m.inputBuf) == 0 {
			b.WriteString(boxStyle.Render("_") + "\n")
		}
		b.WriteString(dimStyle.Render("(ctrl+j for newline)") + "\n")
	} else {
		b.WriteString(boxStyle.Render(m.inputBuf+"_") + "\n")
	}
	help := "Esc back • Enter confirm"
	if field.multiline {
		help += " • ctrl+j newline"
	}
	b.WriteString("\n" + helpStyle.Render(help))
	return b.String()
}

func (m dataFmtModel) viewPhase2() string {
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
