package executer

import (
	"context"
	"fmt"
	"os/exec"
	"slices"
	"strings"
	"unicode"
)

// Validation errors
var (
	ErrEmptyCommand         = fmt.Errorf("command is empty")
	ErrCommandChaining      = fmt.Errorf("command chaining is not allowed")
	ErrCommandSubstitution  = fmt.Errorf("command substitution is not allowed")
	ErrRedirection          = fmt.Errorf("redirection is not allowed")
	ErrUnmatchedQuote       = fmt.Errorf("unmatched quote in argument")
	ErrInvalidMainCommand   = fmt.Errorf("main command is not valid")
	ErrCommandNotAllowed    = fmt.Errorf("command is not allowed")
	ErrSubCommandNotAllowed = fmt.Errorf("sub command is not allowed")
)

type Command struct {
	Parts []string
}

// TerminalExecuterType represents the type of the terminal executer.
type TerminalExecuterType struct {
	AllowedCommands      []string
	AllowedSubCommands   []string
	AllowedPipedCommands []string
}

var (
	// KubernetesExecuterType represents the type of the terminal executer for kubectl commands.
	KubernetesExecuterType = TerminalExecuterType{
		AllowedCommands: []string{"kubectl"},
		AllowedSubCommands: []string{
			"get",
			"describe",
			"logs",
			"top",
			"explain",
		},
		AllowedPipedCommands: []string{
			"grep",
			"awk",
			"sort",
			"uniq",
			"head",
			"tail",
			"cut",
		},
	}
)

// TerminalExecuter is a simple executer that manages shell command execution and caching.
type TerminalExecuter struct {
	executedCommands map[string]string
	executerType     TerminalExecuterType
}

// NewTerminalExecuter creates a new TerminalExecuter.
func NewTerminalExecuter(executerType TerminalExecuterType) *TerminalExecuter {
	return &TerminalExecuter{
		executedCommands: make(map[string]string),
		executerType:     executerType,
	}
}

// Run executes a command and returns the output.
// It caches the results of previously executed commands.
func (tx *TerminalExecuter) Run(ctx context.Context, command string) ExecuterResponse {
	if output, exists := tx.executedCommands[command]; exists {
		return ExecuterResponse{Result: output}
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	output, err := cmd.CombinedOutput()
	resp := strings.TrimSpace(string(output))

	result := ExecuterResponse{Result: resp}
	switch {
	case err == nil:
		tx.executedCommands[command] = resp
	case ctx.Err() == context.DeadlineExceeded:
		result.Error = fmt.Errorf("command execution timed out: %w", ctx.Err())
	default:
		result.Error = fmt.Errorf("command execution failed: %w", err)
	}

	return result
}

// Validate validates a command.
func (tx *TerminalExecuter) Validate(command string) error {

	if command == "" {
		return ErrEmptyCommand
	}

	if _, exists := tx.executedCommands[command]; exists {
		return nil
	}

	cmds := splitCommandsByPipe(command)
	for i, cmd := range cmds {
		if err := tx.validateSingleCommand(cmd, i == 0); err != nil {
			return err
		}
	}

	return nil
}

func splitCommandsByPipe(command string) []Command {
	var commands []Command
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false

	for _, char := range command {
		if escaped {
			current.WriteRune(char)
			escaped = false
			continue
		}

		switch char {
		case '\\':
			escaped = true
			current.WriteRune(char)
		case '\'':
			inSingleQuote = !inSingleQuote
			current.WriteRune(char)
		case '"':
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			}
			current.WriteRune(char)
		case '|':
			if !inSingleQuote && !inDoubleQuote {
				commands = append(commands, Command{Parts: splitCommand(strings.TrimSpace(current.String()))})
				current.Reset()
			} else {
				current.WriteRune(char)
			}
		default:
			current.WriteRune(char)
		}
	}

	if current.Len() > 0 {
		commands = append(commands, Command{Parts: splitCommand(strings.TrimSpace(current.String()))})
	}

	return commands
}

func (tx *TerminalExecuter) validateCommandArguments(args []string) error {
	for _, arg := range args {
		if err := tx.validateArgument(arg); err != nil {
			return err
		}
	}
	return nil
}

func (tx *TerminalExecuter) validateArgument(arg string) error {
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false

	for i, char := range arg {
		if escaped {
			escaped = false
			continue
		}

		switch char {
		case '\\':
			escaped = true
		case '\'':
			if !inDoubleQuote {
				inSingleQuote = !inSingleQuote
			}
		case '"':
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			}
		case ';', '&':
			if !inSingleQuote && !inDoubleQuote {
				return ErrCommandChaining
			}
		case '`':
			if !inSingleQuote && !inDoubleQuote {
				return ErrCommandSubstitution
			}
		case '$':
			if !inSingleQuote && !inDoubleQuote && i+1 < len(arg) && arg[i+1] == '(' {
				return ErrCommandSubstitution
			}
		case '>', '<':
			if !inSingleQuote && !inDoubleQuote {
				return ErrRedirection
			}
		}
	}

	if inSingleQuote || inDoubleQuote {
		return ErrUnmatchedQuote
	}

	return nil
}

func splitCommand(command string) []string {
	var parts []string
	var current strings.Builder
	inQuote := rune(0)
	escaped := false

	for _, char := range command {
		if escaped {
			current.WriteRune(char)
			escaped = false
			continue
		}

		if char == '\\' {
			escaped = true
			current.WriteRune(char)
			continue
		}

		if inQuote != 0 {
			if char == inQuote {
				inQuote = 0
			}
			current.WriteRune(char)
		} else if char == '\'' || char == '"' {
			inQuote = char
			current.WriteRune(char)
		} else if unicode.IsSpace(char) {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		} else {
			current.WriteRune(char)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

func (tx *TerminalExecuter) validateSingleCommand(cmd Command, isMainCommand bool) error {
	if len(cmd.Parts) == 0 {
		return ErrEmptyCommand
	}

	checkSubCmd := false
	minNumParts := 1
	if len(tx.executerType.AllowedSubCommands) > 0 {
		checkSubCmd = true
		minNumParts = 2
	}

	if isMainCommand {
		if len(cmd.Parts) < minNumParts {
			return ErrInvalidMainCommand
		}
		if !slices.Contains(tx.executerType.AllowedCommands, cmd.Parts[0]) {
			return fmt.Errorf("%w: %s", ErrCommandNotAllowed, cmd.Parts[0])
		}
		if checkSubCmd {
			if !slices.Contains(tx.executerType.AllowedSubCommands, cmd.Parts[1]) {
				return fmt.Errorf("%w: %s", ErrSubCommandNotAllowed, cmd.Parts[1])
			}
		}
	} else if !slices.Contains(tx.executerType.AllowedPipedCommands, cmd.Parts[0]) {
		return fmt.Errorf("%w: %s", ErrCommandNotAllowed, cmd.Parts[0])
	}

	return tx.validateCommandArguments(cmd.Parts)
}
