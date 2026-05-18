package handler

import (
	"github.com/gin-gonic/gin"

	"myai-novel-go/internal/domain/character"
	"myai-novel-go/internal/domain/faction"
	"myai-novel-go/internal/domain/item"
	"myai-novel-go/internal/domain/outline"
	"myai-novel-go/internal/domain/relation"
	"myai-novel-go/internal/domain/story_hook"
	"myai-novel-go/internal/domain/world_setting"
	"myai-novel-go/internal/server/middleware"
)

type ResourceHandlers struct {
	Outline      *outline.Service
	WorldSetting *world_setting.Service
	Character    *character.Service
	Faction      *faction.Service
	Relation     *relation.Service
	Item         *item.Service
	StoryHook    *story_hook.Service
}

func (h *ResourceHandlers) Register(r *gin.RouterGroup) {
	h.registerOutlines(r)
	h.registerWorldSettings(r)
	h.registerCharacters(r)
	h.registerFactions(r)
	h.registerRelations(r)
	h.registerItems(r)
	h.registerHooks(r)
}

// ============== outlines ==============

func (h *ResourceHandlers) registerOutlines(r *gin.RouterGroup) {
	r.GET("/api/books/:bookId/outlines", h.outlineList)
	r.POST("/api/books/:bookId/outlines", h.outlineCreate)
	r.GET("/api/books/:bookId/outlines/:id", h.outlineGet)
	r.PATCH("/api/books/:bookId/outlines/:id", h.outlineUpdate)
	r.DELETE("/api/books/:bookId/outlines/:id", h.outlineDelete)
}

func (h *ResourceHandlers) outlineList(c *gin.Context) {
	bookID, err := middleware.ParseInt64Param(c, "bookId")
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	limit, err := parseLimit(c, 50, 200)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	rows, err := h.Outline.List(c.Request.Context(), bookID, limit)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, rows)
}

func (h *ResourceHandlers) outlineCreate(c *gin.Context) {
	bookID, err := middleware.ParseInt64Param(c, "bookId")
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	var in outline.CreateInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	in.BookID = bookID
	row, err := h.Outline.Create(c.Request.Context(), in)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	created(c, row)
}

func (h *ResourceHandlers) outlineGet(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	id, _ := middleware.ParseInt64Param(c, "id")
	row, err := h.Outline.Get(c.Request.Context(), bookID, id)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, row)
}

func (h *ResourceHandlers) outlineUpdate(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	id, _ := middleware.ParseInt64Param(c, "id")
	var in outline.UpdateInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	row, err := h.Outline.Update(c.Request.Context(), bookID, id, in)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, row)
}

func (h *ResourceHandlers) outlineDelete(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	id, _ := middleware.ParseInt64Param(c, "id")
	if err := h.Outline.Remove(c.Request.Context(), bookID, id); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, gin.H{"ok": true})
}

// ============== world settings ==============

func (h *ResourceHandlers) registerWorldSettings(r *gin.RouterGroup) {
	r.GET("/api/books/:bookId/world-settings", h.wsList)
	r.POST("/api/books/:bookId/world-settings", h.wsCreate)
	r.GET("/api/books/:bookId/world-settings/:id", h.wsGet)
	r.PATCH("/api/books/:bookId/world-settings/:id", h.wsUpdate)
	r.DELETE("/api/books/:bookId/world-settings/:id", h.wsDelete)
}

func (h *ResourceHandlers) wsList(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	limit, err := parseLimit(c, 50, 200)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	rows, err := h.WorldSetting.List(c.Request.Context(), bookID, limit, c.Query("status"))
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, rows)
}

func (h *ResourceHandlers) wsCreate(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	var in world_setting.CreateInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	in.BookID = bookID
	row, err := h.WorldSetting.Create(c.Request.Context(), in)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	created(c, row)
}

func (h *ResourceHandlers) wsGet(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	id, _ := middleware.ParseInt64Param(c, "id")
	row, err := h.WorldSetting.Get(c.Request.Context(), bookID, id)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, row)
}

