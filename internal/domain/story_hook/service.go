package story_hook

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
	BookID          int64   `json:"-"`
	Title           string  `json:"title" binding:"required,min=1"`
	HookType        *string `json:"hookType"`
	Description     *string `json:"description"`
	SourceChapterNo *int    `json:"sourceChapterNo"`
	TargetChapterNo *int    `json:"targetChapterNo"`
	Status          *string `json:"status"`
	Importance      *string `json:"importance"`
	AppendNotes     *string `json:"appendNotes"`
	Keywords        *string `json:"keywords"`
}

type UpdateInput struct {
	Title           *string `json:"title"`
	HookType        *string `json:"hookType"`
	Description     *string `json:"description"`
	SourceChapterNo *int    `json:"sourceChapterNo"`
	TargetChapterNo *int    `json:"targetChapterNo"`
	Status          *string `json:"status"`
	Importance      *string `json:"importance"`
	AppendNotes     *string `json:"appendNotes"`
	Keywords        *string `json:"keywords"`
}

func (s *Service) List(ctx context.Context, bookID int64, limit int, status string) ([]models.StoryHook, error) {
	q := s.db.WithContext(ctx).Where("book_id = ?", bookID).Order("id DESC").Limit(limit)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var rows []models.StoryHook
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Service) Get(ctx context.Context, bookID, id int64) (*models.StoryHook, error) {
	var row models.StoryHook
	if err := s.db.WithContext(ctx).Where("book_id = ? AND id = ?", bookID, id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.NotFound("hook not found")
		}
		return nil, err
	}
	return &row, nil
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*models.StoryHook, error) {
	now := shared.NowISO()
	row := models.StoryHook{
		BookID: in.BookID, Title: in.Title, HookType: in.HookType, Description: in.Description,
		SourceChapterNo: in.SourceChapterNo, TargetChapterNo: in.TargetChapterNo,
		Status: "open", Importance: in.Importance, AppendNotes: in.AppendNotes, Keywords: in.Keywords,
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

func (s *Service) Update(ctx context.Context, bookID, id int64, in UpdateInput) (*models.StoryHook, error) {
	row, err := s.Get(ctx, bookID, id)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{"updated_at": shared.NowISO()}
	if in.Title != nil {
		updates["title"] = *in.Title
	}
	if in.HookType != nil {
		updates["hook_type"] = in.HookType
	}
	if in.Description != nil {
		updates["description"] = in.Description
	}
	if in.SourceChapterNo != nil {
		updates["source_chapter_no"] = in.SourceChapterNo
	}
	if in.TargetChapterNo != nil {
		updates["target_chapter_no"] = in.TargetChapterNo
	}
	if in.Status != nil {
		updates["status"] = *in.Status
	}
	if in.Importance != nil {
		updates["importance"] = in.Importance
	}
	if in.AppendNotes != nil {
		updates["append_notes"] = in.AppendNotes
	}
	if in.Keywords != nil {
		updates["keywords"] = in.Keywords
	}
	if err := s.db.WithContext(ctx).Model(&models.StoryHook{}).Where("id = ?", row.ID).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.Get(ctx, bookID, id)
}

func (s *Service) Remove(ctx context.Context, bookID, id int64) error {
	res := s.db.WithContext(ctx).Where("book_id = ? AND id = ?", bookID, id).Delete(&models.StoryHook{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return shared.NotFound("hook not found")
	}
	return nil
}
