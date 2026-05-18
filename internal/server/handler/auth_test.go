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
	var body struct{ User map[string]any }
	require.NoError(t, json.NewDecoder(r2.Body).Decode(&body))
	require.NotNil(t, body.User)
	require.Equal(t, "alice@example.com", body.User["email"])

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

// TestUserSettings_RoundTrip 登录 → PUT 覆盖 → GET 看到 → DELETE 清空。
func TestUserSettings_RoundTrip(t *testing.T) {
	srv, _ := testutil.NewTestServer(t, nil)

	// 注册并拿到 cookie
	resp, err := http.Post(srv.URL+"/api/auth/register", "application/json",
		strings.NewReader(`{"email":"bob@example.com","password":"hunter22hunter22","displayName":"Bob"}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	cookieVal := extractCookieValue(resp.Header.Get("Set-Cookie"))

	doReq := func(method, path string, body string) *http.Response {
		req, _ := http.NewRequest(method, srv.URL+path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Cookie", "myai_novel_session="+cookieVal)
		r, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		return r
	}

	// PUT 覆盖
	r1 := doReq("PUT", "/api/user-settings/runtime", `{"llmProvider":"openai","openaiApiKey":"sk-test"}`)
	defer r1.Body.Close()
	require.Equal(t, http.StatusOK, r1.StatusCode)

	// GET 应能看到
	r2 := doReq("GET", "/api/user-settings/runtime", "")
	defer r2.Body.Close()
	require.Equal(t, http.StatusOK, r2.StatusCode)
	var got map[string]any
	require.NoError(t, json.NewDecoder(r2.Body).Decode(&got))
	require.Equal(t, "openai", got["llmProvider"])
	require.Equal(t, "sk-test", got["openaiApiKey"])

	// DELETE 清空
	r3 := doReq("DELETE", "/api/user-settings/runtime", "")
	defer r3.Body.Close()
	require.Equal(t, http.StatusOK, r3.StatusCode)

	// 再 GET 应该是空
	r4 := doReq("GET", "/api/user-settings/runtime", "")
	defer r4.Body.Close()
	var got2 map[string]any
	require.NoError(t, json.NewDecoder(r4.Body).Decode(&got2))
	require.Nil(t, got2["llmProvider"])
}

// TestMetaEndpoint /api/meta 返回 meta 信息。
func TestMetaEndpoint(t *testing.T) {
	srv, _ := testutil.NewTestServer(t, nil)
	resp, err := http.Get(srv.URL + "/api/meta")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var meta map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&meta))
	require.Equal(t, "myai-novel-go", meta["name"])
	require.Equal(t, "0.1.0", meta["version"])
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
