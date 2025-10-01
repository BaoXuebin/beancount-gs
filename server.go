package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/beancount-gs/script"
	"github.com/beancount-gs/service"
	"github.com/beancount-gs/utils/venv"
	"github.com/gin-gonic/gin"
)

// 全局变量，方便其他模块使用
var venvExecutor *venv.VenvExecutor
var venvPath string // 新增：虚拟环境路径变量

/*
 * 初始化服务器文件
 * 检查账本目录是否存在，如果不存在则创建
 * 返回error，表示操作是否成功
 */
func InitServerFiles() error {
	dataPath := script.GetServerConfig().DataPath
	// 账本目录不存在，则创建
	if dataPath != "" && !script.FileIfExist(dataPath) {
		return script.MkDir(dataPath)
	}
	return nil
}

/*
 * 加载服务器缓存
 * 加载账本配置映射和账户映射
 * 返回error，表示加载过程中是否出错
 */
func LoadServerCache() error {
	err := script.LoadLedgerConfigMap()
	if err != nil {
		return err
	}
	return script.LoadLedgerAccountsMap()
}

/*
 * 授权中间件
 * 检查请求头中的ledgerId是否有效
 * 如果有效则继续处理请求，否则返回未授权错误
 */
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

/*
 * 注册路由
 * 配置静态文件服务、API路由和需要授权的路由组
 */
func RegisterRouter(router *gin.Engine) {
	// fix wildcard and static file router conflict, https://github.com/gin-gonic/gin/issues/360
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/web")
	})
	router.StaticFS("/web", http.Dir("./public"))
	// 公开API路由，无需授权
	router.GET("/api/version", service.QueryVersion)
	router.POST("/api/check", service.CheckBeancount)
	router.GET("/api/config", service.QueryServerConfig)
	router.POST("/api/config", service.UpdateServerConfig)
	router.GET("/api/ledger", service.QueryLedgerList)
	router.POST("/api/ledger", service.OpenOrCreateLedger)
	// 需要授权的API路由组
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
		authorized.GET("/stats/account/flow", service.StatsAccountSankey)
		authorized.GET("/stats/month/total", service.StatsMonthTotal)
		authorized.GET("/stats/month/calendar", service.StatsMonthCalendar)
		authorized.GET("/stats/commodity/price", service.StatsCommodityPrice)
		authorized.GET("/transaction/detail", service.QueryTransactionDetailById)
		authorized.GET("/transaction/raw", service.QueryTransactionRawTextById)
		authorized.GET("/transaction", service.QueryTransactions)
		authorized.POST("/transaction", service.AddTransactions)
		authorized.POST("/transaction/raw", service.UpdateTransactionRawTextById)
		authorized.DELETE("/transaction", service.DeleteTransactionById)
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

// initVenvExecutor 初始化虚拟环境执行器
func initVenvExecutor(venvDir string) {
	venvPath = venvDir
	script.SetVenvPath(venvDir) // 设置路径到 script 包

	// 检查虚拟环境是否存在
	if !venv.CheckVenvExists(venvPath) {
		script.LogSystemError("虚拟环境不存在，请先运行 setup script: " + venvPath)
		fmt.Println("警告: 虚拟环境不存在，某些功能可能无法正常工作")
		fmt.Println("请运行: ./start_dev.sh 或手动创建虚拟环境")
		return
	}

	venvExecutor = venv.NewVenvExecutor(venvPath)
	script.SetVenvExecutor(venvExecutor) // 设置执行器到 script 包

	// 测试 bean-query 是否可用
	_, err := venvExecutor.GetCommandPath("bean-query")
	if err != nil {
		script.LogSystemError("bean-query 不可用: " + err.Error())
		fmt.Println("警告: bean-query 命令不可用，价格查询功能将受限")
	} else {
		script.LogSystemInfo("虚拟环境初始化成功: bean-query 可用, 路径: " + venvPath)
		fmt.Println("虚拟环境初始化成功: " + venvPath)
	}
}

func main() {
	var secret string
	var port int
	var debugFlag bool
	var venvDir string // 新增：虚拟环境目录参数

	flag.StringVar(&secret, "secret", "", "服务器密钥")
	flag.IntVar(&port, "p", 10000, "端口号")
	flag.BoolVar(&debugFlag, "debug", false, "调试模式")
	flag.StringVar(&venvDir, "venv", ".env_beancount-v3", "虚拟环境目录名称，默认值为 .env_beancount-v3") // 新增参数

	flag.Parse()

	// 初始化虚拟环境执行器
	initVenvExecutor(venvDir)

	// 读取配置文件
	err := script.LoadServerConfig()
	if err != nil {
		script.LogSystemError("Failed to load server config, " + err.Error())
		return
	}

	// 如果命令行指定了debug参数，覆盖配置文件中的设置
	if debugFlag {
		err = script.SetDebugMode(true)
		if err != nil {
			fmt.Println("Warning: Failed to set debug mode:", err)
		}
	}

	// 现在可以在任何地方使用 script.IsDebugMode() 来检查调试模式
	if script.IsDebugMode() {
		fmt.Println("调试模式已启用")
	} else {
		fmt.Println("调试模式未启用")
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
