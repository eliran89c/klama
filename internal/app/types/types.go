package types

import "context"

// AgentType represents the type of agent available
type AgentType string

// ExecuterType represents the type of executer available
type ExecuterType string

const (
	// agents:
	AgentTypeKubernetes AgentType = "kubernetes"

	// executers:
	ExecuterTypeTerminal ExecuterType = "terminal"
)

// AgentResponse represents the response from the agent
type AgentResponse struct {
	Answer     string `json:"answer,omitempty"`
	RunCommand string `json:"run_command,omitempty"`
	Reason     string `json:"reason_for_command"`
}

// ExecuterResponse represents the response from the executer
type ExecuterResponse struct {
	Result string
	Error  error
}

// Agent represents the agent interface
type Agent interface {
	Iterate(context.Context, string) (AgentResponse, error)
	Reset()
	LogUsage() string
}

// Executer represents the executer interface
type Executer interface {
	Run(context.Context, string) ExecuterResponse
}
