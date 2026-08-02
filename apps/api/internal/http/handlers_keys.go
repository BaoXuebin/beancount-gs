package httpapi

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/beancount-gs/api/internal/db"
	"github.com/beancount-gs/api/internal/http/gen"
	"github.com/beancount-gs/api/internal/security"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type KeyHandlers struct {
	Store *db.Store
}

func (h *KeyHandlers) List(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未登录", nil)
		return
	}
	keys, err := h.Store.ListApiKeys(c.Request.Context(), user.ID)
	if err != nil {
		Error(c, http.StatusInternalServerError, "INTERNAL", "查询 API Key 失败", nil)
		return
	}
	result := make([]gen.ApiKey, 0, len(keys))
	for _, k := range keys {
		result = append(result, toGenApiKey(k))
	}
	c.JSON(http.StatusOK, result)
}

func (h *KeyHandlers) Create(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未登录", nil)
		return
	}
	var form struct {
		Name  string `json:"name" binding:"required"`
		Scope string `json:"scope" binding:"required,oneof=read-only read-write ai"`
	}
	if err := c.ShouldBindJSON(&form); err != nil {
		BadRequest(c, "参数错误："+err.Error())
		return
	}
	secret := "bgsk_" + security.RandomHex(24)
	key := db.ApiKey{
		ID:         uuid.NewString(),
		UserID:     user.ID,
		Name:       strings.TrimSpace(form.Name),
		SecretHash: security.HashToken(secret),
		Prefix:     secret[:12],
		Scope:      form.Scope,
	}
	if err := h.Store.CreateApiKey(c.Request.Context(), key); err != nil {
		slog.Error("create api key failed", "err", err)
		Error(c, http.StatusInternalServerError, "INTERNAL", "创建 API Key 失败", nil)
		return
	}
	created := toGenApiKey(key)
	c.JSON(http.StatusCreated, gin.H{
		"id":         created.Id,
		"name":       created.Name,
		"scope":      created.Scope,
		"prefix":     created.Prefix,
		"created_at": created.CreatedAt,
		"secret":     secret,
	})
}

func (h *KeyHandlers) Revoke(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未登录", nil)
		return
	}
	err := h.Store.RevokeApiKey(c.Request.Context(), c.Param("key_id"), user.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Error(c, http.StatusNotFound, "NOT_FOUND", "API Key 不存在", nil)
			return
		}
		Error(c, http.StatusInternalServerError, "INTERNAL", "吊销失败", nil)
		return
	}
	c.Status(http.StatusNoContent)
}

func toGenApiKey(k db.ApiKey) gen.ApiKey {
	revoked := k.Revoked
	return gen.ApiKey{
		Id:         k.ID,
		Name:       k.Name,
		Scope:      gen.ApiKeyScope(k.Scope),
		Prefix:     strPtr(k.Prefix),
		Revoked:    &revoked,
		CreatedAt:  parseTime(k.CreatedAt),
		LastUsedAt: parseTime(k.LastUsedAt),
	}
}
