package tui

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"

	"dev-forge-mcp/internal/tools"
	"dev-forge-mcp/internal/tools/datetime"
)

var dateTimeOps = []string{
	"Convert Timestamp",
	"Diff / Add / Subtract",
	"Cron Describe & Next Runs",
	"Date Range",
}

type dateTimeModel struct {
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

func newDateTimeModel(srv *tools.Server) dateTimeModel {
	return dateTimeModel{srv: srv, values: make(map[string]string)}
}

func (m dateTimeModel) Init() tea.Cmd { return nil }

func (m dateTimeModel) Update(msg tea.Msg) (dateTimeModel, tea.Cmd) {
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

func (m dateTimeModel) updatePhase0(msg tea.KeyMsg) (dateTimeModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.opIdx > 0 {
			m.opIdx--
		}
	case "down", "j":
		if m.opIdx < len(dateTimeOps)-1 {
			m.opIdx++
		}
	case "enter", " ":
		m.fields = dateTimeFields(m.opIdx)
		m.fieldIdx = 0
		m.inputBuf = ""
		m.values = make(map[string]string)
		m.phase = 1
	case "esc", "q":
		m.goHome = true
	}
	return m, nil
}

func (m dateTimeModel) updatePhase1(msg tea.KeyMsg) (dateTimeModel, tea.Cmd) {
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

func (m dateTimeModel) updatePhase2(msg tea.KeyMsg) (dateTimeModel, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc", "q":
		m.phase = 0
		m.inputBuf = ""
		m.values = make(map[string]string)
	}
	return m, nil
}

func (m dateTimeModel) execute() string {
	ctx := context.Background()
	switch m.opIdx {
	case 0:
		return datetime.TimeConvert(ctx, datetime.TimeConvertInput{
			Input:      m.values["input"],
			FromFormat: m.values["from_format"],
			ToFormat:   m.values["to_format"],
			Timezone:   m.values["timezone"],
		})
	case 1:
		return datetime.TimeDiff(ctx, datetime.TimeDiffInput{
			Start:     m.values["start"],
			End:       m.values["end"],
			Operation: m.values["operation"],
			Duration:  m.values["duration"],
		})
	case 2:
		return datetime.TimeCron(ctx, datetime.TimeCronInput{
			Expression: m.values["expression"],
			Operation:  m.values["operation"],
		})
	case 3:
		return datetime.TimeDateRange(ctx, datetime.TimeDateRangeInput{
			Start:  m.values["start"],
			End:    m.values["end"],
			Step:   m.values["step"],
			Format: m.values["format"],
		})
	}
	return `{"error":"unknown operation"}`
}

func dateTimeFields(opIdx int) []fieldDef {
	switch opIdx {
	case 0:
		return []fieldDef{
			{"input", "Input", true, "timestamp: unix epoch, ISO 8601, RFC 3339, or human date", false},
			{"from_format", "From Format", false, "unix | iso8601 | rfc3339 | human | auto (default: auto)", false},
			{"to_format", "To Format", false, "unix | iso8601 | rfc3339 | human (default: rfc3339)", false},
			{"timezone", "Timezone", false, "e.g. America/New_York (default: UTC)", false},
		}
	case 1:
		return []fieldDef{
			{"start", "Start Timestamp", true, "start time (ISO 8601, unix, or human)", false},
			{"end", "End Timestamp", false, "end time for diff operation", false},
			{"operation", "Operation", false, "diff | add | subtract (default: diff)", false},
			{"duration", "Duration", false, "e.g. 2h30m or '3 days' (for add/subtract)", false},
		}
	case 2:
		return []fieldDef{
			{"expression", "Cron Expression", true, "e.g. 0 9 * * 1-5", false},
			{"operation", "Operation", false, "describe | next (default: describe)", false},
		}
	case 3:
		return []fieldDef{
			{"start", "Start Date", true, "start date (ISO 8601 or human)", false},
			{"end", "End Date", true, "end date (ISO 8601 or human)", false},
			{"step", "Step", false, "day | week | month (default: day)", false},
			{"format", "Format", false, "iso8601 | unix | human (default: iso8601)", false},
		}
	}
	return nil
}

func (m dateTimeModel) View() string {
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

func (m dateTimeModel) viewPhase0() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("◆ Date & Time") + "\n\n")
	b.WriteString(normalStyle.Render("Select operation:") + "\n")
	for i, op := range dateTimeOps {
		if i == m.opIdx {
			b.WriteString(selectedStyle.Render(fmt.Sprintf("> %s", op)) + "\n")
		} else {
			b.WriteString(normalStyle.Render(fmt.Sprintf("  %s", op)) + "\n")
		}
	}
	b.WriteString("\n" + helpStyle.Render("↑/↓ navigate • Enter select • Esc back"))
	return b.String()
}

func (m dateTimeModel) viewPhase1() string {
	if m.fieldIdx >= len(m.fields) {
		return ""
	}
	field := m.fields[m.fieldIdx]
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("◆ %s (%d/%d)", dateTimeOps[m.opIdx], m.fieldIdx+1, len(m.fields))) + "\n\n")
	b.WriteString(normalStyle.Render(field.label) + "\n")
	if field.hint != "" {
		b.WriteString(dimStyle.Render(field.hint) + "\n")
	}
	b.WriteString("\n")
	b.WriteString(boxStyle.Render(m.inputBuf+"_") + "\n")
	b.WriteString("\n" + helpStyle.Render("Esc back • Enter confirm"))
	return b.String()
}

func (m dateTimeModel) viewPhase2() string {
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
