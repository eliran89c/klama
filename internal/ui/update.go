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

	// Update viewport
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	// Update confirmation input if waiting for confirmation
	if m.waitingForConfirmation {
		m.confirmationInput, cmd = m.confirmationInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update textarea if not typing
	if !m.typing {
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
		if !m.typing {
			m.errorMsg = ""
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
	newModel := InitialModel(m.agent, m.executer, m.debug)
	newModel.windowWidth = m.windowWidth
	newModel.windowHeight = m.windowHeight
	return newModel.Update(tea.WindowSizeMsg{Width: m.windowWidth, Height: m.windowHeight})
}

func (m Model) handleEnterKey() (tea.Model, tea.Cmd) {
	if m.typing {
		return m, nil
	}
	if m.waitingForConfirmation {
		return m.handleConfirmation(m.confirmationInput.Value())
	}

	userMessage := strings.TrimSpace(m.textarea.Value())
	if userMessage == "" {
		m.errorMsg = "Message cannot be empty"
		return m, nil
	}
	m.errorMsg = ""

	m.messages = append(m.messages, m.senderStyle.Render("You: ")+userMessage)
	m.textarea.Reset()
	m.typing = true
	m.typingDots = 0
	m.updateMessages()
	return m, tea.Batch(
		m.waitForResponse(userMessage),
		m.showTypingAnimation(),
	)
}

func (m Model) handleTickMsg() (tea.Model, tea.Cmd) {
	if m.typing {
		m.typingDots = (m.typingDots + 1) % 4
		m.updateMessages()
		return m, m.showTypingAnimation()
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

func (m Model) handleConfirmation(userMessage string) (tea.Model, tea.Cmd) {
	userMessage = strings.TrimSpace(strings.ToLower(userMessage))
	execResponse := types.ExecuterResponse{
		Result: "User did not approve the command, please suggest different command or end the session.",
	}

	switch userMessage {
	case "yes", "y":
		execResponse = m.executer.Run(m.ctx, m.confirmationCmd)
		m.messages = append(m.messages, m.senderStyle.Render("You: ")+"yes...")
	case "no", "n":
		m.messages = append(m.messages, m.senderStyle.Render("You: ")+"no")
	default:
		m.errorMsg = "Please answer with 'yes' or 'no'."
		m.confirmationInput.SetValue("")
		return m, textinput.Blink
	}

	// constract the response message
	var systemResponse string
	if execResponse.Error != nil {
		systemResponse = fmt.Sprintf("Failed to run the command\nError Message:%v\nCommand Output:%v", execResponse.Error.Error(), execResponse.Result)
	} else {
		systemResponse = fmt.Sprintf("Command Output:\n%v", execResponse.Result)
	}

	if m.debug {
		m.messages = append(m.messages, m.systemStyle.Render("System: ")+systemResponse)
	}

	m.waitingForConfirmation = false
	m.updateMessages()
	m.confirmationInput.Blur()
	m.confirmationInput.SetValue("")
	m.textarea.Focus()
	m.confirmationCmd = ""
	m.typing = true
	m.typingDots = 0
	return m, tea.Batch(
		m.waitForResponse(systemResponse),
		m.showTypingAnimation(),
	)
}

func (m Model) waitForResponse(userMessage string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
		defer cancel()

		responseChan := make(chan types.AgentResponse)
		errChan := make(chan error)

		go func() {
			response, err := m.agent.Iterate(ctx, userMessage)
			if err != nil {
				errChan <- err
				return
			}
			responseChan <- response
		}()

		select {
		case response := <-responseChan:
			return response
		case err := <-errChan:
			return errMsg(err)
		case <-ctx.Done():
			return errMsg(ctx.Err())
		}
	}
}
