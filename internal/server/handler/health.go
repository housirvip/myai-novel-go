package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"myai-novel-go/internal/config"
	"myai-novel-go/internal/domain/shared"
	"myai-novel-go/internal/server/middleware"
)

const buildVersion = "0.1.0"

// RegisterHealth 暴露三类健康端点:
//   - /health        liveness:进程能响应即 200,docker HEALTHCHECK / k8s livenessProbe 用
//   - /healthz/ready readiness:进一步检查 db 连通性,k8s readinessProbe 用
//   - /api/meta      运行时元信息(版本 / provider / webui)
func RegisterHealth(r *gin.RouterGroup, cfg *config.Config, gdb *gorm.DB) {
	r.GET("/health", func(c *gin.Context) {
		ok(c, gin.H{"status": "ok"})
	})
	r.GET("/healthz/ready", func(c *gin.Context) {
		sqlDB, err := gdb.DB()
		if err != nil {
			middleware.AbortWithError(c, shared.NewAppErrorWithDetails(http.StatusServiceUnavailable, "service_unavailable", "database not ready", err.Error()))
			return
		}
		if err := sqlDB.PingContext(c.Request.Context()); err != nil {
			middleware.AbortWithError(c, shared.NewAppErrorWithDetails(http.StatusServiceUnavailable, "service_unavailable", "database not ready", err.Error()))
			return
		}
		ok(c, gin.H{"status": "ready"})
	})
	r.GET("/api/meta", func(c *gin.Context) {
		ok(c, gin.H{
			"name":        "myai-novel-go",
			"version":     buildVersion,
			"nodeEnv":     cfg.NodeEnv,
			"llmProvider": cfg.LLMProvider,
			"webui": gin.H{
				"enabled":  cfg.WebUIDistPath != "",
				"distPath": cfg.WebUIDistPath,
			},
		})
	})
}
