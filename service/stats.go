package service

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/beancount-gs/script"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type YearMonth struct {
	Year  string `bql:"distinct year(date)" json:"year"`
	Month string `bql:"month(date)" json:"month"`
}

func MonthsList(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	// 添加排序
	queryParams := script.GetQueryParams(c)
	queryParams.OrderBy = "year, month desc"
	yearMonthList := make([]YearMonth, 0)
	err := script.BQLQueryList(ledgerConfig, &queryParams, &yearMonthList)
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

type StatsResult struct {
	Key   string
	Value string
}

func StatsTotal(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	queryParams := script.GetQueryParams(c)
	selectBql := fmt.Sprintf("SELECT '\\', root(account, 1), '\\', sum(convert(value(position), '%s')), '\\'", ledgerConfig.OperatingCurrency)
	accountTypeTotalList := make([]StatsResult, 0)
	err := script.BQLQueryListByCustomSelect(ledgerConfig, selectBql, &queryParams, &accountTypeTotalList)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	result := make(map[string]string)
	for _, total := range accountTypeTotalList {
		fields := strings.Fields(total.Value)
		if len(fields) > 1 {
			result[total.Key] = fields[0]
		}
	}

	OK(c, result)
}

type StatsQuery struct {
	Prefix string `form:"prefix" binding:"required"`
	Year   int    `form:"year"`
	Month  int    `form:"month"`
	Level  int    `form:"level"`
	Type   string `form:"type"`
}

type AccountPercentQueryResult struct {
	Account  string
	Position string
}

type AccountPercentResult struct {
	Account           string      `json:"account"`
	Amount            json.Number `json:"amount"`
	OperatingCurrency string      `json:"operatingCurrency"`
}

func StatsAccountPercent(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	var statsQuery StatsQuery
	if err := c.ShouldBindQuery(&statsQuery); err != nil {
		BadRequest(c, err.Error())
		return
	}

	queryParams := script.QueryParams{
		AccountLike: statsQuery.Prefix,
		Year:        statsQuery.Year,
		Month:       statsQuery.Month,
		Where:       true,
	}
	var bql string
	if statsQuery.Level != 0 {
		prefixNodeLen := len(strings.Split(strings.Trim(statsQuery.Prefix, ":"), ":"))
		bql = fmt.Sprintf("SELECT '\\', root(account, %d) as subAccount, '\\', sum(convert(value(position), '%s')), '\\'", statsQuery.Level+prefixNodeLen, ledgerConfig.OperatingCurrency)
	} else {
		bql = fmt.Sprintf("SELECT '\\', account, '\\', sum(convert(value(position), '%s')), '\\'", ledgerConfig.OperatingCurrency)
	}

	statsQueryResultList := make([]AccountPercentQueryResult, 0)
	err := script.BQLQueryListByCustomSelect(ledgerConfig, bql, &queryParams, &statsQueryResultList)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	result := make([]AccountPercentResult, 0)
	for _, queryRes := range statsQueryResultList {
		if queryRes.Position != "" {
			fields := strings.Fields(queryRes.Position)
			result = append(result, AccountPercentResult{Account: queryRes.Account, Amount: json.Number(fields[0]), OperatingCurrency: fields[1]})
		}
	}
	OK(c, result)
}

type AccountTrendResult struct {
	Date              string      `json:"date"`
	Amount            json.Number `json:"amount"`
	OperatingCurrency string      `json:"operatingCurrency"`
}

func StatsAccountTrend(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	var statsQuery StatsQuery
	if err := c.ShouldBindQuery(&statsQuery); err != nil {
		BadRequest(c, err.Error())
		return
	}

	queryParams := script.QueryParams{
		AccountLike: statsQuery.Prefix,
		Year:        statsQuery.Year,
		Month:       statsQuery.Month,
		Where:       true,
	}
	var bql string
	switch {
	case statsQuery.Type == "day":
		bql = fmt.Sprintf("SELECT '\\', date, '\\', sum(convert(value(position), '%s')), '\\'", ledgerConfig.OperatingCurrency)
	case statsQuery.Type == "month":
		bql = fmt.Sprintf("SELECT '\\', year, '-', month, '\\', sum(convert(value(position), '%s')), '\\'", ledgerConfig.OperatingCurrency)
	case statsQuery.Type == "year":
		bql = fmt.Sprintf("SELECT '\\', year, '\\', sum(convert(value(position), '%s')), '\\'", ledgerConfig.OperatingCurrency)
	case statsQuery.Type == "sum":
		bql = fmt.Sprintf("SELECT '\\', date, '\\', convert(balance, '%s'), '\\'", ledgerConfig.OperatingCurrency)
	default:
		OK(c, new([]string))
		return
	}

	statsResultList := make([]StatsResult, 0)
	err := script.BQLQueryListByCustomSelect(ledgerConfig, bql, &queryParams, &statsResultList)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	result := make([]AccountTrendResult, 0)
	for _, stats := range statsResultList {
		commodities := strings.Split(stats.Value, ",")
		// 多币种的处理方式：例如 75799.78 USD, 18500.00 IRAUSD, 176 VACHR
		// 选择账本默认（ledgerConfig.OperatingCurrency）币种的值
		var selectedCommodity = commodities[0]
		for _, commodity := range commodities {
			if strings.Contains(commodity, " "+ledgerConfig.OperatingCurrency) {
				selectedCommodity = commodity
				break
			}
		}

		fields := strings.Fields(selectedCommodity)
		amount, _ := decimal.NewFromString(fields[0])

		var date = stats.Key
		// 月格式化日期
		if statsQuery.Type == "month" {
			yearMonth := strings.Split(date, "-")
			date = fmt.Sprintf("%s-%s", strings.Trim(yearMonth[0], " "), strings.Trim(yearMonth[1], " "))
		}

		result = append(result, AccountTrendResult{Date: date, Amount: json.Number(amount.Round(2).String()), OperatingCurrency: fields[1]})
	}
	OK(c, result)
}

type AccountBalanceBQLResult struct {
	Year    string `bql:"year" json:"year"`
	Month   string `bql:"month" json:"month"`
	Day     string `bql:"day" json:"day"`
	Balance string `bql:"balance" json:"balance"`
}

type AccountBalanceResult struct {
	Date              string      `json:"date"`
	Amount            json.Number `json:"amount"`
	OperatingCurrency string      `json:"operatingCurrency"`
}

func StatsAccountBalance(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	var statsQuery StatsQuery
	if err := c.ShouldBindQuery(&statsQuery); err != nil {
		BadRequest(c, err.Error())
		return
	}

	queryParams := script.QueryParams{
		AccountLike: statsQuery.Prefix,
		Year:        statsQuery.Year,
		Month:       statsQuery.Month,
		Where:       true,
	}

	balResultList := make([]AccountBalanceBQLResult, 0)
	bql := fmt.Sprintf("select '\\', year, '\\', month, '\\', day, '\\', last(convert(balance, '%s')), '\\'", ledgerConfig.OperatingCurrency)
	err := script.BQLQueryListByCustomSelect(ledgerConfig, bql, &queryParams, &balResultList)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	resultList := make([]AccountBalanceResult, 0)
	for _, bqlResult := range balResultList {
		if bqlResult.Balance != "" {
			fields := strings.Fields(bqlResult.Balance)
			amount, _ := decimal.NewFromString(fields[0])
			resultList = append(resultList, AccountBalanceResult{
				Date:              bqlResult.Year + "-" + bqlResult.Month + "-" + bqlResult.Day,
				Amount:            json.Number(amount.Round(2).String()),
				OperatingCurrency: fields[1],
			})
		}
	}
	OK(c, resultList)
}

type MonthTotalBQLResult struct {
	Year  int
	Month int
	Value string
}

type MonthTotal struct {
	Type              string      `json:"type"`
	Month             string      `json:"month"`
	Amount            json.Number `json:"amount"`
	OperatingCurrency string      `json:"operatingCurrency"`
}
type MonthTotalSort []MonthTotal

func (s MonthTotalSort) Len() int {
	return len(s)
}
func (s MonthTotalSort) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s MonthTotalSort) Less(i, j int) bool {
	iYearMonth, _ := time.Parse("2006-1", s[i].Month)
	jYearMonth, _ := time.Parse("2006-1", s[j].Month)
	return iYearMonth.Before(jYearMonth)
}

