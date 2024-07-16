package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eliran89c/klama/config"
	"github.com/eliran89c/klama/internal/agent"
	"github.com/eliran89c/klama/internal/executer"
	"github.com/eliran89c/klama/internal/llm"
	"github.com/eliran89c/klama/internal/logger"
	"github.com/eliran89c/klama/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	k8sCmd = &cobra.Command{
		Use:   "k8s",
		Short: "Interact with the Kubernetes debugging assistant",
		Long: `Interact with the Kubernetes debugging assistant to troubleshoot and resolve issues in
Kubernetes clusters.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			debug := viper.GetBool("debug")

			if debug {
				//TODO: get debugger file location from user
				file, err := os.Create("klama.debug")
				if err != nil {
					return fmt.Errorf("failed to create debug file: %w", err)
				}
				logger.Init(file)
				defer file.Close()
			} else {
				logger.Init(io.Discard)
			}

			client := &http.Client{}

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			llmModel := llm.NewModel(client, cfg.Agent)

			k8sAgent, err := agent.New(llmModel, agent.AgentTypeKubernetes)
			if err != nil {
				return fmt.Errorf("failed to initialize agent: %w", err)
			}

			exec := executer.NewTerminalExecuter(executer.KubernetesExecuterType)

			uiConfig := ui.Config{
				Agent:    k8sAgent,
				Executer: exec,
			}

			p := tea.NewProgram(
				ui.InitialModel(uiConfig),
				tea.WithAltScreen(),
				tea.WithMouseCellMotion(),
			)

			if _, err := p.Run(); err != nil {
				return fmt.Errorf("error running program: %w", err)
			}

			return nil
		},
	}
)
