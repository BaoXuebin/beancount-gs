// Package service 提供账本统计相关的服务功能，
// 包括月份列表生成、金额计算等数据处理逻辑。
package service

import (
	"encoding/json"
	"fmt" // 格式化输出

	// 提供基础日志功能
	"regexp"  // 正则表达式支持
	"sort"    // 数据排序功能
	"strconv" // 字符串与基本类型的转换
	"strings" // 字符串处理工具
	"time"    // 时间处理

	"github.com/beancount-gs/script" // 项目自定义工具库
	"github.com/gin-gonic/gin"       // HTTP Web框架
	"github.com/shopspring/decimal"  // 高精度十进制数处理
)

// YearMonth 表示年份和月份的组合结构体
// 用于存储从数据库查询的日期字段拆分结果
//
// 字段说明:
//   - Year  : 年份字符串，通过BQL的 `year(date)` 函数提取
//   - Month : 月份字符串，通过BQL的 `month(date)` 函数提取
//
// 标签说明:
//   - bql   : 定义数据库查询时的字段映射关系
//   - json  : 定义JSON序列化时的字段名称
type YearMonth struct {
	Year  string `bql:"distinct year(date)" json:"year"`
	Month string `bql:"month(date)" json:"month"`
}

// MonthsList 返回账本中所有交易记录的月份列表（格式：YYYY-MM）
//
// 该函数处理流程：
//  1. 从数据库查询去重的年份和月份
//  2. 对原始数据进行清洗和格式化
//  3. 返回标准化后的月份数组
//
// 参数:
//   - c *gin.Context : Gin 框架的请求上下文，包含：
//   - 账本配置（通过中间件注入）
//   - 查询参数（分页/过滤等）
//
// 返回值:
//   - 通过 Gin 上下文返回 JSON 响应：
//   - 成功：HTTP 200 格式 {"data": ["2023-01", "2023-02"]}
//   - 失败：HTTP 500 格式 {"error": "..."}
//
// 数据清洗规则:
//   - 自动去除年份/月份字段前后空格
//   - 处理年份和月份合并存储的情况（如 "2023 01"）
//   - 月份数字标准化为两位数（"1" → "01"）
//   - 保留无法转换的原始格式（如 "2023-Q1"）
//
// 调试模式:
//   - 启用调试模式时打印完整处理流程
//   - 使用 script.IsDebugMode() 判断
func MonthsList(c *gin.Context) {
	// 设置调试上下文
	debugCtx := script.SetupDebugContext(c)
	defer debugCtx.FinishDebugContext()

	// 从上下文获取账本配置和查询参数
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	queryParams := script.GetQueryParams(c)
	queryParams.OrderBy = "year desc, month desc" // 固定排序规则

	// 使用带请求ID的日志
	debugCtx.LogSystemDebug("QueryParams", "QueryParams: %+v", queryParams)

	// 执行数据库查询
	yearMonthList := make([]YearMonth, 0)
	err := script.BQLQueryList(debugCtx, ledgerConfig, &queryParams, &yearMonthList)
	if err != nil {
		debugCtx.LogErrorDetailed(ledgerConfig.Mail, "BQLQuery", "BQL查询失败: %v", err)
		InternalError(c, err.Error())
		return
	}

	debugCtx.LogSystemDebug("QueryResult", "Total records found: %d", len(yearMonthList))

	// 处理每个年份月份组合
	months := make([]string, 0, len(yearMonthList))

	for i, yearMonth := range yearMonthList {
		// 控制调试日志输出频率
		shouldLog := debugCtx.DebugMode && (i < 3 || i%100 == 99 || i == len(yearMonthList)-1)

		if shouldLog {
			switch {
			case i < 3:
				debugCtx.LogDebugF(ledgerConfig.Mail, "First3Items",
					"Processing item [%d]: Year='%s', Month='%s'",
					i, yearMonth.Year, yearMonth.Month)
			case i%100 == 99:
				debugCtx.LogDebugF(ledgerConfig.Mail, "Every100Items",
					"Processing item [%d]: Year='%s', Month='%s'",
					i, yearMonth.Year, yearMonth.Month)
			case i == len(yearMonthList)-1:
				debugCtx.LogDebugF(ledgerConfig.Mail, "LastItem",
					"Processing item [%d]: Year='%s', Month='%s'",
					i, yearMonth.Year, yearMonth.Month)
			}
		}

		// 基础数据清洗
		year := strings.TrimSpace(yearMonth.Year)
		month := strings.TrimSpace(yearMonth.Month)

		/* 特殊场景处理：当月份为空时，尝试从年份字段解析
		   BQL 可能返回格式：
		   - 正常情况: Year="2023", Month="1"
		   - 合并情况: Year="2023 1", Month=""
		*/
		if month == "" && strings.Contains(year, " ") {
			parts := strings.Fields(year)
			if len(parts) >= 2 {
				year = parts[0]
				month = parts[1]
				if shouldLog {
					debugCtx.LogDebugF("SplitYearField",
						"Split year field: original='%s', year='%s', month='%s'",
						yearMonth.Year, year, month)
				}
			}
		}

		// 月份数字标准化处理
		monthNum, err := strconv.Atoi(month)
		if err != nil {
			// 首次转换失败后尝试二次清理（去除可能的多余字符）
			cleanedMonth := strings.TrimSpace(month)
			// 尝试提取数字部分（如果有的话）
			if cleanedMonth != month && shouldLog {
				debugCtx.LogDebugF("CleaningMonth",
					"Cleaning month: '%s' -> '%s'", month, cleanedMonth)
			}

			monthNum, err = strconv.Atoi(cleanedMonth)
			if err != nil {
				// 最终仍转换失败则保留原始格式
				result := fmt.Sprintf("%s-%s", year, cleanedMonth)
				months = append(months, result)
				if shouldLog {
					debugCtx.LogDebugF("AtoiFailed",
						"Atoi failed for Month '%s': %v", month, err)
				}
				continue
			}
			// 如果二次清理成功，使用清理后的值
			month = cleanedMonth
		}

		// 成功转换后格式化为标准字符串
		result := fmt.Sprintf("%s-%02d", year, monthNum)
		months = append(months, result)
		if shouldLog {
			debugCtx.LogDebugF(ledgerConfig.Mail, "ConversionSuccess", "Converted Month '%s' to number: %d, Result: '%s'", month, monthNum, result)
		}
	}

	// 调试输出最终结果摘要
	if debugCtx.DebugMode {
		debugCtx.LogDebugF(ledgerConfig.Mail, "FinalResults", "Final months array length: %d", len(months))
		if len(months) > 0 {
			// 只显示前5个和后5个结果作为示例
			debugCtx.LogDebugF(ledgerConfig.Mail, "FirstResults", "First 5 results: %v", months[:min(5, len(months))])
			if len(months) > 10 {
				debugCtx.LogDebugF(ledgerConfig.Mail, "LastResults", "Last 5 results: %v", months[len(months)-5:])
			}
		}
	}

	// 返回成功响应
	OK(c, months)
}

// 辅助函数，获取最小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// StatsResult 表示统计查询结果的单个条目
//
// 字段说明:
//   - Account: 账户类型名称（Beancount标准账户类型）
//   - 示例值: "Income", "Assets", "Expenses" 等
//   - Amount: 金额字符串，包含数值和货币单位
//   - 示例值: "-5420.36 CNY", "1000.00 USD"
//
// JSON标签:
//   - 保证字段在序列化时使用小写字母
type StatsResult struct {
	Account string `json:"account"` // 账户类型，如 "Income", "Assets" 等
	Amount  string `json:"amount"`  // 金额，如 "-5420.36 CNY" 等
}

