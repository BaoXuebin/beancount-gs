package service

import "github.com/gin-gonic/gin"

func QueryVersion(c *gin.Context) {
	OK(c, "v1.0.1")
}
