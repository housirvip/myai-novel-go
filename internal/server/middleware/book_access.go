package middleware

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"myai-novel-go/internal/db/models"
	"myai-novel-go/internal/domain/shared"
)

// RequireBookAccess 给所有路径里包含 :bookId 的路由加一层"书归属"校验。
// 规则(对齐原 Node 项目 require-book-access):
//   - books.owner_user_id 为 NULL(未分配 owner)→ 任意 actor 可访问;
//     这是为了向后兼容以及"系统初始化前的引导期"(没有用户也能跑工作流)。
//   - books.owner_user_id 有值 → 仅 actor=user 且 actor.UserID == owner 可访问,
//     其它一律 403。
//
// 中间件挂在 r.Use 全局,通过 c.Param("bookId") 自动只对匹配到 :bookId 的路由生效;
// 不匹配 :bookId 的路由(如 POST /api/books)会跳过校验。
func RequireBookAccess(gdb *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.Param("bookId")
		if raw == "" {
			c.Next()
			return
		}
		var bookID int64
		if _, err := fmt.Sscanf(raw, "%d", &bookID); err != nil || bookID <= 0 {
			// 格式错误不在这里报,留给 handler 的 ParseInt64Param 统一处理
			c.Next()
			return
		}
		var book models.Book
		err := gdb.WithContext(c.Request.Context()).
			Select("id", "owner_user_id").
			First(&book, bookID).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 不存在交给 handler 走 404 路径,middleware 不抢
				c.Next()
				return
			}
			AbortWithError(c, err)
			return
		}
		actor := GetActor(c)
		if !ActorMayAccessBook(actor, book.OwnerUserID) {
			AbortWithError(c, shared.Forbidden("forbidden: book belongs to another user"))
			return
		}
		c.Next()
	}
}

// ActorMayAccessBook 把所有权判定独立成函数,
// service 层做 List 过滤时也要用同样规则,所以提到 middleware 包外可见。
func ActorMayAccessBook(a Actor, ownerUserID *int64) bool {
	if ownerUserID == nil {
		return true
	}
	if a.Kind == ActorUser && a.UserID == *ownerUserID {
		return true
	}
	if a.Kind == ActorSystem {
		return true
	}
	return false
}

