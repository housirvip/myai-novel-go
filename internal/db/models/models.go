package models

// 18 张表的 GORM 模型,字段名保持 snake_case 与原 SQLite/MySQL schema 完全对齐。
// JSON 字段(retrieved_context / intent_keywords / participant_entity_refs 等)在 DB 层用
// string 存储,业务侧再 json.Marshal/Unmarshal。

type Book struct {
	ID                  int64   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OwnerUserID         *int64  `gorm:"column:owner_user_id;index:idx_books_owner_user_id" json:"ownerUserId"`
	Title               string  `gorm:"column:title;type:varchar(255);not null" json:"title"`
	Summary             *string `gorm:"column:summary;type:text" json:"summary"`
	TargetChapterCount  *int    `gorm:"column:target_chapter_count" json:"targetChapterCount"`
	CurrentChapterCount int     `gorm:"column:current_chapter_count;not null;default:0" json:"currentChapterCount"`
	Status              string  `gorm:"column:status;type:varchar(32);not null;default:'planning'" json:"status"`
	Metadata            *string `gorm:"column:metadata;type:text" json:"metadata"`
	CreatedAt           string  `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt           string  `gorm:"column:updated_at;not null" json:"updatedAt"`
}

func (Book) TableName() string { return "books" }

type Outline struct {
	ID             int64   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	BookID         int64   `gorm:"column:book_id;not null;index:idx_outlines_book" json:"bookId"`
	VolumeNo       *int    `gorm:"column:volume_no" json:"volumeNo"`
	VolumeTitle    *string `gorm:"column:volume_title;type:varchar(255)" json:"volumeTitle"`
	ChapterStartNo *int    `gorm:"column:chapter_start_no" json:"chapterStartNo"`
	ChapterEndNo   *int    `gorm:"column:chapter_end_no" json:"chapterEndNo"`
	OutlineLevel   string  `gorm:"column:outline_level;type:varchar(32);not null" json:"outlineLevel"`
	Title          string  `gorm:"column:title;type:varchar(255);not null" json:"title"`
	StoryCore      *string `gorm:"column:story_core;type:text" json:"storyCore"`
	MainPlot       *string `gorm:"column:main_plot;type:text" json:"mainPlot"`
	SubPlot        *string `gorm:"column:sub_plot;type:text" json:"subPlot"`
	Foreshadowing  *string `gorm:"column:foreshadowing;type:text" json:"foreshadowing"`
	ExpectedPayoff *string `gorm:"column:expected_payoff;type:text" json:"expectedPayoff"`
	Notes          *string `gorm:"column:notes;type:text" json:"notes"`
	CreatedAt      string  `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt      string  `gorm:"column:updated_at;not null" json:"updatedAt"`
}

func (Outline) TableName() string { return "outlines" }

