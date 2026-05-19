package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"myai-novel-go/internal/config"
	"myai-novel-go/internal/db"
	"myai-novel-go/internal/logger"
	"myai-novel-go/internal/server"
)

func NewTestServer(t *testing.T, envOverrides map[string]string) (*httptest.Server, *config.Config) {
	t.Helper()
	t.Setenv("DB_CLIENT", "sqlite")
	t.Setenv("DB_SQLITE_PATH", t.TempDir()+"/test.db")
	t.Setenv("LLM_PROVIDER", "mock")
	t.Setenv("MOCK_LLM_MODE", "echo")
	t.Setenv("LOG_LEVEL", "warn")
	t.Setenv("LOG_FORMAT", "json")
	t.Setenv("LOG_DIR", t.TempDir())
	t.Setenv("WORKFLOW_MAX_CONCURRENCY", "2")
	for k, v := range envOverrides {
		t.Setenv(k, v)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	zlog, err := logger.New(cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		t.Fatalf("logger.New: %v", err)
	}
	gdb, err := db.Open(cfg)
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	if err := db.Migrate(gdb); err != nil {
		t.Fatalf("db.Migrate: %v", err)
	}
	srv := server.New(cfg, zlog, gdb)
	rootCtx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		srv.Shutdown(context.Background())
	})
	srv.Start(rootCtx)
	httpSrv := httptest.NewServer(srv.Handler())
	t.Cleanup(httpSrv.Close)
	return httpSrv, cfg
}

func MustPostJSON(t *testing.T, base, path string, payload any) (int, []byte) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	resp, err := http.Post(base+path, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post %s: %v", path, err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, out
}

func MustGet(t *testing.T, base, path string) (int, []byte) {
	t.Helper()
	resp, err := http.Get(base + path)
	if err != nil {
		t.Fatalf("get %s: %v", path, err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, out
}

func MustUnmarshal(t *testing.T, raw []byte, into any) {
	t.Helper()
	if err := json.Unmarshal(raw, into); err != nil {
		t.Fatalf("unmarshal %q: %v", string(raw), err)
	}
}
