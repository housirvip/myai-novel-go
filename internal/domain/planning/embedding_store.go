package planning

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	"myai-novel-go/internal/db/models"
	"myai-novel-go/internal/domain/shared"
)

// EmbeddingDocument 描述待写入索引的文档(尚未带向量)。
type EmbeddingDocument struct {
	BookID      int64
	EntityType  string // character / faction / item / hook / world_setting / relation
	EntityID    int64
	ChunkKey    string
	DisplayName string
	Text        string
	// Relation 用的额外元数据
	RelationEndpoints map[string]any `json:",omitempty"`
	RelationMetadata  map[string]any `json:",omitempty"`
}

// IndexedEmbeddingDocument 是已经带向量的文档,
// 写入 retrieval_documents.payload_json 时序列化 vector + displayName + relation*。
type IndexedEmbeddingDocument struct {
	EmbeddingDocument
	Model  string
	Vector []float32
}

// EmbeddingPayload 是 payload_json 的结构。
type EmbeddingPayload struct {
	Vector            []float32      `json:"vector"`
	DisplayName       string         `json:"displayName,omitempty"`
	RelationEndpoints map[string]any `json:"relationEndpoints,omitempty"`
	RelationMetadata  map[string]any `json:"relationMetadata,omitempty"`
}

type EmbeddingStore interface {
	Replace(ctx context.Context, bookID int64, model, entityType string, docs []IndexedEmbeddingDocument) error
	List(ctx context.Context, bookID int64, model, entityType string) ([]IndexedEmbeddingDocument, error)
	Clear(ctx context.Context, bookID int64, model, entityType string) error
}

type DBEmbeddingStore struct {
	db *gorm.DB
}

func NewDBEmbeddingStore(db *gorm.DB) *DBEmbeddingStore {
	return &DBEmbeddingStore{db: db}
}

func (s *DBEmbeddingStore) Replace(ctx context.Context, bookID int64, model, entityType string, docs []IndexedEmbeddingDocument) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.deleteScoped(tx, bookID, model, entityType); err != nil {
			return err
		}
		now := shared.NowISO()
		rows := make([]models.RetrievalDocument, 0, len(docs))
		for _, d := range docs {
			payloadJSON, err := json.Marshal(EmbeddingPayload{
				Vector:            d.Vector,
				DisplayName:       d.DisplayName,
				RelationEndpoints: d.RelationEndpoints,
				RelationMetadata:  d.RelationMetadata,
			})
			if err != nil {
				return err
			}
			payloadStr := string(payloadJSON)
			et := d.EntityType
			eid := d.EntityID
			modelCopy := d.Model
			rows = append(rows, models.RetrievalDocument{
				BookID:         bookID,
				EntityType:     &et,
				EntityID:       &eid,
				Layer:          "embedding",
				ChunkKey:       d.ChunkKey,
				PayloadJSON:    &payloadStr,
				Text:           d.Text,
				EmbeddingModel: &modelCopy,
				Status:         "active",
				CreatedAt:      now,
				UpdatedAt:      now,
			})
		}
		if len(rows) == 0 {
			return nil
		}
		return tx.Create(&rows).Error
	})
}

func (s *DBEmbeddingStore) List(ctx context.Context, bookID int64, model, entityType string) ([]IndexedEmbeddingDocument, error) {
	q := s.db.WithContext(ctx).Where("book_id = ? AND layer = ? AND status = ?", bookID, "embedding", "active")
	if model != "" {
		q = q.Where("embedding_model = ?", model)
	}
	if entityType != "" {
		q = q.Where("entity_type = ?", entityType)
	}
	var rows []models.RetrievalDocument
	if err := q.Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]IndexedEmbeddingDocument, 0, len(rows))
	for _, r := range rows {
		if r.EntityType == nil || r.EntityID == nil || r.PayloadJSON == nil || r.EmbeddingModel == nil {
			continue
		}
		var payload EmbeddingPayload
		if err := json.Unmarshal([]byte(*r.PayloadJSON), &payload); err != nil {
			continue
		}
		if len(payload.Vector) == 0 {
			continue
		}
		display := payload.DisplayName
		if display == "" {
			display = r.ChunkKey
		}
		out = append(out, IndexedEmbeddingDocument{
			EmbeddingDocument: EmbeddingDocument{
				BookID:            r.BookID,
				EntityType:        *r.EntityType,
				EntityID:          *r.EntityID,
				ChunkKey:          r.ChunkKey,
				DisplayName:       display,
				Text:              r.Text,
				RelationEndpoints: payload.RelationEndpoints,
				RelationMetadata:  payload.RelationMetadata,
			},
			Model:  *r.EmbeddingModel,
			Vector: payload.Vector,
		})
	}
	return out, nil
}

func (s *DBEmbeddingStore) Clear(ctx context.Context, bookID int64, model, entityType string) error {
	return s.deleteScoped(s.db.WithContext(ctx), bookID, model, entityType)
}

func (s *DBEmbeddingStore) deleteScoped(tx *gorm.DB, bookID int64, model, entityType string) error {
	q := tx.Where("book_id = ? AND layer = ?", bookID, "embedding")
	if model != "" {
		q = q.Where("embedding_model = ?", model)
	}
	if entityType != "" {
		q = q.Where("entity_type = ?", entityType)
	}
	return q.Delete(&models.RetrievalDocument{}).Error
}

func formatRelationContent(rel map[string]any) string {
	b, err := json.Marshal(rel)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("relation_metadata=%s", string(b))
}
