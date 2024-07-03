package llm

// Role represents the role of a message in a conversation.
type Role string

const (
	SystemRole    Role = "system"
	UserRole      Role = "user"
	AssistantRole Role = "assistant"
)

// ChatResponse represents the response from a chat completion API.
type ChatResponse struct {
	Usage   Usage    `json:"usage"`
	Choices []Choice `json:"choices"`
}

// Choice represents a single choice in a chat completion response.
type Choice struct {
	Message Message `json:"message"`
}

// Usage represents the token usage information for a chat completion.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatRequest represents a request to a chat completion API.
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
}

// Message represents a single message in a conversation.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}
