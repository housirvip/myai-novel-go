package providers

import (
	"context"
	"fmt"
	"net/http"

	"myai-novel-go/internal/config"
	"myai-novel-go/internal/llm"
)

// CustomClient 复用 OpenAI 兼容 chat/completions 协议,只是 baseURL/apiKey/model 自定义。
type CustomClient struct {
	inner *OpenAIClient
}

func NewCustom(cfg *config.Config, httpC *http.Client) *CustomClient {
	inner := NewCompatible(cfg.CustomLLMBaseURL, cfg.CustomLLMAPIKey, cfg.CustomLLMModel, httpC)
	return &CustomClient{inner: inner}
}

func (c *CustomClient) Generate(ctx context.Context, params llm.GenerateParams) (*llm.GenerateResult, error) {
	if c.inner.baseURL == "" {
		return nil, fmt.Errorf("CUSTOM_LLM_BASE_URL is required")
	}
	res, err := c.inner.Generate(ctx, params)
	if err != nil {
		return nil, err
	}
	res.Provider = llm.ProviderCustom
	return res, nil
}
