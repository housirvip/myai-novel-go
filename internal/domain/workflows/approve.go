package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"myai-novel-go/internal/config"
	"myai-novel-go/internal/db/models"
	"myai-novel-go/internal/domain/planning"
	"myai-novel-go/internal/domain/shared"
	"myai-novel-go/internal/llm"
	"myai-novel-go/internal/llmfactory"
)

type ApproveInput struct {
	BookID    int64  `json:"bookId" binding:"required"`
	ChapterNo int    `json:"chapterNo" binding:"required"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	DryRun    bool   `json:"dryRun"`
}

type ApproveOutput struct {
	ChapterID         int64       `json:"chapterId"`
	FinalID           int64       `json:"finalId,omitempty"`
	WordCount         int         `json:"wordCount"`
	FinalContent      string      `json:"finalContent"`
	Diff              ApproveDiff `json:"diff"`
	NewEntityIDs      newIDsView  `json:"newEntityIds"`
	AppliedUpdates    int         `json:"appliedUpdates"`
	SkippedUpdates    int         `json:"skippedUpdates"`
	SidecarFactCount  int         `json:"sidecarFactCount"`
	SidecarEventCount int         `json:"sidecarEventCount"`
	DryRun            bool        `json:"dryRun"`
}

type newIDsView struct {
	CharacterIDs    []int64 `json:"characterIds"`
	FactionIDs      []int64 `json:"factionIds"`
	ItemIDs         []int64 `json:"itemIds"`
	HookIDs         []int64 `json:"hookIds"`
	WorldSettingIDs []int64 `json:"worldSettingIds"`
	RelationIDs     []int64 `json:"relationIds"`
}

type ApproveDiff struct {
	ChapterSummary     string                  `json:"chapterSummary"`
	UnresolvedImpact   *string                 `json:"unresolvedImpact"`
	ContinuitySnapshot ContinuitySnapshotPayload `json:"continuitySnapshot"`
	ActualCharacterIDs []int64                 `json:"actualCharacterIds"`
	ActualFactionIDs   []int64                 `json:"actualFactionIds"`
	ActualItemIDs      []int64                 `json:"actualItemIds"`
	ActualHookIDs      []int64                 `json:"actualHookIds"`
	ActualWorldSettingIDs []int64              `json:"actualWorldSettingIds"`
	NewCharacters      []NewCharacterPayload   `json:"newCharacters"`
	NewFactions        []NewFactionPayload     `json:"newFactions"`
	NewItems           []NewItemPayload        `json:"newItems"`
	NewHooks           []NewHookPayload        `json:"newHooks"`
	NewWorldSettings   []NewWorldSettingPayload `json:"newWorldSettings"`
	NewRelations       []NewRelationPayload    `json:"newRelations"`
	Updates            []EntityUpdate          `json:"updates"`
}

type ContinuitySnapshotPayload struct {
	ClosingBeat              string   `json:"closingBeat"`
	CarryoverFacts           []string `json:"carryoverFacts"`
	OpenLoops                []string `json:"openLoops"`
	CharacterStateChanges    []string `json:"characterStateChanges"`
	RelationStateChanges     []string `json:"relationStateChanges"`
	ItemStateChanges         []string `json:"itemStateChanges"`
	TabooContinuityMistakes  []string `json:"tabooContinuityMistakes"`
}

type NewCharacterPayload struct {
	Name     string   `json:"name"`
	Summary  string   `json:"summary"`
	Keywords []string `json:"keywords"`
}
type NewFactionPayload struct {
	Name     string   `json:"name"`
	Summary  string   `json:"summary"`
	Keywords []string `json:"keywords"`
}
type NewItemPayload struct {
	Name     string   `json:"name"`
	Summary  string   `json:"summary"`
	Keywords []string `json:"keywords"`
}
type NewHookPayload struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Keywords    []string `json:"keywords"`
}
type NewWorldSettingPayload struct {
	Title    string   `json:"title"`
	Category string   `json:"category"`
	Content  string   `json:"content"`
	Keywords []string `json:"keywords"`
}
type NewRelationPayload struct {
	SourceType   string   `json:"sourceType"`
	SourceID     int64    `json:"sourceId"`
	TargetType   string   `json:"targetType"`
	TargetID     int64    `json:"targetId"`
	RelationType string   `json:"relationType"`
	Intensity    *int     `json:"intensity"`
	Status       *string  `json:"status"`
	Description  string   `json:"description"`
	Keywords     []string `json:"keywords"`
}
type EntityUpdate struct {
	EntityType string         `json:"entityType"`
	EntityID   int64          `json:"entityId"`
	Action     string         `json:"action"`
	Payload    map[string]any `json:"payload"`
}

type ApproveWorkflow struct {
	db   *gorm.DB
	cfg  *config.Config
	llmF *llmfactory.Factory
}

func NewApproveWorkflow(db *gorm.DB, cfg *config.Config, llmF *llmfactory.Factory) *ApproveWorkflow {
	return &ApproveWorkflow{db: db, cfg: cfg, llmF: llmF}
}

func (w *ApproveWorkflow) Run(ctx context.Context, in ApproveInput, notify StageNotifier) (*ApproveOutput, error) {
	llmCli, err := w.llmF.Create(llm.ProviderName(in.Provider))
	if err != nil {
		return nil, err
	}
	Notify(notify, shared.WorkflowStageLoadingChapter, 5)
	chapter, err := LoadChapter(ctx, w.db, in.BookID, in.ChapterNo)
	if err != nil {
		return nil, err
	}
	if chapter.CurrentPlanID == nil || chapter.CurrentDraftID == nil {
		return nil, shared.BadRequest("chapter needs current plan and draft before approve")
	}
	var plan models.ChapterPlan
	var draft models.ChapterDraft
	if err := w.db.WithContext(ctx).First(&plan, *chapter.CurrentPlanID).Error; err != nil {
		return nil, err
	}
	if err := w.db.WithContext(ctx).First(&draft, *chapter.CurrentDraftID).Error; err != nil {
		return nil, err
	}

	retrievedCtx, err := LoadRetrievedContext(&plan)
	if err != nil {
		return nil, err
	}

	// 1. 生成 final 正文
	Notify(notify, shared.WorkflowStageGeneratingFinal, 30)
	finalRes, err := llmCli.Generate(ctx, llm.GenerateParams{
		Model: llm.ResolveModel(w.cfg, in.Model, llm.TierHigh),
		Messages: planning.BuildApprovePrompt(planning.ApprovePromptInput{
			PlanContent: plan.Content, DraftContent: draft.Content, RetrievedContext: retrievedCtx,
		}),
	})
	if err != nil {
		return nil, err
	}

	// 2. 抽取结构化 diff
	Notify(notify, shared.WorkflowStageExtractingDiff, 60)
	diffRes, err := llmCli.Generate(ctx, llm.GenerateParams{
		Model:          llm.ResolveModel(w.cfg, in.Model, llm.TierMid),
		ResponseFormat: llm.ResponseFormatJSON,
		Messages: planning.BuildApproveDiffPrompt(struct {
			PlanContent      string
			FinalContent     string
			RetrievedContext *planning.RetrievedContext
		}{plan.Content, finalRes.Content, retrievedCtx}),
	})
	if err != nil {
		return nil, err
	}

	diff, err := parseApproveDiff(diffRes.Content)
	if err != nil {
		return nil, shared.BadRequestDetails("failed to parse approve diff", err.Error())
	}

	wc := shared.EstimateWordCount(finalRes.Content)
	out := &ApproveOutput{
		ChapterID:    chapter.ID,
		WordCount:    wc,
		FinalContent: finalRes.Content,
		Diff:         *diff,
		DryRun:       in.DryRun,
	}

	if in.DryRun {
		return out, nil
	}

	// 3. 事务里:写 final + 应用 updates + 创建 newXxx + 写 sidecar
	Notify(notify, shared.WorkflowStageUpdatingResources, 80)
	err = w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		fresh, err := AssertPointersUnchanged(tx, in.BookID, in.ChapterNo, TakeSnapshot(chapter))
		if err != nil {
			return err
		}

		now := shared.NowISO()
		ver, err := nextVersion(tx, "chapter_finals", fresh.ID)
		if err != nil {
			return err
		}
		summaryStr := diff.ChapterSummary
		final := models.ChapterFinal{
			BookID: in.BookID, ChapterID: fresh.ID, ChapterNo: in.ChapterNo, VersionNo: ver,
			BasedOnDraftID: &draft.ID, Status: "active", Content: finalRes.Content,
			Summary: &summaryStr, WordCount: &wc, SourceType: shared.ChapterSourceApproved,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Create(&final).Error; err != nil {
			return err
		}
		out.FinalID = final.ID

		// new entities
		newIDs, err := createNewEntities(tx, in.BookID, diff, now)
		if err != nil {
			return err
		}
		out.NewEntityIDs = newIDs

		// 合并 actual_*_ids + new ids
		actualChars := mergeIDs(diff.ActualCharacterIDs, newIDs.CharacterIDs)
		actualFactions := mergeIDs(diff.ActualFactionIDs, newIDs.FactionIDs)
		actualItems := mergeIDs(diff.ActualItemIDs, newIDs.ItemIDs)
		actualHooks := mergeIDs(diff.ActualHookIDs, newIDs.HookIDs)
		actualWorldSettings := mergeIDs(diff.ActualWorldSettingIDs, newIDs.WorldSettingIDs)

		// 应用 updates
		applied, skipped := applyEntityUpdates(tx, in.BookID, diff.Updates, now)
		out.AppliedUpdates = applied
		out.SkippedUpdates = skipped

		// sidecar:写 retrieval_facts / story_events / chapter_segments / retrieval_documents
		Notify(notify, shared.WorkflowStagePersistingSidecar, 90)
		factsCount, err := writeSidecarFacts(tx, in.BookID, in.ChapterNo, &final, diff, now)
		if err != nil {
			return err
		}
		eventCount, err := writeSidecarEvents(tx, in.BookID, in.ChapterNo, fresh.ID, diff, now)
		if err != nil {
			return err
		}
		if err := writeChapterSegments(tx, in.BookID, in.ChapterNo, fresh.ID, finalRes.Content, now); err != nil {
			return err
		}
		if err := writeRetrievalDocument(tx, in.BookID, in.ChapterNo, fresh.ID, finalRes.Content, diff.ChapterSummary, now); err != nil {
			return err
		}
		out.SidecarFactCount = factsCount
		out.SidecarEventCount = eventCount

		// 更新 chapter:summary, word_count, actual_*_ids, status, current_final_id
		updates := map[string]any{
			"current_final_id":         final.ID,
			"status":                   shared.ChapterStatusApproved,
			"summary":                  diff.ChapterSummary,
			"word_count":               wc,
			"actual_character_ids":     mustMarshal(actualChars),
			"actual_faction_ids":       mustMarshal(actualFactions),
			"actual_item_ids":          mustMarshal(actualItems),
			"actual_hook_ids":          mustMarshal(actualHooks),
			"actual_world_setting_ids": mustMarshal(actualWorldSettings),
			"updated_at":               now,
		}
		return tx.Model(&models.Chapter{}).Where("id = ?", fresh.ID).Updates(updates).Error
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func parseApproveDiff(raw string) (*ApproveDiff, error) {
	cleaned := stripFence(strings.TrimSpace(raw))
	var diff ApproveDiff
	if err := json.Unmarshal([]byte(cleaned), &diff); err != nil {
		return nil, fmt.Errorf("decode diff: %w", err)
	}
	if diff.ChapterSummary == "" {
		return nil, fmt.Errorf("approve diff missing chapterSummary")
	}
	return &diff, nil
}

func createNewEntities(tx *gorm.DB, bookID int64, diff *ApproveDiff, now string) (newIDsView, error) {
	out := newIDsView{}
	for _, c := range diff.NewCharacters {
		row := models.Character{
			BookID: bookID, Name: c.Name, Background: shared.StrPtr(c.Summary),
			Status: "active", Keywords: shared.StrPtr(strings.Join(c.Keywords, ",")),
			CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Create(&row).Error; err != nil {
			return out, err
		}
		out.CharacterIDs = append(out.CharacterIDs, row.ID)
	}
	for _, f := range diff.NewFactions {
		row := models.Faction{
			BookID: bookID, Name: f.Name, Description: shared.StrPtr(f.Summary),
			Keywords: shared.StrPtr(strings.Join(f.Keywords, ",")),
			CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Create(&row).Error; err != nil {
			return out, err
		}
		out.FactionIDs = append(out.FactionIDs, row.ID)
	}
	for _, it := range diff.NewItems {
		row := models.Item{
			BookID: bookID, Name: it.Name, Description: shared.StrPtr(it.Summary),
			OwnerType: "none", Keywords: shared.StrPtr(strings.Join(it.Keywords, ",")),
			CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Create(&row).Error; err != nil {
			return out, err
		}
		out.ItemIDs = append(out.ItemIDs, row.ID)
	}
	for _, h := range diff.NewHooks {
		row := models.StoryHook{
			BookID: bookID, Title: h.Title, Description: shared.StrPtr(h.Description),
			Status: "open", Keywords: shared.StrPtr(strings.Join(h.Keywords, ",")),
			CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Create(&row).Error; err != nil {
			return out, err
		}
		out.HookIDs = append(out.HookIDs, row.ID)
	}
	for _, ws := range diff.NewWorldSettings {
		row := models.WorldSetting{
			BookID: bookID, Title: ws.Title, Category: ws.Category, Content: ws.Content,
			Status: "active", Keywords: shared.StrPtr(strings.Join(ws.Keywords, ",")),
			CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Create(&row).Error; err != nil {
			return out, err
		}
		out.WorldSettingIDs = append(out.WorldSettingIDs, row.ID)
	}
	for _, rel := range diff.NewRelations {
		// 校验端点存在
		if !endpointExists(tx, bookID, rel.SourceType, rel.SourceID) || !endpointExists(tx, bookID, rel.TargetType, rel.TargetID) {
			continue
		}
		row := models.Relation{
			BookID: bookID, SourceType: rel.SourceType, SourceID: rel.SourceID,
			TargetType: rel.TargetType, TargetID: rel.TargetID,
			RelationType: rel.RelationType, Intensity: rel.Intensity, Status: rel.Status,
			Description: shared.StrPtr(rel.Description),
			Keywords:    shared.StrPtr(strings.Join(rel.Keywords, ",")),
			CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Create(&row).Error; err != nil {
			return out, err
		}
		out.RelationIDs = append(out.RelationIDs, row.ID)
	}
	return out, nil
}

func endpointExists(tx *gorm.DB, bookID int64, kind string, id int64) bool {
	switch kind {
	case "character":
		var n int64
		tx.Model(&models.Character{}).Where("book_id = ? AND id = ?", bookID, id).Count(&n)
		return n > 0
	case "faction":
		var n int64
		tx.Model(&models.Faction{}).Where("book_id = ? AND id = ?", bookID, id).Count(&n)
		return n > 0
	}
	return false
}

func mergeIDs(a, b []int64) []int64 {
	if a == nil {
		a = []int64{}
	}
	return shared.DedupeInt64(append(a, b...))
}

func applyEntityUpdates(tx *gorm.DB, bookID int64, updates []EntityUpdate, now string) (applied, skipped int) {
	for _, u := range updates {
		ok := applySingleUpdate(tx, bookID, u, now)
		if ok {
			applied++
		} else {
			skipped++
		}
	}
	return
}

func applySingleUpdate(tx *gorm.DB, bookID int64, u EntityUpdate, now string) bool {
	switch u.EntityType {
	case "character":
		return applyTo(tx, &models.Character{}, "characters", bookID, u, now, allowedCharacterFields)
	case "faction":
		return applyTo(tx, &models.Faction{}, "factions", bookID, u, now, allowedFactionFields)
	case "item":
		return applyTo(tx, &models.Item{}, "items", bookID, u, now, allowedItemFields)
	case "story_hook":
		return applyTo(tx, &models.StoryHook{}, "story_hooks", bookID, u, now, allowedHookFields)
	case "world_setting":
		return applyTo(tx, &models.WorldSetting{}, "world_settings", bookID, u, now, allowedWorldSettingFields)
	case "relation":
		return applyTo(tx, &models.Relation{}, "relations", bookID, u, now, allowedRelationFields)
	}
	return false
}

var (
	allowedCharacterFields = map[string]bool{
		"alias": true, "gender": true, "age": true, "personality": true, "background": true,
		"current_location": true, "status": true, "professions": true, "levels": true, "currencies": true,
		"abilities": true, "goal": true, "append_notes": true, "keywords": true,
	}
	allowedFactionFields = map[string]bool{
		"category": true, "core_goal": true, "description": true, "leader_character_id": true,
		"headquarter": true, "status": true, "append_notes": true, "keywords": true,
	}
	allowedItemFields = map[string]bool{
		"category": true, "description": true, "owner_type": true, "owner_id": true,
		"rarity": true, "status": true, "append_notes": true, "keywords": true,
	}
	allowedHookFields = map[string]bool{
		"hook_type": true, "description": true, "source_chapter_no": true, "target_chapter_no": true,
		"status": true, "importance": true, "append_notes": true, "keywords": true,
	}
	allowedWorldSettingFields = map[string]bool{
		"category": true, "content": true, "status": true, "append_notes": true, "keywords": true,
	}
	allowedRelationFields = map[string]bool{
		"relation_type": true, "intensity": true, "status": true, "description": true,
		"append_notes": true, "keywords": true,
	}
)

// applyTo 用通用 SQL 在 tx 上执行 update,适用三类 action。
func applyTo(tx *gorm.DB, _ any, table string, bookID int64, u EntityUpdate, now string, allowed map[string]bool) bool {
	var cnt int64
	tx.Table(table).Where("book_id = ? AND id = ?", bookID, u.EntityID).Count(&cnt)
	if cnt == 0 {
		return false
	}
	updates := map[string]any{"updated_at": now}
	switch u.Action {
	case "append_notes":
		note, _ := u.Payload["note"].(string)
		if note == "" {
			return false
		}
		// 读 append_notes 现值并追加
		var current string
		row := tx.Table(table).Select("COALESCE(append_notes,'') as append_notes").Where("book_id = ? AND id = ?", bookID, u.EntityID).Row()
		_ = row.Scan(&current)
		if current == "" {
			updates["append_notes"] = note
		} else {
			updates["append_notes"] = current + "\n" + note
		}
	case "status_change":
		st, _ := u.Payload["status"].(string)
		if st == "" {
			return false
		}
		updates["status"] = st
	case "update_fields":
		for k, v := range u.Payload {
			if !allowed[k] {
				continue
			}
			updates[k] = v
		}
		if len(updates) == 1 {
			return false
		}
	default:
		return false
	}
	if err := tx.Table(table).Where("book_id = ? AND id = ?", bookID, u.EntityID).Updates(updates).Error; err != nil {
		return false
	}
	return true
}

// 写 retrieval_facts:把 carryoverFacts/openLoops/character/relation/itemStateChanges 拆成 fact 行
func writeSidecarFacts(tx *gorm.DB, bookID int64, chapterNo int, _ *models.ChapterFinal, diff *ApproveDiff, now string) (int, error) {
	type pair struct {
		factType string
		texts    []string
	}
	pairs := []pair{
		{"carryover", diff.ContinuitySnapshot.CarryoverFacts},
		{"open_loop", diff.ContinuitySnapshot.OpenLoops},
		{"character_state", diff.ContinuitySnapshot.CharacterStateChanges},
		{"relation_state", diff.ContinuitySnapshot.RelationStateChanges},
		{"item_state", diff.ContinuitySnapshot.ItemStateChanges},
		{"taboo", diff.ContinuitySnapshot.TabooContinuityMistakes},
	}
	count := 0
	cn := chapterNo
	for _, p := range pairs {
		for i, text := range p.texts {
			row := models.RetrievalFact{
				BookID: bookID, ChapterNo: &cn, FactType: p.factType,
				FactKey: fmt.Sprintf("%s:c%d:%d", p.factType, chapterNo, i),
				FactText: text, Status: "active",
				CreatedAt: now, UpdatedAt: now,
			}
			if err := tx.Create(&row).Error; err != nil {
				return count, err
			}
			count++
		}
	}
	return count, nil
}

func writeSidecarEvents(tx *gorm.DB, bookID int64, chapterNo int, chapterID int64, diff *ApproveDiff, now string) (int, error) {
	cn := chapterNo
	row := models.StoryEvent{
		BookID: bookID, ChapterID: &chapterID, ChapterNo: &cn,
		EventType: "chapter_summary",
		Title:     fmt.Sprintf("第%d章 章末事件", chapterNo),
		Summary:   diff.ChapterSummary,
		UnresolvedImpact: diff.UnresolvedImpact,
		Status:    "active",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := tx.Create(&row).Error; err != nil {
		return 0, err
	}
	return 1, nil
}

func writeChapterSegments(tx *gorm.DB, bookID int64, chapterNo int, chapterID int64, content string, now string) error {
	paragraphs := splitParagraphs(content)
	for i, p := range paragraphs {
		row := models.ChapterSegment{
			BookID: bookID, ChapterID: chapterID, ChapterNo: chapterNo,
			SegmentIndex: i, SourceType: "approved", Text: p, Status: "active",
			CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

func writeRetrievalDocument(tx *gorm.DB, bookID int64, chapterNo int, chapterID int64, content, summary string, now string) error {
	cn := chapterNo
	row := models.RetrievalDocument{
		BookID: bookID, EntityType: shared.StrPtr("chapter"), EntityID: shared.Int64Ptr(chapterID),
		Layer: "final", ChunkKey: fmt.Sprintf("chapter:%d", chapterNo), ChapterNo: &cn,
		PayloadJSON: shared.StrPtr(mustMarshal(map[string]any{"summary": summary})),
		Text:        content,
		Status:      "active",
		CreatedAt:   now, UpdatedAt: now,
	}
	return tx.Create(&row).Error
}

func splitParagraphs(s string) []string {
	out := []string{}
	for _, p := range strings.Split(s, "\n\n") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
