// Package tui provides the Bubble Tea TUI for devforge.
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"dev-forge-mcp/internal/config"
	"dev-forge-mcp/internal/tools"
)

// View identifies which TUI view is active.
type View int

const (
	ViewHome View = iota
	ViewAnalyzeLayout
	ViewGenerateLayout
	ViewColorPalettes
	ViewGenerateImages
	ViewOptimizeImages
	ViewGenerateFavicon
	ViewVideo
	ViewAudio
	ViewUI2MD
	ViewMarkdownToPDF
	ViewTextEnc
	ViewDataFmt
	ViewCryptoutil
	ViewHTTPTools
	ViewDateTime
	ViewFileTools
	ViewFrontendTools
	ViewBackendTools
	ViewCodeTools
	ViewSettings
	ViewMCPSetup
	ViewAbout
)

// NavigateTo is a message that triggers view navigation.
type NavigateTo struct{ View View }

// Model is the root Bubble Tea model.
type Model struct {
	currentView View
	width       int
	height      int
	version     string

	// Shared dependencies
	config *config.Config
	srv    *tools.Server

	// Sub-models
	home            homeModel
	analyzeLayout   analyzeLayoutModel
	generateLayout  generateLayoutModel
	generateImages  generateImagesModel
	optimizeImages  optimizeImagesModel
	generateFavicon generateFaviconModel
	video           videoModel
	audio           audioModel
	ui2md           ui2mdModel
	markdownToPDF   markdownToPDFModel
	colorPalettes   colorPalettesModel
	settings        settingsModel
	mcpSetup        mcpSetupModel
	about           aboutModel
	textEnc         textEncModel
	dataFmt         dataFmtModel
	cryptoutil      cryptoutilModel
	httpTools       httpToolsModel
	dateTime        dateTimeModel
	fileTools       fileToolsModel
	frontendTools   frontendToolsModel
	backendTools    backendToolsModel
	codeTools       codeToolsModel

	// Detected stack
	detectedFramework string
	detectedCSSMode   string
}

