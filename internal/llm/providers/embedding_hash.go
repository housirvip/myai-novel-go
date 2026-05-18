package providers

import (
	"context"
	"math"
	"strings"
	"unicode"

	"myai-novel-go/internal/llm"
)

const HashEmbeddingModel = "deterministic-hash-32"

type HashEmbedding struct {
	dim int
}

func NewHashEmbedding() *HashEmbedding {
	return &HashEmbedding{dim: 32}
}

func (h *HashEmbedding) Dimensions() int                 { return h.dim }
func (h *HashEmbedding) Model() string                   { return HashEmbeddingModel }
func (h *HashEmbedding) ProviderName() llm.EmbeddingProvider { return llm.EmbeddingProviderHash }

func (h *HashEmbedding) Embed(_ context.Context, input []string) ([][]float32, error) {
	out := make([][]float32, len(input))
	for i, text := range input {
		out[i] = h.embedOne(text)
	}
	return out, nil
}

func (h *HashEmbedding) embedOne(text string) []float32 {
	v := make([]float32, h.dim)
	for _, tok := range tokenizeForHash(text) {
		v[hashToken(tok)%uint32(h.dim)] += 1
	}
	return l2Normalize(v)
}

func tokenizeForHash(text string) []string {
	text = strings.ToLower(text)
	parts := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_'
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func hashToken(token string) uint32 {
	var h uint32
	for i := 0; i < len(token); i++ {
		h = h*31 + uint32(token[i])
	}
	return h
}

func l2Normalize(v []float32) []float32 {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	if sum == 0 {
		return v
	}
	mag := float32(math.Sqrt(sum))
	for i := range v {
		v[i] /= mag
	}
	return v
}
