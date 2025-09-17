package service

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/beancount-gs/script"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type Transaction struct {
	Id                 string   `bql:"id" json:"id"`
	Account            string   `bql:"account" json:"account"`
	Date               string   `bql:"date" json:"date"`
	Payee              string   `bql:"payee" json:"payee"`
	Narration          string   `bql:"narration" json:"desc"`
	Number             string   `bql:"number" json:"number"`
	Balance            string   `bql:"balance" json:"balance"`
	Currency           string   `bql:"currency" json:"currency"`
	CostDate           string   `bql:"cost_date" json:"costDate"`
	CostPrice          string   `bql:"cost_number" json:"costPrice"` // 交易净值
	CostCurrency       string   `bql:"cost_currency" json:"costCurrency"`
	Price              string   `bql:"price" json:"price"`
	Tags               []string `bql:"tags" json:"tags"`
	CurrencySymbol     string   `json:"currencySymbol,omitempty"`
	CostCurrencySymbol string   `json:"costCurrencySymbol,omitempty"`
	IsAnotherCurrency  bool     `json:"isAnotherCurrency,omitempty"`
}

type RawTransaction struct {
	RawText     string `json:"text"`
	StartLineNo int    `json:"startLineNo"`
	EndLineNo   int    `json:"endLineNo"`
	FilePath    string `json:"filePath,omitempty"`
}

type TransactionSort []Transaction

// 调试模式控制
var debugMode = script.IsDebugMode() // 可以通过配置控制

// 调试日志函数
func debugLog(format string, args ...interface{}) {
	if debugMode {
		logMsg := fmt.Sprintf("DEBUG: "+format, args...)
		log.Println(logMsg)
	}
}

// 详细的调试信息函数
func debugLogDetailed(context, format string, args ...interface{}) {
	if debugMode {
		logMsg := fmt.Sprintf("DEBUG [%s]: "+format, append([]interface{}{context}, args...)...)
		log.Println(logMsg)
	}
}

func (s TransactionSort) Len() int {
	return len(s)
}
func (s TransactionSort) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s TransactionSort) Less(i, j int) bool {
	a, _ := strconv.Atoi(s[i].Number)
	b, _ := strconv.Atoi(s[j].Number)
	return a <= b
}

func QueryTransactionDetailById(c *gin.Context) {
	debugLogDetailed("QueryTransactionDetailById", "函数开始执行")

	queryParams := script.GetQueryParams(c)
	debugLogDetailed("QueryTransactionDetailById", "获取查询参数: ID=%s", queryParams.ID)

	if queryParams.ID == "" {
		debugLogDetailed("QueryTransactionDetailById", "参数验证失败: ID不能为空")
		BadRequest(c, "参数 'id' 不能为空")
		return
	}

	ledgerConfig := script.GetLedgerConfigFromContext(c)
	debugLogDetailed("QueryTransactionDetailById", "获取账本配置: ID=%s", ledgerConfig.Id)

	transactions := make([]Transaction, 0)
	err := script.BQLQueryList(ledgerConfig, &queryParams, &transactions)
	if err != nil {
		debugLogDetailed("QueryTransactionDetailById", "BQL查询失败: %v", err)
		BadRequest(c, err.Error())
		return
	}

	debugLogDetailed("QueryTransactionDetailById", "查询成功，返回 %d 条交易记录", len(transactions))

	if len(transactions) == 0 {
		debugLogDetailed("QueryTransactionDetailById", "未找到交易记录")
		BadRequest(c, "未找到交易记录")
		return
	}

	transactionForm := TransactionForm{}
	transactionForm.Entries = make([]TransactionEntryForm, 0)

	for _, transaction := range transactions {
		if transactionForm.ID == "" {
			transactionForm.ID = transaction.Id
			transactionForm.Date = transaction.Date
			transactionForm.Payee = transaction.Payee
			transactionForm.Desc = transaction.Narration
			transactionForm.Narration = transaction.Narration
			debugLogDetailed("QueryTransactionDetailById", "设置交易基本信息: ID=%s, Date=%s", transaction.Id, transaction.Date)
		}

		transactionEntryForm := TransactionEntryForm{
			Account: transaction.Account,
		}

		if transaction.Number != "" && transaction.Number != "0" {
			transactionEntryForm.Number = decimal.RequireFromString(transaction.Number)
			transactionEntryForm.Currency = transaction.Currency
			transactionEntryForm.IsAnotherCurrency = transaction.IsAnotherCurrency
			debugLogDetailed("QueryTransactionDetailById", "设置交易条目: Account=%s, Number=%s", transaction.Account, transaction.Number)
		}

		if transaction.CostPrice != "" && transaction.CostPrice != "0" {
			transactionEntryForm.Price = decimal.RequireFromString(transaction.CostPrice)
			transactionEntryForm.PriceCurrency = transaction.CostCurrency
			debugLogDetailed("QueryTransactionDetailById", "设置交易价格信息: Price=%s, Currency=%s", transaction.CostPrice, transaction.CostCurrency)
		}

		transactionForm.Entries = append(transactionForm.Entries, transactionEntryForm)
	}

	debugLogDetailed("QueryTransactionDetailById", "交易表单构建完成，共 %d 个条目", len(transactionForm.Entries))
	OK(c, transactionForm)
	debugLogDetailed("QueryTransactionDetailById", "函数执行完成")
}

