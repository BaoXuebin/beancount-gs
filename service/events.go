package service

import "github.com/gin-gonic/gin"

type Event struct {
	Date        string `json:"date"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

func GetAllEvents(c *gin.Context) {
	OK(c, nil)
}

func AddEvent(c *gin.Context) {
	OK(c, nil)
}
