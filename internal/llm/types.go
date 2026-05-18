package llm

import "context"

type ProviderName string

const (
	ProviderMock      ProviderName = "mock"
	ProviderOpenAI    ProviderName = "openai"
	ProviderAnthropic ProviderName = "anthropic"
	ProviderCustom    ProviderName = "custom"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

type ResponseFormat string

const (
	ResponseFormatText ResponseFormat = "text"
	ResponseFormatJSON ResponseFormat = "json"
)

type GenerateParams struct {
	Model          string
	Messages       []Message
	Temperature    *float64
	MaxTokens      *int
	ResponseFormat ResponseFormat
}

type Usage struct {
	InputTokens  int `json:"inputTokens,omitempty"`
	OutputTokens int `json:"outputTokens,omitempty"`
	TotalTokens  int `json:"totalTokens,omitempty"`
}

type GenerateResult struct {
	Provider ProviderName `json:"provider"`
	Model    string       `json:"model"`
	Content  string       `json:"content"`
	Usage    Usage        `json:"usage,omitempty"`
}

type Client interface {
	Generate(ctx context.Context, params GenerateParams) (*GenerateResult, error)
}

type EmbeddingProvider string

const (
	EmbeddingProviderNone   EmbeddingProvider = "none"
	EmbeddingProviderHash   EmbeddingProvider = "hash"
	EmbeddingProviderCustom EmbeddingProvider = "custom"
)

type EmbeddingClient interface {
	Embed(ctx context.Context, input []string) ([][]float32, error)
	Dimensions() int
	Model() string
	ProviderName() EmbeddingProvider
}
