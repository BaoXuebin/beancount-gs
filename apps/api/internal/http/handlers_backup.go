package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/beancount-gs/api/internal/db"
	"github.com/beancount-gs/api/internal/ledger"
	"github.com/beancount-gs/api/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type BackupHandlers struct {
	Store    *db.Store
	Service  *ledger.Service
	DataRoot string
}

// validateAndExtract 解压 zip 到 dest 并校验：必须含根目录 index.bean，且 bean-check 通过。
func (h *BackupHandlers) validateAndExtract(ctx context.Context, file io.Reader, dest string) ([]string, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("读取备份文件失败: %w", err)
	}
	if len(data) == 0 {
		return nil, errors.New("备份文件为空")
	}
	files, err := repository.ExtractZip(bytes.NewReader(data), int64(len(data)), dest)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(dest, "index.bean")); err != nil {
		return nil, errors.New("备份缺少根目录 index.bean")
	}
	lines, err := h.Service.Engine.Check(ctx, filepath.Join(dest, "index.bean"))
	if err != nil {
		return nil, fmt.Errorf("bean-check 执行失败: %w", err)
	}
	if len(lines) > 0 {
		limit := lines
		if len(limit) > 3 {
			limit = limit[:3]
		}
		return nil, fmt.Errorf("bean-check 校验失败: %s", strings.Join(limit, "；"))
	}
	return files, nil
}

// ImportAsNew 从备份 zip 导入并新建账本：POST /api/v2/ledgers/import
func (h *BackupHandlers) ImportAsNew(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未登录", nil)
		return
	}
	if err := c.Request.ParseMultipartForm(64 << 20); err != nil {
		BadRequest(c, "multipart 解析失败："+err.Error())
		return
	}
	teamID := strings.TrimSpace(c.Request.FormValue("team_id"))
	name := strings.TrimSpace(c.Request.FormValue("name"))
	currency := strings.TrimSpace(c.Request.FormValue("operating_currency"))
	if currency == "" {
		currency = "CNY"
	}
	if name == "" {
		BadRequest(c, "账本名称不能为空")
		return
	}
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		BadRequest(c, "缺少 zip 备份文件")
		return
	}
	defer file.Close()

	ctx := c.Request.Context()
	role, err := h.Store.TeamRole(ctx, teamID, user.ID)
	if err != nil || (role != "owner" && role != "editor") {
		Error(c, http.StatusForbidden, "FORBIDDEN", "需要工作区 editor 及以上权限", nil)
		return
	}

	ledgerID := uuid.NewString()
	dataPath := filepath.Join(h.DataRoot, "teams", teamID, "ledgers", ledgerID)
	files, err := h.validateAndExtract(ctx, file, dataPath)
	if err != nil {
		_ = os.RemoveAll(dataPath)
		Error(c, http.StatusUnprocessableEntity, "VALIDATION", "备份校验失败："+err.Error(), nil)
		return
	}
	l, err := h.Store.CreateLedgerWithOwner(ctx, db.NewLedgerParams{
		ID:                ledgerID,
		TeamID:            teamID,
		Name:              name,
		DataPath:          dataPath,
		OperatingCurrency: currency,
		IsBak:             true,
	}, user.ID)
	if err != nil {
		_ = os.RemoveAll(dataPath)
		slog.Error("create ledger from backup failed", "err", err)
		Error(c, http.StatusInternalServerError, "INTERNAL", "创建账本失败："+err.Error(), nil)
		return
	}
	detail, _ := json.Marshal(map[string]any{"files": len(files)})
	if err := h.Store.InsertAuditLog(ctx, db.AuditParams{
		LedgerID: l.ID, UserID: user.ID, Actor: user.GitHubLogin,
		Action: "import_backup_new", Object: ledgerID, Detail: string(detail),
	}); err != nil {
		slog.Warn("audit log failed", "err", err)
	}
	c.JSON(http.StatusCreated, gin.H{"ledger": toGenLedger(l), "files": files})
}

// ImportInto 导入备份 zip 到已有账本：POST /api/v2/ledgers/{ledger_id}/import
func (h *BackupHandlers) ImportInto(c *gin.Context) {
	l, ok := requireLedger(c, h.Store, "editor")
	if !ok {
		return
	}
	user := CurrentUser(c)
	if user == nil {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未登录", nil)
		return
	}
	revision, err := parseRevisionHeader(c)
	if err != nil {
		Error(c, http.StatusUnprocessableEntity, "VALIDATION", "缺少或非法的 If-Revision-Match 头", nil)
		return
	}
	if err := c.Request.ParseMultipartForm(64 << 20); err != nil {
		BadRequest(c, "multipart 解析失败："+err.Error())
		return
	}
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		BadRequest(c, "缺少 zip 备份文件")
		return
	}
	defer file.Close()

	ctx := c.Request.Context()
	tmp := filepath.Join(h.DataRoot, ".tmp-import", uuid.NewString())
	defer os.RemoveAll(tmp)
	files, err := h.validateAndExtract(ctx, file, tmp)
	if err != nil {
		Error(c, http.StatusUnprocessableEntity, "VALIDATION", "备份校验失败："+err.Error(), nil)
		return
	}
	newRev, err := h.Service.ImportBackup(ctx, *l, tmp, revision, ledger.Actor{
		UserID: user.ID, Login: user.GitHubLogin,
	})
	if err != nil {
		if errors.Is(err, db.ErrRevisionConflict) {
			Error(c, http.StatusConflict, "LEDGER_STALE", "账本已被他人修改",
				map[string]any{"current_revision": l.Revision})
			return
		}
		slog.Error("import backup into ledger failed", "err", err)
		Error(c, http.StatusInternalServerError, "INTERNAL", "导入失败："+err.Error(), nil)
		return
	}
	l2, err := h.Store.GetLedger(ctx, l.ID)
	if err != nil {
		slog.Error("reload ledger after import failed", "err", err)
		l2 = l
	}
	c.JSON(http.StatusOK, gin.H{
		"ledger":   toGenLedger(*l2),
		"revision": newRev,
		"files":    files,
	})
}
