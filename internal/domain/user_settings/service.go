package user_settings

import (
	"context"
	"encoding/json"
	"errors"

	"gorm.io/gorm"

	"myai-novel-go/internal/db/models"
	"myai-novel-go/internal/domain/shared"
)

// RuntimeOverrides 是用户级 LLM 配置覆盖,
// 仅持久化非空字段;每个 *string 设为 null 表示"清除该覆盖"。
type RuntimeOverrides struct {
	LLMProvider         *string `json:"llmProvider,omitempty"`
	LLMModel            *string `json:"llmModel,omitempty"`
	LLMLowModel         *string `json:"llmLowModel,omitempty"`
	LLMMidModel         *string `json:"llmMidModel,omitempty"`
	LLMHighModel        *string `json:"llmHighModel,omitempty"`
	LLMDefaultMaxTokens *int    `json:"llmDefaultMaxTokens,omitempty"`
	OpenAIAPIKey        *string `json:"openaiApiKey,omitempty"`
	OpenAIBaseURL       *string `json:"openaiBaseUrl,omitempty"`
	AnthropicAPIKey     *string `json:"anthropicApiKey,omitempty"`
	AnthropicBaseURL    *string `json:"anthropicBaseUrl,omitempty"`
	CustomLLMAPIKey     *string `json:"customLlmApiKey,omitempty"`
	CustomLLMBaseURL    *string `json:"customLlmBaseUrl,omitempty"`
}

// UpdateInput 用 *string + 显式 null 区分"不改"和"清除":
// json 反序列化时,缺失字段 → nil(不改),null → 指向空字符串(清除),非空字符串 → 设值。
type UpdateInput = RuntimeOverrides

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

func (s *Service) Get(ctx context.Context, userID int64) (*RuntimeOverrides, error) {
	row, err := s.fetch(ctx, userID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return &RuntimeOverrides{}, nil
	}
	var out RuntimeOverrides
	if row.RuntimeOverrides != "" {
		if err := json.Unmarshal([]byte(row.RuntimeOverrides), &out); err != nil {
			return nil, shared.Internal("user settings json corrupted")
		}
	}
	return &out, nil
}

// Update merge 进现有覆盖。约定:nil 字段不改,空字符串清除该字段(JSON 中显式 "" 视为清除)。
func (s *Service) Update(ctx context.Context, userID int64, in UpdateInput) (*RuntimeOverrides, error) {
	current, err := s.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	merged := merge(*current, in)
	body, err := json.Marshal(merged)
	if err != nil {
		return nil, err
	}
	now := shared.NowISO()
	row, err := s.fetch(ctx, userID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		row = &models.UserSetting{
			UserID:           userID,
			RuntimeOverrides: string(body),
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
			return nil, err
		}
		return &merged, nil
	}
	if err := s.db.WithContext(ctx).Model(row).Updates(map[string]any{
		"runtime_overrides": string(body),
		"updated_at":        now,
	}).Error; err != nil {
		return nil, err
	}
	return &merged, nil
}

// Clear 把整行删掉,等价"恢复默认 cfg"。
func (s *Service) Clear(ctx context.Context, userID int64) error {
	return s.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.UserSetting{}).Error
}

func (s *Service) fetch(ctx context.Context, userID int64) (*models.UserSetting, error) {
	var row models.UserSetting
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

// merge 把 patch 中非 nil 的字段写入 base,空字符串 → 视为清除(返回 nil)。
func merge(base, patch RuntimeOverrides) RuntimeOverrides {
	apply := func(dst **string, src *string) {
		if src == nil {
			return
		}
		if *src == "" {
			*dst = nil
			return
		}
		v := *src
		*dst = &v
	}
	apply(&base.LLMProvider, patch.LLMProvider)
	apply(&base.LLMModel, patch.LLMModel)
	apply(&base.LLMLowModel, patch.LLMLowModel)
	apply(&base.LLMMidModel, patch.LLMMidModel)
	apply(&base.LLMHighModel, patch.LLMHighModel)
	if patch.LLMDefaultMaxTokens != nil {
		if *patch.LLMDefaultMaxTokens <= 0 {
			base.LLMDefaultMaxTokens = nil
		} else {
			v := *patch.LLMDefaultMaxTokens
			base.LLMDefaultMaxTokens = &v
		}
	}
	apply(&base.OpenAIAPIKey, patch.OpenAIAPIKey)
	apply(&base.OpenAIBaseURL, patch.OpenAIBaseURL)
	apply(&base.AnthropicAPIKey, patch.AnthropicAPIKey)
	apply(&base.AnthropicBaseURL, patch.AnthropicBaseURL)
	apply(&base.CustomLLMAPIKey, patch.CustomLLMAPIKey)
	apply(&base.CustomLLMBaseURL, patch.CustomLLMBaseURL)
	return base
}
