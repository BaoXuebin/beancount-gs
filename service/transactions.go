package service

import (
	"github.com/beancount-gs/script"
	"github.com/gin-gonic/gin"
	"strconv"
)

type Transactions struct {
	Id        string `bql:"id"`
	Date      string `bql:"date"`
	payee     string
	narration string
	account   string
	position  string
	tags      string
}

func getQueryModel(c *gin.Context) script.QueryParams {
	var queryParams script.QueryParams
	if c.Query("year") != "" {
		val, err := strconv.Atoi(c.Query("year"))
		if err == nil {
			queryParams.Year = val
		}
	}
	if c.Query("month") != "" {
		val, err := strconv.Atoi(c.Query("month"))
		if err == nil {
			queryParams.Month = val
		}
	}
	if c.Query("type") != "" {
		queryParams.AccountType = c.Query("type")
	}
	return queryParams
}

func QueryTransactions(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	queryParams := getQueryModel(c)
	transactions := make([]Transactions, 0)
	err := script.BQLQuery(ledgerConfig, queryParams, transactions)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	OK(c, transactions)
}
