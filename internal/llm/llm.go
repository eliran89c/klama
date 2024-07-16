package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"github.com/eliran89c/klama/internal/logger"
)

// SetSystemPrompt sets or updates the system prompt in the model's history.
func (m *Model) SetSystemPrompt(prompt string) {
	if len(m.History) == 0 {
		m.addMessage(SystemRole, prompt)
		return
	}

	if m.History[0].Role == SystemRole {
		m.History[0] = Message{Role: SystemRole, Content: prompt}
	} else {
		m.History = append([]Message{{Role: SystemRole, Content: prompt}}, m.History...)
	}
}

// GuidedAsk sends a prompt to the model, receives a response, and if the response is not valid JSON, it retries the prompt
func (m *Model) GuidedAsk(ctx context.Context, prompt string, maxAttempts int, result interface{}) error {
	resultValue := reflect.ValueOf(result)
	if resultValue.Kind() != reflect.Ptr || resultValue.IsNil() {
		return fmt.Errorf("result must be a non-nil pointer")
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := m.Ask(ctx, prompt, 0)
		if err != nil {
			return fmt.Errorf("failed to interact with the model: %w", err)
		}

		if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), result); err != nil {
			if attempt == maxAttempts {
				return fmt.Errorf("failed to parse model response after %d attempts: %w", maxAttempts, err)
			}
			prompt = fmt.Sprintf("Error: Failed to parse your response. Answer only with the requested JSON format. The error was: %v\n\nOriginal prompt: %s\nDo not apologize or mention the formatting error in your response", err, prompt)
			continue
		}

		return nil
	}

	return fmt.Errorf("failed to get a valid response after %d attempts", maxAttempts)
}

// Ask sends a prompt to the model and returns the response.
func (m *Model) Ask(ctx context.Context, prompt string, temperature float64) (*ChatResponse, error) {
	logger.Debugf("Asking model %s: %s", m.Name, prompt)

	data, err := json.Marshal(ChatRequest{
		Model:       m.Name,
		Temperature: temperature,
		Messages:    append(m.History, Message{Role: UserRole, Content: prompt}),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal chat request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.BaseURL+"/chat/completions", bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.AuthToken)

	resp, err := m.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d\n%s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal chat response: %w", err)
	}

	logger.Debugf("Model %s responded: %s", m.Name, chatResp.Choices[0].Message.Content)

	// Update the model's state with the response
	m.addMessage(UserRole, prompt)
	m.updateUsage(chatResp.Usage)
	m.addMessage(AssistantRole, chatResp.Choices[0].Message.Content)

	return &chatResp, nil
}

func (m *Model) addMessage(role Role, content string) {
	m.History = append(m.History, Message{Role: role, Content: content})
}

func (m *Model) updateUsage(usage Usage) {
	m.Usage.TotalTokens += usage.TotalTokens
	m.Usage.PromptTokens += usage.PromptTokens
	m.Usage.CompletionTokens += usage.CompletionTokens
}

// LogUsage returns a string representation of the model's usage statistics.
func (m *Model) LogUsage() string {
	inputPrice := m.InputPrice * float64(m.Usage.PromptTokens) / 1000
	outputPrice := m.OutputPrice * float64(m.Usage.CompletionTokens) / 1000

	return fmt.Sprintf("%s: %.4f$ for input(%d), %.4f$ for output(%d)",
		m.Name, inputPrice, m.Usage.PromptTokens, outputPrice, m.Usage.CompletionTokens)
}
