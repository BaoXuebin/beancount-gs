package service

import (
	"fmt"
	"github.com/beancount-gs/script"
	"github.com/gin-gonic/gin"
	"sort"
	"strings"
)

type Event struct {
	Date        string   `form:"date" binding:"required" json:"date"`
	Stage       string   `form:"stage" json:"stage"`
	Type        string   `form:"type" json:"type"`
	Types       []string `form:"types" json:"types"`
	Description string   `form:"description" binding:"required" json:"description"`
}

// Events 切片包含多个事件
type Events []Event

func (e Events) Len() int {
	return len(e)
}

func (e Events) Less(i, j int) bool {
	return strings.Compare(e[i].Date, e[j].Date) < 0
}

func (e Events) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func GetAllEvents(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)

	beanFilePath := script.GetLedgerEventsFilePath(ledgerConfig.DataPath)
	bytes, err := script.ReadFile(beanFilePath)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	lines := strings.Split(string(bytes), "\n")
	events := Events{}
	// foreach lines
	for _, line := range lines {
		if strings.Trim(line, " ") == "" {
			continue
		}
		// split line by " "
		words := strings.Fields(line)
		if len(words) < 4 {
			continue
		}
		if words[1] != "event" {
			continue
		}
		events = append(events, Event{
			Date:        words[0],
			Type:        strings.ReplaceAll(words[2], "\"", ""),
			Description: strings.ReplaceAll(words[3], "\"", ""),
		})
	}
	if len(events) > 0 {
		// events 按时间倒序排列
		sort.Sort(sort.Reverse(events))
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

	if event.Type != "" {
		event.Types = []string{event.Type}
	}

	// 定义Event类型的数组
	events := make([]Event, 0)

	if event.Types != nil {
		for _, t := range event.Types {
			events = append(events, Event{
				Date:        event.Date,
				Type:        t,
				Description: event.Description,
			})
			line := fmt.Sprintf("%s event \"%s\" \"%s\"", event.Date, t, event.Description)
			// 写入文件
			err := script.AppendFileInNewLine(filePath, line)
			if err != nil {
				InternalError(c, err.Error())
				return
			}
		}
	}

	OK(c, events)
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