func (h *ResourceHandlers) wsUpdate(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	id, _ := middleware.ParseInt64Param(c, "id")
	var in world_setting.UpdateInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	row, err := h.WorldSetting.Update(c.Request.Context(), bookID, id, in)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, row)
}

func (h *ResourceHandlers) wsDelete(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	id, _ := middleware.ParseInt64Param(c, "id")
	if err := h.WorldSetting.Remove(c.Request.Context(), bookID, id); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, gin.H{"ok": true})
}

// ============== characters ==============

func (h *ResourceHandlers) registerCharacters(r *gin.RouterGroup) {
	r.GET("/api/books/:bookId/characters", h.charList)
	r.POST("/api/books/:bookId/characters", h.charCreate)
	r.GET("/api/books/:bookId/characters/:id", h.charGet)
	r.PATCH("/api/books/:bookId/characters/:id", h.charUpdate)
	r.DELETE("/api/books/:bookId/characters/:id", h.charDelete)
}

func (h *ResourceHandlers) charList(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	limit, err := parseLimit(c, 50, 500)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	rows, err := h.Character.List(c.Request.Context(), bookID, limit, c.Query("status"))
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, rows)
}

func (h *ResourceHandlers) charCreate(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	var in character.CreateInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	in.BookID = bookID
	row, err := h.Character.Create(c.Request.Context(), in)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	created(c, row)
}

func (h *ResourceHandlers) charGet(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	id, _ := middleware.ParseInt64Param(c, "id")
	row, err := h.Character.Get(c.Request.Context(), bookID, id)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, row)
}

func (h *ResourceHandlers) charUpdate(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	id, _ := middleware.ParseInt64Param(c, "id")
	var in character.UpdateInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	row, err := h.Character.Update(c.Request.Context(), bookID, id, in)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, row)
}

func (h *ResourceHandlers) charDelete(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	id, _ := middleware.ParseInt64Param(c, "id")
	if err := h.Character.Remove(c.Request.Context(), bookID, id); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, gin.H{"ok": true})
}

// ============== factions ==============

func (h *ResourceHandlers) registerFactions(r *gin.RouterGroup) {
	r.GET("/api/books/:bookId/factions", h.facList)
	r.POST("/api/books/:bookId/factions", h.facCreate)
	r.GET("/api/books/:bookId/factions/:id", h.facGet)
	r.PATCH("/api/books/:bookId/factions/:id", h.facUpdate)
	r.DELETE("/api/books/:bookId/factions/:id", h.facDelete)
}

func (h *ResourceHandlers) facList(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	limit, err := parseLimit(c, 50, 200)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	rows, err := h.Faction.List(c.Request.Context(), bookID, limit, c.Query("status"))
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, rows)
}

func (h *ResourceHandlers) facCreate(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	var in faction.CreateInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	in.BookID = bookID
	row, err := h.Faction.Create(c.Request.Context(), in)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	created(c, row)
}

func (h *ResourceHandlers) facGet(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	id, _ := middleware.ParseInt64Param(c, "id")
	row, err := h.Faction.Get(c.Request.Context(), bookID, id)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, row)
}

func (h *ResourceHandlers) facUpdate(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	id, _ := middleware.ParseInt64Param(c, "id")
	var in faction.UpdateInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	row, err := h.Faction.Update(c.Request.Context(), bookID, id, in)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, row)
}

func (h *ResourceHandlers) facDelete(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	id, _ := middleware.ParseInt64Param(c, "id")
	if err := h.Faction.Remove(c.Request.Context(), bookID, id); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, gin.H{"ok": true})
}

// ============== relations ==============

func (h *ResourceHandlers) registerRelations(r *gin.RouterGroup) {
	r.GET("/api/books/:bookId/relations", h.relList)
	r.POST("/api/books/:bookId/relations", h.relCreate)
	r.GET("/api/books/:bookId/relations/:id", h.relGet)
	r.PATCH("/api/books/:bookId/relations/:id", h.relUpdate)
	r.DELETE("/api/books/:bookId/relations/:id", h.relDelete)
}

func (h *ResourceHandlers) relList(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	limit, err := parseLimit(c, 50, 200)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	rows, err := h.Relation.List(c.Request.Context(), bookID, limit)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, rows)
}

