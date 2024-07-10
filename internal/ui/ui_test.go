package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eliran89c/klama/internal/agent"
	"github.com/eliran89c/klama/internal/executer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAgent is a mock implementation of the Agent interface
type MockAgent struct {
	mock.Mock
}

func (m *MockAgent) Iterate(ctx context.Context, prompt string) (agent.AgentResponse, error) {
	args := m.Called(ctx, prompt)
	return args.Get(0).(agent.AgentResponse), args.Error(1)
}

func (m *MockAgent) Reset() {
	m.Called()
}

func (m *MockAgent) LogUsage() string {
	args := m.Called()
	return args.String(0)
}

// MockExecuter is a mock implementation of the Executer interface
type MockExecuter struct {
	mock.Mock
}

func (m *MockExecuter) Run(ctx context.Context, command string) executer.ExecuterResponse {
	args := m.Called(ctx, command)
	return args.Get(0).(executer.ExecuterResponse)
}

func TestInitialModel(t *testing.T) {
	mockAgent := new(MockAgent)
	mockExecuter := new(MockExecuter)

	mockAgent.On("LogUsage").Return("Test usage")

	cfg := Config{
		Agent:    mockAgent,
		Executer: mockExecuter,
		Debug:    false,
	}
	model := InitialModel(cfg)

	assert.NotNil(t, model)
	assert.Equal(t, mockAgent, model.agent)
	assert.Equal(t, mockExecuter, model.executer)
	assert.False(t, model.debug)
	assert.NotNil(t, model.textarea)
	assert.NotNil(t, model.viewport)
	assert.NotNil(t, model.confirmationInput)
}

func TestModel_Init(t *testing.T) {
	model := Model{}
	cmd := model.Init()
	assert.NotNil(t, cmd)
}

func TestModel_Update(t *testing.T) {
	mockAgent := new(MockAgent)
	mockExecuter := new(MockExecuter)

	mockAgent.On("LogUsage").Return("Test usage")
	mockAgent.On("Iterate", mock.Anything, mock.Anything).Return(agent.AgentResponse{Answer: "Test answer"}, nil)

	cfg := Config{
		Agent:    mockAgent,
		Executer: mockExecuter,
		Debug:    false,
	}

	model := InitialModel(cfg)

	// Test window size message
	newModel, cmd := model.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	updatedModel := newModel.(Model)
	assert.Equal(t, 100, updatedModel.windowWidth)
	assert.Equal(t, 50, updatedModel.windowHeight)
	assert.Nil(t, cmd) // Changed to Nil as it seems no command is returned for window size updates

	// Test key message (Ctrl+C)
	newModel, cmd = model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	assert.NotNil(t, cmd) // handleExit returns a command

	// Test key message (Enter)
	model.textarea.SetValue("Test message")
	newModel, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updatedModel = newModel.(Model)
	assert.True(t, updatedModel.typing)
	assert.NotNil(t, cmd)

	// Test agent response message
	agentResponse := agent.AgentResponse{Answer: "Test answer", RunCommand: ""}
	newModel, cmd = model.Update(agentResponse)
	updatedModel = newModel.(Model)
	assert.False(t, updatedModel.typing)
	assert.Contains(t, updatedModel.messages[len(updatedModel.messages)-1], "Test answer")
	assert.Nil(t, cmd)

	// Test error message
	testError := errors.New("Test error")
	newModel, cmd = model.Update(errMsg(testError))
	updatedModel = newModel.(Model)
	assert.Equal(t, testError, updatedModel.err)
	assert.Nil(t, cmd)
}

func TestModel_waitForResponse(t *testing.T) {
	mockAgent := new(MockAgent)
	mockExecuter := new(MockExecuter)

	mockAgent.On("LogUsage").Return("Test usage")

	cfg := Config{
		Agent:    mockAgent,
		Executer: mockExecuter,
		Debug:    false,
	}

	model := InitialModel(cfg)

	// Test successful response
	mockAgent.On("Iterate", mock.Anything, "test message").Return(agent.AgentResponse{Answer: "Test answer"}, nil)
	cmd := model.waitForResponse("test message")
	msg := cmd()
	assert.IsType(t, agent.AgentResponse{}, msg)

	// Test error response
	testError := errors.New("Test error")
	mockAgent.On("Iterate", mock.Anything, "error message").Return(agent.AgentResponse{}, testError)
	cmd = model.waitForResponse("error message")
	msg = cmd()
	assert.Error(t, msg.(error))
	assert.Equal(t, testError.Error(), msg.(error).Error())

	// Test context timeout
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	model.ctx = timeoutCtx
	mockAgent.On("Iterate", mock.Anything, "timeout message").Return(agent.AgentResponse{}, context.DeadlineExceeded)
	cmd = model.waitForResponse("timeout message")
	msg = cmd()
	assert.Error(t, msg.(error))
	assert.Equal(t, context.DeadlineExceeded.Error(), msg.(error).Error())
}

