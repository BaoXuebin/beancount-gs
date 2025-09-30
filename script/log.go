package script

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// ==================== 基础日志函数 ====================

// Info级别日志函数组

// LogInfo 记录信息级别的日志
// ledgerName: 账本名称，用于标识日志来源
// message: 需要记录的日志信息
func LogInfo(ledgerName string, message string) {
	fmt.Printf("[Info] [%s] [%s]: %s\n", time.Now().Format("2006-01-02 15:04:05"), ledgerName, message)
}

// LogSystemInfo 记录系统信息日志
// message: 要记录的系统信息消息
func LogSystemInfo(message string) {
	LogInfo("System", message)
}

// Warn级别日志函数组

// LogWarn 记录警告级别的日志
// ledgerName: 账本名称，用于标识日志来源
// context: 日志上下文，提供更多分类信息
// format: 格式化字符串，定义日志消息格式
// args: 可变参数，用于填充格式化字符串
func LogWarn(ledgerName string, context string, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Printf("[Warn] [%s] [%s][%s]: %s\n",
		time.Now().Format("2006-01-02 15:04:05"),
		ledgerName,
		context,
		message)
}

// LogSystemWarn 记录系统级警告日志
// context: 日志上下文标识
// format: 格式化字符串
// args: 格式化参数
func LogSystemWarn(context string, format string, args ...interface{}) {
	LogWarn("System", context, format, args...)
}

// Error级别日志函数组

// LogError 记录错误日志，格式为：[Error] [时间] [账本名称]: 错误信息
// ledgerName: 账本名称
// message: 需要记录的错误信息
func LogError(ledgerName string, message string) {
	fmt.Printf("[Error] [%s] [%s]: %s\n", time.Now().Format("2006-01-02 15:04:05"), ledgerName, message)
}

// LogErrorDetailed 记录带有详细信息的错误日志
// ledgerName: 账本名称，用于标识日志来源
// context: 上下文信息，帮助定位错误发生的场景
// format: 格式化字符串，用于构建错误信息
// args: 格式化字符串的参数
// 日志格式为：[Error] [时间] [账本名称][上下文]: 错误信息
func LogErrorDetailed(ledgerName string, context string, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Printf("[Error] [%s] [%s][%s]: %s\n",
		time.Now().Format("2006-01-02 15:04:05"),
		ledgerName,
		context,
		message)
}

// LogSystemError 记录系统级错误信息
// message: 错误描述信息
// 该函数是对LogError的封装，专门用于记录系统模块的错误日志
func LogSystemError(message string) {
	LogError("System", message)
}

// LogSystemErrorDetailed 记录系统级错误日志，包含详细上下文信息
// context: 错误发生的上下文环境
// format: 格式化字符串，用于描述错误信息
// args: 格式化字符串的参数
func LogSystemErrorDetailed(context string, format string, args ...interface{}) {
	LogErrorDetailed("System", context, format, args...)
}

// Debug级别日志函数组

// LogDebug 在调试模式下记录调试日志，格式为：[Debug] [时间] [账本名称]: 消息
// 仅当 IsDebugMode() 返回 true 时才会输出日志
func LogDebug(ledgerName string, message string) {
	if IsDebugMode() {
		fmt.Printf("[Debug] [%s] [%s]: %s\n", time.Now().Format("2006-01-02 15:04:05"), ledgerName, message)
	}
}

// LogDebugDetailed 在调试模式下记录详细的调试日志
// ledgerName: 账本名称，用于标识日志来源
// context: 上下文信息，提供额外的日志分类
// format: 格式化字符串，用于构建日志消息
// args: 格式化字符串的参数
// 日志格式: [Debug] [时间] [账本名称][上下文]: 消息内容
// 注意: 仅在调试模式(IsDebugMode返回true)下输出日志
func LogDebugDetailed(ledgerName string, context string, format string, args ...interface{}) {
	if IsDebugMode() {
		message := fmt.Sprintf(format, args...)
		fmt.Printf("[Debug] [%s] [%s][%s]: %s\n",
			time.Now().Format("2006-01-02 15:04:05"),
			ledgerName,
			context,
			message)
	}
}

// LogSystemDebug 记录系统级别的调试日志
// message: 需要记录的调试信息
func LogSystemDebug(message string) {
	LogDebug("System", message)
}

// LogSystemDebugDetailed 记录系统级调试日志，包含详细上下文信息
// context: 日志上下文标识
// format: 日志格式字符串
// args: 格式化参数
func LogSystemDebugDetailed(context string, format string, args ...interface{}) {
	LogDebugDetailed("System", context, format, args...)
}

// 特殊功能日志函数

