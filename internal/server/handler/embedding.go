package handler

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"myai-novel-go/internal/domain/planning"
	"myai-novel-go/internal/domain/shared"
	"myai-novel-go/internal/server/middleware"
)

type EmbeddingHandler struct {
	db        *gorm.DB
	retrieval *planning.RetrievalService
}

func NewEmbeddingHandler(db *gorm.DB, retrieval *planning.RetrievalService) *EmbeddingHandler {
	return &EmbeddingHandler{db: db, retrieval: retrieval}
}

func (h *EmbeddingHandler) Register(r *gin.RouterGroup) {
	r.POST("/api/books/:bookId/embeddings/refresh", h.refresh)
}

// refresh 触发同步全量刷新。当 PLANNING_RETRIEVAL_EMBEDDING_PROVIDER=none 时直接 409。
func (h *EmbeddingHandler) refresh(c *gin.Context) {
	bookID, err := middleware.ParseInt64Param(c, "bookId")
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	client := h.retrieval.EmbeddingClient()
	store := h.retrieval.EmbeddingStore()
	if client == nil || store == nil {
		middleware.AbortWithError(c, shared.Conflict("embedding provider disabled (PLANNING_RETRIEVAL_EMBEDDING_PROVIDER=none)", nil))
		return
	}
	svc := planning.NewRefreshService(h.db, client, store)
	counts, err := svc.Refresh(c.Request.Context(), bookID)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, gin.H{"refreshed": counts, "model": client.Model()})
}
