package httpapi

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/beancount-gs/api/internal/db"
	"github.com/beancount-gs/api/internal/http/gen"
	"github.com/beancount-gs/api/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type LedgerHandlers struct {
	Store       *db.Store
	DataRoot    string
	TemplateDir string
}

func (h *LedgerHandlers) List(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未登录", nil)
		return
	}
	ledgers, err := h.Store.ListLedgersForUser(c.Request.Context(), user.ID)
	if err != nil {
		slog.Error("list ledgers failed", "err", err)
		Error(c, http.StatusInternalServerError, "INTERNAL", "查询账本失败", nil)
		return
	}
	result := make([]gen.Ledger, 0, len(ledgers))
	for _, l := range ledgers {
		result = append(result, toGenLedger(l))
	}
	c.JSON(http.StatusOK, result)
}

func (h *LedgerHandlers) Create(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未登录", nil)
		return
	}
	var form gen.LedgerCreate
	if err := c.ShouldBindJSON(&form); err != nil {
		BadRequest(c, "参数错误："+err.Error())
		return
	}
	if strings.TrimSpace(form.Name) == "" {
		BadRequest(c, "账本名称不能为空")
		return
	}
	ctx := c.Request.Context()
	role, err := h.Store.TeamRole(ctx, form.TeamId, user.ID)
	if err != nil || (role != "owner" && role != "editor") {
		Error(c, http.StatusForbidden, "FORBIDDEN", "需要工作区 editor 及以上权限", nil)
		return
	}

	ledgerID := uuid.NewString()
	dataPath := filepath.Join(h.DataRoot, "teams", form.TeamId, "ledgers", ledgerID)
	startDate := ""
	if form.StartDate != nil {
		startDate = form.StartDate.String()
	}
	openingBalances := "Equity:OpeningBalances"
	if form.OpeningBalances != nil && *form.OpeningBalances != "" {
		openingBalances = *form.OpeningBalances
	}
	isBak := true
	if form.IsBak != nil {
		isBak = *form.IsBak
	}

	if err := repository.InitLedgerFiles(dataPath, h.TemplateDir, startDate, form.OperatingCurrency); err != nil {
		slog.Error("init ledger files failed", "err", err)
		Error(c, http.StatusInternalServerError, "INTERNAL", "初始化账本文件失败："+err.Error(), nil)
		return
	}
	ledger, err := h.Store.CreateLedgerWithOwner(ctx, db.NewLedgerParams{
		ID:                ledgerID,
		TeamID:            form.TeamId,
		Name:              strings.TrimSpace(form.Name),
		DataPath:          dataPath,
		OperatingCurrency: form.OperatingCurrency,
		StartDate:         startDate,
		OpeningBalances:   openingBalances,
		IsBak:             isBak,
	}, user.ID)
	if err != nil {
		slog.Error("create ledger failed", "err", err)
		Error(c, http.StatusInternalServerError, "INTERNAL", "创建账本失败："+err.Error(), nil)
		return
	}
	if err := h.Store.InsertAuditLog(ctx, db.AuditParams{
		LedgerID: ledger.ID, UserID: user.ID, Actor: user.GitHubLogin, Action: "create_ledger", Object: ledger.ID,
	}); err != nil {
		slog.Warn("audit log failed", "err", err)
	}
	c.JSON(http.StatusCreated, toGenLedger(ledger))
}

func (h *LedgerHandlers) Get(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未登录", nil)
		return
	}
	ledger, err := h.Store.GetLedgerForUser(c.Request.Context(), c.Param("ledger_id"), user.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Error(c, http.StatusNotFound, "NOT_FOUND", "账本不存在或无权访问", nil)
			return
		}
		slog.Error("get ledger failed", "err", err)
		Error(c, http.StatusInternalServerError, "INTERNAL", "查询账本失败", nil)
		return
	}
	c.JSON(http.StatusOK, toGenLedger(*ledger))
}

func (h *LedgerHandlers) Revision(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未登录", nil)
		return
	}
	ledger, err := h.Store.GetLedgerForUser(c.Request.Context(), c.Param("ledger_id"), user.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Error(c, http.StatusNotFound, "NOT_FOUND", "账本不存在或无权访问", nil)
			return
		}
		Error(c, http.StatusInternalServerError, "INTERNAL", "查询账本失败", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ledger_id": ledger.ID, "revision": ledger.Revision})
}
