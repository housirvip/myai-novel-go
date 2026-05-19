package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"myai-novel-go/internal/config"
	"myai-novel-go/internal/server/handler"
	"myai-novel-go/internal/server/middleware"
)

func TestHealthzReady_ErrorEnvelopeWhenDBUnavailable(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(middleware.ErrorResponder())
	gdb, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := gdb.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
	handler.RegisterHealth(&r.RouterGroup, &config.Config{}, gdb)

	req := httptest.NewRequest(http.MethodGet, "/healthz/ready", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	require.Equal(t, http.StatusServiceUnavailable, resp.Code)
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Details string `json:"details"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.Equal(t, "service_unavailable", body.Error.Code)
	require.Equal(t, "database not ready", body.Error.Message)
	require.NotEmpty(t, body.Error.Details)
}
