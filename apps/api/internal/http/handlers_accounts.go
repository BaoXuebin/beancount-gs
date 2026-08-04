package httpapi

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/beancount-gs/api/internal/db"
	"github.com/beancount-gs/api/internal/http/gen"
	"github.com/beancount-gs/api/internal/ledger"
	"github.com/gin-gonic/gin"
)

type AccountHandlers struct {
	Store   *db.Store
	Service *ledger.Service
}

func (h *AccountHandlers) List(c *gin.Context) {
	l, ok := requireLedger(c, h.Store, "")
	if !ok {
		return
	}
	includeClosed := c.Query("status") == "closed"
	accounts, err := h.Service.ListAccounts(c.Request.Context(), *l, includeClosed)
	if err != nil {
		slog.Error("list accounts failed", "err", err)
		Error(c, http.StatusInternalServerError, "INTERNAL", "查询账户失败", nil)
		return
	}
	result := make([]gen.Account, 0, len(accounts))
	for _, a := range accounts {
		result = append(result, toGenAccount(a))
	}
	c.JSON(http.StatusOK, result)
}

func (h *AccountHandlers) Get(c *gin.Context) {
	l, ok := requireLedger(c, h.Store, "")
	if !ok {
		return
	}
	account, err := h.Service.GetAccount(c.Request.Context(), *l, c.Param("account"))
	if err != nil {
		if errors.Is(err, ledger.ErrNotFound) {
			Error(c, http.StatusNotFound, "NOT_FOUND", "账户不存在", nil)
			return
		}
		Error(c, http.StatusInternalServerError, "INTERNAL", "查询账户失败", nil)
		return
	}
	c.JSON(http.StatusOK, toGenAccount(*account))
}

func (h *AccountHandlers) Open(c *gin.Context) {
	l, ok := requireLedger(c, h.Store, "editor")
	if !ok {
		return
	}
	revision, err := parseRevisionHeader(c)
	if err != nil {
		Error(c, http.StatusUnprocessableEntity, "VALIDATION", "缺少或非法的 If-Revision-Match 头", nil)
		return
	}
	var form gen.AccountOpen
	if err := c.ShouldBindJSON(&form); err != nil {
		BadRequest(c, "参数错误："+err.Error())
		return
	}
	booking := ""
	if form.Booking != nil {
		booking = string(*form.Booking)
	}
	currency := ""
	if form.Currency != nil {
		currency = *form.Currency
	}
	user := CurrentUser(c)
	account, err := h.Service.OpenAccount(c.Request.Context(), *l, ledger.OpenAccount{
		Account: form.Account, OpenedOn: form.OpenedOn.String(),
		Currency: currency, Booking: booking,
	}, revision, ledger.Actor{UserID: user.ID, Login: user.GitHubLogin})
	if err != nil {
		h.writeAccountError(c, *l, err)
		return
	}
	c.JSON(http.StatusCreated, toGenAccount(*account))
}

func (h *AccountHandlers) BatchOpen(c *gin.Context) {
	l, ok := requireLedger(c, h.Store, "editor")
	if !ok {
		return
	}
	revision, err := parseRevisionHeader(c)
	if err != nil {
		Error(c, http.StatusUnprocessableEntity, "VALIDATION", "缺少或非法的 If-Revision-Match 头", nil)
		return
	}
	var form gen.AccountOpenBatch
	if err := c.ShouldBindJSON(&form); err != nil {
		BadRequest(c, "参数错误："+err.Error())
		return
	}
	opens := make([]ledger.OpenAccount, 0, len(form.Accounts))
	for _, a := range form.Accounts {
		booking := ""
		if a.Booking != nil {
			booking = string(*a.Booking)
		}
		currency := ""
		if a.Currency != nil {
			currency = *a.Currency
		}
		opens = append(opens, ledger.OpenAccount{
			Account: a.Account, OpenedOn: a.OpenedOn.String(),
			Currency: currency, Booking: booking,
		})
	}
	user := CurrentUser(c)
	result, err := h.Service.BatchOpenAccounts(c.Request.Context(), *l, opens, revision, ledger.Actor{UserID: user.ID, Login: user.GitHubLogin})
	if err != nil {
		h.writeAccountError(c, *l, err)
		return
	}
	created := make([]gen.Account, 0, len(result.Created))
	for _, a := range result.Created {
		created = append(created, toGenAccount(a))
	}
	c.JSON(http.StatusCreated, gin.H{"created": created, "skipped": result.Skipped})
}
func (h *AccountHandlers) Close(c *gin.Context) {
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
		ClosedOn string `json:"closed_on" binding:"required"`
	}
	if err := c.ShouldBindJSON(&form); err != nil {
		BadRequest(c, "参数错误："+err.Error())
		return
	}
	user := CurrentUser(c)
	account, err := h.Service.CloseAccount(c.Request.Context(), *l, c.Param("account"), form.ClosedOn,
		revision, ledger.Actor{UserID: user.ID, Login: user.GitHubLogin})
	if err != nil {
		h.writeAccountError(c, *l, err)
		return
	}
	c.JSON(http.StatusOK, toGenAccount(*account))
}