type StatsTotalQuery struct {
	Year  int `form:"year"`
	Month int `form:"month"`
	// 可以根据需要添加其他参数
	Account string `form:"account"`
	Tag     string `form:"tag"`
}

// StatsTotalQueryResult 用于解析StatsTotal查询的中间结果
type StatsTotalQueryResult struct {
	Account string `json:"account"`
	Amount  string `json:"amount"` // 格式如: "-5420.36 CNY"
}

// StatsTotal 计算并返回账本中各主要账户类型的总额统计
//
// 功能说明:
//  1. 查询所有一级账户(Income/Assets/Expenses等)的汇总金额
//  2. 自动转换金额到账本运营货币(OperatingCurrency)
//  3. 返回标准化格式的统计结果
//
// 参数:
//   - c *gin.Context: Gin请求上下文，包含:
//   - 账本配置(通过中间件注入)
//   - 查询参数(分页/过滤等)
//
// 返回值:
//   - 通过Gin上下文返回JSON响应:
//   - 成功: HTTP 200 {"Assets":"10000.00","Income":"5000.00",...}
//   - 失败: HTTP 500 {"error":"..."}
//
// 数据处理流程:
//  1. 构建BQL查询语句获取原始数据
//  2. 过滤非标准账户类型(root account)
//  3. 解析金额字符串(处理"-100.00 CNY"格式)
//  4. 验证货币单位一致性
//  5. 四舍五入保留2位小数
//
// 调试模式:
//   - 通过script.IsDebugMode()激活
//   - 记录完整处理流程到日志
//   - 包含: 查询参数、SQL语句、中间结果等
func StatsTotal(c *gin.Context) {
	debugCtx := script.SetupDebugContext(c)
	defer debugCtx.FinishDebugContext()
	// 获取账本配置
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	script.LogSystemDebugDetailed("RequestDebug", "=== DEBUG MODE START ===\nRequest URL: %s", c.Request.URL.String())
	debugCtx.LogDebugF(ledgerConfig.Mail, "StatsTotal",
		"[INFO]: 获取账本配置: %+v", ledgerConfig)

	// 绑定查询参数
	var statsTotalQuery StatsTotalQuery
	if err := c.ShouldBindQuery(&statsTotalQuery); err != nil {
		debugCtx.LogDebugF(ledgerConfig.Mail, "StatsTotal", "参数绑定失败: %v", err)
		BadRequest(c, err.Error())
		return
	}
	debugCtx.LogDebugF(ledgerConfig.Mail, "StatsTotal", "绑定查询参数: %+v", statsTotalQuery)

	// 设置查询参数
	queryParams := script.QueryParams{
		Year:  statsTotalQuery.Year,
		Month: statsTotalQuery.Month,
		Where: true,
		// GroupBy 不需要设置，因为分隔符字段会导致 GROUP BY 错误
	}
	debugCtx.LogDebugF(ledgerConfig.Mail, "StatsTotal", "构建查询参数: %+v", queryParams)

	// 构建BQL查询语句 - 使用自定义分隔符，参考StatsMonthCalendar
	selectBql := fmt.Sprintf(
		"SELECT '\\\\', root(account, 1), '\\\\', sum(convert(value(position), '%s')) AS amount, '\\\\' ",
		ledgerConfig.OperatingCurrency,
	)
	debugCtx.LogDebugF(ledgerConfig.Mail, "StatsTotal", "生成BQL语句:\n%s", selectBql)

	// 准备结果列表 - 使用与StatsMonthCalendar类似的结构体
	queryResultList := make([]StatsTotalQueryResult, 0)
	debugCtx.LogDebugF(ledgerConfig.Mail, "StatsTotal",
		"准备执行查询, 结果指针类型: %T", &queryResultList)

	// 执行BQL查询
	err := script.BQLQueryListByCustomSelect(debugCtx, ledgerConfig, selectBql, &queryParams, &queryResultList)
	if err != nil {
		script.LogErrorDetailed(ledgerConfig.Mail, "StatsTotal", "BQL查询失败: %v", err)
		InternalError(c, err.Error())
		return
	}

	debugCtx.LogDebugF(ledgerConfig.Mail, "StatsTotal",
		"查询执行成功, 获取结果数量: %d", len(queryResultList))

	// 处理查询结果 - 参考StatsMonthCalendar的处理方式
	result := make(map[string]string)
	processedCount := 0
	skippedCount := 0

	for i, queryRes := range queryResultList {
		// 调试：打印原始结果结构
		debugCtx.LogDebugF(ledgerConfig.Mail, "StatsTotal",
			"原始结果[%d]: 完整结构=%+v", i, queryRes)

		if queryRes.Amount != "" {
			// 使用与StatsMonthCalendar相同的解析逻辑
			fields := strings.Fields(queryRes.Amount)
			if len(fields) >= 2 {
				account := strings.TrimSpace(queryRes.Account)
				amountStr := fields[0]
				currency := fields[1]

				// 检查账户类型是否为主要账户类型
				validAccounts := []string{"Income", "Assets", "Liabilities", "Expenses", "Equity"}
				isValidAccount := false
				for _, validAccount := range validAccounts {
					if account == validAccount {
						isValidAccount = true
						break
					}
				}

				if isValidAccount {
					// 解析金额
					amount, err := decimal.NewFromString(amountStr)
					if err != nil {
						debugCtx.LogDebugF(ledgerConfig.Mail, "StatsTotal",
							"结果[%d] 金额解析失败: AmountStr=%s, Error=%v",
							i, amountStr, err)
						skippedCount++
						continue
					}

					// 验证货币是否匹配
					if currency != ledgerConfig.OperatingCurrency {
						debugCtx.LogDebugF(ledgerConfig.Mail, "StatsTotal",
							"结果[%d] 货币不匹配: 预期=%s, 实际=%s",
							i, ledgerConfig.OperatingCurrency, currency)
					}

					result[account] = amount.Round(2).String()
					processedCount++

					debugCtx.LogDebugF(ledgerConfig.Mail, "StatsTotal",
						"处理结果[%d]: Account=%s, Amount=%s, Currency=%s",
						i, account, amountStr, currency)
				} else {
					debugCtx.LogDebugF(ledgerConfig.Mail, "StatsTotal",
						"结果[%d] 跳过非根账户: Account=%s", i, account)
					skippedCount++
				}
			} else {
				script.LogWarn(ledgerConfig.Mail, "StatsTotal",
					"结果格式异常[%d]: Amount=%s", i, queryRes.Amount)
				skippedCount++
			}
		}
	}

	debugCtx.LogDebugF(ledgerConfig.Mail, "StatsTotal",
		"结果处理完成: 成功处理=%d, 跳过=%d, 最终结果条目=%d",
		processedCount, skippedCount, len(result))

	// 记录最终结果
	for account, amount := range result {
		debugCtx.LogDebugF(ledgerConfig.Mail, "StatsTotal",
			"最终结果 - Account=%s, Amount=%s", account, amount)
	}
	script.LogSystemDebugDetailed("RequestDebug", "=== DEBUG MODE END ===")

	OK(c, result)
}

