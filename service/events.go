package service

import (
	"fmt"
	"github.com/beancount-gs/script"
	"github.com/gin-gonic/gin"
	"strings"
)

type Event struct {
	Date        string `form:"date" binding:"required" json:"date"`
	Type        string `form:"type" binding:"required" json:"type"`
	Description string `form:"description" binding:"required" json:"description"`
}

func GetAllEvents(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	output := script.BeanReportAllEvents(ledgerConfig)
	script.LogInfo(ledgerConfig.Mail, output)

	events := make([]Event, 0)
	lines := strings.Split(output, "\n")
	// foreach lines
	for idx, line := range lines {
		if idx < 2 || idx > len(lines)-3 {
			continue
		}
		if strings.Trim(line, " ") == "" {
			continue
		}
		// split line by " "
		words := strings.Fields(line)
		events = append(events, Event{
			Date:        words[0],
			Type:        words[1],
			Description: words[2],
		})
	}
	OK(c, events)
}

func AddEvent(c *gin.Context) {
	var event Event
	if err := c.ShouldBindJSON(&event); err != nil {
		BadRequest(c, err.Error())
		return
	}

	ledgerConfig := script.GetLedgerConfigFromContext(c)
	filePath := script.GetLedgerEventsFilePath(ledgerConfig.DataPath)

	line := fmt.Sprintf("%s event \"%s\" \"%s\"", event.Date, event.Type, event.Description)
	// 写入文件
	err := script.AppendFileInNewLine(filePath, line)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	OK(c, event)
}

func DeleteEvent(c *gin.Context) {
	var event Event
	if err := c.ShouldBindJSON(&event); err != nil {
		BadRequest(c, err.Error())
		return
	}

	ledgerConfig := script.GetLedgerConfigFromContext(c)
	filePath := script.GetLedgerEventsFilePath(ledgerConfig.DataPath)

	line := fmt.Sprintf("%s event \"%s\" \"%s\"", event.Date, event.Type, event.Description)
	err := script.DeleteLinesWithText(filePath, line)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	OK(c, nil)
}
