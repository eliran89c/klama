package ui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eliran89c/klama/internal/agent"
	"github.com/eliran89c/klama/internal/executer"
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

// Agent represents the agent interface
type Agent interface {
	Iterate(context.Context, string) (agent.AgentResponse, error)
	Reset()
	LogUsage() string
}

// Executer represents the executer interface
type Executer interface {
	Run(context.Context, string) executer.ExecuterResponse
}

// Model represents the application state.
type Model struct {
	// Dependencies
	agent    Agent
	executer Executer

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
	Agent    Agent
	Executer Executer
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

	newStyle := func(color string) lipgloss.Style {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	}

	return Model{
		agent:             cfg.Agent,
		executer:          cfg.Executer,
		textarea:          textArea,
		messages:          []string{},
		viewport:          viewPort,
		senderStyle:       newStyle(ColorSender),
		klamaStyle:        newStyle(ColorKlama),
		systemStyle:       newStyle(ColorSystem),
		errorStyle:        newStyle(ColorError),
		helpStyle:         newStyle(ColorHelp),
		priceStyle:        newStyle(ColorPrice),
		typingStyle:       newStyle(ColorHelp),
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
