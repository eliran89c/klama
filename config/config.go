package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// ModelConfig holds the configuration for the agent model
type ModelConfig struct {
	Name      string  `mapstructure:"name" yaml:"name"`
	BaseURL   string  `mapstructure:"base_url" yaml:"base_url"`
	AuthToken string  `mapstructure:"auth_token" yaml:"auth_token"`
	Pricing   Pricing `mapstructure:"pricing" yaml:"pricing"`
}

type Pricing struct {
	Input  float64 `mapstructure:"input" yaml:"input"`
	Output float64 `mapstructure:"output" yaml:"output"`
}

type Config struct {
	Agent ModelConfig `mapstructure:"agent" yaml:"agent"`
}

// Load reads the configuration from the file and environment and returns a Config struct
func Load(configPath string) (*Config, error) {
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("error getting user home directory: %v", err)
		}

		// Try to find config in XDG_CONFIG_HOME
		xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfigHome == "" {
			xdgConfigHome = filepath.Join(home, ".config")
		}
		xdgConfigPath := filepath.Join(xdgConfigHome, "klama", "config.yaml")
		if _, err := os.Stat(xdgConfigPath); os.IsNotExist(err) {
			// Try to find config in the old location (home/.klama.yaml)
			legacyConfigPath := filepath.Join(home, ".klama.yaml")
			if _, err := os.Stat(legacyConfigPath); os.IsNotExist(err) {
				// Create a new XDG config folder and file with default content if no config exists
				if err := createDefaultConfig(xdgConfigPath); err != nil {
					return nil, fmt.Errorf("error creating default config: %v", err)
				}
				configPath = xdgConfigPath
				fmt.Println("[INFO] Created default config file at", xdgConfigPath)
			} else {
				fmt.Println("[WARNING] Using legacy config file location. Please move your config to", xdgConfigPath)
				configPath = legacyConfigPath
			}
		} else {
			configPath = xdgConfigPath
		}
	}

	// read config file
	viper.SetConfigFile(configPath)

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("unable to read config: %v", err)
	}

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

func createDefaultConfig(path string) error {
	configDir := filepath.Dir(path)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	defaultConfig := Config{
		Agent: ModelConfig{
			Name:    "gpt-4o-mini",
			BaseURL: "https://api.openai.com/v1",
			Pricing: Pricing{
				Input:  0.00015,
				Output: 0.0006,
			},
		},
	}
	data, err := yaml.Marshal(defaultConfig)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}
	return nil
}
