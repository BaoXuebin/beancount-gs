package httpapi

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/beancount-gs/api/internal/db"
	"github.com/beancount-gs/api/internal/http/gen"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TeamsHandlers struct {
	Store *db.Store
}

func (h *TeamsHandlers) List(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未登录", nil)
		return
	}
	teams, err := h.Store.ListTeamsForUser(c.Request.Context(), user.ID)
	if err != nil {
		slog.Error("list teams failed", "err", err)
		Error(c, http.StatusInternalServerError, "INTERNAL", "查询工作区失败", nil)
		return
	}
	result := make([]gen.Team, 0, len(teams))
	for _, t := range teams {
		result = append(result, toGenTeam(t))
	}
	c.JSON(http.StatusOK, result)
}

func (h *TeamsHandlers) Create(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未登录", nil)
		return
	}
	var form gen.TeamCreate
	if err := c.ShouldBindJSON(&form); err != nil {
		BadRequest(c, "参数错误："+err.Error())
		return
	}
	name := strings.TrimSpace(form.Name)
	if name == "" {
		BadRequest(c, "工作区名称不能为空")
		return
	}
	ctx := c.Request.Context()
	team, err := h.Store.CreateTeamWithOwner(ctx, uuid.NewString(), name, user.ID)
	if err != nil {
		slog.Error("create team failed", "err", err)
		Error(c, http.StatusInternalServerError, "INTERNAL", "创建工作区失败", nil)
		return
	}
	if err := h.Store.InsertAuditLog(ctx, db.AuditParams{
		UserID: user.ID, Actor: user.GitHubLogin, Action: "create_team", Object: team.ID,
	}); err != nil {
		slog.Warn("audit log failed", "err", err)
	}
	c.JSON(http.StatusCreated, toGenTeam(team))
}
