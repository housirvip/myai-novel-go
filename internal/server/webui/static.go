package webui

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Register 在 cfg.WebUIDistPath 非空且为目录时把它挂到 /app/* 下,
// 并对所有 /app 子路径走 SPA fallback(回到 index.html)。
// /api/* 与其它路由不受影响。
func Register(r *gin.Engine, distPath string, logger *zap.Logger) {
	distPath = strings.TrimSpace(distPath)
	if distPath == "" {
		return
	}
	abs, err := filepath.Abs(distPath)
	if err != nil {
		logger.Warn("webui.dist_path.invalid", zap.String("path", distPath), zap.Error(err))
		return
	}
	stat, err := os.Stat(abs)
	if err != nil || !stat.IsDir() {
		logger.Warn("webui.dist_path.not_dir", zap.String("path", abs))
		return
	}
	indexPath := filepath.Join(abs, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		logger.Warn("webui.index_missing", zap.String("path", indexPath))
		return
	}
	r.StaticFS("/app", http.Dir(abs))
	r.GET("/", func(c *gin.Context) { c.Redirect(http.StatusFound, "/app/") })
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/app") || path == "/" {
			c.File(indexPath)
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "not_found", "message": "Route not found: " + c.Request.Method + " " + path}})
	})
	logger.Info("webui.mounted", zap.String("dist", abs))
}