func StatsMonthTotal(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)

	monthSet := make(map[string]bool)
	queryParams := script.QueryParams{
		AccountLike: "Income",
		Where:       true,
		OrderBy:     "year, month",
	}
	// 按月查询收入
	queryIncomeBql := fmt.Sprintf("select '\\', year, '\\', month, '\\', neg(sum(convert(value(position), '%s'))), '\\'", ledgerConfig.OperatingCurrency)
	monthIncomeTotalResultList := make([]MonthTotalBQLResult, 0)
	err := script.BQLQueryListByCustomSelect(ledgerConfig, queryIncomeBql, &queryParams, &monthIncomeTotalResultList)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	monthIncomeMap := make(map[string]MonthTotalBQLResult)
	for _, income := range monthIncomeTotalResultList {
		month := fmt.Sprintf("%d-%d", income.Year, income.Month)
		monthSet[month] = true
		monthIncomeMap[month] = income
	}

	// 按月查询支出
	queryParams.AccountLike = "Expenses"
	queryExpensesBql := fmt.Sprintf("select '\\', year, '\\', month, '\\', sum(convert(value(position), '%s')), '\\'", ledgerConfig.OperatingCurrency)
	monthExpensesTotalResultList := make([]MonthTotalBQLResult, 0)
	err = script.BQLQueryListByCustomSelect(ledgerConfig, queryExpensesBql, &queryParams, &monthExpensesTotalResultList)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	monthExpensesMap := make(map[string]MonthTotalBQLResult)
	for _, expenses := range monthExpensesTotalResultList {
		month := fmt.Sprintf("%d-%d", expenses.Year, expenses.Month)
		monthSet[month] = true
		monthExpensesMap[month] = expenses
	}

	monthTotalResult := make([]MonthTotal, 0)
	// 合并结果
	var monthIncome, monthExpenses MonthTotal
	var monthIncomeAmount, monthExpensesAmount decimal.Decimal
	for month := range monthSet {
		if monthIncomeMap[month].Value != "" {
			fields := strings.Fields(monthIncomeMap[month].Value)
			amount, _ := decimal.NewFromString(fields[0])
			monthIncomeAmount = amount
			monthIncome = MonthTotal{Type: "收入", Month: month, Amount: json.Number(amount.Round(2).String()), OperatingCurrency: fields[1]}
		} else {
			monthIncome = MonthTotal{Type: "收入", Month: month, Amount: "0", OperatingCurrency: ledgerConfig.OperatingCurrency}
		}
		monthTotalResult = append(monthTotalResult, monthIncome)

		if monthExpensesMap[month].Value != "" {
			fields := strings.Fields(monthExpensesMap[month].Value)
			amount, _ := decimal.NewFromString(fields[0])
			monthExpensesAmount = amount
			monthExpenses = MonthTotal{Type: "支出", Month: month, Amount: json.Number(amount.Round(2).String()), OperatingCurrency: fields[1]}
		} else {
			monthExpenses = MonthTotal{Type: "支出", Month: month, Amount: "0", OperatingCurrency: ledgerConfig.OperatingCurrency}
		}
		monthTotalResult = append(monthTotalResult, monthExpenses)
		monthTotalResult = append(monthTotalResult, MonthTotal{Type: "结余", Month: month, Amount: json.Number(monthIncomeAmount.Sub(monthExpensesAmount).Round(2).String()), OperatingCurrency: ledgerConfig.OperatingCurrency})
	}
	sort.Sort(MonthTotalSort(monthTotalResult))
	OK(c, monthTotalResult)
}

