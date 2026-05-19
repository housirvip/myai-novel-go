package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"myai-novel-go/internal/testutil"
)

// TestAuthFlow:register → /api/auth/session(带 cookie)→ login → logout 全套校验。
func TestAuthFlow(t *testing.T) {
	srv, _ := testutil.NewTestServer(t, map[string]string{
		"AUTH_SESSION_SECRET": "test-secret-must-be-at-least-16-chars",
	})

	registerBody := `{"email":"alice@example.com","password":"hunter22hunter22","displayName":"Alice"}`
	resp, err := http.Post(srv.URL+"/api/auth/register", "application/json", strings.NewReader(registerBody))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	cookie := resp.Header.Get("Set-Cookie")
	require.NotEmpty(t, cookie, "register 应签发 Set-Cookie")
	require.Contains(t, cookie, "myai_novel_session=")

	// 用 cookie 访问 /api/auth/session,断言能拿到 user
	sessionHdr := extractCookieValue(cookie)
	req, _ := http.NewRequest("GET", srv.URL+"/api/auth/session", nil)
	req.Header.Set("Cookie", "myai_novel_session="+sessionHdr)
	r2, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer r2.Body.Close()
	var body struct{ Data struct{ User map[string]any } }
	require.NoError(t, json.NewDecoder(r2.Body).Decode(&body))
	require.NotNil(t, body.Data.User)
	require.Equal(t, "alice@example.com", body.Data.User["email"])

	// 错误密码 login → 401
	bad, err := http.Post(srv.URL+"/api/auth/login", "application/json",
		strings.NewReader(`{"email":"alice@example.com","password":"wrongwrongwrong"}`))
	require.NoError(t, err)
	defer bad.Body.Close()
	require.Equal(t, http.StatusUnauthorized, bad.StatusCode)

	// 正确 login → 200
	good, err := http.Post(srv.URL+"/api/auth/login", "application/json",
		strings.NewReader(`{"email":"alice@example.com","password":"hunter22hunter22"}`))
	require.NoError(t, err)
	defer good.Body.Close()
	require.Equal(t, http.StatusOK, good.StatusCode)
	require.NotEmpty(t, good.Header.Get("Set-Cookie"))

	// duplicate register → 409
	dup, err := http.Post(srv.URL+"/api/auth/register", "application/json", strings.NewReader(registerBody))
	require.NoError(t, err)
	defer dup.Body.Close()
	require.Equal(t, http.StatusConflict, dup.StatusCode)
}

