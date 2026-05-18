package planning

import (
	"fmt"
	"strings"

	"myai-novel-go/internal/llm"
)

// 注:这里的 prompt 文本和原 src/domain/planning/prompts.ts 的语义对齐(都是中文 prompt + 关键字
// 触发原 mock provider 命中分支),但具体措辞做了 Go-side 简化。重要的是关键字保留,
// 这样 mock provider 才能匹配到对应分支。

func userMsg(content string) llm.Message    { return llm.Message{Role: llm.RoleUser, Content: content} }
func systemMsg(content string) llm.Message  { return llm.Message{Role: llm.RoleSystem, Content: content} }

type PlanPromptInput struct {
	BookTitle         string
	ChapterNo         int
	AuthorIntent      string
	IntentConstraints IntentConstraints
	RetrievedContext  *RetrievedContext
	TargetWords       int
}

func BuildIntentGenerationPrompt(in struct {
	BookTitle        string
	ChapterNo        int
	OutlinesText     string
	RecentChapterText string
	ManualFocusText  string
	RiskReminderText string
}) []llm.Message {
	sys := "你是协助小说作者的写作助手,请基于已有上下文为下一章生成一份'作者意图草案',字数不超过 200 字。"
	user := fmt.Sprintf(`请为《%s》第 %d 章生成一段作者意图草案,聚焦本章主线、关键冲突和必带钩子。

【相关大纲】
%s

【最近章节摘要】
%s

【手动指定焦点实体】
%s

【风险提醒】
%s

请用 1~3 句话陈述本章的写作意图,不要返回 JSON。`,
		in.BookTitle, in.ChapterNo, in.OutlinesText, in.RecentChapterText, in.ManualFocusText, in.RiskReminderText)
	return []llm.Message{systemMsg(sys), userMsg(user)}
}

func BuildKeywordExtractionPrompt(authorIntent string) []llm.Message {
	sys := "你是关键词提取助手。"
	user := fmt.Sprintf(`基于下面的作者意图,抽取一组关键词与必须包含/必须避免清单。

作者意图:
%s

请输出 JSON,字段包括 intentSummary / keywords / mustInclude / mustAvoid / entityHints / continuityCues / settingCues / sceneCues。`, authorIntent)
	return []llm.Message{systemMsg(sys), userMsg(user)}
}

func BuildPlanPrompt(in PlanPromptInput) []llm.Message {
	sys := "你是小说章节规划助手。请基于已有设定和检索到的事实输出本章规划。"
	user := fmt.Sprintf(`请输出章节规划。

书名:《%s》
当前章节:第 %d 章
目标字数:%d

作者意图:
%s

意图摘要:%s
必须包含:%s
必须避免:%s

【相关大纲】
%s

【最近章节摘要】
%s

【硬约束实体】
%s

【参考实体】
%s

请按照'章节目标 / 主线 / 支线 / 出场角色 / 关键道具 / 钩子推进 / 风险提醒'分块输出,不要返回 JSON。
注意:章节规划:`,
		in.BookTitle, in.ChapterNo, in.TargetWords,
		in.AuthorIntent, in.IntentConstraints.IntentSummary,
		strings.Join(in.IntentConstraints.MustInclude, "/"),
		strings.Join(in.IntentConstraints.MustAvoid, "/"),
		formatOutlines(in.RetrievedContext.Outlines),
		formatRecentChapters(in.RetrievedContext.RecentChapters),
		formatHardConstraints(in.RetrievedContext.HardConstraints),
		formatSoftEntities(in.RetrievedContext.SoftReferences.Entities),
	)
	return []llm.Message{systemMsg(sys), userMsg(user)}
}

type DraftPromptInput struct {
	PlanContent       string
	IntentConstraints IntentConstraints
	RetrievedContext  *RetrievedContext
	TargetWords       int
}

func BuildDraftPrompt(in DraftPromptInput) []llm.Message {
	sys := "你是小说创作助手。"
	user := fmt.Sprintf(`请根据章节规划创作正文,目标字数约 %d 字。
注意:必须包含:%s,必须避免:%s。

章节规划:
%s

【参考实体】
%s

请直接输出正文,不要 JSON 不要标题序号。`,
		in.TargetWords,
		strings.Join(in.IntentConstraints.MustInclude, "/"),
		strings.Join(in.IntentConstraints.MustAvoid, "/"),
		in.PlanContent,
		formatHardConstraints(in.RetrievedContext.HardConstraints),
	)
	return []llm.Message{systemMsg(sys), userMsg(user)}
}

