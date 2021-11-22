package service

import (
	"encoding/json"
	"github.com/beancount-gs/script"
	"github.com/gin-gonic/gin"
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

func QueryTransactions(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	queryParams := script.GetQueryParams(c)
	// 倒序查询
	queryParams.OrderBy = "date desc"
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

type transactionsPayee struct {
	Value string `bql:"distinct payee" json:"value"`
}

func QueryTransactionsPayee(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	payeeList := make([]transactionsPayee, 0)
	queryParams := script.QueryParams{Where: false, OrderBy: "date desc", Limit: 100}
	err := script.BQLQueryList(ledgerConfig, &queryParams, &payeeList)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	result := make([]string, 0)
	for _, payee := range payeeList {
		if payee.Value != "" {
			result = append(result, payee.Value)
		}
	}
	OK(c, result)
}

type transactionsTemplate struct {
	Id           string                       `json:"id"`
	Date         string                       `json:"date"`
	TemplateName string                       `json:"templateName"`
	Payee        string                       `json:"payee"`
	Desc         string                       `json:"desc"`
	Entries      []transactionsTemplateEntity `json:"entries"`
}

type transactionsTemplateEntity struct {
	Account   string `json:"account"`
	Commodity string `json:"commodity"`
	Amount    string `json:"amount"`
}

func QueryTransactionsTemplate(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	filePath := script.GetLedgerTransactionsTemplateFilePath(ledgerConfig.DataPath)
	if script.FileIfExist(filePath) {
		bytes, err := script.ReadFile(filePath)
		if err != nil {
			InternalError(c, err.Error())
			return
		}
		result := make([]transactionsTemplate, 0)
		err = json.Unmarshal(bytes, &result)
		if err != nil {
			InternalError(c, err.Error())
			return
		}
		OK(c, result)
	} else {
		OK(c, new([]string))
	}
}