// TestUserSettings_RequireAuth 未登录访问 /api/user-settings/runtime 应 401。
func TestUserSettings_RequireAuth(t *testing.T) {
	srv, _ := testutil.NewTestServer(t, nil)
	resp, err := http.Get(srv.URL + "/api/user-settings/runtime")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// TestUserSettings_RoundTrip 登录 → PUT 覆盖 → GET 看到 view → DELETE 回落到默认值。
func TestUserSettings_RoundTrip(t *testing.T) {
	srv, _ := testutil.NewTestServer(t, map[string]string{
		"LLM_PROVIDER":           "mock",
		"LLM_DEFAULT_MAX_TOKENS": "4096",
		"LLM_LOW_MODEL":          "env-low",
		"LLM_MID_MODEL":          "env-mid",
		"LLM_HIGH_MODEL":         "env-high",
		"OPENAI_BASE_URL":        "https://env-openai.example/v1",
	})

	resp, err := http.Post(srv.URL+"/api/auth/register", "application/json",
		strings.NewReader(`{"email":"bob@example.com","password":"hunter22hunter22","displayName":"Bob"}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	cookieVal := extractCookieValue(resp.Header.Get("Set-Cookie"))

	type secretField struct {
		HasValue    bool    `json:"hasValue"`
		MaskedValue *string `json:"maskedValue"`
	}
	type runtimeSettingsView struct {
		Provider         *string     `json:"provider"`
		Model            *string     `json:"model"`
		LowModel         *string     `json:"lowModel"`
		MidModel         *string     `json:"midModel"`
		HighModel        *string     `json:"highModel"`
		DefaultMaxTokens *int        `json:"defaultMaxTokens"`
		OpenAIAPIKey     secretField `json:"openaiApiKey"`
		OpenAIBaseURL    *string     `json:"openaiBaseUrl"`
		AnthropicAPIKey  secretField `json:"anthropicApiKey"`
		CustomLLMAPIKey  secretField `json:"customLlmApiKey"`
	}
	type settingsView struct {
		Overrides      runtimeSettingsView `json:"overrides"`
		ServerDefaults runtimeSettingsView `json:"serverDefaults"`
		Effective      runtimeSettingsView `json:"effective"`
		Capabilities   struct {
			AllowedProviders           []string        `json:"allowedProviders"`
			ProviderAvailability       map[string]bool `json:"providerAvailability"`
			SupportsSensitiveOverrides bool            `json:"supportsSensitiveOverrides"`
		} `json:"capabilities"`
	}
	type envelope struct {
		Data settingsView `json:"data"`
	}

	doReq := func(method, path string, body string) *http.Response {
		req, _ := http.NewRequest(method, srv.URL+path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Cookie", "myai_novel_session="+cookieVal)
		r, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		return r
	}

	getSettings := func(resp *http.Response) envelope {
		defer resp.Body.Close()
		var got envelope
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
		return got
	}

	initialResp := doReq("GET", "/api/user-settings/runtime", "")
	require.Equal(t, http.StatusOK, initialResp.StatusCode)
	initial := getSettings(initialResp)
	require.Equal(t, "mock", *initial.Data.ServerDefaults.Provider)
	require.Equal(t, "mock-v1", *initial.Data.ServerDefaults.Model)
	require.Equal(t, "env-low", *initial.Data.ServerDefaults.LowModel)
	require.Equal(t, "env-mid", *initial.Data.ServerDefaults.MidModel)
	require.Equal(t, "env-high", *initial.Data.ServerDefaults.HighModel)
	require.Equal(t, 4096, *initial.Data.ServerDefaults.DefaultMaxTokens)
	require.False(t, initial.Data.Overrides.OpenAIAPIKey.HasValue)
	require.Equal(t, []string{"mock", "openai", "anthropic", "custom"}, initial.Data.Capabilities.AllowedProviders)
	require.False(t, initial.Data.Capabilities.ProviderAvailability["openai"])
	require.True(t, initial.Data.Capabilities.SupportsSensitiveOverrides)

	putResp := doReq("PUT", "/api/user-settings/runtime", `{"llmProvider":"openai","llmModel":"user-generic-model","llmLowModel":"user-low-model","llmDefaultMaxTokens":8192,"openaiApiKey":"sk-user-secret-1234","openaiBaseUrl":"https://openai.example.test/v1"}`)
	require.Equal(t, http.StatusOK, putResp.StatusCode)
	updated := getSettings(putResp)
	require.Equal(t, "openai", *updated.Data.Overrides.Provider)
	require.Equal(t, "user-generic-model", *updated.Data.Overrides.Model)
	require.Equal(t, "user-low-model", *updated.Data.Overrides.LowModel)
	require.Equal(t, 8192, *updated.Data.Overrides.DefaultMaxTokens)
	require.True(t, updated.Data.Overrides.OpenAIAPIKey.HasValue)
	require.NotNil(t, updated.Data.Overrides.OpenAIAPIKey.MaskedValue)
	require.Equal(t, "sk-u...1234", *updated.Data.Overrides.OpenAIAPIKey.MaskedValue)
	require.Equal(t, "openai", *updated.Data.Effective.Provider)
	require.Equal(t, "user-generic-model", *updated.Data.Effective.Model)
	require.Equal(t, "user-low-model", *updated.Data.Effective.LowModel)
	require.Equal(t, "env-mid", *updated.Data.Effective.MidModel)
	require.Equal(t, "env-high", *updated.Data.Effective.HighModel)
	require.Equal(t, 8192, *updated.Data.Effective.DefaultMaxTokens)
	require.True(t, updated.Data.Capabilities.ProviderAvailability["openai"])

	deleteResp := doReq("DELETE", "/api/user-settings/runtime", "")
	require.Equal(t, http.StatusOK, deleteResp.StatusCode)
	cleared := getSettings(deleteResp)
	require.Nil(t, cleared.Data.Overrides.Provider)
	require.Equal(t, "mock", *cleared.Data.Effective.Provider)
	require.Equal(t, "mock-v1", *cleared.Data.Effective.Model)
	require.Equal(t, "env-low", *cleared.Data.Effective.LowModel)
	require.Equal(t, "env-mid", *cleared.Data.Effective.MidModel)
	require.Equal(t, "env-high", *cleared.Data.Effective.HighModel)
	require.Equal(t, 4096, *cleared.Data.Effective.DefaultMaxTokens)
	require.False(t, cleared.Data.Effective.OpenAIAPIKey.HasValue)
}

// TestMetaEndpoint /api/meta 返回 meta 信息。
func TestMetaEndpoint(t *testing.T) {
	srv, _ := testutil.NewTestServer(t, nil)
	resp, err := http.Get(srv.URL + "/api/meta")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var meta struct {
		Data struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		}
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&meta))
	require.Equal(t, "myai-novel-go", meta.Data.Name)
	require.Equal(t, "0.1.0", meta.Data.Version)
}

// extractCookieValue 从 Set-Cookie 头里取出 cookie value(分号前那一段去掉 name=)。
func extractCookieValue(setCookie string) string {
	parts := strings.SplitN(setCookie, ";", 2)
	if len(parts) == 0 {
		return ""
	}
	kv := strings.SplitN(parts[0], "=", 2)
	if len(kv) < 2 {
		return ""
	}
	return kv[1]
}
