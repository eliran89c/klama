package llm

import (
	"net/http"
	"testing"

	"github.com/eliran89c/klama/config"
	"github.com/stretchr/testify/assert"
)

func TestNewModel(t *testing.T) {
	client := &http.Client{}
	modelConfig := config.ModelConfig{
		Name:      "test-model",
		BaseURL:   "http://test.com",
		AuthToken: "test-token",
		Pricing: config.Pricing{
			Input:  0.01,
			Output: 0.02,
		},
	}

	model := NewModel(client, modelConfig)

	assert.Equal(t, client, model.Client)
	assert.Equal(t, "test-model", model.Name)
	assert.Equal(t, "http://test.com", model.BaseURL)
	assert.Equal(t, "test-token", model.AuthToken)
	assert.Equal(t, 0.01, model.InputPrice)
	assert.Equal(t, 0.02, model.OutputPrice)
	assert.Empty(t, model.History)
	assert.Equal(t, Usage{}, model.Usage)
}