type WorldSetting struct {
	ID          int64   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	BookID      int64   `gorm:"column:book_id;not null;index:idx_world_settings_book_category,priority:1;index:idx_world_settings_book_status,priority:1" json:"bookId"`
	Title       string  `gorm:"column:title;type:varchar(255);not null" json:"title"`
	Category    string  `gorm:"column:category;type:varchar(64);not null;index:idx_world_settings_book_category,priority:2" json:"category"`
	Content     string  `gorm:"column:content;type:text;not null" json:"content"`
	Status      string  `gorm:"column:status;type:varchar(32);not null;default:'active';index:idx_world_settings_book_status,priority:2" json:"status"`
	AppendNotes *string `gorm:"column:append_notes;type:text" json:"appendNotes"`
	Keywords    *string `gorm:"column:keywords;type:text" json:"keywords"`
	CreatedAt   string  `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt   string  `gorm:"column:updated_at;not null" json:"updatedAt"`
}

func (WorldSetting) TableName() string { return "world_settings" }

type Character struct {
	ID              int64   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	BookID          int64   `gorm:"column:book_id;not null;index:idx_characters_book_name,priority:1;index:idx_characters_book_status,priority:1" json:"bookId"`
	Name            string  `gorm:"column:name;type:varchar(255);not null;index:idx_characters_book_name,priority:2" json:"name"`
	Alias           *string `gorm:"column:alias;type:text" json:"alias"`
	Gender          *string `gorm:"column:gender;type:varchar(32)" json:"gender"`
	Age             *int    `gorm:"column:age" json:"age"`
	Personality     *string `gorm:"column:personality;type:text" json:"personality"`
	Background      *string `gorm:"column:background;type:text" json:"background"`
	CurrentLocation *string `gorm:"column:current_location;type:varchar(255)" json:"currentLocation"`
	Status          string  `gorm:"column:status;type:varchar(32);not null;default:'active';index:idx_characters_book_status,priority:2" json:"status"`
	Professions     *string `gorm:"column:professions;type:text" json:"professions"`
	Levels          *string `gorm:"column:levels;type:text" json:"levels"`
	Currencies      *string `gorm:"column:currencies;type:text" json:"currencies"`
	Abilities       *string `gorm:"column:abilities;type:text" json:"abilities"`
	Goal            *string `gorm:"column:goal;type:text" json:"goal"`
	AppendNotes     *string `gorm:"column:append_notes;type:text" json:"appendNotes"`
	Keywords        *string `gorm:"column:keywords;type:text" json:"keywords"`
	CreatedAt       string  `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt       string  `gorm:"column:updated_at;not null" json:"updatedAt"`
}

func (Character) TableName() string { return "characters" }

type Faction struct {
	ID                int64   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	BookID            int64   `gorm:"column:book_id;not null;index:idx_factions_book_name,priority:1;index:idx_factions_book_status,priority:1" json:"bookId"`
	Name              string  `gorm:"column:name;type:varchar(255);not null;index:idx_factions_book_name,priority:2" json:"name"`
	Category          *string `gorm:"column:category;type:varchar(64)" json:"category"`
	CoreGoal          *string `gorm:"column:core_goal;type:text" json:"coreGoal"`
	Description       *string `gorm:"column:description;type:text" json:"description"`
	LeaderCharacterID *int64  `gorm:"column:leader_character_id" json:"leaderCharacterId"`
	Headquarter       *string `gorm:"column:headquarter;type:varchar(255)" json:"headquarter"`
	Status            *string `gorm:"column:status;type:varchar(32);index:idx_factions_book_status,priority:2" json:"status"`
	AppendNotes       *string `gorm:"column:append_notes;type:text" json:"appendNotes"`
	Keywords          *string `gorm:"column:keywords;type:text" json:"keywords"`
	CreatedAt         string  `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt         string  `gorm:"column:updated_at;not null" json:"updatedAt"`
}

func (Faction) TableName() string { return "factions" }

type Relation struct {
	ID           int64   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	BookID       int64   `gorm:"column:book_id;not null;index:idx_relations_book_source,priority:1;index:idx_relations_book_target,priority:1" json:"bookId"`
	SourceType   string  `gorm:"column:source_type;type:varchar(32);not null;index:idx_relations_book_source,priority:2" json:"sourceType"`
	SourceID     int64   `gorm:"column:source_id;not null;index:idx_relations_book_source,priority:3" json:"sourceId"`
	TargetType   string  `gorm:"column:target_type;type:varchar(32);not null;index:idx_relations_book_target,priority:2" json:"targetType"`
	TargetID     int64   `gorm:"column:target_id;not null;index:idx_relations_book_target,priority:3" json:"targetId"`
	RelationType string  `gorm:"column:relation_type;type:varchar(64);not null" json:"relationType"`
	Intensity    *int    `gorm:"column:intensity" json:"intensity"`
	Status       *string `gorm:"column:status;type:varchar(32)" json:"status"`
	Description  *string `gorm:"column:description;type:text" json:"description"`
	AppendNotes  *string `gorm:"column:append_notes;type:text" json:"appendNotes"`
	Keywords     *string `gorm:"column:keywords;type:text" json:"keywords"`
	CreatedAt    string  `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt    string  `gorm:"column:updated_at;not null" json:"updatedAt"`
}

