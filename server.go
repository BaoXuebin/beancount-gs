package main

import (
	"github.com/beancount-gs/script"
	"github.com/beancount-gs/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

func InitServerFiles() error {
	dataPath := script.GetServerConfig().DataPath
	// 账本目录不存在，则创建
	if !script.FileIfExist(dataPath) {
		return script.MkDir(dataPath)
	}
	return nil
}

func LoadServerCache() error {
	err := script.LoadLedgerConfigMap()
	if err != nil {
		return err
	}
	return script.LoadLedgerAccountsMap()
}

func AuthorizedHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ledgerId := c.GetHeader("ledgerId")
		ledgerConfig := script.GetLedgerConfig(ledgerId)
		if ledgerConfig != nil {
			c.Set("LedgerConfig", ledgerConfig)
			c.Next()
		} else {
			service.Unauthorized(c)
			c.Abort()
		}
	}
}

func RegisterRouter(router *gin.Engine) {
	// fix wildcard and static file router conflict, https://github.com/gin-gonic/gin/issues/360
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/web")
	})
	router.StaticFS("/web", http.Dir("./public"))
	router.POST("/api/ledger", service.OpenOrCreateLedger)
	authorized := router.Group("/api/auth/")
	authorized.Use(AuthorizedHandler())
	{
		// need authorized
		authorized.GET("/account/valid", service.QueryValidAccount)
		authorized.GET("/account/all", service.QueryAllAccount)
		authorized.GET("/account/type", service.QueryAccountType)
		authorized.GET("/stats/months", service.MonthsList)
		authorized.GET("/stats/total", service.StatsTotal)
		authorized.GET("/transactions", service.QueryTransactions)
		authorized.GET("/transactions/payee", service.QueryTransactionsPayee)
		authorized.GET("/transactions/template", service.QueryTransactionsTemplate)
		authorized.GET("/tags", service.QueryTags)
		authorized.GET("/file/dir", service.QueryLedgerSourceFileDir)
		authorized.GET("/file/content", service.QueryLedgerSourceFileContent)
		authorized.POST("/file", service.UpdateLedgerSourceFileContent)

		// 兼容旧版本
		authorized.GET("/entry", service.QueryTransactions)
		authorized.GET("/payee", service.QueryTransactionsPayee)
		authorized.GET("/transaction/template", service.QueryTransactionsTemplate)
	}
}

func main() {
	// 读取配置文件
	err := script.LoadServerConfig()
	if err != nil {
		script.LogSystemError("Failed to load server config, " + err.Error())
		return
	}
	// 初始化账本文件结构
	err = InitServerFiles()
	if err != nil {
		script.LogSystemError("Failed to init server files, " + err.Error())
		return
	}
	// 加载缓存
	err = LoadServerCache()
	if err != nil {
		script.LogSystemError("Failed to load server cache, " + err.Error())
		return
	}
	router := gin.Default()
	// 注册路由
	RegisterRouter(router)
	// 启动服务
	var port = ":3001"
	err = router.Run(port)
	if err != nil {
		script.LogSystemError("Failed to start server, " + err.Error())
	}
}
