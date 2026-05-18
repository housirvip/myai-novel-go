package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"myai-novel-go/internal/llm"
)

const maxCustomEmbeddingInputChars = 8192

type CustomEmbedding struct {
	baseURL   string
	apiKey    string
	model     string
	batchSize int
	dim       int
	http      *http.Client
}

type CustomEmbeddingOptions struct {
	BaseURL   string
	APIKey    string
	Model     string
	BatchSize int
}

func NewCustomEmbedding(opt CustomEmbeddingOptions) *CustomEmbedding {
	bs := opt.BatchSize
	if bs <= 0 {
		bs = 10
	}
	return &CustomEmbedding{
		baseURL:   strings.TrimRight(opt.BaseURL, "/"),
		apiKey:    opt.APIKey,
		model:     opt.Model,
		batchSize: bs,
		http:      &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *CustomEmbedding) Model() string                       { return c.model }
func (c *CustomEmbedding) ProviderName() llm.EmbeddingProvider { return llm.EmbeddingProviderCustom }

// Dimensions 返回观测到的向量维度,首次 Embed 后才有意义。
func (c *CustomEmbedding) Dimensions() int { return c.dim }

type customEmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type customEmbedResponse struct {
	Data []struct {
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

func (c *CustomEmbedding) Embed(ctx context.Context, input []string) ([][]float32, error) {
	if len(input) == 0 {
		return nil, nil
	}
	clean := make([]string, len(input))
	for i, s := range input {
		clean[i] = sanitizeEmbeddingInput(s)
	}
	url := c.baseURL + "/embeddings"
	all := make([][]float32, 0, len(clean))
	for start := 0; start < len(clean); start += c.batchSize {
		end := start + c.batchSize
		if end > len(clean) {
			end = len(clean)
		}
		chunk := clean[start:end]
		got, err := c.callOnce(ctx, url, chunk)
		if err != nil {
			return nil, err
		}
		if len(got) != len(chunk) {
			return nil, fmt.Errorf("embedding count mismatch: expected %d got %d", len(chunk), len(got))
		}
		all = append(all, got...)
	}
	if len(all) > 0 && c.dim == 0 {
		c.dim = len(all[0])
	}
	return all, nil
}

func (c *CustomEmbedding) callOnce(ctx context.Context, url string, chunk []string) ([][]float32, error) {
	body, err := json.Marshal(customEmbedRequest{Model: c.model, Input: chunk})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("custom embedding request failed: %d %s", resp.StatusCode, string(raw))
	}
	var parsed customEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	if len(parsed.Data) == 0 && len(chunk) > 0 {
		return nil, fmt.Errorf("custom embedding response did not include embedding data")
	}
	sort.Slice(parsed.Data, func(i, j int) bool { return parsed.Data[i].Index < parsed.Data[j].Index })
	out := make([][]float32, len(parsed.Data))
	for i, item := range parsed.Data {
		out[i] = item.Embedding
	}
	return out, nil
}

func sanitizeEmbeddingInput(text string) string {
	t := strings.TrimSpace(text)
	if len(t) > maxCustomEmbeddingInputChars {
		t = t[:maxCustomEmbeddingInputChars]
	}
	if t == "" {
		return " "
	}
	return t
}