func (Relation) TableName() string { return "relations" }

type Item struct {
	ID          int64   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	BookID      int64   `gorm:"column:book_id;not null;index:idx_items_book_owner,priority:1;index:idx_items_book_name,priority:1" json:"bookId"`
	Name        string  `gorm:"column:name;type:varchar(255);not null;index:idx_items_book_name,priority:2" json:"name"`
	Category    *string `gorm:"column:category;type:varchar(64)" json:"category"`
	Description *string `gorm:"column:description;type:text" json:"description"`
	OwnerType   string  `gorm:"column:owner_type;type:varchar(32);not null;index:idx_items_book_owner,priority:2" json:"ownerType"`
	OwnerID     *int64  `gorm:"column:owner_id;index:idx_items_book_owner,priority:3" json:"ownerId"`
	Rarity      *string `gorm:"column:rarity;type:varchar(32)" json:"rarity"`
	Status      *string `gorm:"column:status;type:varchar(32)" json:"status"`
	AppendNotes *string `gorm:"column:append_notes;type:text" json:"appendNotes"`
	Keywords    *string `gorm:"column:keywords;type:text" json:"keywords"`
	CreatedAt   string  `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt   string  `gorm:"column:updated_at;not null" json:"updatedAt"`
}

func (Item) TableName() string { return "items" }

type StoryHook struct {
	ID              int64   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	BookID          int64   `gorm:"column:book_id;not null;index:idx_story_hooks_book_status,priority:1" json:"bookId"`
	Title           string  `gorm:"column:title;type:varchar(255);not null" json:"title"`
	HookType        *string `gorm:"column:hook_type;type:varchar(64)" json:"hookType"`
	Description     *string `gorm:"column:description;type:text" json:"description"`
	SourceChapterNo *int    `gorm:"column:source_chapter_no" json:"sourceChapterNo"`
	TargetChapterNo *int    `gorm:"column:target_chapter_no" json:"targetChapterNo"`
	Status          string  `gorm:"column:status;type:varchar(32);not null;default:'open';index:idx_story_hooks_book_status,priority:2" json:"status"`
	Importance      *string `gorm:"column:importance;type:varchar(32)" json:"importance"`
	AppendNotes     *string `gorm:"column:append_notes;type:text" json:"appendNotes"`
	Keywords        *string `gorm:"column:keywords;type:text" json:"keywords"`
	CreatedAt       string  `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt       string  `gorm:"column:updated_at;not null" json:"updatedAt"`
}

func (StoryHook) TableName() string { return "story_hooks" }

