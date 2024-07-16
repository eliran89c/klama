package executer

import (
	"context"
	"reflect"
	"testing"
)

var testExecuterType = TerminalExecuterType{
	AllowedCommands:      []string{"echo", "cat"},
	AllowedSubCommands:   []string{"hello", "world"},
	AllowedPipedCommands: []string{"grep", "wc"},
}

func TestNewTerminalExecuter(t *testing.T) {
	te := NewTerminalExecuter(testExecuterType)
	if te == nil {
		t.Error("NewTerminalExecuter returned nil")
	}
}

func TestTerminalExecuter_Run(t *testing.T) {
	te := NewTerminalExecuter(testExecuterType)
	ctx := context.Background()

	tests := []struct {
		name     string
		command  string
		hasError bool
	}{
		{"Simple echo", "echo hello", false},
		{"Echo with quotes", `echo "hello world"`, false},
		{"Invalid command", "invalid_command", true},
		{"Command with pipe", "echo hello | grep h", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := te.Run(ctx, tt.command)
			if (result.Error != nil) != tt.hasError {
				t.Errorf("Run() error = %v, wantErr %v", result.Error, tt.hasError)
				return
			}
		})
	}
}

func TestTerminalExecuter_Validate(t *testing.T) {
	te := NewTerminalExecuter(testExecuterType)

	tests := []struct {
		name    string
		command string
		wantErr bool
	}{
		{"Valid echo command", "echo hello", false},
		{"Valid echo with subcommand", "echo hello world", false},
		{"Valid command with pipe", "echo hello | grep h", false},
		{"Empty command", "", true},
		{"Invalid main command", "invalid_command", true},
		{"Invalid subcommand", "echo invalid_subcommand", true},
		{"Invalid piped command", "echo hello | invalid_pipe", true},
		{"Command chaining", "echo hello; echo world", true},
		{"Command substitution", "echo `ls`", true},
		{"Command substitution with $()", "echo $(ls)", true},
		{"Redirection", "echo hello > file.txt", true},
		{"Valid command with quotes", `echo "hello world"`, true},
		{"Unmatched quote", `echo "hello world`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := te.Validate(tt.command)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSplitCommandsByPipe(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected []Command
	}{
		{
			"Simple command",
			"echo hello",
			[]Command{{Parts: []string{"echo", "hello"}}},
		},
		{
			"Command with pipe",
			"echo hello | grep h",
			[]Command{
				{Parts: []string{"echo", "hello"}},
				{Parts: []string{"grep", "h"}},
			},
		},
		{
			"Command with quoted pipe",
			`echo "hello | world" | grep hello`,
			[]Command{
				{Parts: []string{"echo", "\"hello | world\""}},
				{Parts: []string{"grep", "hello"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitCommandsByPipe(tt.command)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("splitCommandsByPipe() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSplitCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected []string
	}{
		{"Simple command", "echo hello", []string{"echo", "hello"}},
		{"Command with quotes", `echo "hello world"`, []string{"echo", "\"hello world\""}},
		{"Command with escaped quotes", `echo "hello \"world\""`, []string{"echo", "\"hello \\\"world\\\"\""}},
		{"Command with single quotes", "echo 'hello world'", []string{"echo", "'hello world'"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitCommand(tt.command)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("splitCommand() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTerminalExecuter_validateSingleCommand(t *testing.T) {
	te := NewTerminalExecuter(testExecuterType)

	tests := []struct {
		name          string
		cmd           Command
		isMainCommand bool
		wantErr       bool
	}{
		{"Valid main command", Command{Parts: []string{"echo", "hello"}}, true, false},
		{"Invalid main command", Command{Parts: []string{"invalid", "hello"}}, true, true},
		{"Valid piped command", Command{Parts: []string{"grep", "hello"}}, false, false},
		{"Invalid piped command", Command{Parts: []string{"invalid", "hello"}}, false, true},
		{"Empty command", Command{Parts: []string{}}, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := te.validateSingleCommand(tt.cmd, tt.isMainCommand)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSingleCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTerminalExecuter_validateArgument(t *testing.T) {
	te := NewTerminalExecuter(testExecuterType)

	tests := []struct {
		name    string
		arg     string
		wantErr bool
	}{
		{"Simple argument", "hello", false},
		{"Argument with spaces", "hello world", false},
		{"Argument with quotes", "\"hello world\"", false},
		{"Argument with command chaining", "hello; world", true},
		{"Argument with command substitution", "`ls`", true},
		{"Argument with $()", "$(ls)", true},
		{"Argument with redirection", "hello > file.txt", true},
		{"Argument with unmatched quote", "\"hello world", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := te.validateArgument(tt.arg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateArgument() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
