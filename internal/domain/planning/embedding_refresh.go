package planning

import (
	"context"

	"gorm.io/gorm"

	"myai-novel-go/internal/llm"
)

// RefreshService 把 book 下所有(或指定类型)实体重新生成嵌入文档,
// 等价原 EmbeddingRefreshService。每个实体类型独立写库,以保证替换语义不会跨类型冲掉无关索引。
type RefreshService struct {
	db     *gorm.DB
	client llm.EmbeddingClient
	store  EmbeddingStore
}

func NewRefreshService(db *gorm.DB, client llm.EmbeddingClient, store EmbeddingStore) *RefreshService {
	return &RefreshService{db: db, client: client, store: store}
}

// Refresh 全量重建(按实体类型分批写入)。返回 map: entityType → 写入文档数。
func (s *RefreshService) Refresh(ctx context.Context, bookID int64) (map[string]int, error) {
	docs, err := BuildEmbeddingDocumentsForBook(ctx, s.db, bookID)
	if err != nil {
		return nil, err
	}
	groups := groupByEntityType(docs)
	out := make(map[string]int, len(groups))
	for entityType, group := range groups {
		texts := make([]string, len(group))
		for i, d := range group {
			texts[i] = d.Text
		}
		vecs, err := s.client.Embed(ctx, texts)
		if err != nil {
			return out, err
		}
		indexed := make([]IndexedEmbeddingDocument, len(group))
		for i, d := range group {
			var v []float32
			if i < len(vecs) {
				v = vecs[i]
			}
			indexed[i] = IndexedEmbeddingDocument{
				EmbeddingDocument: d,
				Model:             s.client.Model(),
				Vector:            v,
			}
		}
		if err := s.store.Replace(ctx, bookID, s.client.Model(), entityType, indexed); err != nil {
			return out, err
		}
		out[entityType] = len(indexed)
	}
	return out, nil
}

// Clear 清空指定 model 下整本书的嵌入文档。
func (s *RefreshService) Clear(ctx context.Context, bookID int64) error {
	return s.store.Clear(ctx, bookID, s.client.Model(), "")
}

func groupByEntityType(docs []EmbeddingDocument) map[string][]EmbeddingDocument {
	out := make(map[string][]EmbeddingDocument, 6)
	for _, d := range docs {
		out[d.EntityType] = append(out[d.EntityType], d)
	}
	return out
}
