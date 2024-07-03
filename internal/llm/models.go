package llm

import (
	"net/http"

	"github.com/eliran89c/klama/config"
)

// Model represents a language model and its associated data.
type Model struct {
	Client      *http.Client
	Name        string
	BaseURL     string
	AuthToken   string
	InputPrice  float64 // price per 1K input tokens
	OutputPrice float64 // price per 1K output tokens
	History     []Message
	Usage       Usage
}

// NewModel creates a new Model instance.
func NewModel(client *http.Client, modelConfig config.ModelConfig) *Model {
	return &Model{
		Client:      client,
		Name:        modelConfig.Name,
		BaseURL:     modelConfig.BaseURL,
		AuthToken:   modelConfig.AuthToken,
		InputPrice:  modelConfig.Pricing.Input,
		OutputPrice: modelConfig.Pricing.Output,
		History:     []Message{},
	}
}