// StatsQuery 定义统计查询的请求参数结构
//
// 字段说明:
//   - Prefix : 账户前缀过滤条件（如 "Assets"）
//   - 对应URL参数: ?prefix=Assets
//   - Year   : 年份过滤（0表示不限制）
//   - 示例: 2023
//   - Month  : 月份过滤（1-12，0表示不限制）
//   - Level  : 账户层级深度（1表示一级账户）
//   - Type   : 特殊查询类型标识
//   - 可选值: "income", "expense" 等
//
// 标签说明:
//   - form : 定义Gin框架的URL参数绑定名称
type StatsQuery struct {
	Prefix string `form:"prefix"`
	Year   int    `form:"year"`
	Month  int    `form:"month"`
	Level  int    `form:"level"`
	Type   string `form:"type"`
}

// AccountPercentQueryResult 表示账户百分比查询的原始结果
//
// 注意:
//   - 该结构体用于接收BQL查询的原始数据
//   - 字段无JSON标签，仅用于中间处理
//
// 字段说明:
//   - Account  : 完整账户路径（如 "Assets:Bank"）
//   - Position : 金额表达式字符串（如 "100.00 CNY"）
type AccountPercentQueryResult struct {
	Account  string
	Position string
}

// AccountPercentResult 表示最终返回的账户百分比数据
//
// 字段说明:
//   - Account           : 账户名称（可能被截断到指定层级）
//   - Amount            : 十进制精确金额
//   - 使用 decimal.Decimal 避免浮点精度问题
//   - OperatingCurrency : 金额对应的货币单位
//   - 保证与账本配置的运营货币一致
//
// JSON标签:
//   - 字段名使用camelCase风格
//   - 金额始终序列化为字符串格式
type AccountPercentResult struct {
	Account           string          `json:"account"`
	Amount            decimal.Decimal `json:"amount"`
	OperatingCurrency string          `json:"operatingCurrency"`
}

// StatsAccountPercent 计算账户金额占比统计
//
// 功能说明:
//   - 根据请求参数过滤账户数据
//   - 计算各账户在指定条件下的金额汇总
//   - 返回标准化格式的账户金额列表
//
// 请求参数:
//   - 通过StatsQuery结构体绑定URL参数:
//   - prefix: 账户前缀过滤
//   - year/month: 时间范围过滤
//   - level: 账户层级控制(1表示只显示一级账户)
//
// 处理流程:
//  1. 绑定查询参数并验证
//  2. 构建BQL查询语句(自动转换到运营货币)
//  3. 执行查询获取原始数据
//  4. 处理账户层级(当level=1时转换账户显示格式)
//  5. 聚合重复账户的金额
//
// 返回值:
//   - 成功: HTTP 200 格式示例:
//     [{
//     "account": "Assets:Bank",
//     "amount": "1000.00",
//     "operatingCurrency": "CNY"
//     }]
//   - 失败: 4xx/5xx错误响应
func StatsAccountPercent(c *gin.Context) {
	// 从上下文获取账本配置
	debugCtx := script.SetupDebugContext(c)
	defer debugCtx.FinishDebugContext()
	ledgerConfig := script.GetLedgerConfigFromContext(c)

	// 绑定查询参数
	var statsQuery StatsQuery
	if err := c.ShouldBindQuery(&statsQuery); err != nil {
		BadRequest(c, err.Error())
		return
	}

	// 构建查询参数
	queryParams := script.QueryParams{
		AccountLike: statsQuery.Prefix, // 账户前缀过滤
		Year:        statsQuery.Year,   // 年份过滤
		Month:       statsQuery.Month,  // 月份过滤
		Where:       true,              // 启用WHERE子句
	}

	// 构建BQL查询语句(自动货币转换)
	bql := fmt.Sprintf("SELECT '\\', account, '\\', sum(convert(value(position), '%s')), '\\'",
		ledgerConfig.OperatingCurrency)

	// 执行BQL查询
	statsQueryResultList := make([]AccountPercentQueryResult, 0)
	err := script.BQLQueryListByCustomSelect(debugCtx, ledgerConfig, bql, &queryParams, &statsQueryResultList)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	// 处理查询结果
	result := make([]AccountPercentResult, 0)
	for _, queryRes := range statsQueryResultList {
		if queryRes.Position != "" {
			// 解析金额字符串(如"100.00 CNY")
			fields := strings.Fields(queryRes.Position)

			// 处理账户显示格式
			account := queryRes.Account
			if statsQuery.Level == 1 {
				// 一级账户转换为"类型:名称"格式
				accountType := script.GetAccountType(ledgerConfig.Id, queryRes.Account)
				account = accountType.Key + ":" + accountType.Name
			}

			// 转换金额为decimal类型
			amount, err := decimal.NewFromString(fields[0])
			if err == nil {
				result = append(result, AccountPercentResult{
					Account:           account,
					Amount:            amount,
					OperatingCurrency: fields[1],
				})
			}
		}
	}

	// 返回聚合后的结果
	OK(c, aggregateAccountPercentList(result))
}

// aggregateAccountPercentList 聚合重复账户的金额
//
// 功能说明:
//   - 合并相同账户名的金额记录
//   - 使用map实现高效去重和汇总
//
// 参数:
//   - result: 原始账户金额列表
//
// 返回值:
//   - 去重后的账户列表，金额为同名账户的累加和
//
// 性能说明:
//   - 时间复杂度: O(n)
//   - 空间复杂度: O(n)
func aggregateAccountPercentList(result []AccountPercentResult) []AccountPercentResult {
	// 使用map实现账户聚合
	nodeMap := make(map[string]AccountPercentResult)

	for _, account := range result {
		acc := account.Account
		if exist, found := nodeMap[acc]; found {
			// 账户已存在时累加金额
			exist.Amount = exist.Amount.Add(account.Amount)
			nodeMap[acc] = exist
		} else {
			// 新账户直接存入map
			nodeMap[acc] = account
		}
	}

	// 将map转换为切片
	aggregateResult := make([]AccountPercentResult, 0)
	for _, value := range nodeMap {
		aggregateResult = append(aggregateResult, value)
	}

	return aggregateResult
}

// AccountTrendResult 表示账户趋势数据的单个记录
//
// 字段说明:
//   - Date: 日期字符串，格式根据查询类型变化：
//   - "day": "2023-01-15"
//   - "month": "2023-01"
//   - "year": "2023"
//   - Amount: 金额数值，使用json.Number保证精度
//   - OperatingCurrency: 金额对应的货币单位
//
// JSON序列化:
//   - 所有字段使用camelCase命名
//   - 金额始终序列化为字符串格式
type AccountTrendResult struct {
	Date              string      `json:"date"`
	Amount            json.Number `json:"amount"`
	OperatingCurrency string      `json:"operatingCurrency"`
}

