package service

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/beancount-gs/script"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// DateRange 查询账本中交易记录的时间范围
type DateRange struct {
	MinDate string `bql:"min(date)" json:"minDate"`
	MaxDate string `bql:"max(date)" json:"maxDate"`
}

// YearMonthOption 年月选项
type YearMonthOption struct {
	Year  int `json:"year"`
	Month int `json:"month"` // 0 表示全年
	Count int `json:"count"` // 该月份的交易数量
}

type Transaction struct {
	Id                 string   `bql:"distinct id" json:"id"`
	Account            string   `bql:"account" json:"account"`
	Date               string   `bql:"date" json:"date"`
	Payee              string   `bql:"payee" json:"payee"`
	Narration          string   `bql:"narration" json:"desc"`
	Number             string   `bql:"number" json:"number"`
	Balance            string   `bql:"balance" json:"balance"`
	Currency           string   `bql:"currency" json:"currency"`
	CostDate           string   `bql:"cost_date" json:"costDate"`
	CostPrice          string   `bql:"cost_number" json:"costPrice"`
	CostCurrency       string   `bql:"cost_currency" json:"costCurrency"`
	Price              string   `bql:"price" json:"price"`
	Tags               []string `bql:"tags" json:"tags"`
	CurrencySymbol     string   `json:"currencySymbol,omitempty"`
	CostCurrencySymbol string   `json:"costCurrencySymbol,omitempty"`
	IsAnotherCurrency  bool     `json:"isAnotherCurrency,omitempty"`
}

type TransactionForm struct {
	ID             string                 `form:"id" json:"id"`
	Date           string                 `form:"date" binding:"required" json:"date"`
	Payee          string                 `form:"payee" json:"payee,omitempty"`
	Desc           string                 `form:"desc" binding:"required" json:"desc"`
	Narration      string                 `form:"narration" json:"narration,omitempty"`
	Tags           []string               `form:"tags" json:"tags,omitempty"`
	DivideDateList []string               `form:"divideDateList" json:"divideDateList,omitempty"`
	Entries        []TransactionEntryForm `form:"entries" json:"entries"`
	RawText        string                 `json:"rawText,omitempty"`
}

type TransactionEntryForm struct {
	Account           string          `form:"account" binding:"required" json:"account"`
	Number            decimal.Decimal `form:"number" json:"number,omitempty"`
	Currency          string          `form:"currency" json:"currency"`
	Price             decimal.Decimal `form:"price" json:"price,omitempty"`
	PriceCurrency     string          `form:"priceCurrency" json:"priceCurrency,omitempty"`
	IsAnotherCurrency bool            `form:"isAnotherCurrency" json:"isAnotherCurrency,omitempty"`
}

type UpdateRawTextTransactionForm struct {
	ID      string `form:"id" binding:"required" json:"id"`
	RawText string `form:"rawText" json:"rawText,omitempty" binding:"required"`
}

type TransactionQuery struct {
	Year    int    `form:"year"`
	Month   int    `form:"month"`
	Account string `form:"account"`
	Tag     string `form:"tag"`
	Type    string `form:"type"`
	Limit   int    `form:"limit"`
	Offset  int    `form:"offset"`
}

type TransactionTemplate struct {
	Id           string                 `json:"id"`
	Date         string                 `form:"date" binding:"required" json:"date"`
	TemplateName string                 `form:"templateName" binding:"required" json:"templateName"`
	Payee        string                 `form:"payee" json:"payee"`
	Desc         string                 `form:"desc" binding:"required" json:"desc"`
	Entries      []TransactionEntryForm `form:"entries" json:"entries"`
}

// ==================== 工具函数 ====================

// getTransactionDateRange 获取交易记录的时间范围
func getTransactionDateRange(ledgerConfig *script.Config) (*DateRange, error) {
	var dateRange []DateRange
	queryParams := script.QueryParams{
		Where: false,
	}

	err := script.BQLQueryList(ledgerConfig, &queryParams, &dateRange)
	if err != nil {
		return nil, err
	}

	if len(dateRange) == 0 {
		currentDate := time.Now().Format("2006-01-02")
		return &DateRange{
			MinDate: currentDate,
			MaxDate: currentDate,
		}, nil
	}

	return &dateRange[0], nil
}

func buildQuery(params map[string]string) string {
	var buf strings.Builder
	for k, v := range params {
		if buf.Len() > 0 {
			buf.WriteByte('&')
		}
		buf.WriteString(url.QueryEscape(k))
		buf.WriteByte('=')
		buf.WriteString(url.QueryEscape(v))
	}
	return buf.String()
}

