package config

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	viper.Reset()
	viper.SetConfigType("yaml")

	viper.Set("agent.name", "test-agent")
	viper.Set("agent.base_url", "http://test.com")
	viper.Set("agent.auth_token", "test-token")
	viper.Set("agent.pricing.input", 0.01)
	viper.Set("agent.pricing.output", 0.02)

	cfg, err := Load("")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "test-agent", cfg.Agent.Name)
	assert.Equal(t, "http://test.com", cfg.Agent.BaseURL)
	assert.Equal(t, "test-token", cfg.Agent.AuthToken)
	assert.Equal(t, 0.01, cfg.Agent.Pricing.Input)
	assert.Equal(t, 0.02, cfg.Agent.Pricing.Output)
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "Valid config",
			config: &Config{
				Agent: ModelConfig{
					Name:    "test-agent",
					BaseURL: "http://test.com",
				},
			},
			wantErr: false,
		},
		{
			name: "Missing agent base URL",
			config: &Config{
				Agent: ModelConfig{
					Name: "test-agent",
				},
			},
			wantErr: true,
		},
		{
			name: "Missing agent name",
			config: &Config{
				Agent: ModelConfig{
					BaseURL: "http://test.com",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoadWithEnvironmentVariables(t *testing.T) {
	os.Setenv("KLAMA_AGENT_TOKEN", "env-agent-token")
	os.Setenv("KLAMA_VALIDATION_TOKEN", "env-validation-token")
	defer func() {
		os.Unsetenv("KLAMA_AGENT_TOKEN")
		os.Unsetenv("KLAMA_VALIDATION_TOKEN")
	}()

	viper.Reset()
	viper.SetConfigType("yaml")

	viper.Set("agent.name", "test-agent")
	viper.Set("agent.base_url", "http://test.com")
	viper.Set("validation.name", "test-validation")
	viper.Set("validation.base_url", "http://validation.com")

	cfg, err := Load("")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "env-agent-token", cfg.Agent.AuthToken)
}
