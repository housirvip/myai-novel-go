package planning

import (
	"context"

	"gorm.io/gorm"
)

// CandidateBundle 是检索流水线第一阶段(候选召回)的产物,
// 由 CandidateProvider.LoadCandidates 产出,再交给 Reranker.Rerank。
// 字段含义与原项目 retrieval-pipeline.ts 中的 RetrievalCandidateBundle 对齐。
type CandidateBundle struct {
	Outlines       []RetrievedOutline
	RecentChapters []RetrievedChapterSummary
	EntityGroups   EntityGroups
}

// CandidateProvider 是规则候选 / 嵌入候选 / 混合候选的统一接口。
type CandidateProvider interface {
	LoadCandidates(ctx context.Context, db *gorm.DB, params RetrieveParams) (*CandidateBundle, error)
}

// Reranker 是候选打分重排接口。
// 实现可以是启发式打分(HeuristicReranker),也可以是 LLM rerank 等。
type Reranker interface {
	Rerank(ctx context.Context, params RetrieveParams, in *CandidateBundle) (*CandidateBundle, error)
}

// PassthroughReranker 在 cfg.PlanningRetrievalReranker = "none" 时使用。
type PassthroughReranker struct{}

func (PassthroughReranker) Rerank(_ context.Context, _ RetrieveParams, in *CandidateBundle) (*CandidateBundle, error) {
	return in, nil
}