// StatsAccountTrend 获取账户金额趋势数据
//
// 功能说明:
//   - 支持按日/月/年/累计四种统计维度
//   - 自动处理多币种情况，优先匹配运营货币
//   - 返回标准化格式的趋势数据
//
// 请求参数:
//   - type: 统计类型，必须为以下值之一:
//   - "day": 按日统计
//   - "month": 按月统计
//   - "year": 按年统计
//   - "sum": 累计统计
//   - 其他参数继承自StatsQuery
//
// 处理流程:
//  1. 参数绑定和验证
//  2. 根据统计类型构建不同的BQL查询
//  3. 执行查询并处理多币种情况
//  4. 格式化日期和金额数据
//  5. 返回标准化结果
//
// 返回值:
//   - 成功: HTTP 200 格式示例:
//     [{
//     "date": "2023-01",
//     "amount": "1500.00",
//     "operatingCurrency": "CNY"
//     }]
//   - 失败: 4xx/5xx错误响应
func StatsAccountTrend(c *gin.Context) {
	// 获取账本配置
	debugCtx := script.SetupDebugContext(c)
	defer debugCtx.FinishDebugContext()
	ledgerConfig := script.GetLedgerConfigFromContext(c)

	// 绑定查询参数
	var statsQuery StatsQuery
	if err := c.ShouldBindQuery(&statsQuery); err != nil {
		BadRequest(c, err.Error())
		return
	}

	// 设置基础查询参数
	queryParams := script.QueryParams{
		AccountLike: statsQuery.Prefix, // 账户前缀过滤
		Year:        statsQuery.Year,   // 年份过滤
		Month:       statsQuery.Month,  // 月份过滤
		Where:       true,              // 启用WHERE条件
	}

	// 根据统计类型构建不同的BQL查询
	var bql string
	switch {
	case statsQuery.Type == "day":
		// 按日统计查询
		bql = fmt.Sprintf("SELECT '\\', date, '\\', sum(convert(value(position), '%s')), '\\'",
			ledgerConfig.OperatingCurrency)
	case statsQuery.Type == "month":
		// 按月统计查询
		bql = fmt.Sprintf("SELECT '\\', year, '-', month, '\\', sum(convert(value(position), '%s')), '\\'",
			ledgerConfig.OperatingCurrency)
	case statsQuery.Type == "year":
		// 按年统计查询
		bql = fmt.Sprintf("SELECT '\\', year, '\\', sum(convert(value(position), '%s')), '\\'",
			ledgerConfig.OperatingCurrency)
	case statsQuery.Type == "sum":
		// 累计统计查询
		bql = fmt.Sprintf("SELECT '\\', date, '\\', convert(balance, '%s'), '\\'",
			ledgerConfig.OperatingCurrency)
	default:
		// 无效统计类型返回空数组
		OK(c, new([]string))
		return
	}

	// 执行BQL查询
	statsResultList := make([]StatsResult, 0)
	err := script.BQLQueryListByCustomSelect(debugCtx, ledgerConfig, bql, &queryParams, &statsResultList)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	// 处理查询结果
	result := make([]AccountTrendResult, 0)
	for _, stats := range statsResultList {
		// 处理多币种情况
		commodities := strings.Split(stats.Amount, ",")
		var selectedCommodity string

		/* 多币种选择策略:
		   1. 优先选择包含运营货币的币种
		   2. 找不到则使用第一个可用币种
		   3. 无数据则跳过该记录
		*/
		if len(commodities) > 1 {
			// 多币种情况
			for _, commodity := range commodities {
				if strings.Contains(strings.TrimSpace(commodity), ledgerConfig.OperatingCurrency) {
					selectedCommodity = strings.TrimSpace(commodity)
					break
				}
			}
			// 默认选择第一个币种
			if selectedCommodity == "" && len(commodities) > 0 {
				selectedCommodity = strings.TrimSpace(commodities[0])
			}
		} else if len(commodities) == 1 {
			// 单币种直接使用
			selectedCommodity = strings.TrimSpace(commodities[0])
		} else {
			// 无金额数据跳过
			continue
		}

		// 解析金额和货币
		amount, currency, err := parseAmountAndCurrency(selectedCommodity)
		if err != nil {
			script.LogError(ledgerConfig.Mail,
				fmt.Sprintf("无法解析金额: %s, error: %v", selectedCommodity, err))
			continue
		}

		// 特殊处理月份格式
		var date = stats.Account
		if statsQuery.Type == "month" {
			yearMonth := strings.Split(date, "-")
			if len(yearMonth) >= 2 {
				date = fmt.Sprintf("%s-%s",
					strings.TrimSpace(yearMonth[0]),
					strings.TrimSpace(yearMonth[1]))
			}
		}

		// 构建最终结果
		result = append(result, AccountTrendResult{
			Date:              date,
			Amount:            json.Number(amount.Round(2).String()),
			OperatingCurrency: currency,
		})
	}

	// 返回处理后的结果
	OK(c, result)
}

// parseAmountAndCurrency 解析金额和货币字符串
//
// 功能说明:
//   - 从"100.00 CNY"格式的字符串中分离金额和货币单位
//   - 采用两级解析策略:
//     1. 优先使用高性能的字符串分割
//     2. 失败时使用更健壮的正则表达式
//
// 参数:
//   - amountStr: 待解析的金额字符串，格式为"[数值][空格][货币]"
//
// 返回值:
//   - decimal.Decimal: 解析出的十进制金额
//   - string: 货币代码(如"CNY")
//   - error: 解析错误信息
//
// 示例:
//   - 输入 "-100.50 USD" → 返回 (-100.50, "USD", nil)
//   - 输入 "invalid" → 返回 (0, "", error)
func parseAmountAndCurrency(amountStr string) (decimal.Decimal, string, error) {
	// 先尝试简单的字符串分割（性能更好）
	parts := strings.Fields(strings.TrimSpace(amountStr))
	if len(parts) >= 2 {
		// 检查第一部分是否是数字
		if amount, err := decimal.NewFromString(parts[0]); err == nil {
			// 第二部分应该是货币
			return amount, parts[1], nil
		}
	}

	// 如果简单方法失败，使用正则表达式（更健壮）
	re := regexp.MustCompile(`([-\d.]+)\s+(\w+)`)
	matches := re.FindStringSubmatch(amountStr)
	if len(matches) >= 3 {
		amount, err := decimal.NewFromString(matches[1])
		if err != nil {
			return decimal.Zero, "", err
		}
		return amount, matches[2], nil
	}

	return decimal.Zero, "", fmt.Errorf("无法解析金额字符串: %s", amountStr)
}

// AccountBalanceBQLResult 表示账户余额BQL查询的原始结果
//
// 注意:
//   - 用于接收数据库原始查询结果
//   - bql标签定义数据库字段映射
//
// 字段说明:
//   - Year:    交易年份
//   - Month:   交易月份(1-12)
//   - Day:     交易日期(1-31)
//   - Balance: 余额字符串(如"100.00 CNY")
type AccountBalanceBQLResult struct {
	Year    string `bql:"year" json:"year"`
	Month   string `bql:"month" json:"month"`
	Day     string `bql:"day" json:"day"`
	Balance string `bql:"balance" json:"balance"`
}

// AccountBalanceResult 表示最终返回的账户余额数据
//
// 字段说明:
//   - Date:             日期字符串(格式"YYYY-MM-DD")
//   - Amount:           精确到小数点后2位的金额
//   - OperatingCurrency: 货币代码(与账本配置一致)
type AccountBalanceResult struct {
	Date              string      `json:"date"`
	Amount            json.Number `json:"amount"`
	OperatingCurrency string      `json:"operatingCurrency"`
}

