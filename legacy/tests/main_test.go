package tests

import (
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPingRoute(t *testing.T) {
	// 设置Gin的模式为测试模式
	gin.SetMode(gin.TestMode)

	// 创建一个Gin引擎
	r := gin.Default()

	// 创建一个模拟的HTTP请求
	req, err := http.NewRequest(http.MethodGet, "/ping", nil)
	assert.NoError(t, err)

	// 使用httptest包创建一个ResponseRecorder，用于记录响应
	w := httptest.NewRecorder()

	// 使用Gin的ServeHTTP方法处理请求
	r.ServeHTTP(w, req)

	// 断言状态码为200
	assert.Equal(t, http.StatusOK, w.Code)

	// 断言响应体中的内容
	assert.JSONEq(t, `{"message": "pong"}`, w.Body.String())
}
