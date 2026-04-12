package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"

	"dev-forge-mcp/internal/tools"
	"dev-forge-mcp/internal/tools/backend"
)

var backendOps = []string{
	"SQL Format",
	"Connection String Builder/Parser",
	"Log Parser",
	"Env File Inspector",
	"MQ Payload Builder",
	"CIDR / Subnet Calculator",
}

type backendToolsModel struct {
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

func newBackendToolsModel(srv *tools.Server) backendToolsModel {
	return backendToolsModel{srv: srv, values: make(map[string]string)}
}

func (m backendToolsModel) Init() tea.Cmd { return nil }

func (m backendToolsModel) Update(msg tea.Msg) (backendToolsModel, tea.Cmd) {
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

func (m backendToolsModel) updatePhase0(msg tea.KeyMsg) (backendToolsModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.opIdx > 0 {
			m.opIdx--
		}
	case "down", "j":
		if m.opIdx < len(backendOps)-1 {
			m.opIdx++
		}
	case "enter", " ":
		m.fields = backendFields(m.opIdx)
		m.fieldIdx = 0
		m.inputBuf = ""
		m.values = make(map[string]string)
		m.phase = 1
	case "esc", "q":
		m.goHome = true
	}
	return m, nil
}

func (m backendToolsModel) updatePhase1(msg tea.KeyMsg) (backendToolsModel, tea.Cmd) {
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

func (m backendToolsModel) updatePhase2(msg tea.KeyMsg) (backendToolsModel, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc", "q":
		m.phase = 0
		m.inputBuf = ""
		m.values = make(map[string]string)
	}
	return m, nil
}

func (m backendToolsModel) execute() string {
	ctx := context.Background()
	switch m.opIdx {
	case 0: // SQL Format
		return backend.SQLFormat(ctx, backend.SQLFormatInput{
			SQL:              m.values["sql"],
			Dialect:          m.values["dialect"],
			Indent:           m.values["indent"],
			UppercaseKeyword: m.values["uppercase"] != "false",
		})
	case 1: // Connection String
		return backend.ConnString(ctx, backend.ConnStringInput{
			Operation:        m.values["operation"],
			DBType:           m.values["db_type"],
			ConnectionString: m.values["connection_string"],
			Host:             m.values["host"],
			Database:         m.values["database"],
			Username:         m.values["username"],
			Password:         m.values["password"],
		})
	case 2: // Log Parser
		return backend.LogParse(ctx, backend.LogParseInput{
			Log:    m.values["log"],
			Format: m.values["format"],
		})
	case 3: // Env Inspector
		return backend.EnvInspect(ctx, backend.EnvInspectInput{
			EnvContent: m.values["env_content"],
			Schema:     m.values["schema"],
			Operation:  m.values["operation"],
		})
	case 4: // MQ Payload
		return backend.MQPayload(ctx, backend.MQPayloadInput{
			Broker:    m.values["broker"],
			Operation: m.values["operation"],
			Topic:     m.values["topic"],
			Payload:   m.values["payload"],
		})
	case 5: // CIDR / Subnet Calculator
		includeAll := true
		if strings.EqualFold(strings.TrimSpace(m.values["include_all"]), "false") {
			includeAll = false
		}
		limit := 256
		if v := strings.TrimSpace(m.values["limit"]); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				limit = n
			}
		}
		return backend.CIDRSubnet(ctx, backend.CIDRSubnetInput{
			CIDR:       m.values["cidr"],
			IncludeAll: includeAll,
			Limit:      limit,
		})
	}
	return `{"error":"unknown operation"}`
}

func backendFields(opIdx int) []fieldDef {
	switch opIdx {
	case 0:
		return []fieldDef{
			{"sql", "SQL Query", true, "SQL statement to format", false},
			{"dialect", "Dialect", false, "postgresql | mysql | sqlite | generic (default: generic)", false},
			{"indent", "Indent", false, "indent string (default: two spaces)", false},
			{"uppercase", "Uppercase Keywords", false, "true | false (default: true)", false},
		}
	case 1:
		return []fieldDef{
			{"operation", "Operation", true, "build | parse", false},
			{"db_type", "DB Type", true, "postgresql | mysql | mongodb | redis", false},
			{"connection_string", "Connection String", false, "connection string (for parse)", false},
			{"host", "Host", false, "hostname (default: localhost)", false},
			{"database", "Database", false, "database name", false},
			{"username", "Username", false, "database username", false},
			{"password", "Password", false, "database password", false},
		}
	case 2:
		return []fieldDef{
			{"log", "Log Content", true, "paste log lines here", false},
			{"format", "Format", false, "json | ndjson | apache | nginx | auto (default: auto)", false},
		}
	case 3:
		return []fieldDef{
			{"env_content", "Env File Content", true, "paste .env file contents", false},
			{"schema", "Schema JSON", false, "JSON schema for validation (optional)", false},
			{"operation", "Operation", false, "validate | generate_example (default: validate)", false},
		}
	case 4:
		return []fieldDef{
			{"broker", "Broker", true, "kafka | rabbitmq | sqs", false},
			{"operation", "Operation", false, "build | serialize | format (default: build)", false},
			{"topic", "Topic / Queue", false, "topic or queue name", false},
			{"payload", "Payload (JSON)", false, "message body as JSON", false},
		}
	case 5:
		return []fieldDef{
			{"cidr", "CIDR", true, "IPv4 CIDR block, e.g. 10.0.0.0/24", false},
			{"include_all", "Include Host List", false, "true | false (default: true)", false},
			{"limit", "Host List Limit", false, "max hosts to return when include_all=true (default: 256)", false},
		}
	}
	return nil
}

func (m backendToolsModel) View() string {
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

func (m backendToolsModel) viewPhase0() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("◆ Backend Utilities") + "\n\n")
	b.WriteString(normalStyle.Render("Select operation:") + "\n")
	for i, op := range backendOps {
		if i == m.opIdx {
			b.WriteString(selectedStyle.Render(fmt.Sprintf("> %s", op)) + "\n")
		} else {
			b.WriteString(normalStyle.Render(fmt.Sprintf("  %s", op)) + "\n")
		}
	}
	b.WriteString("\n" + helpStyle.Render("↑/↓ navigate • Enter select • Esc back"))
	return b.String()
}

func (m backendToolsModel) viewPhase1() string {
	if m.fieldIdx >= len(m.fields) {
		return ""
	}
	field := m.fields[m.fieldIdx]
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("◆ %s (%d/%d)", backendOps[m.opIdx], m.fieldIdx+1, len(m.fields))) + "\n\n")
	b.WriteString(normalStyle.Render(field.label) + "\n")
	if field.hint != "" {
		b.WriteString(dimStyle.Render(field.hint) + "\n")
	}
	b.WriteString("\n")
	b.WriteString(boxStyle.Render(m.inputBuf+"_") + "\n")
	b.WriteString("\n" + helpStyle.Render("Esc back • Enter confirm"))
	return b.String()
}

func (m backendToolsModel) viewPhase2() string {
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
