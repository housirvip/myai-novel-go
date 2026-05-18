package server_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"myai-novel-go/internal/testutil"
)

// TestServerSmoke 跑一遍 plan→draft→review→repair→approve 链路,
// 用 mock provider,断言每步 200 + 最终 lifecycle 为 approved。
func TestServerSmoke(t *testing.T) {
	srv, _ := testutil.NewTestServer(t, nil)
	base := srv.URL

	status, _ := testutil.MustGet(t, base, "/health")
	require.Equal(t, http.StatusOK, status)

	// create book
	status, body := testutil.MustPostJSON(t, base, "/api/books", map[string]any{
		"title": "测试书", "targetChapterCount": 100,
	})
	require.Equal(t, http.StatusCreated, status, string(body))
	var book struct{ ID int64 }
	testutil.MustUnmarshal(t, body, &book)
	require.NotZero(t, book.ID)

	// create chapter
	status, body = testutil.MustPostJSON(t, base, "/api/books/1/chapters", map[string]any{
		"chapterNo": 1, "title": "起势",
	})
	require.Equal(t, http.StatusCreated, status, string(body))

	// 给 chapter 1 添加一些设定方便检索
	for _, p := range []map[string]any{
		{"path": "/api/books/1/characters", "body": map[string]any{"name": "林夜", "keywords": "林夜"}},
		{"path": "/api/books/1/items", "body": map[string]any{"name": "黑铁令", "ownerType": "none", "keywords": "黑铁令"}},
		{"path": "/api/books/1/hooks", "body": map[string]any{"title": "黑铁令异常", "hookType": "mystery", "keywords": "黑铁令"}},
	} {
		s, b := testutil.MustPostJSON(t, base, p["path"].(string), p["body"])
		require.Equal(t, http.StatusCreated, s, string(b))
	}

	// 走完五阶段
	for _, ep := range []string{"plan", "draft", "review", "repair", "approve"} {
		s, b := testutil.MustPostJSON(t, base, "/api/workflows/"+ep, map[string]any{
			"bookId": 1, "chapterNo": 1, "provider": "mock",
		})
		require.Equal(t, http.StatusOK, s, "%s failed: %s", ep, string(b))
	}

	status, body = testutil.MustGet(t, base, "/api/books/1/chapters/1/lifecycle")
	require.Equal(t, http.StatusOK, status)
	var life struct{ Status string }
	testutil.MustUnmarshal(t, body, &life)
	require.Equal(t, "approved", life.Status)
}

// TestEmbeddingDisabled 默认 PLANNING_RETRIEVAL_EMBEDDING_PROVIDER=none,
// /embeddings/refresh 应返回 409。
func TestEmbeddingRefresh_DisabledByDefault(t *testing.T) {
	srv, _ := testutil.NewTestServer(t, nil)
	// 先建一本书
	status, _ := testutil.MustPostJSON(t, srv.URL, "/api/books", map[string]any{"title": "T"})
	require.Equal(t, http.StatusCreated, status)
	status, _ = testutil.MustPostJSON(t, srv.URL, "/api/books/1/embeddings/refresh", map[string]any{})
	require.Equal(t, http.StatusConflict, status)
}

// TestEmbeddingRefresh_HashProvider:启用 hash provider,书没设定时刷新应返回空 map。
func TestEmbeddingRefresh_HashProvider(t *testing.T) {
	srv, _ := testutil.NewTestServer(t, map[string]string{
		"PLANNING_RETRIEVAL_EMBEDDING_PROVIDER":   "hash",
		"PLANNING_RETRIEVAL_EMBEDDING_SEARCH_MODE": "basic",
	})
	status, _ := testutil.MustPostJSON(t, srv.URL, "/api/books", map[string]any{"title": "T"})
	require.Equal(t, http.StatusCreated, status)
	status, _ = testutil.MustPostJSON(t, srv.URL, "/api/books/1/characters", map[string]any{
		"name": "林夜", "keywords": "林夜",
	})
	require.Equal(t, http.StatusCreated, status)

	status, body := testutil.MustPostJSON(t, srv.URL, "/api/books/1/embeddings/refresh", map[string]any{})
	require.Equal(t, http.StatusOK, status, string(body))
	var resp struct {
		Refreshed map[string]int
		Model     string
	}
	testutil.MustUnmarshal(t, body, &resp)
	require.Equal(t, 1, resp.Refreshed["character"])
	require.Equal(t, "deterministic-hash-32", resp.Model)
}
