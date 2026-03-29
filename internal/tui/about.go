package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const aboutLogo = `██████╗ ███████╗██╗   ██╗███████╗ ██████╗ ██████╗  ██████╗ ███████╗
██╔══██╗██╔════╝██║   ██║██╔════╝██╔═══██╗██╔══██╗██╔════╝ ██╔════╝
██║  ██║█████╗  ██║   ██║█████╗  ██║   ██║██████╔╝██║  ███╗█████╗
██║  ██║██╔══╝  ╚██╗ ██╔╝██╔══╝  ██║   ██║██╔══██╗██║   ██║██╔══╝
██████╔╝███████╗ ╚████╔╝ ██║     ╚██████╔╝██║  ██║╚██████╔╝███████╗
╚═════╝ ╚══════╝  ╚═══╝  ╚═╝      ╚═════╝ ╚═╝  ╚═╝ ╚═════╝ ╚══════╝`

type aboutModel struct {
	version string
}

func newAboutModel(ver string) aboutModel { return aboutModel{version: ver} }

func (m aboutModel) Init() tea.Cmd { return nil }

func (m aboutModel) Update(msg tea.Msg) (aboutModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "enter", "q":
			return m, func() tea.Msg { return NavigateTo{ViewHome} }
		}
	}
	return m, nil
}

func (m aboutModel) View() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	shareDir := home + "/.local/share/devforge/current"
	binDir := home + "/.local/bin"
	dbPath := shareDir + "/devforge.db"
	cfgPath := home + "/.config/devforge/config.json"

	var b strings.Builder

	logoStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	b.WriteString(logoStyle.Render(aboutLogo))
	b.WriteString("\n\n")

	b.WriteString(titleStyle.Render("DevForge MCP"))
	b.WriteString("\n")
	b.WriteString(normalStyle.Render("  Version  "))
	b.WriteString(dimStyle.Render(fmt.Sprintf("v%s", m.version)))
	b.WriteString("\n\n")

	b.WriteString(normalStyle.Render("DevForge MCP is an MCP server built in Go that acts as the core acceleration"))
	b.WriteString("\n")
	b.WriteString(normalStyle.Render("layer for the software development lifecycle. It integrates an ecosystem of"))
	b.WriteString("\n")
	b.WriteString(normalStyle.Render("tools, utilities, skills, and specialized sub-agents that work together to"))
	b.WriteString("\n")
	b.WriteString(normalStyle.Render("reduce friction at every phase of development — from architecture design to"))
	b.WriteString("\n")
	b.WriteString(normalStyle.Render("the delivery of sophisticated, production-ready interfaces."))
	b.WriteString("\n\n")
	b.WriteString(normalStyle.Render("More than a code generator, DevForge MCP is a cross-stack intelligence layer"))
	b.WriteString("\n")
	b.WriteString(normalStyle.Render("that guarantees structural consistency, replicable quality, and modern designs"))
	b.WriteString("\n")
	b.WriteString(normalStyle.Render("across all projects, regardless of the technology layer."))
	b.WriteString("\n\n")

	b.WriteString(titleStyle.Render("CONFIGURATION"))
	b.WriteString("\n")
	b.WriteString(normalStyle.Render("  Config file: "))
	b.WriteString(dimStyle.Render("~/.config/devforge/config.json"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  (or override with DEV_FORGE_CONFIG env var)"))
	b.WriteString("\n\n")

	b.WriteString(titleStyle.Render("INSTALLATION"))
	b.WriteString("\n")
	b.WriteString(normalStyle.Render("  Binaries  : "))
	b.WriteString(dimStyle.Render(shareDir + "/"))
	b.WriteString("\n")
	b.WriteString(normalStyle.Render("  Symlinks  : "))
	b.WriteString(dimStyle.Render(binDir + "/  (devforge, devforge-mcp, dpf)"))
	b.WriteString("\n")
	b.WriteString(normalStyle.Render("  Database  : "))
	b.WriteString(dimStyle.Render(dbPath))
	b.WriteString("\n")
	b.WriteString(normalStyle.Render("  Config    : "))
	b.WriteString(dimStyle.Render(cfgPath))
	b.WriteString("\n\n")

	b.WriteString(titleStyle.Render("BINARIES"))
	b.WriteString("\n")
	b.WriteString(normalStyle.Render("  devforge           "))
	b.WriteString(dimStyle.Render("— Interactive TUI for all DevForge tools (this app)"))
	b.WriteString("\n")
	b.WriteString(normalStyle.Render("  devforge-mcp      "))
	b.WriteString(dimStyle.Render("— MCP server (stdio) — attach to Claude or any MCP client"))
	b.WriteString("\n")
	b.WriteString(normalStyle.Render("  dpf                 "))
	b.WriteString(dimStyle.Render("— High-performance Rust media processing utility (images, video, audio),"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("                       invoked by the MCP server for optimization, transcoding, and generation"))
	b.WriteString("\n\n")

	b.WriteString(titleStyle.Render("DEVELOPER"))
	b.WriteString("\n")
	b.WriteString(normalStyle.Render("  Gustavo A. Gutiérrez Mercado"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  Bogotá, Colombia  © 2026"))
	b.WriteString("\n\n")

	b.WriteString(helpStyle.Render("Esc / Enter / q  go back"))

	return boxStyle.Render(b.String())
}
