package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eliran89c/klama/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	model := &llm.Model{}

	ag, err := New(model, AgentTypeKubernetes)
	assert.NoError(t, err)
	assert.NotNil(t, ag)
	assert.Equal(t, model, ag.AgentModel)

	// Test with nil model
	ag, err = New(nil, AgentTypeKubernetes)
	assert.Error(t, err)
	assert.Nil(t, ag)
}

func TestAgent_Iterate(t *testing.T) {
	testCases := []struct {
		name          string
		mockResponses []string
		wantResp      AgentResponse
		wantErr       bool
	}{
		{
			name:          "successful interaction",
			mockResponses: []string{`{"answer": "Test answer", "command_to_run": ""}`},
			wantResp:      AgentResponse{Answer: "Test answer", RunCommand: ""},
			wantErr:       false,
		},
		{
			name:          "invalid JSON response",
			mockResponses: []string{`invalid JSON`, `{"answer": "Corrected answer", "command_to_run": ""}`},
			wantResp:      AgentResponse{Answer: "Corrected answer", RunCommand: ""},
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

			ag, err := New(model, AgentTypeKubernetes)
			require.NoError(t, err)

			got, err := ag.Iterate(context.Background(), "Test prompt")

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantResp, got)
			}
		})
	}
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

	ag, err := New(model, AgentTypeKubernetes)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err = ag.Iterate(ctx, "Test query")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}