// StatsAccountBalance 获取账户每日余额数据
//
// 功能说明:
//   - 查询指定账户的每日最后余额
//   - 自动转换到运营货币
//   - 返回标准化格式的余额历史
//
// 请求参数:
//   - prefix: 账户前缀过滤
//   - year/month: 时间范围过滤
//
// 返回值:
//   - 成功: HTTP 200 格式示例:
//     [{
//     "date": "2023-01-01",
//     "amount": "1000.00",
//     "operatingCurrency": "CNY"
//     }]
//   - 失败: 4xx/5xx错误响应
func StatsAccountBalance(c *gin.Context) {
	// 获取账本配置
	debugCtx := script.SetupDebugContext(c)
	defer debugCtx.FinishDebugContext()
	ledgerConfig := script.GetLedgerConfigFromContext(c)

	// 绑定查询参数
	var statsQuery StatsQuery
	if err := c.ShouldBindQuery(&statsQuery); err != nil {
		BadRequest(c, err.Error())
		return
	}

	// 设置查询参数
	queryParams := script.QueryParams{
		AccountLike: statsQuery.Prefix, // 账户前缀过滤
		Year:        statsQuery.Year,   // 年份过滤
		Month:       statsQuery.Month,  // 月份过滤
		Where:       true,              // 启用WHERE条件
	}

	// 执行BQL查询(获取每日最后余额)
	balResultList := make([]AccountBalanceBQLResult, 0)
	bql := fmt.Sprintf("select '\\', year, '\\', month, '\\', day, '\\', last(convert(balance, '%s')), '\\'",
		ledgerConfig.OperatingCurrency)
	err := script.BQLQueryListByCustomSelect(debugCtx, ledgerConfig, bql, &queryParams, &balResultList)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	// 转换查询结果
	resultList := make([]AccountBalanceResult, 0)
	for _, bqlResult := range balResultList {
		if bqlResult.Balance != "" {
			// 解析余额字符串
			fields := strings.Fields(bqlResult.Balance)
			amount, _ := decimal.NewFromString(fields[0])

			// 构建结果对象
			resultList = append(resultList, AccountBalanceResult{
				Date:              fmt.Sprintf("%s-%s-%s", bqlResult.Year, bqlResult.Month, bqlResult.Day),
				Amount:            json.Number(amount.Round(2).String()),
				OperatingCurrency: fields[1],
			})
		}
	}

	OK(c, resultList)
}

// AccountSankeyResult 表示桑基图数据格式
//
// 数据结构说明:
//   - 符合标准桑基图数据格式
//   - 用于展示账户间资金流动
//
// 字段说明:
//   - Nodes: 节点列表(账户)
//   - Links: 连接线列表(资金流向)
type AccountSankeyResult struct {
	Nodes []AccountSankeyNode `json:"nodes"`
	Links []AccountSankeyLink `json:"links"`
}

// AccountSankeyNode 表示桑基图中的单个节点
//
// 字段说明:
//   - Name: 账户名称(显示文本)
type AccountSankeyNode struct {
	Name string `json:"name"`
}

// AccountSankeyLink 表示桑基图中的资金流向
//
// 字段说明:
//   - Source: 源节点索引(对应Nodes数组下标)
//   - Target: 目标节点索引
//   - Value:  流转金额(使用decimal保证精度)
type AccountSankeyLink struct {
	Source int             `json:"source"`
	Target int             `json:"target"`
	Value  decimal.Decimal `json:"value"`
}

// NewAccountSankeyLink 创建桑基图连接线默认实例
//
// 返回值:
//   - 初始化Source/Target为-1(表示未连接状态)
func NewAccountSankeyLink() *AccountSankeyLink {
	return &AccountSankeyLink{
		Source: -1,
		Target: -1,
	}
}

// TransactionAccountPositionBQLResult 交易账户位置BQL查询结果
//
// 注意:
//   - 用于接收原始BQL查询数据
//   - 不直接暴露给API
//
// 字段说明:
//   - Id:       交易ID
//   - Account:  账户路径
//   - Position: 金额字符串(如"100.00 CNY")
type TransactionAccountPositionBQLResult struct {
	Id       string
	Account  string
	Position string
}

// TransactionAccountPosition 交易账户位置处理结果
//
// 字段说明:
//   - Id:                交易ID
//   - Account:           完整账户路径
//   - AccountName:       显示用账户名(可能被截断)
//   - Value:             精确金额
//   - OperatingCurrency: 货币单位
type TransactionAccountPosition struct {
	Id                string
	Account           string
	AccountName       string
	Value             decimal.Decimal
	OperatingCurrency string
}

