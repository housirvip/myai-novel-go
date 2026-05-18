package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"myai-novel-go/internal/config"
	"myai-novel-go/internal/llm"
)

type OpenAIClient struct {
	cfg     *config.Config
	baseURL string
	apiKey  string
	model   string
	httpC   *http.Client
}

func NewOpenAI(cfg *config.Config, httpC *http.Client) *OpenAIClient {
	base := cfg.OpenAIBaseURL
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	return &OpenAIClient{cfg: cfg, baseURL: base, apiKey: cfg.OpenAIAPIKey, model: cfg.OpenAIModel, httpC: httpC}
}

// NewCompatible 用于 custom provider:复用 openai chat/completions 协议但走自定义 baseURL/apiKey/model。
func NewCompatible(baseURL, apiKey, model string, httpC *http.Client) *OpenAIClient {
	return &OpenAIClient{baseURL: baseURL, apiKey: apiKey, model: model, httpC: httpC}
}

type openAIChatReq struct {
	Model               string         `json:"model"`
	Messages            []openAIMsg    `json:"messages"`
	Temperature         *float64       `json:"temperature,omitempty"`
	MaxCompletionTokens *int           `json:"max_completion_tokens,omitempty"`
	ResponseFormat      *openAIRespFmt `json:"response_format,omitempty"`
}

type openAIMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIRespFmt struct {
	Type string `json:"type"`
}

type openAIChatResp struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func (c *OpenAIClient) Generate(ctx context.Context, params llm.GenerateParams) (*llm.GenerateResult, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("api key is required")
	}
	model := params.Model
	if model == "" {
		model = c.model
	}

	msgs := make([]openAIMsg, 0, len(params.Messages)+1)
	for _, m := range params.Messages {
		msgs = append(msgs, openAIMsg{Role: string(m.Role), Content: m.Content})
	}
	if params.ResponseFormat == llm.ResponseFormatJSON {
		msgs = append(msgs, openAIMsg{Role: "system", Content: "Return valid JSON only. Do not include markdown fences or extra explanation."})
	}

	body := openAIChatReq{Model: model, Messages: msgs, Temperature: params.Temperature, MaxCompletionTokens: params.MaxTokens}
	if params.ResponseFormat == llm.ResponseFormatJSON {
		body.ResponseFormat = &openAIRespFmt{Type: "json_object"}
	}

	bs, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(bs))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpC.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai request: %w", err)
	}
	defer resp.Body.Close()

	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai request failed: %d %s", resp.StatusCode, string(rb))
	}

	var parsed openAIChatResp
	if err := json.Unmarshal(rb, &parsed); err != nil {
		return nil, fmt.Errorf("decode openai response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("openai returned no choices")
	}

	return &llm.GenerateResult{
		Provider: llm.ProviderOpenAI,
		Model:    firstNonEmpty(parsed.Model, model),
		Content:  parsed.Choices[0].Message.Content,
		Usage: llm.Usage{
			InputTokens:  parsed.Usage.PromptTokens,
			OutputTokens: parsed.Usage.CompletionTokens,
			TotalTokens:  parsed.Usage.TotalTokens,
		},
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
