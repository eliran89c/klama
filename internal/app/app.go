// Package app provides the main application logic for Klama.
package app

import (
	"fmt"
	"net/http"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eliran89c/klama/config"
	kagent "github.com/eliran89c/klama/internal/agent/kubernetes"
	"github.com/eliran89c/klama/internal/app/types"
	"github.com/eliran89c/klama/internal/executer"
	"github.com/eliran89c/klama/internal/llm"
	"github.com/eliran89c/klama/internal/ui"
)

// Run initializes and runs the Klama application.
func Run(debug bool, agentType types.AgentType, execType types.ExecuterType) error {
	client := &http.Client{Timeout: 45 * time.Second}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	agent, err := initAgent(agentType, client, cfg.Agent)
	if err != nil {
		return fmt.Errorf("failed to initialize agent: %w", err)
	}

	exec, err := initExecuter(execType)
	if err != nil {
		return fmt.Errorf("failed to initialize executer: %w", err)
	}

	uiConfig := ui.Config{
		Agent:    agent,
		Executer: exec,
		Debug:    debug,
	}

	p := tea.NewProgram(ui.InitialModel(uiConfig), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running program: %w", err)
	}

	return nil
}

func initAgent(agentType types.AgentType, client *http.Client, cfg config.ModelConfig) (types.Agent, error) {
	switch agentType {
	case types.AgentTypeKubernetes:
		agentModel := llm.NewModel(client, cfg)
		return kagent.New(agentModel)
	default:
		return nil, fmt.Errorf("unsupported agent type: %v", agentType)
	}
}

func initExecuter(execType types.ExecuterType) (types.Executer, error) {
	switch execType {
	case types.ExecuterTypeTerminal:
		return executer.NewTerminalExecuter(), nil
	default:
		return nil, fmt.Errorf("unsupported executer type: %v", execType)
	}
}