func QueryTransactionRawTextById(c *gin.Context) {
	debugLogDetailed("QueryTransactionRawTextById", "函数开始执行")

	queryParams := script.GetQueryParams(c)
	debugLogDetailed("QueryTransactionRawTextById", "获取查询参数: ID=%s", queryParams.ID)

	if queryParams.ID == "" {
		debugLogDetailed("QueryTransactionRawTextById", "参数验证失败: ID不能为空")
		BadRequest(c, "参数 'id' 不能为空")
		return
	}

	ledgerConfig := script.GetLedgerConfigFromContext(c)
	debugLogDetailed("QueryTransactionRawTextById", "获取账本配置: ID=%s", ledgerConfig.Id)

	result, err := script.BQLPrint(ledgerConfig, queryParams.ID)
	if err != nil {
		debugLogDetailed("QueryTransactionRawTextById", "BQL打印失败: %v", err)
		InternalError(c, err.Error())
		return
	}

	debugLogDetailed("QueryTransactionRawTextById", "获取原始文本成功，长度: %d 字符", len(result))
	OK(c, result)
	debugLogDetailed("QueryTransactionRawTextById", "函数执行完成")
}

func QueryTransactions(c *gin.Context) {
	debugLogDetailed("QueryTransactions", "函数开始执行")

	ledgerConfig := script.GetLedgerConfigFromContext(c)
	debugLogDetailed("QueryTransactions", "获取账本配置 - ID: %s, 运营货币: %s", ledgerConfig.Id, ledgerConfig.OperatingCurrency)

	queryParams := script.GetQueryParams(c)
	debugLogDetailed("QueryTransactions", "初始查询参数: %+v", queryParams)

	// 倒序查询
	queryParams.OrderBy = "date desc"
	debugLogDetailed("QueryTransactions", "设置排序方式: %s", queryParams.OrderBy)

	transactions := make([]Transaction, 0)
	debugLogDetailed("QueryTransactions", "初始化空交易切片")

	err := script.BQLQueryList(ledgerConfig, &queryParams, &transactions)
	if err != nil {
		debugLogDetailed("QueryTransactions", "BQL查询列表失败: %v", err)
		InternalError(c, err.Error())
		return
	}
	debugLogDetailed("QueryTransactions", "BQL查询成功，返回 %d 条交易记录", len(transactions))

	if len(transactions) > 0 {
		debugLogDetailed("QueryTransactions", "第一条交易样例 - 账户: %s, 金额: %s",
			transactions[0].Account, transactions[0].Number)
	}

	currencyMap := script.GetLedgerCurrencyMap(ledgerConfig.Id)
	debugLogDetailed("QueryTransactions", "获取货币映射表，包含 %d 个条目", len(currencyMap))

	// 格式化金额
	debugLogDetailed("QueryTransactions", "开始格式化 %d 条交易记录", len(transactions))

	for i := 0; i < len(transactions); i++ {
		if i < 3 { // 只记录前3个交易的详细调试信息，避免日志过多
			debugLogDetailed("QueryTransactions", "处理交易 %d - 货币: %s, 余额: %s",
				i, transactions[i].Currency, transactions[i].Balance)
		}

		_, ok := currencyMap[transactions[i].Currency]
		if ok {
			transactions[i].IsAnotherCurrency = transactions[i].Currency != ledgerConfig.OperatingCurrency
			if i < 3 {
				debugLogDetailed("QueryTransactions", "交易 %d - 是否为其他货币: %t", i, transactions[i].IsAnotherCurrency)
			}
		} else {
			if i < 3 {
				debugLogDetailed("QueryTransactions", "交易 %d - 货币 %s 未在货币映射表中找到", i, transactions[i].Currency)
			}
		}

		symbol := script.GetCommoditySymbol(ledgerConfig.Id, transactions[i].Currency)
		transactions[i].CurrencySymbol = symbol
		transactions[i].CostCurrencySymbol = symbol
		if i < 3 {
			debugLogDetailed("QueryTransactions", "交易 %d - 设置货币符号: %s", i, symbol)
		}

		if transactions[i].Price != "" {
			oldPrice := transactions[i].Price
			transactions[i].Price = strings.Fields(transactions[i].Price)[0]
			if i < 3 {
				debugLogDetailed("QueryTransactions", "交易 %d - 价格格式化: %s -> %s", i, oldPrice, transactions[i].Price)
			}
		}

		if transactions[i].Balance != "" {
			oldBalance := transactions[i].Balance
			transactions[i].Balance = strings.Fields(transactions[i].Balance)[0]
			if i < 3 {
				debugLogDetailed("QueryTransactions", "交易 %d - 余额格式化: %s -> %s", i, oldBalance, transactions[i].Balance)
			}
		}

		// 每处理100个交易记录一次进度
		if (i+1)%100 == 0 {
			debugLogDetailed("QueryTransactions", "已处理 %d 条交易记录", i+1)
		}
	}

	debugLogDetailed("QueryTransactions", "完成所有 %d 条交易记录的处理", len(transactions))

	if len(transactions) > 0 {
		debugLogDetailed("QueryTransactions", "最终交易样例 - 格式化余额: %s%s",
			transactions[0].CurrencySymbol, transactions[0].Balance)
	}

	OK(c, transactions)
	debugLogDetailed("QueryTransactions", "函数执行完成")
}