func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}

func safeString(s string, defaultValue string) string {
	if s == "" {
		return defaultValue
	}
	return s
}

func safeTags(tags []string) string {
	if len(tags) == 0 {
		return "[]"
	}
	return fmt.Sprintf("%v", tags)
}

func logTransactionMultiline(mail string, index int, tx *Transaction) {
	script.LogDebugDetailed(mail, "TransactionDetails", `
交易[%d]详细信息:
├── ID: %s
├── 账户: %s
├── 日期: %s
├── 收款方: %s
├── 描述: %s
├── 金额: %s
├── 货币: %s
├── 余额: %s
├── 成本日期: %s
├── 成本价格: %s
├── 成本货币: %s
├── 价格: %s
└── 标签: %v`,
		index,
		safeString(tx.Id, "N/A"),
		safeString(tx.Account, "N/A"),
		safeString(tx.Date, "N/A"),
		safeString(tx.Payee, "N/A"),
		truncateString(tx.Narration, 40),
		safeString(tx.Number, "N/A"),
		safeString(tx.Currency, "N/A"),
		safeString(tx.Balance, "N/A"),
		safeString(tx.CostDate, "N/A"),
		safeString(tx.CostPrice, "N/A"),
		safeString(tx.CostCurrency, "N/A"),
		safeString(tx.Price, "N/A"),
		safeTags(tx.Tags))
}

func filterEmptyStrings(arr []string) []string {
	var result []string
	for _, str := range arr {
		if script.CleanString(str) != "" {
			result = append(result, str)
		}
	}
	return result
}

func sum(entries []TransactionEntryForm, openingBalances string) decimal.Decimal {
	sumVal := decimal.NewFromInt(0)
	for _, entry := range entries {
		if entry.Account == openingBalances {
			return decimal.NewFromInt(0)
		}
		pVal, _ := entry.Price.Float64()
		if pVal == 0 {
			sumVal = entry.Number.Add(sumVal)
		} else {
			sumVal = entry.Number.Mul(entry.Price).Add(sumVal)
		}
	}
	return sumVal
}

// ==================== 交易查询相关 ====================

func QueryTransactionDetailById(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	queryParams := script.GetQueryParams(c)

	if queryParams.ID == "" {
		BadRequest(c, "参数 'id' 不能为空")
		return
	}

	transactions := make([]Transaction, 0)
	err := script.BQLQueryList(ledgerConfig, &queryParams, &transactions)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	if len(transactions) == 0 {
		BadRequest(c, "未找到交易记录")
		return
	}

	transactionForm := TransactionForm{
		Entries: make([]TransactionEntryForm, 0),
	}

	for _, transaction := range transactions {
		if transactionForm.ID == "" {
			transactionForm.ID = transaction.Id
			transactionForm.Date = transaction.Date
			transactionForm.Payee = transaction.Payee
			transactionForm.Desc = transaction.Narration
			transactionForm.Narration = transaction.Narration
		}

		entry := TransactionEntryForm{Account: transaction.Account}

		if transaction.Number != "" && transaction.Number != "0" {
			entry.Number = decimal.RequireFromString(transaction.Number)
			entry.Currency = transaction.Currency
			entry.IsAnotherCurrency = transaction.IsAnotherCurrency
		}

		if transaction.CostPrice != "" && transaction.CostPrice != "0" {
			entry.Price = decimal.RequireFromString(transaction.CostPrice)
			entry.PriceCurrency = transaction.CostCurrency
		}

		transactionForm.Entries = append(transactionForm.Entries, entry)
	}

	OK(c, transactionForm)
}

