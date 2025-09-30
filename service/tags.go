package service

import (
	"github.com/beancount-gs/script"
	"github.com/gin-gonic/gin"
)

type Tags struct {
	Value string `bql:"distinct tags" json:"value"`
}

func QueryTags(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	tags := make([]Tags, 0)
	err := script.BQLQueryList(ledgerConfig, nil, &tags)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	result := make([]string, 0)
	for _, t := range tags {
		if t.Value != "" {
			result = append(result, t.Value)
		}
	}

	OK(c, result)
}
