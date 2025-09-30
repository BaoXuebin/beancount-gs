package service

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
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
	Prefix string `form:"prefix"`
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
	Account           string          `json:"account"`
	Amount            decimal.Decimal `json:"amount"`
	OperatingCurrency string          `json:"operatingCurrency"`
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

	bql := fmt.Sprintf("SELECT '\\', account, '\\', sum(convert(value(position), '%s')), '\\'", ledgerConfig.OperatingCurrency)

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
			account := queryRes.Account
			if statsQuery.Level == 1 {
				accountType := script.GetAccountType(ledgerConfig.Id, queryRes.Account)
				account = accountType.Key + ":" + accountType.Name
			}
			amount, err := decimal.NewFromString(fields[0])
			if err == nil {
				result = append(result, AccountPercentResult{Account: account, Amount: amount, OperatingCurrency: fields[1]})
			}
		}
	}

	OK(c, aggregateAccountPercentList(result))
}

func aggregateAccountPercentList(result []AccountPercentResult) []AccountPercentResult {
	// 创建一个映射来存储连接
	nodeMap := make(map[string]AccountPercentResult)
	for _, account := range result {
		acc := account.Account
		if exist, found := nodeMap[acc]; found {
			exist.Amount = exist.Amount.Add(account.Amount)
			nodeMap[acc] = exist
		} else {
			nodeMap[acc] = account
		}
	}
	aggregateResult := make([]AccountPercentResult, 0)
	for _, value := range nodeMap {
		aggregateResult = append(aggregateResult, value)
	}
	return aggregateResult
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

type AccountSankeyResult struct {
	Nodes []AccountSankeyNode `json:"nodes"`
	Links []AccountSankeyLink `json:"links"`
}

type AccountSankeyNode struct {
	Name string `json:"name"`
}
type AccountSankeyLink struct {
	Source int             `json:"source"`
	Target int             `json:"target"`
	Value  decimal.Decimal `json:"value"`
}

func NewAccountSankeyLink() *AccountSankeyLink {
	return &AccountSankeyLink{
		Source: -1,
		Target: -1,
	}
}

type TransactionAccountPositionBQLResult struct {
	Id       string
	Account  string
	Position string
}

type TransactionAccountPosition struct {
	Id                string
	Account           string
	AccountName       string
	Value             decimal.Decimal
	OperatingCurrency string
}

// StatsAccountSankey 统计账户流向
func StatsAccountSankey(c *gin.Context) {
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
	statsQueryResultList := make([]TransactionAccountPositionBQLResult, 0)
	var bql string
	// 账户不为空，则查询时间范围内所有涉及该账户的交易记录
	if statsQuery.Prefix != "" {
		bql = "SELECT '\\', id, '\\'"
		err := script.BQLQueryListByCustomSelect(ledgerConfig, bql, &queryParams, &statsQueryResultList)
		if err != nil {
			InternalError(c, err.Error())
			return
		}
		// 清空 account 查询条件，改为使用 ID 查询包含该账户所有交易记录
		queryParams.AccountLike = ""
		queryParams.IDList = "|"
		if len(statsQueryResultList) != 0 {
			idSet := make(map[string]bool)
			for _, bqlResult := range statsQueryResultList {
				idSet[bqlResult.Id] = true
			}
			idList := make([]string, 0, len(idSet))
			for id := range idSet {
				idList = append(idList, id)
			}
			queryParams.IDList = strings.Join(idList, "|")
		}
	}
	// 查询全部account的交易数据
	bql = fmt.Sprintf("SELECT '\\', id, '\\', account, '\\', sum(convert(value(position), '%s')), '\\'", ledgerConfig.OperatingCurrency)

	statsQueryResultList = make([]TransactionAccountPositionBQLResult, 0)
	err := script.BQLQueryListByCustomSelect(ledgerConfig, bql, &queryParams, &statsQueryResultList)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	result := make([]Transaction, 0)
	for _, queryRes := range statsQueryResultList {
		if queryRes.Position != "" {
			fields := strings.Fields(queryRes.Position)
			account := queryRes.Account
			if statsQuery.Level == 1 {
				accountType := script.GetAccountType(ledgerConfig.Id, account)
				account = accountType.Key + ":" + accountType.Name
			}
			result = append(result, Transaction{
				Id:       queryRes.Id,
				Account:  account,
				Number:   fields[0],
				Currency: fields[1],
			})
		}
	}

	OK(c, buildSankeyResult(result))
}

func buildSankeyResult(transactions []Transaction) AccountSankeyResult {
	accountSankeyResult := AccountSankeyResult{}
	accountSankeyResult.Nodes = make([]AccountSankeyNode, 0)
	accountSankeyResult.Links = make([]AccountSankeyLink, 0)
	// 构建 nodes 和 links
	var nodes []AccountSankeyNode

	// 遍历 transactions 中按id进行分组
	if len(transactions) > 0 {
		for _, transaction := range transactions {
			// 如果nodes中不存在该节点，则添加
			account := transaction.Account
			if !contains(nodes, account) {
				nodes = append(nodes, AccountSankeyNode{Name: account})
			}
		}
		accountSankeyResult.Nodes = nodes

		transactionsMap := groupTransactionsByID(transactions)
		// 声明 links
		links := make([]AccountSankeyLink, 0)
		// 遍历 transactionsMap
		for _, transactions := range transactionsMap {
			// 拼接成 links
			sourceTransaction := Transaction{}
			targetTransaction := Transaction{}
			currentLinkNode := NewAccountSankeyLink()
			// transactions 的最大长度
			maxCycle := len(transactions) * 2

			for {
				if len(transactions) == 0 || maxCycle == 0 {
					break
				}
				transaction := transactions[0]
				transactions = transactions[1:]

				account := transaction.Account
				num, err := decimal.NewFromString(transaction.Number)
				if err != nil {
					continue
				}
				if currentLinkNode.Source == -1 && num.IsNegative() {
					if sourceTransaction.Account == "" {
						sourceTransaction = transaction
					}
					currentLinkNode.Source = indexOf(nodes, account)
					if currentLinkNode.Target == -1 {
						currentLinkNode.Value = num
					} else {
						// 比较 link node value 和 num 大小
						delta := currentLinkNode.Value.Add(num)
						if delta.IsZero() {
							currentLinkNode.Value = num.Abs()
						} else if delta.IsNegative() { // source > target
							targetNumber, _ := decimal.NewFromString(targetTransaction.Number)
							currentLinkNode.Value = targetNumber.Abs()
							sourceTransaction.Number = delta.String()
							transactions = append(transactions, sourceTransaction)
						} else { // source < target
							targetTransaction.Number = delta.String()
							transactions = append(transactions, targetTransaction)
						}
						// 完成一个 linkNode 的构建，重置判定条件
						sourceTransaction.Account = ""
						targetTransaction.Account = ""
						links = append(links, *currentLinkNode)
						currentLinkNode = NewAccountSankeyLink()
					}
				} else if currentLinkNode.Target == -1 && num.IsPositive() {
					if targetTransaction.Account == "" {
						targetTransaction = transaction
					}
					currentLinkNode.Target = indexOf(nodes, account)
					if currentLinkNode.Source == -1 {
						currentLinkNode.Value = num
					} else {
						delta := currentLinkNode.Value.Add(num)
						if delta.IsZero() {
							currentLinkNode.Value = num.Abs()
						} else if delta.IsNegative() { // source > target
							currentLinkNode.Value = num.Abs()
							sourceTransaction.Number = delta.String()
							transactions = append(transactions, sourceTransaction)
						} else { // source < target
							sourceNumber, _ := decimal.NewFromString(sourceTransaction.Number)
							currentLinkNode.Value = sourceNumber.Abs()
							targetTransaction.Number = delta.String()
							transactions = append(transactions, targetTransaction)
						}
						// 完成一个 linkNode 的构建，重置判定条件
						sourceTransaction.Account = ""
						targetTransaction.Account = ""
						links = append(links, *currentLinkNode)
						currentLinkNode = NewAccountSankeyLink()
					}
				} else {
					// 将当前的 transaction 加入到队列末尾
					transactions = append(transactions, transaction)
				}
				maxCycle -= 1
			}
		}
		accountSankeyResult.Links = links
		// 同样source和target的link进行归并
		accountSankeyResult.Links = aggregateLinkNodes(accountSankeyResult.Links)
		//// source/target相反的link进行合并
		//accountSankeyResult.Nodes = nodes
		// 处理桑基图的link循环指向的问题
		if hasCycle(accountSankeyResult.Links) {
			newNodes, newLinks := breakCycleAndAddNode(accountSankeyResult.Nodes, accountSankeyResult.Links)
			accountSankeyResult.Nodes = newNodes
			accountSankeyResult.Links = newLinks
		}
	}
	// 过滤 source 和 target 相同的节点

	return accountSankeyResult
}

// 检查是否存在循环引用
func hasCycle(links []AccountSankeyLink) bool {
	visited := make(map[int]bool)
	recStack := make(map[int]bool)

	var dfs func(node int) bool
	dfs = func(node int) bool {
		if recStack[node] {
			return true // 找到循环
		}
		if visited[node] {
			return false // 已访问过，不再检查
		}

		visited[node] = true
		recStack[node] = true

		// 检查所有 links，看是否有从当前节点指向其他节点
		for _, link := range links {
			if link.Source == node {
				if dfs(link.Target) {
					return true
				}
			}
		}
		recStack[node] = false // 当前节点的 DFS 结束
		return false
	}

	// 遍历所有节点
	for _, link := range links {
		if dfs(link.Source) {
			return true // 发现循环
		}
	}

	return false // 没有循环
}

// 打破循环引用，添加新的节点
func breakCycleAndAddNode(nodes []AccountSankeyNode, links []AccountSankeyLink) ([]AccountSankeyNode, []AccountSankeyLink) {
	visited := make(map[int]bool)
	recStack := make(map[int]bool)
	newNodeCount := 0 // 计数新节点

	var dfs func(node int) bool
	newNodes := make(map[int]int) // 记录新节点的映射

	dfs = func(node int) bool {
		if recStack[node] {
			return true // 找到循环
		}
		if visited[node] {
			return false // 已访问过，不再检查
		}

		visited[node] = true
		recStack[node] = true

		// 遍历所有 links，看是否有从当前节点指向其他节点
		for _, link := range links {
			if link.Source == node {
				if dfs(link.Target) {
					// 检测到循环，创建新节点
					originalNode := nodes[node]
					newNode := AccountSankeyNode{
						Name: originalNode.Name + "1", // 新节点名称
					}

					// 将新节点添加到 nodes 列表中
					nodes = append(nodes, newNode)
					newNodeIndex := len(nodes) - 1
					newNodes[node] = newNodeIndex // 记录原节点到新节点的映射

					// 更新当前节点的所有链接，将 target 指向新节点
					for i := range links {
						if links[i].Source == node {
							links[i].Target = newNodeIndex
						}
					}

					newNodeCount++ // 增加新节点计数
				}
			}
		}
		recStack[node] = false // 当前节点的 DFS 结束
		return false
	}

	// 遍历所有节点，检测循环
	for _, link := range links {
		if !visited[link.Source] {
			dfs(link.Source) // 如果未访问过，则调用 DFS
		}
	}

	return nodes, links
}

func contains(nodes []AccountSankeyNode, str string) bool {
	for _, s := range nodes {
		if s.Name == str {
			return true
		}
	}
	return false
}

func indexOf(nodes []AccountSankeyNode, str string) int {
	idx := 0
	for _, s := range nodes {
		if s.Name == str {
			return idx
		}
		idx += 1
	}
	return -1
}

func groupTransactionsByID(transactions []Transaction) map[string][]Transaction {
	grouped := make(map[string][]Transaction)

	for _, transaction := range transactions {
		grouped[transaction.Id] = append(grouped[transaction.Id], transaction)
	}

	return grouped
}

// 聚合函数，聚合相同 source 和 target（相反方向）的值
func aggregateLinkNodes(links []AccountSankeyLink) []AccountSankeyLink {
	// 创建一个映射来存储连接
	nodeMap := make(map[string]decimal.Decimal)

	for _, link := range links {
		if link.Source == link.Target {
			fmt.Printf("%-%s-%d", link.Source, link.Target, link.Value)
			continue
		}

		key := fmt.Sprintf("%d-%d", link.Source, link.Target)
		reverseKey := fmt.Sprintf("%d-%d", link.Target, link.Source)
		if existingValue, found := nodeMap[key]; found {
			// 如果已存在相同方向，累加 value
			nodeMap[key] = existingValue.Add(link.Value)
		} else if existingValue, found := nodeMap[reverseKey]; found {
			// 如果存在相反方向，确定最终的 source 和 target
			totalValue := existingValue.Sub(link.Value)
			if totalValue.IsPositive() {
				nodeMap[reverseKey] = totalValue
			} else if totalValue.IsZero() {
				delete(nodeMap, reverseKey)
			} else {
				delete(nodeMap, reverseKey)
				nodeMap[key] = totalValue.Abs()
			}
		} else {
			// 否则直接插入新的 value
			nodeMap[key] = link.Value
		}
	}

	// 将结果转换为 slice
	result := make([]AccountSankeyLink, 0)
	for key, value := range nodeMap {
		var parts = strings.Split(key, "-")
		source, _ := strconv.Atoi(parts[0])
		target, _ := strconv.Atoi(parts[1])
		result = append(result, AccountSankeyLink{Source: source, Target: target, Value: value})
	}

	return result
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
					CurrencySymbol: script.GetCommoditySymbol(ledgerConfig.Id, fields[1]),
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
		// 交易账户名称非空
		if l.Payee != "" {
			payee := StatsPayeeResult{
				Payee:    l.Payee,
				Currency: ledgerConfig.OperatingCurrency,
			}
			//查询交易次数
			if statsQuery.Type == "cot" {
				payee.Value = json.Number(decimal.NewFromInt32(l.Count).String())
			} else {
				//查询交易金额，要过滤掉空白交易金额的科目，
				// 比如 记账购买后又全额退款导致科目交易条目数>0但是累计金额=0
				if l.Position != "" {
					// 读取交易金额相关信息
					fields := strings.Fields(l.Position)
					// 交易金额
					total, err := decimal.NewFromString(fields[0])
					// 错误处理
					if err != nil {
						panic(err)
					}

					if statsQuery.Type == "avg" {
						// 如果是查询平均交易金额
						payee.Value = json.Number(total.Div(decimal.NewFromInt32(l.Count)).Round(2).String())
					} else {
						// 如果是查询总交易金额
						payee.Value = json.Number(fields[0])
					}
				}
			}
			result = append(result, payee)
		}
	}
	sort.Sort(StatsPayeeResultSort(result))
	OK(c, result)
}

func StatsCommodityPrice(c *gin.Context) {
	OK(c, script.BeanReportAllPrices(script.GetLedgerConfigFromContext(c)))
}
