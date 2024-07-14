package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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

func (m model) headerView() string {
	title := titleStyle.Render("Klama")
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m model) footerView() string {
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

// View renders the current state of the application.
func (m model) View() string {
	return fmt.Sprintf("%s\n%s\n%s",
		m.headerView(),
		m.viewport.View(),
		m.footerView(),
	)
}

// renderInputArea renders the input area based on the current state.
func (m model) renderInputArea() string {
	switch m.state {
	case stateAsking:
		return m.typingStyle.Render("\n\nKlama is typing" + strings.Repeat(".", m.waitingDots))

	case stateExecuting:
		return m.typingStyle.Render("\n\nCommand executing" + strings.Repeat(".", m.waitingDots))

	default:
		return m.textarea.View()
	}
}

// renderErrorMessage renders the error message if present.
func (m model) renderErrorMessage() string {
	if m.err != nil {
		return m.errorStyle.Render(m.err.Error())
	}
	return ""
}

// renderHelpText renders the help text.
func (m model) renderHelpText() string {
	helpText := "Ctrl+C: exit, Ctrl+R: restart, Scroll with ↑, ↓, Page Up, Page Down and mouse wheel."
	return m.helpStyle.Width(m.width).Render(helpText)
}

// renderPriceText renders the price information.
func (m model) renderPriceText() string {
	return m.priceStyle.Width(m.width).Render(m.agent.LogUsage())
}

// updateChat updates the chat with the current messages.
func (m *model) updateChat(style lipgloss.Style, prefix, message string) {
	m.messages = append(m.messages, style.Render(prefix+": ")+message)
	wrapped := lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.messages, "\n\n"))
	m.viewport.SetContent(wrapped)
	m.textarea.Reset()
	m.viewport.GotoBottom()
}
