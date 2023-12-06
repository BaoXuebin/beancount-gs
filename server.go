package main

import (
	"flag"
	"fmt"
	"github.com/beancount-gs/script"
	"github.com/beancount-gs/service"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"os"
)

func InitServerFiles() error {
	dataPath := script.GetServerConfig().DataPath
	// 账本目录不存在，则创建
	if dataPath != "" && !script.FileIfExist(dataPath) {
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
	router.GET("/api/version", service.QueryVersion)
	router.POST("/api/check", service.CheckBeancount)
	router.GET("/api/config", service.QueryServerConfig)
	router.POST("/api/config", service.UpdateServerConfig)
	router.GET("/api/ledger", service.QueryLedgerList)
	router.POST("/api/ledger", service.OpenOrCreateLedger)
	authorized := router.Group("/api/auth/")
	authorized.Use(AuthorizedHandler())
	{
		// need authorized
		authorized.GET("/account/valid", service.QueryValidAccount)
		authorized.GET("/account/all", service.QueryAllAccount)
		authorized.GET("/account/type", service.QueryAccountType)
		authorized.POST("/account", service.AddAccount)
		authorized.POST("/account/type", service.AddAccountType)
		authorized.POST("/account/close", service.CloseAccount)
		authorized.POST("/account/icon", service.ChangeAccountIcon)
		authorized.POST("/account/balance", service.BalanceAccount)
		authorized.POST("/account/refresh", service.RefreshAccountCache)
		authorized.POST("/commodity/price", service.SyncCommodityPrice)
		authorized.GET("/commodity/currencies", service.QueryAllCurrencies)
		authorized.GET("/stats/months", service.MonthsList)
		authorized.GET("/stats/total", service.StatsTotal)
		authorized.GET("/stats/payee", service.StatsPayee)
		authorized.GET("/stats/account/percent", service.StatsAccountPercent)
		authorized.GET("/stats/account/trend", service.StatsAccountTrend)
		authorized.GET("/stats/account/balance", service.StatsAccountBalance)
		authorized.GET("/stats/month/total", service.StatsMonthTotal)
		authorized.GET("/stats/month/calendar", service.StatsMonthCalendar)
		authorized.GET("/stats/commodity/price", service.StatsCommodityPrice)
		authorized.GET("/transaction", service.QueryTransactions)
		authorized.POST("/transaction", service.AddTransactions)
		authorized.POST("/transaction/batch", service.AddBatchTransactions)
		authorized.GET("/transaction/payee", service.QueryTransactionPayees)
		authorized.GET("/transaction/template", service.QueryTransactionTemplates)
		authorized.POST("/transaction/template", service.AddTransactionTemplate)
		authorized.DELETE("/transaction/template", service.DeleteTransactionTemplate)
		authorized.GET("/event/all", service.GetAllEvents)
		authorized.POST("/event", service.AddEvent)
		authorized.DELETE("/event", service.DeleteEvent)
		authorized.GET("/tags", service.QueryTags)
		authorized.GET("/file/dir", service.QueryLedgerSourceFileDir)
		authorized.GET("/file/content", service.QueryLedgerSourceFileContent)
		authorized.POST("/file", service.UpdateLedgerSourceFileContent)
		authorized.POST("/import/alipay", service.ImportAliPayCSV)
		authorized.POST("/import/wx", service.ImportWxPayCSV)
		authorized.POST("/import/icbc", service.ImportICBCCSV)
		authorized.POST("/import/abc", service.ImportABCCSV)
		authorized.GET("/ledger/check", service.CheckLedger)
		authorized.DELETE("/ledger", service.DeleteLedger)
	}
}

func main() {
	var secret string
	var port int
	flag.StringVar(&secret, "secret", "", "服务器密钥")
	flag.IntVar(&port, "p", 10000, "端口号")
	flag.Parse()

	// 读取配置文件
	err := script.LoadServerConfig()
	if err != nil {
		script.LogSystemError("Failed to load server config, " + err.Error())
		return
	}
	serverConfig := script.GetServerConfig()
	// 若 DataPath == "" 则配置未初始化
	if serverConfig.DataPath != "" {
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
	}
	// gin 日志设置
	gin.DisableConsoleColor()
	fs, _ := os.Create("logs/gin.log")
	gin.DefaultWriter = io.MultiWriter(fs, os.Stdout)
	router := gin.Default()
	// 注册路由
	RegisterRouter(router)

	portStr := fmt.Sprintf(":%d", port)
	url := "http://localhost" + portStr
	ip := script.GetIpAddress()
	startLog := "beancount-gs start at " + url
	if ip != "" {
		startLog += " or http://" + ip + portStr
	}
	script.LogSystemInfo(startLog)
	// 打开浏览器
	script.OpenBrowser(url)
	// 打印密钥
	script.LogSystemInfo("Secret token is " + script.GenerateServerSecret(secret))
	// 启动服务
	err = router.Run(portStr)
	if err != nil {
		script.LogSystemError("Failed to start server, " + err.Error())
	}
}