type Chapter struct {
	ID                    int64   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	BookID                int64   `gorm:"column:book_id;not null;uniqueIndex:uniq_chapters_book_chapter_no,priority:1;index:idx_chapters_book_status,priority:1" json:"bookId"`
	ChapterNo             int     `gorm:"column:chapter_no;not null;uniqueIndex:uniq_chapters_book_chapter_no,priority:2" json:"chapterNo"`
	Title                 *string `gorm:"column:title;type:varchar(255)" json:"title"`
	Summary               *string `gorm:"column:summary;type:text" json:"summary"`
	WordCount             *int    `gorm:"column:word_count" json:"wordCount"`
	TargetWordCount       *int    `gorm:"column:target_word_count" json:"targetWordCount"`
	Status                string  `gorm:"column:status;type:varchar(32);not null;default:'todo';index:idx_chapters_book_status,priority:2" json:"status"`
	CurrentPlanID         *int64  `gorm:"column:current_plan_id" json:"currentPlanId"`
	CurrentDraftID        *int64  `gorm:"column:current_draft_id" json:"currentDraftId"`
	CurrentReviewID       *int64  `gorm:"column:current_review_id" json:"currentReviewId"`
	CurrentFinalID        *int64  `gorm:"column:current_final_id" json:"currentFinalId"`
	ActualCharacterIDs    *string `gorm:"column:actual_character_ids;type:text" json:"actualCharacterIds"`
	ActualFactionIDs      *string `gorm:"column:actual_faction_ids;type:text" json:"actualFactionIds"`
	ActualItemIDs         *string `gorm:"column:actual_item_ids;type:text" json:"actualItemIds"`
	ActualHookIDs         *string `gorm:"column:actual_hook_ids;type:text" json:"actualHookIds"`
	ActualWorldSettingIDs *string `gorm:"column:actual_world_setting_ids;type:text" json:"actualWorldSettingIds"`
	CreatedAt             string  `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt             string  `gorm:"column:updated_at;not null" json:"updatedAt"`
}

func (Chapter) TableName() string { return "chapters" }

type ChapterPlan struct {
	ID                int64   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	BookID            int64   `gorm:"column:book_id;not null;index:idx_chapter_plans_book_status,priority:1" json:"bookId"`
	ChapterID         int64   `gorm:"column:chapter_id;not null;index" json:"chapterId"`
	ChapterNo         int     `gorm:"column:chapter_no;not null" json:"chapterNo"`
	VersionNo         int     `gorm:"column:version_no;not null" json:"versionNo"`
	Status            string  `gorm:"column:status;type:varchar(32);not null;default:'active';index:idx_chapter_plans_book_status,priority:2" json:"status"`
	AuthorIntent      *string `gorm:"column:author_intent;type:text" json:"authorIntent"`
	IntentSource      string  `gorm:"column:intent_source;type:varchar(32);not null" json:"intentSource"`
	IntentSummary     *string `gorm:"column:intent_summary;type:text" json:"intentSummary"`
	IntentKeywords    *string `gorm:"column:intent_keywords;type:text" json:"intentKeywords"`
	IntentMustInclude *string `gorm:"column:intent_must_include;type:text" json:"intentMustInclude"`
	IntentMustAvoid   *string `gorm:"column:intent_must_avoid;type:text" json:"intentMustAvoid"`
	ManualEntityRefs  *string `gorm:"column:manual_entity_refs;type:text" json:"manualEntityRefs"`
	RetrievedContext  *string `gorm:"column:retrieved_context;type:longtext" json:"retrievedContext"`
	Content           string  `gorm:"column:content;type:longtext;not null" json:"content"`
	Model             *string `gorm:"column:model;type:varchar(255)" json:"model"`
	Provider          *string `gorm:"column:provider;type:varchar(64)" json:"provider"`
	SourceType        string  `gorm:"column:source_type;type:varchar(32);not null" json:"sourceType"`
	CreatedAt         string  `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt         string  `gorm:"column:updated_at;not null" json:"updatedAt"`
}

func (ChapterPlan) TableName() string { return "chapter_plans" }

