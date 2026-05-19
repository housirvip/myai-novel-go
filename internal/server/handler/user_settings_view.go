package handler

import (
	"strings"

	"myai-novel-go/internal/config"
	usersettings "myai-novel-go/internal/domain/user_settings"
)

type userRuntimeSettingsView struct {
	Overrides      llmRuntimeSettingsView  `json:"overrides"`
	ServerDefaults llmRuntimeSettingsView  `json:"serverDefaults"`
	Effective      llmRuntimeSettingsView  `json:"effective"`
	Capabilities   runtimeCapabilitiesView `json:"capabilities"`
}

type llmRuntimeSettingsView struct {
	Provider         *string        `json:"provider,omitempty"`
	Model            *string        `json:"model,omitempty"`
	LowModel         *string        `json:"lowModel,omitempty"`
	MidModel         *string        `json:"midModel,omitempty"`
	HighModel        *string        `json:"highModel,omitempty"`
	DefaultMaxTokens *int           `json:"defaultMaxTokens,omitempty"`
	OpenAIAPIKey     llmSecretField `json:"openaiApiKey"`
	OpenAIBaseURL    *string        `json:"openaiBaseUrl,omitempty"`
	AnthropicAPIKey  llmSecretField `json:"anthropicApiKey"`
	AnthropicBaseURL *string        `json:"anthropicBaseUrl,omitempty"`
	CustomLLMAPIKey  llmSecretField `json:"customLlmApiKey"`
	CustomLLMBaseURL *string        `json:"customLlmBaseUrl,omitempty"`
}

type llmSecretField struct {
	HasValue    bool    `json:"hasValue"`
	MaskedValue *string `json:"maskedValue"`
}

type runtimeCapabilitiesView struct {
	AllowedProviders           []string        `json:"allowedProviders"`
	ProviderAvailability       map[string]bool `json:"providerAvailability"`
	SupportsSensitiveOverrides bool            `json:"supportsSensitiveOverrides"`
}

type resolvedRuntimeSettings struct {
	Provider         string
	Model            string
	LowModel         string
	MidModel         string
	HighModel        string
	DefaultMaxTokens int
	OpenAIAPIKey     string
	OpenAIBaseURL    string
	AnthropicAPIKey  string
	AnthropicBaseURL string
	CustomLLMAPIKey  string
	CustomLLMBaseURL string
}

func buildUserRuntimeSettingsView(cfg *config.Config, overrides *usersettings.RuntimeOverrides) userRuntimeSettingsView {
	if overrides == nil {
		overrides = &usersettings.RuntimeOverrides{}
	}
	serverDefaults := buildServerDefaultRuntimeSettings(cfg)
	effective := resolveUserRuntimeSettings(cfg, overrides)
	return userRuntimeSettingsView{
		Overrides:      buildOverridesView(overrides),
		ServerDefaults: buildResolvedRuntimeSettingsView(serverDefaults),
		Effective:      buildResolvedRuntimeSettingsView(effective),
		Capabilities:   buildRuntimeCapabilitiesView(effective),
	}
}

func buildOverridesView(overrides *usersettings.RuntimeOverrides) llmRuntimeSettingsView {
	return llmRuntimeSettingsView{
		Provider:         optionalStringPtr(overrides.LLMProvider),
		Model:            optionalStringPtr(overrides.LLMModel),
		LowModel:         optionalStringPtr(overrides.LLMLowModel),
		MidModel:         optionalStringPtr(overrides.LLMMidModel),
		HighModel:        optionalStringPtr(overrides.LLMHighModel),
		DefaultMaxTokens: optionalIntPtr(overrides.LLMDefaultMaxTokens),
		OpenAIAPIKey:     toSecretField(optionalStringValue(overrides.OpenAIAPIKey)),
		OpenAIBaseURL:    optionalStringPtr(overrides.OpenAIBaseURL),
		AnthropicAPIKey:  toSecretField(optionalStringValue(overrides.AnthropicAPIKey)),
		AnthropicBaseURL: optionalStringPtr(overrides.AnthropicBaseURL),
		CustomLLMAPIKey:  toSecretField(optionalStringValue(overrides.CustomLLMAPIKey)),
		CustomLLMBaseURL: optionalStringPtr(overrides.CustomLLMBaseURL),
	}
}

