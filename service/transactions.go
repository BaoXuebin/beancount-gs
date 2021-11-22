package service

import (
	"github.com/beancount-gs/script"
	"github.com/gin-gonic/gin"
	"strconv"
	"strings"
)

type Transactions struct {
	Id              string   `bql:"id" json:"id"`
	Date            string   `bql:"date" json:"date"`
	Payee           string   `bql:"payee" json:"payee"`
	Narration       string   `bql:"narration" json:"desc"`
	Account         string   `bql:"account" json:"account"`
	Tags            []string `bql:"tags" json:"tags"`
	Position        string   `bql:"position" json:"position"`
	Amount          string   `json:"amount"`
	Commodity       string   `json:"commodity"`
	CommoditySymbol string   `json:"commoditySymbol"`
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
	err := script.BQLQueryList(ledgerConfig, &queryParams, &transactions)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	// 格式化金额
	for i := 0; i < len(transactions); i++ {
		pos := strings.Split(transactions[i].Position, " ")
		if len(pos) == 2 {
			transactions[i].Amount = pos[0]
			transactions[i].Commodity = pos[1]
			transactions[i].CommoditySymbol = script.GetCommoditySymbol(pos[1])
		}
		transactions[i].Position = ""
	}
	OK(c, transactions)
}
