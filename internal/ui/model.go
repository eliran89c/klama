package ui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eliran89c/klama/internal/agent"
	"github.com/eliran89c/klama/internal/executer"
)

const (
	// color constants
	colorSender     = "5"
	colorKlama      = "2"
	colorSystem     = "3"
	colorError      = "1"
	colorHelp       = "241"
	colorPrice      = "6"
	colorBackground = "0"

	// state constants
	stateTyping modelState = iota
	stateAsking
	stateExecuting
	stateWaitingForConfirmation
)

type (
	errMsg     error
	tickMsg    time.Time
	modelState int
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
type model struct {
	// Dependencies
	agent    Agent
	executer Executer

	// UI Components
	viewport viewport.Model
	textarea textarea.Model

	// Styles
	senderStyle lipgloss.Style
	klamaStyle  lipgloss.Style
	systemStyle lipgloss.Style
	errorStyle  lipgloss.Style
	helpStyle   lipgloss.Style
	priceStyle  lipgloss.Style
	typingStyle lipgloss.Style

	// State
	messages        []string
	err             error
	debug           bool
	state           modelState
	waitingDots     int
	confirmationCmd string

	// Window size
	width  int
	height int

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
func InitialModel(cfg Config) model {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()
	ta.Prompt = "┃ "
	ta.CharLimit = 280
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)
	ta.SetHeight(3)

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	vp := viewport.New(80, 20) // Arbitrary starting size
	vp.SetContent("Welcome to Klama!\nEnter your question or issue.")

	ctx, cancel := context.WithCancel(context.Background())

	newStyle := func(color string) lipgloss.Style {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	}

	return model{
		agent:    cfg.Agent,
		executer: cfg.Executer,
		textarea: ta,
		viewport: vp,
		messages: []string{},

		senderStyle: newStyle(colorSender),
		klamaStyle:  newStyle(colorKlama),
		systemStyle: newStyle(colorSystem),
		errorStyle:  newStyle(colorError),
		helpStyle:   newStyle(colorHelp),
		priceStyle:  newStyle(colorPrice),
		typingStyle: newStyle(colorHelp),

		ctx:    ctx,
		cancel: cancel,
		debug:  cfg.Debug,
		state:  stateTyping,
	}
}

// Init initializes the Model.
func (m model) Init() tea.Cmd {
	return textarea.Blink
}
