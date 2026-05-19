package planning

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"myai-novel-go/internal/db/models"
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

func TestJoinNonEmpty(t *testing.T) {
	got := joinNonEmpty([]string{" 姓名:林夜 ", "别名:", "", "状态: active", "\t关键词:剑修\t"})
	require.Equal(t, "姓名:林夜\n状态: active\n关键词:剑修", got)
}

func TestBuildEntityTextHelpers(t *testing.T) {
	targetChapterNo := 7
	status := "active"
	notes := "补充说明"
	keywords := "林夜,剑修"
	alias := "小夜"
	personality := "冷静"
	category := "门派"
	description := "势力描述"
	ownerType := "none"
	rarity := "rare"
	hookType := "mystery"
	hookDesc := "黑铁令引出变故"
	importance := "high"
	categoryWS := "地理"
	content := "山门云雾缭绕"
	relationDesc := "师徒关系"
	relationStatus := "active"

	cases := []struct {
		name string
		got  string
		want string
	}{
		{
			name: "character",
			got: buildCharacterText(models.Character{
				Name: "林夜", Alias: &alias, Personality: &personality, Status: status,
				AppendNotes: &notes, Keywords: &keywords,
			}),
			want: "姓名:林夜\n别名:小夜\n性格:冷静\n状态:active\n备注:补充说明\n关键词:林夜,剑修",
		},
		{
			name: "faction",
			got: buildFactionText(models.Faction{
				Name: "青岳宗", Category: &category, Description: &description, Status: &status,
				AppendNotes: &notes, Keywords: &keywords,
			}),
			want: "势力:青岳宗\n类别:门派\n描述:势力描述\n状态:active\n备注:补充说明\n关键词:林夜,剑修",
		},
		{
			name: "item",
			got: buildItemText(models.Item{
				Name: "黑铁令", Category: &category, Description: &description, OwnerType: ownerType,
				Rarity: &rarity, Status: &status, AppendNotes: &notes, Keywords: &keywords,
			}),
			want: "物品:黑铁令\n类别:门派\n描述:势力描述\n归属:none\n稀有度:rare\n状态:active\n备注:补充说明\n关键词:林夜,剑修",
		},
		{
			name: "hook with target",
			got: buildHookText(models.StoryHook{
				Title: "黑铁令异常", HookType: &hookType, Description: &hookDesc, Importance: &importance,
				Status: "open", TargetChapterNo: &targetChapterNo, AppendNotes: &notes, Keywords: &keywords,
			}),
			want: "钩子:黑铁令异常\n类型:mystery\n描述:黑铁令引出变故\n重要度:high\n状态:open\ntarget_chapter_no=7\n备注:补充说明\n关键词:林夜,剑修",
		},
		{
			name: "world setting",
			got: buildWorldSettingText(models.WorldSetting{
				Title: "山门", Category: categoryWS, Content: content, AppendNotes: &notes, Keywords: &keywords,
			}),
			want: "设定:山门\n类别:地理\n内容:山门云雾缭绕\n备注:补充说明\n关键词:林夜,剑修",
		},
		{
			name: "relation",
			got: buildRelationText(models.Relation{
				RelationType: "师徒", SourceType: "character", SourceID: 1, TargetType: "character", TargetID: 2,
				Description: &relationDesc, Status: &relationStatus, AppendNotes: &notes, Keywords: &keywords,
			}),
			want: "关系类型:师徒\n源:character#1\n目标:character#2\n描述:师徒关系\n状态:active\n备注:补充说明\n关键词:林夜,剑修",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.got)
		})
	}
}

// fakeEmbedding 把 token list 用第一个字符的 byte 偏移映射到 16 维向量,
// 跑测试时不依赖任何 provider,但产生稳定的相似度排序。
type fakeEmbedding struct{ dim int }

func (f *fakeEmbedding) Dimensions() int                     { return f.dim }
func (f *fakeEmbedding) Model() string                       { return "fake" }
func (f *fakeEmbedding) ProviderName() llm.EmbeddingProvider { return llm.EmbeddingProviderHash }
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
