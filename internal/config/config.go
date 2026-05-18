package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type DBClient string

const (
	DBClientSQLite DBClient = "sqlite"
	DBClientMySQL  DBClient = "mysql"
)

type LLMProvider string

const (
	LLMProviderMock      LLMProvider = "mock"
	LLMProviderOpenAI    LLMProvider = "openai"
	LLMProviderAnthropic LLMProvider = "anthropic"
	LLMProviderCustom    LLMProvider = "custom"
)

type MockLLMMode string

const (
	MockLLMModeEcho    MockLLMMode = "echo"
	MockLLMModeFixture MockLLMMode = "fixture"
	MockLLMModeJSON    MockLLMMode = "json"
)

type Config struct {
	NodeEnv string

	LogLevel             string
	LogFormat            string
	LogDir               string
	LogLLMContent        bool
	LogLLMContentMaxChar int

	DBClient     DBClient
	DBSQLitePath string
	DBHost       string
	DBPort       int
	DBName       string
	DBUser       string
	DBPassword   string
	DBPoolMax    int

	LLMProvider          LLMProvider
	MockLLMMode          MockLLMMode
	MockLLMResponseText  string
	MockLLMFixturePath   string
	MockLLMModel         string
	LLMDefaultMaxTokens  int
	OpenAIAPIKey         string
	OpenAIBaseURL        string
	OpenAIModel          string
	AnthropicAPIKey      string
	AnthropicBaseURL     string
	AnthropicModel       string
	CustomLLMBaseURL     string
	CustomLLMAPIKey      string
	CustomLLMModel       string
	LLMLowModel          string
	LLMMidModel          string
	LLMHighModel         string
	LLMRateLimitRPS      int
	LLMRequestTimeoutSec int

	PlanningKeywordMaxLength            int
	PlanningIntentKeywordLimit          int
	PlanningIntentMustIncludeLimit      int
	PlanningIntentMustAvoidLimit        int
	PlanningRetrievalOutlineLimit       int
	PlanningRetrievalRecentChapterLimit int
	PlanningRetrievalRecentChapterScan  int
	PlanningRetrievalHookLimit          int
	PlanningRetrievalCharacterLimit     int
	PlanningRetrievalFactionLimit       int
	PlanningRetrievalItemLimit          int
	PlanningRetrievalRelationLimit      int
	PlanningRetrievalWorldSettingLimit  int
	PlanningRetrievalEntityScanLimit    int
	PlanningRetrievalPersistedFactLimit int
	PlanningRetrievalPersistedEventLimt int
	PlanningContinuitySnapshotLimit     int

	PlanningRetrievalReranker             string
	PlanningRetrievalEmbeddingProvider    string
	PlanningRetrievalEmbeddingSearchMode  string
	PlanningRetrievalEmbeddingMinScore    float64
	PlanningRetrievalEmbeddingOnlyMinScr  float64
	PlanningRetrievalEmbeddingLimitBasic  int
	PlanningRetrievalEmbeddingLimitHybrid int

	CustomEmbeddingBaseURL   string
	CustomEmbeddingAPIKey    string
	CustomEmbeddingModel     string
	CustomEmbeddingBatchSize int

	DraftLengthRepairMaxRounds int

	ServerHost      string
	ServerPort      int
	ServerBodyLimit int

	WebUIDistPath string

	AuthSessionSecret   string
	AuthSessionTTLHours int
	AuthCookieName      string
	AuthCookieSecure    bool

	WorkflowMaxConcurrency int
	ShutdownTimeoutSec     int
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		NodeEnv: getStr("NODE_ENV", "development"),

		LogLevel:             getEnumStr("LOG_LEVEL", "info", []string{"trace", "debug", "info", "warn", "error", "fatal"}),
		LogFormat:            getEnumStr("LOG_FORMAT", "pretty", []string{"pretty", "json"}),
		LogDir:               getStr("LOG_DIR", "./logs"),
		LogLLMContent:        getBool("LOG_LLM_CONTENT_ENABLED", false),
		LogLLMContentMaxChar: getInt("LOG_LLM_CONTENT_MAX_CHARS", 4000),

		DBClient:     DBClient(getEnumStr("DB_CLIENT", "sqlite", []string{"sqlite", "mysql"})),
		DBSQLitePath: getStr("DB_SQLITE_PATH", "./data/novel.db"),
		DBHost:       getStr("DB_HOST", "127.0.0.1"),
		DBPort:       getInt("DB_PORT", 3306),
		DBName:       getStr("DB_NAME", "myai_novel"),
		DBUser:       getStr("DB_USER", "root"),
		DBPassword:   getStr("DB_PASSWORD", ""),
		DBPoolMax:    getInt("DB_POOL_MAX", 10),

		LLMProvider:          LLMProvider(getEnumStr("LLM_PROVIDER", "mock", []string{"mock", "openai", "anthropic", "custom"})),
		MockLLMMode:          MockLLMMode(getEnumStr("MOCK_LLM_MODE", "echo", []string{"echo", "fixture", "json"})),
		MockLLMResponseText:  getStr("MOCK_LLM_RESPONSE_TEXT", "Mock response"),
		MockLLMFixturePath:   strings.TrimSpace(os.Getenv("MOCK_LLM_FIXTURE_PATH")),
		MockLLMModel:         getStr("MOCK_LLM_MODEL", "mock-v1"),
		LLMDefaultMaxTokens:  getInt("LLM_DEFAULT_MAX_TOKENS", 2048),
		OpenAIAPIKey:         strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
		OpenAIBaseURL:        strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")),
		OpenAIModel:          getStr("OPENAI_MODEL", "gpt-4o-mini"),
		AnthropicAPIKey:      strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")),
		AnthropicBaseURL:     strings.TrimSpace(os.Getenv("ANTHROPIC_BASE_URL")),
		AnthropicModel:       getStr("ANTHROPIC_MODEL", "claude-sonnet-4-20250514"),
		CustomLLMBaseURL:     strings.TrimSpace(os.Getenv("CUSTOM_LLM_BASE_URL")),
		CustomLLMAPIKey:      strings.TrimSpace(os.Getenv("CUSTOM_LLM_API_KEY")),
		CustomLLMModel:       getStr("CUSTOM_LLM_MODEL", "custom-default"),
		LLMLowModel:          strings.TrimSpace(os.Getenv("LLM_LOW_MODEL")),
		LLMMidModel:          strings.TrimSpace(os.Getenv("LLM_MID_MODEL")),
		LLMHighModel:         strings.TrimSpace(os.Getenv("LLM_HIGH_MODEL")),
		LLMRateLimitRPS:      getInt("LLM_RATE_LIMIT_RPS", 20),
		LLMRequestTimeoutSec: getInt("LLM_REQUEST_TIMEOUT_SECONDS", 120),

		PlanningKeywordMaxLength:            getInt("PLANNING_KEYWORD_MAX_LENGTH", 8),
		PlanningIntentKeywordLimit:          getInt("PLANNING_INTENT_KEYWORD_LIMIT", 20),
		PlanningIntentMustIncludeLimit:      getInt("PLANNING_INTENT_MUST_INCLUDE_LIMIT", 20),
		PlanningIntentMustAvoidLimit:        getInt("PLANNING_INTENT_MUST_AVOID_LIMIT", 20),
		PlanningRetrievalOutlineLimit:       getInt("PLANNING_RETRIEVAL_OUTLINE_LIMIT", 3),
		PlanningRetrievalRecentChapterLimit: getInt("PLANNING_RETRIEVAL_RECENT_CHAPTER_LIMIT", 8),
		PlanningRetrievalRecentChapterScan:  getInt("PLANNING_RETRIEVAL_RECENT_CHAPTER_SCAN_MULTIPLIER", 12),
		PlanningRetrievalHookLimit:          getInt("PLANNING_RETRIEVAL_HOOK_LIMIT", 24),
		PlanningRetrievalCharacterLimit:     getInt("PLANNING_RETRIEVAL_CHARACTER_LIMIT", 32),
		PlanningRetrievalFactionLimit:       getInt("PLANNING_RETRIEVAL_FACTION_LIMIT", 20),
		PlanningRetrievalItemLimit:          getInt("PLANNING_RETRIEVAL_ITEM_LIMIT", 20),
		PlanningRetrievalRelationLimit:      getInt("PLANNING_RETRIEVAL_RELATION_LIMIT", 32),
		PlanningRetrievalWorldSettingLimit:  getInt("PLANNING_RETRIEVAL_WORLD_SETTING_LIMIT", 20),
		PlanningRetrievalEntityScanLimit:    getInt("PLANNING_RETRIEVAL_ENTITY_SCAN_LIMIT", 1500),
		PlanningRetrievalPersistedFactLimit: getInt("PLANNING_RETRIEVAL_PERSISTED_FACT_LIMIT", 14),
		PlanningRetrievalPersistedEventLimt: getInt("PLANNING_RETRIEVAL_PERSISTED_EVENT_LIMIT", 8),
		PlanningContinuitySnapshotLimit:     getInt("PLANNING_RETRIEVAL_CONTINUITY_SNAPSHOT_LIMIT", 3),

		PlanningRetrievalReranker:             getEnumStr("PLANNING_RETRIEVAL_RERANKER", "heuristic", []string{"heuristic", "none"}),
		PlanningRetrievalEmbeddingProvider:    getEnumStr("PLANNING_RETRIEVAL_EMBEDDING_PROVIDER", "none", []string{"none", "hash", "custom"}),
		PlanningRetrievalEmbeddingSearchMode:  getEnumStr("PLANNING_RETRIEVAL_EMBEDDING_SEARCH_MODE", "basic", []string{"basic", "hybrid"}),
		PlanningRetrievalEmbeddingMinScore:    getFloat("PLANNING_RETRIEVAL_EMBEDDING_MIN_SCORE", 0.64),
		PlanningRetrievalEmbeddingOnlyMinScr:  getFloat("PLANNING_RETRIEVAL_EMBEDDING_ONLY_MIN_SCORE", 0.72),
		PlanningRetrievalEmbeddingLimitBasic:  getInt("PLANNING_RETRIEVAL_EMBEDDING_LIMIT_BASIC", 32),
		PlanningRetrievalEmbeddingLimitHybrid: getInt("PLANNING_RETRIEVAL_EMBEDDING_LIMIT_HYBRID", 40),

		CustomEmbeddingBaseURL:   strings.TrimSpace(os.Getenv("CUSTOM_EMBEDDING_BASE_URL")),
		CustomEmbeddingAPIKey:    strings.TrimSpace(os.Getenv("CUSTOM_EMBEDDING_API_KEY")),
		CustomEmbeddingModel:     getStr("CUSTOM_EMBEDDING_MODEL", "custom-embedding-v1"),
		CustomEmbeddingBatchSize: getInt("CUSTOM_EMBEDDING_BATCH_SIZE", 10),

		DraftLengthRepairMaxRounds: getInt("DRAFT_LENGTH_REPAIR_MAX_ROUNDS", 3),

		ServerHost:      getStr("SERVER_HOST", "127.0.0.1"),
		ServerPort:      getInt("SERVER_PORT", 3000),
		ServerBodyLimit: getInt("SERVER_BODY_LIMIT", 1048576),

		WebUIDistPath: strings.TrimSpace(os.Getenv("WEBUI_DIST_PATH")),

		AuthSessionSecret:   getStr("AUTH_SESSION_SECRET", "dev-session-secret-change-me"),
		AuthSessionTTLHours: getInt("AUTH_SESSION_TTL_HOURS", 168),
		AuthCookieName:      getStr("AUTH_COOKIE_NAME", "myai_novel_session"),
		AuthCookieSecure:    getBool("AUTH_COOKIE_SECURE", false),

		WorkflowMaxConcurrency: getInt("WORKFLOW_MAX_CONCURRENCY", 4),
		ShutdownTimeoutSec:     getInt("SHUTDOWN_TIMEOUT_SECONDS", 15),
	}

	cfg.LogDir = absPath(cfg.LogDir)
	cfg.DBSQLitePath = absPath(cfg.DBSQLitePath)
	if cfg.MockLLMFixturePath != "" {
		cfg.MockLLMFixturePath = absPath(cfg.MockLLMFixturePath)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	switch c.LLMProvider {
	case LLMProviderOpenAI:
		if c.OpenAIAPIKey == "" {
			return errors.New("OPENAI_API_KEY is required when LLM_PROVIDER=openai")
		}
	case LLMProviderAnthropic:
		if c.AnthropicAPIKey == "" {
			return errors.New("ANTHROPIC_API_KEY is required when LLM_PROVIDER=anthropic")
		}
	case LLMProviderCustom:
		if c.CustomLLMBaseURL == "" {
			return errors.New("CUSTOM_LLM_BASE_URL is required when LLM_PROVIDER=custom")
		}
	}
	if c.MockLLMMode == MockLLMModeFixture && c.MockLLMFixturePath == "" {
		return errors.New("MOCK_LLM_FIXTURE_PATH is required when MOCK_LLM_MODE=fixture")
	}
	if c.PlanningRetrievalEmbeddingProvider == "custom" {
		if c.CustomEmbeddingBaseURL == "" {
			return errors.New("CUSTOM_EMBEDDING_BASE_URL is required when PLANNING_RETRIEVAL_EMBEDDING_PROVIDER=custom")
		}
		if c.CustomEmbeddingAPIKey == "" {
			return errors.New("CUSTOM_EMBEDDING_API_KEY is required when PLANNING_RETRIEVAL_EMBEDDING_PROVIDER=custom")
		}
	}
	if c.ServerPort < 0 || c.ServerPort > 65535 {
		return fmt.Errorf("SERVER_PORT out of range: %d", c.ServerPort)
	}
	if c.NodeEnv == "production" && c.AuthSessionSecret == "dev-session-secret-change-me" {
		return errors.New("AUTH_SESSION_SECRET must be explicitly configured in production")
	}
	if len(c.AuthSessionSecret) < 16 {
		return errors.New("AUTH_SESSION_SECRET must be at least 16 characters")
	}
	return nil
}

func getStr(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

func getEnumStr(key, def string, allowed []string) string {
	v := getStr(key, def)
	for _, a := range allowed {
		if v == a {
			return v
		}
	}
	return def
}

func getInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func getFloat(key string, def float64) float64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return n
}

func getBool(key string, def bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if v == "" {
		return def
	}
	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}
	return def
}

func absPath(p string) string {
	if p == "" {
		return ""
	}
	if filepath.IsAbs(p) {
		return p
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return abs
}
