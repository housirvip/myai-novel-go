package handler

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"myai-novel-go/internal/config"
	"myai-novel-go/internal/domain/auth"
	"myai-novel-go/internal/server/middleware"
)

type AuthHandler struct {
	cfg *config.Config
	svc *auth.Service
}

func NewAuthHandler(cfg *config.Config, svc *auth.Service) *AuthHandler {
	return &AuthHandler{cfg: cfg, svc: svc}
}

func (h *AuthHandler) Register(r *gin.RouterGroup) {
	r.POST("/api/auth/register", h.register)
	r.POST("/api/auth/login", h.login)
	r.POST("/api/auth/logout", h.logout)
	r.GET("/api/auth/session", h.session)
}

type registerRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	DisplayName string `json:"displayName" binding:"required,min=1"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=1"`
}

func (h *AuthHandler) register(c *gin.Context) {
	var in registerRequest
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	res, err := h.svc.Register(c.Request.Context(), in.Email, in.Password, in.DisplayName)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	h.writeSessionCookie(c, res.SessionToken)
	created(c, res.User)
}

func (h *AuthHandler) login(c *gin.Context) {
	var in loginRequest
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	res, err := h.svc.Login(c.Request.Context(), in.Email, in.Password)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	h.writeSessionCookie(c, res.SessionToken)
	ok(c, res.User)
}

func (h *AuthHandler) logout(c *gin.Context) {
	raw := readSessionCookie(c.GetHeader("Cookie"), h.cfg.AuthCookieName)
	token := h.svc.VerifySignedSessionToken(raw)
	if token != "" {
		_ = h.svc.Logout(c.Request.Context(), token)
	}
	h.clearSessionCookie(c)
	ok(c, gin.H{"ok": true})
}

func (h *AuthHandler) session(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	ok(c, gin.H{"user": user})
}

// writeSessionCookie 写 Set-Cookie:value=signed_token, HttpOnly, Path=/, SameSite=Lax。
func (h *AuthHandler) writeSessionCookie(c *gin.Context, rawToken string) {
	signed := h.svc.SignSessionToken(rawToken)
	maxAge := h.cfg.AuthSessionTTLHours * 3600
	parts := []string{
		fmt.Sprintf("%s=%s", h.cfg.AuthCookieName, signed),
		"Path=/",
		"HttpOnly",
		"SameSite=Lax",
		fmt.Sprintf("Max-Age=%d", maxAge),
	}
	if h.cfg.AuthCookieSecure {
		parts = append(parts, "Secure")
	}
	c.Writer.Header().Add("Set-Cookie", strings.Join(parts, "; "))
}

func (h *AuthHandler) clearSessionCookie(c *gin.Context) {
	parts := []string{
		fmt.Sprintf("%s=", h.cfg.AuthCookieName),
		"Path=/",
		"HttpOnly",
		"SameSite=Lax",
		"Max-Age=0",
	}
	if h.cfg.AuthCookieSecure {
		parts = append(parts, "Secure")
	}
	c.Writer.Header().Add("Set-Cookie", strings.Join(parts, "; "))
}

func readSessionCookie(headerValue, name string) string {
	if headerValue == "" {
		return ""
	}
	for _, segment := range strings.Split(headerValue, ";") {
		seg := strings.TrimSpace(segment)
		if strings.HasPrefix(seg, name+"=") {
			return strings.TrimPrefix(seg, name+"=")
		}
	}
	return ""
}

