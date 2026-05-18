package planning

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"

	"myai-novel-go/internal/config"
	"myai-novel-go/internal/db/models"
	"myai-novel-go/internal/domain/shared"
)

// RuleCandidateProvider 把原 retrieval_service.go 中的规则召回 + 启发式打分搬过来,
// 行为与之前完全一致(关键词命中 + manual 加权 + open-hook 自动纳入)。
type RuleCandidateProvider struct {
	cfg *config.Config
}

func NewRuleCandidateProvider(cfg *config.Config) *RuleCandidateProvider {
	return &RuleCandidateProvider{cfg: cfg}
}

func (p *RuleCandidateProvider) LoadCandidates(ctx context.Context, db *gorm.DB, params RetrieveParams) (*CandidateBundle, error) {
	kwLower := lowerSet(params.Keywords)

	outlines, err := p.loadOutlines(ctx, db, params.BookID, params.ChapterNo, kwLower)
	if err != nil {
		return nil, err
	}
	if len(outlines) > p.cfg.PlanningRetrievalOutlineLimit {
		outlines = outlines[:p.cfg.PlanningRetrievalOutlineLimit]
	}

	recents, err := p.loadRecentChapters(ctx, db, params.BookID, params.ChapterNo)
	if err != nil {
		return nil, err
	}

	chars, err := p.loadCharacters(ctx, db, params.BookID, kwLower, params.ManualRefs.CharacterIDs, p.cfg.PlanningRetrievalCharacterLimit)
	if err != nil {
		return nil, err
	}
	factions, err := p.loadFactions(ctx, db, params.BookID, kwLower, params.ManualRefs.FactionIDs, p.cfg.PlanningRetrievalFactionLimit)
	if err != nil {
		return nil, err
	}
	items, err := p.loadItems(ctx, db, params.BookID, kwLower, params.ManualRefs.ItemIDs, p.cfg.PlanningRetrievalItemLimit)
	if err != nil {
		return nil, err
	}
	hooks, err := p.loadHooks(ctx, db, params.BookID, kwLower, params.ManualRefs.HookIDs, p.cfg.PlanningRetrievalHookLimit)
	if err != nil {
		return nil, err
	}
	rels, err := p.loadRelations(ctx, db, params.BookID, kwLower, params.ManualRefs.RelationIDs, p.cfg.PlanningRetrievalRelationLimit)
	if err != nil {
		return nil, err
	}
	wss, err := p.loadWorldSettings(ctx, db, params.BookID, kwLower, params.ManualRefs.WorldSettingIDs, p.cfg.PlanningRetrievalWorldSettingLimit)
	if err != nil {
		return nil, err
	}

	return &CandidateBundle{
		Outlines:       outlines,
		RecentChapters: recents,
		EntityGroups: EntityGroups{
			Hooks:         hooks,
			Characters:    chars,
			Factions:      factions,
			Items:         items,
			Relations:     rels,
			WorldSettings: wss,
		},
	}, nil
}

func (p *RuleCandidateProvider) loadOutlines(ctx context.Context, db *gorm.DB, bookID int64, chapterNo int, kwLower map[string]bool) ([]RetrievedOutline, error) {
	var rows []models.Outline
	if err := db.WithContext(ctx).Where("book_id = ?", bookID).Order("id DESC").Limit(200).Find(&rows).Error; err != nil {
		return nil, err
	}
	type scored struct {
		row    models.Outline
		score  float64
		reason string
	}
	out := make([]scored, 0, len(rows))
	for _, r := range rows {
		score := 0.0
		reason := []string{}
		if r.ChapterStartNo != nil && r.ChapterEndNo != nil {
			if chapterNo >= *r.ChapterStartNo && chapterNo <= *r.ChapterEndNo {
				score += 5
				reason = append(reason, "章节区间命中")
			}
		}
		text := strings.ToLower(r.Title + " " + shared.DerefStr(r.StoryCore) + " " + shared.DerefStr(r.MainPlot))
		hit := keywordHits(text, kwLower)
		if hit > 0 {
			score += float64(hit)
			reason = append(reason, "关键词命中")
		}
		if score > 0 {
			out = append(out, scored{row: r, score: score, reason: strings.Join(reason, "|")})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].score > out[j].score })
	result := make([]RetrievedOutline, 0, len(out))
	for _, it := range out {
		content := strings.TrimSpace(strings.Join([]string{
			"标题:" + it.row.Title,
			"主线:" + shared.DerefStr(it.row.MainPlot),
			"故事核:" + shared.DerefStr(it.row.StoryCore),
		}, "\n"))
		result = append(result, RetrievedOutline{ID: it.row.ID, Title: it.row.Title, Reason: it.reason, Content: content})
	}
	return result, nil
}

