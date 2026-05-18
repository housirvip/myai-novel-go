package faction

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
	BookID            int64   `json:"-"`
	Name              string  `json:"name" binding:"required,min=1"`
	Category          *string `json:"category"`
	CoreGoal          *string `json:"coreGoal"`
	Description       *string `json:"description"`
	LeaderCharacterID *int64  `json:"leaderCharacterId"`
	Headquarter       *string `json:"headquarter"`
	Status            *string `json:"status"`
	AppendNotes       *string `json:"appendNotes"`
	Keywords          *string `json:"keywords"`
}

type UpdateInput struct {
	Name              *string `json:"name"`
	Category          *string `json:"category"`
	CoreGoal          *string `json:"coreGoal"`
	Description       *string `json:"description"`
	LeaderCharacterID *int64  `json:"leaderCharacterId"`
	Headquarter       *string `json:"headquarter"`
	Status            *string `json:"status"`
	AppendNotes       *string `json:"appendNotes"`
	Keywords          *string `json:"keywords"`
}

func (s *Service) List(ctx context.Context, bookID int64, limit int, status string) ([]models.Faction, error) {
	q := s.db.WithContext(ctx).Where("book_id = ?", bookID).Order("id DESC").Limit(limit)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var rows []models.Faction
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Service) Get(ctx context.Context, bookID, id int64) (*models.Faction, error) {
	var row models.Faction
	if err := s.db.WithContext(ctx).Where("book_id = ? AND id = ?", bookID, id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.NotFound("faction not found")
		}
		return nil, err
	}
	return &row, nil
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*models.Faction, error) {
	now := shared.NowISO()
	row := models.Faction{
		BookID: in.BookID, Name: in.Name, Category: in.Category, CoreGoal: in.CoreGoal,
		Description: in.Description, LeaderCharacterID: in.LeaderCharacterID, Headquarter: in.Headquarter,
		Status: in.Status, AppendNotes: in.AppendNotes, Keywords: in.Keywords,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) Update(ctx context.Context, bookID, id int64, in UpdateInput) (*models.Faction, error) {
	row, err := s.Get(ctx, bookID, id)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{"updated_at": shared.NowISO()}
	if in.Name != nil {
		updates["name"] = *in.Name
	}
	if in.Category != nil {
		updates["category"] = in.Category
	}
	if in.CoreGoal != nil {
		updates["core_goal"] = in.CoreGoal
	}
	if in.Description != nil {
		updates["description"] = in.Description
	}
	if in.LeaderCharacterID != nil {
		updates["leader_character_id"] = in.LeaderCharacterID
	}
	if in.Headquarter != nil {
		updates["headquarter"] = in.Headquarter
	}
	if in.Status != nil {
		updates["status"] = in.Status
	}
	if in.AppendNotes != nil {
		updates["append_notes"] = in.AppendNotes
	}
	if in.Keywords != nil {
		updates["keywords"] = in.Keywords
	}
	if err := s.db.WithContext(ctx).Model(&models.Faction{}).Where("id = ?", row.ID).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.Get(ctx, bookID, id)
}

func (s *Service) Remove(ctx context.Context, bookID, id int64) error {
	res := s.db.WithContext(ctx).Where("book_id = ? AND id = ?", bookID, id).Delete(&models.Faction{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return shared.NotFound("faction not found")
	}
	return nil
}