// ... 其余代码保持不变，但可以在关键函数中添加类似的调试日志 ...

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

type UpdateRawTextTransactionForm struct {
	ID      string `form:"id" binding:"required" json:"id"`
	RawText string `form:"rawText" json:"rawText,omitempty" binding:"required"`
}

type TransactionEntryForm struct {
	Account           string          `form:"account" binding:"required" json:"account"`
	Number            decimal.Decimal `form:"number" json:"number,omitempty"`
	Currency          string          `form:"currency" json:"currency"`
	Price             decimal.Decimal `form:"price" json:"price,omitempty"`
	PriceCurrency     string          `form:"priceCurrency" json:"priceCurrency,omitempty"`
	IsAnotherCurrency bool            `form:"isAnotherCurrency" json:"isAnotherCurrency,omitempty"`
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
		} else {
			script.LogError(ledgerConfig.Mail, err.Error())
		}
	}
	OK(c, result)
}

func AddTransactions(c *gin.Context) {
	debugLogDetailed("AddTransactions", "函数开始执行")

	var addTransactionForm TransactionForm
	if err := c.ShouldBindJSON(&addTransactionForm); err != nil {
		debugLogDetailed("AddTransactions", "JSON绑定失败: %v", err)
		BadRequest(c, err.Error())
		return
	}
	debugLogDetailed("AddTransactions", "成功解析交易表单: ID=%s, Date=%s", addTransactionForm.ID, addTransactionForm.Date)

	ledgerConfig := script.GetLedgerConfigFromContext(c)

	// 判断是否分期
	var err error
	var divideCount = len(addTransactionForm.DivideDateList)
	debugLogDetailed("AddTransactions", "分期数量: %d", divideCount)

	if divideCount <= 0 {
		debugLogDetailed("AddTransactions", "执行单次交易保存")
		err = saveTransaction(c, addTransactionForm, ledgerConfig)
	} else {
		debugLogDetailed("AddTransactions", "执行分期交易保存")
		for idx, entry := range addTransactionForm.Entries {
			// 保留 3 位小数
			addTransactionForm.Entries[idx].Number = entry.Number.Div(decimal.NewFromInt(int64(divideCount))).Round(3)
			debugLogDetailed("AddTransactions", "分期计算: 条目 %d, 原金额: %s, 分期后: %s",
				idx, entry.Number.String(), addTransactionForm.Entries[idx].Number.String())
		}
		for _, date := range addTransactionForm.DivideDateList {
			addTransactionForm.Date = date
			debugLogDetailed("AddTransactions", "保存分期交易: 日期=%s", date)
			err = saveTransaction(c, addTransactionForm, ledgerConfig)
			if err != nil {
				debugLogDetailed("AddTransactions", "分期交易保存失败: %v", err)
				break
			}
		}
	}

	if err != nil {
		debugLogDetailed("AddTransactions", "交易保存失败: %v", err)
		script.LogError(ledgerConfig.Mail, err.Error())
		return
	}

	debugLogDetailed("AddTransactions", "交易保存成功")
	OK(c, nil)
	debugLogDetailed("AddTransactions", "函数执行完成")
}

