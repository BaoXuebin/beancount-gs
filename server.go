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
	return script.LoadLedgerConfigMap()
}

func RegisterRouter(router *gin.Engine) {
	router.StaticFS("/", http.Dir("./public"))
	router.POST("/api/ledger", service.OpenOrCreateLedger)
}

func main() {
	// 读取配置文件
	err := script.LoadServerConfig()
	if err != nil {
		script.LogError("Failed to load server config, " + err.Error())
		return
	}
	// 初始化账本文件结构
	err = InitServerFiles()
	if err != nil {
		script.LogError("Failed to init server files, " + err.Error())
		return
	}
	// 加载缓存
	err = LoadServerCache()
	if err != nil {
		script.LogError("Failed to load server cache, " + err.Error())
		return
	}
	router := gin.Default()
	// 注册路由
	RegisterRouter(router)
	// 启动服务
	var port = ":3001"
	err = router.Run(port)
	if err != nil {
		script.LogError("Failed to start server, " + err.Error())
	}
}
