package httpapi

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/beancount-gs/api/internal/db"
	"github.com/beancount-gs/api/internal/http/gen"
	"github.com/beancount-gs/api/internal/ledger"
	"github.com/gin-gonic/gin"
)

type TransactionHandlers struct {
	Store   *db.Store
	Service *ledger.Service
}

func (h *TransactionHandlers) List(c *gin.Context) {
	l, ok := h.requireLedger(c, "")
	if !ok {
		return
	}
	f := ledger.Filters{
		From:    c.Query("from"),
		To:      c.Query("to"),
		Month:   c.Query("month"),
		Account: c.Query("account"),
		Tag:     c.Query("tag"),
		Q:       c.Query("q"),
		Order:   c.Query("order"),
	}
	txns, err := h.Service.List(c.Request.Context(), *l, f)
	if err != nil {
		slog.Error("list transactions failed", "ledger", l.ID, "err", err)
		Error(c, http.StatusInternalServerError, "INTERNAL", "查询交易失败："+err.Error(), nil)
		return
	}
	items := make([]gen.Transaction, 0, len(txns))
	for _, t := range txns {
		items = append(items, toGenTransaction(t))
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func (h *TransactionHandlers) Get(c *gin.Context) {
	l, ok := h.requireLedger(c, "")
	if !ok {
		return
	}
	txn, err := h.Service.Get(c.Request.Context(), *l, c.Param("transaction_id"))
	if err != nil {
		if errors.Is(err, ledger.ErrNotFound) {
			Error(c, http.StatusNotFound, "NOT_FOUND", "交易不存在", nil)
			return
		}
		slog.Error("get transaction failed", "err", err)
		Error(c, http.StatusInternalServerError, "INTERNAL", "查询交易失败", nil)
		return
	}
	c.JSON(http.StatusOK, toGenTransaction(*txn))
}

func (h *TransactionHandlers) Create(c *gin.Context) {
	l, ok := h.requireLedger(c, "editor")
	if !ok {
		return
	}
	revision, err := parseRevisionHeader(c)
	if err != nil {
		Error(c, http.StatusUnprocessableEntity, "VALIDATION", "缺少或非法的 If-Revision-Match 头", nil)
		return
	}
	var form gen.TransactionCreate
	if err := c.ShouldBindJSON(&form); err != nil {
		BadRequest(c, "参数错误："+err.Error())
		return
	}
	user := CurrentUser(c)
	created, err := h.Service.Create(c.Request.Context(), *l,
		fromGenTransactionCreate(form), revision, ledger.Actor{UserID: user.ID, Login: user.GitHubLogin})
	if err != nil {
		switch {
		case errors.Is(err, db.ErrRevisionConflict):
			details := map[string]any{"current_revision": l.Revision}
			Error(c, http.StatusConflict, "LEDGER_STALE", "账本已被他人修改", details)
		case errors.Is(err, ledger.ErrNotBalanced):
			Error(c, http.StatusUnprocessableEntity, "UNBALANCED", "交易借贷不平衡", nil)
		case errors.Is(err, ledger.ErrInvalidDate), errors.Is(err, ledger.ErrNoPostings):
			Error(c, http.StatusUnprocessableEntity, "VALIDATION", err.Error(), nil)
		default:
			slog.Error("create transaction failed", "err", err)
			Error(c, http.StatusInternalServerError, "INTERNAL", "创建交易失败："+err.Error(), nil)
		}
		return
	}
	c.JSON(http.StatusCreated, toGenTransaction(*created))
}

func (h *TransactionHandlers) requireLedger(c *gin.Context, minRole string) (*db.Ledger, bool) {
	user := CurrentUser(c)
	if user == nil {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未登录", nil)
		return nil, false
	}
	l, err := h.Store.GetLedgerForUser(c.Request.Context(), c.Param("ledger_id"), user.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Error(c, http.StatusNotFound, "NOT_FOUND", "账本不存在或无权访问", nil)
			return nil, false
		}
		slog.Error("get ledger failed", "err", err)
		Error(c, http.StatusInternalServerError, "INTERNAL", "查询账本失败", nil)
		return nil, false
	}
	if minRole == "editor" && l.Role == "viewer" {
		Error(c, http.StatusForbidden, "FORBIDDEN", "viewer 无写权限", nil)
		return nil, false
	}
	return l, true
}

func parseRevisionHeader(c *gin.Context) (int64, error) {
	raw := c.GetHeader("If-Revision-Match")
	if raw == "" {
		return 0, errors.New("missing revision header")
	}
	return strconv.ParseInt(raw, 10, 64)
}
