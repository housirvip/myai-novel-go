package planning

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"myai-novel-go/internal/db/models"
	"myai-novel-go/internal/domain/shared"
)

// BuildEmbeddingDocumentsForBook 从 DB 拉出整本书的 6 类实体并拼成嵌入文档,
// 文本拼接规则对齐原 embedding-text-*.ts。
func BuildEmbeddingDocumentsForBook(ctx context.Context, db *gorm.DB, bookID int64) ([]EmbeddingDocument, error) {
	docs := make([]EmbeddingDocument, 0, 256)

	var chars []models.Character
	if err := db.WithContext(ctx).Where("book_id = ?", bookID).Find(&chars).Error; err != nil {
		return nil, err
	}
	for _, r := range chars {
		text := buildCharacterText(r)
		if text == "" {
			continue
		}
		docs = append(docs, EmbeddingDocument{
			BookID: bookID, EntityType: "character", EntityID: r.ID,
			ChunkKey: fmt.Sprintf("character:%d", r.ID), DisplayName: r.Name, Text: text,
		})
	}

	var factions []models.Faction
	if err := db.WithContext(ctx).Where("book_id = ?", bookID).Find(&factions).Error; err != nil {
		return nil, err
	}
	for _, r := range factions {
		text := buildFactionText(r)
		if text == "" {
			continue
		}
		docs = append(docs, EmbeddingDocument{
			BookID: bookID, EntityType: "faction", EntityID: r.ID,
			ChunkKey: fmt.Sprintf("faction:%d", r.ID), DisplayName: r.Name, Text: text,
		})
	}

	var items []models.Item
	if err := db.WithContext(ctx).Where("book_id = ?", bookID).Find(&items).Error; err != nil {
		return nil, err
	}
	for _, r := range items {
		text := buildItemText(r)
		if text == "" {
			continue
		}
		docs = append(docs, EmbeddingDocument{
			BookID: bookID, EntityType: "item", EntityID: r.ID,
			ChunkKey: fmt.Sprintf("item:%d", r.ID), DisplayName: r.Name, Text: text,
		})
	}

	var hooks []models.StoryHook
	if err := db.WithContext(ctx).Where("book_id = ?", bookID).Find(&hooks).Error; err != nil {
		return nil, err
	}
	for _, r := range hooks {
		text := buildHookText(r)
		if text == "" {
			continue
		}
		docs = append(docs, EmbeddingDocument{
			BookID: bookID, EntityType: "hook", EntityID: r.ID,
			ChunkKey: fmt.Sprintf("hook:%d", r.ID), DisplayName: r.Title, Text: text,
		})
	}

	var ws []models.WorldSetting
	if err := db.WithContext(ctx).Where("book_id = ?", bookID).Find(&ws).Error; err != nil {
		return nil, err
	}
	for _, r := range ws {
		text := buildWorldSettingText(r)
		if text == "" {
			continue
		}
		docs = append(docs, EmbeddingDocument{
			BookID: bookID, EntityType: "world_setting", EntityID: r.ID,
			ChunkKey: fmt.Sprintf("world_setting:%d", r.ID), DisplayName: r.Title, Text: text,
		})
	}

	var rels []models.Relation
	if err := db.WithContext(ctx).Where("book_id = ?", bookID).Find(&rels).Error; err != nil {
		return nil, err
	}
	for _, r := range rels {
		text := buildRelationText(r)
		if text == "" {
			continue
		}
		docs = append(docs, EmbeddingDocument{
			BookID: bookID, EntityType: "relation", EntityID: r.ID,
			ChunkKey:    fmt.Sprintf("relation:%d", r.ID),
			DisplayName: fmt.Sprintf("%s/%s#%d-%s#%d", r.RelationType, r.SourceType, r.SourceID, r.TargetType, r.TargetID),
			Text:        text,
			RelationEndpoints: map[string]any{
				"sourceType": r.SourceType, "sourceId": r.SourceID,
				"targetType": r.TargetType, "targetId": r.TargetID,
			},
			RelationMetadata: map[string]any{"relationType": r.RelationType},
		})
	}
	return docs, nil
}

func buildCharacterText(r models.Character) string {
	parts := []string{
		"姓名:" + r.Name,
		"别名:" + shared.DerefStr(r.Alias),
		"性格:" + shared.DerefStr(r.Personality),
		"背景:" + shared.DerefStr(r.Background),
		"目标:" + shared.DerefStr(r.Goal),
		"位置:" + shared.DerefStr(r.CurrentLocation),
		"状态:" + r.Status,
		"备注:" + shared.DerefStr(r.AppendNotes),
		"关键词:" + shared.DerefStr(r.Keywords),
	}
	return joinNonEmpty(parts)
}

func buildFactionText(r models.Faction) string {
	parts := []string{
		"势力:" + r.Name,
		"类别:" + shared.DerefStr(r.Category),
		"核心目标:" + shared.DerefStr(r.CoreGoal),
		"描述:" + shared.DerefStr(r.Description),
		"状态:" + shared.DerefStr(r.Status),
		"备注:" + shared.DerefStr(r.AppendNotes),
		"关键词:" + shared.DerefStr(r.Keywords),
	}
	return joinNonEmpty(parts)
}

func buildItemText(r models.Item) string {
	parts := []string{
		"物品:" + r.Name,
		"类别:" + shared.DerefStr(r.Category),
		"描述:" + shared.DerefStr(r.Description),
		"归属:" + r.OwnerType,
		"稀有度:" + shared.DerefStr(r.Rarity),
		"状态:" + shared.DerefStr(r.Status),
		"备注:" + shared.DerefStr(r.AppendNotes),
		"关键词:" + shared.DerefStr(r.Keywords),
	}
	return joinNonEmpty(parts)
}

func buildHookText(r models.StoryHook) string {
	target := ""
	if r.TargetChapterNo != nil {
		target = fmt.Sprintf("target_chapter_no=%d", *r.TargetChapterNo)
	}
	parts := []string{
		"钩子:" + r.Title,
		"类型:" + shared.DerefStr(r.HookType),
		"描述:" + shared.DerefStr(r.Description),
		"重要度:" + shared.DerefStr(r.Importance),
		"状态:" + r.Status,
		target,
		"备注:" + shared.DerefStr(r.AppendNotes),
		"关键词:" + shared.DerefStr(r.Keywords),
	}
	return joinNonEmpty(parts)
}

func buildWorldSettingText(r models.WorldSetting) string {
	parts := []string{
		"设定:" + r.Title,
		"类别:" + r.Category,
		"内容:" + r.Content,
		"备注:" + shared.DerefStr(r.AppendNotes),
		"关键词:" + shared.DerefStr(r.Keywords),
	}
	return joinNonEmpty(parts)
}

func buildRelationText(r models.Relation) string {
	parts := []string{
		"关系类型:" + r.RelationType,
		fmt.Sprintf("源:%s#%d", r.SourceType, r.SourceID),
		fmt.Sprintf("目标:%s#%d", r.TargetType, r.TargetID),
		"描述:" + shared.DerefStr(r.Description),
		"状态:" + shared.DerefStr(r.Status),
		"备注:" + shared.DerefStr(r.AppendNotes),
		"关键词:" + shared.DerefStr(r.Keywords),
	}
	return joinNonEmpty(parts)
}

func joinNonEmpty(parts []string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		// 跳过仅有冒号前缀(空值)的占位行
		if p == "" || strings.HasSuffix(p, ":") {
			continue
		}
		out = append(out, p)
	}
	return strings.Join(out, "\n")
}
