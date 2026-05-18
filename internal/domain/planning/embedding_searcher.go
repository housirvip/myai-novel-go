package planning

import (
	"context"
	"math"
	"sort"
	"strings"
	"unicode"

	"myai-novel-go/internal/llm"
)

// EmbeddingMatch 是一次嵌入搜索得到的命中,
// score 是已经融合(混合搜索时)/cosine(基础搜索时)后的总分。
type EmbeddingMatch struct {
	EntityType  string
	EntityID    int64
	ChunkKey    string
	DisplayName string
	Text        string
	Score       float64
}

type EmbeddingSearcher interface {
	// Search 在 model 范围内对查询文本做检索,返回 top-K 候选。
	Search(ctx context.Context, bookID int64, queryText string, limit int) ([]EmbeddingMatch, error)
}

// BasicEmbeddingSearcher:纯 cosine 相似度。
type BasicEmbeddingSearcher struct {
	client llm.EmbeddingClient
	store  EmbeddingStore
}

func NewBasicEmbeddingSearcher(client llm.EmbeddingClient, store EmbeddingStore) *BasicEmbeddingSearcher {
	return &BasicEmbeddingSearcher{client: client, store: store}
}

func (s *BasicEmbeddingSearcher) Search(ctx context.Context, bookID int64, queryText string, limit int) ([]EmbeddingMatch, error) {
	queryText = strings.TrimSpace(queryText)
	if queryText == "" {
		return nil, nil
	}
	docs, err := s.store.List(ctx, bookID, s.client.Model(), "")
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, nil
	}
	vecs, err := s.client.Embed(ctx, []string{queryText})
	if err != nil || len(vecs) == 0 {
		return nil, err
	}
	q := vecs[0]
	matches := make([]EmbeddingMatch, 0, len(docs))
	for _, d := range docs {
		score := cosineSimilarity(q, d.Vector)
		matches = append(matches, EmbeddingMatch{
			EntityType:  d.EntityType,
			EntityID:    d.EntityID,
			ChunkKey:    d.ChunkKey,
			DisplayName: d.DisplayName,
			Text:        d.Text,
			Score:       float64(score),
		})
	}
	return topKMatches(matches, limit), nil
}

// HybridEmbeddingSearcher:cosine·w + lexical·w + 实体类型微调,对齐原 embedding-searcher-hybrid.ts。
type HybridEmbeddingSearcher struct {
	client llm.EmbeddingClient
	store  EmbeddingStore
}

func NewHybridEmbeddingSearcher(client llm.EmbeddingClient, store EmbeddingStore) *HybridEmbeddingSearcher {
	return &HybridEmbeddingSearcher{client: client, store: store}
}

func (s *HybridEmbeddingSearcher) Search(ctx context.Context, bookID int64, queryText string, limit int) ([]EmbeddingMatch, error) {
	queryText = strings.TrimSpace(queryText)
	if queryText == "" {
		return nil, nil
	}
	docs, err := s.store.List(ctx, bookID, s.client.Model(), "")
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, nil
	}
	vecs, err := s.client.Embed(ctx, []string{queryText})
	if err != nil || len(vecs) == 0 {
		return nil, err
	}
	q := vecs[0]

	queryTokens := tokenizeForHash(queryText)
	ruleHeavy := isRuleHeavyQuery(queryTokens)
	semWeight, lexWeight := 0.55, 0.45
	if ruleHeavy {
		semWeight, lexWeight = 0.45, 0.55
	}

	matches := make([]EmbeddingMatch, 0, len(docs))
	for _, d := range docs {
		semantic := float64(cosineSimilarity(q, d.Vector))
		lexical := lexicalOverlap(queryTokens, tokenizeForHash(d.Text))
		bonus := entityTypeBonus(d.EntityType, lexical, ruleHeavy)
		combined := semantic*semWeight + lexical*lexWeight + bonus
		matches = append(matches, EmbeddingMatch{
			EntityType:  d.EntityType,
			EntityID:    d.EntityID,
			ChunkKey:    d.ChunkKey,
			DisplayName: d.DisplayName,
			Text:        d.Text,
			Score:       combined,
		})
	}
	return topKMatches(matches, limit), nil
}

func cosineSimilarity(a, b []float32) float32 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, am, bm float64
	for i := range a {
		av := float64(a[i])
		bv := float64(b[i])
		dot += av * bv
		am += av * av
		bm += bv * bv
	}
	if am == 0 || bm == 0 {
		return 0
	}
	return float32(dot / (math.Sqrt(am) * math.Sqrt(bm)))
}

func lexicalOverlap(query, doc []string) float64 {
	if len(query) == 0 || len(doc) == 0 {
		return 0
	}
	docSet := make(map[string]bool, len(doc))
	for _, t := range doc {
		docSet[t] = true
	}
	hits := 0
	for _, t := range query {
		if docSet[t] {
			hits++
		}
	}
	return float64(hits) / float64(len(query))
}

func isRuleHeavyQuery(tokens []string) bool {
	heavy := map[string]bool{"规则": true, "制度": true, "令牌": true, "登记": true}
	for _, t := range tokens {
		if heavy[t] {
			return true
		}
	}
	return false
}

func entityTypeBonus(entityType string, lexical float64, ruleHeavy bool) float64 {
	if !ruleHeavy || lexical <= 0 {
		return 0
	}
	switch entityType {
	case "world_setting":
		return 0.08
	case "faction":
		return 0.03
	}
	return 0
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

func topKMatches(in []EmbeddingMatch, limit int) []EmbeddingMatch {
	sort.Slice(in, func(i, j int) bool {
		if in[i].Score != in[j].Score {
			return in[i].Score > in[j].Score
		}
		return in[i].EntityID < in[j].EntityID
	})
	if limit > 0 && len(in) > limit {
		in = in[:limit]
	}
	return in
}
