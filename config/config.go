package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// ModelConfig holds the configuration for the agent model
type ModelConfig struct {
	Name      string  `mapstructure:"name"`
	BaseURL   string  `mapstructure:"base_url"`
	AuthToken string  `mapstructure:"auth_token"`
	Pricing   Pricing `mapstructure:"pricing"`
}

type Pricing struct {
	Input  float64 `mapstructure:"input"`
	Output float64 `mapstructure:"output"`
}

type Config struct {
	Agent ModelConfig `mapstructure:"agent"`
}

// Load reads the configuration from the environment and returns a Config struct
func Load() (*Config, error) {
	var config Config

	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config into struct: %v", err)
	}

	// Validate required fields
	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	// Override with environment variables
	if envToken := os.Getenv("KLAMA_AGENT_TOKEN"); envToken != "" {
		config.Agent.AuthToken = envToken
	}

	return &config, nil
}

func validateConfig(config *Config) error {
	if config.Agent.BaseURL == "" {
		return fmt.Errorf("agent base URL is required in the configuration")
	}
	if config.Agent.Name == "" {
		return fmt.Errorf("agent name is required in the configuration")
	}

	return nil
}
