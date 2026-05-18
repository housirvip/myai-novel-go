package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"myai-novel-go/internal/domain/auth"
	"myai-novel-go/internal/domain/shared"
)

const (
	CtxActorKey       = "actor"
	CtxCurrentUserKey = "currentUser"
)

// ActorKind 与原项目 AppActor.kind 对齐。
type ActorKind string

const (
	ActorAnonymous ActorKind = "anonymous"
	ActorUser      ActorKind = "user"
	ActorSystem    ActorKind = "system"
)

type Actor struct {
	Kind   ActorKind
	UserID int64 // 仅 ActorUser 有效
}

// AnonymousActor 是默认占位,不持任何用户身份。
func AnonymousActor() Actor { return Actor{Kind: ActorAnonymous} }

// SessionContext 从 Gin context 读出 actor 与 currentUser(GetCurrentUser 可能为空)。
func GetActor(c *gin.Context) Actor {
	if v, ok := c.Get(CtxActorKey); ok {
		if a, ok := v.(Actor); ok {
			return a
		}
	}
	return AnonymousActor()
}

func GetCurrentUser(c *gin.Context) *auth.SessionUser {
	if v, ok := c.Get(CtxCurrentUserKey); ok {
		if u, ok := v.(*auth.SessionUser); ok {
			return u
		}
	}
	return nil
}

// SessionMiddleware 从 cookie 读 session token,验签后查 user。
// 没有 cookie / 无效 cookie / 用户被禁用 → actor 为 anonymous,不抛错(允许匿名访问)。
// 业务侧的 require-book-access 之类决定是否拒绝匿名。
func SessionMiddleware(authSvc *auth.Service, cookieName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(CtxActorKey, AnonymousActor())
		raw := readCookie(c.GetHeader("Cookie"), cookieName)
		if raw == "" {
			// 也支持 Authorization: Bearer <signed-token>,便于无 cookie 的 CLI/集成测试。
			if h := c.GetHeader("Authorization"); strings.HasPrefix(h, "Bearer ") {
				raw = strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
			}
		}
		if raw == "" {
			c.Next()
			return
		}
		token := authSvc.VerifySignedSessionToken(raw)
		if token == "" {
			c.Next()
			return
		}
		user, err := authSvc.GetSessionUser(c.Request.Context(), token)
		if err != nil || user == nil {
			c.Next()
			return
		}
		c.Set(CtxActorKey, Actor{Kind: ActorUser, UserID: user.ID})
		c.Set(CtxCurrentUserKey, user)
		c.Next()
	}
}

// RequireUser 拦截匿名访问,用于强制登录的端点(如 /api/user-settings/runtime)。
func RequireUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		a := GetActor(c)
		if a.Kind != ActorUser {
			AbortWithError(c, shared.Unauthorized("authentication required"))
			return
		}
		c.Next()
	}
}

func readCookie(headerValue, name string) string {
	if headerValue == "" {
		return ""
	}
	for _, c := range strings.Split(headerValue, ";") {
		c = strings.TrimSpace(c)
		if !strings.HasPrefix(c, name+"=") {
			continue
		}
		return strings.TrimPrefix(c, name+"=")
	}
	return ""
}
