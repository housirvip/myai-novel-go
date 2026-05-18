package llm

import "myai-novel-go/internal/config"

type ModelTier string

const (
	TierLow  ModelTier = "low"
	TierMid  ModelTier = "mid"
	TierHigh ModelTier = "high"
)

func ResolveModel(cfg *config.Config, explicit string, tier ModelTier) string {
	if explicit != "" {
		return explicit
	}
	switch tier {
	case TierLow:
		return cfg.LLMLowModel
	case TierMid:
		return cfg.LLMMidModel
	case TierHigh:
		return cfg.LLMHighModel
	}
	return ""
}