// New creates the root model with all dependencies.
func New(cfg *config.Config, srv *tools.Server, framework, cssMode, ver string) Model {
	m := Model{
		currentView:       ViewHome,
		config:            cfg,
		srv:               srv,
		version:           ver,
		detectedFramework: framework,
		detectedCSSMode:   cssMode,
	}
	m.home = newHomeModel(ver)
	m.analyzeLayout = newAnalyzeLayoutModel(srv, framework, cssMode)
	m.generateLayout = newGenerateLayoutModel(srv, framework, cssMode)
	m.generateImages = newGenerateImagesModel(srv, cfg)
	m.optimizeImages = newOptimizeImagesModel(srv)
	m.generateFavicon = newGenerateFaviconModel(srv)
	m.video = newVideoModel(srv)
	m.audio = newAudioModel(srv)
	m.ui2md = newUI2MDModel(srv, cfg)
	m.markdownToPDF = newMarkdownToPDFModel(srv)
	m.colorPalettes = newColorPalettesModel(srv)
	m.settings = newSettingsModel(cfg)
	m.mcpSetup = newMCPSetupModel()
	m.about = newAboutModel(ver)
	m.textEnc = newTextEncModel(srv)
	m.dataFmt = newDataFmtModel(srv)
	m.cryptoutil = newCryptoutilModel(srv)
	m.httpTools = newHTTPToolsModel(srv)
	m.dateTime = newDateTimeModel(srv)
	m.fileTools = newFileToolsModel(srv)
	m.frontendTools = newFrontendToolsModel(srv)
	m.backendTools = newBackendToolsModel(srv)
	m.codeTools = newCodeToolsModel(srv)
	return m
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.home.Init(), tea.EnableBracketedPaste)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case NavigateTo:
		m.currentView = msg.View
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	// Delegate to sub-model
	switch m.currentView {
	case ViewHome:
		updated, cmd := m.home.Update(msg)
		m.home = updated
		if m.home.selected >= 0 {
			view := m.homeItemToView(m.home.selected)
			m.home.selected = -1
			m.currentView = view
			return m, nil
		}
		return m, cmd

	case ViewAnalyzeLayout:
		updated, cmd := m.analyzeLayout.Update(msg)
		m.analyzeLayout = updated
		if m.analyzeLayout.goHome {
			m.analyzeLayout.goHome = false
			m.currentView = ViewHome
		}
		return m, cmd

	case ViewGenerateLayout:
		updated, cmd := m.generateLayout.Update(msg)
		m.generateLayout = updated
		if m.generateLayout.goHome {
			m.generateLayout.goHome = false
			m.currentView = ViewHome
		}
		return m, cmd

	case ViewGenerateImages:
		updated, cmd := m.generateImages.Update(msg)
		m.generateImages = updated
		if m.generateImages.goHome {
			m.generateImages.goHome = false
			m.currentView = ViewHome
		}
		if m.generateImages.goSettings {
			m.generateImages.goSettings = false
			m.currentView = ViewSettings
		}
		return m, cmd

	case ViewOptimizeImages:
		updated, cmd := m.optimizeImages.Update(msg)
		m.optimizeImages = updated
		if m.optimizeImages.goHome {
			m.optimizeImages.goHome = false
			m.currentView = ViewHome
		}
		return m, cmd

	case ViewGenerateFavicon:
		updated, cmd := m.generateFavicon.Update(msg)
		m.generateFavicon = updated
		if m.generateFavicon.goHome {
			m.generateFavicon.goHome = false
			m.currentView = ViewHome
		}
		return m, cmd

	case ViewVideo:
		updated, cmd := m.video.Update(msg)
		if v, ok := updated.(videoModel); ok {
			m.video = v
		}
		if m.video.goHome {
			m.video.goHome = false
			m.currentView = ViewHome
		}
		return m, cmd

	case ViewAudio:
		updated, cmd := m.audio.Update(msg)
		if a, ok := updated.(audioModel); ok {
			m.audio = a
		}
		if m.audio.goHome {
			m.audio.goHome = false
			m.currentView = ViewHome
		}
		return m, cmd

	case ViewUI2MD:
		updated, cmd := m.ui2md.Update(msg)
		m.ui2md = updated
		if m.ui2md.goHome {
			m.ui2md.goHome = false
			m.currentView = ViewHome
		}
		return m, cmd

	case ViewMarkdownToPDF:
		updated, cmd := m.markdownToPDF.Update(msg)
		m.markdownToPDF = updated
		if m.markdownToPDF.goHome {
			m.markdownToPDF.goHome = false
			m.currentView = ViewHome
		}
		return m, cmd

	case ViewColorPalettes:
		updated, cmd := m.colorPalettes.Update(msg)
		m.colorPalettes = updated
		if m.colorPalettes.goHome {
			m.colorPalettes.goHome = false
			m.currentView = ViewHome
		}
		return m, cmd

	case ViewSettings:
		updated, cmd := m.settings.Update(msg)
		m.settings = updated
		if m.settings.goHome {
			m.settings.goHome = false
			if m.settings.saved {
				m.config = m.settings.cfg
				m.generateImages.cfg = m.config
				m.settings.saved = false
			}
			m.currentView = ViewHome
		}
		return m, cmd

	case ViewMCPSetup:
		updated, cmd := m.mcpSetup.Update(msg)
		m.mcpSetup = updated
		if m.mcpSetup.goHome {
			m.mcpSetup = newMCPSetupModel()
			m.currentView = ViewHome
		}
		return m, cmd

	case ViewAbout:
		updated, cmd := m.about.Update(msg)
		m.about = updated
		return m, cmd

	case ViewTextEnc:
		updated, cmd := m.textEnc.Update(msg)
		m.textEnc = updated
		if m.textEnc.goHome {
			m.textEnc = newTextEncModel(m.srv)
			m.currentView = ViewHome
		}
		return m, cmd

	case ViewDataFmt:
		updated, cmd := m.dataFmt.Update(msg)
		m.dataFmt = updated
		if m.dataFmt.goHome {
			m.dataFmt = newDataFmtModel(m.srv)
			m.currentView = ViewHome
		}
		return m, cmd

	case ViewCryptoutil:
		updated, cmd := m.cryptoutil.Update(msg)
		m.cryptoutil = updated
		if m.cryptoutil.goHome {
			m.cryptoutil = newCryptoutilModel(m.srv)
			m.currentView = ViewHome
		}
		return m, cmd

	case ViewHTTPTools:
		updated, cmd := m.httpTools.Update(msg)
		m.httpTools = updated
		if m.httpTools.goHome {
			m.httpTools = newHTTPToolsModel(m.srv)
			m.currentView = ViewHome
		}
		return m, cmd

	case ViewDateTime:
		updated, cmd := m.dateTime.Update(msg)
		m.dateTime = updated
		if m.dateTime.goHome {
			m.dateTime = newDateTimeModel(m.srv)
			m.currentView = ViewHome
		}
		return m, cmd

	case ViewFileTools:
		updated, cmd := m.fileTools.Update(msg)
		m.fileTools = updated
		if m.fileTools.goHome {
			m.fileTools = newFileToolsModel(m.srv)
			m.currentView = ViewHome
		}
		return m, cmd

	case ViewFrontendTools:
		updated, cmd := m.frontendTools.Update(msg)
		m.frontendTools = updated
		if m.frontendTools.goHome {
			m.frontendTools = newFrontendToolsModel(m.srv)
			m.currentView = ViewHome
		}
		return m, cmd

	case ViewBackendTools:
		updated, cmd := m.backendTools.Update(msg)
		m.backendTools = updated
		if m.backendTools.goHome {
			m.backendTools = newBackendToolsModel(m.srv)
			m.currentView = ViewHome
		}
		return m, cmd

	case ViewCodeTools:
		updated, cmd := m.codeTools.Update(msg)
		m.codeTools = updated
		if m.codeTools.goHome {
			m.codeTools = newCodeToolsModel(m.srv)
			m.currentView = ViewHome
		}
		return m, cmd
	}

	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	switch m.currentView {
	case ViewHome:
		return m.home.View()
	case ViewAnalyzeLayout:
		return m.analyzeLayout.View()
	case ViewGenerateLayout:
		return m.generateLayout.View()
	case ViewGenerateImages:
		return m.generateImages.View()
	case ViewOptimizeImages:
		return m.optimizeImages.View()
	case ViewGenerateFavicon:
		return m.generateFavicon.View()
	case ViewVideo:
		return m.video.View()
	case ViewAudio:
		return m.audio.View()
	case ViewUI2MD:
		return m.ui2md.View()
	case ViewMarkdownToPDF:
		return m.markdownToPDF.View()
	case ViewColorPalettes:
		return m.colorPalettes.View()
	case ViewSettings:
		return m.settings.View()
	case ViewMCPSetup:
		return m.mcpSetup.View()
	case ViewAbout:
		return m.about.View()
	case ViewTextEnc:
		return m.textEnc.View()
	case ViewDataFmt:
		return m.dataFmt.View()
	case ViewCryptoutil:
		return m.cryptoutil.View()
	case ViewHTTPTools:
		return m.httpTools.View()
	case ViewDateTime:
		return m.dateTime.View()
	case ViewFileTools:
		return m.fileTools.View()
	case ViewFrontendTools:
		return m.frontendTools.View()
	case ViewBackendTools:
		return m.backendTools.View()
	case ViewCodeTools:
		return m.codeTools.View()
	}
	return "Unknown view"
}

