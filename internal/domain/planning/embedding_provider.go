package planning

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

// EmbeddingCandidateProvider 在规则候选基础上叠加嵌入命中。
// 不替换 base provider 的结果,而是把嵌入命中的实体 ID 用 reason="embedding_match"
// (或 embedding_support)合并进现有候选,分数加成由 reranker 阶段统一处理。
type EmbeddingCandidateProvider struct {
	base     CandidateProvider
	searcher EmbeddingSearcher
	model    string
	limit    int
	minScore float64
}

func NewEmbeddingCandidateProvider(base CandidateProvider, searcher EmbeddingSearcher, model string, limit int, minScore float64) *EmbeddingCandidateProvider {
	return &EmbeddingCandidateProvider{base: base, searcher: searcher, model: model, limit: limit, minScore: minScore}
}

func (p *EmbeddingCandidateProvider) LoadCandidates(ctx context.Context, db *gorm.DB, params RetrieveParams) (*CandidateBundle, error) {
	bundle, err := p.base.LoadCandidates(ctx, db, params)
	if err != nil {
		return nil, err
	}
	queryText := strings.TrimSpace(params.QueryText)
	if queryText == "" && len(params.Keywords) > 0 {
		queryText = strings.Join(params.Keywords, " ")
	}
	if queryText == "" {
		return bundle, nil
	}

	limit := p.limit
	if limit <= 0 {
		limit = 32
	}
	matches, err := p.searcher.Search(ctx, params.BookID, queryText, limit)
	if err != nil {
		// 嵌入层失败不阻断主链路,降级回规则候选。
		return bundle, nil
	}
	for _, m := range matches {
		if m.Score < p.minScore {
			continue
		}
		mergeEmbeddingMatch(bundle, m)
	}
	return bundle, nil
}

// mergeEmbeddingMatch 把一次嵌入命中合并到候选里:
//   - 如果命中实体已经在 bundle 中,追加 reason embedding_support 并把分数加上 6
//   - 否则按 entityType 注入新候选,reason embedding_match,分数为 cosine 转换后的整型(score*100)。
func mergeEmbeddingMatch(b *CandidateBundle, m EmbeddingMatch) {
	groups := []*[]RetrievedEntity{
		&b.EntityGroups.Hooks, &b.EntityGroups.Characters, &b.EntityGroups.Factions,
		&b.EntityGroups.Items, &b.EntityGroups.Relations, &b.EntityGroups.WorldSettings,
	}
	for _, g := range groups {
		for i := range *g {
			if (*g)[i].ID == m.EntityID && entityGroupMatches(g, m.EntityType, b) {
				if !strings.Contains((*g)[i].Reason, "embedding_support") {
					(*g)[i].Reason = appendReason((*g)[i].Reason, "embedding_support")
				}
				(*g)[i].Score += 6
				return
			}
		}
	}
	added := RetrievedEntity{
		ID:      m.EntityID,
		Reason:  "embedding_match",
		Content: m.Text,
		Score:   m.Score * 100, // cosine 0..1 → 0..100,与规则打分量级对齐
	}
	switch m.EntityType {
	case "hook":
		added.Title = m.DisplayName
		b.EntityGroups.Hooks = append(b.EntityGroups.Hooks, added)
	case "character":
		added.Name = m.DisplayName
		b.EntityGroups.Characters = append(b.EntityGroups.Characters, added)
	case "faction":
		added.Name = m.DisplayName
		b.EntityGroups.Factions = append(b.EntityGroups.Factions, added)
	case "item":
		added.Name = m.DisplayName
		b.EntityGroups.Items = append(b.EntityGroups.Items, added)
	case "relation":
		b.EntityGroups.Relations = append(b.EntityGroups.Relations, added)
	case "world_setting":
		added.Title = m.DisplayName
		b.EntityGroups.WorldSettings = append(b.EntityGroups.WorldSettings, added)
	}
}

// entityGroupMatches 判断当前 group 切片是否对应嵌入命中的 entityType。
// 因为我们按指针扫所有组,需要对得上才更新分数(避免 ID 撞车)。
func entityGroupMatches(g *[]RetrievedEntity, et string, b *CandidateBundle) bool {
	switch et {
	case "hook":
		return g == &b.EntityGroups.Hooks
	case "character":
		return g == &b.EntityGroups.Characters
	case "faction":
		return g == &b.EntityGroups.Factions
	case "item":
		return g == &b.EntityGroups.Items
	case "relation":
		return g == &b.EntityGroups.Relations
	case "world_setting":
		return g == &b.EntityGroups.WorldSettings
	}
	return false
}

func appendReason(existing, add string) string {
	existing = strings.TrimSpace(existing)
	if existing == "" {
		return add
	}
	return existing + "|" + add
}