func (p *RuleCandidateProvider) loadRecentChapters(ctx context.Context, db *gorm.DB, bookID int64, chapterNo int) ([]RetrievedChapterSummary, error) {
	var rows []models.Chapter
	if err := db.WithContext(ctx).Where("book_id = ? AND chapter_no < ?", bookID, chapterNo).
		Order("chapter_no DESC").Limit(p.cfg.PlanningRetrievalRecentChapterLimit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]RetrievedChapterSummary, 0, len(rows))
	for _, r := range rows {
		out = append(out, RetrievedChapterSummary{ID: r.ID, ChapterNo: r.ChapterNo, Title: r.Title, Summary: r.Summary, Status: r.Status})
	}
	return out, nil
}

func (p *RuleCandidateProvider) loadCharacters(ctx context.Context, db *gorm.DB, bookID int64, kwLower map[string]bool, manualIDs []int64, limit int) ([]RetrievedEntity, error) {
	var rows []models.Character
	if err := db.WithContext(ctx).Where("book_id = ?", bookID).Limit(p.cfg.PlanningRetrievalEntityScanLimit).Find(&rows).Error; err != nil {
		return nil, err
	}
	manualSet := int64Set(manualIDs)
	scored := make([]RetrievedEntity, 0)
	for _, r := range rows {
		text := strings.ToLower(r.Name + " " + shared.DerefStr(r.Alias) + " " + shared.DerefStr(r.Background) + " " + shared.DerefStr(r.Keywords))
		score := float64(keywordHits(text, kwLower))
		reason := []string{}
		if score > 0 {
			reason = append(reason, "关键词命中")
		}
		if manualSet[r.ID] {
			score += 100
			reason = append(reason, "手动指定")
		}
		if score == 0 {
			continue
		}
		content := strings.TrimSpace(strings.Join([]string{
			"姓名:" + r.Name,
			"性格:" + shared.DerefStr(r.Personality),
			"背景:" + shared.DerefStr(r.Background),
			"目标:" + shared.DerefStr(r.Goal),
		}, "\n"))
		scored = append(scored, RetrievedEntity{ID: r.ID, Name: r.Name, Reason: strings.Join(reason, "|"), Content: content, Score: score})
	}
	return topNEntities(scored, limit), nil
}

func (p *RuleCandidateProvider) loadFactions(ctx context.Context, db *gorm.DB, bookID int64, kwLower map[string]bool, manualIDs []int64, limit int) ([]RetrievedEntity, error) {
	var rows []models.Faction
	if err := db.WithContext(ctx).Where("book_id = ?", bookID).Limit(p.cfg.PlanningRetrievalEntityScanLimit).Find(&rows).Error; err != nil {
		return nil, err
	}
	manualSet := int64Set(manualIDs)
	scored := make([]RetrievedEntity, 0)
	for _, r := range rows {
		text := strings.ToLower(r.Name + " " + shared.DerefStr(r.CoreGoal) + " " + shared.DerefStr(r.Description) + " " + shared.DerefStr(r.Keywords))
		score := float64(keywordHits(text, kwLower))
		reason := []string{}
		if score > 0 {
			reason = append(reason, "关键词命中")
		}
		if manualSet[r.ID] {
			score += 100
			reason = append(reason, "手动指定")
		}
		if score == 0 {
			continue
		}
		content := strings.TrimSpace(strings.Join([]string{
			"势力:" + r.Name,
			"核心目标:" + shared.DerefStr(r.CoreGoal),
			"描述:" + shared.DerefStr(r.Description),
		}, "\n"))
		scored = append(scored, RetrievedEntity{ID: r.ID, Name: r.Name, Reason: strings.Join(reason, "|"), Content: content, Score: score})
	}
	return topNEntities(scored, limit), nil
}

