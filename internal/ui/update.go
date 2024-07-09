package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/eliran89c/klama/internal/app/types"
)

// Update handles all the application logic and state transitions.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	if m.waitingForConfirmation {
		m.confirmationInput, cmd = m.confirmationInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	if !m.typing && !m.executing {
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSizeMsg(msg)
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case tickMsg:
		return m.handleTickMsg()
	case types.AgentResponse:
		return m.handleResponseMsg(msg)
	case types.ExecuterResponse:
		return m.handleExecutionResponse(msg)
	case errMsg:
		m.err = msg
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.windowWidth = msg.Width
	m.windowHeight = msg.Height
	m.viewport.Width = msg.Width
	m.viewport.Height = msg.Height - 7
	m.textarea.SetWidth(msg.Width - 4)
	m.textarea.SetHeight(3)
	m.confirmationInput.Width = msg.Width - 4
	if !m.textarea.Focused() && !m.typing {
		m.textarea.Focus()
	}
	return m, nil
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		return m.handleExit()
	case tea.KeyCtrlR:
		return m.handleReset()
	case tea.KeyEnter:
		return m.handleEnterKey()
	default:
		if !m.typing && !m.executing {
			m.err = nil
		}
	}
	return m, nil
}

func (m Model) handleExit() (tea.Model, tea.Cmd) {
	m.cancel()
	return m, tea.Quit
}

func (m Model) handleReset() (tea.Model, tea.Cmd) {
	m.cancel()
	m.agent.Reset()
	newModel := InitialModel(Config{
		Agent:    m.agent,
		Executer: m.executer,
		Debug:    m.debug,
	})
	newModel.windowWidth = m.windowWidth
	newModel.windowHeight = m.windowHeight
	return newModel.Update(tea.WindowSizeMsg{Width: m.windowWidth, Height: m.windowHeight})
}

func (m Model) handleEnterKey() (tea.Model, tea.Cmd) {
	if m.typing || m.executing {
		return m, nil
	}
	if m.waitingForConfirmation {
		return m.handleConfirmation(m.confirmationInput.Value())
	}

	userMessage := strings.TrimSpace(m.textarea.Value())
	if userMessage == "" {
		m.err = fmt.Errorf("message cannot be empty")
		return m, nil
	}
	m.err = nil

	m.messages = append(m.messages, m.senderStyle.Render("You: ")+userMessage)
	m.textarea.Reset()
	m.typing = true
	m.typingDots = 0
	m.updateMessages()
	return m, tea.Batch(
		m.waitForResponse(userMessage),
		m.showWaitingAnimation(),
	)
}

func (m Model) handleTickMsg() (tea.Model, tea.Cmd) {
	if m.typing {
		m.typingDots = (m.typingDots + 1) % 4
		m.updateMessages()
		return m, m.showWaitingAnimation()
	}
	if m.executing {
		m.executingDots = (m.executingDots + 1) % 4
		m.updateMessages()
		return m, m.showWaitingAnimation()
	}
	return m, nil
}

func (m Model) handleResponseMsg(msg types.AgentResponse) (tea.Model, tea.Cmd) {
	m.typing = false

	if msg.RunCommand != "" {
		m.waitingForConfirmation = true
		m.confirmationCmd = msg.RunCommand
		m.messages = append(m.messages, m.klamaStyle.Render("Klama: ")+fmt.Sprintf("I suggest running the command "+m.systemStyle.Render(msg.RunCommand)+"\n%v", msg.Reason))
		m.updateMessages()
		m.confirmationInput.Focus()
		m.textarea.Blur()
		return m, textinput.Blink
	} else {
		m.messages = append(m.messages, m.klamaStyle.Render("Klama: ")+msg.Answer)
		m.updateMessages()
		m.textarea.Focus()
		return m, nil
	}
}

func (m Model) handleExecutionResponse(msg types.ExecuterResponse) (tea.Model, tea.Cmd) {
	m.executing = false
	var systemResponse string

	if msg.Error != nil {
		systemResponse = fmt.Sprintf("Failed to run the command\nError Message:%v\nCommand Output:%v", msg.Error.Error(), msg.Result)
	} else {
		systemResponse = fmt.Sprintf("Command Output:\n%v", msg.Result)
	}

	if m.debug {
		m.messages = append(m.messages, m.systemStyle.Render("System: ")+systemResponse)
	}

	m.typing = true
	return m, tea.Batch(
		m.waitForResponse(systemResponse),
		m.showWaitingAnimation(),
	)
}

func (m Model) handleConfirmation(userMessage string) (tea.Model, tea.Cmd) {
	userMessage = strings.TrimSpace(strings.ToLower(userMessage))

	var callback func(string) tea.Cmd
	var message string

	switch userMessage {
	case "yes", "y":
		m.messages = append(m.messages, m.senderStyle.Render("You: ")+"yes...")
		m.executing = true
		m.executingDots = 0
		callback = m.waitForExecution
		message = m.confirmationCmd
	case "no", "n":
		m.messages = append(m.messages, m.senderStyle.Render("You: ")+"no")
		m.typing = true
		m.typingDots = 0
		callback = m.waitForResponse
		message = "User did not approve the command, please suggest different command or end the session."
	default:
		m.err = fmt.Errorf("please answer with 'yes' or 'no'")
		m.confirmationInput.SetValue("")
		return m, textinput.Blink
	}

	m.waitingForConfirmation = false
	m.updateMessages()
	m.confirmationInput.Blur()
	m.confirmationInput.SetValue("")
	m.textarea.Focus()
	m.confirmationCmd = ""

	return m, tea.Batch(
		callback(message),
		m.showWaitingAnimation(),
	)
}

func (m Model) waitForResponse(userMessage string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
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
