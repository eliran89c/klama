package ui

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eliran89c/klama/internal/agent"
	"github.com/eliran89c/klama/internal/executer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockAgent struct {
	mock.Mock
}

func (m *MockAgent) Iterate(ctx context.Context, input string) (agent.AgentResponse, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(agent.AgentResponse), args.Error(1)
}

func (m *MockAgent) Reset() {
	m.Called()
}

func (m *MockAgent) LogUsage() string {
	args := m.Called()
	return args.String(0)
}

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

	model := InitialModel(Config{
		Agent:    mockAgent,
		Executer: mockExecuter,
		Debug:    true,
	})

	assert.NotNil(t, model)
	assert.Equal(t, StateTyping, model.state)
	assert.True(t, model.debug)
}

func TestModel_Init(t *testing.T) {
	model := InitialModel(Config{})
	cmd := model.Init()
	assert.NotNil(t, cmd)
}

func TestModel_Update(t *testing.T) {
	mockAgent := new(MockAgent)
	mockExecuter := new(MockExecuter)

	model := InitialModel(Config{
		Agent:    mockAgent,
		Executer: mockExecuter,
	})

	mockAgent.On("LogUsage").Return("Test usage")
	mockAgent.On("Reset").Return()

	tests := []struct {
		name     string
		msg      tea.Msg
		expected modelState
	}{
		{"WindowSize", tea.WindowSizeMsg{Width: 100, Height: 50}, StateTyping},
		{"KeyUp", tea.KeyMsg{Type: tea.KeyUp}, StateTyping},
		{"KeyEnter", tea.KeyMsg{Type: tea.KeyEnter}, StateTyping},
		{"Tick", tickMsg(time.Now()), StateTyping},
		{"CtrlR", tea.KeyMsg{Type: tea.KeyCtrlR}, StateTyping},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newModel, _ := model.Update(tt.msg)
			assert.Equal(t, tt.expected, newModel.(Model).state)
		})
	}

	mockAgent.AssertExpectations(t)
}

func TestModel_View(t *testing.T) {
	mockAgent := new(MockAgent)
	mockExecuter := new(MockExecuter)

	mockAgent.On("LogUsage").Return("Test usage")

	model := InitialModel(Config{
		Agent:    mockAgent,
		Executer: mockExecuter,
	})

	model.width = 100
	model.height = 50

	view := model.View()

	assert.Contains(t, view, "Klama")
	assert.Contains(t, view, "Test usage")

	mockAgent.AssertExpectations(t)
}

func TestModel_handleKeyMsg(t *testing.T) {
	mockAgent := new(MockAgent)
	mockExecuter := new(MockExecuter)

	model := InitialModel(Config{
		Agent:    mockAgent,
		Executer: mockExecuter,
	})

	mockAgent.On("LogUsage").Return("Test usage")
	mockAgent.On("Reset").Return()

	tests := []struct {
		name     string
		key      tea.KeyType
		expected modelState
	}{
		{"Escape", tea.KeyEsc, StateTyping},
		{"CtrlC", tea.KeyCtrlC, StateTyping},
		{"CtrlR", tea.KeyCtrlR, StateTyping},
		{"Enter", tea.KeyEnter, StateTyping},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newModel, _ := model.handleKeyMsg(tea.KeyMsg{Type: tt.key})
			assert.Equal(t, tt.expected, newModel.(Model).state)
		})
	}

	mockAgent.AssertExpectations(t)
}

func TestModel_handleEnterKey(t *testing.T) {
	mockAgent := new(MockAgent)
	mockExecuter := new(MockExecuter)

	model := InitialModel(Config{
		Agent:    mockAgent,
		Executer: mockExecuter,
	})

	InitialModel, _ := model.handleEnterKey()
	assert.Equal(t, StateTyping, InitialModel.(Model).state)
	assert.NotNil(t, InitialModel.(Model).err)

	model.textarea.SetValue("Test input")
	mockAgent.On("Iterate", mock.Anything, "Test input").Return(agent.AgentResponse{Answer: "Test response"}, nil)
	mockAgent.On("LogUsage").Return("Test usage")
	InitialModel, _ = model.handleEnterKey()
	assert.Equal(t, StateAsking, InitialModel.(Model).state)
}

func TestModel_handleAgentResponse(t *testing.T) {
	model := InitialModel(Config{})

	tests := []struct {
		name     string
		response agent.AgentResponse
		expected modelState
	}{
		{"Normal response", agent.AgentResponse{Answer: "Test answer"}, StateTyping},
		{"Command response", agent.AgentResponse{RunCommand: "test command", Reason: "Test reason"}, StateWaitingForConfirmation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InitialModel, _ := model.handleAgentResponse(tt.response)
			assert.Equal(t, tt.expected, InitialModel.(Model).state)
		})
	}
}