type ChapterDraft struct {
	ID              int64   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	BookID          int64   `gorm:"column:book_id;not null;index:idx_chapter_drafts_book_status,priority:1" json:"bookId"`
	ChapterID       int64   `gorm:"column:chapter_id;not null;index" json:"chapterId"`
	ChapterNo       int     `gorm:"column:chapter_no;not null" json:"chapterNo"`
	VersionNo       int     `gorm:"column:version_no;not null" json:"versionNo"`
	BasedOnPlanID   *int64  `gorm:"column:based_on_plan_id" json:"basedOnPlanId"`
	BasedOnDraftID  *int64  `gorm:"column:based_on_draft_id" json:"basedOnDraftId"`
	BasedOnReviewID *int64  `gorm:"column:based_on_review_id" json:"basedOnReviewId"`
	Status          string  `gorm:"column:status;type:varchar(32);not null;default:'active';index:idx_chapter_drafts_book_status,priority:2" json:"status"`
	Content         string  `gorm:"column:content;type:longtext;not null" json:"content"`
	Summary         *string `gorm:"column:summary;type:text" json:"summary"`
	WordCount       *int    `gorm:"column:word_count" json:"wordCount"`
	Model           *string `gorm:"column:model;type:varchar(255)" json:"model"`
	Provider        *string `gorm:"column:provider;type:varchar(64)" json:"provider"`
	SourceType      string  `gorm:"column:source_type;type:varchar(32);not null" json:"sourceType"`
	CreatedAt       string  `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt       string  `gorm:"column:updated_at;not null" json:"updatedAt"`
}

func (ChapterDraft) TableName() string { return "chapter_drafts" }

type ChapterReview struct {
	ID                int64   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	BookID            int64   `gorm:"column:book_id;not null;index:idx_chapter_reviews_book_status,priority:1" json:"bookId"`
	ChapterID         int64   `gorm:"column:chapter_id;not null;index" json:"chapterId"`
	ChapterNo         int     `gorm:"column:chapter_no;not null" json:"chapterNo"`
	DraftID           int64   `gorm:"column:draft_id;not null" json:"draftId"`
	VersionNo         int     `gorm:"column:version_no;not null" json:"versionNo"`
	Status            string  `gorm:"column:status;type:varchar(32);not null;default:'active';index:idx_chapter_reviews_book_status,priority:2" json:"status"`
	Summary           *string `gorm:"column:summary;type:text" json:"summary"`
	Issues            *string `gorm:"column:issues;type:text" json:"issues"`
	Risks             *string `gorm:"column:risks;type:text" json:"risks"`
	ContinuityChecks  *string `gorm:"column:continuity_checks;type:text" json:"continuityChecks"`
	RepairSuggestions *string `gorm:"column:repair_suggestions;type:text" json:"repairSuggestions"`
	RawResult         string  `gorm:"column:raw_result;type:longtext;not null" json:"rawResult"`
	Model             *string `gorm:"column:model;type:varchar(255)" json:"model"`
	Provider          *string `gorm:"column:provider;type:varchar(64)" json:"provider"`
	SourceType        string  `gorm:"column:source_type;type:varchar(32);not null" json:"sourceType"`
	CreatedAt         string  `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt         string  `gorm:"column:updated_at;not null" json:"updatedAt"`
}

func (ChapterReview) TableName() string { return "chapter_reviews" }

type ChapterFinal struct {
	ID             int64   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	BookID         int64   `gorm:"column:book_id;not null;index:idx_chapter_finals_book_status,priority:1" json:"bookId"`
	ChapterID      int64   `gorm:"column:chapter_id;not null;index" json:"chapterId"`
	ChapterNo      int     `gorm:"column:chapter_no;not null" json:"chapterNo"`
	VersionNo      int     `gorm:"column:version_no;not null" json:"versionNo"`
	BasedOnDraftID *int64  `gorm:"column:based_on_draft_id" json:"basedOnDraftId"`
	Status         string  `gorm:"column:status;type:varchar(32);not null;default:'active';index:idx_chapter_finals_book_status,priority:2" json:"status"`
	Content        string  `gorm:"column:content;type:longtext;not null" json:"content"`
	Summary        *string `gorm:"column:summary;type:text" json:"summary"`
	WordCount      *int    `gorm:"column:word_count" json:"wordCount"`
	SourceType     string  `gorm:"column:source_type;type:varchar(32);not null" json:"sourceType"`
	CreatedAt      string  `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt      string  `gorm:"column:updated_at;not null" json:"updatedAt"`
}

func (ChapterFinal) TableName() string { return "chapter_finals" }

type RetrievalDocument struct {
	ID                 int64   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	BookID             int64   `gorm:"column:book_id;not null;index:idx_retrieval_documents_book_layer,priority:1;index:idx_retrieval_documents_book_entity,priority:1;index:idx_retrieval_documents_book_chapter,priority:1" json:"bookId"`
	EntityType         *string `gorm:"column:entity_type;type:varchar(32);index:idx_retrieval_documents_book_entity,priority:2" json:"entityType"`
	EntityID           *int64  `gorm:"column:entity_id;index:idx_retrieval_documents_book_entity,priority:3" json:"entityId"`
	Layer              string  `gorm:"column:layer;type:varchar(32);not null;index:idx_retrieval_documents_book_layer,priority:2" json:"layer"`
	ChunkKey           string  `gorm:"column:chunk_key;type:varchar(255);not null" json:"chunkKey"`
	ChapterNo          *int    `gorm:"column:chapter_no;index:idx_retrieval_documents_book_chapter,priority:2" json:"chapterNo"`
	PayloadJSON        *string `gorm:"column:payload_json;type:longtext" json:"payloadJson"`
	Text               string  `gorm:"column:text;type:longtext;not null" json:"text"`
	EmbeddingModel     *string `gorm:"column:embedding_model;type:varchar(128)" json:"embeddingModel"`
	EmbeddingVectorRef *string `gorm:"column:embedding_vector_ref;type:varchar(255)" json:"embeddingVectorRef"`
	Status             string  `gorm:"column:status;type:varchar(32);not null;default:'active'" json:"status"`
	CreatedAt          string  `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt          string  `gorm:"column:updated_at;not null" json:"updatedAt"`
}