func QueryTransactionRawTextById(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	queryParams := script.GetQueryParams(c)

	if queryParams.ID == "" {
		BadRequest(c, "参数 'id' 不能为空")
		return
	}

	result, err := script.BQLPrint(ledgerConfig, queryParams.ID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	OK(c, result)
}

func QueryTransactions(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)

	// 参数预处理
	queryValues := c.Request.URL.Query()
	params := make(map[string]string)
	for k, v := range queryValues {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}

	// 处理undefined参数
	now := time.Now()
	if params["year"] == "undefined" {
		params["year"] = strconv.Itoa(now.Year())
	}
	if params["month"] == "undefined" {
		params["month"] = strconv.Itoa(int(now.Month()))
	}

	c.Request.URL.RawQuery = buildQuery(params)

	// 绑定查询参数
	var transactionQuery TransactionQuery
	if err := c.ShouldBindQuery(&transactionQuery); err != nil {
		BadRequest(c, "无效的查询参数")
		return
	}

	// 获取账本时间范围
	dateRange, err := getTransactionDateRange(ledgerConfig)
	if err != nil {
		InternalError(c, "获取账本时间范围失败")
		return
	}

	// 解析最小日期作为默认起始点
	minDate, err := time.Parse("2006-01-02", dateRange.MinDate)
	if err != nil {
		InternalError(c, "解析账本最小日期失败")
		return
	}

	// 设置默认年月
	if transactionQuery.Year <= 0 {
		transactionQuery.Year = minDate.Year()
	}
	if transactionQuery.Month <= 0 || transactionQuery.Month > 12 {
		transactionQuery.Month = int(minDate.Month())
	}

	// 构建查询参数
	queryParams := script.QueryParams{
		FromYear:  minDate.Year(),
		FromMonth: int(minDate.Month()),
		Year:      transactionQuery.Year,
		Month:     transactionQuery.Month,
		Account:   transactionQuery.Account,
		Tag:       transactionQuery.Tag,
		Where:     true,
		OrderBy:   "date desc",
		Limit:     transactionQuery.Limit,
		From:      true, // 启用起始日期查询
		DateRange: true, // 启用日期范围查询
	}

	// 设置日期范围
	if transactionQuery.Year <= 0 {
		queryParams.Year = now.Year()
		queryParams.Month = int(now.Month())
	}

	// 设置科目匹配模式
	if transactionQuery.Account != "" {
		queryParams.StrictAccountMatch = true
	}

	// 记录完整的查询参数
	script.LogDebugDetailed(ledgerConfig.Mail, "QueryTransactions-FinalParams",
		"完整查询参数: %+v (账本时间范围: %s 至 %s)",
		queryParams, dateRange.MinDate, dateRange.MaxDate)

	// 设置合理的limit
	if queryParams.Limit <= 0 || queryParams.Limit > 1000 {
		queryParams.Limit = 100
	}

	// 执行查询
	transactions := make([]Transaction, 0)
	err = script.BQLQueryList(ledgerConfig, &queryParams, &transactions)
	if err != nil {
		InternalError(c, "查询交易记录失败")
		return
	}

	// 处理交易记录
	currencyMap := script.GetLedgerCurrencyMap(ledgerConfig.Id)
	for i := range transactions {
		if i < 3 {
			logTransactionMultiline(ledgerConfig.Mail, i, &transactions[i])
		}

		_, ok := currencyMap[transactions[i].Currency]
		if ok {
			transactions[i].IsAnotherCurrency = transactions[i].Currency != ledgerConfig.OperatingCurrency
		}

		symbol := script.GetCommoditySymbol(ledgerConfig.Id, transactions[i].Currency)
		transactions[i].CurrencySymbol = symbol
		transactions[i].CostCurrencySymbol = symbol

		if transactions[i].Price != "" {
			transactions[i].Price = strings.Fields(transactions[i].Price)[0]
		}
		if transactions[i].Balance != "" {
			transactions[i].Balance = strings.Fields(transactions[i].Balance)[0]
		}
	}

	OK(c, transactions)
}

// ==================== 交易操作相关 ====================

func AddBatchTransactions(c *gin.Context) {
	var addTransactionForms []TransactionForm
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
		}
	}
	OK(c, result)
}

func AddTransactions(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	var addTransactionForm TransactionForm

	if err := c.ShouldBindJSON(&addTransactionForm); err != nil {
		BadRequest(c, err.Error())
		return
	}

	var err error
	divideCount := len(addTransactionForm.DivideDateList)

	if divideCount <= 0 {
		err = saveTransaction(c, addTransactionForm, ledgerConfig)
	} else {
		// 分期处理
		for idx, entry := range addTransactionForm.Entries {
			addTransactionForm.Entries[idx].Number = entry.Number.Div(decimal.NewFromInt(int64(divideCount))).Round(3)
		}
		for _, date := range addTransactionForm.DivideDateList {
			addTransactionForm.Date = date
			err = saveTransaction(c, addTransactionForm, ledgerConfig)
			if err != nil {
				break
			}
		}
	}

	if err != nil {
		InternalError(c, err.Error())
		return
	}

	OK(c, nil)
}

