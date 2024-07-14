package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eliran89c/klama/internal/agent"
	"github.com/eliran89c/klama/internal/executer"
)

// Update handles all the application logic and state transitions.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var ta, vp tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// store window size
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())

		// set viewport and textarea width and height
		m.viewport.Height = msg.Height - headerHeight - footerHeight
		m.viewport.Width = msg.Width - 2 // 2 spaces padding
		m.textarea.SetWidth(msg.Width - 2)

		return m, nil

	// update viewport on mouse scroll
	case tea.MouseMsg:
		m.viewport, vp = m.viewport.Update(msg)

	case tea.KeyMsg:
		switch msg.Type {

		// update viewport on up and down events
		case tea.KeyUp, tea.KeyDown, tea.KeyPgUp, tea.KeyPgDown:
			m.viewport, vp = m.viewport.Update(msg)

		case tea.KeyCtrlC, tea.KeyEsc:
			m.cancel()
			return m, tea.Quit

		case tea.KeyCtrlR:
			m.cancel()
			m.agent.Reset()
			newModel := InitialModel(Config{
				Agent:    m.agent,
				Executer: m.executer,
				Debug:    m.debug,
			})
			return newModel.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})

		case tea.KeyEnter:
			switch m.state {
			case stateTyping:
				// user is typing
				query := m.textarea.Value()
				if query == "" {
					m.err = fmt.Errorf("message cannot be empty")
					return m, nil
				}
				m.updateChat(m.senderStyle, "You", query)
				m.state = stateAsking
				return m, tea.Batch(
					m.waitForAgentResponse(query),
					m.think(),
				)

			case stateWaitingForConfirmation:
				// waiting for user confirmation
				userInput := strings.TrimSpace(strings.ToLower(m.textarea.Value()))

				// handle user confirmation
				switch userInput {
				case "yes", "y":
					m.state = stateExecuting
					m.updateChat(m.systemStyle, "System", fmt.Sprintf("Executing command `%v`", m.confirmationCmd))
					return m, tea.Batch(
						m.waitForExecution(m.confirmationCmd),
						m.think(),
					)

				case "no", "n":
					m.state = stateAsking
					rejectMsg := "User did not approve the command, please suggest different command or end the session."
					m.updateChat(m.systemStyle, "System", rejectMsg)
					return m, tea.Batch(
						m.waitForAgentResponse(rejectMsg),
						m.think(),
					)

				case "ask", "a":
					m.state = stateTyping
					m.updateChat(m.systemStyle, "System", "Breaking out to ask a question")
					return m, nil

				default:
					m.err = fmt.Errorf("please answer with 'yes', 'no' or 'ask'")
					m.textarea.Reset()
					return m, nil
				}
			}

		default:
			// reset error message if user is typing
			if m.state == stateTyping || m.state == stateWaitingForConfirmation {
				m.err = nil
				m.textarea, ta = m.textarea.Update(msg)
			}
		}

	case tickMsg:
		m.waitingDots = (m.waitingDots + 1) % 4
		return m, m.think()

	case agent.AgentResponse:
		m.state = stateTyping
		if msg.RunCommand != "" {
			m.state = stateWaitingForConfirmation
			m.confirmationCmd = msg.RunCommand

			// create klama response
			var klamaResp string

			if msg.Answer != "" {
				klamaResp += msg.Answer + "\n"
			}

			klamaResp += "I suggest running the command `" + m.systemStyle.Render(msg.RunCommand)
			klamaResp += fmt.Sprintf("`\n%v", msg.Reason)

			m.updateChat(m.klamaStyle, "Klama", klamaResp)
			m.updateChat(m.systemStyle, "System", "Enter 'yes' to approve, 'no' to reject, 'ask' to break out and ask a question")

		} else {
			m.updateChat(m.klamaStyle, "Klama", msg.Answer)
		}

		return m, nil

	case executer.ExecuterResponse:
		m.state = stateAsking
		var systemResponse string

		if msg.Error != nil {
			systemResponse = fmt.Sprintf("Error executing command: %v\n%v", msg.Error.Error(), msg.Result)
		} else {
			systemResponse = fmt.Sprintf("Command output:\n%v", msg.Result)
		}

		if m.debug {
			m.updateChat(m.systemStyle, "System", systemResponse)
		}

		return m, tea.Batch(
			m.waitForAgentResponse(systemResponse),
			m.think(),
		)

	case errMsg:
		m.err = msg
		if m.state == stateAsking || m.state == stateExecuting {
			m.state = stateTyping
		}
		return m, nil
	}

	return m, tea.Batch(ta, vp)
}

func (m model) waitForAgentResponse(userMessage string) tea.Cmd {
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

func (m model) waitForExecution(command string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
		defer cancel()

		return m.executer.Run(ctx, command)
	}
}

// think creates the typing animation effect.
func (m model) think() tea.Cmd {
	return tea.Tick(time.Millisecond*300, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
