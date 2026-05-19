package handler

import (
	"github.com/gin-gonic/gin"

	"myai-novel-go/internal/config"
	usersettings "myai-novel-go/internal/domain/user_settings"
	"myai-novel-go/internal/server/middleware"
)

type UserSettingsHandler struct {
	cfg *config.Config
	svc *usersettings.Service
}

func NewUserSettingsHandler(cfg *config.Config, svc *usersettings.Service) *UserSettingsHandler {
	return &UserSettingsHandler{cfg: cfg, svc: svc}
}

func (h *UserSettingsHandler) Register(r *gin.RouterGroup) {
	g := r.Group("", middleware.RequireUser())
	g.GET("/api/user-settings/runtime", h.get)
	g.PUT("/api/user-settings/runtime", h.update)
	g.DELETE("/api/user-settings/runtime", h.clear)
}

func (h *UserSettingsHandler) get(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	out, err := h.svc.Get(c.Request.Context(), user.ID)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, buildUserRuntimeSettingsView(h.cfg, out))
}

func (h *UserSettingsHandler) update(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	var in usersettings.UpdateInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	out, err := h.svc.Update(c.Request.Context(), user.ID, in)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, buildUserRuntimeSettingsView(h.cfg, out))
}

func (h *UserSettingsHandler) clear(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if err := h.svc.Clear(c.Request.Context(), user.ID); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, buildUserRuntimeSettingsView(h.cfg, &usersettings.RuntimeOverrides{}))
}
