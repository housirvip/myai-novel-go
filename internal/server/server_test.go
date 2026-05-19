package server_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"myai-novel-go/internal/testutil"
)

func TestServerSmoke(t *testing.T) {
	srv, _ := testutil.NewTestServer(t, nil)
	base := srv.URL

	status, _ := testutil.MustGet(t, base, "/health")
	require.Equal(t, http.StatusOK, status)

	status, body := testutil.MustPostJSON(t, base, "/api/books", map[string]any{
		"title": "测试书", "targetChapterCount": 100,
	})
	require.Equal(t, http.StatusCreated, status, string(body))

	status, body = testutil.MustPostJSON(t, base, "/api/books/1/chapters", map[string]any{
		"chapterNo": 1, "title": "起势",
	})
	require.Equal(t, http.StatusCreated, status, string(body))

	for _, p := range []map[string]any{
		{"path": "/api/books/1/characters", "body": map[string]any{"name": "林夜", "keywords": "林夜"}},
		{"path": "/api/books/1/items", "body": map[string]any{"name": "黑铁令", "ownerType": "none", "keywords": "黑铁令"}},
		{"path": "/api/books/1/hooks", "body": map[string]any{"title": "黑铁令异常", "hookType": "mystery", "keywords": "黑铁令"}},
	} {
		s, b := testutil.MustPostJSON(t, base, p["path"].(string), p["body"])
		require.Equal(t, http.StatusCreated, s, string(b))
	}

	status, body = testutil.MustPostJSON(t, base, "/api/workflows/plan/tasks", map[string]any{
		"bookId": 1, "chapterNo": 1, "provider": "mock",
	})
	require.Equal(t, http.StatusAccepted, status, string(body))

	require.Eventually(t, func() bool {
		status, body := testutil.MustGet(t, base, "/api/workflow-tasks/1")
		if status != http.StatusOK {
			return false
		}
		var task struct{ Data struct{ Status string } }
		testutil.MustUnmarshal(t, body, &task)
		return task.Data.Status == "succeeded"
	}, 10*time.Second, 200*time.Millisecond)
}

func TestEmbeddingRefresh_DisabledByDefault(t *testing.T) {
	srv, _ := testutil.NewTestServer(t, nil)
	status, _ := testutil.MustPostJSON(t, srv.URL, "/api/books", map[string]any{"title": "T"})
	require.Equal(t, http.StatusCreated, status)
	status, _ = testutil.MustPostJSON(t, srv.URL, "/api/books/1/embeddings/refresh", map[string]any{})
	require.Equal(t, http.StatusConflict, status)
}

func TestEmbeddingRefresh_HashProvider(t *testing.T) {
	srv, _ := testutil.NewTestServer(t, map[string]string{
		"PLANNING_RETRIEVAL_EMBEDDING_PROVIDER":    "hash",
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
		Data struct {
			Refreshed map[string]int
			Model     string
		}
	}
	testutil.MustUnmarshal(t, body, &resp)
	require.Equal(t, 1, resp.Data.Refreshed["character"])
	require.Equal(t, "deterministic-hash-32", resp.Data.Model)
}
