package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"myai-novel-go/internal/domain/shared"
)

const (
	CtxLoggerKey   = "logger"
	CtxRequestID   = "requestId"
	HeaderRequestID = "X-Request-ID"
)

func RequestContext(rootLogger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		reqID := strings.TrimSpace(c.GetHeader(HeaderRequestID))
		if reqID == "" {
			reqID = uuid.NewString()
		}
		c.Header(HeaderRequestID, reqID)

		l := rootLogger.With(
			zap.String("requestId", reqID),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
		)
		c.Set(CtxLoggerKey, l)
		c.Set(CtxRequestID, reqID)

		l.Info("http.request.start")
		c.Next()
		l.Info("http.request.finish",
			zap.Int("statusCode", c.Writer.Status()),
			zap.Int64("durationMs", time.Since(started).Milliseconds()),
		)
	}
}

func Recovery(rootLogger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				l := getLogger(c, rootLogger)
				l.Error("http.request.panic",
					zap.Any("panic", r),
					zap.String("stack", string(debug.Stack())),
				)
				writeErrorJSON(c, 500, "internal_error", "Internal server error", nil)
			}
		}()
		c.Next()
	}
}

// ErrorResponder 统一把 c.Errors 中的最后一个错误转成 JSON 响应。
func ErrorResponder() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 {
			return
		}
		// 已经写过响应,跳过
		if c.Writer.Written() {
			return
		}
		err := c.Errors.Last().Err
		writeError(c, err)
	}
}

func writeError(c *gin.Context, err error) {
	var appErr *shared.AppError
	if errors.As(err, &appErr) {
		writeErrorJSON(c, appErr.Status, appErr.Code, appErr.Message, appErr.Details)
		return
	}
	writeErrorJSON(c, http.StatusInternalServerError, "internal_error", err.Error(), nil)
}

func writeErrorJSON(c *gin.Context, status int, code, message string, details any) {
	body := gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	}
	if details != nil {
		body["error"].(gin.H)["details"] = details
	}
	c.AbortWithStatusJSON(status, body)
}

func getLogger(c *gin.Context, fallback *zap.Logger) *zap.Logger {
	v, ok := c.Get(CtxLoggerKey)
	if ok {
		if l, ok := v.(*zap.Logger); ok {
			return l
		}
	}
	return fallback
}

// Logger 返回请求 scope 的 logger,用于 handler 内部
func Logger(c *gin.Context) *zap.Logger {
	if v, ok := c.Get(CtxLoggerKey); ok {
		if l, ok := v.(*zap.Logger); ok {
			return l
		}
	}
	return zap.NewNop()
}

// AbortWithError 是 handler 的统一错误抛出口。
func AbortWithError(c *gin.Context, err error) {
	_ = c.Error(err)
	c.Abort()
}

func ParseInt64Param(c *gin.Context, name string) (int64, error) {
	raw := c.Param(name)
	if raw == "" {
		return 0, shared.BadRequest(fmt.Sprintf("missing path param: %s", name))
	}
	var n int64
	_, err := fmt.Sscanf(raw, "%d", &n)
	if err != nil || n <= 0 {
		return 0, shared.BadRequest(fmt.Sprintf("invalid path param %s: %s", name, raw))
	}
	return n, nil
}

func ParseIntParam(c *gin.Context, name string) (int, error) {
	v, err := ParseInt64Param(c, name)
	if err != nil {
		return 0, err
	}
	return int(v), nil
}
