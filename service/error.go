package service

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func OK(c *gin.Context, data string) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "data": data})
}

func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": message})
}

func InternalError(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": message})
}

func LedgerIsNotExist(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 1006, "message": "ledger is not exist"})
}

func LedgerIsNotAllowAccess(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 1006, "message": "ledger is not allow access"})
}
