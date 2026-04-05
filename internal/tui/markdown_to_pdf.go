package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"dev-forge-mcp/internal/tools"
)

type markdownToPDFModel struct {
	srv          *tools.Server
	inputPath    string
	outputPath   string
	markdownText string
	theme        string
	pageSize     string
	layoutMode   string
	inline       bool
	field        int
	result       string
	err          string
	generating   bool
	goHome       bool
}

func newMarkdownToPDFModel(srv *tools.Server) markdownToPDFModel {
	return markdownToPDFModel{
		srv:        srv,
		theme:      "engineering",
		pageSize:   "a4",
		layoutMode: "paged",
	}
}

func (m markdownToPDFModel) Init() tea.Cmd { return nil }

func (m markdownToPDFModel) Update(msg tea.Msg) (markdownToPDFModel, tea.Cmd) {
	switch msg := msg.(type) {
	case markdownToPDFResultMsg:
		m.generating = false
		raw := string(msg)
		var out tools.MarkdownToPDFOutput
		if json.Unmarshal([]byte(raw), &out) == nil {
			m.result = raw
			m.err = ""
		} else {
			m.err = raw
			m.result = ""
		}
		return m, nil
	case tea.KeyMsg:
		if m.generating {
			return m, nil
		}
		switch msg.String() {
		case "esc":
			if m.result != "" || m.err != "" {
				m.result = ""
				m.err = ""
			} else {
				m.goHome = true
			}
		case "tab":
			m.field = (m.field + 1) % 6
		case "shift+tab":
			m.field = (m.field + 5) % 6
		case "i":
			m.inline = !m.inline
		case "enter":
			m.generating = true
			m.result = ""
			m.err = ""
			return m, m.generate()
		case "backspace":
			switch m.field {
			case 0:
				if len(m.inputPath) > 0 {
					m.inputPath = m.inputPath[:len(m.inputPath)-1]
				}
			case 1:
				if len(m.outputPath) > 0 {
					m.outputPath = m.outputPath[:len(m.outputPath)-1]
				}
			case 2:
				if len(m.markdownText) > 0 {
					m.markdownText = m.markdownText[:len(m.markdownText)-1]
				}
			case 3:
				if len(m.theme) > 0 {
					m.theme = m.theme[:len(m.theme)-1]
				}
			case 4:
				if len(m.pageSize) > 0 {
					m.pageSize = m.pageSize[:len(m.pageSize)-1]
				}
			case 5:
				if len(m.layoutMode) > 0 {
					m.layoutMode = m.layoutMode[:len(m.layoutMode)-1]
				}
			}
		default:
			if msg.Paste {
				m = m.appendText(string(msg.Runes))
			} else if len(msg.String()) == 1 {
				m = m.appendText(msg.String())
			}
		}
	}
	return m, nil
}

type markdownToPDFResultMsg string

func (m markdownToPDFModel) appendText(text string) markdownToPDFModel {
	switch m.field {
	case 0:
		m.inputPath += text
	case 1:
		m.outputPath += text
	case 2:
		m.markdownText += text
	case 3:
		m.theme += text
	case 4:
		m.pageSize += text
	case 5:
		m.layoutMode += text
	}
	return m
}

func (m markdownToPDFModel) generate() tea.Cmd {
	return func() tea.Msg {
		input := tools.MarkdownToPDFInput{
			Input:        strings.TrimSpace(m.inputPath),
			MarkdownText: strings.TrimSpace(m.markdownText),
			Output:       strings.TrimSpace(m.outputPath),
			Inline:       m.inline,
			Theme:        strings.TrimSpace(m.theme),
			PageSize:     strings.TrimSpace(m.pageSize),
			LayoutMode:   strings.TrimSpace(m.layoutMode),
		}
		result := m.srv.MarkdownToPDF(context.Background(), input)
		return markdownToPDFResultMsg(result)
	}
}

func (m markdownToPDFModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Markdown to PDF") + "\n\n")
	b.WriteString(dimStyle.Render("Use file input for Markdown with local assets. Leave Input empty to render Markdown text inline.") + "\n\n")

	fieldNames := []string{
		"Markdown file path",
		"Output PDF path",
		"Markdown text",
		"Theme",
		"Page size",
		"Layout mode",
	}
	fieldVals := []string{m.inputPath, m.outputPath, m.markdownText, m.theme, m.pageSize, m.layoutMode}

	for i, name := range fieldNames {
		cursor := "  "
		if m.field == i {
			cursor = "> "
		}
		val := fieldVals[i]
		if m.field == i {
			val += "_"
		}
		line := fmt.Sprintf("%s%-20s %s", cursor, name+":", val)
		if m.field == i {
			b.WriteString(selectedStyle.Render(line) + "\n")
		} else {
			b.WriteString(normalStyle.Render(line) + "\n")
		}
	}

	inlineState := "false"
	if m.inline {
		inlineState = "true"
	}
	b.WriteString("\n" + dimStyle.Render("Inline output: "+inlineState+"  • press i to toggle"))

	if m.generating {
		b.WriteString("\n" + helpStyle.Render("Please wait..."))
	} else {
		b.WriteString("\n" + helpStyle.Render("Tab move field • i toggle inline • Enter render • Esc back"))
	}

	if m.generating {
		b.WriteString("\n\n" + dimStyle.Render("Rendering PDF..."))
	}

	if m.result != "" {
		var out tools.MarkdownToPDFOutput
		if json.Unmarshal([]byte(m.result), &out) == nil {
			b.WriteString("\n\n" + successStyle.Render(fmt.Sprintf("✓ PDF rendered in %d ms", out.ElapsedMs)))
			for _, file := range out.Outputs {
				line := fmt.Sprintf("  %s  %d bytes", file.Path, file.SizeBytes)
				if file.Path == "" && file.DataBase64 != "" {
					line = fmt.Sprintf("  inline PDF  base64 length %d", len(file.DataBase64))
				}
				b.WriteString("\n" + dimStyle.Render(line))
			}
		} else {
			b.WriteString("\n\n" + successStyle.Render("✓ Result: "+m.result))
		}
	}
	if m.err != "" {
		b.WriteString("\n\n" + errorStyle.Render("✗ Error: "+m.err))
	}

	return b.String()
}
