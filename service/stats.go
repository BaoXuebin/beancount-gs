package service

import (
	"fmt"
	"github.com/beancount-gs/script"
	"github.com/gin-gonic/gin"
	"strings"
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

type statsAccountTypeTotal struct {
	AccountType string
	Amount      string
}

func StatsTotal(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	queryParams := script.GetQueryParams(c)
	selectBql := fmt.Sprintf("SELECT '\\', root(account, 1), '\\', sum(convert(value(position), '%s')), '\\'", ledgerConfig.OperatingCurrency)
	accountTypeTotalList := make([]statsAccountTypeTotal, 0)
	err := script.BQLQueryListByCustomSelect(ledgerConfig, selectBql, &queryParams, &accountTypeTotalList)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	result := make(map[string]string, 0)
	for _, total := range accountTypeTotalList {
		fields := strings.Fields(total.Amount)
		if len(fields) > 1 {
			result[total.AccountType] = fields[0]
		}
	}

	OK(c, result)
}