// homeItemToView maps selectable menu item index (0-based, skipping sections) to a View.
func (m Model) homeItemToView(idx int) View {
	switch idx {
	case 0:
		return ViewAnalyzeLayout
	case 1:
		return ViewGenerateLayout
	case 2:
		return ViewColorPalettes
	case 3:
		return ViewGenerateImages
	case 4:
		return ViewOptimizeImages
	case 5:
		return ViewGenerateFavicon
	case 6:
		return ViewVideo
	case 7:
		return ViewAudio
	case 8:
		return ViewUI2MD
	case 9:
		return ViewMarkdownToPDF
	case 10:
		return ViewTextEnc
	case 11:
		return ViewDataFmt
	case 12:
		return ViewCryptoutil
	case 13:
		return ViewHTTPTools
	case 14:
		return ViewDateTime
	case 15:
		return ViewFileTools
	case 16:
		return ViewFrontendTools
	case 17:
		return ViewBackendTools
	case 18:
		return ViewCodeTools
	case 19:
		return ViewSettings
	case 20:
		return ViewMCPSetup
	case 21:
		return ViewAbout
	}
	return ViewHome
}

// Shared styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")).
			Background(lipgloss.Color("18"))

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("7"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			MarginTop(1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("12")).
			Padding(1, 2)
)
