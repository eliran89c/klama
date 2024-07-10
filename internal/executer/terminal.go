package executer

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// TerminalExecuter is a simple executer that manages command execution and caching.
type TerminalExecuter struct {
	executedCommands map[string]string
}

// NewTerminalExecuter creates a new TerminalExecuter.
func NewTerminalExecuter() *TerminalExecuter {
	return &TerminalExecuter{
		executedCommands: make(map[string]string),
	}
}

// Run executes a command and returns the output.
// It caches the results of previously executed commands.
func (te *TerminalExecuter) Run(ctx context.Context, command string) ExecuterResponse {
	if output, exists := te.executedCommands[command]; exists {
		return ExecuterResponse{
			Result: output,
			Error:  nil,
		}
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	output, err := cmd.CombinedOutput()
	resp := strings.TrimSpace(string(output))

	result := ExecuterResponse{Result: resp}
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			result.Error = fmt.Errorf("command execution timed out: %w", ctx.Err())
		} else {
			result.Error = fmt.Errorf("command execution failed: %w", err)
		}
	} else {
		te.executedCommands[command] = resp
	}

	return result
}