type StatsMonthQuery struct {
	Year  int `form:"year"`
	Month int `form:"month"`
}
type StatsCalendarQueryResult struct {
	Date     string
	Account  string
	Position string
}
type StatsCalendarResult struct {
	Date           string      `json:"date"`
	Account        string      `json:"account"`
	Amount         json.Number `json:"amount"`
	Currency       string      `json:"currency"`
	CurrencySymbol string      `json:"currencySymbol"`
}

func StatsMonthCalendar(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	var statsMonthQuery StatsMonthQuery
	if err := c.ShouldBindQuery(&statsMonthQuery); err != nil {
		BadRequest(c, err.Error())
		return
	}

	queryParams := script.QueryParams{
		Year:  statsMonthQuery.Year,
		Month: statsMonthQuery.Month,
		Where: true,
	}

	bql := fmt.Sprintf("SELECT '\\', date, '\\', root(account, 1), '\\', sum(convert(value(position), '%s')), '\\'", ledgerConfig.OperatingCurrency)
	statsCalendarQueryResult := make([]StatsCalendarQueryResult, 0)
	err := script.BQLQueryListByCustomSelect(ledgerConfig, bql, &queryParams, &statsCalendarQueryResult)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	resultList := make([]StatsCalendarResult, 0)
	for _, queryRes := range statsCalendarQueryResult {
		if queryRes.Position != "" {
			fields := strings.Fields(queryRes.Position)
			resultList = append(resultList,
				StatsCalendarResult{
					Date:           queryRes.Date,
					Account:        queryRes.Account,
					Amount:         json.Number(fields[0]),
					Currency:       fields[1],
					CurrencySymbol: script.GetCommoditySymbol(fields[1]),
				})
		}
	}
	OK(c, resultList)
}