func buildServerDefaultRuntimeSettings(cfg *config.Config) resolvedRuntimeSettings {
	provider := string(cfg.LLMProvider)
	return resolvedRuntimeSettings{
		Provider:         provider,
		Model:            defaultProviderModel(cfg, provider),
		LowModel:         strings.TrimSpace(cfg.LLMLowModel),
		MidModel:         strings.TrimSpace(cfg.LLMMidModel),
		HighModel:        strings.TrimSpace(cfg.LLMHighModel),
		DefaultMaxTokens: cfg.LLMDefaultMaxTokens,
		OpenAIAPIKey:     strings.TrimSpace(cfg.OpenAIAPIKey),
		OpenAIBaseURL:    strings.TrimSpace(cfg.OpenAIBaseURL),
		AnthropicAPIKey:  strings.TrimSpace(cfg.AnthropicAPIKey),
		AnthropicBaseURL: strings.TrimSpace(cfg.AnthropicBaseURL),
		CustomLLMAPIKey:  strings.TrimSpace(cfg.CustomLLMAPIKey),
		CustomLLMBaseURL: strings.TrimSpace(cfg.CustomLLMBaseURL),
	}
}

func resolveUserRuntimeSettings(cfg *config.Config, overrides *usersettings.RuntimeOverrides) resolvedRuntimeSettings {
	serverDefaults := buildServerDefaultRuntimeSettings(cfg)
	provider := serverDefaults.Provider
	if v := optionalStringValue(overrides.LLMProvider); v != "" {
		provider = v
	}
	model := optionalStringValue(overrides.LLMModel)
	if model == "" {
		model = defaultProviderModel(cfg, provider)
	}
	lowModel := optionalStringValue(overrides.LLMLowModel)
	if lowModel == "" {
		if serverDefaults.LowModel != "" {
			lowModel = serverDefaults.LowModel
		} else {
			lowModel = model
		}
	}
	midModel := optionalStringValue(overrides.LLMMidModel)
	if midModel == "" {
		if serverDefaults.MidModel != "" {
			midModel = serverDefaults.MidModel
		} else {
			midModel = model
		}
	}
	highModel := optionalStringValue(overrides.LLMHighModel)
	if highModel == "" {
		if serverDefaults.HighModel != "" {
			highModel = serverDefaults.HighModel
		} else {
			highModel = model
		}
	}
	defaultMaxTokens := cfg.LLMDefaultMaxTokens
	if overrides.LLMDefaultMaxTokens != nil && *overrides.LLMDefaultMaxTokens > 0 {
		defaultMaxTokens = *overrides.LLMDefaultMaxTokens
	}
	openAIAPIKey, openAIBaseURL := resolveProviderConnection(optionalStringValue(overrides.OpenAIAPIKey), optionalStringValue(overrides.OpenAIBaseURL), serverDefaults.OpenAIAPIKey, serverDefaults.OpenAIBaseURL, true)
	anthropicAPIKey, anthropicBaseURL := resolveProviderConnection(optionalStringValue(overrides.AnthropicAPIKey), optionalStringValue(overrides.AnthropicBaseURL), serverDefaults.AnthropicAPIKey, serverDefaults.AnthropicBaseURL, true)
	customLLMAPIKey, customLLMBaseURL := resolveProviderConnection(optionalStringValue(overrides.CustomLLMAPIKey), optionalStringValue(overrides.CustomLLMBaseURL), serverDefaults.CustomLLMAPIKey, serverDefaults.CustomLLMBaseURL, false)
	return resolvedRuntimeSettings{
		Provider:         provider,
		Model:            model,
		LowModel:         lowModel,
		MidModel:         midModel,
		HighModel:        highModel,
		DefaultMaxTokens: defaultMaxTokens,
		OpenAIAPIKey:     openAIAPIKey,
		OpenAIBaseURL:    openAIBaseURL,
		AnthropicAPIKey:  anthropicAPIKey,
		AnthropicBaseURL: anthropicBaseURL,
		CustomLLMAPIKey:  customLLMAPIKey,
		CustomLLMBaseURL: customLLMBaseURL,
	}
}

