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

type AnthropicClient struct {
	cfg     *config.Config
	baseURL string
	httpC   *http.Client
}

func NewAnthropic(cfg *config.Config, httpC *http.Client) *AnthropicClient {
	base := cfg.AnthropicBaseURL
	if base == "" {
		base = "https://api.anthropic.com"
	}
	return &AnthropicClient{cfg: cfg, baseURL: base, httpC: httpC}
}

type anthropicReq struct {
	Model     string         `json:"model"`
	System    string         `json:"system,omitempty"`
	Messages  []anthropicMsg `json:"messages"`
	MaxTokens int            `json:"max_tokens"`
	Temperature *float64     `json:"temperature,omitempty"`
}

type anthropicMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResp struct {
	Model   string `json:"model"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (c *AnthropicClient) Generate(ctx context.Context, params llm.GenerateParams) (*llm.GenerateResult, error) {
	if c.cfg.AnthropicAPIKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY is required")
	}
	model := params.Model
	if model == "" {
		model = c.cfg.AnthropicModel
	}

	var system string
	msgs := make([]anthropicMsg, 0, len(params.Messages))
	for _, m := range params.Messages {
		if m.Role == llm.RoleSystem {
			if system != "" {
				system += "\n\n"
			}
			system += m.Content
			continue
		}
		msgs = append(msgs, anthropicMsg{Role: string(m.Role), Content: m.Content})
	}
	if params.ResponseFormat == llm.ResponseFormatJSON {
		if system != "" {
			system += "\n\n"
		}
		system += "Return valid JSON only. Do not include markdown fences or extra explanation."
	}

	maxTokens := c.cfg.LLMDefaultMaxTokens
	if params.MaxTokens != nil {
		maxTokens = *params.MaxTokens
	}

	body := anthropicReq{Model: model, System: system, Messages: msgs, MaxTokens: maxTokens, Temperature: params.Temperature}
	bs, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/messages", bytes.NewReader(bs))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.cfg.AnthropicAPIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpC.Do(req)
	if err != nil {
		return nil, fmt.Errorf("anthropic request: %w", err)
	}
	defer resp.Body.Close()

	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic request failed: %d %s", resp.StatusCode, string(rb))
	}

	var parsed anthropicResp
	if err := json.Unmarshal(rb, &parsed); err != nil {
		return nil, fmt.Errorf("decode anthropic response: %w", err)
	}

	var content bytes.Buffer
	for _, p := range parsed.Content {
		if p.Type == "text" {
			content.WriteString(p.Text)
		}
	}

	return &llm.GenerateResult{
		Provider: llm.ProviderAnthropic,
		Model:    firstNonEmpty(parsed.Model, model),
		Content:  content.String(),
		Usage: llm.Usage{
			InputTokens:  parsed.Usage.InputTokens,
			OutputTokens: parsed.Usage.OutputTokens,
			TotalTokens:  parsed.Usage.InputTokens + parsed.Usage.OutputTokens,
		},
	}, nil
}
