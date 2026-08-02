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

	"github.com/beancount-gs/api/internal/http/gen"
	"github.com/gin-gonic/gin"
)

// version 由构建时注入：-ldflags "-X main.version=v2.0.0"
var version = "v2.0.0-dev"

func main() {
	var port int
	flag.IntVar(&port, "p", 10000, "服务端口")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
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

	addr := ":" + strconv.Itoa(port)
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