func TestWrapMessage(t *testing.T) {
	message := "Sender: This is a long message that needs to be wrapped"
	wrapped := wrapMessage(message, 20)
	lines := strings.Split(wrapped, "\n")
	assert.GreaterOrEqual(t, len(lines), 4)
	assert.Equal(t, "Sender: This is a", lines[0])
	assert.Contains(t, wrapped, "long message")
	assert.Contains(t, wrapped, "that needs")
	assert.Contains(t, wrapped, "to be")
	assert.Contains(t, wrapped, "wrapped")
}

func TestWordWrap(t *testing.T) {
	text := "This is a long sentence that needs to be wrapped"
	wrapped := wordWrap(text, 10)
	assert.Equal(t, 6, len(wrapped))
	assert.Equal(t, "This is a", wrapped[0])
	assert.Equal(t, "long", wrapped[1])
	assert.Equal(t, "sentence", wrapped[2])
	assert.Equal(t, "that needs", wrapped[3])
	assert.Equal(t, "to be", wrapped[4])
	assert.Equal(t, "wrapped", wrapped[5])
}

func TestShowTypingAnimation(t *testing.T) {
	model := Model{}
	cmd := model.showWaitingAnimation()
	assert.NotNil(t, cmd)
}

func TestRenderInputArea(t *testing.T) {
	mockAgent := new(MockAgent)
	mockExecuter := new(MockExecuter)

	mockAgent.On("LogUsage").Return("Test usage")

	cfg := Config{
		Agent:    mockAgent,
		Executer: mockExecuter,
		Debug:    false,
	}

	model := InitialModel(cfg)

	// Test normal state
	result := model.renderInputArea()
	assert.Contains(t, result, "Send a message...")

	// Test waiting for confirmation
	model.waitingForConfirmation = true
	result = model.renderInputArea()
	assert.Contains(t, result, "'yes' to approve, 'no' to reject, 'ask' to break out and ask a question")

	// Test typing state
	model.waitingForConfirmation = false
	model.typing = true
	result = model.renderInputArea()
	assert.Contains(t, result, "Klama is typing")
}

func TestRenderErrorMessage(t *testing.T) {
	mockAgent := new(MockAgent)
	mockExecuter := new(MockExecuter)

	mockAgent.On("LogUsage").Return("Test usage")

	cfg := Config{
		Agent:    mockAgent,
		Executer: mockExecuter,
		Debug:    false,
	}

	model := InitialModel(cfg)

	model.err = fmt.Errorf("Test error")
	result := model.renderErrorMessage()
	assert.Contains(t, result, "Test error")
}

func TestRenderHelpText(t *testing.T) {
	mockAgent := new(MockAgent)
	mockExecuter := new(MockExecuter)

	mockAgent.On("LogUsage").Return("Test usage")

	cfg := Config{
		Agent:    mockAgent,
		Executer: mockExecuter,
		Debug:    false,
	}

	model := InitialModel(cfg)

	result := model.renderHelpText()
	assert.Contains(t, result, "Ctrl+C: exit")
	assert.Contains(t, result, "Ctrl+R: restart")
	assert.Contains(t, result, "↑: scroll up")
	assert.Contains(t, result, "↓: scroll down")
}

func TestRenderPriceText(t *testing.T) {
	mockAgent := new(MockAgent)
	mockAgent.On("LogUsage").Return("Test usage")
	cfg := Config{
		Agent:    mockAgent,
		Executer: new(MockExecuter),
		Debug:    false,
	}

	model := InitialModel(cfg)
	result := model.renderPriceText()
	assert.Contains(t, result, "Test usage")
}
