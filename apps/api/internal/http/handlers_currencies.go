package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/beancount-gs/api/internal/db"
	"github.com/beancount-gs/api/internal/ledger"
	"github.com/gin-gonic/gin"
)

type CurrencyHandlers struct {
	Store   *db.Store
	Service *ledger.Service
}

func (h *CurrencyHandlers) List(c *gin.Context) {
	l, ok := requireLedger(c, h.Store, "")
	if !ok {
		return
	}
	list, err := h.Service.ListCurrencies(c.Request.Context(), *l)
	if err != nil {
		slog.Error("list currencies failed", "err", err)
		Error(c, http.StatusInternalServerError, "INTERNAL", "查询币种失败", nil)
		return
	}
	c.JSON(http.StatusOK, toGenCurrencies(list))
}

func (h *CurrencyHandlers) Add(c *gin.Context) {
	l, ok := requireLedger(c, h.Store, "editor")
	if !ok {
		return
	}
	revision, err := parseRevisionHeader(c)
	if err != nil {
		Error(c, http.StatusUnprocessableEntity, "VALIDATION", "缺少或非法的 If-Revision-Match 头", nil)
		return
	}
	var form struct {
		Currency string `json:"currency" binding:"required"`
		Name     string `json:"name" binding:"required"`
		Symbol   string `json:"symbol"`
	}
	if err := c.ShouldBindJSON(&form); err != nil {
		BadRequest(c, "参数错误："+err.Error())
		return
	}
	user := CurrentUser(c)
	if err := h.Service.AddCurrency(c.Request.Context(), *l, form.Currency, form.Name, form.Symbol,
		revision, ledger.Actor{UserID: user.ID, Login: user.GitHubLogin}); err != nil {
		h.writeCurrencyError(c, *l, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"currency": strings.ToUpper(strings.TrimSpace(form.Currency))})
}

func (h *CurrencyHandlers) Sync(c *gin.Context) {
	l, ok := requireLedger(c, h.Store, "editor")
	if !ok {
		return
	}
	revision, err := parseRevisionHeader(c)
	if err != nil {
		Error(c, http.StatusUnprocessableEntity, "VALIDATION", "缺少或非法的 If-Revision-Match 头", nil)
		return
	}
	user := CurrentUser(c)
	list, err := h.Service.SyncCurrencies(c.Request.Context(), *l, revision,
		ledger.Actor{UserID: user.ID, Login: user.GitHubLogin})
	if err != nil {
		h.writeCurrencyError(c, *l, err)
		return
	}
	c.JSON(http.StatusOK, toGenCurrencies(list))
}

func (h *CurrencyHandlers) writeCurrencyError(c *gin.Context, l db.Ledger, err error) {
	switch {
	case errors.Is(err, db.ErrRevisionConflict):
		Error(c, http.StatusConflict, "LEDGER_STALE", "账本已被他人修改", map[string]any{"current_revision": l.Revision})
	case errors.Is(err, ledger.ErrInvalidDate):
		Error(c, http.StatusUnprocessableEntity, "VALIDATION", err.Error(), nil)
	default:
		slog.Error("currency write failed", "err", err)
		Error(c, http.StatusInternalServerError, "INTERNAL", "操作失败："+err.Error(), nil)
	}
}