func BuildDraftLengthRepairPrompt(in struct {
	DraftContent     string
	PlanContent      string
	TargetWords      int
	CurrentWordCount int
}) []llm.Message {
	sys := "你是小说修稿助手,负责按目标字数对正文做长度调整,但严格保留原情节顺序与关键事实。"
	user := fmt.Sprintf(`目标字数 %d,当前字数 %d。请把以下正文调整到目标字数附近,内容核心保持不变:

【章节规划摘要】
%s

【当前正文】
%s

请直接输出修改后的正文。`, in.TargetWords, in.CurrentWordCount, in.PlanContent, in.DraftContent)
	return []llm.Message{systemMsg(sys), userMsg(user)}
}

func BuildDraftAggressiveCompressionPrompt(in struct {
	DraftContent     string
	PlanContent      string
	TargetWords      int
	CurrentWordCount int
}) []llm.Message {
	sys := "你是小说修稿助手,擅长大幅压缩冗长正文。"
	user := fmt.Sprintf(`正文已经超出目标字数 %d 太多,当前 %d 字。请大幅删减描写性段落、合并镜头,保留关键情节节点。

【章节规划摘要】
%s

【当前正文】
%s

请直接输出压缩后的正文。`, in.TargetWords, in.CurrentWordCount, in.PlanContent, in.DraftContent)
	return []llm.Message{systemMsg(sys), userMsg(user)}
}

type ReviewPromptInput struct {
	PlanContent      string
	DraftContent     string
	RetrievedContext *RetrievedContext
}

func BuildReviewPrompt(in ReviewPromptInput) []llm.Message {
	sys := "你是小说审校助手,请输出 JSON 格式的审校结果。"
	user := fmt.Sprintf(`请审校以下章节正文,输出 JSON,字段包括 summary, issues, risks, continuity_checks, repair_suggestions(均为字符串数组,summary 为字符串)。

【章节规划】
%s

【章节正文】
%s

【关键参考事实】
%s

请只返回 JSON 不要 markdown 代码块。修复建议必须可执行。`,
		in.PlanContent, in.DraftContent, formatHardConstraints(in.RetrievedContext.HardConstraints))
	return []llm.Message{systemMsg(sys), userMsg(user)}
}

type RepairPromptInput struct {
	PlanContent       string
	DraftContent      string
	ReviewContent     string
	IntentConstraints IntentConstraints
	RetrievedContext  *RetrievedContext
}

func BuildRepairPrompt(in RepairPromptInput) []llm.Message {
	sys := "你是小说修稿助手,基于审校结论修订正文,保持原有情节顺序与人物动机一致。"
	user := fmt.Sprintf(`请基于审校建议修订下面的正文。

【章节规划】
%s

【当前正文】
%s

【审校结论 JSON】
%s

请直接输出修订后的完整正文。`, in.PlanContent, in.DraftContent, in.ReviewContent)
	return []llm.Message{systemMsg(sys), userMsg(user)}
}

type ApprovePromptInput struct {
	PlanContent      string
	DraftContent     string
	RetrievedContext *RetrievedContext
}

func BuildApprovePrompt(in ApprovePromptInput) []llm.Message {
	sys := "你是小说定稿助手,产出最终发表正文。"
	user := fmt.Sprintf(`请基于章节规划和当前正文,产出最终发表版本(精修语言、保留关键事实)。

【章节规划】
%s

【当前正文】
%s

请直接输出最终正文,不要附带说明。`, in.PlanContent, in.DraftContent)
	return []llm.Message{systemMsg(sys), userMsg(user)}
}

