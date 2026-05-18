package planning

// 注:这里的检索类型是原 src/domain/planning/types.ts 的 Go 投影,
// 字段名遵循 JSON camelCase 以保证前后端 API 兼容,但实际写入 chapter_plans.retrieved_context
// 的内容只需要可序列化即可。

type ManualEntityRefs struct {
	CharacterIDs    []int64 `json:"characterIds"`
	FactionIDs      []int64 `json:"factionIds"`
	ItemIDs         []int64 `json:"itemIds"`
	HookIDs         []int64 `json:"hookIds"`
	RelationIDs     []int64 `json:"relationIds"`
	WorldSettingIDs []int64 `json:"worldSettingIds"`
}

func EmptyManualRefs() ManualEntityRefs {
	return ManualEntityRefs{
		CharacterIDs:    []int64{},
		FactionIDs:      []int64{},
		ItemIDs:         []int64{},
		HookIDs:         []int64{},
		RelationIDs:     []int64{},
		WorldSettingIDs: []int64{},
	}
}

type RetrievedEntity struct {
	ID      int64   `json:"id"`
	Name    string  `json:"name,omitempty"`
	Title   string  `json:"title,omitempty"`
	Reason  string  `json:"reason"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

type RetrievedOutline struct {
	ID      int64  `json:"id"`
	Title   string `json:"title"`
	Reason  string `json:"reason"`
	Content string `json:"content"`
}

type RetrievedChapterSummary struct {
	ID        int64   `json:"id"`
	ChapterNo int     `json:"chapterNo"`
	Title     *string `json:"title"`
	Summary   *string `json:"summary"`
	Status    string  `json:"status"`
}

type EntityGroups struct {
	Hooks         []RetrievedEntity `json:"hooks"`
	Characters    []RetrievedEntity `json:"characters"`
	Factions      []RetrievedEntity `json:"factions"`
	Items         []RetrievedEntity `json:"items"`
	Relations     []RetrievedEntity `json:"relations"`
	WorldSettings []RetrievedEntity `json:"worldSettings"`
}

type RiskReminder struct {
	Text string `json:"text"`
}

type RetrievedContext struct {
	Book struct {
		ID                  int64   `json:"id"`
		Title               string  `json:"title"`
		Summary             *string `json:"summary"`
		TargetChapterCount  *int    `json:"targetChapterCount"`
		CurrentChapterCount int     `json:"currentChapterCount"`
	} `json:"book"`
	Outlines        []RetrievedOutline        `json:"outlines"`
	RecentChapters  []RetrievedChapterSummary `json:"recentChapters"`
	Hooks           []RetrievedEntity         `json:"hooks"`
	Characters      []RetrievedEntity         `json:"characters"`
	Factions        []RetrievedEntity         `json:"factions"`
	Items           []RetrievedEntity         `json:"items"`
	Relations       []RetrievedEntity         `json:"relations"`
	WorldSettings   []RetrievedEntity         `json:"worldSettings"`
	HardConstraints EntityGroups              `json:"hardConstraints"`
	SoftReferences  struct {
		Outlines       []RetrievedOutline        `json:"outlines"`
		RecentChapters []RetrievedChapterSummary `json:"recentChapters"`
		Entities       EntityGroups              `json:"entities"`
	} `json:"softReferences"`
	RiskReminders []RiskReminder `json:"riskReminders"`
}

type ExtractedIntent struct {
	IntentSummary string   `json:"intentSummary"`
	Keywords      []string `json:"keywords"`
	MustInclude   []string `json:"mustInclude"`
	MustAvoid     []string `json:"mustAvoid"`
	EntityHints   struct {
		Characters    []string `json:"characters"`
		Factions      []string `json:"factions"`
		Items         []string `json:"items"`
		Relations     []string `json:"relations"`
		Hooks         []string `json:"hooks"`
		WorldSettings []string `json:"worldSettings"`
	} `json:"entityHints"`
	ContinuityCues []string `json:"continuityCues"`
	SettingCues    []string `json:"settingCues"`
	SceneCues      []string `json:"sceneCues"`
}

type IntentConstraints struct {
	IntentSummary string   `json:"intentSummary,omitempty"`
	MustInclude   []string `json:"mustInclude,omitempty"`
	MustAvoid     []string `json:"mustAvoid,omitempty"`
}
