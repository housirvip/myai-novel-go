package relation

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"myai-novel-go/internal/db/models"
	"myai-novel-go/internal/domain/shared"
)

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

type CreateInput struct {
	BookID       int64   `json:"-"`
	SourceType   string  `json:"sourceType" binding:"required,oneof=character faction"`
	SourceID     int64   `json:"sourceId" binding:"required"`
	TargetType   string  `json:"targetType" binding:"required,oneof=character faction"`
	TargetID     int64   `json:"targetId" binding:"required"`
	RelationType string  `json:"relationType" binding:"required,min=1"`
	Intensity    *int    `json:"intensity"`
	Status       *string `json:"status"`
	Description  *string `json:"description"`
	AppendNotes  *string `json:"appendNotes"`
	Keywords     *string `json:"keywords"`
}

type UpdateInput struct {
	RelationType *string `json:"relationType"`
	Intensity    *int    `json:"intensity"`
	Status       *string `json:"status"`
	Description  *string `json:"description"`
	AppendNotes  *string `json:"appendNotes"`
	Keywords     *string `json:"keywords"`
}

func (s *Service) List(ctx context.Context, bookID int64, limit int) ([]models.Relation, error) {
	var rows []models.Relation
	if err := s.db.WithContext(ctx).Where("book_id = ?", bookID).Order("id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Service) Get(ctx context.Context, bookID, id int64) (*models.Relation, error) {
	var row models.Relation
	if err := s.db.WithContext(ctx).Where("book_id = ? AND id = ?", bookID, id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.NotFound("relation not found")
		}
		return nil, err
	}
	return &row, nil
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*models.Relation, error) {
	if err := s.validateEndpoint(ctx, in.BookID, in.SourceType, in.SourceID, "source"); err != nil {
		return nil, err
	}
	if err := s.validateEndpoint(ctx, in.BookID, in.TargetType, in.TargetID, "target"); err != nil {
		return nil, err
	}
	now := shared.NowISO()
	row := models.Relation{
		BookID: in.BookID, SourceType: in.SourceType, SourceID: in.SourceID,
		TargetType: in.TargetType, TargetID: in.TargetID, RelationType: in.RelationType,
		Intensity: in.Intensity, Status: in.Status, Description: in.Description,
		AppendNotes: in.AppendNotes, Keywords: in.Keywords,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) validateEndpoint(ctx context.Context, bookID int64, kind string, id int64, label string) error {
	switch kind {
	case "character":
		var n int64
		s.db.WithContext(ctx).Model(&models.Character{}).Where("book_id = ? AND id = ?", bookID, id).Count(&n)
		if n == 0 {
			return shared.BadRequest(label + " character not found in book")
		}
	case "faction":
		var n int64
		s.db.WithContext(ctx).Model(&models.Faction{}).Where("book_id = ? AND id = ?", bookID, id).Count(&n)
		if n == 0 {
			return shared.BadRequest(label + " faction not found in book")
		}
	default:
		return shared.BadRequest("unsupported endpoint type: " + kind)
	}
	return nil
}

func (s *Service) Update(ctx context.Context, bookID, id int64, in UpdateInput) (*models.Relation, error) {
	row, err := s.Get(ctx, bookID, id)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{"updated_at": shared.NowISO()}
	if in.RelationType != nil {
		updates["relation_type"] = *in.RelationType
	}
	if in.Intensity != nil {
		updates["intensity"] = in.Intensity
	}
	if in.Status != nil {
		updates["status"] = in.Status
	}
	if in.Description != nil {
		updates["description"] = in.Description
	}
	if in.AppendNotes != nil {
		updates["append_notes"] = in.AppendNotes
	}
	if in.Keywords != nil {
		updates["keywords"] = in.Keywords
	}
	if err := s.db.WithContext(ctx).Model(&models.Relation{}).Where("id = ?", row.ID).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.Get(ctx, bookID, id)
}

func (s *Service) Remove(ctx context.Context, bookID, id int64) error {
	res := s.db.WithContext(ctx).Where("book_id = ? AND id = ?", bookID, id).Delete(&models.Relation{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return shared.NotFound("relation not found")
	}
	return nil
}