func (RetrievalDocument) TableName() string { return "retrieval_documents" }

type RetrievalFact struct {
	ID                     int64   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	BookID                 int64   `gorm:"column:book_id;not null;index:idx_retrieval_facts_book_fact_type,priority:1;index:idx_retrieval_facts_book_entity,priority:1" json:"bookId"`
	ChapterNo              *int    `gorm:"column:chapter_no" json:"chapterNo"`
	EntityType             *string `gorm:"column:entity_type;type:varchar(32);index:idx_retrieval_facts_book_entity,priority:2" json:"entityType"`
	EntityID               *int64  `gorm:"column:entity_id;index:idx_retrieval_facts_book_entity,priority:3" json:"entityId"`
	EventID                *int64  `gorm:"column:event_id" json:"eventId"`
	FactType               string  `gorm:"column:fact_type;type:varchar(32);not null;index:idx_retrieval_facts_book_fact_type,priority:2" json:"factType"`
	FactKey                string  `gorm:"column:fact_key;type:varchar(255);not null" json:"factKey"`
	FactText               string  `gorm:"column:fact_text;type:longtext;not null" json:"factText"`
	PayloadJSON            *string `gorm:"column:payload_json;type:longtext" json:"payloadJson"`
	Importance             *int    `gorm:"column:importance" json:"importance"`
	RiskLevel              *int    `gorm:"column:risk_level" json:"riskLevel"`
	EffectiveFromChapterNo *int    `gorm:"column:effective_from_chapter_no" json:"effectiveFromChapterNo"`
	EffectiveToChapterNo   *int    `gorm:"column:effective_to_chapter_no" json:"effectiveToChapterNo"`
	SupersededByFactID     *int64  `gorm:"column:superseded_by_fact_id" json:"supersededByFactId"`
	Status                 string  `gorm:"column:status;type:varchar(32);not null;default:'active'" json:"status"`
	CreatedAt              string  `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt              string  `gorm:"column:updated_at;not null" json:"updatedAt"`
}

func (RetrievalFact) TableName() string { return "retrieval_facts" }

type StoryEvent struct {
	ID                    int64   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	BookID                int64   `gorm:"column:book_id;not null;index:idx_story_events_book_chapter,priority:1;index:idx_story_events_book_status,priority:1" json:"bookId"`
	ChapterID             *int64  `gorm:"column:chapter_id" json:"chapterId"`
	ChapterNo             *int    `gorm:"column:chapter_no;index:idx_story_events_book_chapter,priority:2" json:"chapterNo"`
	EventType             string  `gorm:"column:event_type;type:varchar(32);not null" json:"eventType"`
	Title                 string  `gorm:"column:title;type:varchar(255);not null" json:"title"`
	Summary               string  `gorm:"column:summary;type:text;not null" json:"summary"`
	ParticipantEntityRefs *string `gorm:"column:participant_entity_refs;type:text" json:"participantEntityRefs"`
	LocationLabel         *string `gorm:"column:location_label;type:varchar(255)" json:"locationLabel"`
	TriggerText           *string `gorm:"column:trigger_text;type:text" json:"triggerText"`
	OutcomeText           *string `gorm:"column:outcome_text;type:text" json:"outcomeText"`
	UnresolvedImpact      *string `gorm:"column:unresolved_impact;type:text" json:"unresolvedImpact"`
	HookRefs              *string `gorm:"column:hook_refs;type:text" json:"hookRefs"`
	Status                string  `gorm:"column:status;type:varchar(32);not null;default:'active';index:idx_story_events_book_status,priority:2" json:"status"`
	CreatedAt             string  `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt             string  `gorm:"column:updated_at;not null" json:"updatedAt"`
}

