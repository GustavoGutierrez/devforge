package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"dev-forge-mcp/internal/tools"
)

const videoToolIntro = `Process video files with various operations like transcoding, resizing, trimming, and thumbnail generation.`

var videoMenuItems = []string{
	"Transcode video",
	"Resize video",
	"Trim video",
	"Extract thumbnail",
	"Apply encoding profile",
	"Back",
}

type videoModel struct {
	selectedTool int
	srv          *tools.Server
	result       string
	loading      bool
	goHome       bool
	goSettings   bool
}

func newVideoModel(srv *tools.Server) videoModel {
	return videoModel{selectedTool: 0, srv: srv}
}

func (m videoModel) Init() tea.Cmd { return nil }

func (m videoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selectedTool > 0 {
				m.selectedTool--
			}
		case "down", "j":
			if m.selectedTool < len(videoMenuItems)-1 {
				m.selectedTool++
			}
		case "enter", " ":
			if m.selectedTool == len(videoMenuItems)-1 {
				m.goHome = true
			}
		case "q", "esc":
			m.goHome = true
		}
	}
	return m, nil
}

func (m videoModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("🎬 Video Processing"))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render(videoToolIntro))
	b.WriteString("\n\n")

	b.WriteString(helpStyle.Render("↑ ↓  move   Enter  select   Esc/q  go back") + "\n\n")

	for i, item := range videoMenuItems {
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
