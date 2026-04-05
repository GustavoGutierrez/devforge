package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"dev-forge-mcp/internal/tools"
)

const audioToolIntro = `Process audio files with various operations like transcoding, trimming, normalization, and silence removal.`

var audioMenuItems = []string{
	"Transcode audio",
	"Trim audio",
	"Normalize loudness",
	"Remove silence",
	"Back",
}

type audioModel struct {
	selectedTool int
	srv          *tools.Server
	result       string
	loading      bool
	goHome       bool
	goSettings   bool
}

func newAudioModel(srv *tools.Server) audioModel {
	return audioModel{selectedTool: 0, srv: srv}
}

func (m audioModel) Init() tea.Cmd { return nil }

func (m audioModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selectedTool > 0 {
				m.selectedTool--
			}
		case "down", "j":
			if m.selectedTool < len(audioMenuItems)-1 {
				m.selectedTool++
			}
		case "enter", " ":
			if m.selectedTool == len(audioMenuItems)-1 {
				m.goHome = true
			}
		case "q", "esc":
			m.goHome = true
		}
	}
	return m, nil
}

func (m audioModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("🎵 Audio Processing"))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render(audioToolIntro))
	b.WriteString("\n\n")

	b.WriteString(helpStyle.Render("↑ ↓  move   Enter  select   Esc/q  go back") + "\n\n")

	for i, item := range audioMenuItems {
		cursor := "  "
		if m.selectedTool == i {
			cursor = "> "
		}

		var line string
		if m.selectedTool == i {
			line = selectedStyle.Render(fmt.Sprintf("%s%s", cursor, item))
		} else {
			line = normalStyle.Render(fmt.Sprintf("%s%s", cursor, item))
		}
		b.WriteString(line + "\n")
	}

	if m.result != "" {
		b.WriteString("\n")
		b.WriteString(boxStyle.Render(m.result))
	}

	return b.String()
}
