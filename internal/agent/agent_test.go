package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eliran89c/klama/internal/llm"
	"github.com/eliran89c/klama/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockExecuter is a mock implementation of the Executer interface
type MockExecuter struct {
	ValidateFunc func(ctx context.Context, command string) (bool, string, error)
	RunFunc      func(ctx context.Context, command string) (string, error)
}

func (m *MockExecuter) Validate(ctx context.Context, command string) (bool, string, error) {
	return m.ValidateFunc(ctx, command)
}

func (m *MockExecuter) Run(ctx context.Context, command string) (string, error) {
	return m.RunFunc(ctx, command)
}

func TestNew(t *testing.T) {
	model := &llm.Model{}
	log := logger.EmptyLogger()

	ag, err := New(model, log)
	assert.NoError(t, err)
	assert.NotNil(t, ag)
	assert.Equal(t, model, ag.AgentModel)
	assert.Equal(t, log, ag.Logger)

	// Test with nil model
	ag, err = New(nil, log)
	assert.Error(t, err)
	assert.Nil(t, ag)

	// Test with nil logger
	ag, err = New(model, nil)
	assert.NoError(t, err)
	assert.NotNil(t, ag)
	assert.NotNil(t, ag.Logger)
}

func TestAgent_StartSession(t *testing.T) {
	testCases := []struct {
		name          string
		mockResponses []AgentResponse
		execResponses []string
		wantFinal     string
		wantErr       bool
	}{
		{
			name: "successful session",
			mockResponses: []AgentResponse{
				{FinalAnswer: "The issue has been resolved.", NeedMoreData: false},
			},
			wantFinal: "The issue has been resolved.",
			wantErr:   false,
		},
		{
			name: "session with command execution",
			mockResponses: []AgentResponse{
				{RunCommand: "kubectl get pods", NeedMoreData: true},
				{FinalAnswer: "All pods are running.", NeedMoreData: false},
			},
			execResponses: []string{"pod1 Running\npod2 Running"},
			wantFinal:     "All pods are running.",
			wantErr:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var resp AgentResponse
				if len(tc.mockResponses) > 0 {
					resp = tc.mockResponses[0]
					tc.mockResponses = tc.mockResponses[1:]
				}
				json.NewEncoder(w).Encode(map[string]interface{}{
					"choices": []map[string]interface{}{
						{"message": map[string]interface{}{"content": mustMarshal(resp)}},
					},
				})
			}))
			defer mockServer.Close()

			model := &llm.Model{
				Client:  mockServer.Client(),
				BaseURL: mockServer.URL,
			}

			execCounter := 0
			mockExec := &MockExecuter{
				ValidateFunc: func(ctx context.Context, command string) (bool, string, error) {
					return true, "Valid command", nil
				},
				RunFunc: func(ctx context.Context, command string) (string, error) {
					if execCounter < len(tc.execResponses) {
						resp := tc.execResponses[execCounter]
						execCounter++
						return resp, nil
					}
					return "", nil
				},
			}

			ag, err := New(model, logger.EmptyLogger())
			require.NoError(t, err)

			got, err := ag.StartSession(context.Background(), mockExec, "Test query")

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantFinal, got)
			}
		})
	}
}

func TestAgent_interactWithModel(t *testing.T) {
	testCases := []struct {
		name          string
		mockResponses []string
		wantResp      AgentResponse
		wantErr       bool
	}{
		{
			name:          "successful interaction",
			mockResponses: []string{`{"final_answer": "Test answer", "need_more_data": false}`},
			wantResp:      AgentResponse{FinalAnswer: "Test answer", NeedMoreData: false},
			wantErr:       false,
		},
		{
			name:          "invalid JSON response",
			mockResponses: []string{`invalid JSON`, `{"final_answer": "Corrected answer", "need_more_data": false}`},
			wantResp:      AgentResponse{FinalAnswer: "Corrected answer", NeedMoreData: false},
			wantErr:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var resp string
				if len(tc.mockResponses) > 0 {
					resp = tc.mockResponses[0]
					tc.mockResponses = tc.mockResponses[1:]
				}
				json.NewEncoder(w).Encode(map[string]interface{}{
					"choices": []map[string]interface{}{
						{"message": map[string]interface{}{"content": resp}},
					},
				})
			}))
			defer mockServer.Close()

			model := &llm.Model{
				Client:  mockServer.Client(),
				BaseURL: mockServer.URL,
			}

			ag, err := New(model, logger.EmptyLogger())
			require.NoError(t, err)

			got, err := ag.interactWithModel(context.Background(), "Test prompt")

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantResp, got)
			}
		})
	}
}

