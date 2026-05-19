package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"myai-novel-go/internal/testutil"
)

// TestBookOwnership 验证多租户落地:
//   - alice 创建的 book bob 看不到 / 拿不到 / 改不了
//   - 匿名仍可访问 owner_user_id IS NULL 的 book
func TestBookOwnership(t *testing.T) {
	srv, _ := testutil.NewTestServer(t, nil)
	base := srv.URL

	// 1) 注册 alice
	aliceCookie := registerAndCookie(t, base, "alice@example.com", "hunter22hunter22", "Alice")
	// 2) alice 创建一本书
	resp := doAuthed(t, base, aliceCookie, "POST", "/api/books",
		`{"title":"Alice 的书","targetChapterCount":10}`)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var aliceBook struct {
		Data struct {
			ID          int64 `json:"id"`
			OwnerUserID int64 `json:"ownerUserId"`
		}
	}
	mustDecode(t, resp, &aliceBook)
	require.NotZero(t, aliceBook.Data.ID)
	require.NotZero(t, aliceBook.Data.OwnerUserID, "登录用户创建的 book 应自动归属本人")

	// 3) bob 注册
	bobCookie := registerAndCookie(t, base, "bob@example.com", "hunter22hunter22", "Bob")

	// 4) bob list 不到 alice 的 book
	resp = doAuthed(t, base, bobCookie, "GET", "/api/books", "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var bobList struct{ Data []struct{ ID int64 } }
	mustDecode(t, resp, &bobList)
	for _, b := range bobList.Data {
		require.NotEqual(t, aliceBook.Data.ID, b.ID, "bob 不应能看到 alice 的 book")
	}

	// 5) bob 直接 GET alice 的 book → 403
	resp = doAuthed(t, base, bobCookie, "GET", path("/api/books/%d", aliceBook.Data.ID), "")
	require.Equal(t, http.StatusForbidden, resp.StatusCode)

	// 6) bob 改 alice 的 book → 403
	resp = doAuthed(t, base, bobCookie, "PATCH", path("/api/books/%d", aliceBook.Data.ID), `{"title":"hijack"}`)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)

	// 7) alice 自己仍能看到/改
	resp = doAuthed(t, base, aliceCookie, "GET", path("/api/books/%d", aliceBook.Data.ID), "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestAnonymousBook 仍可访问 owner_user_id IS NULL 的 book(向后兼容引导期场景)。
func TestAnonymousBook_BackwardCompat(t *testing.T) {
	srv, _ := testutil.NewTestServer(t, nil)
	// 匿名直接创建(没有 cookie)→ 该 book owner_user_id 应为 NULL
	resp, err := http.Post(srv.URL+"/api/books", "application/json",
		strings.NewReader(`{"title":"匿名书"}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var b struct {
		Data struct {
			ID          int64  `json:"id"`
			OwnerUserID *int64 `json:"ownerUserId"`
		}
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&b))
	require.NotZero(t, b.Data.ID)
	require.Nil(t, b.Data.OwnerUserID, "匿名创建的 book owner_user_id 应为 nil")

	// 匿名再 GET 该 book 仍 200
	resp2, err := http.Get(srv.URL + path("/api/books/%d", b.Data.ID))
	require.NoError(t, err)
	defer resp2.Body.Close()
	require.Equal(t, http.StatusOK, resp2.StatusCode)
}

// TestHealthzReady /healthz/ready 在 db 健康时返回 200。
func TestHealthzReady(t *testing.T) {
	srv, _ := testutil.NewTestServer(t, nil)
	resp, err := http.Get(srv.URL + "/healthz/ready")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var body struct{ Data struct{ Status string } }
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "ready", body.Data.Status)
}

// ----- helpers -----

func registerAndCookie(t *testing.T, base, email, password, name string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"email": email, "password": password, "displayName": name})
	resp, err := http.Post(base+"/api/auth/register", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	return extractCookieValue(resp.Header.Get("Set-Cookie"))
}

func doAuthed(t *testing.T, base, cookie, method, p, body string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(method, base+p, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != "" {
		req.Header.Set("Cookie", "myai_novel_session="+cookie)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func mustDecode(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	require.NoError(t, json.NewDecoder(resp.Body).Decode(v))
}

func path(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}