func resolveProviderConnection(overrideAPIKey, overrideBaseURL, envAPIKey, envBaseURL string, requireAPIKeyPairing bool) (string, string) {
	hasOverrideAPIKey := overrideAPIKey != ""
	hasOverrideBaseURL := overrideBaseURL != ""
	if !hasOverrideAPIKey && !hasOverrideBaseURL {
		return envAPIKey, envBaseURL
	}
	apiKey := envAPIKey
	if hasOverrideAPIKey {
		apiKey = overrideAPIKey
	} else if requireAPIKeyPairing {
		apiKey = ""
	}
	baseURL := envBaseURL
	if hasOverrideBaseURL {
		baseURL = overrideBaseURL
	}
	return apiKey, baseURL
}

func buildResolvedRuntimeSettingsView(settings resolvedRuntimeSettings) llmRuntimeSettingsView {
	return llmRuntimeSettingsView{
		Provider:         valueStringPtr(settings.Provider),
		Model:            valueStringPtr(settings.Model),
		LowModel:         valueStringPtr(settings.LowModel),
		MidModel:         valueStringPtr(settings.MidModel),
		HighModel:        valueStringPtr(settings.HighModel),
		DefaultMaxTokens: valueIntPtr(settings.DefaultMaxTokens),
		OpenAIAPIKey:     toSecretField(settings.OpenAIAPIKey),
		OpenAIBaseURL:    valueStringPtr(settings.OpenAIBaseURL),
		AnthropicAPIKey:  toSecretField(settings.AnthropicAPIKey),
		AnthropicBaseURL: valueStringPtr(settings.AnthropicBaseURL),
		CustomLLMAPIKey:  toSecretField(settings.CustomLLMAPIKey),
		CustomLLMBaseURL: valueStringPtr(settings.CustomLLMBaseURL),
	}
}

func buildRuntimeCapabilitiesView(settings resolvedRuntimeSettings) runtimeCapabilitiesView {
	return runtimeCapabilitiesView{
		AllowedProviders: []string{"mock", "openai", "anthropic", "custom"},
		ProviderAvailability: map[string]bool{
			"mock":      true,
			"openai":    settings.OpenAIAPIKey != "",
			"anthropic": settings.AnthropicAPIKey != "",
			"custom":    settings.CustomLLMBaseURL != "",
		},
		SupportsSensitiveOverrides: true,
	}
}

func defaultProviderModel(cfg *config.Config, provider string) string {
	switch provider {
	case "openai":
		return strings.TrimSpace(cfg.OpenAIModel)
	case "anthropic":
		return strings.TrimSpace(cfg.AnthropicModel)
	case "custom":
		return strings.TrimSpace(cfg.CustomLLMModel)
	default:
		return strings.TrimSpace(cfg.MockLLMModel)
	}
}

func toSecretField(value string) llmSecretField {
	if value == "" {
		return llmSecretField{HasValue: false, MaskedValue: nil}
	}
	masked := maskSecretValue(value)
	return llmSecretField{HasValue: true, MaskedValue: &masked}
}

func maskSecretValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= 8 {
		return trimmed[:1] + "***" + trimmed[len(trimmed)-1:]
	}
	return trimmed[:4] + "..." + trimmed[len(trimmed)-4:]
}

func optionalStringPtr(v *string) *string {
	if v == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*v)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func optionalStringValue(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}

func optionalIntPtr(v *int) *int {
	if v == nil {
		return nil
	}
	if *v <= 0 {
		return nil
	}
	value := *v
	return &value
}

func valueStringPtr(v string) *string {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func valueIntPtr(v int) *int {
	if v <= 0 {
		return nil
	}
	value := v
	return &value
}
