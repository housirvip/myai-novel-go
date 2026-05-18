package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"myai-novel-go/internal/config"
	"myai-novel-go/internal/llm"
)

type MockClient struct {
	cfg *config.Config
}

func NewMock(cfg *config.Config) *MockClient {
	return &MockClient{cfg: cfg}
}

func (m *MockClient) Generate(ctx context.Context, params llm.GenerateParams) (*llm.GenerateResult, error) {
	model := params.Model
	if model == "" {
		model = m.cfg.MockLLMModel
	}

	content, err := m.buildContent(params)
	if err != nil {
		return nil, err
	}

	inputLen := 0
	for _, msg := range params.Messages {
		inputLen += len(msg.Content)
	}
	outputLen := len(content)

	return &llm.GenerateResult{
		Provider: llm.ProviderMock,
		Model:    model,
		Content:  content,
		Usage: llm.Usage{
			InputTokens:  max1(inputLen / 4),
			OutputTokens: max1(outputLen / 4),
			TotalTokens:  max1(inputLen/4) + max1(outputLen/4),
		},
	}, nil
}

func (m *MockClient) buildContent(params llm.GenerateParams) (string, error) {
	if m.cfg.MockLLMMode == config.MockLLMModeFixture {
		if m.cfg.MockLLMFixturePath == "" {
			return "", fmt.Errorf("MOCK_LLM_FIXTURE_PATH is required when MOCK_LLM_MODE=fixture")
		}
		b, err := os.ReadFile(m.cfg.MockLLMFixturePath)
		if err != nil {
			return "", fmt.Errorf("read fixture: %w", err)
		}
		return string(b), nil
	}

	combined := strings.Builder{}
	var lastUser string
	for _, msg := range params.Messages {
		if msg.Role == llm.RoleUser {
			lastUser = msg.Content
		}
		combined.WriteString(msg.Content)
		combined.WriteString("\n")
	}
	combinedText := combined.String()

	// 关键字命中 → 返回预设响应,与原 mock.ts 行为对齐
	switch {
	case strings.Contains(combinedText, "小说审校助手") || strings.Contains(combinedText, "修复建议"):
		return jsonString(map[string]any{
			"summary":            "本章主线明确,黑铁令线索建立有效,但势力反应仍可再加强。",
			"issues":             []string{"外门其他弟子对黑铁令的反应略快,缺少一两句铺垫。", "执事长老的态度可以更鲜明,以增强悬念。"},
			"risks":              []string{"若后续不解释黑铁令特殊性,当前悬念强度可能提前透支。"},
			"continuity_checks":  []string{"主角当前身份仍为外门弟子,未出现越级资源获取。", "黑铁令当前归属与既有设定一致。"},
			"repair_suggestions": []string{"补一段旁观弟子的窃语,强化令牌不寻常。", "增加执事短暂迟疑,暗示令牌来历复杂。"},
		}), nil

	case strings.Contains(combinedText, "小说修稿助手"):
		return strings.Join([]string{
			"林夜踏入外门山门时,天色尚未完全亮起,石阶尽头的钟声一下一下传开。",
			"执事长老翻到他的名字时,指尖明显顿了一下,这才将那枚沉甸甸的黑铁令拍进他掌心。",
			"旁边几名弟子原本还在低声说笑,等看清令牌纹路后,声音却像被人掐断似地停住,只剩一句压得极低的窃语,说那东西不该出现在外门。",
			"林夜压下追问的冲动,把令牌收入袖中,却清楚地意识到,这绝不是普通的入门凭证,而是一把会将他卷进更深暗流的钥匙。",
		}, "\n\n"), nil

	case strings.Contains(combinedText, "结构化事实变更") || strings.Contains(combinedText, "updates 中的 entityType"):
		return jsonString(map[string]any{
			"chapterSummary":        "林夜入宗并获得黑铁令,正式察觉其背后隐藏的异常线索。",
			"unresolvedImpact":      "黑铁令与宗门旧案的关联仍未查清,后续必须继续追索来源。",
			"continuitySnapshot":    map[string]any{"closingBeat": "林夜将黑铁令收入袖中,踏进外门。", "carryoverFacts": []string{"林夜持有黑铁令"}, "openLoops": []string{"黑铁令来历未明"}, "characterStateChanges": []string{"林夜开始警觉宗门暗流"}, "relationStateChanges": []string{"林夜与外门弟子关系紧张"}, "itemStateChanges": []string{"黑铁令归属林夜"}, "tabooContinuityMistakes": []string{}},
			"actualCharacterIds":    []int{1},
			"actualFactionIds":      []int{1},
			"actualItemIds":         []int{1},
			"actualHookIds":         []int{1},
			"actualWorldSettingIds": []int{1},
			"newCharacters":         []any{},
			"newFactions":           []any{},
			"newItems":              []any{},
			"newHooks": []any{
				map[string]any{
					"title":       "黑铁令与宗门旧案",
					"description": "黑铁令可能与宗门旧案相关,后续需要追查来源。",
					"keywords":    []string{"黑铁令", "旧案"},
				},
			},
			"newWorldSettings": []any{},
			"newRelations": []any{
				map[string]any{
					"sourceType": "character", "sourceId": 1,
					"targetType": "faction", "targetId": 1,
					"relationType": "member", "intensity": 60, "status": "active",
					"description": "林夜正式以外门弟子身份进入青岳宗。",
					"keywords":    []string{"林夜", "青岳宗", "外门"},
				},
			},
			"updates": []any{
				map[string]any{"entityType": "story_hook", "entityId": 1, "action": "append_notes",
					"payload": map[string]any{"note": "第2章确认黑铁令在外门内部极不寻常,并引出宗门旧案方向。"}},
				map[string]any{"entityType": "item", "entityId": 1, "action": "append_notes",
					"payload": map[string]any{"note": "第2章确认黑铁令会引发外门弟子与执事的异常反应。"}},
				map[string]any{"entityType": "character", "entityId": 1, "action": "append_notes",
					"payload": map[string]any{"note": "第2章起对黑铁令来源产生明确追查意图。"}},
			},
		}), nil

	case strings.Contains(combinedText, "小说定稿助手"):
		return strings.Join([]string{
			"晨雾还压在山门石阶上,外门的钟声已经一圈圈荡开。",
			"林夜踏上最后一级台阶时,执事长老正翻看名册。对方翻到他的名字,指尖忽然顿住,随后才从木匣里取出一枚沉甸甸的黑铁令,啪地一声拍进他掌心。",
			"令牌入手冰冷,边缘刻痕像被岁月反复摩挲过。更奇怪的是,周围几名外门弟子一见那令牌,原本散漫的神色立刻变了,有人甚至下意识后退了半步。",
			"\"那东西怎么会在他手里?\"一声压得极低的窃语飘进耳中,却在执事长老抬眼的一瞬间戛然而止。",
			"林夜没有追问,只将黑铁令缓缓收入袖中。他能感觉到,自己踏进宗门的这一刻,真正推开的不是外门,而是一道更深也更危险的暗门。",
		}, "\n\n"), nil

	case strings.Contains(combinedText, "作者意图草案"):
		return "本章重点推进主角当前主线,抛出关键线索,并为后续冲突埋下钩子。", nil

	case strings.Contains(combinedText, "小说阶段摘要助手") || strings.Contains(combinedText, "阶段摘要"):
		return "本段正文概括了当前阶段的主线推进,并点出了关键冲突或结果。", nil

	case strings.Contains(combinedText, "根据章节规划创作") || strings.Contains(combinedText, "章节规划:"):
		return strings.Join([]string{
			"林夜踏入外门山门时,天色尚未完全亮起,石阶尽头的钟声正一下一下荡开。",
			"执事长老没有多看他,只在名册上划了一笔,随后将一枚沉甸甸的黑铁令拍进他掌心。",
			"那令牌冰冷异常,边缘刻痕却像被人反复摩挲过。林夜压下疑惑,将它收入袖中,却注意到周围几名弟子的目光明显变了。",
			"他很快意识到,这枚令牌不只是入门凭证,更像一把会把人拖进更深暗流的钥匙。",
		}, "\n\n"), nil

	case strings.Contains(combinedText, "请输出章节规划"):
		return strings.Join([]string{
			"本章目标:推进当前主线并强化冲突。",
			"主线:主角围绕关键线索展开行动。",
			"支线:补充人物关系和势力态度变化。",
			"出场角色:主角、关键配角、相关势力成员。",
			"关键道具:延续当前章节命中的重要物品。",
			"钩子推进:保持已有未回收钩子的悬念与推进。",
			"风险提醒:避免设定冲突,注意承接上一章状态。",
		}, "\n"), nil
	}

	if m.cfg.MockLLMMode == config.MockLLMModeJSON || params.ResponseFormat == llm.ResponseFormatJSON {
		if strings.Contains(combinedText, "intentSummary") && strings.Contains(combinedText, "mustInclude") {
			return jsonString(map[string]any{
				"intentSummary": "推进当前章节主线并埋下后续冲突",
				"keywords":      []string{"主线", "冲突", "线索"},
				"mustInclude":   []string{"主角", "关键线索"},
				"mustAvoid":     []string{"设定冲突", "人物失真"},
				"entityHints": map[string]any{
					"characters":    []string{"主角"},
					"factions":      []string{},
					"items":         []string{"关键线索"},
					"relations":     []string{},
					"hooks":         []string{"后续冲突"},
					"worldSettings": []string{},
				},
				"continuityCues": []string{"承接上一章状态", "保持人物动机一致"},
				"settingCues":    []string{"维持既有设定边界"},
				"sceneCues":      []string{"强化本章冲突场景"},
			}), nil
		}
		return jsonString(map[string]any{
			"provider": "mock",
			"summary":  m.cfg.MockLLMResponseText,
			"echo":     lastUser,
		}), nil
	}

	if lastUser == "" {
		return m.cfg.MockLLMResponseText, nil
	}
	return strings.TrimSpace(m.cfg.MockLLMResponseText + "\n\n" + lastUser), nil
}

func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

func jsonString(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
