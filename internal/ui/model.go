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
	// Dependencies
	agent    types.Agent
	executer types.Executer

	// UI Components
	viewport          viewport.Model
	textarea          textarea.Model
	confirmationInput textinput.Model

	// Styles
	senderStyle lipgloss.Style
	klamaStyle  lipgloss.Style
	systemStyle lipgloss.Style
	errorStyle  lipgloss.Style
	helpStyle   lipgloss.Style
	priceStyle  lipgloss.Style
	typingStyle lipgloss.Style

	// State
	messages               []string
	err                    error
	typing                 bool
	executing              bool
	debug                  bool
	typingDots             int
	executingDots          int
	waitingForConfirmation bool
	confirmationCmd        string

	// Window
	windowWidth  int
	windowHeight int

	// Context
	ctx    context.Context
	cancel context.CancelFunc
}

// Config holds the configuration for initializing the Model
type Config struct {
	Agent    types.Agent
	Executer types.Executer
	Debug    bool
}

// InitialModel creates and returns a new instance of Model with default values.
func InitialModel(cfg Config) Model {
	textArea := textarea.New()
	textArea.Placeholder = "Send a message..."
	textArea.Focus()
	textArea.Prompt = "┃ "
	textArea.CharLimit = 280
	textArea.ShowLineNumbers = false
	textArea.KeyMap.InsertNewline.SetEnabled(false)

	viewPort := viewport.New(80, 20)
	viewPort.SetContent(`Welcome to Klama!
Enter your question or issue.`)

	confirmationInput := textinput.New()
	confirmationInput.Placeholder = "To approve the command, type 'yes'. To reject it, type 'no'."
	confirmationInput.CharLimit = 3

	ctx, cancel := context.WithCancel(context.Background())

	return Model{
		agent:             cfg.Agent,
		executer:          cfg.Executer,
		textarea:          textArea,
		messages:          []string{},
		viewport:          viewPort,
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
		confirmationInput: confirmationInput,
		debug:             cfg.Debug,
	}
}

// Init initializes the Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, textinput.Blink)
}
