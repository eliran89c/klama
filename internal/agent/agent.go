package agent

import (
	"context"
	"fmt"

	"github.com/eliran89c/klama/internal/llm"
	"github.com/eliran89c/klama/internal/logger"
)

const (
	numberOfQueries         = 7
	modelCorrectionAttempts = 3
	systemPrompt            = `
You are an expert Kubernetes (K8s) debugging assistant, designed to help users troubleshoot and resolve issues in their Kubernetes clusters. Your primary task is to assist in identifying and analyzing Kubernetes-related problems efficiently and effectively.

Core Guidelines that you must follow:
1. Always output your responses in the following JSON format:
   {
     "final_answer": string,
     "run_command": string,
     "need_more_data": bool,
     "reason_for_command": string
   }
2. Never provide a final answer until all necessary data has been gathered and you are certain of the issue.
3. Request the user to run specific kubectl commands to gather information. Only suggest one command at a time. Explain why the command is needed in the "reason_for_command" field.
4. Do not provide explanations or comments outside the JSON schema. All information should be contained within the specified fields.
5. You are allowed to get, list, and describe any resource except secrets. You can also get pod logs if needed. Examples of allowed commands:
   - kubectl get pods -A
   - kubectl describe deployment myapp -n mynamespace
   - kubectl logs mypod -n mynamespace
6. IMPORTANT: If you need to pull logs, limit the output up to a maximum of 150 lines. Use the '--tail=150' flag with the 'kubectl logs' command.
7. You are not allowed to run any write/mutation commands like create, update, patch, or delete.
8. IMPORTANT: Never suggest commands that switch Kubernetes contexts. Assume all operations are performed within the current context.
9. If the resource scope is set to a namespace, ensure your commands search across all namespaces to comprehensively address the issue. Use the '-A' or '--all-namespaces' flag when appropriate.
10. If you need to find logs or data for multiple resources, start with the first one and proceed sequentially.
11. Always run the next logical command based on the information you have. Never make assumptions about the state of the cluster or the cause of the issue without verifying.
12. If you need to collect logs from multiple pods, start with the first one and proceed sequentially until you have all the necessary data.
13. If the user asks a non-K8s related question, end the session while using the response schema.
`
)

// Executer interface defines the methods required to execute commands.
type Executer interface {
	Run(ctx context.Context, command string) (string, error)
	Validate(ctx context.Context, command string) (bool, string, error)
}

// AgentResponse represents the structured response from the AI model.
type AgentResponse struct {
	FinalAnswer  string `json:"final_answer,omitempty"`
	RunCommand   string `json:"run_command,omitempty"`
	NeedMoreData bool   `json:"need_more_data,omitempty"`
	Reason       string `json:"reason_for_command"`
}

// Agent represents the Kubernetes debugging assistant.
type Agent struct {
	Logger     *logger.Logger
	AgentModel *llm.Model
}

// New creates a new Agent with the given options.
func New(agent *llm.Model, log *logger.Logger) (*Agent, error) {
	if agent == nil {
		return nil, fmt.Errorf("agent model is required")
	}

	agent.SetSystemPrompt(systemPrompt)

	// Disable logger output if not provided
	if log == nil {
		log = logger.EmptyLogger()
	}

	return &Agent{
		Logger:     log,
		AgentModel: agent,
	}, nil
}

// StartSession begins a debugging session with the given context and query.
func (ag *Agent) StartSession(ctx context.Context, exec Executer, q string) (string, error) {
	ag.Logger.Debug("Agent model: %s", ag.AgentModel.Name)
	ag.Logger.Debug("Query: %s", q)
	ag.Logger.Info("Analyzing your Kubernetes issue...")

	modelResp := AgentResponse{NeedMoreData: true}
	prompt := q
	executedCommands := make(map[string]string) // Cache for executed commands
	maxQueries := numberOfQueries

	for queryCount := 0; modelResp.NeedMoreData && queryCount < maxQueries; queryCount++ {
		ag.Logger.Debug("Query count: %d", queryCount+1)
		ag.Logger.Debug("Prompt:\n%s", prompt)

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			var err error
			modelResp, err = ag.interactWithModel(ctx, prompt)
			if err != nil {
				return "", err
			}

			if modelResp.RunCommand != "" && modelResp.NeedMoreData {
				prompt = ag.handleCommand(ctx, exec, modelResp.RunCommand, executedCommands)

			} else if modelResp.NeedMoreData {
				prompt = "Please suggest a command to run or end the session."
			}
		}
	}

	if modelResp.NeedMoreData {
		ag.Logger.Debug("Reached maximum number of queries (%d)", maxQueries)
		return "Analysis incomplete. Reached maximum number of queries.", nil
	}

	ag.Logger.Success("Analysis complete.")
	return modelResp.FinalAnswer, nil
}

// interactWithModel sends a prompt to the AI model and returns the response.
func (ag *Agent) interactWithModel(ctx context.Context, prompt string) (AgentResponse, error) {
	var modelResp AgentResponse
	ag.Logger.StartThinking()
	err := ag.AgentModel.GuidedAsk(ctx, prompt, modelCorrectionAttempts, &modelResp)
	ag.Logger.StopThinking()

	if err != nil {
		return AgentResponse{}, err
	}

	ag.Logger.Debug("Model response:\n%+v", modelResp)

	return modelResp, nil
}

// handleCommand executes a command and returns the result as a prompt for the next iteration.
func (ag *Agent) handleCommand(ctx context.Context, exec Executer, command string, executedCommands map[string]string) string {
	if output, exists := executedCommands[command]; exists {
		ag.Logger.Debug("Command already executed: `%s`", command)
		return output
	}

	ag.Logger.Info("Model asks to run command: `%s`", command)

	// Validate the command
	isReadOnly, reason, err := exec.Validate(ctx, command)
	if err != nil {
		return fmt.Sprintf("Failed to validate command: %s", err.Error())
	}

	if !isReadOnly {
		return reason
	}

	// Execute the command
	ag.Logger.Debug("Executing command: `%s`", command)
	ag.Logger.StartThinking()
	out, err := exec.Run(ctx, command)
	ag.Logger.StopThinking()
	if err != nil {
		return fmt.Sprintf("Command failed: %s", err.Error())
	}

	if out == "" {
		out = "No output"
	}
	executedCommands[command] = out
	return out
}
