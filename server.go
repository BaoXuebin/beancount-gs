package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func InitLedgerFiles() {

}

func LoadLedgerCache() {

}

func RegisterRoute(route *gin.Engine) {
	route.StaticFS("/", http.Dir("./public"))
}

func main() {
	// 默认端口号
	var port = ":3001"
	// 初始化账本文件结构
	InitLedgerFiles()
	// 加载缓存
	LoadLedgerCache()
	// 启动服务
	route := gin.Default()
	// 注册路由
	RegisterRoute(route)
	_ = http.ListenAndServe(port, nil)
}
