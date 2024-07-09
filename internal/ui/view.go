package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// View renders the current state of the application.
func (m Model) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.viewport.View(),
		m.renderInputArea(),
		m.renderErrorMessage(),
		m.renderHelpText(),
		m.renderPriceText(),
	)
}

// renderInputArea renders the input area based on the current state.
func (m Model) renderInputArea() string {
	switch {
	case m.waitingForConfirmation:
		return m.confirmationInput.View()
	case m.typing:
		return m.typingStyle.Render("Klama is typing...")
	case m.executing:
		return m.systemStyle.Render("Command executing...")
	default:
		return m.textarea.View()
	}
}

// renderErrorMessage renders the error message if present.
func (m Model) renderErrorMessage() string {
	return m.errorStyle.Render(m.errorMsg)
}

// renderHelpText renders the help text.
func (m Model) renderHelpText() string {
	helpText := "Ctrl+C: exit, Ctrl+R: restart"
	if !m.typing {
		helpText += ", ↑: scroll up, ↓: scroll down"
	}
	return m.helpStyle.Render(helpText)
}

// renderPriceText renders the price information.
func (m Model) renderPriceText() string {
	return m.priceStyle.Render(m.agent.LogUsage())
}

// updateMessages updates the viewport content with the current messages.
func (m *Model) updateMessages() {
	var messages []string
	for _, msg := range m.messages {
		messages = append(messages, wrapMessage(msg, m.viewport.Width))
	}
	if m.typing {
		messages = append(messages, m.klamaStyle.Render("Klama: ")+strings.Repeat(".", m.typingDots))
	}
	if m.executing {
		messages = append(messages, m.systemStyle.Render("System: ")+strings.Repeat(".", m.executingDots))
	}
	m.viewport.SetContent(strings.Join(messages, "\n\n"))
	m.viewport.GotoBottom()
}

// wrapMessage wraps a single message to fit within the given width, preserving formatting and new lines.
func wrapMessage(message string, width int) string {
	parts := strings.SplitN(message, ": ", 2)
	if len(parts) != 2 {
		return message // Return as is if it doesn't follow the expected format
	}

	sender, content := parts[0], parts[1]
	lines := strings.Split(content, "\n")
	var wrappedLines []string

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			wrappedLines = append(wrappedLines, "")
		} else {
			wrappedLines = append(wrappedLines, wordWrap(line, width-len(sender)-2)...)
		}
	}

	wrappedContent := strings.Join(wrappedLines, "\n")
	return fmt.Sprintf("%s: %s", sender, wrappedContent)
}

// wordWrap wraps the given text to fit within the specified line width.
func wordWrap(text string, lineWidth int) []string {
	words := strings.Fields(strings.TrimSpace(text))
	if len(words) == 0 {
		return []string{text}
	}

	var lines []string
	var currentLine string

	for _, word := range words {
		if len(currentLine)+len(word)+1 > lineWidth {
			if len(currentLine) > 0 {
				lines = append(lines, strings.TrimSpace(currentLine))
				currentLine = ""
			}
			if len(word) > lineWidth {
				for len(word) > 0 {
					if len(word) > lineWidth {
						lines = append(lines, word[:lineWidth])
						word = word[lineWidth:]
					} else {
						currentLine = word
						break
					}
				}
			} else {
				currentLine = word
			}
		} else {
			if len(currentLine) > 0 {
				currentLine += " "
			}
			currentLine += word
		}
	}

	if len(currentLine) > 0 {
		lines = append(lines, strings.TrimSpace(currentLine))
	}

	return lines
}

// showWaitingAnimation creates the typing animation effect.
func (m Model) showWaitingAnimation() tea.Cmd {
	return tea.Tick(time.Millisecond*300, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
