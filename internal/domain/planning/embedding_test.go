package planning

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"myai-novel-go/internal/llm"
)

func TestCosineSimilarity_OrthogonalAndIdentical(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{0, 1, 0}
	require.InDelta(t, 0.0, cosineSimilarity(a, b), 1e-6)
	require.InDelta(t, 1.0, cosineSimilarity(a, a), 1e-6)
}

func TestLexicalOverlap(t *testing.T) {
	q := []string{"林夜", "宗门", "令牌"}
	doc := []string{"林夜", "进入", "宗门"}
	got := lexicalOverlap(q, doc)
	require.InDelta(t, 2.0/3.0, got, 1e-6)
	require.Equal(t, 0.0, lexicalOverlap(nil, doc))
	require.Equal(t, 0.0, lexicalOverlap(q, nil))
}

func TestEntityTypeBonus_OnlyWhenRuleHeavyAndLexical(t *testing.T) {
	require.Equal(t, 0.0, entityTypeBonus("world_setting", 0.5, false))
	require.Equal(t, 0.0, entityTypeBonus("world_setting", 0, true))
	require.Equal(t, 0.08, entityTypeBonus("world_setting", 0.5, true))
	require.Equal(t, 0.03, entityTypeBonus("faction", 0.5, true))
}

// fakeEmbedding 把 token list 用第一个字符的 byte 偏移映射到 16 维向量,
// 跑测试时不依赖任何 provider,但产生稳定的相似度排序。
type fakeEmbedding struct{ dim int }

func (f *fakeEmbedding) Dimensions() int                      { return f.dim }
func (f *fakeEmbedding) Model() string                        { return "fake" }
func (f *fakeEmbedding) ProviderName() llm.EmbeddingProvider  { return llm.EmbeddingProviderHash }
func (f *fakeEmbedding) Embed(_ context.Context, in []string) ([][]float32, error) {
	out := make([][]float32, len(in))
	for i, s := range in {
		v := make([]float32, f.dim)
		for j, r := range s {
			v[j%f.dim] += float32(r)
		}
		// L2 normalize 让 cosine 有意义
		var sum float64
		for _, x := range v {
			sum += float64(x) * float64(x)
		}
		if sum > 0 {
			mag := float32(math.Sqrt(sum))
			for k := range v {
				v[k] /= mag
			}
		}
		out[i] = v
	}
	return out, nil
}

type memStore struct{ docs []IndexedEmbeddingDocument }

func (m *memStore) Replace(_ context.Context, _ int64, _ string, _ string, docs []IndexedEmbeddingDocument) error {
	m.docs = docs
	return nil
}
func (m *memStore) List(_ context.Context, _ int64, _ string, _ string) ([]IndexedEmbeddingDocument, error) {
	return m.docs, nil
}
func (m *memStore) Clear(_ context.Context, _ int64, _ string, _ string) error {
	m.docs = nil
	return nil
}

func TestBasicEmbeddingSearcher_TopKByScore(t *testing.T) {
	emb := &fakeEmbedding{dim: 16}
	store := &memStore{}
	docs := []EmbeddingDocument{
		{BookID: 1, EntityType: "character", EntityID: 1, ChunkKey: "character:1", Text: "林夜 宗门"},
		{BookID: 1, EntityType: "character", EntityID: 2, ChunkKey: "character:2", Text: "完全无关 的内容"},
	}
	vecs, err := emb.Embed(context.Background(), []string{docs[0].Text, docs[1].Text})
	require.NoError(t, err)
	indexed := make([]IndexedEmbeddingDocument, 2)
	for i, d := range docs {
		indexed[i] = IndexedEmbeddingDocument{EmbeddingDocument: d, Model: "fake", Vector: vecs[i]}
	}
	require.NoError(t, store.Replace(context.Background(), 1, "fake", "character", indexed))

	s := NewBasicEmbeddingSearcher(emb, store)
	matches, err := s.Search(context.Background(), 1, "林夜 宗门", 1)
	require.NoError(t, err)
	require.Len(t, matches, 1)
	require.Equal(t, int64(1), matches[0].EntityID, "highest cosine should win")
}
