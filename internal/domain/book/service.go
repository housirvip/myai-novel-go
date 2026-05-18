package book

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"myai-novel-go/internal/db/models"
	"myai-novel-go/internal/domain/shared"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

type CreateInput struct {
	Title              string  `json:"title" binding:"required,min=1"`
	Summary            *string `json:"summary"`
	TargetChapterCount *int    `json:"targetChapterCount"`
	Status             *string `json:"status"`
	// OwnerUserID 由 handler 从 actor 注入,匿名时为 nil。
	OwnerUserID *int64 `json:"-"`
}

type UpdateInput struct {
	Title              *string `json:"title"`
	Summary            *string `json:"summary"`
	TargetChapterCount *int    `json:"targetChapterCount"`
	Status             *string `json:"status"`
}

// ListForActor 按 actor 视角列书:
//   - 匿名:owner_user_id IS NULL(系统级 / 引导期书)
//   - 用户:owner_user_id IS NULL OR = actorUserID
//
// 这样既保留了向后兼容(匿名仍能看引导期书),又把多租户边界落到 service 层。
func (s *Service) ListForActor(ctx context.Context, ownerUserID *int64, limit int) ([]models.Book, error) {
	q := s.db.WithContext(ctx).Order("id DESC").Limit(limit)
	if ownerUserID == nil {
		q = q.Where("owner_user_id IS NULL")
	} else {
		q = q.Where("owner_user_id IS NULL OR owner_user_id = ?", *ownerUserID)
	}
	var rows []models.Book
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// List 是兼容旧调用:不做 owner 过滤,返回全部。
func (s *Service) List(ctx context.Context, limit int) ([]models.Book, error) {
	var rows []models.Book
	if err := s.db.WithContext(ctx).Order("id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Service) Get(ctx context.Context, id int64) (*models.Book, error) {
	var row models.Book
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.NotFound("book not found")
		}
		return nil, err
	}
	return &row, nil
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*models.Book, error) {
	now := shared.NowISO()
	row := models.Book{
		Title:              in.Title,
		Summary:            in.Summary,
		TargetChapterCount: in.TargetChapterCount,
		OwnerUserID:        in.OwnerUserID,
		Status:             "planning",
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if in.Status != nil && *in.Status != "" {
		row.Status = *in.Status
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) Update(ctx context.Context, id int64, in UpdateInput) (*models.Book, error) {
	row, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{"updated_at": shared.NowISO()}
	if in.Title != nil {
		updates["title"] = *in.Title
	}
	if in.Summary != nil {
		updates["summary"] = in.Summary
	}
	if in.TargetChapterCount != nil {
		updates["target_chapter_count"] = in.TargetChapterCount
	}
	if in.Status != nil {
		updates["status"] = *in.Status
	}
	if err := s.db.WithContext(ctx).Model(&models.Book{}).Where("id = ?", row.ID).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *Service) Remove(ctx context.Context, id int64) error {
	res := s.db.WithContext(ctx).Delete(&models.Book{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return shared.NotFound("book not found")
	}
	return nil
}
