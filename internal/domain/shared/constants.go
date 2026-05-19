package shared

const (
	ChapterStatusTodo     = "todo"
	ChapterStatusPlanned  = "planned"
	ChapterStatusDrafted  = "drafted"
	ChapterStatusReviewed = "reviewed"
	ChapterStatusRepaired = "repaired"
	ChapterStatusApproved = "approved"
)

const (
	ChapterSourceAIGenerated = "ai_generated"
	ChapterSourceRepaired    = "repaired"
	ChapterSourceImported    = "imported"
	ChapterSourceApproved    = "approved"
)

const (
	PlanIntentSourceUserInput   = "user_input"
	PlanIntentSourceAIGenerated = "ai_generated"
	PlanIntentSourceManual      = "manual_import"
)

const (
	WorkflowTaskTypePlan         = "plan"
	WorkflowTaskTypeDraft        = "draft"
	WorkflowTaskTypeReview       = "review"
	WorkflowTaskTypeRepair       = "repair"
	WorkflowTaskTypeApprove      = "approve"
	WorkflowTaskTypeAuthorIntent = "author_intent"
)

const (
	WorkflowTaskStatusPending   = "pending"
	WorkflowTaskStatusClaimed   = "claimed"
	WorkflowTaskStatusRunning   = "running"
	WorkflowTaskStatusSucceeded = "succeeded"
	WorkflowTaskStatusFailed    = "failed"
)

const (
	WorkflowStageQueued                 = "queued"
	WorkflowStageLoadingChapter         = "loading_chapter"
	WorkflowStageRetrievingInitial      = "retrieving_initial_context"
	WorkflowStageGeneratingAuthorIntent = "generating_author_intent"
	WorkflowStageExtractingKeywords     = "extracting_intent_keywords"
	WorkflowStageRetrievingFinal        = "retrieving_final_context"
	WorkflowStageGeneratingPlan         = "generating_plan"
	WorkflowStageLoadingPlanContext     = "loading_plan_context"
	WorkflowStageGeneratingDraft        = "generating_draft"
	WorkflowStageGeneratingReview       = "generating_review"
	WorkflowStageLoadingReviewContext   = "loading_review_context"
	WorkflowStageGeneratingRepair       = "generating_repair"
	WorkflowStageGeneratingFinal        = "generating_final"
	WorkflowStageExtractingDiff         = "extracting_diff"
	WorkflowStageUpdatingResources      = "updating_resources"
	WorkflowStagePersistingSidecar      = "persisting_sidecar_artifacts"
	WorkflowStageRepairingLength        = "repairing_length"
	WorkflowStageSavingArtifacts        = "saving_artifacts"
)

const StageNamePlan = "plan"
const StageNameDraft = "draft"
const StageNameReview = "review"
const StageNameFinal = "final"