func (h *ResourceHandlers) relCreate(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	var in relation.CreateInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	in.BookID = bookID
	row, err := h.Relation.Create(c.Request.Context(), in)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	created(c, row)
}

func (h *ResourceHandlers) relGet(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	id, _ := middleware.ParseInt64Param(c, "id")
	row, err := h.Relation.Get(c.Request.Context(), bookID, id)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, row)
}

func (h *ResourceHandlers) relUpdate(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	id, _ := middleware.ParseInt64Param(c, "id")
	var in relation.UpdateInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	row, err := h.Relation.Update(c.Request.Context(), bookID, id, in)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, row)
}

func (h *ResourceHandlers) relDelete(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	id, _ := middleware.ParseInt64Param(c, "id")
	if err := h.Relation.Remove(c.Request.Context(), bookID, id); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, gin.H{"ok": true})
}

// ============== items ==============

func (h *ResourceHandlers) registerItems(r *gin.RouterGroup) {
	r.GET("/api/books/:bookId/items", h.itemList)
	r.POST("/api/books/:bookId/items", h.itemCreate)
	r.GET("/api/books/:bookId/items/:id", h.itemGet)
	r.PATCH("/api/books/:bookId/items/:id", h.itemUpdate)
	r.DELETE("/api/books/:bookId/items/:id", h.itemDelete)
}

func (h *ResourceHandlers) itemList(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	limit, err := parseLimit(c, 50, 200)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	rows, err := h.Item.List(c.Request.Context(), bookID, limit)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, rows)
}

func (h *ResourceHandlers) itemCreate(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	var in item.CreateInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	in.BookID = bookID
	row, err := h.Item.Create(c.Request.Context(), in)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	created(c, row)
}

func (h *ResourceHandlers) itemGet(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	id, _ := middleware.ParseInt64Param(c, "id")
	row, err := h.Item.Get(c.Request.Context(), bookID, id)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, row)
}

func (h *ResourceHandlers) itemUpdate(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	id, _ := middleware.ParseInt64Param(c, "id")
	var in item.UpdateInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	row, err := h.Item.Update(c.Request.Context(), bookID, id, in)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, row)
}

func (h *ResourceHandlers) itemDelete(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	id, _ := middleware.ParseInt64Param(c, "id")
	if err := h.Item.Remove(c.Request.Context(), bookID, id); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, gin.H{"ok": true})
}

// ============== story hooks ==============

func (h *ResourceHandlers) registerHooks(r *gin.RouterGroup) {
	r.GET("/api/books/:bookId/hooks", h.hookList)
	r.POST("/api/books/:bookId/hooks", h.hookCreate)
	r.GET("/api/books/:bookId/hooks/:id", h.hookGet)
	r.PATCH("/api/books/:bookId/hooks/:id", h.hookUpdate)
	r.DELETE("/api/books/:bookId/hooks/:id", h.hookDelete)
}

func (h *ResourceHandlers) hookList(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	limit, err := parseLimit(c, 50, 200)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	rows, err := h.StoryHook.List(c.Request.Context(), bookID, limit, c.Query("status"))
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, rows)
}

func (h *ResourceHandlers) hookCreate(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	var in story_hook.CreateInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	in.BookID = bookID
	row, err := h.StoryHook.Create(c.Request.Context(), in)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	created(c, row)
}

func (h *ResourceHandlers) hookGet(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	id, _ := middleware.ParseInt64Param(c, "id")
	row, err := h.StoryHook.Get(c.Request.Context(), bookID, id)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, row)
}

func (h *ResourceHandlers) hookUpdate(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	id, _ := middleware.ParseInt64Param(c, "id")
	var in story_hook.UpdateInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	row, err := h.StoryHook.Update(c.Request.Context(), bookID, id, in)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, row)
}

func (h *ResourceHandlers) hookDelete(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	id, _ := middleware.ParseInt64Param(c, "id")
	if err := h.StoryHook.Remove(c.Request.Context(), bookID, id); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, gin.H{"ok": true})
}