func saveTransaction(c *gin.Context, form TransactionForm, ledgerConfig *script.Config) error {
	// 余额检查
	sumVal := sum(form.Entries, ledgerConfig.OpeningBalances)
	val, _ := decimal.NewFromString("0.1")
	if sumVal.Abs().GreaterThan(val) {
		if c != nil {
			TransactionNotBalance(c)
		}
		return errors.New("交易不平衡")
	}

	// 构建交易文本
	line := fmt.Sprintf("\r\n%s * \"%s\" \"%s\"", form.Date, form.Payee, form.Desc)

	// 添加标签
	for _, tag := range form.Tags {
		line += "#" + tag + " "
	}

	currencyMap := script.GetLedgerCurrencyMap(ledgerConfig.Id)
	zero := decimal.NewFromInt(0)

	for _, entry := range form.Entries {
		if entry.Account == ledgerConfig.OpeningBalances {
			line += fmt.Sprintf("\r\n %s", entry.Account)
		} else {
			line += fmt.Sprintf("\r\n %s %s %s", entry.Account, entry.Number.Round(2).StringFixedBank(2), entry.Currency)
		}

		// 多币种处理
		if entry.Currency != ledgerConfig.OperatingCurrency && entry.Account != ledgerConfig.OpeningBalances {
			if entry.Price.LessThanOrEqual(zero) {
				continue
			}

			currency, isCurrency := currencyMap[entry.Currency]
			currencyPrice := entry.Price
			if currencyPrice.Equal(zero) {
				currencyPrice, _ = decimal.NewFromString(currency.Price)
			}

			if !isCurrency {
				if entry.Number.GreaterThan(zero) {
					line += fmt.Sprintf(" {%s %s, %s}", entry.Price, ledgerConfig.OperatingCurrency, form.Date)
				} else {
					line += fmt.Sprintf(" {} @ %s %s", entry.Price, ledgerConfig.OperatingCurrency)
				}
			} else {
				line += fmt.Sprintf(" {%s %s}", currencyPrice, ledgerConfig.OperatingCurrency)
			}

			// 更新价格文件
			priceLine := fmt.Sprintf("%s price %s %s %s", form.Date, entry.Currency, entry.Price, ledgerConfig.OperatingCurrency)
			if err := script.AppendFileInNewLine(script.GetLedgerPriceFilePath(ledgerConfig.DataPath), priceLine); err != nil {
				return errors.New("internal error")
			}

			if isCurrency {
				if err := script.LoadLedgerCurrencyMap(ledgerConfig); err != nil {
					return errors.New("internal error")
				}
			}
		}
	}

	// 确定文件路径
	month, err := time.Parse("2006-01-02", form.Date)
	if err != nil {
		return errors.New("internal error")
	}

	monthStr := month.Format("2006-01")
	if err := CreateMonthBeanFileIfNotExist(ledgerConfig.DataPath, monthStr); err != nil {
		return err
	}

	beanFilePath := script.GetLedgerMonthFilePath(ledgerConfig.DataPath, monthStr)

	if form.ID != "" {
		// 更新交易
		return updateTransaction(ledgerConfig, form, beanFilePath, line)
	} else {
		// 新增交易
		return script.AppendFileInNewLine(beanFilePath, line)
	}
}

func updateTransaction(ledgerConfig *script.Config, form TransactionForm, beanFilePath, newContent string) error {
	result, err := script.BQLPrint(ledgerConfig, form.ID)
	if err != nil {
		return err
	}

	oldLines := filterEmptyStrings(strings.Split(result, "\n"))
	startLine, endLine, err := script.FindConsecutiveMultilineTextInFile(beanFilePath, oldLines)
	if err != nil {
		return err
	}

	lines, err := script.RemoveLines(beanFilePath, startLine, endLine)
	if err != nil {
		return err
	}

	newLines := filterEmptyStrings(strings.Split(newContent, "\n"))
	newLines = append(newLines, "")
	lines, err = script.InsertLines(lines, startLine, newLines)
	if err != nil {
		return err
	}

	return script.WriteToFile(beanFilePath, lines)
}

// ==================== 其他功能 ====================

