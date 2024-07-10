package agent

import (
	"context"
	"fmt"

	"github.com/eliran89c/klama/internal/llm"
)

const (
	modelCorrectionAttempts = 3
)

// AgentResponse represents the response from the agent
type AgentResponse struct {
	Answer     string `json:"answer,omitempty"`
	RunCommand string `json:"run_command,omitempty"`
	Reason     string `json:"reason_for_command"`
}

// Agent represents an AI assistant.
type Agent struct {
	AgentModel *llm.Model
	Type       AgentType
}

// New creates a new Agent with the given options.
func New(agent *llm.Model, agentType AgentType) (*Agent, error) {
	if agent == nil {
		return nil, fmt.Errorf("agent model is required")
	}

	agent.SetSystemPrompt(string(agentType))

	return &Agent{
		AgentModel: agent,
		Type:       agentType,
	}, nil
}

// Iterate sends a prompt to the AI model and returns the response.
func (ag *Agent) Iterate(ctx context.Context, prompt string) (AgentResponse, error) {
	if prompt == "" {
		return AgentResponse{}, fmt.Errorf("prompt is required")
	}

	var modelResp AgentResponse
	err := ag.AgentModel.GuidedAsk(ctx, prompt, modelCorrectionAttempts, &modelResp)
	if err != nil {
		return AgentResponse{}, err
	}

	return modelResp, nil
}

// Reset clears the agent's history and resets the conversation.
func (ag *Agent) Reset() {
	ag.AgentModel.History = []llm.Message{}
	ag.AgentModel.SetSystemPrompt(
		string(ag.Type),
	)
}

// LogUsage returns the agent's model usage log.
func (ag *Agent) LogUsage() string {
	return ag.AgentModel.LogUsage()
}
