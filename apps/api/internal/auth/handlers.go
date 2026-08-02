package auth

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/beancount-gs/api/internal/db"
	httpapi "github.com/beancount-gs/api/internal/http"
	"github.com/beancount-gs/api/internal/security"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handlers struct {
	Store       *db.Store
	OAuth       *GitHubOAuth
	PublicURL   string
	CookieName  string
	StateCookie string
	SessionTTL  time.Duration
}

func (h *Handlers) Login(c *gin.Context) {
	if h.OAuth.ClientID == "" || h.OAuth.ClientSecret == "" {
		httpapi.Error(c, http.StatusServiceUnavailable, "GITHUB_NOT_CONFIGURED",
			"服务端未配置 GitHub OAuth（GITHUB_CLIENT_ID / GITHUB_CLIENT_SECRET）", nil)
		return
	}
	state := security.RandomHex(16)
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     h.StateCookie,
		Value:    state,
		Path:     "/api/v2/auth",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	})
	c.Redirect(http.StatusFound, h.OAuth.AuthorizeURL(state))
}

func (h *Handlers) Callback(c *gin.Context) {
	stateCookie, err := c.Cookie(h.StateCookie)
	if err != nil || stateCookie == "" || stateCookie != c.Query("state") {
		httpapi.Error(c, http.StatusBadRequest, "INVALID_STATE", "OAuth state 校验失败", nil)
		return
	}
	code := c.Query("code")
	if code == "" {
		httpapi.Error(c, http.StatusBadRequest, "MISSING_CODE", "缺少 GitHub 授权 code", nil)
		return
	}
	clearCookie(c, h.StateCookie, "/api/v2/auth")

	ctx := c.Request.Context()
	accessToken, err := h.OAuth.Exchange(ctx, code)
	if err != nil {
		slog.Error("github token exchange failed", "err", err)
		httpapi.Error(c, http.StatusBadGateway, "GITHUB_UPSTREAM", "GitHub 授权失败："+err.Error(), nil)
		return
	}
	gh, err := h.OAuth.FetchUser(ctx, accessToken)
	if err != nil {
		slog.Error("github user fetch failed", "err", err)
		httpapi.Error(c, http.StatusBadGateway, "GITHUB_UPSTREAM", "获取 GitHub 用户失败："+err.Error(), nil)
		return
	}
	displayName := gh.Name
	if displayName == "" {
		displayName = gh.Login
	}
	user, err := h.Store.UpsertGitHubUser(ctx, strconv.FormatInt(gh.ID, 10), gh.Login, gh.Email, displayName)
	if err != nil {
		slog.Error("upsert user failed", "err", err)
		httpapi.Error(c, http.StatusInternalServerError, "INTERNAL", "保存用户失败", nil)
		return
	}

	// 首次登录自动创建个人工作区
	if err := h.ensurePersonalTeam(ctx, user); err != nil {
		slog.Error("ensure personal team failed", "err", err)
	}

	token := security.RandomHex(32)
	expires := time.Now().UTC().Add(h.SessionTTL)
	if err := h.Store.CreateSession(ctx, uuid.NewString(), user.ID, security.HashToken(token), expires); err != nil {
		slog.Error("create session failed", "err", err)
		httpapi.Error(c, http.StatusInternalServerError, "INTERNAL", "创建会话失败", nil)
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     h.CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  expires,
		MaxAge:   int(h.SessionTTL.Seconds()),
	})
	c.Redirect(http.StatusFound, h.PublicURL)
}

func (h *Handlers) Logout(c *gin.Context) {
	token, err := c.Cookie(h.CookieName)
	if err == nil && token != "" {
		if err := h.Store.DeleteSession(c.Request.Context(), security.HashToken(token)); err != nil {
			slog.Error("delete session failed", "err", err)
		}
	}
	clearCookie(c, h.CookieName, "/")
	c.Status(http.StatusNoContent)
}

func (h *Handlers) ensurePersonalTeam(ctx context.Context, user db.User) error {
	teams, err := h.Store.ListTeamsForUser(ctx, user.ID)
	if err != nil {
		return err
	}
	if len(teams) > 0 {
		return nil
	}
	_, err = h.Store.CreateTeamWithOwner(ctx, uuid.NewString(), user.DisplayName+" 的个人空间", user.ID)
	return err
}

func clearCookie(c *gin.Context, name, path string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     path,
		HttpOnly: true,
		MaxAge:   -1,
	})
}
