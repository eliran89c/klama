package llm

import (
	"net/http"
	"net/url"

	"github.com/eliran89c/klama/config"
)

// Model represents a language model and its associated data.
type Model struct {
	Client      *http.Client
	Name        string
	URL         string
	AuthToken   AuthToken
	InputPrice  float64 // price per 1K input tokens
	OutputPrice float64 // price per 1K output tokens
	History     []Message
	Usage       Usage
}

// AuthToken represents the authentication token for the model.
type AuthToken struct {
	Key   string
	Value string
}

// NewModel creates a new Model instance.
func NewModel(client *http.Client, modelConfig config.ModelConfig) *Model {
	auth := AuthToken{
		Key:   "Authorization",
		Value: "Bearer " + modelConfig.AuthToken,
	}

	// build the baseURL
	modelURL := modelConfig.BaseURL + "/chat/completions"

	// add the Azure API version as query parameter if set
	if modelConfig.AzureAPIVersion != "" {
		params := url.Values{}
		params.Add("api-version", modelConfig.AzureAPIVersion)
		modelURL += "?" + params.Encode()

		// update the auth token key for azure models
		auth.Key = "api-key"
		auth.Value = modelConfig.AuthToken
	}

	return &Model{
		Client:      client,
		Name:        modelConfig.Name,
		URL:         modelURL,
		AuthToken:   auth,
		InputPrice:  modelConfig.Pricing.Input,
		OutputPrice: modelConfig.Pricing.Output,
		History:     []Message{},
	}
}
