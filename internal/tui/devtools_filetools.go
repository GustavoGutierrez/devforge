package tui

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"

	"dev-forge-mcp/internal/tools"
	"dev-forge-mcp/internal/tools/filetools"
)

var fileToolsOps = []string{
	"File Checksum",
	"Archive (Create/Extract)",
	"File Diff",
	"Line Endings",
	"Hex View",
}

type fileToolsModel struct {
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

func newFileToolsModel(srv *tools.Server) fileToolsModel {
	return fileToolsModel{srv: srv, values: make(map[string]string)}
}

func (m fileToolsModel) Init() tea.Cmd { return nil }

func (m fileToolsModel) Update(msg tea.Msg) (fileToolsModel, tea.Cmd) {
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

func (m fileToolsModel) updatePhase0(msg tea.KeyMsg) (fileToolsModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.opIdx > 0 {
			m.opIdx--
		}
	case "down", "j":
		if m.opIdx < len(fileToolsOps)-1 {
			m.opIdx++
		}
	case "enter", " ":
		m.fields = fileToolsFields(m.opIdx)
		m.fieldIdx = 0
		m.inputBuf = ""
		m.values = make(map[string]string)
		m.phase = 1
	case "esc", "q":
		m.goHome = true
	}
	return m, nil
}

func (m fileToolsModel) updatePhase1(msg tea.KeyMsg) (fileToolsModel, tea.Cmd) {
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

func (m fileToolsModel) updatePhase2(msg tea.KeyMsg) (fileToolsModel, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc", "q":
		m.phase = 0
		m.inputBuf = ""
		m.values = make(map[string]string)
	}
	return m, nil
}

func (m fileToolsModel) execute() string {
	ctx := context.Background()
	switch m.opIdx {
	case 0:
		return filetools.Checksum(ctx, filetools.ChecksumInput{
			Path:      m.values["path"],
			Algorithm: m.values["algorithm"],
		})
	case 1:
		return filetools.Archive(ctx, filetools.ArchiveInput{
			Operation: m.values["operation"],
			Format:    m.values["format"],
			Source:    m.values["source"],
			Output:    m.values["output"],
			Archive:   m.values["archive"],
			Dest:      m.values["dest"],
		})
	case 2:
		return filetools.Diff(ctx, filetools.DiffInput{
			A:    m.values["a"],
			B:    m.values["b"],
			Mode: m.values["mode"],
		})
	case 3:
		return filetools.LineEndings(ctx, filetools.LineEndingsInput{
			Input:     m.values["input"],
			Mode:      m.values["mode"],
			Operation: m.values["operation"],
			Target:    m.values["target"],
		})
	case 4:
		return filetools.HexView(ctx, filetools.HexViewInput{
			Input: m.values["input"],
			Mode:  m.values["mode"],
		})
	}
	return `{"error":"unknown operation"}`
}

func fileToolsFields(opIdx int) []fieldDef {
	switch opIdx {
	case 0:
		return []fieldDef{
			{"path", "File Path", true, "absolute or relative path to file", false},
			{"algorithm", "Algorithm", false, "md5 | sha256 | sha512 (default: sha256)", false},
		}
	case 1:
		return []fieldDef{
			{"operation", "Operation", true, "create | extract", false},
			{"format", "Format", false, "zip | tar.gz (default: zip)", false},
			{"source", "Source", false, "source path (for create)", false},
			{"output", "Output", false, "output archive path (for create)", false},
			{"archive", "Archive", false, "archive path (for extract)", false},
			{"dest", "Destination", false, "destination directory (for extract)", false},
		}
	case 2:
		return []fieldDef{
			{"a", "File A / Text A", true, "first file path or text", false},
			{"b", "File B / Text B", true, "second file path or text", false},
			{"mode", "Mode", false, "file | text (default: file)", false},
		}
	case 3:
		return []fieldDef{
			{"input", "Input", true, "file path or raw text", false},
			{"mode", "Mode", false, "file | text (default: file)", false},
			{"operation", "Operation", false, "detect | normalize | convert (default: detect)", false},
			{"target", "Target", false, "lf | crlf (default: lf)", false},
		}
	case 4:
		return []fieldDef{
			{"input", "Input", true, "file path or base64-encoded bytes", false},
			{"mode", "Mode", false, "file | base64 (default: file)", false},
		}
	}
	return nil
}

func (m fileToolsModel) View() string {
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

func (m fileToolsModel) viewPhase0() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("◆ File & Archive") + "\n\n")
	b.WriteString(normalStyle.Render("Select operation:") + "\n")
	for i, op := range fileToolsOps {
		if i == m.opIdx {
			b.WriteString(selectedStyle.Render(fmt.Sprintf("> %s", op)) + "\n")
		} else {
			b.WriteString(normalStyle.Render(fmt.Sprintf("  %s", op)) + "\n")
		}
	}
	b.WriteString("\n" + helpStyle.Render("↑/↓ navigate • Enter select • Esc back"))
	return b.String()
}

func (m fileToolsModel) viewPhase1() string {
	if m.fieldIdx >= len(m.fields) {
		return ""
	}
	field := m.fields[m.fieldIdx]
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("◆ %s (%d/%d)", fileToolsOps[m.opIdx], m.fieldIdx+1, len(m.fields))) + "\n\n")
	b.WriteString(normalStyle.Render(field.label) + "\n")
	if field.hint != "" {
		b.WriteString(dimStyle.Render(field.hint) + "\n")
	}
	b.WriteString("\n")
	b.WriteString(boxStyle.Render(m.inputBuf+"_") + "\n")
	b.WriteString("\n" + helpStyle.Render("Esc back • Enter confirm"))
	return b.String()
}

func (m fileToolsModel) viewPhase2() string {
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
