package main

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	// Tabs
	homeTabTitle    = "Home"
	boardTabTitle   = "Board"
	homeTabContent  = "Home"
	boardTabContent = "Board"

	// Colors
	inactiveColorHex = "#555555"
	activeColorHex   = "#ed3462"
	lightColorHex    = "#874BFD"
	darkColorHex     = "#ed3462"

	// Border symbols
	tabBottomSymbol = "─"

	// Layout
	docPaddingTop    = 2
	docPaddingRight  = 2
	docPaddingBottom = 1
	docPaddingLeft   = 2

	tabPaddingVertical   = 0
	tabPaddingHorizontal = 1

	windowPaddingVertical   = 2
	windowPaddingHorizontal = 4

	windowWidthOffset  = 10
	windowHeightOffset = 10

	// Navigation keys
	keyQuit     = "q"
	keyCtrlC    = "ctrl+c"
	keyRight    = "right"
	keyLeft     = "left"
	keyVimRight = "l"
	keyVimLeft  = "h"
	keyNext     = "n"
	keyPrevious = "p"
	keyTab      = "tab"
	keyShiftTab = "shift+tab"

	// Messages
	programErrorMessage = "Error running program:"
)

type styles struct {
	doc         lipgloss.Style
	inactiveTab lipgloss.Style
	activeTab   lipgloss.Style
	window      lipgloss.Style
}

type model struct {
	tabs       []string
	tabContent []string
	styles     *styles
	activeTab  int
	width      int
	height     int
}

func main() {
	tabs := []string{
		homeTabTitle,
		boardTabTitle,
	}

	tabContent := []string{
		homeTabContent,
		boardTabContent,
	}

	initialModel := model{
		tabs:       tabs,
		tabContent: tabContent,
		styles:     newStyles(true),
	}

	program := tea.NewProgram(initialModel)

	if _, err := program.Run(); err != nil {
		fmt.Println(programErrorMessage, err)
		os.Exit(1)
	}
}

func newStyles(backgroundIsDark bool) *styles {
	lightDark := lipgloss.LightDark(backgroundIsDark)

	inactiveColor := lipgloss.Color(inactiveColorHex)
	activeColor := lipgloss.Color(activeColorHex)

	highlightColor := lightDark(
		lipgloss.Color(lightColorHex),
		lipgloss.Color(darkColorHex),
	)

	tabsBorder := tabBorderWithBottom(tabBottomSymbol)

	return &styles{
		doc: lipgloss.NewStyle().
			Padding(
				docPaddingTop,
				docPaddingRight,
				docPaddingBottom,
				docPaddingLeft,
			),

		inactiveTab: lipgloss.NewStyle().
			Border(tabsBorder, true).
			BorderForeground(inactiveColor).
			Padding(
				tabPaddingVertical,
				tabPaddingHorizontal,
			),

		activeTab: lipgloss.NewStyle().
			Border(tabsBorder, true).
			BorderForeground(activeColor).
			Foreground(activeColor).
			Bold(true).
			Padding(
				tabPaddingVertical,
				tabPaddingHorizontal,
			),

		window: lipgloss.NewStyle().
			BorderForeground(highlightColor).
			Padding(
				windowPaddingVertical,
				windowPaddingHorizontal,
			).
			Align(lipgloss.Left).
			Border(lipgloss.NormalBorder()),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyPressMsg:
		switch msg.String() {
		case keyCtrlC, keyQuit:
			return m, tea.Quit

		case keyRight, keyVimRight, keyNext, keyTab:
			m.activeTab = min(
				m.activeTab+1,
				len(m.tabs)-1,
			)

		case keyLeft, keyVimLeft, keyPrevious, keyShiftTab:
			m.activeTab = max(
				m.activeTab-1,
				0,
			)
		}
	}

	return m, nil
}

func (m model) View() tea.View {
	if m.styles == nil {
		return tea.NewView("")
	}

	var document strings.Builder
	var renderedTabs []string

	for tabIndex, tabTitle := range m.tabs {
		tabStyle := m.styles.inactiveTab

		if tabIndex == m.activeTab {
			tabStyle = m.styles.activeTab
		}

		renderedTab := tabStyle.Render(tabTitle)
		renderedTabs = append(renderedTabs, renderedTab)
	}

	tabsRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		renderedTabs...,
	)

	windowWidth := max(m.width-windowWidthOffset, 0)
	windowHeight := max(m.height-windowHeightOffset, 0)

	activeContent := m.tabContent[m.activeTab]

	renderedWindow := m.styles.window.
		Width(windowWidth).
		Height(windowHeight).
		Render(activeContent)

	document.WriteString(tabsRow)
	document.WriteString("\n")
	document.WriteString(renderedWindow)

	view := tea.NewView(
		m.styles.doc.Render(document.String()),
	)

	view.AltScreen = true

	return view
}

func tabBorderWithBottom(middleSymbol string) lipgloss.Border {
	border := lipgloss.RoundedBorder()

	border.Bottom = middleSymbol

	return border
}