func (StoryEvent) TableName() string { return "story_events" }

type ChapterSegment struct {
	ID           int64   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	BookID       int64   `gorm:"column:book_id;not null;index:idx_chapter_segments_book_chapter,priority:1" json:"bookId"`
	ChapterID    int64   `gorm:"column:chapter_id;not null" json:"chapterId"`
	ChapterNo    int     `gorm:"column:chapter_no;not null;index:idx_chapter_segments_book_chapter,priority:2" json:"chapterNo"`
	SegmentIndex int     `gorm:"column:segment_index;not null" json:"segmentIndex"`
	SourceType   string  `gorm:"column:source_type;type:varchar(32);not null" json:"sourceType"`
	Text         string  `gorm:"column:text;type:longtext;not null" json:"text"`
	Summary      *string `gorm:"column:summary;type:text" json:"summary"`
	EventRefs    *string `gorm:"column:event_refs;type:text" json:"eventRefs"`
	Metadata     *string `gorm:"column:metadata;type:text" json:"metadata"`
	Status       string  `gorm:"column:status;type:varchar(32);not null;default:'active'" json:"status"`
	CreatedAt    string  `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt    string  `gorm:"column:updated_at;not null" json:"updatedAt"`
}

func (ChapterSegment) TableName() string { return "chapter_segments" }

type WorkflowTask struct {
	ID              int64   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	BookID          int64   `gorm:"column:book_id;not null;index:idx_workflow_tasks_book_chapter_type,priority:1;index:idx_workflow_tasks_book_chapter_created,priority:1" json:"bookId"`
	ChapterID       int64   `gorm:"column:chapter_id;not null" json:"chapterId"`
	ChapterNo       int     `gorm:"column:chapter_no;not null;index:idx_workflow_tasks_book_chapter_type,priority:2;index:idx_workflow_tasks_book_chapter_created,priority:2" json:"chapterNo"`
	WorkflowType    string  `gorm:"column:workflow_type;type:varchar(32);not null;index:idx_workflow_tasks_book_chapter_type,priority:3" json:"workflowType"`
	Status          string  `gorm:"column:status;type:varchar(32);not null;index:idx_workflow_tasks_status_updated,priority:1;index:idx_workflow_tasks_status_scheduled,priority:1" json:"status"`
	Stage           *string `gorm:"column:stage;type:varchar(64)" json:"stage"`
	ScheduledAt     string  `gorm:"column:scheduled_at;type:varchar(32);not null;default:'';index:idx_workflow_tasks_status_scheduled,priority:2" json:"scheduledAt"`
	LeaseOwner      *string `gorm:"column:lease_owner;type:varchar(255)" json:"leaseOwner"`
	LeaseToken      *string `gorm:"column:lease_token;type:varchar(255)" json:"leaseToken"`
	LeaseExpiresAt  *string `gorm:"column:lease_expires_at" json:"leaseExpiresAt"`
	RequestPayload  string  `gorm:"column:request_payload;type:longtext;not null" json:"requestPayload"`
	ResultPayload   *string `gorm:"column:result_payload;type:longtext" json:"resultPayload"`
	CurrentPlanID   *int64  `gorm:"column:current_plan_id" json:"currentPlanId"`
	CurrentDraftID  *int64  `gorm:"column:current_draft_id" json:"currentDraftId"`
	ErrorCode       *string `gorm:"column:error_code;type:varchar(64)" json:"errorCode"`
	ErrorMessage    *string `gorm:"column:error_message;type:varchar(512)" json:"errorMessage"`
	ErrorDetails    *string `gorm:"column:error_details;type:longtext" json:"errorDetails"`
	ProgressPercent *int    `gorm:"column:progress_percent" json:"progressPercent"`
	AttemptCount    int     `gorm:"column:attempt_count;not null;default:1" json:"attemptCount"`
	StartedAt       *string `gorm:"column:started_at" json:"startedAt"`
	FinishedAt      *string `gorm:"column:finished_at" json:"finishedAt"`
	CreatedAt       string  `gorm:"column:created_at;not null;index:idx_workflow_tasks_book_chapter_created,priority:3" json:"createdAt"`
	UpdatedAt       string  `gorm:"column:updated_at;not null;index:idx_workflow_tasks_status_updated,priority:2" json:"updatedAt"`
}

func (WorkflowTask) TableName() string { return "workflow_tasks" }

type User struct {
	ID           int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Email        string `gorm:"column:email;type:varchar(255);not null;uniqueIndex:idx_users_email" json:"email"`
	PasswordHash string `gorm:"column:password_hash;type:varchar(255);not null" json:"-"`
	DisplayName  string `gorm:"column:display_name;type:varchar(255);not null" json:"displayName"`
	Status       string `gorm:"column:status;type:varchar(32);not null;default:'active'" json:"status"`
	CreatedAt    string `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt    string `gorm:"column:updated_at;not null" json:"updatedAt"`
}

