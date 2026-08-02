package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/beancount-gs/api/internal/db"
	"github.com/beancount-gs/api/internal/ledger"
	"github.com/gin-gonic/gin"
)

type ImportHandlers struct {
	Store   *db.Store
	Service *ledger.Service
}

func (h *ImportHandlers) Preview(c *gin.Context) {
	l, ok := requireLedger(c, h.Store, "editor")
	if !ok {
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		BadRequest(c, "缺少上传文件")
		return
	}
	f, err := file.Open()
	if err != nil {
		BadRequest(c, "无法读取上传文件")
		return
	}
	defer f.Close()
	rows, err := h.Service.ImportPreview(c.Request.Context(), *l, c.Param("source"), f)
	if err != nil {
		BadRequest(c, "解析失败："+err.Error())
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *ImportHandlers) Confirm(c *gin.Context) {
	l, ok := requireLedger(c, h.Store, "editor")
	if !ok {
		return
	}
	if _, err := parseRevisionHeader(c); err != nil {
		Error(c, http.StatusUnprocessableEntity, "VALIDATION", "缺少或非法的 If-Revision-Match 头", nil)
		return
	}
	var form struct {
		Rows []ledger.ImportRow `json:"rows" binding:"required"`
	}
	if err := c.ShouldBindJSON(&form); err != nil {
		BadRequest(c, "参数错误："+err.Error())
		return
	}
	user := CurrentUser(c)
	result, err := h.Service.ImportConfirm(c.Request.Context(), *l, c.Param("source"), form.Rows,
		ledger.Actor{UserID: user.ID, Login: user.GitHubLogin})
	if err != nil {
		slog.Error("import confirm failed", "err", err)
		Error(c, http.StatusInternalServerError, "INTERNAL", "导入失败："+err.Error(), nil)
		return
	}
	c.JSON(http.StatusCreated, result)
}
