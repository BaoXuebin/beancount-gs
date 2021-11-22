package service

import (
	"github.com/beancount-gs/script"
	"github.com/gin-gonic/gin"
)

type YearMonth struct {
	Year  string `bql:"distinct year(date)" json:"year"`
	Month string `bql:"month(date)" json:"month"`
}

func MonthsList(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	yearMonthList := make([]YearMonth, 0)
	err := script.BQLQueryList(ledgerConfig, nil, &yearMonthList)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	months := make([]string, 0)
	for _, yearMonth := range yearMonthList {
		months = append(months, yearMonth.Year+"-"+yearMonth.Month)
	}
	OK(c, months)
}
