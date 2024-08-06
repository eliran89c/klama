package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetSystemPrompt(t *testing.T) {
	model := &Model{}

	model.SetSystemPrompt("Test prompt")
	assert.Equal(t, 1, len(model.History))
	assert.Equal(t, SystemRole, model.History[0].Role)
	assert.Equal(t, "Test prompt", model.History[0].Content)

	model.SetSystemPrompt("Updated prompt")
	assert.Equal(t, 1, len(model.History))
	assert.Equal(t, SystemRole, model.History[0].Role)
	assert.Equal(t, "Updated prompt", model.History[0].Content)
}

func TestAsk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices":[{"message":{"content":"Test response"}}],"usage":{"total_tokens":10,"prompt_tokens":5,"completion_tokens":5}}`))
	}))
	defer server.Close()

	model := &Model{
		Client: server.Client(),
		URL:    server.URL,
		Name:   "test-model",
		AuthToken: AuthToken{
			Key:   "test-header",
			Value: "test-token",
		},
	}

	resp, err := model.Ask(context.Background(), "Test prompt", 0.5)
	assert.NoError(t, err)
	assert.Equal(t, "Test response", resp.Choices[0].Message.Content)
	assert.Equal(t, 10, resp.Usage.TotalTokens)
	assert.Equal(t, 5, resp.Usage.PromptTokens)
	assert.Equal(t, 5, resp.Usage.CompletionTokens)
}

func TestLogUsage(t *testing.T) {
	model := &Model{
		Name:        "test-model",
		InputPrice:  0.01,
		OutputPrice: 0.02,
		Usage: Usage{
			TotalTokens:      100,
			PromptTokens:     50,
			CompletionTokens: 50,
		},
	}

	usage := model.LogUsage()
	assert.Contains(t, usage, "test-model")
	assert.Contains(t, usage, "0.0005$")
	assert.Contains(t, usage, "0.0010$")
}

func TestAddMessage(t *testing.T) {
	model := &Model{}

	model.addMessage(UserRole, "Test message")
	assert.Equal(t, 1, len(model.History))
	assert.Equal(t, UserRole, model.History[0].Role)
	assert.Equal(t, "Test message", model.History[0].Content)
}

func TestUpdateUsage(t *testing.T) {
	model := &Model{}

	model.updateUsage(Usage{
		TotalTokens:      100,
		PromptTokens:     50,
		CompletionTokens: 50,
	})

	assert.Equal(t, 100, model.Usage.TotalTokens)
	assert.Equal(t, 50, model.Usage.PromptTokens)
	assert.Equal(t, 50, model.Usage.CompletionTokens)

	model.updateUsage(Usage{
		TotalTokens:      50,
		PromptTokens:     25,
		CompletionTokens: 25,
	})

	assert.Equal(t, 150, model.Usage.TotalTokens)
	assert.Equal(t, 75, model.Usage.PromptTokens)
	assert.Equal(t, 75, model.Usage.CompletionTokens)
}

func TestModel_GuidedAsk(t *testing.T) {
	type TestResponse struct {
		Message string `json:"message"`
		Number  int    `json:"number"`
	}

	tests := []struct {
		name            string
		serverResponses []string
		maxAttempts     int
		expectedResult  TestResponse
		expectedError   string
	}{
		{
			name:            "Successful response on first attempt",
			serverResponses: []string{`{"message": "Hello", "number": 42}`},
			maxAttempts:     3,
			expectedResult:  TestResponse{Message: "Hello", Number: 42},
			expectedError:   "",
		},
		{
			name:            "Successful response after invalid JSON",
			serverResponses: []string{`invalid json`, `{"message": "Retry", "number": 24}`},
			maxAttempts:     3,
			expectedResult:  TestResponse{Message: "Retry", Number: 24},
			expectedError:   "",
		},
		{
			name:            "Failure after max attempts",
			serverResponses: []string{`invalid json`, `still invalid`, `{"incomplete": true`},
			maxAttempts:     3,
			expectedResult:  TestResponse{},
			expectedError:   "failed to parse model response after 3 attempts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock server
			serverResponseIndex := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				response := map[string]interface{}{
					"choices": []map[string]interface{}{
						{
							"message": map[string]interface{}{
								"content": tt.serverResponses[serverResponseIndex],
							},
						},
					},
				}
				json.NewEncoder(w).Encode(response)
				if serverResponseIndex < len(tt.serverResponses)-1 {
					serverResponseIndex++
				}
			}))
			defer server.Close()

			model := &Model{
				Client: server.Client(),
				URL:    server.URL,
				Name:   "test-model",
				AuthToken: AuthToken{
					Key:   "test-header",
					Value: "test-token",
				},
			}

			ctx := context.Background()

			var result TestResponse
			err := model.GuidedAsk(ctx, "Test prompt", tt.maxAttempts, &result)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}
