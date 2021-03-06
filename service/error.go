package service

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "data": data})
}

func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusOK, gin.H{"code": 400, "message": message})
}

func Unauthorized(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 401})
}

func InternalError(c *gin.Context, message string) {
	c.JSON(http.StatusOK, gin.H{"code": 500, "message": message})
}

func TransactionNotBalance(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 1001})
}

func LedgerIsNotExist(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 1006})
}

func LedgerIsNotAllowAccess(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 1006})
}

func DuplicateAccount(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 1007})
}

func ServerSecretNotMatch(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 1008})
}