type StatsPayeeQueryResult struct {
	Payee    string
	Count    int32
	Position string
}
type StatsPayeeResult struct {
	Payee    string      `json:"payee"`
	Currency string      `json:"operatingCurrency"`
	Value    json.Number `json:"value"`
}
type StatsPayeeResultSort []StatsPayeeResult

func (s StatsPayeeResultSort) Len() int {
	return len(s)
}
func (s StatsPayeeResultSort) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s StatsPayeeResultSort) Less(i, j int) bool {
	a, _ := s[i].Value.Float64()
	b, _ := s[j].Value.Float64()
	return a <= b
}
func StatsPayee(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	var statsQuery StatsQuery
	if err := c.ShouldBindQuery(&statsQuery); err != nil {
		BadRequest(c, err.Error())
		return
	}

	queryParams := script.QueryParams{
		AccountLike: statsQuery.Prefix,
		Year:        statsQuery.Year,
		Month:       statsQuery.Month,
		Where:       true,
		Currency:    ledgerConfig.OperatingCurrency,
	}

	bql := fmt.Sprintf("SELECT '\\', payee, '\\', count(payee), '\\', sum(convert(value(position), '%s')), '\\'", ledgerConfig.OperatingCurrency)
	statsPayeeQueryResultList := make([]StatsPayeeQueryResult, 0)
	err := script.BQLQueryListByCustomSelect(ledgerConfig, bql, &queryParams, &statsPayeeQueryResultList)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	result := make([]StatsPayeeResult, 0)
	for _, l := range statsPayeeQueryResultList {
		if l.Payee != "" {
			payee := StatsPayeeResult{
				Payee:    l.Payee,
				Currency: ledgerConfig.OperatingCurrency,
			}
			if statsQuery.Type == "cot" {
				payee.Value = json.Number(decimal.NewFromInt32(l.Count).String())
			} else {
				fields := strings.Fields(l.Position)
				total, err := decimal.NewFromString(fields[0])
				if err != nil {
					panic(err)
				}
				if statsQuery.Type == "avg" {
					payee.Value = json.Number(total.Div(decimal.NewFromInt32(l.Count)).Round(2).String())
				} else {
					payee.Value = json.Number(fields[0])
				}
			}
			result = append(result, payee)
		}
	}
	sort.Sort(StatsPayeeResultSort(result))
	OK(c, result)
}

type StatsPricesResult struct {
	Date      string `json:"date"`
	Commodity string `json:"commodity"`
	Currency  string `json:"operatingCurrency"`
	Value     string `json:"value"`
}

func StatsCommodityPrice(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	output := script.BeanReportAllPrices(ledgerConfig)
	script.LogInfo(ledgerConfig.Mail, output)

	statsPricesResultList := make([]StatsPricesResult, 0)
	lines := strings.Split(output, "\n")
	// foreach lines
	for _, line := range lines {
		if strings.Trim(line, " ") == "" {
			continue
		}
		// split line by " "
		words := strings.Fields(line)
		statsPricesResultList = append(statsPricesResultList, StatsPricesResult{
			Date:      words[0],
			Commodity: words[2],
			Value:     words[3],
			Currency:  words[4],
		})
	}
	OK(c, statsPricesResultList)
}