func TestModel_handleExecuterResponse(t *testing.T) {
	mockAgent := new(MockAgent)
	model := InitialModel(Config{Agent: mockAgent, Debug: true})

	mockAgent.On("Iterate", mock.Anything, mock.Anything).Return(agent.AgentResponse{Answer: "Test response"}, nil)
	mockAgent.On("LogUsage").Return("Test usage")

	tests := []struct {
		name     string
		response executer.ExecuterResponse
		expected modelState
	}{
		{"Successful execution", executer.ExecuterResponse{Result: "Test result"}, StateAsking},
		{"Failed execution", executer.ExecuterResponse{Error: assert.AnError}, StateAsking},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InitialModel, _ := model.handleExecuterResponse(tt.response)
			assert.Equal(t, tt.expected, InitialModel.(Model).state)
		})
	}
}

func TestModel_handleConfirmation(t *testing.T) {
	mockAgent := new(MockAgent)
	mockExecuter := new(MockExecuter)

	model := InitialModel(Config{
		Agent:    mockAgent,
		Executer: mockExecuter,
	})
	model.confirmationCmd = "test command"
	model.state = StateWaitingForConfirmation

	tests := []struct {
		name     string
		input    string
		expected modelState
	}{
		{"Yes", "yes", StateExecuting},
		{"No", "no", StateAsking},
		{"Ask", "ask", StateTyping},
		{"Invalid", "invalid", StateWaitingForConfirmation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model.textarea.SetValue(tt.input)
			mockAgent.On("Iterate", mock.Anything, mock.Anything).Return(agent.AgentResponse{Answer: "Test response"}, nil).Maybe()
			mockExecuter.On("Run", mock.Anything, mock.Anything).Return(executer.ExecuterResponse{Result: "Test result"}).Maybe()

			newModel, _ := model.handleConfirmation()
			assert.Equal(t, tt.expected, newModel.(Model).state)
		})
	}

	mockAgent.AssertExpectations(t)
	mockExecuter.AssertExpectations(t)
}

func TestModel_updateChat(t *testing.T) {
	model := InitialModel(Config{})
	model.updateChat(model.senderStyle, "Test", "Test message")

	assert.Contains(t, model.viewport.View(), "Test: Test message")
}

func TestModel_headerView(t *testing.T) {
	mockAgent := new(MockAgent)
	mockAgent.On("LogUsage").Return("Test usage").Maybe()

	model := InitialModel(Config{Agent: mockAgent})
	model.width = 100

	header := model.headerView()

	assert.Contains(t, header, "Klama")

	mockAgent.AssertExpectations(t)
}

func TestModel_footerView(t *testing.T) {
	mockAgent := new(MockAgent)
	mockAgent.On("LogUsage").Return("Test usage")

	model := InitialModel(Config{Agent: mockAgent})
	model.width = 100

	footer := model.footerView()

	assert.Contains(t, footer, "Ctrl+C: to exit")
	assert.Contains(t, footer, "Test usage")

	mockAgent.AssertExpectations(t)
}

func TestModel_renderInputArea(t *testing.T) {
	model := InitialModel(Config{})

	tests := []struct {
		name     string
		state    modelState
		expected string
	}{
		{"Typing", StateTyping, "Send a message..."},
		{"Asking", StateAsking, "Klama is typing"},
		{"Executing", StateExecuting, "Command executing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model.state = tt.state
			inputArea := model.renderInputArea()
			assert.Contains(t, inputArea, tt.expected)
		})
	}
}

func TestModel_renderErrorMessage(t *testing.T) {
	model := InitialModel(Config{})
	model.err = assert.AnError

	errorMsg := model.renderErrorMessage()
	assert.Contains(t, errorMsg, assert.AnError.Error())
}

func TestModel_renderHelpText(t *testing.T) {
	model := InitialModel(Config{})
	helpText := model.renderHelpText()

	assert.Contains(t, helpText, "Ctrl+C: to exit")
	assert.Contains(t, helpText, "Ctrl+R: to restart")
}

func TestModel_renderPriceText(t *testing.T) {
	mockAgent := new(MockAgent)
	model := InitialModel(Config{Agent: mockAgent})

	mockAgent.On("LogUsage").Return("Test usage")

	priceText := model.renderPriceText()
	assert.Contains(t, priceText, "Test usage")
}
