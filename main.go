package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type TabInfo struct {
	Name   string
	Number int
}

const (
	// Tabs
	homeTabTitle     = "Home"
	boardTabTitle    = "Board"
	terminalTabTitle = "Terminal"

	// Colors
	inactiveColorHex = "#555555"
	activeColorHex   = "#ed3462"
	lightColorHex    = "#874BFD"
	darkColorHex     = "#ed3462"

	// Border symbols
	tabBottomSymbol   = "─"
	columnSidesSymbol = "│"

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

	keyAdd       = "a"
	keyEnter     = "enter"
	keyEscape    = "esc"
	keyBackspace = "backspace"
	keyUp        = "up"
	keyDown      = "down"

	// Messages
	programErrorMessage = "Error running program:"
)

type task struct {
	Id     int
	Title  string
	Status taskStatus
	JiraId int
}

type taskStatus int

type shellFinishedMsg struct {
	err error
}
type taskType int

const (
	statusTodo taskStatus = iota
	statusInProgress
	statusDone
)

type styles struct {
	doc         lipgloss.Style
	inactiveTab lipgloss.Style
	activeTab   lipgloss.Style
	window      lipgloss.Style
	footer      lipgloss.Style
	homeTitle   lipgloss.Style

	boardColumn lipgloss.Style
	taskCommand lipgloss.Style

	columnTitle  lipgloss.Style
	task         lipgloss.Style
	selectedTask lipgloss.Style
	activeInput  lipgloss.Style
}

type model struct {
	tabs        []string
	styles      *styles
	currentTime time.Time
	activeTab   int
	width       int
	height      int

	tasks        []task
	selectedTask int

	commandMode  bool
	commandValue string
}

func main() {

	logFile, err := os.OpenFile(
		"debug.log",
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0666,
	)
	if err != nil {
		fmt.Println("could not open log file:", err)
		os.Exit(1)
	}
	defer logFile.Close()

	log.SetOutput(logFile)

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	initialTasks := []task{
		{
			Id:     1,
			Title:  "Create local task model",
			Status: statusTodo,
		},
		{
			Id:     2,
			Title:  "Build board layout",
			Status: statusInProgress,
		},
		{
			Id:     3,
			Title:  "Create Home page",
			Status: statusDone,
		},
	}

	tabs := []string{
		homeTabTitle,
		boardTabTitle,
		terminalTabTitle,
	}

	initialModel := model{
		tabs:        tabs,
		tasks:       initialTasks,
		styles:      newStyles(true),
		currentTime: time.Now(),
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

		footer: lipgloss.NewStyle().
			BorderForeground(highlightColor),

		homeTitle: lipgloss.NewStyle().
			Foreground(activeColor).
			Bold(true).
			Align(lipgloss.Center),

		boardColumn: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#555555")).
			Padding(1, 2),

		taskCommand: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#777777")).
			PaddingLeft(1),
		columnTitle: lipgloss.NewStyle().
			Foreground(activeColor).
			Bold(true).
			MarginBottom(1),

		task: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AAAAAA")).
			PaddingLeft(1),

		selectedTask: lipgloss.NewStyle().
			Foreground(activeColor).
			Bold(true).
			PaddingLeft(1),

		activeInput: lipgloss.NewStyle().
			Foreground(activeColor).
			Bold(true).
			PaddingLeft(1),
	}
}

func (m model) Init() tea.Cmd {
	return tickCmd()
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		m.currentTime = time.Time(msg)
		return m, tickCmd()

	case tea.KeyPressMsg:
		key := msg.String()
		if key >= "1" && key <= "9" {
			tabIndex := int(key[0] - '1')

			if tabIndex < len(m.tabs) {
				m.activeTab = tabIndex
			}

			return m, nil
		}

		if m.activeTab == 2 && key == "enter" {
			return m, openShell()
		}

		if m.commandMode {
			return m.updateCommand(msg)
		}

		switch key {
		case keyCtrlC, keyQuit:
			return m, tea.Quit
		//case keyRight, keyVimRight, keyNext, keyTab:
		//	m.activeTab = min(
		//		m.activeTab+1,
		//		len(m.tabs)-1,
		//	)
		//
		//case keyLeft, keyVimLeft, keyPrevious, keyShiftTab:
		//	m.activeTab = max(
		//		m.activeTab-1,
		//		0,
		//	)
		case "a":
			if !m.commandMode && m.activeTab == 1 {
				m.commandMode = true
				m.commandValue = "add "
			}
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

		renderedTabs = append(
			renderedTabs,
			tabStyle.Render(fmt.Sprintf("(%d) %s", tabIndex+1, tabTitle)),
		)
	}

	tabsRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		renderedTabs...,
	)

	windowWidth := max(m.width-windowWidthOffset, 0)
	windowHeight := max(m.height-windowHeightOffset, 0)

	innerWidth := max(windowWidth-windowPaddingHorizontal*2, 0)
	innerHeight := max(windowHeight-windowPaddingVertical*2, 0)

	activeContent := "None"

	switch m.activeTab {
	case 0:
		activeContent = m.renderHome(innerWidth, innerHeight)

	case 1:
		activeContent = m.renderBoard(innerWidth, innerHeight)

	case 2:
		activeContent = m.renderTerminal(innerWidth, innerHeight)
	}

	renderedWindow := m.styles.window.
		Width(windowWidth).
		Height(windowHeight).
		Render(activeContent)

	footerText := m.currentTime.Format("2006-01-02 15:04:05")

	renderedFooter := m.styles.footer.
		Width(lipgloss.Width(renderedWindow)).
		Align(lipgloss.Right).
		Render(footerText)

	document.WriteString(tabsRow)
	document.WriteString("\n")
	document.WriteString(renderedWindow)
	document.WriteString("\n")
	document.WriteString(renderedFooter)

	view := tea.NewView(
		m.styles.doc.Render(document.String()),
	)

	view.AltScreen = true

	return view
}

