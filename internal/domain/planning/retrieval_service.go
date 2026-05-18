package planning

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"myai-novel-go/internal/config"
	"myai-novel-go/internal/db/models"
	"myai-novel-go/internal/domain/shared"
	"myai-novel-go/internal/llm"
)

// RetrievalService 是检索流水线的入口,
// 内部按 cfg 装配:base candidate provider(规则) → 可选嵌入候选层 → reranker(启发式或透传)。
type RetrievalService struct {
	db        *gorm.DB
	cfg       *config.Config
	provider  CandidateProvider
	reranker  Reranker
	embedding llm.EmbeddingClient // 持有以便外部触发刷新;nil 表示禁用
	store     EmbeddingStore
}

func NewRetrievalService(db *gorm.DB, cfg *config.Config) *RetrievalService {
	return NewRetrievalServiceWithEmbedding(db, cfg, nil)
}

func NewRetrievalServiceWithEmbedding(db *gorm.DB, cfg *config.Config, embedding llm.EmbeddingClient) *RetrievalService {
	base := NewRuleCandidateProvider(cfg)
	var provider CandidateProvider = base
	var store EmbeddingStore
	if embedding != nil && embedding.ProviderName() != llm.EmbeddingProviderNone {
		store = NewDBEmbeddingStore(db)
		var searcher EmbeddingSearcher
		if cfg.PlanningRetrievalEmbeddingSearchMode == "hybrid" {
			searcher = NewHybridEmbeddingSearcher(embedding, store)
		} else {
			searcher = NewBasicEmbeddingSearcher(embedding, store)
		}
		limit := cfg.PlanningRetrievalEmbeddingLimitBasic
		if cfg.PlanningRetrievalEmbeddingSearchMode == "hybrid" {
			limit = cfg.PlanningRetrievalEmbeddingLimitHybrid
		}
		provider = NewEmbeddingCandidateProvider(base, searcher, embedding.Model(), limit, cfg.PlanningRetrievalEmbeddingMinScore)
	}
	var reranker Reranker = PassthroughReranker{}
	if cfg.PlanningRetrievalReranker == "heuristic" {
		reranker = NewHeuristicReranker()
	}
	return &RetrievalService{db: db, cfg: cfg, provider: provider, reranker: reranker, embedding: embedding, store: store}
}

// EmbeddingClient 暴露给上层(refresh 端点 / CLI)调用。
func (s *RetrievalService) EmbeddingClient() llm.EmbeddingClient { return s.embedding }

// EmbeddingStore 同上,暴露给 RefreshService 等。
func (s *RetrievalService) EmbeddingStore() EmbeddingStore { return s.store }

type RetrieveParams struct {
	BookID     int64
	ChapterNo  int
	Keywords   []string
	QueryText  string
	ManualRefs ManualEntityRefs
}

// Retrieve 等价原 retrievePlanContext:加载 book → 候选召回 → 重排 → 拼上下文。
func (s *RetrievalService) Retrieve(ctx context.Context, params RetrieveParams) (*RetrievedContext, error) {
	var book models.Book
	if err := s.db.WithContext(ctx).First(&book, params.BookID).Error; err != nil {
		return nil, shared.NotFound(fmt.Sprintf("book not found: %d", params.BookID))
	}

	bundle, err := s.provider.LoadCandidates(ctx, s.db, params)
	if err != nil {
		return nil, err
	}
	bundle, err = s.reranker.Rerank(ctx, params, bundle)
	if err != nil {
		return nil, err
	}

	out := &RetrievedContext{}
	out.Book.ID = book.ID
	out.Book.Title = book.Title
	out.Book.Summary = book.Summary
	out.Book.TargetChapterCount = book.TargetChapterCount
	out.Book.CurrentChapterCount = book.CurrentChapterCount
	out.Outlines = bundle.Outlines
	out.RecentChapters = bundle.RecentChapters
	out.RiskReminders = []RiskReminder{}
	out.Hooks = bundle.EntityGroups.Hooks
	out.Characters = bundle.EntityGroups.Characters
	out.Factions = bundle.EntityGroups.Factions
	out.Items = bundle.EntityGroups.Items
	out.Relations = bundle.EntityGroups.Relations
	out.WorldSettings = bundle.EntityGroups.WorldSettings
	out.SoftReferences.Outlines = bundle.Outlines
	out.SoftReferences.RecentChapters = bundle.RecentChapters
	out.SoftReferences.Entities = bundle.EntityGroups
	out.HardConstraints = pickManualHardConstraints(bundle.EntityGroups, params.ManualRefs)
	return out, nil
}

func pickManualHardConstraints(soft EntityGroups, refs ManualEntityRefs) EntityGroups {
	out := EntityGroups{Hooks: []RetrievedEntity{}, Characters: []RetrievedEntity{}, Factions: []RetrievedEntity{}, Items: []RetrievedEntity{}, Relations: []RetrievedEntity{}, WorldSettings: []RetrievedEntity{}}
	out.Characters = pickByIDs(soft.Characters, refs.CharacterIDs)
	out.Factions = pickByIDs(soft.Factions, refs.FactionIDs)
	out.Items = pickByIDs(soft.Items, refs.ItemIDs)
	out.Hooks = pickByIDs(soft.Hooks, refs.HookIDs)
	out.Relations = pickByIDs(soft.Relations, refs.RelationIDs)
	out.WorldSettings = pickByIDs(soft.WorldSettings, refs.WorldSettingIDs)
	return out
}

func pickByIDs(in []RetrievedEntity, ids []int64) []RetrievedEntity {
	idSet := make(map[int64]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}
	out := []RetrievedEntity{}
	for _, e := range in {
		if idSet[e.ID] {
			out = append(out, e)
		}
	}
	return out
}
