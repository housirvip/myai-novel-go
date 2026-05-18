package character

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
	Name            string  `json:"name" binding:"required,min=1"`
	Alias           *string `json:"alias"`
	Gender          *string `json:"gender"`
	Age             *int    `json:"age"`
	Personality     *string `json:"personality"`
	Background      *string `json:"background"`
	CurrentLocation *string `json:"currentLocation"`
	Status          *string `json:"status"`
	Professions     *string `json:"professions"`
	Levels          *string `json:"levels"`
	Currencies      *string `json:"currencies"`
	Abilities       *string `json:"abilities"`
	Goal            *string `json:"goal"`
	AppendNotes     *string `json:"appendNotes"`
	Keywords        *string `json:"keywords"`
}

type UpdateInput struct {
	Name            *string `json:"name"`
	Alias           *string `json:"alias"`
	Gender          *string `json:"gender"`
	Age             *int    `json:"age"`
	Personality     *string `json:"personality"`
	Background      *string `json:"background"`
	CurrentLocation *string `json:"currentLocation"`
	Status          *string `json:"status"`
	Professions     *string `json:"professions"`
	Levels          *string `json:"levels"`
	Currencies      *string `json:"currencies"`
	Abilities       *string `json:"abilities"`
	Goal            *string `json:"goal"`
	AppendNotes     *string `json:"appendNotes"`
	Keywords        *string `json:"keywords"`
}

func (s *Service) List(ctx context.Context, bookID int64, limit int, status string) ([]models.Character, error) {
	q := s.db.WithContext(ctx).Where("book_id = ?", bookID).Order("id DESC").Limit(limit)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var rows []models.Character
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Service) Get(ctx context.Context, bookID, id int64) (*models.Character, error) {
	var row models.Character
	if err := s.db.WithContext(ctx).Where("book_id = ? AND id = ?", bookID, id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.NotFound("character not found")
		}
		return nil, err
	}
	return &row, nil
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*models.Character, error) {
	now := shared.NowISO()
	row := models.Character{
		BookID: in.BookID, Name: in.Name, Alias: in.Alias, Gender: in.Gender, Age: in.Age,
		Personality: in.Personality, Background: in.Background, CurrentLocation: in.CurrentLocation,
		Status: "active", Professions: in.Professions, Levels: in.Levels, Currencies: in.Currencies,
		Abilities: in.Abilities, Goal: in.Goal, AppendNotes: in.AppendNotes, Keywords: in.Keywords,
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

func (s *Service) Update(ctx context.Context, bookID, id int64, in UpdateInput) (*models.Character, error) {
	row, err := s.Get(ctx, bookID, id)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{"updated_at": shared.NowISO()}
	if in.Name != nil {
		updates["name"] = *in.Name
	}
	if in.Alias != nil {
		updates["alias"] = in.Alias
	}
	if in.Gender != nil {
		updates["gender"] = in.Gender
	}
	if in.Age != nil {
		updates["age"] = in.Age
	}
	if in.Personality != nil {
		updates["personality"] = in.Personality
	}
	if in.Background != nil {
		updates["background"] = in.Background
	}
	if in.CurrentLocation != nil {
		updates["current_location"] = in.CurrentLocation
	}
	if in.Status != nil {
		updates["status"] = *in.Status
	}
	if in.Professions != nil {
		updates["professions"] = in.Professions
	}
	if in.Levels != nil {
		updates["levels"] = in.Levels
	}
	if in.Currencies != nil {
		updates["currencies"] = in.Currencies
	}
	if in.Abilities != nil {
		updates["abilities"] = in.Abilities
	}
	if in.Goal != nil {
		updates["goal"] = in.Goal
	}
	if in.AppendNotes != nil {
		updates["append_notes"] = in.AppendNotes
	}
	if in.Keywords != nil {
		updates["keywords"] = in.Keywords
	}
	if err := s.db.WithContext(ctx).Model(&models.Character{}).Where("id = ?", row.ID).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.Get(ctx, bookID, id)
}

func (s *Service) Remove(ctx context.Context, bookID, id int64) error {
	res := s.db.WithContext(ctx).Where("book_id = ? AND id = ?", bookID, id).Delete(&models.Character{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return shared.NotFound("character not found")
	}
	return nil
}
