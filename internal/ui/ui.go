package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eliran89c/klama/internal/agent"
	"github.com/eliran89c/klama/internal/executer"
	"github.com/eliran89c/klama/internal/logger"
)

type (
	modelState int
	errMsg     error
	tickMsg    time.Time
)

const (
	StateTyping modelState = iota
	StateAsking
	StateExecuting
	StateWaitingForConfirmation
)

const (
	colorSender     = "2"   // green
	colorKlama      = "5"   // magenta
	colorSystem     = "3"   // yellow
	colorError      = "1"   // red
	colorHelp       = "241" // light gray
	colorPrice      = "6"   // cyan
	colorBackground = "0"   // black

	welcomeMsg = "Welcome to Klama!\nEnter your question or issue."
)

var (
	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.BorderStyle(b)
	}()
)

// Agent represents the interface for interacting with an AI agent.
type Agent interface {
	Iterate(context.Context, string) (agent.AgentResponse, error)
	Reset()
	LogUsage() string
}

// Executer represents the interface for executing commands.
type Executer interface {
	Run(context.Context, string) executer.ExecuterResponse
	Validate(string) error
}

// Model represents the application state.
type Model struct {
	agent    Agent
	executer Executer
	ready    bool

	viewport viewport.Model
	textarea textarea.Model

	senderStyle lipgloss.Style
	klamaStyle  lipgloss.Style
	systemStyle lipgloss.Style
	errorStyle  lipgloss.Style
	helpStyle   lipgloss.Style
	priceStyle  lipgloss.Style
	typingStyle lipgloss.Style

	messages        []string
	err             error
	state           modelState
	waitingDots     int
	confirmationCmd string
	showCmdResponse bool

	width  int
	height int

	ctx    context.Context
	cancel context.CancelFunc
}

// Config holds the configuration for initializing the Model.
type Config struct {
	Agent    Agent
	Executer Executer
}

// InitialModel creates and returns a new instance of Model with default values.
func InitialModel(cfg Config) Model {
	logger.Debug("Initializing UI model")

	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()
	ta.Prompt = "┃ "
	ta.CharLimit = 280
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)
	ta.SetHeight(3)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	vp := viewport.New(80, 20)
	vp.SetContent(welcomeMsg)

	ctx, cancel := context.WithCancel(context.Background())

	newStyle := func(color string) lipgloss.Style {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	}

	return Model{
		agent:       cfg.Agent,
		executer:    cfg.Executer,
		textarea:    ta,
		viewport:    vp,
		messages:    []string{},
		senderStyle: newStyle(colorSender),
		klamaStyle:  newStyle(colorKlama),
		systemStyle: newStyle(colorSystem),
		errorStyle:  newStyle(colorError),
		helpStyle:   newStyle(colorHelp),
		priceStyle:  newStyle(colorPrice),
		typingStyle: newStyle(colorHelp),
		ctx:         ctx,
		cancel:      cancel,
		state:       StateTyping,
	}
}

// Init initializes the Model.
func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

// View renders the current state of the application.
func (m Model) View() string {
	return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
}

func (m Model) headerView() string {
	title := titleStyle.Render("Klama")
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m Model) footerView() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	border := lipgloss.JoinHorizontal(lipgloss.Center, line, info)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		border,
		m.renderInputArea(),
		m.renderErrorMessage(),
		m.renderHelpText(),
		m.renderPriceText(),
	)
}

func (m Model) renderInputArea() string {
	switch m.state {
	case StateAsking:
		return m.typingStyle.Render("\n\nKlama is typing" + strings.Repeat(".", m.waitingDots))
	case StateExecuting:
		return m.typingStyle.Render("\n\nCommand executing" + strings.Repeat(".", m.waitingDots))
	default:
		return m.textarea.View()
	}
}

func (m Model) renderErrorMessage() string {
	if m.err != nil {
		return m.errorStyle.Render("Error: " + m.err.Error())
	}
	return ""
}

func (m Model) renderHelpText() string {
	var helpText string
	if m.showCmdResponse {
		helpText += "Ctrl+S: to hide command response."
	} else {
		helpText += "Ctrl+S: to show command response."
	}

	helpText += "\nCtrl+C: to exit, Ctrl+R: to restart. Scroll with ↑, ↓, Page Up, Page Down, and mouse wheel."

	return m.helpStyle.Width(m.width).Render(helpText)
}

func (m Model) renderPriceText() string {
	return m.priceStyle.Width(m.width).Render(m.agent.LogUsage())
}

func (m *Model) updateChat(style lipgloss.Style, prefix, message string) {
	m.messages = append(m.messages, style.Render(prefix+": ")+message)
	m.updateViewportContent()

}

func (m *Model) updateViewportContent() {
	content := lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.messages, "\n\n"))
	m.viewport.SetContent(content)
	m.textarea.Reset()
	m.viewport.GotoBottom()
}

// Update handles all the application logic and state transitions.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSizeMsg(msg)

	case tea.MouseMsg:
		return m.handleMouseMsg(msg)

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tickMsg:
		m.waitingDots = (m.waitingDots + 1) % 4
		return m, m.think()

	case agent.AgentResponse:
		return m.handleAgentResponse(msg)

	case executer.ExecuterResponse:
		return m.handleExecuterResponse(msg)

	case errMsg:
		m.err = msg
		if m.state == StateAsking || m.state == StateExecuting {
			m.state = StateTyping
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	logger.Debugf("Window size message received: %v\n", msg)

	m.width = msg.Width
	m.height = msg.Height

	headerHeight := lipgloss.Height(m.headerView())
	footerHeight := lipgloss.Height(m.footerView())

	m.viewport.Width = msg.Width - 2
	m.viewport.Height = msg.Height - headerHeight - footerHeight
	m.textarea.SetWidth(msg.Width - 2)

	// update chat history if the session is on `ready` state
	if m.ready {
		m.updateViewportContent()
	}

	return m, nil
}

