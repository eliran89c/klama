package cmd

import (
	"context"
	"net/http"
	"time"

	"github.com/eliran89c/klama/config"
	"github.com/eliran89c/klama/internal/agent"
	"github.com/eliran89c/klama/internal/executer"
	"github.com/eliran89c/klama/internal/llm"
	"github.com/eliran89c/klama/internal/logger"
)

func run(prompt string, debug, showUsage bool) {
	// setup logger
	log := logger.New(debug)

	// set up context
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// set up http client
	client := &http.Client{Timeout: 30 * time.Second}

	// load config
	cfg, err := config.Load()
	if err != nil {
		log.Error("Failed to load config: %v", err)
		return
	}

	// setup executer
	var exec agent.Executer
	var validationModel *llm.Model

	if cfg.UseModelForValidation() {
		validationModel = llm.NewModel(client, cfg.Validation)
		exec, err = executer.NewLLMExecuter(validationModel, log)
		if err != nil {
			log.Error("Failed to create executer: %v", err)
			return
		}
	} else {
		exec = executer.NewUserExecuter(log)
	}

	// setup agent
	agentModel := llm.NewModel(client, cfg.Agent)
	agent, err := agent.New(agentModel, log)
	if err != nil {
		log.Error("Failed to create agent: %v", err)
		return
	}

	// start session
	log.Info("Starting Kubernetes debugging session...")
	resp, err := agent.StartSession(ctx, exec, prompt)
	if err != nil {
		log.Error("Session failed: %v", err)
	}

	// output result
	if resp != "" {
		log.Print("\n") // Add an extra newline for spacing
		log.Result("Result:\n%s", resp)
	}

	// print usage
	if showUsage {
		log.Print("\n") // Add an extra newline for spacing
		log.CostBreakdown("Session Cost Breakdown:")
		log.Print(agentModel.LogUsage())
		if cfg.UseModelForValidation() {
			log.Print(validationModel.LogUsage())
		}
	}
}