func BuildApproveDiffPrompt(in struct {
	PlanContent      string
	FinalContent     string
	RetrievedContext *RetrievedContext
}) []llm.Message {
	sys := "你是结构化事实变更抽取助手,需要输出 updates 中的 entityType 和动作。"
	user := fmt.Sprintf(`请基于本章规划与最终正文,产出本章引发的事实变更 JSON。

字段说明:
- chapterSummary, unresolvedImpact
- continuitySnapshot { closingBeat, carryoverFacts[], openLoops[], characterStateChanges[], relationStateChanges[], itemStateChanges[], tabooContinuityMistakes[] }
- actualCharacterIds[], actualFactionIds[], actualItemIds[], actualHookIds[], actualWorldSettingIds[]
- newCharacters[]/newFactions[]/newItems[]/newHooks[]/newWorldSettings[]/newRelations[]
- updates[] { entityType, entityId, action(append_notes|update_fields|status_change), payload }

【章节规划】
%s

【最终正文】
%s

只返回 JSON 不要 markdown。`, in.PlanContent, in.FinalContent)
	return []llm.Message{systemMsg(sys), userMsg(user)}
}

func BuildStageSummaryPrompt(stage, content string) []llm.Message {
	sys := "你是小说阶段摘要助手,请输出该阶段的内容要点摘要。"
	user := fmt.Sprintf("阶段摘要:%s\n\n%s\n\n请用 1~2 句话总结本阶段的核心内容,不要 JSON。", stage, content)
	return []llm.Message{systemMsg(sys), userMsg(user)}
}

func formatOutlines(outlines []RetrievedOutline) string {
	if len(outlines) == 0 {
		return "暂无命中大纲。"
	}
	lines := make([]string, 0, len(outlines))
	for _, o := range outlines {
		lines = append(lines, "- "+strings.ReplaceAll(o.Content, "\n", " / "))
	}
	return strings.Join(lines, "\n")
}

func formatRecentChapters(rows []RetrievedChapterSummary) string {
	if len(rows) == 0 {
		return "暂无前文章节。"
	}
	lines := make([]string, 0, len(rows))
	for _, r := range rows {
		title := ""
		if r.Title != nil {
			title = *r.Title
		}
		summary := ""
		if r.Summary != nil {
			summary = *r.Summary
		}
		lines = append(lines, fmt.Sprintf("- 第%d章 %s:%s", r.ChapterNo, title, summary))
	}
	return strings.Join(lines, "\n")
}

func formatHardConstraints(hc EntityGroups) string {
	parts := []string{}
	if names := entityNames("人物", hc.Characters); names != "" {
		parts = append(parts, names)
	}
	if names := entityNames("势力", hc.Factions); names != "" {
		parts = append(parts, names)
	}
	if names := entityNames("物品", hc.Items); names != "" {
		parts = append(parts, names)
	}
	if names := entityNames("钩子", hc.Hooks); names != "" {
		parts = append(parts, names)
	}
	if names := entityNames("世界设定", hc.WorldSettings); names != "" {
		parts = append(parts, names)
	}
	if len(parts) == 0 {
		return "无"
	}
	return strings.Join(parts, "\n")
}

func formatSoftEntities(eg EntityGroups) string {
	lines := []string{}
	for _, e := range eg.Characters {
		lines = append(lines, "- 人物 "+nameOrTitle(e)+":"+firstLine(e.Content))
	}
	for _, e := range eg.Factions {
		lines = append(lines, "- 势力 "+nameOrTitle(e)+":"+firstLine(e.Content))
	}
	for _, e := range eg.Items {
		lines = append(lines, "- 物品 "+nameOrTitle(e)+":"+firstLine(e.Content))
	}
	for _, e := range eg.Hooks {
		lines = append(lines, "- 钩子 "+nameOrTitle(e)+":"+firstLine(e.Content))
	}
	if len(lines) == 0 {
		return "无"
	}
	return strings.Join(lines, "\n")
}

func entityNames(label string, list []RetrievedEntity) string {
	if len(list) == 0 {
		return ""
	}
	names := make([]string, 0, len(list))
	for _, e := range list {
		names = append(names, nameOrTitle(e))
	}
	return label + ":" + strings.Join(names, "、")
}

func nameOrTitle(e RetrievedEntity) string {
	if e.Name != "" {
		return e.Name
	}
	if e.Title != "" {
		return e.Title
	}
	return fmt.Sprintf("ID:%d", e.ID)
}

func firstLine(content string) string {
	idx := strings.Index(content, "\n")
	if idx < 0 {
		return content
	}
	return content[:idx]
}