func TestAgent_handleCommand(t *testing.T) {
	testCases := []struct {
		name           string
		command        string
		executedBefore bool
		validateResult bool
		validateReason string
		executeResp    string
		wantPrompt     string
	}{
		{
			name:           "successful command execution",
			command:        "kubectl get pods",
			validateResult: true,
			validateReason: "Valid command",
			executeResp:    "pod1 Running\npod2 Running",
			wantPrompt:     "pod1 Running\npod2 Running",
		},
		{
			name:           "command already executed",
			command:        "kubectl get pods",
			executedBefore: true,
			wantPrompt:     "pod1 Running\npod2 Running",
		},
		{
			name:           "invalid command",
			command:        "kubectl delete pod myapp",
			validateResult: false,
			validateReason: "Delete operation not allowed",
			wantPrompt:     "Delete operation not allowed",
		},
		{
			name:           "execution error",
			command:        "kubectl get pods",
			validateResult: true,
			validateReason: "Valid command",
			executeResp:    "",
			wantPrompt:     "Command failed: execution error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockExec := &MockExecuter{
				ValidateFunc: func(ctx context.Context, command string) (bool, string, error) {
					return tc.validateResult, tc.validateReason, nil
				},
				RunFunc: func(ctx context.Context, command string) (string, error) {
					if tc.executeResp == "" {
						return "", fmt.Errorf("execution error")
					}
					return tc.executeResp, nil
				},
			}

			ag, err := New(&llm.Model{}, logger.EmptyLogger())
			require.NoError(t, err)

			executedCommands := make(map[string]string)
			if tc.executedBefore {
				executedCommands[tc.command] = "pod1 Running\npod2 Running"
			}

			got := ag.handleCommand(context.Background(), mockExec, tc.command, executedCommands)
			assert.Equal(t, tc.wantPrompt, got)
		})
	}
}

func TestAgent_handleCommand_ExecutionError(t *testing.T) {
	mockExec := &MockExecuter{
		ValidateFunc: func(ctx context.Context, command string) (bool, string, error) {
			return true, "Valid command", nil
		},
		RunFunc: func(ctx context.Context, command string) (string, error) {
			return "", assert.AnError
		},
	}

	ag, err := New(&llm.Model{}, logger.EmptyLogger())
	require.NoError(t, err)

	got := ag.handleCommand(context.Background(), mockExec, "kubectl get pods", make(map[string]string))

	assert.Equal(t, "Command failed: assert.AnError general error for testing", got)
}

func TestAgent_StartSession_MaxQueries(t *testing.T) {
	mockResponses := make([]AgentResponse, numberOfQueries+1)
	for i := 0; i < numberOfQueries; i++ {
		mockResponses[i] = AgentResponse{RunCommand: "kubectl get pods", NeedMoreData: true}
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var resp AgentResponse
		if len(mockResponses) > 0 {
			resp = mockResponses[0]
			mockResponses = mockResponses[1:]
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]interface{}{"content": mustMarshal(resp)}},
			},
		})
	}))
	defer mockServer.Close()

	model := &llm.Model{
		Client:  mockServer.Client(),
		BaseURL: mockServer.URL,
	}

	mockExec := &MockExecuter{
		ValidateFunc: func(ctx context.Context, command string) (bool, string, error) {
			return true, "Valid command", nil
		},
		RunFunc: func(ctx context.Context, command string) (string, error) {
			return "pod1 Running\npod2 Running", nil
		},
	}

	ag, err := New(model, logger.EmptyLogger())
	require.NoError(t, err)

	got, err := ag.StartSession(context.Background(), mockExec, "Test query")

	assert.NoError(t, err)
	assert.Equal(t, "Analysis incomplete. Reached maximum number of queries.", got)
}

func TestAgent_StartSession_ContextCancellation(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a long-running operation
		select {
		case <-r.Context().Done():
			return
		case <-time.After(100 * time.Millisecond):
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]interface{}{"content": `{"need_more_data": true}`}},
			},
		})
	}))
	defer mockServer.Close()

	model := &llm.Model{
		Client:  mockServer.Client(),
		BaseURL: mockServer.URL,
	}

	mockExec := &MockExecuter{
		ValidateFunc: func(ctx context.Context, command string) (bool, string, error) {
			return true, "Valid command", nil
		},
		RunFunc: func(ctx context.Context, command string) (string, error) {
			return "pod1 Running\npod2 Running", nil
		},
	}

	ag, err := New(model, logger.EmptyLogger())
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err = ag.StartSession(ctx, mockExec, "Test query")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func mustMarshal(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