func (User) TableName() string { return "users" }

type UserSession struct {
	ID               int64   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID           int64   `gorm:"column:user_id;not null;index:idx_user_sessions_user_id" json:"userId"`
	SessionTokenHash string  `gorm:"column:session_token_hash;type:varchar(128);not null;uniqueIndex:idx_user_sessions_token_hash" json:"-"`
	ExpiresAt        string  `gorm:"column:expires_at;not null" json:"expiresAt"`
	CreatedAt        string  `gorm:"column:created_at;not null" json:"createdAt"`
	LastSeenAt       *string `gorm:"column:last_seen_at" json:"lastSeenAt"`
}

func (UserSession) TableName() string { return "user_sessions" }

type UserSetting struct {
	ID               int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID           int64  `gorm:"column:user_id;not null;uniqueIndex:idx_user_settings_user_id" json:"userId"`
	RuntimeOverrides string `gorm:"column:runtime_overrides;type:longtext;not null" json:"runtimeOverrides"`
	CreatedAt        string `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt        string `gorm:"column:updated_at;not null" json:"updatedAt"`
}

func (UserSetting) TableName() string { return "user_settings" }

func All() []interface{} {
	return []interface{}{
		&User{}, &UserSession{}, &UserSetting{},
		&Book{}, &Outline{}, &WorldSetting{}, &Character{}, &Faction{}, &Relation{},
		&Item{}, &StoryHook{}, &Chapter{}, &ChapterPlan{}, &ChapterDraft{},
		&ChapterReview{}, &ChapterFinal{}, &RetrievalDocument{}, &RetrievalFact{},
		&StoryEvent{}, &ChapterSegment{}, &WorkflowTask{},
	}
}