func (h *AccountHandlers) Reopen(c *gin.Context) {
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
		OpenedOn string `json:"opened_on" binding:"required"`
	}
	if err := c.ShouldBindJSON(&form); err != nil {
		BadRequest(c, "参数错误："+err.Error())
		return
	}
	user := CurrentUser(c)
	account, err := h.Service.ReopenAccount(c.Request.Context(), *l, c.Param("account"), form.OpenedOn,
		revision, ledger.Actor{UserID: user.ID, Login: user.GitHubLogin})
	if err != nil {
		h.writeAccountError(c, *l, err)
		return
	}
	c.JSON(http.StatusOK, toGenAccount(*account))
}

func (h *AccountHandlers) Balance(c *gin.Context) {
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
		Date   string `json:"date" binding:"required"`
		Number string `json:"number" binding:"required"`
	}
	if err := c.ShouldBindJSON(&form); err != nil {
		BadRequest(c, "参数错误："+err.Error())
		return
	}
	user := CurrentUser(c)
	err = h.Service.BalanceAccount(c.Request.Context(), *l, c.Param("account"), form.Date, form.Number,
		revision, ledger.Actor{UserID: user.ID, Login: user.GitHubLogin})
	if err != nil {
		h.writeAccountError(c, *l, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AccountHandlers) ListTypes(c *gin.Context) {
	l, ok := requireLedger(c, h.Store, "")
	if !ok {
		return
	}
	types, err := h.Service.ListAccountTypes(c.Request.Context(), *l)
	if err != nil {
		Error(c, http.StatusInternalServerError, "INTERNAL", "查询账户类型失败", nil)
		return
	}
	result := make([]gen.AccountTypeMapping, 0, len(types))
	for _, t := range types {
		result = append(result, gen.AccountTypeMapping{Prefix: t.Prefix, Name: t.Name})
	}
	c.JSON(http.StatusOK, result)
}

func (h *AccountHandlers) AddType(c *gin.Context) {
	l, ok := requireLedger(c, h.Store, "editor")
	if !ok {
		return
	}
	revision, err := parseRevisionHeader(c)
	if err != nil {
		Error(c, http.StatusUnprocessableEntity, "VALIDATION", "缺少或非法的 If-Revision-Match 头", nil)
		return
	}
	var form gen.AccountTypeMapping
	if err := c.ShouldBindJSON(&form); err != nil {
		BadRequest(c, "参数错误："+err.Error())
		return
	}
	user := CurrentUser(c)
	err = h.Service.UpsertAccountType(c.Request.Context(), *l, ledger.AccountType{Prefix: form.Prefix, Name: form.Name},
		revision, ledger.Actor{UserID: user.ID, Login: user.GitHubLogin})
	if err != nil {
		h.writeAccountError(c, *l, err)
		return
	}
	c.JSON(http.StatusCreated, form)
}

func (h *AccountHandlers) writeAccountError(c *gin.Context, l db.Ledger, err error) {
	switch {
	case errors.Is(err, db.ErrRevisionConflict):
		Error(c, http.StatusConflict, "LEDGER_STALE", "账本已被他人修改", map[string]any{"current_revision": l.Revision})
	case errors.Is(err, ledger.ErrDuplicateAccount):
		Error(c, http.StatusConflict, "DUPLICATE_ACCOUNT", "账户已存在", nil)
	case errors.Is(err, ledger.ErrInvalidDate), errors.Is(err, ledger.ErrAccountNotClosed):
		Error(c, http.StatusUnprocessableEntity, "VALIDATION", err.Error(), nil)
	default:
		slog.Error("account write failed", "err", err)
		Error(c, http.StatusInternalServerError, "INTERNAL", "操作失败："+err.Error(), nil)
	}
}
