package planning_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"myai-novel-go/internal/config"
	"myai-novel-go/internal/db/models"
	"myai-novel-go/internal/domain/planning"
	"myai-novel-go/internal/domain/shared"
	"myai-novel-go/internal/testutil"
)

func TestRetrievalService_KeywordAndManualID(t *testing.T) {
	gdb := testutil.OpenTempDB(t)
	now := shared.NowISO()
	require.NoError(t, gdb.Create(&models.Book{Title: "T", CreatedAt: now, UpdatedAt: now}).Error)

	bookID := int64(1)
	chars := []models.Character{
		{BookID: bookID, Name: "林夜", Status: "active", Keywords: ptrStr("林夜"), CreatedAt: now, UpdatedAt: now},
		{BookID: bookID, Name: "无关人物", Status: "active", Keywords: ptrStr("路人"), CreatedAt: now, UpdatedAt: now},
		{BookID: bookID, Name: "陈雪", Status: "active", Keywords: ptrStr("陈雪"), CreatedAt: now, UpdatedAt: now},
	}
	for i := range chars {
		require.NoError(t, gdb.Create(&chars[i]).Error)
	}
	hooks := []models.StoryHook{
		{BookID: bookID, Title: "黑铁令异常", Status: "open", HookType: ptrStr("mystery"), CreatedAt: now, UpdatedAt: now},
	}
	for i := range hooks {
		require.NoError(t, gdb.Create(&hooks[i]).Error)
	}

	cfg := defaultPlanningCfg()
	svc := planning.NewRetrievalService(gdb, cfg)
	out, err := svc.Retrieve(context.Background(), planning.RetrieveParams{
		BookID: bookID, ChapterNo: 5, Keywords: []string{"林夜"},
		ManualRefs: planning.ManualEntityRefs{CharacterIDs: []int64{3}}, // 手动指定陈雪
	})
	require.NoError(t, err)
	require.NotEmpty(t, out.Characters, "应返回命中关键词或被手动指定的人物")
	// 检查"林夜"和"陈雪"都进了候选,"无关人物"没有
	names := map[string]bool{}
	for _, c := range out.Characters {
		names[c.Name] = true
	}
	require.True(t, names["林夜"])
	require.True(t, names["陈雪"], "manual_id 应进入候选即使无关键词命中")
	require.False(t, names["无关人物"])
	// open hook 自动纳入
	require.NotEmpty(t, out.Hooks)
}

func ptrStr(s string) *string { return &s }

func defaultPlanningCfg() *config.Config {
	return &config.Config{
		PlanningRetrievalOutlineLimit:       3,
		PlanningRetrievalRecentChapterLimit: 8,
		PlanningRetrievalHookLimit:          24,
		PlanningRetrievalCharacterLimit:     32,
		PlanningRetrievalFactionLimit:       20,
		PlanningRetrievalItemLimit:          20,
		PlanningRetrievalRelationLimit:      32,
		PlanningRetrievalWorldSettingLimit:  20,
		PlanningRetrievalEntityScanLimit:    1500,
		PlanningRetrievalReranker:           "heuristic",
		PlanningRetrievalEmbeddingProvider:  "none",
	}
}
