package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/beancount-gs/api/internal/auth"
	"github.com/beancount-gs/api/internal/beancount"
	"github.com/beancount-gs/api/internal/config"
	"github.com/beancount-gs/api/internal/db"
	httpapi "github.com/beancount-gs/api/internal/http"
	"github.com/beancount-gs/api/internal/http/gen"
	"github.com/beancount-gs/api/internal/ledger"
	mcpapi "github.com/beancount-gs/api/internal/mcp"
	"github.com/beancount-gs/api/internal/repository"
	"github.com/gin-gonic/gin"
)

// version 由构建时注入：-ldflags "-X main.version=v2.0.0"
var version = "v2.0.0-dev"

func main() {
	cfg := config.Load()
	flag.IntVar(&cfg.Port, "p", cfg.Port, "服务端口")
	flag.StringVar(&cfg.DBPath, "db", cfg.DBPath, "SQLite 数据库路径")
	flag.Parse()
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	store, err := db.Open(cfg.DBPath)
	if err != nil {
		slog.Error("failed to open database", "path", cfg.DBPath, "err", err)
		os.Exit(1)
	}
	defer store.Close()
	slog.Info("database ready", "path", cfg.DBPath)
	if err := repository.EnsureDataRoot(cfg.DataRoot); err != nil {
		slog.Error("failed to prepare data root", "path", cfg.DataRoot, "err", err)
		os.Exit(1)
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	api := router.Group("/api/v2")
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	api.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"version": version})
	})
	// 暴露 OpenAPI 契约，供文档页 / 客户端生成使用
	api.GET("/openapi.json", func(c *gin.Context) {
		swagger, err := gen.GetSwagger()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, swagger)
	})

	oauth := &auth.GitHubOAuth{
		ClientID:     cfg.GitHubClientID,
		ClientSecret: cfg.GitHubClientSecret,
		RedirectURL:  cfg.AppPublicURL + "/api/v2/auth/github/callback",
		HTTP:         &http.Client{Timeout: 15 * time.Second},
	}
	authHandlers := &auth.Handlers{
		Store:       store,
		OAuth:       oauth,
		PublicURL:   cfg.AppPublicURL,
		CookieName:  cfg.SessionCookie,
		StateCookie: cfg.StateCookie,
		SessionTTL:  30 * 24 * time.Hour,
	}

	api.GET("/auth/github/login", authHandlers.Login)
	api.GET("/auth/github/callback", authHandlers.Callback)
	api.POST("/auth/logout", authHandlers.Logout)

	authed := api.Group("", httpapi.RequireSession(store, cfg.SessionCookie))
	authed.GET("/users/me", usersMe)

	teamsHandlers := &httpapi.TeamsHandlers{Store: store}
	ledgerHandlers := &httpapi.LedgerHandlers{
		Store:       store,
		DataRoot:    cfg.DataRoot,
		TemplateDir: cfg.TemplateDir,
	}
	authed.GET("/teams", teamsHandlers.List)
	authed.POST("/teams", teamsHandlers.Create)
	authed.GET("/ledgers", ledgerHandlers.List)
	authed.POST("/ledgers", ledgerHandlers.Create)
	authed.GET("/ledgers/:ledger_id", ledgerHandlers.Get)
	authed.GET("/ledgers/:ledger_id/revision", ledgerHandlers.Revision)

	ledgerService := &ledger.Service{Store: store, Engine: beancount.CmdEngine{}}
	txnHandlers := &httpapi.TransactionHandlers{Store: store, Service: ledgerService}
	authed.GET("/ledgers/:ledger_id/transactions", txnHandlers.List)
	authed.POST("/ledgers/:ledger_id/transactions", txnHandlers.Create)
	authed.GET("/ledgers/:ledger_id/transactions/:transaction_id", txnHandlers.Get)
	authed.PUT("/ledgers/:ledger_id/transactions/:transaction_id", txnHandlers.Update)
	authed.DELETE("/ledgers/:ledger_id/transactions/:transaction_id", txnHandlers.Delete)
	authed.GET("/ledgers/:ledger_id/transactions/:transaction_id/raw", txnHandlers.RawText)
	authed.PUT("/ledgers/:ledger_id/transactions/:transaction_id/raw", txnHandlers.UpdateRawText)

	accountHandlers := &httpapi.AccountHandlers{Store: store, Service: ledgerService}
	authed.GET("/ledgers/:ledger_id/accounts", accountHandlers.List)
	authed.POST("/ledgers/:ledger_id/accounts", accountHandlers.Open)
	authed.GET("/ledgers/:ledger_id/accounts/:account", accountHandlers.Get)
	authed.POST("/ledgers/:ledger_id/accounts/:account", accountHandlers.Close)
	authed.POST("/ledgers/:ledger_id/accounts/:account/balance", accountHandlers.Balance)
	authed.GET("/ledgers/:ledger_id/account-types", accountHandlers.ListTypes)
	authed.POST("/ledgers/:ledger_id/account-types", accountHandlers.AddType)

	statsHandlers := &httpapi.StatsHandlers{Store: store, Service: ledgerService}
	authed.GET("/ledgers/:ledger_id/stats/total", statsHandlers.Total)
	authed.GET("/ledgers/:ledger_id/stats/payee", statsHandlers.Payee)
	authed.GET("/ledgers/:ledger_id/stats/account-trend", statsHandlers.Trend)
	authed.GET("/ledgers/:ledger_id/stats/account-flow", statsHandlers.Flow)

	keyHandlers := &httpapi.KeyHandlers{Store: store}
	authed.GET("/api-keys", keyHandlers.List)
	authed.POST("/api-keys", keyHandlers.Create)
	authed.DELETE("/api-keys/:key_id", keyHandlers.Revoke)

	// MCP Server（Streamable HTTP，Bearer API Key 认证）
	mcpServer := mcpapi.New(store, ledgerService, cfg.DataRoot)
	api.Any("/mcp", httpapi.RequireApiKey(store), gin.WrapH(mcpServer))

	addr := ":" + strconv.Itoa(cfg.Port)
	srv := &http.Server{Addr: addr, Handler: router}

	go func() {
		slog.Info("beancount-gs api listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
	slog.Info("server stopped")
}

func usersMe(c *gin.Context) {
	user := httpapi.CurrentUser(c)
	if user == nil {
		httpapi.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未登录", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":            user.ID,
		"github_login":  user.GitHubLogin,
		"display_name":  user.DisplayName,
		"email":         user.Email,
		"created_at":    user.CreatedAt,
	})
}
