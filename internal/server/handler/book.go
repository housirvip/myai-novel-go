package handler

import (
	"github.com/gin-gonic/gin"

	"myai-novel-go/internal/domain/book"
	"myai-novel-go/internal/server/middleware"
)

type BookHandler struct {
	svc *book.Service
}

func NewBookHandler(svc *book.Service) *BookHandler { return &BookHandler{svc: svc} }

func (h *BookHandler) Register(r *gin.RouterGroup) {
	r.GET("/api/books", h.list)
	r.POST("/api/books", h.create)
	r.GET("/api/books/:bookId", h.get)
	r.PATCH("/api/books/:bookId", h.update)
	r.DELETE("/api/books/:bookId", h.delete)
}

func (h *BookHandler) list(c *gin.Context) {
	limit, err := parseLimit(c, 50, 200)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	actor := middleware.GetActor(c)
	var ownerUserID *int64
	if actor.Kind == middleware.ActorUser {
		uid := actor.UserID
		ownerUserID = &uid
	}
	rows, err := h.svc.ListForActor(c.Request.Context(), ownerUserID, limit)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, rows)
}

func (h *BookHandler) create(c *gin.Context) {
	var in book.CreateInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	// 登录用户创建的 book 自动归属本人;匿名仍可创建未归属的"系统级"book
	actor := middleware.GetActor(c)
	if actor.Kind == middleware.ActorUser {
		uid := actor.UserID
		in.OwnerUserID = &uid
	}
	row, err := h.svc.Create(c.Request.Context(), in)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	created(c, row)
}

func (h *BookHandler) get(c *gin.Context) {
	id, err := middleware.ParseInt64Param(c, "bookId")
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	row, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, row)
}

func (h *BookHandler) update(c *gin.Context) {
	id, err := middleware.ParseInt64Param(c, "bookId")
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	var in book.UpdateInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	row, err := h.svc.Update(c.Request.Context(), id, in)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, row)
}

func (h *BookHandler) delete(c *gin.Context) {
	id, err := middleware.ParseInt64Param(c, "bookId")
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	if err := h.svc.Remove(c.Request.Context(), id); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, gin.H{"ok": true})
}
