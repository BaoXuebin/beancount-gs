package service

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/beancount-gs/script"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"io"
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
	Account       string          `form:"account" binding:"required" json:"account"`
	Number        decimal.Decimal `form:"number" json:"number"`
	Currency      string          `form:"currency" json:"currency"`
	Price         decimal.Decimal `form:"price" json:"price"`
	PriceCurrency string          `form:"priceCurrency" json:"priceCurrency"`
}

func sum(entries []AddTransactionEntryForm, openingBalances string) decimal.Decimal {
	sumVal := decimal.NewFromInt(0)
	for _, entry := range entries {
		if entry.Account == openingBalances {
			return decimal.NewFromInt(0)
		}
		if entry.Price.IntPart() == 0 {
			sumVal = entry.Number.Add(sumVal)
		} else {
			sumVal = entry.Number.Mul(entry.Price).Add(sumVal)
		}
	}
	return sumVal
}

func AddBatchTransactions(c *gin.Context) {
	var addTransactionForms []AddTransactionForm
	if err := c.ShouldBindJSON(&addTransactionForms); err != nil {
		BadRequest(c, err.Error())
		return
	}
	result := make([]string, 0)
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	for _, form := range addTransactionForms {
		err := saveTransaction(nil, form, ledgerConfig)
		if err == nil {
			result = append(result, form.Date+form.Payee+form.Desc)
		} else {
			script.LogError(ledgerConfig.Mail, err.Error())
		}
	}
	OK(c, result)
}

func AddTransactions(c *gin.Context) {
	var addTransactionForm AddTransactionForm
	if err := c.ShouldBindJSON(&addTransactionForm); err != nil {
		BadRequest(c, err.Error())
		return
	}
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	err := saveTransaction(c, addTransactionForm, ledgerConfig)
	if err != nil {
		return
	}
	OK(c, nil)
}

func saveTransaction(c *gin.Context, addTransactionForm AddTransactionForm, ledgerConfig *script.Config) error {
	// 账户是否平衡
	sumVal := sum(addTransactionForm.Entries, ledgerConfig.OpeningBalances)
	val, _ := decimal.NewFromString("0.1")
	if sumVal.Abs().GreaterThan(val) {
		if c != nil {
			TransactionNotBalance(c)
		}
		return errors.New("transaction not balance")
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
			line += fmt.Sprintf("\r\n %s %s %s", entry.Account, entry.Number.Round(2).StringFixedBank(2), account.Currency)
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
				if c != nil {
					InternalError(c, err.Error())
				}
				return errors.New("internal error")
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
		if c != nil {
			InternalError(c, err.Error())
		}
		return errors.New("internal error")
	}
	monthStr := month.Format("2006-01")
	filePath := fmt.Sprintf("%s/month/%s.bean", ledgerConfig.DataPath, monthStr)

	// 文件不存在，则创建
	if !script.FileIfExist(filePath) {
		err = script.CreateFile(filePath)
		if err != nil {
			if c != nil {
				InternalError(c, err.Error())
			}
			return errors.New("internal error")
		}
		// include ./2021-11.bean
		err = script.AppendFileInNewLine(script.GetLedgerMonthsFilePath(ledgerConfig.DataPath), fmt.Sprintf("include \"./%s.bean\"", monthStr))
		if err != nil {
			if c != nil {
				InternalError(c, err.Error())
			}
			return errors.New("internal error")
		}
	}

	err = script.AppendFileInNewLine(filePath, line)
	if err != nil {
		if c != nil {
			InternalError(c, err.Error())
		}
		return errors.New("internal error")
	}
	return nil
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

type TransactionTemplate struct {
	Id           string                    `json:"id"`
	Date         string                    `form:"date" binding:"required" json:"date"`
	TemplateName string                    `form:"templateName" binding:"required" json:"templateName"`
	Payee        string                    `form:"payee" json:"payee"`
	Desc         string                    `form:"desc" binding:"required" json:"desc"`
	Entries      []AddTransactionEntryForm `form:"entries" json:"entries"`
}

func QueryTransactionTemplates(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	filePath := script.GetLedgerTransactionsTemplateFilePath(ledgerConfig.DataPath)
	templates, err := getLedgerTransactionTemplates(filePath)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	OK(c, templates)
}

func AddTransactionTemplate(c *gin.Context) {
	var transactionTemplate TransactionTemplate
	if err := c.ShouldBindJSON(&transactionTemplate); err != nil {
		BadRequest(c, err.Error())
		return
	}

	ledgerConfig := script.GetLedgerConfigFromContext(c)
	filePath := script.GetLedgerTransactionsTemplateFilePath(ledgerConfig.DataPath)
	templates, err := getLedgerTransactionTemplates(filePath)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	t := sha1.New()
	_, err = io.WriteString(t, time.Now().String())
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	transactionTemplate.Id = hex.EncodeToString(t.Sum(nil))
	templates = append(templates, transactionTemplate)

	err = writeLedgerTransactionTemplates(filePath, templates)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	OK(c, transactionTemplate)
}

func DeleteTransactionTemplate(c *gin.Context) {
	templateId := c.Query("id")
	if templateId == "" {
		BadRequest(c, "templateId is not blank")
		return
	}

	ledgerConfig := script.GetLedgerConfigFromContext(c)
	filePath := script.GetLedgerTransactionsTemplateFilePath(ledgerConfig.DataPath)

	oldTemplates, err := getLedgerTransactionTemplates(filePath)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	newTemplates := make([]TransactionTemplate, 0)
	for _, template := range oldTemplates {
		if template.Id != templateId {
			newTemplates = append(newTemplates, template)
		}
	}

	err = writeLedgerTransactionTemplates(filePath, newTemplates)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	OK(c, templateId)
}

func getLedgerTransactionTemplates(filePath string) ([]TransactionTemplate, error) {
	result := make([]TransactionTemplate, 0)
	if script.FileIfExist(filePath) {
		bytes, err := script.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(bytes, &result)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func writeLedgerTransactionTemplates(filePath string, templates []TransactionTemplate) error {
	if !script.FileIfExist(filePath) {
		err := script.CreateFile(filePath)
		if err != nil {
			return err
		}
	}

	bytes, err := json.Marshal(templates)
	if err != nil {
		return err
	}
	err = script.WriteFile(filePath, string(bytes))
	if err != nil {
		return err
	}
	return nil
}