func (m Model) handleMouseMsg(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp, tea.KeyDown, tea.KeyPgUp, tea.KeyPgDown:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	case tea.KeyCtrlC, tea.KeyEsc:
		m.cancel()
		return m, tea.Quit

	case tea.KeyCtrlR:
		logger.Debug("Restarting the session")
		m.cancel()
		m.agent.Reset()
		newModel := InitialModel(Config{
			Agent:    m.agent,
			Executer: m.executer,
		})
		return newModel.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})

	case tea.KeyCtrlS:
		logger.Debug("Toggling command response visibility")
		m.showCmdResponse = !m.showCmdResponse
		return m, nil

	case tea.KeyEnter:
		return m.handleEnterKey()

	default:
		if m.state == StateTyping || m.state == StateWaitingForConfirmation {
			m.err = nil
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m Model) handleEnterKey() (tea.Model, tea.Cmd) {
	if !m.ready {
		m.ready = true
	}

	switch m.state {
	case StateTyping:
		query := m.textarea.Value()
		if query == "" {
			m.err = fmt.Errorf("message cannot be empty")
			return m, nil
		}
		m.updateChat(m.senderStyle, "You", query)
		m.state = StateAsking
		return m, tea.Batch(
			m.waitForAgentResponse(query),
			m.think(),
		)

	case StateWaitingForConfirmation:
		return m.handleConfirmation()
	}

	return m, nil
}

func (m Model) handleConfirmation() (tea.Model, tea.Cmd) {
	userInput := strings.TrimSpace(strings.ToLower(m.textarea.Value()))

	switch userInput {
	case "yes", "y":
		m.state = StateExecuting
		m.updateChat(m.systemStyle, "System", fmt.Sprintf("Executing command `%v`", m.systemStyle.Render(m.confirmationCmd)))
		return m, tea.Batch(
			m.waitForExecution(m.confirmationCmd),
			m.think(),
		)

	case "no", "n":
		m.state = StateAsking
		rejectMsg := "User did not approve the command. Please suggest a different command or end the session."
		m.updateChat(m.systemStyle, "System", rejectMsg)
		return m, tea.Batch(
			m.waitForAgentResponse(rejectMsg),
			m.think(),
		)

	case "ask", "a":
		m.state = StateTyping
		m.updateChat(m.systemStyle, "System", "Breaking out to ask a question")
		return m, nil

	default:
		m.err = fmt.Errorf("please answer with 'yes', 'no', or 'ask'")
		m.textarea.Reset()
		return m, nil
	}
}

func (m Model) handleAgentResponse(msg agent.AgentResponse) (tea.Model, tea.Cmd) {
	m.state = StateTyping
	if msg.RunCommand != "" {
		logger.Debugf("Agent suggested a command to run: `%v`\n", msg.RunCommand)
		// validate the command
		if err := m.executer.Validate(msg.RunCommand); err != nil {
			logger.Debug(err)
			// command is invalid, return to the agent
			prompt := fmt.Sprintf("The suggested command is invalid: %v\nDo not apologize or mention the incorrect suggestion in your response", err)
			m.state = StateAsking
			return m, tea.Batch(
				m.waitForAgentResponse(prompt),
				m.think(),
			)
		}

		m.state = StateWaitingForConfirmation
		m.confirmationCmd = msg.RunCommand

		var klamaResp string
		if msg.Answer != "" {
			klamaResp += msg.Answer + "\n"
		}
		klamaResp += "I suggest running the command `" + m.systemStyle.Render(msg.RunCommand)
		klamaResp += fmt.Sprintf("`\n%v", msg.Reason)

		m.updateChat(m.klamaStyle, "Klama", klamaResp)
		m.updateChat(m.systemStyle, "System", "Enter 'yes' to approve, 'no' to reject, or 'ask' to break out and ask a question.")
	} else {
		m.updateChat(m.klamaStyle, "Klama", msg.Answer)
	}

	return m, nil
}

func (m Model) handleExecuterResponse(msg executer.ExecuterResponse) (tea.Model, tea.Cmd) {
	m.state = StateAsking
	var systemResponse string

	if msg.Error != nil {
		systemResponse = fmt.Sprintf("Error executing command: %v\n%v\nFOLLOW YOUR GUIDELINES", msg.Error.Error(), msg.Result)
	} else {
		systemResponse = fmt.Sprintf("Command output:\n%v", msg.Result)
	}

	if m.showCmdResponse {
		m.updateChat(m.systemStyle, "System", systemResponse)
	}

	return m, tea.Batch(
		m.waitForAgentResponse(systemResponse),
		m.think(),
	)
}

func (m Model) waitForAgentResponse(userMessage string) tea.Cmd {
	return func() tea.Msg {
		//TODO: get timeout from config
		ctx, cancel := context.WithTimeout(m.ctx, 90*time.Second)
		defer cancel()

		response, err := m.agent.Iterate(ctx, userMessage)
		if err != nil {
			return errMsg(err)
		}
		return response
	}
}

func (m Model) waitForExecution(command string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
		defer cancel()

		return m.executer.Run(ctx, command)
	}
}

func (m Model) think() tea.Cmd {
	return tea.Tick(time.Millisecond*300, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