// LogBQLQueryDebug 在调试模式下记录BQL查询日志
// ledgerName: 账本名称
// context: 查询上下文信息
// format: 日志格式字符串
// args: 格式化参数
// 注意: 仅在调试模式开启时才会记录日志
func LogBQLQueryDebug(ledgerName string, context string, format string, args ...interface{}) {
	if IsDebugMode() {
		LogDebugDetailed(ledgerName, "BQL:"+context, format, args...)
	}
}

// ==================== 带请求ID的调试上下文 ====================

// DebugContext 调试上下文
type DebugContext struct {
	RequestID string
	DebugMode bool
}

// NewDebugContext 创建新的调试上下文（不依赖gin.Context）
func NewDebugContext() *DebugContext {
	debugMode := IsDebugMode()
	requestID := generateRequestID()

	if debugMode {
		LogSystemDebugDetailed("RequestDebug",
			"=== DEBUG MODE START ===\nRequest ID: %s", requestID)
	}

	return &DebugContext{
		RequestID: requestID,
		DebugMode: debugMode,
	}
}

// SetupDebugContext 设置调试上下文
func SetupDebugContext(c *gin.Context) *DebugContext {
	debugMode := IsDebugMode()
	requestID := generateRequestID()

	if debugMode {
		LogSystemDebugDetailed("RequestDebug",
			"=== DEBUG MODE START ===\nRequest ID: %s\nURL: %s",
			requestID, c.Request.URL.String())
	}

	return &DebugContext{
		RequestID: requestID,
		DebugMode: debugMode,
	}
}

// FinishDebugContext 结束调试上下文
func (ctx *DebugContext) FinishDebugContext() {
	if ctx.DebugMode {
		LogSystemDebugDetailed("RequestDebug",
			"=== DEBUG MODE END ===\nRequest ID: %s", ctx.RequestID)
	}
}

// LogDebugF 带请求ID的调试日志（格式化版本）
func (ctx *DebugContext) LogDebugF(ledgerName, context, format string, args ...interface{}) {
	if ctx.DebugMode {
		format = "[ReqID:" + ctx.RequestID + "] " + format
		LogDebugDetailed(ledgerName, context, format, args...)
	}
}

// LogSystemDebug 带请求ID的系统调试日志
func (ctx *DebugContext) LogSystemDebug(module, format string, args ...interface{}) {
	if ctx.DebugMode {
		format = "[ReqID:" + ctx.RequestID + "] " + format
		LogSystemDebugDetailed(module, format, args...)
	}
}

// LogInfo 带请求ID的信息日志
func (ctx *DebugContext) LogInfo(ledgerName, message string) {
	message = "[ReqID:" + ctx.RequestID + "] " + message
	LogInfo(ledgerName, message)
}

// LogSystemInfo 带请求ID的系统信息日志
func (ctx *DebugContext) LogSystemInfo(message string) {
	ctx.LogInfo("System", message)
}

// LogWarn 带请求ID的警告日志
func (ctx *DebugContext) LogWarn(ledgerName, context, format string, args ...interface{}) {
	format = "[ReqID:" + ctx.RequestID + "] " + format
	LogWarn(ledgerName, context, format, args...)
}

// LogSystemWarn 带请求ID的系统警告日志
func (ctx *DebugContext) LogSystemWarn(context, format string, args ...interface{}) {
	ctx.LogWarn("System", context, format, args...)
}

// LogError 带请求ID的错误日志
func (ctx *DebugContext) LogError(ledgerName, message string) {
	message = "[ReqID:" + ctx.RequestID + "] " + message
	LogError(ledgerName, message)
}

// LogErrorDetailed 带请求ID的详细错误日志
func (ctx *DebugContext) LogErrorDetailed(ledgerName, context, format string, args ...interface{}) {
	format = "[ReqID:" + ctx.RequestID + "] " + format
	LogErrorDetailed(ledgerName, context, format, args...)
}

// LogSystemError 带请求ID的系统错误日志
func (ctx *DebugContext) LogSystemError(message string) {
	ctx.LogError("System", message)
}

// LogSystemErrorDetailed 带请求ID的系统详细错误日志
func (ctx *DebugContext) LogSystemErrorDetailed(context, format string, args ...interface{}) {
	ctx.LogErrorDetailed("System", context, format, args...)
}

// LogBQLQueryDebug 带请求ID的BQL查询调试日志
func (ctx *DebugContext) LogBQLQueryDebug(ledgerName, context, format string, args ...interface{}) {
	if ctx.DebugMode {
		format = "[ReqID:" + ctx.RequestID + "] " + format
		LogBQLQueryDebug(ledgerName, context, format, args...)
	}
}

// 生成请求ID
func generateRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
