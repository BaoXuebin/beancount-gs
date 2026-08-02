package httpapi

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/beancount-gs/api/internal/db"
	"github.com/beancount-gs/api/internal/mcp"
	"github.com/beancount-gs/api/internal/security"
	"github.com/gin-gonic/gin"
)

// RequireApiKey 供 MCP / 外部 Agent 使用 Bearer API Key 认证。
func RequireApiKey(store *db.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := bearerToken(c.GetHeader("Authorization"))
		if !ok || token == "" {
			Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "缺少 Bearer API Key", nil)
			return
		}
		ctx := c.Request.Context()
		key, err := store.GetApiKeyByHash(ctx, security.HashToken(token))
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "API Key 无效或已吊销", nil)
				return
			}
			slog.Error("api key lookup failed", "err", err)
			Error(c, http.StatusInternalServerError, "INTERNAL", "API Key 校验失败", nil)
			return
		}
		user, err := store.GetUserByID(ctx, key.UserID)
		if err != nil {
			Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "API Key 所属用户不存在", nil)
			return
		}
		_ = store.TouchApiKey(ctx, key.ID)
		c.Request = c.Request.WithContext(mcp.WithAuthUser(ctx, *user, key.Scope))
		c.Next()
	}
}

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1], true
	}
	return "", false
}