// StatsAccountSankey 生成账户资金流向桑基图数据
//
// 功能说明:
//   - 分析指定账户或全部账户的资金流动情况
//   - 支持按时间范围和账户层级过滤
//   - 返回符合桑基图要求的数据结构
//
// 处理流程:
//  1. 如果指定了账户前缀(Prefix):
//     - 先查询该账户涉及的所有交易ID
//     - 转换为ID列表查询条件
//  2. 查询满足条件的交易数据:
//     - 自动转换金额到运营货币
//     - 按level参数处理账户显示名称
//  3. 构建桑基图节点和连接线数据
//
// 请求参数:
//   - prefix: 账户前缀过滤(为空则分析全部账户)
//   - year/month: 时间范围过滤
//   - level: 账户层级(1表示只显示一级账户)
//
// 返回值:
//   - 成功: HTTP 200 桑基图数据结构
//     {
//     "nodes": [{"name":"账户1"},...],
//     "links": [{"source":0,"target":1,"value":100},...]
//     }
//   - 失败: 4xx/5xx错误响应
func StatsAccountSankey(c *gin.Context) {
	debugCtx := script.SetupDebugContext(c)
	defer debugCtx.FinishDebugContext()
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
		err := script.BQLQueryListByCustomSelect(debugCtx, ledgerConfig, bql, &queryParams, &statsQueryResultList)
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
	err := script.BQLQueryListByCustomSelect(debugCtx, ledgerConfig, bql, &queryParams, &statsQueryResultList)
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

// buildSankeyResult 构建桑基图数据结构
//
// 功能说明:
//   - 将交易数据转换为桑基图所需的节点和连接线结构
//   - 处理资金流动方向(正负值表示流向)
//   - 自动处理循环引用和重复连接
//
// 处理流程:
//  1. 收集所有唯一账户作为节点
//  2. 按交易ID分组处理资金流动:
//     - 负值金额作为流出(source)
//     - 正值金额作为流入(target)
//  3. 特殊处理:
//     - 合并相同节点间的多条连接
//     - 检测并打破循环引用
//     - 过滤自循环节点(source=target)
//
// 参数:
//   - transactions: 交易记录列表，需包含:
//   - Account: 账户名称
//   - Number: 金额字符串(带正负号)
//   - Id: 交易ID(用于分组)
//
// 返回值:
//   - AccountSankeyResult: 完整的桑基图数据结构
//   - Nodes: 账户节点列表
//   - Links: 资金流向连接列表
//
// 注意事项:
//   - 使用decimal处理金额保证精度
//   - 循环引用会添加中间节点打破循环
//   - 最大迭代次数限制防止死循环
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

// hasCycle 检测桑基图连接中是否存在循环引用
//
// 参数:
//   - links: 桑基图连接线列表
//
// 返回值:
//   - bool: true表示存在循环引用，false表示无循环
//
// 实现说明:
//
//	使用深度优先搜索(DFS)算法检测有向图中的环
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

// breakCycleAndAddNode 打破桑基图中的循环引用
//
// 处理逻辑:
//  1. 检测到循环时创建新节点
//  2. 将循环连接重定向到新节点
//  3. 新节点命名为原节点名+"1"
//
// 参数:
//   - nodes: 原始节点列表
//   - links: 原始连接线列表
//
// 返回值:
//   - []AccountSankeyNode: 处理后的节点列表(可能包含新节点)
//   - []AccountSankeyLink: 处理后的连接线列表
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

// contains 检查节点列表中是否包含指定名称的节点
//
// 参数:
//   - nodes: 节点列表
//   - str: 要查找的节点名称
//
// 返回值:
//   - bool: true表示包含，false表示不包含
func contains(nodes []AccountSankeyNode, str string) bool {
	for _, s := range nodes {
		if s.Name == str {
			return true
		}
	}
	return false
}

// indexOf 查找节点在列表中的索引位置
//
// 参数:
//   - nodes: 节点列表
//   - str: 要查找的节点名称
//
// 返回值:
//   - int: 节点索引(未找到返回-1)
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

// groupTransactionsByID 按交易ID分组交易记录
//
// 参数:
//   - transactions: 交易记录列表
//
// 返回值:
//   - map[string][]Transaction: 按ID分组的交易记录映射
func groupTransactionsByID(transactions []Transaction) map[string][]Transaction {
	grouped := make(map[string][]Transaction)

	for _, transaction := range transactions {
		grouped[transaction.Id] = append(grouped[transaction.Id], transaction)
	}

	return grouped
}

// aggregateLinkNodes 聚合桑基图连接线
//
// 处理逻辑:
//  1. 合并相同方向的连接(值相加)
//  2. 抵消相反方向的连接(值相减)
//  3. 过滤自循环连接(source=target)
//
// 参数:
//   - links: 原始连接线列表
//
// 返回值:
//   - []AccountSankeyLink: 聚合后的连接线列表
func aggregateLinkNodes(links []AccountSankeyLink) []AccountSankeyLink {
	// 创建一个映射来存储连接
	nodeMap := make(map[string]decimal.Decimal)

	for _, link := range links {
		if link.Source == link.Target {

			fmt.Printf("%v-%v-%v", link.Source, link.Target, link.Value)
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

// MonthTotalBQLResult 表示月度统计查询的原始结果结构体
//
// 字段说明:
//   - Year  : 年份数值(如2023)
//   - Month : 月份数值(1-12)
//   - Value : 金额字符串(格式:"100.00 CNY")
//
// 注意:
//   - 用于接收BQL查询的原始数据
//   - 不直接暴露给API接口
type MonthTotalBQLResult struct {
	Year  int
	Month int
	Value string
}

// MonthTotal 表示最终返回的月度统计数据
//
// 字段说明:
//   - Type              : 统计类型("收入"/"支出"/"结余")
//   - Month             : 年月字符串(格式:"YYYY-MM")
//   - Amount            : 精确金额(使用json.Number保证精度)
//   - OperatingCurrency : 运营货币代码(如"CNY")
//
// JSON标签:
//   - 所有字段使用小写命名
//   - 金额始终序列化为字符串格式
type MonthTotal struct {
	Type              string      `json:"type"`
	Month             string      `json:"month"`
	Amount            json.Number `json:"amount"`
	OperatingCurrency string      `json:"operatingCurrency"`
}

// MonthTotalSort 实现sort.Interface用于MonthTotal切片排序
//
// 功能说明:
//   - 按月份升序排列
//   - 支持跨年排序(如2022-12, 2023-01)
//
// 实现方法:
//   - Len()  : 获取切片长度
//   - Swap() : 交换元素位置
//   - Less() : 比较时间先后
type MonthTotalSort []MonthTotal

// MonthTotalSort 实现 sort.Interface 接口用于 MonthTotal 切片排序
//
// 功能说明:
//   - 提供按月份升序排列的能力
//   - 支持跨年排序(如 2022-12, 2023-01)
//
// 实现方法:
//   - Len(): 返回切片长度
//   - Swap(i, j int): 交换两个元素的位置
//   - Less(i, j int): 比较两个元素的月份先后
//
// 排序规则:
//   - 使用 time.Parse 解析 "2006-1" 格式的月份字符串
//   - 比较实际时间先后顺序
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

// StatsMonthTotal 获取月度收支统计数据
//
// 功能说明:
//   - 查询指定时间范围内的收入和支出数据
//   - 自动计算每月结余
//   - 返回按月份排序的完整统计数据
//
// 处理流程:
//  1. 查询收入数据(Income 账户)
//     - 使用 neg() 将支出转为正数
//  2. 查询支出数据(Expenses 账户)
//  3. 合并收入和支出数据
//  4. 计算每月结余(收入-支出)
//  5. 按月份排序后返回
//
// 参数:
//   - c *gin.Context: Gin 请求上下文
//   - 自动获取账本配置
//
// 返回值:
//   - HTTP 200: 成功返回排序后的月度统计数据
//   - HTTP 500: 查询失败返回错误信息
//
// 数据结构:
//   - 每月包含三条记录:
//     1. 收入记录(Type="收入")
//     2. 支出记录(Type="支出")
//     3. 结余记录(Type="结余")
//
// 货币处理:
//   - 自动转换到账本运营货币(OperatingCurrency)
//   - 金额保留2位小数
func StatsMonthTotal(c *gin.Context) {
	debugCtx := script.SetupDebugContext(c)
	defer debugCtx.FinishDebugContext()
	ledgerConfig := script.GetLedgerConfigFromContext(c)

	// 准备查询参数
	monthSet := make(map[string]bool) // 记录存在的月份
	queryParams := script.QueryParams{
		AccountLike: "Income", // 收入账户
		Where:       true,
		OrderBy:     "year, month", // 按年月排序
	}

	// 1. 查询收入数据
	queryIncomeBql := fmt.Sprintf("select '\\', year, '\\', month, '\\', neg(sum(convert(value(position), '%s'))), '\\'",
		ledgerConfig.OperatingCurrency)
	monthIncomeTotalResultList := make([]MonthTotalBQLResult, 0)
	err := script.BQLQueryListByCustomSelect(debugCtx, ledgerConfig, queryIncomeBql, &queryParams, &monthIncomeTotalResultList)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	// 存储收入数据
	monthIncomeMap := make(map[string]MonthTotalBQLResult)
	for _, income := range monthIncomeTotalResultList {
		month := fmt.Sprintf("%d-%d", income.Year, income.Month)
		monthSet[month] = true
		monthIncomeMap[month] = income
	}

	// 2. 查询支出数据
	queryParams.AccountLike = "Expenses" // 支出账户
	queryExpensesBql := fmt.Sprintf("select '\\', year, '\\', month, '\\', sum(convert(value(position), '%s')), '\\'",
		ledgerConfig.OperatingCurrency)
	monthExpensesTotalResultList := make([]MonthTotalBQLResult, 0)
	err = script.BQLQueryListByCustomSelect(debugCtx, ledgerConfig, queryExpensesBql, &queryParams, &monthExpensesTotalResultList)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	// 存储支出数据
	monthExpensesMap := make(map[string]MonthTotalBQLResult)
	for _, expenses := range monthExpensesTotalResultList {
		month := fmt.Sprintf("%d-%d", expenses.Year, expenses.Month)
		monthSet[month] = true
		monthExpensesMap[month] = expenses
	}

	// 3. 合并数据并计算结余
	monthTotalResult := make([]MonthTotal, 0)
	var monthIncome, monthExpenses MonthTotal
	var monthIncomeAmount, monthExpensesAmount decimal.Decimal

	for month := range monthSet {
		// 处理收入数据
		if monthIncomeMap[month].Value != "" {
			fields := strings.Fields(monthIncomeMap[month].Value)
			amount, _ := decimal.NewFromString(fields[0])
			monthIncomeAmount = amount
			monthIncome = MonthTotal{
				Type:              "收入",
				Month:             month,
				Amount:            json.Number(amount.Round(2).String()),
				OperatingCurrency: fields[1],
			}
		} else {
			// 无收入数据时填充0值
			monthIncome = MonthTotal{
				Type:              "收入",
				Month:             month,
				Amount:            "0",
				OperatingCurrency: ledgerConfig.OperatingCurrency,
			}
		}

		// 处理支出数据
		if monthExpensesMap[month].Value != "" {
			fields := strings.Fields(monthExpensesMap[month].Value)
			amount, _ := decimal.NewFromString(fields[0])
			monthExpensesAmount = amount
			monthExpenses = MonthTotal{
				Type:              "支出",
				Month:             month,
				Amount:            json.Number(amount.Round(2).String()),
				OperatingCurrency: fields[1],
			}
		} else {
			// 无支出数据时填充0值
			monthExpenses = MonthTotal{
				Type:              "支出",
				Month:             month,
				Amount:            "0",
				OperatingCurrency: ledgerConfig.OperatingCurrency,
			}
		}

		// 添加收入、支出记录
		monthTotalResult = append(monthTotalResult, monthIncome, monthExpenses)

		// 计算并添加结余记录
		monthTotalResult = append(monthTotalResult, MonthTotal{
			Type:              "结余",
			Month:             month,
			Amount:            json.Number(monthIncomeAmount.Sub(monthExpensesAmount).Round(2).String()),
			OperatingCurrency: ledgerConfig.OperatingCurrency,
		})
	}

	// 4. 按月份排序
	sort.Sort(MonthTotalSort(monthTotalResult))

	// 返回结果
	OK(c, monthTotalResult)
}

// StatsMonthQuery 定义月度日历查询参数结构
//
// 字段说明:
//   - Year:  查询年份(如2023)
//   - Month: 查询月份(1-12)
//
// 标签说明:
//   - form: 定义Gin框架的URL参数绑定名称
//   - 示例: /api/stats/calendar?year=2023&month=7
type StatsMonthQuery struct {
	Year  int `form:"year"`
	Month int `form:"month"`
}

// StatsCalendarQueryResult 表示日历查询的原始结果
//
// 注意:
//   - 用于接收BQL查询的原始数据
//   - 不直接暴露给API接口
//
// 字段说明:
//   - Date:     交易日期(格式"YYYY-MM-DD")
//   - Account:  一级账户名称(如"Assets:Bank")
//   - Position: 金额字符串(格式"100.00 CNY")
type StatsCalendarQueryResult struct {
	Date     string
	Account  string
	Position string
}

// StatsCalendarResult 表示最终返回的日历统计数据
//
// 字段说明:
//   - Date:           交易日期(格式"YYYY-MM-DD")
//   - Account:        一级账户名称(如"Assets:Bank")
//   - Amount:         精确金额(使用json.Number保证精度)
//   - Currency:       货币代码(如"CNY")
//   - CurrencySymbol: 货币符号(如"¥")
//
// JSON标签:
//   - 所有字段使用camelCase命名
//   - 金额始终序列化为字符串格式
type StatsCalendarResult struct {
	Date           string      `json:"date"`
	Account        string      `json:"account"`
	Amount         json.Number `json:"amount"`
	Currency       string      `json:"currency"`
	CurrencySymbol string      `json:"currencySymbol"`
}

// StatsMonthCalendar 获取指定月份的日历统计数据
//
// 功能说明:
//   - 查询指定年月的每日交易数据
//   - 按一级账户分类汇总金额
//   - 返回包含货币符号的格式化数据
//
// 请求参数:
//   - year: 查询年份(如2023)
//   - month: 查询月份(1-12)
//   - 通过StatsMonthQuery结构体绑定URL参数
//
// 处理流程:
//  1. 绑定并验证查询参数
//  2. 构建BQL查询语句:
//     - 按日期和一级账户分组
//     - 自动转换金额到运营货币
//  3. 执行查询并处理结果:
//     - 解析金额和货币信息
//     - 添加货币符号
//
// 返回值:
//   - 成功: HTTP 200 返回[]StatsCalendarResult, 结构如下:
//     [{
//     "date": "2023-07-01",
//     "account": "Assets:Bank",
//     "amount": "1000.00",
//     "currency": "CNY",
//     "currencySymbol": "¥"
//     }]
//   - 失败:
//   - HTTP 400: 参数绑定错误
//   - HTTP 500: 查询执行错误
//
// 数据结构说明:
//   - 每条记录包含日期、账户、金额和货币信息
//   - 金额使用json.Number保证精度
//   - 货币符号通过script.GetCommoditySymbol获取
func StatsMonthCalendar(c *gin.Context) {
	debugCtx := script.SetupDebugContext(c)
	defer debugCtx.FinishDebugContext()
	// 获取账本配置
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	debugCtx.LogDebugF(ledgerConfig.Mail, "StatsMonthCalendar",
		"[INFO]: 获取账本配置: %+v", ledgerConfig)

	// 绑定查询参数
	var statsMonthQuery StatsMonthQuery
	if err := c.ShouldBindQuery(&statsMonthQuery); err != nil {
		debugCtx.LogDebugF(ledgerConfig.Mail, "StatsMonthCalendar", "参数绑定失败: %v", err)
		BadRequest(c, err.Error())
		return
	}
	debugCtx.LogDebugF(ledgerConfig.Mail, "StatsMonthCalendar", "绑定查询参数: %+v", statsMonthQuery)

	// 设置查询参数
	queryParams := script.QueryParams{
		Year:  statsMonthQuery.Year,
		Month: statsMonthQuery.Month,
		Where: true,
		// GroupBy: "date, root(account, 1)",
	}
	debugCtx.LogDebugF(ledgerConfig.Mail, "StatsMonthCalendar", "构建查询参数: %+v", queryParams)

	// 构建BQL查询语句
	bql := fmt.Sprintf(
		"SELECT '\\', date, '\\', root(account, 1), '\\', sum(convert(value(position), '%s')) AS amount, '\\' ",
		ledgerConfig.OperatingCurrency,
	)
	// bql := fmt.Sprintf(
	// 	"SELECT  date,  root(account, 1),  sum(convert(value(position), '%s')) AS amount  ",
	// 	ledgerConfig.OperatingCurrency,
	// )
	debugCtx.LogDebugF(ledgerConfig.Mail, "StatsMonthCalendar", "生成BQL语句:\n%s", bql)

	// 执行查询
	statsCalendarQueryResult := make([]StatsCalendarQueryResult, 0)
	debugCtx.LogDebugF(ledgerConfig.Mail, "StatsMonthCalendar",
		"准备执行查询, 结果指针类型: %T", &statsCalendarQueryResult)

	err := script.BQLQueryListByCustomSelect(debugCtx, ledgerConfig, bql, &queryParams, &statsCalendarQueryResult)
	if err != nil {
		script.LogError(ledgerConfig.Mail,
			fmt.Sprintf("StatsMonthCalendar - 查询执行失败: %v", err))
		InternalError(c, err.Error())
		return
	}
	debugCtx.LogDebugF(ledgerConfig.Mail, "StatsMonthCalendar",
		"查询执行成功, 获取结果数量: %d", len(statsCalendarQueryResult))

	// 处理查询结果
	resultList := make([]StatsCalendarResult, 0)
	for i, queryRes := range statsCalendarQueryResult {
		if queryRes.Position != "" {
			fields := strings.Fields(queryRes.Position)
			if len(fields) >= 2 {
				resultList = append(resultList,
					StatsCalendarResult{
						Date:           queryRes.Date,
						Account:        queryRes.Account,
						Amount:         json.Number(fields[0]),
						Currency:       fields[1],
						CurrencySymbol: script.GetCommoditySymbol(ledgerConfig.Id, fields[1]),
					})
				debugCtx.LogDebugF(ledgerConfig.Mail, "StatsMonthCalendar",
					"处理结果[%d]: Date=%s, Account=%s, Amount=%s, Currency=%s",
					i,
					queryRes.Date,
					queryRes.Account,
					fields[0],
					fields[1])
			} else {
				script.LogWarn(ledgerConfig.Mail, "StatsMonthCalendar",
					"结果格式异常[%d]: Position=%s", i, queryRes.Position)
			}
		}
	}

	debugCtx.LogDebugF(ledgerConfig.Mail, "StatsMonthCalendar",
		"返回结果数量: %d", len(resultList))
	OK(c, resultList)
}

// StatsPayeeQueryResult 表示收款人统计查询的原始结果
//
// 注意:
//   - 用于接收BQL查询的原始数据
//   - 不直接暴露给API接口
//
// 字段说明:
//   - Payee:    收款人名称
//   - Count:    交易次数
//   - Position: 金额字符串(格式:"100.00 CNY")
type StatsPayeeQueryResult struct {
	Payee    string
	Count    int32
	Position string
}

// StatsPayeeResult 表示最终返回的收款人统计数据
//
// 字段说明:
//   - Payee:    收款人名称
//   - Currency: 货币代码(如"CNY")
//   - Value:    统计值(交易次数或金额)
//
// JSON标签:
//   - 所有字段使用camelCase命名
//   - 金额始终序列化为字符串格式
type StatsPayeeResult struct {
	Payee    string      `json:"payee"`
	Currency string      `json:"operatingCurrency"`
	Value    json.Number `json:"value"`
}

// StatsPayeeResultSort 实现sort.Interface用于StatsPayeeResult切片排序
//
// 功能说明:
//   - 按Value值升序排列
//   - 支持按交易次数或金额排序
//
// 实现方法:
//   - Len(): 返回切片长度
//   - Swap(i, j int): 交换两个元素的位置
//   - Less(i, j int): 比较两个元素的Value大小
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

// StatsPayee 获取收款人统计数据
//
// 功能说明:
//   - 查询指定条件下的收款人交易数据
//   - 支持按交易次数或金额统计
//   - 返回排序后的收款人统计结果
//
// 请求参数:
//   - prefix: 账户前缀过滤
//   - year/month: 时间范围过滤
//   - type: 统计类型(cot=交易次数, avg=平均金额, 其他=总金额)
//   - 通过StatsQuery结构体绑定URL参数
//
// 处理流程:
//  1. 绑定并验证查询参数
//  2. 构建BQL查询语句:
//     - 按收款人分组统计
//     - 自动转换金额到运营货币
//  3. 执行查询并处理结果:
//     - 根据type参数计算不同统计值
//     - 过滤无效数据(如金额为0的交易)
//     - 排序结果
//
// 返回值:
//   - 成功: HTTP 200 返回[]StatsPayeeResult, 结构如下:
//     [{
//     "payee": "收款人A",
//     "operatingCurrency": "CNY",
//     "value": "1000.00"
//     }]
//   - 失败:
//   - HTTP 400: 参数绑定错误
//   - HTTP 500: 查询执行错误
//
// 数据结构说明:
//   - 每条记录包含收款人名称、货币和统计值
//   - 统计值可能是交易次数、平均金额或总金额
//   - 使用json.Number保证精度
func StatsPayee(c *gin.Context) {
	debugCtx := script.SetupDebugContext(c)
	defer debugCtx.FinishDebugContext()
	// 获取账本配置
	ledgerConfig := script.GetLedgerConfigFromContext(c)

	// 绑定查询参数
	var statsQuery StatsQuery
	if err := c.ShouldBindQuery(&statsQuery); err != nil {
		BadRequest(c, err.Error())
		return
	}

	// 设置查询参数
	queryParams := script.QueryParams{
		AccountLike: statsQuery.Prefix,
		Year:        statsQuery.Year,
		Month:       statsQuery.Month,
		Where:       true,
		Currency:    ledgerConfig.OperatingCurrency,
	}

	// 构建BQL查询语句
	bql := fmt.Sprintf(
		"SELECT '\\', payee, '\\', count(payee), '\\', sum(convert(value(position), '%s')), '\\'",
		ledgerConfig.OperatingCurrency,
	)

	// 执行查询
	statsPayeeQueryResultList := make([]StatsPayeeQueryResult, 0)
	err := script.BQLQueryListByCustomSelect(debugCtx, ledgerConfig, bql, &queryParams, &statsPayeeQueryResultList)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	// 处理查询结果
	result := make([]StatsPayeeResult, 0)
	for _, l := range statsPayeeQueryResultList {
		// 过滤空收款人
		if l.Payee != "" {
			payee := StatsPayeeResult{
				Payee:    l.Payee,
				Currency: ledgerConfig.OperatingCurrency,
			}

			// 根据查询类型处理不同统计值
			if statsQuery.Type == "cot" {
				// 交易次数统计
				payee.Value = json.Number(decimal.NewFromInt32(l.Count).String())
			} else {
				// 金额统计(过滤金额为0的交易)
				if l.Position != "" {
					fields := strings.Fields(l.Position)
					total, err := decimal.NewFromString(fields[0])
					if err != nil {
						panic(err)
					}

					if statsQuery.Type == "avg" {
						// 平均金额统计
						payee.Value = json.Number(total.Div(decimal.NewFromInt32(l.Count)).Round(2).String())
					} else {
						// 总金额统计
						payee.Value = json.Number(fields[0])
					}
				}
			}
			result = append(result, payee)
		}
	}

	// 排序结果
	sort.Sort(StatsPayeeResultSort(result))

	// 返回成功响应
	OK(c, result)
}

// StatsCommodityPrice 获取商品价格数据
//
// 功能说明:
//   - 查询账本中所有商品的最新价格
//   - 返回标准化格式的价格数据
//
// 参数:
//   - c *gin.Context: Gin请求上下文
//   - 自动获取账本配置
//
// 返回值:
//   - HTTP 200: 成功返回商品价格数据
//   - 数据结构由script.BeanReportAllPrices决定
//
// 注意事项:
//   - 直接调用底层script包的BeanReportAllPrices函数
//   - 返回数据格式与底层实现保持一致
func StatsCommodityPrice(c *gin.Context) {
	debugCtx := script.SetupDebugContext(c)
	// 获取账本配置并查询价格数据
	OK(c, script.BeanReportAllPrices(debugCtx, script.GetLedgerConfigFromContext(c)))
}
