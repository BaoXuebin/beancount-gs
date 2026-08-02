package httpapi

import (
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

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
	l, ok := requireLedger(c, h.Store, "")
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
	l, ok := requireLedger(c, h.Store, "")
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
	l, ok := requireLedger(c, h.Store, "editor")
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
		h.writeCreateError(c, *l, err)
		return
	}
	c.JSON(http.StatusCreated, toGenTransaction(*created))
}

func (h *TransactionHandlers) Update(c *gin.Context) {
	l, ok := requireLedger(c, h.Store, "editor")
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
	updated, err := h.Service.Update(c.Request.Context(), *l, c.Param("transaction_id"),
		fromGenTransactionCreate(form), revision, ledger.Actor{UserID: user.ID, Login: user.GitHubLogin})
	if err != nil {
		h.writeCreateError(c, *l, err)
		return
	}
	c.JSON(http.StatusOK, toGenTransaction(*updated))
}

func (h *TransactionHandlers) Delete(c *gin.Context) {
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
	err = h.Service.Delete(c.Request.Context(), *l, c.Param("transaction_id"), revision,
		ledger.Actor{UserID: user.ID, Login: user.GitHubLogin})
	if err != nil {
		h.writeCreateError(c, *l, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *TransactionHandlers) RawText(c *gin.Context) {
	l, ok := requireLedger(c, h.Store, "")
	if !ok {
		return
	}
	text, err := h.Service.RawText(c.Request.Context(), *l, c.Param("transaction_id"))
	if err != nil {
		slog.Error("get raw text failed", "err", err)
		Error(c, http.StatusInternalServerError, "INTERNAL", "读取原始文本失败", nil)
		return
	}
	if strings.TrimSpace(text) == "" {
		Error(c, http.StatusNotFound, "NOT_FOUND", "交易不存在", nil)
		return
	}
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(text))
}

func (h *TransactionHandlers) UpdateRawText(c *gin.Context) {
	l, ok := requireLedger(c, h.Store, "editor")
	if !ok {
		return
	}
	revision, err := parseRevisionHeader(c)
	if err != nil {
		Error(c, http.StatusUnprocessableEntity, "VALIDATION", "缺少或非法的 If-Revision-Match 头", nil)
		return
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		BadRequest(c, "读取请求体失败")
		return
	}
	raw := string(body)
	if strings.TrimSpace(raw) == "" {
		BadRequest(c, "原始文本不能为空")
		return
	}
	user := CurrentUser(c)
	err = h.Service.UpdateRawText(c.Request.Context(), *l, c.Param("transaction_id"), raw, revision,
		ledger.Actor{UserID: user.ID, Login: user.GitHubLogin})
	if err != nil {
		h.writeCreateError(c, *l, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *TransactionHandlers) writeCreateError(c *gin.Context, l db.Ledger, err error) {
	switch {
	case errors.Is(err, db.ErrRevisionConflict):
		Error(c, http.StatusConflict, "LEDGER_STALE", "账本已被他人修改", map[string]any{"current_revision": l.Revision})
	case errors.Is(err, ledger.ErrNotFound):
		Error(c, http.StatusNotFound, "NOT_FOUND", "交易不存在", nil)
	case errors.Is(err, ledger.ErrNotBalanced):
		Error(c, http.StatusUnprocessableEntity, "UNBALANCED", "交易借贷不平衡", nil)
	case errors.Is(err, ledger.ErrInvalidDate), errors.Is(err, ledger.ErrNoPostings):
		Error(c, http.StatusUnprocessableEntity, "VALIDATION", err.Error(), nil)
	default:
		slog.Error("transaction write failed", "err", err)
		Error(c, http.StatusInternalServerError, "INTERNAL", "操作失败："+err.Error(), nil)
	}
}

func requireLedger(c *gin.Context, store *db.Store, minRole string) (*db.Ledger, bool) {
	user := CurrentUser(c)
	if user == nil {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未登录", nil)
		return nil, false
	}
	l, err := store.GetLedgerForUser(c.Request.Context(), c.Param("ledger_id"), user.ID)
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
