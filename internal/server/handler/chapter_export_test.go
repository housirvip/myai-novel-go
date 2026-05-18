package handler_test

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"myai-novel-go/internal/testutil"
)

// TestChapterExportImport_RoundTrip 走完整流程:
// 建书+章 → 跑 plan/draft → export draft → 修改文件 → import 回 draft。
func TestChapterExportImport_RoundTrip(t *testing.T) {
	srv, _ := testutil.NewTestServer(t, nil)
	base := srv.URL

	status, _ := testutil.MustPostJSON(t, base, "/api/books", map[string]any{"title": "T", "targetChapterCount": 10})
	require.Equal(t, http.StatusCreated, status)
	status, _ = testutil.MustPostJSON(t, base, "/api/books/1/chapters", map[string]any{"chapterNo": 1, "title": "首"})
	require.Equal(t, http.StatusCreated, status)
	for _, ep := range []string{"plan", "draft"} {
		s, b := testutil.MustPostJSON(t, base, "/api/workflows/"+ep, map[string]any{
			"bookId": 1, "chapterNo": 1, "provider": "mock",
		})
		require.Equal(t, http.StatusOK, s, "%s: %s", ep, string(b))
	}

	resp, err := http.Get(base + "/api/books/1/chapters/1/stages/draft/export")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, resp.Header.Get("Content-Type"), "text/markdown")
	rawBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Contains(t, string(rawBody), "stage: draft")
	require.Contains(t, string(rawBody), "## Content")

	// 修改正文,确认 import 回去是新版本
	modified := strings.Replace(string(rawBody), "## Content", "## Content\n\n--人工补充--", 1)
	importResp, err := http.Post(
		base+"/api/books/1/chapters/1/stages/draft/import",
		"text/markdown",
		bytes.NewReader([]byte(modified)),
	)
	require.NoError(t, err)
	defer importResp.Body.Close()
	require.Equal(t, http.StatusCreated, importResp.StatusCode)
}
