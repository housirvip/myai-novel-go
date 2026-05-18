package world_setting

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
	BookID      int64   `json:"-"`
	Title       string  `json:"title" binding:"required,min=1"`
	Category    string  `json:"category" binding:"required,min=1"`
	Content     string  `json:"content" binding:"required,min=1"`
	Status      *string `json:"status"`
	AppendNotes *string `json:"appendNotes"`
	Keywords    *string `json:"keywords"`
}

type UpdateInput struct {
	Title       *string `json:"title"`
	Category    *string `json:"category"`
	Content     *string `json:"content"`
	Status      *string `json:"status"`
	AppendNotes *string `json:"appendNotes"`
	Keywords    *string `json:"keywords"`
}

func (s *Service) List(ctx context.Context, bookID int64, limit int, status string) ([]models.WorldSetting, error) {
	q := s.db.WithContext(ctx).Where("book_id = ?", bookID).Order("id DESC").Limit(limit)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var rows []models.WorldSetting
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Service) Get(ctx context.Context, bookID, id int64) (*models.WorldSetting, error) {
	var row models.WorldSetting
	if err := s.db.WithContext(ctx).Where("book_id = ? AND id = ?", bookID, id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.NotFound("world setting not found")
		}
		return nil, err
	}
	return &row, nil
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*models.WorldSetting, error) {
	now := shared.NowISO()
	row := models.WorldSetting{
		BookID: in.BookID, Title: in.Title, Category: in.Category, Content: in.Content,
		Status: "active", AppendNotes: in.AppendNotes, Keywords: in.Keywords,
		CreatedAt: now, UpdatedAt: now,
	}
	if in.Status != nil && *in.Status != "" {
		row.Status = *in.Status
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) Update(ctx context.Context, bookID, id int64, in UpdateInput) (*models.WorldSetting, error) {
	row, err := s.Get(ctx, bookID, id)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{"updated_at": shared.NowISO()}
	if in.Title != nil {
		updates["title"] = *in.Title
	}
	if in.Category != nil {
		updates["category"] = *in.Category
	}
	if in.Content != nil {
		updates["content"] = *in.Content
	}
	if in.Status != nil {
		updates["status"] = *in.Status
	}
	if in.AppendNotes != nil {
		updates["append_notes"] = in.AppendNotes
	}
	if in.Keywords != nil {
		updates["keywords"] = in.Keywords
	}
	if err := s.db.WithContext(ctx).Model(&models.WorldSetting{}).Where("id = ?", row.ID).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.Get(ctx, bookID, id)
}

func (s *Service) Remove(ctx context.Context, bookID, id int64) error {
	res := s.db.WithContext(ctx).Where("book_id = ? AND id = ?", bookID, id).Delete(&models.WorldSetting{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return shared.NotFound("world setting not found")
	}
	return nil
}
