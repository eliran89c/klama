package executer

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewTerminalExecuter(t *testing.T) {
	ux := NewTerminalExecuter()
	assert.NotNil(t, ux)
	assert.Empty(t, ux.executedCommands)
}

func TestTerminalExecuter_Run(t *testing.T) {
	ux := NewTerminalExecuter()

	tests := []struct {
		name    string
		command string
		wantErr bool
	}{
		{
			name:    "Simple echo command",
			command: "echo 'Hello, World!'",
			wantErr: false,
		},
		{
			name:    "Invalid command",
			command: "invalid_command",
			wantErr: true,
		},
		{
			name:    "Long-running command with timeout",
			command: "sleep 5",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			result := ux.Run(ctx, tt.command)

			if tt.wantErr {
				assert.Error(t, result.Error)
			} else {
				assert.NoError(t, result.Error)
				assert.NotEmpty(t, result.Result)
			}
		})
	}
}

func TestTerminalExecuter_Run_CachedCommand(t *testing.T) {
	ux := NewTerminalExecuter()
	command := "echo 'Cached command'"

	// Run the command for the first time
	result1 := ux.Run(context.Background(), command)
	assert.NoError(t, result1.Error)
	assert.NotEmpty(t, result1.Result)

	// Run the same command again
	result2 := ux.Run(context.Background(), command)
	assert.NoError(t, result2.Error)
	assert.Equal(t, result1.Result, result2.Result)
}

func TestTerminalExecuter_Run_ContextCancellation(t *testing.T) {
	ux := NewTerminalExecuter()
	command := "sleep 5" // Simulate a long-running command

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result := ux.Run(ctx, command)

	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "context deadline exceeded")
}
