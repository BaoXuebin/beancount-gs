package httpapi

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/beancount-gs/api/internal/db"
	"github.com/beancount-gs/api/internal/security"
	"github.com/gin-gonic/gin"
)

func RequireSession(store *db.Store, cookieName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(cookieName)
		if err != nil || token == "" {
			Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未登录或会话已过期", nil)
			return
		}
		user, err := store.UserBySession(c.Request.Context(), security.HashToken(token))
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未登录或会话已过期", nil)
				return
			}
			slog.Error("session lookup failed", "err", err)
			Error(c, http.StatusInternalServerError, "INTERNAL", "会话校验失败", nil)
			return
		}
		c.Set("user", user)
		c.Next()
	}
}

func CurrentUser(c *gin.Context) *db.User {
	value, ok := c.Get("user")
	if !ok {
		return nil
	}
	user, _ := value.(*db.User)
	return user
}

// RequestLogger 记录每个 HTTP 请求的方法、路径、状态码与耗时。
// 按状态分级：>=500 Error、>=400 Warn、其余 Info。
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		if raw := c.Request.URL.RawQuery; raw != "" {
			path = path + "?" + raw
		}
		c.Next()

		status := c.Writer.Status()
		attrs := []any{
			"method", c.Request.Method,
			"path", path,
			"status", status,
			"duration", time.Since(start).String(),
			"ip", c.ClientIP(),
		}
		if user := CurrentUser(c); user != nil {
			attrs = append(attrs, "user", user.ID)
		}
		switch {
		case status >= 500:
			slog.Error("http request", attrs...)
		case status >= 400:
			slog.Warn("http request", attrs...)
		default:
			slog.Info("http request", attrs...)
		}
	}
}