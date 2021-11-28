package service

import (
	"encoding/json"
	"fmt"
	"github.com/beancount-gs/script"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"strings"
	"time"
)

type Transaction struct {
	Id                 string   `bql:"id" json:"id"`
	Account            string   `bql:"account" json:"account"`
	Date               string   `bql:"date" json:"date"`
	Payee              string   `bql:"payee" json:"payee"`
	Narration          string   `bql:"narration" json:"desc"`
	Number             string   `bql:"number" json:"number"`
	Currency           string   `bql:"currency" json:"currency"`
	CostDate           string   `bql:"cost_date" json:"costDate"`
	CostPrice          string   `bql:"cost_number" json:"costPrice"` // 交易净值
	CostCurrency       string   `bql:"cost_currency" json:"costCurrency"`
	Price              string   `bql:"price" json:"price"`
	Tags               []string `bql:"tags" json:"tags"`
	CurrencySymbol     string   `json:"currencySymbol,omitempty"`
	CostCurrencySymbol string   `json:"costCurrencySymbol,omitempty"`
}

func QueryTransactions(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	queryParams := script.GetQueryParams(c)
	// 倒序查询
	queryParams.OrderBy = "date desc"
	transactions := make([]Transaction, 0)
	err := script.BQLQueryList(ledgerConfig, &queryParams, &transactions)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	// 格式化金额
	for i := 0; i < len(transactions); i++ {
		symbol := script.GetCommoditySymbol(transactions[i].Currency)
		transactions[i].CurrencySymbol = symbol
		transactions[i].CostCurrencySymbol = symbol
		if transactions[i].Price != "" {
			transactions[i].Price = strings.Fields(transactions[i].Price)[0]
		}
	}
	OK(c, transactions)
}

type AddTransactionForm struct {
	Date    string                    `form:"date" binding:"required"`
	Payee   string                    `form:"payee"`
	Desc    string                    `form:"desc" binding:"required"`
	Tags    []string                  `form:"tags"`
	Entries []AddTransactionEntryForm `form:"entries"`
}

type AddTransactionEntryForm struct {
	Account string          `form:"account" binding:"required"`
	Number  decimal.Decimal `form:"number"`
	//Currency      string          `form:"currency"`
	Price decimal.Decimal `form:"price"`
	//PriceCurrency string          `form:"priceCurrency"`
}

func sum(entries []AddTransactionEntryForm, openingBalances string) decimal.Decimal {
	sumVal := decimal.NewFromInt(0)
	for _, entry := range entries {
		if entry.Account == openingBalances {
			return sumVal
		}
		if entry.Price.IntPart() == 0 {
			sumVal = entry.Number.Add(sumVal)
		} else {
			sumVal = entry.Number.Mul(entry.Price).Add(sumVal)
		}
	}
	return sumVal
}

func AddTransactions(c *gin.Context) {
	var addTransactionForm AddTransactionForm
	if err := c.ShouldBindJSON(&addTransactionForm); err != nil {
		BadRequest(c, err.Error())
		return
	}
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	// 账户是否平衡
	sumVal := sum(addTransactionForm.Entries, ledgerConfig.OpeningBalances)
	val, _ := decimal.NewFromString("0.01")
	if sumVal.Abs().GreaterThan(val) {
		TransactionNotBalance(c)
		return
	}

	// 2021-09-29 * "支付宝" "黄金补仓X元" #Invest
	line := fmt.Sprintf("\r\n%s * \"%s\" \"%s\"", addTransactionForm.Date, addTransactionForm.Payee, addTransactionForm.Desc)
	if len(addTransactionForm.Tags) > 0 {
		for _, tag := range addTransactionForm.Tags {
			line += "#" + tag + " "
		}
	}

	var autoBalance bool
	for _, entry := range addTransactionForm.Entries {
		account := script.GetLedgerAccount(ledgerConfig.Id, entry.Account)
		if entry.Account == ledgerConfig.OpeningBalances {
			line += fmt.Sprintf("\r\n %s", entry.Account)
		} else {
			line += fmt.Sprintf("\r\n %s %s %s", entry.Account, entry.Number.Round(2).String(), account.Currency)
		}
		// 判断是否设计多币种的转换
		if account.Currency != ledgerConfig.OperatingCurrency && entry.Account != ledgerConfig.OpeningBalances {
			autoBalance = true
			// 根据 number 的正负来判断是买入还是卖出
			if entry.Number.GreaterThan(decimal.NewFromInt(0)) {
				// {351.729 CNY, 2021-09-29}
				line += fmt.Sprintf(" {%s %s, %s}", entry.Price, ledgerConfig.OperatingCurrency, addTransactionForm.Date)
			} else {
				// {} @ 359.019 CNY
				line += fmt.Sprintf(" {} @ %s %s", entry.Price, ledgerConfig.OperatingCurrency)
			}
			priceLine := fmt.Sprintf("%s price %s %s %s", addTransactionForm.Date, account.Currency, entry.Price, ledgerConfig.OperatingCurrency)
			err := script.AppendFileInNewLine(script.GetLedgerPriceFilePath(ledgerConfig.DataPath), priceLine)
			if err != nil {
				InternalError(c, err.Error())
				return
			}
		}
	}

	// 平衡小数点误差
	if autoBalance {
		line += "\r\n " + ledgerConfig.OpeningBalances
	}
	// 记账的日期
	month, err := time.Parse("2006-01-02", addTransactionForm.Date)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	monthStr := month.Format("2006-01")
	filePath := fmt.Sprintf("%s/month/%s.bean", ledgerConfig.DataPath, monthStr)

	// 文件不存在，则创建
	if !script.FileIfExist(filePath) {
		err = script.CreateFile(filePath)
		if err != nil {
			InternalError(c, err.Error())
			return
		}
		// include ./2021-11.bean
		err = script.AppendFileInNewLine(script.GetLedgerMonthsFilePath(ledgerConfig.DataPath), fmt.Sprintf("include \"./%s.bean\"", monthStr))
		if err != nil {
			InternalError(c, err.Error())
			return
		}
	}

	err = script.AppendFileInNewLine(filePath, line)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	OK(c, nil)
}

type transactionPayee struct {
	Value string `bql:"distinct payee" json:"value"`
}

func QueryTransactionPayees(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	payeeList := make([]transactionPayee, 0)
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

type transactionTemplate struct {
	Id           string                      `json:"id"`
	Date         string                      `json:"date"`
	TemplateName string                      `json:"templateName"`
	Payee        string                      `json:"payee"`
	Desc         string                      `json:"desc"`
	Entries      []transactionTemplateEntity `json:"entries"`
}

type transactionTemplateEntity struct {
	Account   string `json:"account"`
	Commodity string `json:"commodity"`
	Amount    string `json:"amount"`
}

func QueryTransactionTemplates(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	filePath := script.GetLedgerTransactionsTemplateFilePath(ledgerConfig.DataPath)
	if script.FileIfExist(filePath) {
		bytes, err := script.ReadFile(filePath)
		if err != nil {
			InternalError(c, err.Error())
			return
		}
		result := make([]transactionTemplate, 0)
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
