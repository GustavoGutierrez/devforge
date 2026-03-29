package tui

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"

	"dev-forge-mcp/internal/tools"
	"dev-forge-mcp/internal/tools/httptools"
)

var httpOps = []string{
	"HTTP Request",
	"Curl to Code",
	"Webhook Replay",
	"Signed URL",
	"URL Parse/Build",
}

type httpToolsModel struct {
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

func newHTTPToolsModel(srv *tools.Server) httpToolsModel {
	return httpToolsModel{srv: srv, values: make(map[string]string)}
}

func (m httpToolsModel) Init() tea.Cmd { return nil }

func (m httpToolsModel) Update(msg tea.Msg) (httpToolsModel, tea.Cmd) {
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

func (m httpToolsModel) updatePhase0(msg tea.KeyMsg) (httpToolsModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.opIdx > 0 {
			m.opIdx--
		}
	case "down", "j":
		if m.opIdx < len(httpOps)-1 {
			m.opIdx++
		}
	case "enter", " ":
		m.fields = httpFields(m.opIdx)
		m.fieldIdx = 0
		m.inputBuf = ""
		m.values = make(map[string]string)
		m.phase = 1
	case "esc", "q":
		m.goHome = true
	}
	return m, nil
}

func (m httpToolsModel) updatePhase1(msg tea.KeyMsg) (httpToolsModel, tea.Cmd) {
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

func (m httpToolsModel) updatePhase2(msg tea.KeyMsg) (httpToolsModel, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc", "q":
		m.phase = 0
		m.inputBuf = ""
		m.values = make(map[string]string)
	}
	return m, nil
}

func (m httpToolsModel) execute() string {
	ctx := context.Background()
	switch m.opIdx {
	case 0:
		return httptools.HTTPRequest(ctx, httptools.HTTPRequestInput{
			URL:    m.values["url"],
			Method: m.values["method"],
			Body:   m.values["body"],
		})
	case 1:
		return httptools.HTTPCurlConvert(ctx, httptools.HTTPCurlConvertInput{
			Curl:   m.values["curl"],
			Target: m.values["target"],
		})
	case 2:
		return httptools.HTTPWebhookReplay(ctx, httptools.HTTPWebhookReplayInput{
			URL:    m.values["url"],
			Method: m.values["method"],
			Body:   m.values["body"],
		})
	case 3:
		return httptools.HTTPSignedURL(ctx, httptools.HTTPSignedURLInput{
			URL:    m.values["url"],
			Secret: m.values["secret"],
			Method: m.values["method"],
		})
	case 4:
		return httptools.HTTPURLParse(ctx, httptools.HTTPURLParseInput{
			URL:    m.values["url"],
			Action: m.values["action"],
		})
	}
	return `{"error":"unknown operation"}`
}

func httpFields(opIdx int) []fieldDef {
	switch opIdx {
	case 0:
		return []fieldDef{
			{"url", "URL", true, "full URL (https://...)", false},
			{"method", "Method", false, "GET | POST | PUT | DELETE (default: GET)", false},
			{"body", "Body", false, "request body (optional)", false},
		}
	case 1:
		return []fieldDef{
			{"curl", "Curl Command", true, "paste a curl command", false},
			{"target", "Target", true, "go | typescript | python", false},
		}
	case 2:
		return []fieldDef{
			{"url", "Target URL", true, "URL to replay the webhook to", false},
			{"method", "Method", false, "POST | PUT (default: POST)", false},
			{"body", "Body", true, "webhook payload body", true},
		}
	case 3:
		return []fieldDef{
			{"url", "URL", true, "URL to sign", false},
			{"secret", "Secret", true, "HMAC signing secret", false},
			{"method", "Method", false, "query | header (default: query)", false},
		}
	case 4:
		return []fieldDef{
			{"url", "URL", true, "URL to parse", false},
			{"action", "Action", false, "parse | build (default: parse)", false},
		}
	}
	return nil
}

func (m httpToolsModel) View() string {
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

func (m httpToolsModel) viewPhase0() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("◆ HTTP & Networking") + "\n\n")
	b.WriteString(normalStyle.Render("Select operation:") + "\n")
	for i, op := range httpOps {
		if i == m.opIdx {
			b.WriteString(selectedStyle.Render(fmt.Sprintf("> %s", op)) + "\n")
		} else {
			b.WriteString(normalStyle.Render(fmt.Sprintf("  %s", op)) + "\n")
		}
	}
	b.WriteString("\n" + helpStyle.Render("↑/↓ navigate • Enter select • Esc back"))
	return b.String()
}

func (m httpToolsModel) viewPhase1() string {
	if m.fieldIdx >= len(m.fields) {
		return ""
	}
	field := m.fields[m.fieldIdx]
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("◆ %s (%d/%d)", httpOps[m.opIdx], m.fieldIdx+1, len(m.fields))) + "\n\n")
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
	b.WriteString("\n" + helpStyle.Render("Esc back • Enter confirm"))
	return b.String()
}

func (m httpToolsModel) viewPhase2() string {
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
