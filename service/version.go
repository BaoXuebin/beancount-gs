package service

import "github.com/gin-gonic/gin"

func QueryVersion(c *gin.Context) {
	OK(c, "v0.0.1")
}
