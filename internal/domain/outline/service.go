package outline

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
	BookID         int64   `json:"-"`
	VolumeNo       *int    `json:"volumeNo"`
	VolumeTitle    *string `json:"volumeTitle"`
	ChapterStartNo *int    `json:"chapterStartNo"`
	ChapterEndNo   *int    `json:"chapterEndNo"`
	OutlineLevel   string  `json:"outlineLevel"`
	Title          string  `json:"title" binding:"required,min=1"`
	StoryCore      *string `json:"storyCore"`
	MainPlot       *string `json:"mainPlot"`
	SubPlot        *string `json:"subPlot"`
	Foreshadowing  *string `json:"foreshadowing"`
	ExpectedPayoff *string `json:"expectedPayoff"`
	Notes          *string `json:"notes"`
}

type UpdateInput struct {
	VolumeNo       *int    `json:"volumeNo"`
	VolumeTitle    *string `json:"volumeTitle"`
	ChapterStartNo *int    `json:"chapterStartNo"`
	ChapterEndNo   *int    `json:"chapterEndNo"`
	OutlineLevel   *string `json:"outlineLevel"`
	Title          *string `json:"title"`
	StoryCore      *string `json:"storyCore"`
	MainPlot       *string `json:"mainPlot"`
	SubPlot        *string `json:"subPlot"`
	Foreshadowing  *string `json:"foreshadowing"`
	ExpectedPayoff *string `json:"expectedPayoff"`
	Notes          *string `json:"notes"`
}

func (s *Service) List(ctx context.Context, bookID int64, limit int) ([]models.Outline, error) {
	var rows []models.Outline
	if err := s.db.WithContext(ctx).Where("book_id = ?", bookID).Order("id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Service) Get(ctx context.Context, bookID, id int64) (*models.Outline, error) {
	var row models.Outline
	if err := s.db.WithContext(ctx).Where("book_id = ? AND id = ?", bookID, id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.NotFound("outline not found")
		}
		return nil, err
	}
	return &row, nil
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*models.Outline, error) {
	now := shared.NowISO()
	if in.OutlineLevel == "" {
		in.OutlineLevel = "volume"
	}
	row := models.Outline{
		BookID: in.BookID, VolumeNo: in.VolumeNo, VolumeTitle: in.VolumeTitle,
		ChapterStartNo: in.ChapterStartNo, ChapterEndNo: in.ChapterEndNo,
		OutlineLevel: in.OutlineLevel, Title: in.Title, StoryCore: in.StoryCore,
		MainPlot: in.MainPlot, SubPlot: in.SubPlot, Foreshadowing: in.Foreshadowing,
		ExpectedPayoff: in.ExpectedPayoff, Notes: in.Notes,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) Update(ctx context.Context, bookID, id int64, in UpdateInput) (*models.Outline, error) {
	row, err := s.Get(ctx, bookID, id)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{"updated_at": shared.NowISO()}
	if in.VolumeNo != nil {
		updates["volume_no"] = in.VolumeNo
	}
	if in.VolumeTitle != nil {
		updates["volume_title"] = in.VolumeTitle
	}
	if in.ChapterStartNo != nil {
		updates["chapter_start_no"] = in.ChapterStartNo
	}
	if in.ChapterEndNo != nil {
		updates["chapter_end_no"] = in.ChapterEndNo
	}
	if in.OutlineLevel != nil {
		updates["outline_level"] = *in.OutlineLevel
	}
	if in.Title != nil {
		updates["title"] = *in.Title
	}
	if in.StoryCore != nil {
		updates["story_core"] = in.StoryCore
	}
	if in.MainPlot != nil {
		updates["main_plot"] = in.MainPlot
	}
	if in.SubPlot != nil {
		updates["sub_plot"] = in.SubPlot
	}
	if in.Foreshadowing != nil {
		updates["foreshadowing"] = in.Foreshadowing
	}
	if in.ExpectedPayoff != nil {
		updates["expected_payoff"] = in.ExpectedPayoff
	}
	if in.Notes != nil {
		updates["notes"] = in.Notes
	}
	if err := s.db.WithContext(ctx).Model(&models.Outline{}).Where("id = ?", row.ID).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.Get(ctx, bookID, id)
}

func (s *Service) Remove(ctx context.Context, bookID, id int64) error {
	res := s.db.WithContext(ctx).Where("book_id = ? AND id = ?", bookID, id).Delete(&models.Outline{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return shared.NotFound("outline not found")
	}
	return nil
}
