package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"sort"
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
	IsMine bool
	Order  int
	JiraId int
}

type taskStatus int

type shellFinishedMsg struct {
	err error
}

const (
	FastCheck taskStatus = (iota)
	WaitMe
	Test
	Review
	Meeting
	Done
	None
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

	task        lipgloss.Style
	statusTitle lipgloss.Style
	taskItem    lipgloss.Style
	emptyTask   lipgloss.Style
	mineLabel   lipgloss.Style
	otherLabel  lipgloss.Style

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

	tasks          []task
	selectedTaskId int

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
			Status: Done,
			IsMine: true,
			Order:  0,
		},
		{
			Id:     2,
			Title:  "Build board layout",
			Status: FastCheck,
			IsMine: true,
			Order:  1,
		},
		{
			Id:     3,
			Title:  "Build board test",
			Status: FastCheck,
			IsMine: true,
			Order:  0,
		},
		{
			Id:     4,
			Title:  "Create Home page",
			Status: Test,
			IsMine: false,
			Order:  0,
		},
		{
			Id:     5,
			Title:  "meeting",
			Status: Meeting,
			IsMine: true,
			Order:  0,
		},
	}

	tabs := []string{
		homeTabTitle,
		boardTabTitle,
		terminalTabTitle,
	}

	initialModel := model{
		tabs:           tabs,
		tasks:          initialTasks,
		styles:         newStyles(true),
		currentTime:    time.Now(),
		selectedTaskId: 3,
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

		task: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AAAAAA")).
			PaddingLeft(1),

		statusTitle: lipgloss.NewStyle().
			Foreground(activeColor).
			Bold(true),

		taskItem: lipgloss.NewStyle().
			PaddingLeft(2),

		emptyTask: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555")).
			PaddingLeft(2),

		mineLabel: lipgloss.NewStyle().
			Foreground(activeColor).
			Bold(true),

		otherLabel: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#777777")),

		activeInput: lipgloss.NewStyle().
			Foreground(activeColor).
			Bold(true).
			PaddingLeft(1),
	}
}

func (m model) Init() tea.Cmd {
	return nil
	//return tickCmd()
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
		if m.activeTab == 1 && key == keyDown {
			m.moveNextTask()
			return m, nil
		}

		if m.activeTab == 1 && key == keyUp {
			m.movePreviousTask()
			return m, nil
		}

		if m.activeTab == 2 && key == keyEnter {
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
	taskStatusOrder := []taskStatus{
		FastCheck,
		WaitMe,
		Test,
		Review,
		Meeting,
		Done,
		None,
	}

	var sections []string

	for _, status := range taskStatusOrder {

		var statusTasks []task
		var taskLines []string

		for _, currentTask := range m.tasks {
			if currentTask.Status == status {
				statusTasks = append(statusTasks, currentTask)
			}
		}

		sort.Slice(statusTasks, func(i, j int) bool {
			return statusTasks[i].Order < statusTasks[j].Order
		})

		for _, currentTask := range statusTasks {
			if currentTask.Status != status {
				continue
			}
			log.Printf("Task status: %s", currentTask.Status)
			log.Printf("Task Order: %d", currentTask.Order)

			taskTitle := currentTask.Title
			if currentTask.Id == m.selectedTaskId {
				taskTitle = "> " + taskTitle
			}

			ownerLabel := m.styles.otherLabel.Render("OTHER")

			if currentTask.IsMine {
				ownerLabel = m.styles.mineLabel.Render("ME")
			}

			taskLine := m.styles.taskItem.Render(
				fmt.Sprintf(
					"%s [%s]",
					taskTitle,
					ownerLabel,
				),
			)

			taskLines = append(taskLines, taskLine)

		}

		if len(taskLines) == 0 {
			continue
		}

		section := lipgloss.JoinVertical(
			lipgloss.Left,
			m.styles.statusTitle.Render(statusTitle(status)),
			lipgloss.JoinVertical(lipgloss.Left, taskLines...),
		)

		sections = append(sections, section)
	}

	taskList := lipgloss.NewStyle().
		Width(width).
		Height(max(height-2, 0)).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				sections...,
			),
		)

	commandText := "> Task Command"

	if m.commandMode {
		commandText = "> " + m.commandValue + "█"
	}

	commandBar := m.styles.taskCommand.
		Width(width).
		Render(commandText)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		taskList,
		commandBar,
	)
}