func UpdateTransactionRawTextById(c *gin.Context) {
	var form UpdateRawTextTransactionForm
	if err := c.ShouldBindJSON(&form); err != nil {
		BadRequest(c, err.Error())
		return
	}

	ledgerConfig := script.GetLedgerConfigFromContext(c)
	beanFilePath, err := getBeanFilePathByTransactionId(form.ID, ledgerConfig)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	result, err := script.BQLPrint(ledgerConfig, form.ID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	oldLines := filterEmptyStrings(strings.Split(result, "\n"))
	startLine, endLine, err := script.FindConsecutiveMultilineTextInFile(beanFilePath, oldLines)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	lines, err := script.RemoveLines(beanFilePath, startLine, endLine)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	newLines := filterEmptyStrings(strings.Split(form.RawText, "\n"))
	if len(newLines) > 0 {
		lines, err = script.InsertLines(lines, startLine, newLines)
		if err != nil {
			InternalError(c, err.Error())
			return
		}
	}

	if err := script.WriteToFile(beanFilePath, lines); err != nil {
		InternalError(c, err.Error())
		return
	}

	OK(c, true)
}

func DeleteTransactionById(c *gin.Context) {
	queryParams := script.GetQueryParams(c)
	if queryParams.ID == "" {
		BadRequest(c, "Param 'id' must not be blank.")
		return
	}

	ledgerConfig := script.GetLedgerConfigFromContext(c)
	beanFilePath, err := getBeanFilePathByTransactionId(queryParams.ID, ledgerConfig)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	result, err := script.BQLPrint(ledgerConfig, queryParams.ID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	oldLines := filterEmptyStrings(strings.Split(result, "\n"))
	startLine, endLine, err := script.FindConsecutiveMultilineTextInFile(beanFilePath, oldLines)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	lines, err := script.RemoveLines(beanFilePath, startLine, endLine)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	if err := script.WriteToFile(beanFilePath, lines); err != nil {
		InternalError(c, err.Error())
		return
	}

	OK(c, true)
}

func getBeanFilePathByTransactionId(transactionId string, ledgerConfig *script.Config) (string, error) {
	queryParams := script.QueryParams{ID: transactionId, Where: true}
	transactions := make([]Transaction, 0)
	err := script.BQLQueryList(ledgerConfig, &queryParams, &transactions)
	if err != nil {
		return "", err
	}
	if len(transactions) == 0 {
		return "", errors.New("no transaction found")
	}

	month, err := script.GetMonth(transactions[0].Date)
	if err != nil {
		return "", err
	}

	return script.GetLedgerMonthFilePath(ledgerConfig.DataPath, month), nil
}

// ==================== 模板相关 ====================

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
	var template TransactionTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
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

	// 生成唯一ID
	t := sha1.New()
	io.WriteString(t, time.Now().String())
	template.Id = hex.EncodeToString(t.Sum(nil))
	templates = append(templates, template)

	if err := writeLedgerTransactionTemplates(filePath, templates); err != nil {
		InternalError(c, err.Error())
		return
	}

	OK(c, template)
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

	if err := writeLedgerTransactionTemplates(filePath, newTemplates); err != nil {
		InternalError(c, err.Error())
		return
	}

	OK(c, templateId)
}

func getLedgerTransactionTemplates(filePath string) ([]TransactionTemplate, error) {
	result := make([]TransactionTemplate, 0)
	if !script.FileIfExist(filePath) {
		return result, nil
	}

	bytes, err := script.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(bytes, &result)
	return result, err
}

func writeLedgerTransactionTemplates(filePath string, templates []TransactionTemplate) error {
	if !script.FileIfExist(filePath) {
		if err := script.CreateFile(filePath); err != nil {
			return err
		}
	}

	bytes, err := json.Marshal(templates)
	if err != nil {
		return err
	}

	return script.WriteFile(filePath, string(bytes))
}

// ==================== 辅助查询 ====================

type transactionPayee struct {
	Value string `bql:"distinct payee" json:"value"`
}

func QueryTransactionPayees(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	payeeList := make([]transactionPayee, 0)

	queryParams := script.QueryParams{
		Where:   false,
		OrderBy: "date desc",
		Limit:   100,
	}

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

func QueryAvailableYearMonths(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)

	var yearMonths []struct {
		Year  int `bql:"year" json:"year"`
		Month int `bql:"month" json:"month"`
		Count int `bql:"count" json:"count"`
	}

	queryParams := script.QueryParams{
		Where:   false,
		GroupBy: "year, month",
		OrderBy: "year desc, month desc",
	}

	err := script.BQLQueryList(ledgerConfig, &queryParams, &yearMonths)
	if err != nil {
		InternalError(c, "查询可用年月失败")
		return
	}

	result := make([]YearMonthOption, 0)
	result = append(result, YearMonthOption{Year: 0, Month: 0, Count: -1})

	for _, ym := range yearMonths {
		result = append(result, YearMonthOption{
			Year:  ym.Year,
			Month: ym.Month,
			Count: ym.Count,
		})
	}

	OK(c, result)
}
