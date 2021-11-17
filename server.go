package main

import (
	"github.com/beancount-gs/script"
	"github.com/gin-gonic/gin"
	"net/http"
)

var GlobalConfig Config

func InitLedgerFiles() {
	// 账本目录不存在，则创建
	if !script.FileIfExist(GlobalConfig.DataPath) {
		script.MkDir(GlobalConfig.DataPath)
		script.LogInfo("Success mkdir " + GlobalConfig.DataPath)
	}
}

func LoadLedgerCache() {

}

func RegisterRoute(route *gin.Engine) {
	route.StaticFS("/", http.Dir("./public"))
}

func main() {
	// 默认端口号
	var port = ":3001"
	// 读取配置文件
	GlobalConfig = LoadConfig(GlobalConfig)
	// 初始化账本文件结构
	InitLedgerFiles()
	// 加载缓存
	LoadLedgerCache()
	route := gin.Default()
	// 注册路由
	RegisterRoute(route)
	// 启动服务
	_ = http.ListenAndServe(port, nil)
}