func (m model) renderBoard(width, height int) string {
	columnGap := 2
	inputHeight := 1
	columnCount := 3

	boardHeight := max(height-inputHeight-2, 0)
	columnWidth := max((width-columnGap*2)/columnCount, 0)

	todoContent := "TODO\n\n" + m.renderTasksByStatus(statusTodo)

	todoColumn := m.styles.boardColumn.
		Border(columnsBorder()).
		Width(columnWidth).
		Height(boardHeight).
		Align(lipgloss.Left).
		Render(todoContent)

	inProgressContent := "IN PROGRESS\n\n" + m.renderTasksByStatus(statusInProgress)

	inProgressColumn := m.styles.boardColumn.
		Border(columnsBorder()).
		Width(columnWidth).
		Height(boardHeight).
		Align(lipgloss.Left).
		Render(inProgressContent)

	doneContent := "DONE\n\n" + m.renderTasksByStatus(statusDone)
	doneColumn := m.styles.boardColumn.
		Border(columnsBorder()).
		Width(columnWidth).
		Height(boardHeight).
		Align(lipgloss.Left).
		Render(doneContent)

	board := lipgloss.JoinHorizontal(
		lipgloss.Top,
		todoColumn,
		inProgressColumn,
		doneColumn,
	)

	commandText := "> Task Command"

	if m.commandMode {
		commandText = "> " + m.commandValue + "█"
	}
	log.Printf(m.commandValue)

	commandBar := m.styles.taskCommand.
		Width(width).
		Render(commandText)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		board,
		commandBar,
	)
}

func (m model) renderHome(width int, height int) string {
	return m.styles.homeTitle.
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Render("Buongiorno, principessa!")
}

func tabBorderWithBottom(middleSymbol string) lipgloss.Border {
	border := lipgloss.NormalBorder()

	border.Bottom = middleSymbol

	return border
}

func (m model) renderTasksByStatus(status taskStatus) string {
	var content strings.Builder

	for _, currentTask := range m.tasks {

		if currentTask.Status != status {
			continue
		}
		jiraLabel := "LOCAL"

		if currentTask.JiraId != 0 {
			jiraLabel = fmt.Sprintf("%d", currentTask.JiraId)
		}

		content.WriteString("• ")
		content.WriteString(currentTask.Title)
		content.WriteString(" [")
		content.WriteString(jiraLabel)
		content.WriteString("]\n")
	}
	if content.Len() == 0 {
		return "No tasks"
	}

	return strings.TrimSuffix(content.String(), "\n")
}

func (m model) renderTerminal(width, height int) string {
	content := `
Press Enter to open your system shell.

Type "exit" to return.
`

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content)
}

func (m model) updateCommand(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	log.Printf(msg.String())
	switch msg.String() {
	case "esc":
		m.commandMode = false
		m.commandValue = ""
		return m, nil

	case "enter":
		m.executeCommand()

		m.commandMode = false
		m.commandValue = ""
		return m, nil

	case "backspace":
		runes := []rune(m.commandValue)

		if len(runes) > 0 {
			m.commandValue = string(runes[:len(runes)-1])
		}

		return m, nil
	}

	if msg.Text != "" {
		log.Printf("im in")
		m.commandValue += msg.Text
	}

	return m, nil
}
func (m *model) executeCommand() {
	command := strings.TrimSpace(m.commandValue)

	if !strings.HasPrefix(command, "add ") {
		return
	}

	taskTitle := strings.TrimSpace(
		strings.TrimPrefix(command, "add "),
	)

	if taskTitle == "" {
		return
	}

	m.tasks = append(m.tasks, task{
		Title:  taskTitle,
		Status: statusTodo,
	})
}
func columnsBorder() lipgloss.Border {
	border := lipgloss.HiddenBorder()

	border.Left = columnSidesSymbol
	border.Bottom = ""
	border.Right = columnSidesSymbol
	border.Top = ""

	return border
}

func openShell() tea.Cmd {
	var shell *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		shell = exec.Command("powershell.exe")

	default:
		shellPath := os.Getenv("SHELL")

		if shellPath == "" {
			shellPath = "/bin/bash"
		}

		shell = exec.Command(shellPath)
	}

	return tea.ExecProcess(shell, func(err error) tea.Msg {
		return shellFinishedMsg{err: err}
	})
}