// 在 saveTransaction 函数中也添加类似的调试日志
func saveTransaction(c *gin.Context, addTransactionForm TransactionForm, ledgerConfig *script.Config) error {
	debugLogDetailed("saveTransaction", "开始保存交易: ID=%s, Date=%s", addTransactionForm.ID, addTransactionForm.Date)

	// 账户是否平衡
	sumVal := sum(addTransactionForm.Entries, ledgerConfig.OpeningBalances)
	val, _ := decimal.NewFromString("0.1")
	debugLogDetailed("saveTransaction", "交易余额检查: 计算总和=%s, 阈值=%s", sumVal.String(), val.String())

	if sumVal.Abs().GreaterThan(val) {
		debugLogDetailed("saveTransaction", "交易不平衡: 差异过大")
		if c != nil {
			TransactionNotBalance(c)
		}
		return errors.New("交易不平衡")
	}

	// 构建交易文本
	line := fmt.Sprintf("\r\n%s * \"%s\" \"%s\"", addTransactionForm.Date, addTransactionForm.Payee, addTransactionForm.Desc)
	debugLogDetailed("saveTransaction", "构建交易头: %s", line)

	if len(addTransactionForm.Tags) > 0 {
		for _, tag := range addTransactionForm.Tags {
			line += "#" + tag + " "
		}
		debugLogDetailed("saveTransaction", "添加标签: %v", addTransactionForm.Tags)
	}

	currencyMap := script.GetLedgerCurrencyMap(ledgerConfig.Id)

	var autoBalance bool
	for _, entry := range addTransactionForm.Entries {
		if entry.Account == ledgerConfig.OpeningBalances {
			autoBalance = false
			line += fmt.Sprintf("\r\n %s", entry.Account)
		} else {
			line += fmt.Sprintf("\r\n %s %s %s", entry.Account, entry.Number.Round(2).StringFixedBank(2), entry.Currency)
		}
		zero := decimal.NewFromInt(0)
		// 判断是否涉及多币种的转换
		if entry.Currency != ledgerConfig.OperatingCurrency && entry.Account != ledgerConfig.OpeningBalances {
			// 汇率值小于等于0，则不进行汇率转换
			if entry.Price.LessThanOrEqual(zero) {
				continue
			}

			currency, isCurrency := currencyMap[entry.Currency]
			currencyPrice := entry.Price
			if currencyPrice.Equal(zero) {
				currencyPrice, _ = decimal.NewFromString(currency.Price)
			}
			// 货币跳过汇率转换
			if !isCurrency {
				// 根据 number 的正负来判断是买入还是卖出
				if entry.Number.GreaterThan(zero) {
					// {351.729 CNY, 2021-09-29}
					line += fmt.Sprintf(" {%s %s, %s}", entry.Price, ledgerConfig.OperatingCurrency, addTransactionForm.Date)
				} else {
					// {} @ 359.019 CNY
					line += fmt.Sprintf(" {} @ %s %s", entry.Price, ledgerConfig.OperatingCurrency)
				}
			} else {
				// 外币种格式：Assets:Fixed:三顿半咖啡 -1.00 SATURN_BIRD {5.61 CNY}
				// fix issue #66 https://github.com/BaoXuebin/beancount-gs/issues/66
				line += fmt.Sprintf(" {%s %s}", currencyPrice, ledgerConfig.OperatingCurrency)
			}

			priceLine := fmt.Sprintf("%s price %s %s %s", addTransactionForm.Date, entry.Currency, entry.Price, ledgerConfig.OperatingCurrency)
			err := script.AppendFileInNewLine(script.GetLedgerPriceFilePath(ledgerConfig.DataPath), priceLine)
			if err != nil {
				if c != nil {
					InternalError(c, err.Error())
				}
				return errors.New("internal error")
			}
			// 刷新币种汇率
			if isCurrency {
				err = script.LoadLedgerCurrencyMap(ledgerConfig)
				if err != nil {
					InternalError(c, err.Error())
					return errors.New("internal error")
				}
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

	// 交易的月份信息
	monthStr := month.Format("2006-01")
	err = CreateMonthBeanFileIfNotExist(ledgerConfig.DataPath, monthStr)
	if err != nil {
		if c != nil {
			InternalError(c, err.Error())
		}
		return err
	}

	beanFilePath := script.GetLedgerMonthFilePath(ledgerConfig.DataPath, monthStr)
	if addTransactionForm.ID != "" { // 更新交易
		result, e := script.BQLPrint(ledgerConfig, addTransactionForm.ID)
		if e != nil {
			InternalError(c, e.Error())
			return errors.New(e.Error())
		}
		// 使用 \r\t 分割多行文本片段，并清理每一行的空白
		oldLines := filterEmptyStrings(strings.Split(result, "\n"))
		startLine, endLine, e := script.FindConsecutiveMultilineTextInFile(beanFilePath, oldLines)
		if e != nil {
			InternalError(c, e.Error())
			return errors.New(e.Error())
		}
		lines, e := script.RemoveLines(beanFilePath, startLine, endLine)
		if e != nil {
			InternalError(c, e.Error())
			return errors.New(e.Error())
		}
		newLines := filterEmptyStrings(strings.Split(line, "\n"))
		newLines = append(newLines, "")
		lines, e = script.InsertLines(lines, startLine, newLines)
		if e != nil {
			InternalError(c, e.Error())
			return errors.New(e.Error())
		}
		e = script.WriteToFile(beanFilePath, lines)
		if e != nil {
			InternalError(c, e.Error())
			return errors.New(e.Error())
		}
	} else { // 新增交易
		err = script.AppendFileInNewLine(beanFilePath, line)
	}
	if err != nil {
		if c != nil {
			InternalError(c, err.Error())
		}
		return errors.New("internal error")
	}

	debugLogDetailed("saveTransaction", "交易保存完成")
	return nil
}

// 过滤字符串数组中的空字符串
func filterEmptyStrings(arr []string) []string {
	// 创建一个新切片来存储非空字符串
	var result []string
	for _, str := range arr {
		if script.CleanString(str) != "" { // 检查字符串是否为空
			result = append(result, str)
		}
	}
	return result
}

func UpdateTransactionRawTextById(c *gin.Context) {
	var rawTextUpdateTransactionForm UpdateRawTextTransactionForm
	if err := c.ShouldBindJSON(&rawTextUpdateTransactionForm); err != nil {
		BadRequest(c, err.Error())
		return
	}
	ledgerConfig := script.GetLedgerConfigFromContext(c)

	beanFilePath, err := getBeanFilePathByTransactionId(rawTextUpdateTransactionForm.ID, ledgerConfig)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	result, e := script.BQLPrint(ledgerConfig, rawTextUpdateTransactionForm.ID)
	if e != nil {
		InternalError(c, e.Error())
		return
	}

	oldLines := filterEmptyStrings(strings.Split(result, "\n"))
	startLine, endLine, err := script.FindConsecutiveMultilineTextInFile(beanFilePath, oldLines)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	lines, e := script.RemoveLines(beanFilePath, startLine, endLine)
	if e != nil {
		InternalError(c, e.Error())
		return
	}
	newLines := filterEmptyStrings(strings.Split(rawTextUpdateTransactionForm.RawText, "\n"))
	if len(newLines) > 0 {
		lines, e = script.InsertLines(lines, startLine, newLines)
		if e != nil {
			InternalError(c, e.Error())
			return
		}
	}
	err = script.WriteToFile(beanFilePath, lines)
	if err != nil {
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

	result, e := script.BQLPrint(ledgerConfig, queryParams.ID)
	if e != nil {
		InternalError(c, e.Error())
		return
	}

	beanFilePath, err := getBeanFilePathByTransactionId(queryParams.ID, ledgerConfig)
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
	lines, e := script.RemoveLines(beanFilePath, startLine, endLine)
	if e != nil {
		InternalError(c, e.Error())
		return
	}
	err = script.WriteToFile(beanFilePath, lines)
	if err != nil {
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
	// 交易记录所在文件位置
	beanFilePath := script.GetLedgerMonthFilePath(ledgerConfig.DataPath, month)
	return beanFilePath, nil
}

type transactionPayee struct {
	Value string `bql:"distinct payee" json:"value"`
}

func QueryTransactionPayees(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	payeeList := make([]transactionPayee, 0)

	// 使用空字符串而不是 false
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

type TransactionTemplate struct {
	Id           string                 `json:"id"`
	Date         string                 `form:"date" binding:"required" json:"date"`
	TemplateName string                 `form:"templateName" binding:"required" json:"templateName"`
	Payee        string                 `form:"payee" json:"payee"`
	Desc         string                 `form:"desc" binding:"required" json:"desc"`
	Entries      []TransactionEntryForm `form:"entries" json:"entries"`
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
