package executer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/eliran89c/klama/internal/llm"
	"github.com/eliran89c/klama/internal/logger"
	"github.com/stretchr/testify/assert"
)

func TestNewLLMExecuter(t *testing.T) {
	model := &llm.Model{}
	log := logger.EmptyLogger()

	exec, _ := NewLLMExecuter(model, log)
	assert.NotNil(t, exec)
	assert.Equal(t, model, exec.Model)
	assert.Equal(t, log, exec.Logger)

	// Test with nil logger
	exec, _ = NewLLMExecuter(model, nil)
	assert.NotNil(t, exec)
	assert.NotNil(t, exec.Logger)
}

func TestNewUserExecuter(t *testing.T) {
	log := logger.EmptyLogger()

	exec := NewUserExecuter(log)
	assert.NotNil(t, exec)
	assert.Equal(t, log, exec.Logger)

	// Test with nil logger
	exec = NewUserExecuter(nil)
	assert.NotNil(t, exec)
	assert.NotNil(t, exec.Logger)
}

func TestLLMExecuter_Validate(t *testing.T) {
	testCases := []struct {
		name        string
		command     string
		modelResp   ModelResp
		wantIsValid bool
		wantErr     bool
	}{
		{
			name:        "valid command",
			command:     "kubectl get pods",
			modelResp:   ModelResp{IsReadOnly: true, Reason: "Read-only command"},
			wantIsValid: true,
			wantErr:     false,
		},
		{
			name:        "invalid command",
			command:     "kubectl delete pod myapp",
			modelResp:   ModelResp{IsReadOnly: false, Reason: "Delete operation"},
			wantIsValid: false,
			wantErr:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"choices": []map[string]interface{}{
						{"message": map[string]interface{}{"content": mustMarshal(tc.modelResp)}},
					},
				})
			}))
			defer mockServer.Close()

			model := &llm.Model{
				Client:  mockServer.Client(),
				BaseURL: mockServer.URL,
			}

			exec, _ := NewLLMExecuter(model, logger.EmptyLogger())
			isValid, reason, err := exec.Validate(context.Background(), tc.command)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantIsValid, isValid)
				assert.Equal(t, tc.modelResp.Reason, reason)
			}
		})
	}
}

func TestUserExecuter_Validate(t *testing.T) {
	testCases := []struct {
		name        string
		command     string
		userInput   string
		wantIsValid bool
		wantErr     bool
	}{
		{
			name:        "user approves",
			command:     "kubectl get pods",
			userInput:   "yes\n",
			wantIsValid: true,
			wantErr:     false,
		},
		{
			name:        "user rejects",
			command:     "kubectl delete pod myapp",
			userInput:   "no\n",
			wantIsValid: false,
			wantErr:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Mock user input
			oldStdin := os.Stdin
			defer func() { os.Stdin = oldStdin }()
			r, w, _ := os.Pipe()
			os.Stdin = r
			w.Write([]byte(tc.userInput))
			w.Close()

			exec := NewUserExecuter(logger.EmptyLogger())
			isValid, _, err := exec.Validate(context.Background(), tc.command)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantIsValid, isValid)
			}
		})
	}
}

func TestExecute(t *testing.T) {
	testCases := []struct {
		name       string
		command    string
		wantOutput string
		wantErr    bool
	}{
		{
			name:       "echo command",
			command:    "echo 'Hello, World!'",
			wantOutput: "Hello, World!",
			wantErr:    false,
		},
		{
			name:    "invalid command",
			command: "invalidcommand",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			exec := NewUserExecuter(logger.EmptyLogger()) // We can use either executor here
			output, err := exec.Run(context.Background(), tc.command)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantOutput, strings.TrimSpace(output))
			}
		})
	}
}

func mustMarshal(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