func (p *RuleCandidateProvider) loadItems(ctx context.Context, db *gorm.DB, bookID int64, kwLower map[string]bool, manualIDs []int64, limit int) ([]RetrievedEntity, error) {
	var rows []models.Item
	if err := db.WithContext(ctx).Where("book_id = ?", bookID).Limit(p.cfg.PlanningRetrievalEntityScanLimit).Find(&rows).Error; err != nil {
		return nil, err
	}
	manualSet := int64Set(manualIDs)
	scored := make([]RetrievedEntity, 0)
	for _, r := range rows {
		text := strings.ToLower(r.Name + " " + shared.DerefStr(r.Description) + " " + shared.DerefStr(r.Keywords))
		score := float64(keywordHits(text, kwLower))
		reason := []string{}
		if score > 0 {
			reason = append(reason, "关键词命中")
		}
		if manualSet[r.ID] {
			score += 100
			reason = append(reason, "手动指定")
		}
		if score == 0 {
			continue
		}
		content := strings.TrimSpace(strings.Join([]string{
			"物品:" + r.Name,
			"类别:" + shared.DerefStr(r.Category),
			"描述:" + shared.DerefStr(r.Description),
			"归属:" + r.OwnerType,
		}, "\n"))
		scored = append(scored, RetrievedEntity{ID: r.ID, Name: r.Name, Reason: strings.Join(reason, "|"), Content: content, Score: score})
	}
	return topNEntities(scored, limit), nil
}

func (p *RuleCandidateProvider) loadHooks(ctx context.Context, db *gorm.DB, bookID int64, kwLower map[string]bool, manualIDs []int64, limit int) ([]RetrievedEntity, error) {
	var rows []models.StoryHook
	if err := db.WithContext(ctx).Where("book_id = ?", bookID).Limit(p.cfg.PlanningRetrievalEntityScanLimit).Find(&rows).Error; err != nil {
		return nil, err
	}
	manualSet := int64Set(manualIDs)
	scored := make([]RetrievedEntity, 0)
	for _, r := range rows {
		text := strings.ToLower(r.Title + " " + shared.DerefStr(r.Description) + " " + shared.DerefStr(r.Keywords))
		score := float64(keywordHits(text, kwLower))
		reason := []string{}
		if score > 0 {
			reason = append(reason, "关键词命中")
		}
		if manualSet[r.ID] {
			score += 100
			reason = append(reason, "手动指定")
		}
		if r.Status == "open" && score == 0 {
			score = 0.5
			reason = append(reason, "未回收钩子")
		}
		if score == 0 {
			continue
		}
		hookType := shared.DerefStr(r.HookType)
		targetCh := ""
		if r.TargetChapterNo != nil {
			targetCh = fmt.Sprintf("target_chapter_no=%d", *r.TargetChapterNo)
		}
		content := strings.TrimSpace(strings.Join([]string{
			"钩子:" + r.Title,
			"类型:" + hookType,
			"描述:" + shared.DerefStr(r.Description),
			"状态:" + r.Status,
			targetCh,
		}, "\n"))
		scored = append(scored, RetrievedEntity{ID: r.ID, Title: r.Title, Reason: strings.Join(reason, "|"), Content: content, Score: score})
	}
	return topNEntities(scored, limit), nil
}

