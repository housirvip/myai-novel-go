package handler

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"myai-novel-go/internal/domain/chapter"
	"myai-novel-go/internal/domain/shared"
	"myai-novel-go/internal/server/middleware"
)

type ChapterHandler struct {
	svc *chapter.Service
}

func NewChapterHandler(svc *chapter.Service) *ChapterHandler { return &ChapterHandler{svc: svc} }

func (h *ChapterHandler) Register(r *gin.RouterGroup) {
	r.GET("/api/books/:bookId/chapters", h.list)
	r.POST("/api/books/:bookId/chapters", h.create)
	r.GET("/api/books/:bookId/chapters/:chapterNo", h.get)
	r.PATCH("/api/books/:bookId/chapters/:chapterNo", h.update)
	r.DELETE("/api/books/:bookId/chapters/:chapterNo", h.delete)
	r.GET("/api/books/:bookId/chapters/:chapterNo/stages/:stage", h.getStage)
	r.GET("/api/books/:bookId/chapters/:chapterNo/stages/:stage/history", h.stageHistory)
	r.PUT("/api/books/:bookId/chapters/:chapterNo/stages/:stage", h.writeStage)
	r.GET("/api/books/:bookId/chapters/:chapterNo/workflow-state", h.workflowState)
	r.GET("/api/books/:bookId/chapters/:chapterNo/lifecycle", h.workflowState)
	r.GET("/api/books/:bookId/chapters/:chapterNo/stages/:stage/export", h.exportStage)
	r.POST("/api/books/:bookId/chapters/:chapterNo/stages/:stage/import", h.importStage)
}

func (h *ChapterHandler) list(c *gin.Context) {
	bookID, err := middleware.ParseInt64Param(c, "bookId")
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	limit, err := parseLimit(c, 50, 500)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	rows, err := h.svc.List(c.Request.Context(), bookID, limit, c.Query("status"))
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, rows)
}

func (h *ChapterHandler) create(c *gin.Context) {
	bookID, err := middleware.ParseInt64Param(c, "bookId")
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	var in chapter.CreateInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	in.BookID = bookID
	row, err := h.svc.Create(c.Request.Context(), in)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	created(c, row)
}

func (h *ChapterHandler) get(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	cn, err := middleware.ParseIntParam(c, "chapterNo")
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	row, err := h.svc.Get(c.Request.Context(), bookID, cn)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, row)
}

func (h *ChapterHandler) update(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	cn, err := middleware.ParseIntParam(c, "chapterNo")
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	var in chapter.UpdateInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	row, err := h.svc.Update(c.Request.Context(), bookID, cn, in)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, row)
}

func (h *ChapterHandler) delete(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	cn, _ := middleware.ParseIntParam(c, "chapterNo")
	if err := h.svc.Remove(c.Request.Context(), bookID, cn); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, gin.H{"ok": true})
}

func (h *ChapterHandler) getStage(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	cn, _ := middleware.ParseIntParam(c, "chapterNo")
	stage := c.Param("stage")
	if !validStage(stage) {
		middleware.AbortWithError(c, shared.BadRequest("invalid stage: "+stage))
		return
	}
	v, err := h.svc.GetStage(c.Request.Context(), bookID, cn, stage)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, v)
}

func (h *ChapterHandler) stageHistory(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	cn, _ := middleware.ParseIntParam(c, "chapterNo")
	stage := c.Param("stage")
	if !validStage(stage) {
		middleware.AbortWithError(c, shared.BadRequest("invalid stage: "+stage))
		return
	}
	limit := 20
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 || n > 50 {
			middleware.AbortWithError(c, shared.BadRequest("invalid limit"))
			return
		}
		limit = n
	}
	rows, err := h.svc.ListStageHistory(c.Request.Context(), bookID, cn, stage, limit)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, rows)
}

func (h *ChapterHandler) writeStage(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	cn, _ := middleware.ParseIntParam(c, "chapterNo")
	stage := c.Param("stage")
	if !validWritableStage(stage) {
		middleware.AbortWithError(c, shared.BadRequest("stage not writable: "+stage))
		return
	}
	var in chapter.WriteStageInput
	if err := c.ShouldBindJSON(&in); err != nil {
		middleware.AbortWithError(c, shared.BadRequestDetails("invalid request body", err.Error()))
		return
	}
	in.BookID = bookID
	in.ChapterNo = cn
	in.Stage = stage
	v, err := h.svc.WriteStage(c.Request.Context(), in)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, v)
}

func (h *ChapterHandler) workflowState(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	cn, _ := middleware.ParseIntParam(c, "chapterNo")
	v, err := h.svc.GetWorkflowState(c.Request.Context(), bookID, cn)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, v)
}

func (h *ChapterHandler) exportStage(c *gin.Context) {
	bookID, err := middleware.ParseInt64Param(c, "bookId")
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	cn, err := middleware.ParseIntParam(c, "chapterNo")
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	stage := c.Param("stage")
	if !validStage(stage) {
		middleware.AbortWithError(c, shared.BadRequest("unsupported stage: "+stage))
		return
	}
	md, _, err := h.svc.ExportStage(c.Request.Context(), bookID, cn, stage)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	filename := fmt.Sprintf("chapter-%04d-%s.md", cn, stage)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "text/markdown; charset=utf-8", []byte(md))
}

type importStageRequest struct {
	Markdown string `json:"markdown"`
	Force    bool   `json:"force"`
}

func (h *ChapterHandler) importStage(c *gin.Context) {
	bookID, err := middleware.ParseInt64Param(c, "bookId")
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	cn, err := middleware.ParseIntParam(c, "chapterNo")
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	stage := c.Param("stage")
	if !validWritableStage(stage) {
		middleware.AbortWithError(c, shared.BadRequest("stage not importable: "+stage))
		return
	}

	var raw string
	var force bool
	ct := c.GetHeader("Content-Type")
	switch {
	case ct == "" || ct == "text/markdown" || ct == "text/markdown; charset=utf-8" || ct == "text/plain":
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			middleware.AbortWithError(c, shared.BadRequest("failed to read body"))
			return
		}
		raw = string(body)
		force = c.Query("force") == "true" || c.Query("force") == "1"
	default:
		var in importStageRequest
		if err := bind(c, &in); err != nil {
			middleware.AbortWithError(c, err)
			return
		}
		raw = in.Markdown
		force = in.Force
	}
	if raw == "" {
		middleware.AbortWithError(c, shared.BadRequest("empty markdown body"))
		return
	}
	entry, err := h.svc.ImportStage(c.Request.Context(), bookID, cn, stage, raw, force)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	created(c, entry)
}

func validStage(s string) bool {
	return s == shared.StageNamePlan || s == shared.StageNameDraft || s == shared.StageNameReview || s == shared.StageNameFinal
}

func validWritableStage(s string) bool {
	return s == shared.StageNamePlan || s == shared.StageNameDraft || s == shared.StageNameFinal
}
