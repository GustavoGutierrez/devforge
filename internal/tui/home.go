package tui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const asciiLogo = `██████╗ ███████╗██╗   ██╗███████╗ ██████╗ ██████╗  ██████╗ ███████╗
██╔══██╗██╔════╝██║   ██║██╔════╝██╔═══██╗██╔══██╗██╔════╝ ██╔════╝
██║  ██║█████╗  ██║   ██║█████╗  ██║   ██║██████╔╝██║  ███╗█████╗
██║  ██║██╔══╝  ╚██╗ ██╔╝██╔══╝  ██║   ██║██╔══██╗██║   ██║██╔══╝
██████╔╝███████╗ ╚████╔╝ ██║     ╚██████╔╝██║  ██║╚██████╔╝███████╗
╚═════╝ ╚══════╝  ╚═══╝  ╚═╝      ╚═════╝ ╚═╝  ╚═╝ ╚═════╝ ╚══════╝`

type menuEntry struct {
	label     string
	isSection bool
}

var menuEntries = []menuEntry{
	{label: "── Design & Layout", isSection: true},
	{label: "Browse patterns"},
	{label: "Browse architectures"},
	{label: "Analyze layout file"},
	{label: "Generate layout"},
	{label: "Explore color palettes"},
	{label: "── Images & Media", isSection: true},
	{label: "Generate Image (Gemini)"},
	{label: "Optimize images"},
	{label: "Generate favicon"},
	{label: "Process video"},
	{label: "Process audio"},
	{label: "UI to Markdown"},
	{label: "Markdown to PDF"},
	{label: "── Developer Tools", isSection: true},
	{label: "Text & Encoding"},
	{label: "Data Format"},
	{label: "Security & Cryptography"},
	{label: "HTTP & Networking"},
	{label: "Date & Time"},
	{label: "File & Archive"},
	{label: "Frontend Utilities"},
	{label: "Backend Utilities"},
	{label: "Code Utilities"},
	{label: "── System", isSection: true},
	{label: "Settings"},
	{label: "Add Record"},
	{label: "Setup MCP Clients"},
	{label: "About"},
	{label: "Quit"},
}

// latestVersionMsg carries the result of the update check.
type latestVersionMsg struct {
	version string
}

type homeModel struct {
	cursor         int
	selected       int
	version        string
	latestVersion  string
	checkingUpdate bool
}

func newHomeModel(ver string) homeModel {
	return homeModel{cursor: firstSelectable(), selected: -1, version: ver}
}

// firstSelectable returns the index of the first non-section entry.
func firstSelectable() int {
	for i, e := range menuEntries {
		if !e.isSection {
			return i
		}
	}
	return 0
}

func (m homeModel) Init() tea.Cmd {
	m.checkingUpdate = true
	return checkLatestVersion
}

// checkLatestVersion queries the GitHub releases API for the latest tag.
func checkLatestVersion() tea.Msg {
	client := &http.Client{Timeout: 5 * 1e9} // 5s timeout
	req, err := http.NewRequest("GET", "https://api.github.com/repos/GustavoGutierrez/devforge/releases/latest", nil)
	if err != nil {
		return latestVersionMsg{}
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return latestVersionMsg{}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return latestVersionMsg{}
	}
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &release); err != nil {
		return latestVersionMsg{}
	}
	tag := strings.TrimPrefix(release.TagName, "v")
	return latestVersionMsg{version: tag}
}

func (m homeModel) Update(msg tea.Msg) (homeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case latestVersionMsg:
		m.checkingUpdate = false
		m.latestVersion = msg.version
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m.moveCursor(-1)
		case "down", "j":
			m.moveCursor(1)
		case "enter", " ":
			entry := menuEntries[m.cursor]
			if entry.isSection {
				return m, nil
			}
			if entry.label == "Quit" {
				return m, tea.Quit
			}
			m.selected = m.selectableIndex()
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

// moveCursor moves the cursor by delta, skipping section headers.
func (m *homeModel) moveCursor(delta int) {
	next := m.cursor + delta
	for next >= 0 && next < len(menuEntries) {
		if !menuEntries[next].isSection {
			m.cursor = next
			return
		}
		next += delta
	}
}

// selectableIndex counts the number of non-section items before (and including) cursor.
func (m homeModel) selectableIndex() int {
	count := 0
	for i := 0; i < m.cursor; i++ {
		if !menuEntries[i].isSection {
			count++
		}
	}
	return count
}

func (m homeModel) View() string {
	var b strings.Builder

	logoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12"))
	b.WriteString(logoStyle.Render(asciiLogo) + "\n\n")

	b.WriteString(dimStyle.Render("  AI-powered design & developer toolkit") + "\n")
	b.WriteString(dimStyle.Render("  Design · Layout · Media · Cryptography · HTTP · Code · 60+ tools") + "\n")

	versionLine := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render(fmt.Sprintf("  v%s", m.version))
	if m.latestVersion != "" && m.latestVersion != m.version {
		badge := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render(fmt.Sprintf("  ⬆ v%s available", m.latestVersion))
		versionLine += badge
	}
	b.WriteString(versionLine + "\n")

	b.WriteString(helpStyle.Render("↑ ↓  move   Enter  select   Tab  switch field   Esc  go back   q  quit") + "\n\n")

	for i, entry := range menuEntries {
		if entry.isSection {
			b.WriteString(dimStyle.Render("  "+entry.label) + "\n")
			continue
		}

		cursor := "  "
		if m.cursor == i {
			cursor = "> "
		}

		var line string
		if m.cursor == i {
			line = selectedStyle.Render(fmt.Sprintf("%s%s", cursor, entry.label))
		} else {
			line = normalStyle.Render(fmt.Sprintf("%s%s", cursor, entry.label))
		}
		b.WriteString(line + "\n")
	}

	return boxStyle.Render(b.String())
}