func (p *RuleCandidateProvider) loadRelations(ctx context.Context, db *gorm.DB, bookID int64, kwLower map[string]bool, manualIDs []int64, limit int) ([]RetrievedEntity, error) {
	var rows []models.Relation
	if err := db.WithContext(ctx).Where("book_id = ?", bookID).Limit(p.cfg.PlanningRetrievalEntityScanLimit).Find(&rows).Error; err != nil {
		return nil, err
	}
	manualSet := int64Set(manualIDs)
	scored := make([]RetrievedEntity, 0)
	for _, r := range rows {
		text := strings.ToLower(r.RelationType + " " + shared.DerefStr(r.Description) + " " + shared.DerefStr(r.Keywords))
		score := float64(keywordHits(text, kwLower))
		reason := []string{}
		if score > 0 {
			reason = append(reason, "关键词命中")
		}
		if manualSet[r.ID] {
			score += 100
			reason = append(reason, "手动指定")
		}
		if score == 0 {
			continue
		}
		content := strings.TrimSpace(strings.Join([]string{
			fmt.Sprintf("关系类型:%s", r.RelationType),
			fmt.Sprintf("源:%s#%d", r.SourceType, r.SourceID),
			fmt.Sprintf("目标:%s#%d", r.TargetType, r.TargetID),
			"描述:" + shared.DerefStr(r.Description),
		}, "\n"))
		scored = append(scored, RetrievedEntity{ID: r.ID, Reason: strings.Join(reason, "|"), Content: content, Score: score})
	}
	return topNEntities(scored, limit), nil
}

func (p *RuleCandidateProvider) loadWorldSettings(ctx context.Context, db *gorm.DB, bookID int64, kwLower map[string]bool, manualIDs []int64, limit int) ([]RetrievedEntity, error) {
	var rows []models.WorldSetting
	if err := db.WithContext(ctx).Where("book_id = ?", bookID).Limit(p.cfg.PlanningRetrievalEntityScanLimit).Find(&rows).Error; err != nil {
		return nil, err
	}
	manualSet := int64Set(manualIDs)
	scored := make([]RetrievedEntity, 0)
	for _, r := range rows {
		text := strings.ToLower(r.Title + " " + r.Category + " " + r.Content + " " + shared.DerefStr(r.Keywords))
		score := float64(keywordHits(text, kwLower))
		reason := []string{}
		if score > 0 {
			reason = append(reason, "关键词命中")
		}
		if manualSet[r.ID] {
			score += 100
			reason = append(reason, "手动指定")
		}
		if score == 0 {
			continue
		}
		content := strings.TrimSpace(strings.Join([]string{
			"设定:" + r.Title,
			"类别:" + r.Category,
			"内容:" + r.Content,
		}, "\n"))
		scored = append(scored, RetrievedEntity{ID: r.ID, Title: r.Title, Reason: strings.Join(reason, "|"), Content: content, Score: score})
	}
	return topNEntities(scored, limit), nil
}

func topNEntities(in []RetrievedEntity, n int) []RetrievedEntity {
	sort.Slice(in, func(i, j int) bool { return in[i].Score > in[j].Score })
	if len(in) > n {
		in = in[:n]
	}
	return in
}

func keywordHits(text string, kwLower map[string]bool) int {
	if len(kwLower) == 0 || text == "" {
		return 0
	}
	hits := 0
	for kw := range kwLower {
		if kw == "" {
			continue
		}
		if strings.Contains(text, kw) {
			hits++
		}
	}
	return hits
}

func lowerSet(in []string) map[string]bool {
	out := make(map[string]bool, len(in))
	for _, v := range in {
		v = strings.TrimSpace(strings.ToLower(v))
		if v != "" {
			out[v] = true
		}
	}
	return out
}

func int64Set(in []int64) map[int64]bool {
	out := make(map[int64]bool, len(in))
	for _, v := range in {
		out[v] = true
	}
	return out
}
