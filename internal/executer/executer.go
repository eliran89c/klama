package executer

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/eliran89c/klama/internal/llm"
	"github.com/eliran89c/klama/internal/logger"
)

const (
	modelCorrectionAttempts = 3
	systemPrompt            = `
You are a classification model designed to validate Kubernetes (K8s) commands provided by the user. Your primary task is to ensure that the commands are valid, safe to run, and are read-only operations.

Core Guidelines that you must follow:
1. Always output your responses in the following JSON format:
   {
     "is_read_only": bool,
     "reason": string
   }
2. Do not provide explanations or comments outside the JSON schema. All information should be contained within the specified fields.
3. Explain why the command is read-only or not in the "reason" field.
4. Allowed operations:
   - Getting, listing, and describing resources (except secrets)
   - Getting pod logs
5. Disallowed operations:
   - Creating, updating, deleting, or modifying resources
   - Reading sensitive information like secrets
   - IMPORTANT: Any command that switches Kubernetes contexts
`
)

// LLMExecuter is an executer that uses a language model to verify if a command is safe to run.
type LLMExecuter struct {
	Model  *llm.Model
	Logger *logger.Logger
}

// UserExecuter is a simple executer that asks the user to verify if a command is safe to run.
type UserExecuter struct {
	Logger *logger.Logger
}

// ModelResp represents the structured response from the classification model.
type ModelResp struct {
	IsReadOnly bool   `json:"is_read_only"`
	Reason     string `json:"reason"`
}

func NewLLMExecuter(model *llm.Model, log *logger.Logger) (*LLMExecuter, error) {
	// Disable logger output if not provided
	if log == nil {
		log = logger.EmptyLogger()
	}

	if model == nil {
		return nil, fmt.Errorf("model is required")
	}

	model.SetSystemPrompt(systemPrompt)
	return &LLMExecuter{
		Model:  model,
		Logger: log,
	}, nil
}

func NewUserExecuter(log *logger.Logger) *UserExecuter {
	// Disable logger output if not provided
	if log == nil {
		log = logger.EmptyLogger()
	}

	return &UserExecuter{
		Logger: log,
	}
}

// Validate method for LLMExecuter
func (e *LLMExecuter) Validate(ctx context.Context, command string) (bool, string, error) {
	prompt := fmt.Sprintf("Classify the following command: %s", command)

	var modelResp ModelResp
	e.Logger.Debug("Classifying model: %s", e.Model.Name)
	e.Logger.StartThinking()
	err := e.Model.GuidedAsk(ctx, prompt, modelCorrectionAttempts, &modelResp)
	e.Logger.StopThinking()

	if err != nil {
		e.Logger.Debug("Failed to classify the command: %v", err)
		return false, "", fmt.Errorf("failed to classify the command: %w", err)
	}

	e.Logger.Debug("Classifying model response:\n%+v", modelResp)

	return modelResp.IsReadOnly, modelResp.Reason, nil
}

// Validate method for UserExecuter
func (e *UserExecuter) Validate(ctx context.Context, command string) (bool, string, error) {
	fmt.Printf("Do you want to run the following command? (yes/no)\n\t> %s\n", command)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, "", fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response == "yes" {
		return true, "User approved the command", nil
	}
	return false, "User did not approve the command, please suggest different command or end the session.", nil
}

// Execute method (common for both executors)
func (e *LLMExecuter) Run(ctx context.Context, command string) (string, error) {
	return executeCommand(ctx, command)
}

func (e *UserExecuter) Run(ctx context.Context, command string) (string, error) {
	return executeCommand(ctx, command)
}

// executeCommand is a helper function to execute the command
func executeCommand(ctx context.Context, command string) (string, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	output, err := cmd.CombinedOutput()
	resp := strings.TrimSpace(string(output))

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("command execution timed out")
		}
		return resp, fmt.Errorf("command execution failed: %w\n%v", err, resp)
	}

	return resp, nil
}
