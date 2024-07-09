package ui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eliran89c/klama/internal/app/types"
)

// Color constants for better maintainability
const (
	ColorSender     = "5"
	ColorKlama      = "2"
	ColorSystem     = "3"
	ColorError      = "1"
	ColorHelp       = "241"
	ColorPrice      = "6"
	ColorBackground = "0"
)

type (
	errMsg  error
	tickMsg time.Time
)

// Model represents the application state.
type Model struct {
	agent                  types.Agent
	executer               types.Executer
	viewport               viewport.Model
	messages               []string
	textarea               textarea.Model
	senderStyle            lipgloss.Style
	klamaStyle             lipgloss.Style
	systemStyle            lipgloss.Style
	errorStyle             lipgloss.Style
	helpStyle              lipgloss.Style
	priceStyle             lipgloss.Style
	typingStyle            lipgloss.Style
	err                    error
	errorMsg               string
	typing                 bool
	executing              bool
	debug                  bool
	typingDots             int
	executingDots          int
	waitingForConfirmation bool
	confirmationInput      textinput.Model
	confirmationCmd        string
	windowWidth            int
	windowHeight           int
	ctx                    context.Context
	cancel                 context.CancelFunc
}

// Init initializes the Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, textinput.Blink)
}

// InitialModel creates and returns a new instance of Model with default values.
func InitialModel(agent types.Agent, executer types.Executer, debug bool) Model {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()
	ta.Prompt = "┃ "
	ta.CharLimit = 280
	ta.ShowLineNumbers = false

	vp := viewport.New(80, 20)
	vp.SetContent(`Welcome to Klama!
Enter your question or issue.`)

	ta.KeyMap.InsertNewline.SetEnabled(false)

	ci := textinput.New()
	ci.Placeholder = "To approve the command, type 'yes'. To reject it, type 'no'."
	ci.CharLimit = 3

	ctx, cancel := context.WithCancel(context.Background())

	return Model{
		agent:             agent,
		executer:          executer,
		textarea:          ta,
		messages:          []string{},
		viewport:          vp,
		senderStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSender)),
		klamaStyle:        lipgloss.NewStyle().Foreground(lipgloss.Color(ColorKlama)),
		systemStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSystem)),
		errorStyle:        lipgloss.NewStyle().Foreground(lipgloss.Color(ColorError)),
		helpStyle:         lipgloss.NewStyle().Foreground(lipgloss.Color(ColorHelp)),
		priceStyle:        lipgloss.NewStyle().Foreground(lipgloss.Color(ColorPrice)),
		typingStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color(ColorHelp)),
		windowWidth:       80,
		windowHeight:      24,
		ctx:               ctx,
		cancel:            cancel,
		confirmationInput: ci,
		debug:             debug,
	}
}