func statusTitle(status taskStatus) string {
	switch status {
	case FastCheck:
		return "Fast-Check"
	case WaitMe:
		return "Wait-Me"
	case Test:
		return "Test"
	case Review:
		return "Review"
	case Meeting:
		return "Meeting"
	case Done:
		return "Done"
	default:
		return "None"
	}
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
		Status: FastCheck,
		Order:  m.getLastTaskOrder(FastCheck) + 1,
		IsMine: true,
	})
}
func (m model) getLastTaskOrder(status taskStatus) int {
	maxOrder := 0
	for _, t := range m.tasks {

		if t.Status == status && t.Order > maxOrder {
			maxOrder = t.Order
		}
	}
	return maxOrder
}
func (m *model) moveNextTask() {
	if len(m.tasks) < 1 {
		return
	}

	var currentTask task
	for _, t := range m.tasks {
		if m.selectedTaskId == t.Id {
			currentTask = t
		}
	}

	nextTask := currentTask
	for _, t := range m.tasks {
		if t.Order > currentTask.Order && t.Status == currentTask.Status {
			nextTask = t
		}
	}

	if nextTask.Order != currentTask.Order {
		m.selectedTaskId = nextTask.Id
		return
	}

	nextStatus := currentTask.Status + 1

	taskStatusOrder := []taskStatus{
		FastCheck,
		WaitMe,
		Test,
		Review,
		Meeting,
		Done,
		None,
	}

	for {
		for _, status := range taskStatusOrder {
			if status < nextStatus {
				continue
			}

			var statusTasks []task

			for _, t := range m.tasks {
				if t.Status == status {
					statusTasks = append(statusTasks, t)
				}
			}

			sort.Slice(statusTasks, func(i, j int) bool {
				return statusTasks[i].Order < statusTasks[j].Order
			})
			log.Printf("Moving task %s", statusTasks)

			if len(statusTasks) > 0 {
				nextTask = statusTasks[0]
				m.selectedTaskId = nextTask.Id
				return
			}
		}
		nextStatus = FastCheck

	}
}

func (m *model) movePreviousTask() {
	if len(m.tasks) < 1 {
		return
	}
	var currentTask task

	for _, t := range m.tasks {
		if m.selectedTaskId == t.Id {
			currentTask = t
			break
		}
	}

	previousTask := currentTask
	for _, t := range m.tasks {
		if t.Order < currentTask.Order && t.Status == currentTask.Status {
			previousTask = t
		}
	}

	if previousTask.Id != currentTask.Id {
		m.selectedTaskId = previousTask.Id
		return
	}

	previousStatus := currentTask.Status - 1

	taskStatusOrder := []taskStatus{
		FastCheck,
		WaitMe,
		Test,
		Review,
		Meeting,
		Done,
		None,
	}

	for {
		for index := len(taskStatusOrder) - 1; index >= 0; index-- {
			status := taskStatusOrder[index]

			if status > previousStatus {
				continue
			}

			var statusTasks []task

			for _, t := range m.tasks {
				if t.Status == status {
					statusTasks = append(statusTasks, t)
				}
			}

			sort.Slice(statusTasks, func(i, j int) bool {
				return statusTasks[i].Order > statusTasks[j].Order
			})

			if len(statusTasks) > 0 {
				m.selectedTaskId = statusTasks[0].Id
				return
			}
		}

		previousStatus = None
	}
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
