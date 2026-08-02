package httpapi

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"

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
