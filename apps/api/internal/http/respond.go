package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorBody struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func JSON(c *gin.Context, status int, body any) {
	c.JSON(status, body)
}

func Error(c *gin.Context, status int, code, message string, details map[string]any) {
	c.AbortWithStatusJSON(status, ErrorBody{
		Code:    code,
		Message: message,
		Details: details,
	})
}

func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, "VALIDATION", message, nil)
}
