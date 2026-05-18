package planning

import (
	"context"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// HeuristicReranker:多因子启发式打分,对齐原 retrieval-reranker-heuristic.ts。
// baseScore + manual=40 + keyword=15 + embedding(match=10/support=6)+ continuity + hook 章节距离 bonus
type HeuristicReranker struct{}

func NewHeuristicReranker() *HeuristicReranker { return &HeuristicReranker{} }

func (r *HeuristicReranker) Rerank(_ context.Context, params RetrieveParams, in *CandidateBundle) (*CandidateBundle, error) {
	out := &CandidateBundle{
		Outlines:       in.Outlines,
		RecentChapters: in.RecentChapters,
		EntityGroups: EntityGroups{
			Hooks:         rerankGroup(in.EntityGroups.Hooks, params, "hook"),
			Characters:    rerankGroup(in.EntityGroups.Characters, params, "character"),
			Factions:      rerankGroup(in.EntityGroups.Factions, params, "faction"),
			Items:         rerankGroup(in.EntityGroups.Items, params, "item"),
			Relations:     rerankGroup(in.EntityGroups.Relations, params, "relation"),
			WorldSettings: rerankGroup(in.EntityGroups.WorldSettings, params, "world_setting"),
		},
	}
	return out, nil
}

func rerankGroup(entities []RetrievedEntity, params RetrieveParams, entityType string) []RetrievedEntity {
	out := append([]RetrievedEntity(nil), entities...)
	scores := make(map[int64]float64, len(out))
	for _, e := range out {
		scores[e.ID] = computeHeuristicScore(e, params, entityType)
	}
	sort.SliceStable(out, func(i, j int) bool {
		si, sj := scores[out[i].ID], scores[out[j].ID]
		if si != sj {
			return si > sj
		}
		return out[i].ID < out[j].ID
	})
	for i := range out {
		out[i].Score = scores[out[i].ID]
	}
	return out
}

func computeHeuristicScore(e RetrievedEntity, params RetrieveParams, entityType string) float64 {
	content := strings.ToLower(e.Content)
	base := e.Score
	manual := 0.0
	if hasReason(e.Reason, "手动指定") || hasReason(e.Reason, "manual_id") {
		manual = 40
	}
	keyword := 0.0
	if hasReason(e.Reason, "关键词命中") || hasReason(e.Reason, "keyword_hit") {
		keyword = 15
	}
	embedding := 0.0
	if hasReason(e.Reason, "embedding_match") {
		embedding += 10
	}
	if hasReason(e.Reason, "embedding_support") {
		embedding += 6
	}
	continuity := float64(countKeywordHits(params.Keywords, content))*4 + continuityBonus(content, entityType)
	hookBonus := 0.0
	if entityType == "hook" {
		hookBonus = hookChapterBonus(content, params.ChapterNo)
	}
	return base + manual + keyword + embedding + continuity + hookBonus
}

func hasReason(reason, want string) bool {
	for _, r := range strings.Split(reason, "|") {
		if strings.TrimSpace(r) == want {
			return true
		}
	}
	return false
}

func countKeywordHits(keywords []string, contentLower string) int {
	if contentLower == "" || len(keywords) == 0 {
		return 0
	}
	hits := 0
	for _, kw := range keywords {
		kw = strings.ToLower(strings.TrimSpace(kw))
		if kw != "" && strings.Contains(contentLower, kw) {
			hits++
		}
	}
	return hits
}

// continuityBonus:hook/world_setting 类对"未回收/活跃"等连续性词加微小分数。
func continuityBonus(contentLower, entityType string) float64 {
	if entityType == "hook" {
		if strings.Contains(contentLower, "状态:open") || strings.Contains(contentLower, "未回收") {
			return 6
		}
	}
	if entityType == "world_setting" {
		if strings.Contains(contentLower, "状态:active") || strings.Contains(contentLower, "活跃") {
			return 2
		}
	}
	return 0
}

var hookTargetChapterRE = regexp.MustCompile(`target_chapter_no=(\d+)`)

func hookChapterBonus(content string, chapterNo int) float64 {
	m := hookTargetChapterRE.FindStringSubmatch(content)
	if len(m) < 2 {
		return 0
	}
	target, err := strconv.Atoi(m[1])
	if err != nil {
		return 0
	}
	d := target - chapterNo
	if d < 0 {
		d = -d
	}
	switch d {
	case 0:
		return 30
	case 1:
		return 20
	case 2:
		return 10
	}
	return 0
}
