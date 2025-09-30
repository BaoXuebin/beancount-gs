package script

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// 调试模式下的错误处理
// 修改 handleError 函数中的提示
// handleError 处理错误，根据调试模式决定处理方式：
// - 调试模式下 panic 抛出错误
// - 生产环境下记录警告日志
// 参数：
//
//	err: 错误对象
//	message: 自定义错误信息
func handleError(err error, message string) {
	if IsDebugMode() {
		panic(fmt.Sprintf("%s: %v", message, err))
	} else {
		// 生产环境下记录警告日志
		log.Printf("警告: %s: %v", message, err)
	}
}

/*
QueryParams 结构体定义了查询参数，包含多个字段用于构建查询条件。
每个字段都带有 bql 标签，用于指定在 BQL 查询中的对应字段名。
*/
type QueryParams struct {
	From               bool   `bql:"From"`             // 是否包含 From 子句
	FromYear           int    `bql:"year ="`           // From 子句中的年份条件
	FromMonth          int    `bql:"month ="`          // From 子句中的月份条件
	Where              bool   `bql:"where"`            // 是否包含 Where 子句
	ID                 string `bql:"id ="`             // ID 等于条件
	IDList             string `bql:"id in"`            // ID 列表条件
	Currency           string `bql:"currency ="`       // 货币等于条件
	Year               int    `bql:"year ="`           // 年份等于条件
	Month              int    `bql:"month ="`          // 月份等于条件
	Tag                string `bql:"in tags"`          // 用于 tag in tags 条件
	TagNotNull         string `bql:"tags IS NOT NULL"` // 标签非空条件
	Account            string `bql:"account ="`        // 账户等于条件
	AccountLike        string `bql:"account ~"`        // 账户模糊匹配条件
	StrictAccountMatch bool   // true表示账户必须使用精确匹配，false表示使用模糊匹配
	GroupBy            string `bql:"group by"` // 分组条件
	OrderBy            string `bql:"order by"` // 排序条件
	Limit              int    `bql:"limit"`    // 限制结果数量
	Path               string // 查询路径
	Offset             int    // 查询偏移量
	DateRange          bool   // 启用日期范围查询
}

// HasConditions 检查查询参数是否包含任何条件
// 返回 true 如果 Year, Month, Account, AccountLike, Tag, ID 或 Currency 任一字段不为零值
func (queryParams *QueryParams) HasConditions() bool {
	return queryParams.Year != 0 || queryParams.Month != 0 ||
		queryParams.Account != "" || queryParams.AccountLike != "" ||
		queryParams.Tag != "" || queryParams.ID != "" ||
		queryParams.Currency != ""
}

// GetQueryParams 从 gin.Context 中解析查询参数并返回 QueryParams 结构体
// 支持以下查询参数：
//   - year/month: 数值型参数，用于时间范围筛选
//   - tag/account/type/id/idList/currency/path: 字符串型参数
//   - groupBy/orderBy: 分组和排序参数
//   - limit: 限制返回结果数量
//   - strictMode: 布尔值，控制账户匹配模式（精确/模糊）
//
// 默认值：
//   - StrictAccountMatch: true (精确匹配)
//   - OrderBy: "date desc"
//   - Limit: 100
//   - Year/Month: 0 (表示不限制)
//
// 注意：所有参数都会经过有效性检查（非空且不为"undefined"/"null"）
func GetQueryParams(c *gin.Context) QueryParams {
	queryParams := QueryParams{
		StrictAccountMatch: true,
		OrderBy:            "date desc",
		Limit:              100,
	}

	// 辅助函数检查有效值
	isValidParam := func(val string) bool {
		return val != "" && val != "undefined" && val != "null" && val != "None"
	}

	// 数值型条件
	if year := c.Query("year"); isValidParam(year) {
		if val, err := strconv.Atoi(year); err == nil && val > 0 {
			queryParams.Year = val
			DebugLogWithContext("GetQueryParams", "设置有效年份: %d", val)
		} else {
			WarnLogWithContext("GetQueryParams", "无效年份参数: %s", year)
		}
	}

	if month := c.Query("month"); isValidParam(month) {
		if val, err := strconv.Atoi(month); err == nil && val > 0 && val <= 12 {
			queryParams.Month = val
			DebugLogWithContext("GetQueryParams", "设置有效月份: %d", val)
		} else {
			WarnLogWithContext("GetQueryParams", "无效月份参数: %s", month)
		}
	}

	// 字符串条件
	setStringParam := func(param *string, queryKey string) {
		if val := c.Query(queryKey); isValidParam(val) {
			*param = val
			DebugLogWithContext("GetQueryParams", "设置%s: %s", queryKey, val)
		}
	}

	setStringParam(&queryParams.Tag, "tag")
	setStringParam(&queryParams.Account, "account")
	setStringParam(&queryParams.ID, "id")
	setStringParam(&queryParams.IDList, "idList")
	setStringParam(&queryParams.Currency, "currency")
	setStringParam(&queryParams.Path, "path")
	setStringParam(&queryParams.GroupBy, "groupBy")

	// 特殊处理type参数
	if accType := c.Query("type"); isValidParam(accType) {
		queryParams.AccountLike = accType
		queryParams.StrictAccountMatch = false
		DebugLogWithContext("GetQueryParams", "设置账户类型(模糊匹配): %s", accType)
	}

	// 处理排序参数
	if orderBy := c.Query("orderBy"); isValidParam(orderBy) {
		queryParams.OrderBy = orderBy
	} else if queryParams.Year > 0 || queryParams.Month > 0 {
		queryParams.OrderBy = "year desc, month desc"
	}

	// 处理limit参数
	if limit := c.Query("limit"); isValidParam(limit) {
		if val, err := strconv.Atoi(limit); err == nil && val > 0 {
			queryParams.Limit = val
		}
	}

	// 处理严格模式参数
	if strictMode := c.Query("strictMode"); isValidParam(strictMode) {
		if strict, err := strconv.ParseBool(strictMode); err == nil {
			queryParams.StrictAccountMatch = strict
			DebugLogWithContext("GetQueryParams", "设置严格模式: %t", strict)
		}
	}

	// 验证必要参数
	if queryParams.Year == 0 && queryParams.Month == 0 &&
		queryParams.Account == "" && queryParams.AccountLike == "" {
		WarnLogWithContext("GetQueryParams", "缺少必要查询参数")
	}

	queryParams.Where = queryParams.HasConditions()
	return queryParams
}

//func BQLQueryOne(ledgerConfig *Config, queryParams *QueryParams, queryResultPtr interface{}) error {
//	assertQueryResultIsPointer(queryResultPtr)
//	output, err := bqlRawQuery(ledgerConfig, "", queryParams, queryResultPtr)
//	if err != nil {
//		return err
//	}
//	err = parseResult(output, queryResultPtr, true)
//	if err != nil {
//		return err
//	}
//	return nil
//}

func BQLPrint(debugCtx *DebugContext, ledgerConfig *Config, transactionId string) (string, error) {
	// PRINT FROM id = 'xxx'
	output, err := queryByBQL(debugCtx, ledgerConfig, "PRINT FROM id = '"+transactionId+"'")
	if err != nil {
		return "", err
	}
	utf8, err := ConvertGBKToUTF8(output)
	if err != nil {
		return "", err
	}
	return utf8, nil
}

// BQLQueryList 执行BQL查询并将结果解析到指定的指针中
//
// 参数:
//
//	ledgerConfig: 账本配置信息
//	queryParams: 查询参数
//	queryResultPtr: 指向查询结果容器的指针，必须是指针类型
//
// 返回值:
//
//	error: 如果查询或解析过程中发生错误，返回错误信息
//
// 注意:
//  1. 函数内部包含详细的调试日志，可通过调试模式控制
//  2. queryResultPtr 必须是指针类型，否则会触发断言错误
//  3. 输出结果会被自动解析到queryResultPtr指向的变量中
func BQLQueryList(debugCtx *DebugContext, ledgerConfig *Config, queryParams *QueryParams, queryResultPtr interface{}) error {
	// 调试模式设置，默认为true（开启调试模式）

	// 调试信息：函数开始执行
	if debugCtx != nil {
		debugCtx.LogDebugF(ledgerConfig.Mail, "BQLQueryList",
			"函数开始执行\n----------------------\n输入查询参数: %+v\n----------------------\nqueryResultPtr 类型: %T",
			queryParams, queryResultPtr)
	} else {
		debugCtx.LogDebugF(ledgerConfig.Mail, "BQLQueryList",
			"函数开始执行\n----------------------\n输入查询参数: %+v\n----------------------\nqueryResultPtr 类型: %T",
			queryParams, queryResultPtr)
	}

	assertQueryResultIsPointer(queryResultPtr)

	// 调试信息：执行bqlRawQuery前
	debugCtx.LogDebugF(ledgerConfig.Mail, "BQLQuery",
		"正在执行 bqlRawQuery...")

	output, err := bqlRawQuery(debugCtx, ledgerConfig, "", queryParams, queryResultPtr)

	// 调试信息：bqlRawQuery执行结果
	if err != nil {
		LogError(ledgerConfig.Mail,
			fmt.Sprintf("bqlRawQuery执行失败: %v", err))
		return fmt.Errorf("BQL查询失败: %v", err)
	} else {
		// 限制输出长度，避免日志过大
		outputPreview := output
		if len(output) > 500 {
			outputPreview = output[:500] + "... (输出被截断)"
		}
		debugCtx.LogDebugF(ledgerConfig.Mail, "BQLQuery",
			"bqlRawQuery执行成功，输出预览: %s", outputPreview)
	}

	// 调试信息：执行parseResult前
	debugCtx.LogDebugF(ledgerConfig.Mail, "BQLQuery",
		"正在使用parseResult解析查询结果...")

	parseErr := parseResult(output, queryResultPtr, false)

	// 调试信息：parseResult执行结果
	if parseErr != nil {
		debugCtx.LogDebugF(ledgerConfig.Mail, "BQLParse",
			"parseResult解析失败: %v", parseErr)
	} else {
		resultValue := reflect.ValueOf(queryResultPtr).Elem()
		logMessage := fmt.Sprintf("parseResult解析成功\n结果类型: %s\n种类: %s",
			resultValue.Type().String(),
			resultValue.Kind().String())

		if resultValue.Kind() == reflect.Slice {
			logMessage += fmt.Sprintf("\n结果包含 %d 个项目", resultValue.Len())
		}

		debugCtx.LogDebugF(ledgerConfig.Mail, "BQLParse", logMessage)
	}

	return parseErr
}

func BQLQueryListByCustomSelect(debugCtx *DebugContext, ledgerConfig *Config, selectBql string, queryParams *QueryParams, queryResultPtr interface{}) error {
	const contextTag = "QueryListByCustomSelect"

	// 调试信息：函数开始执行
	if debugCtx != nil {
		debugCtx.LogDebugF(ledgerConfig.Mail, contextTag,
			"函数开始执行\n自定义 selectBql: %s\n输入查询参数: %+v\nqueryResultPtr 类型: %T",
			selectBql, queryParams, queryResultPtr)
	} else {
		LogBQLQueryDebug(ledgerConfig.Mail, contextTag,
			"函数开始执行\n自定义 selectBql: %s\n输入查询参数: %+v\nqueryResultPtr 类型: %T",
			selectBql, queryParams, queryResultPtr)
	}

	assertQueryResultIsPointer(queryResultPtr)

	// 调试信息：执行bqlRawQuery前
	LogBQLQueryDebug(ledgerConfig.Mail, contextTag, "正在执行 bqlRawQuery...")

	output, err := bqlRawQuery(debugCtx, ledgerConfig, selectBql, queryParams, queryResultPtr)

	// 调试信息：bqlRawQuery执行结果
	if err != nil {
		LogBQLQueryDebug(ledgerConfig.Mail, contextTag, "bqlRawQuery 执行失败: %v", err)
	} else {
		// 限制输出长度，避免日志过大
		outputPreview := output
		if len(output) > 500 {
			outputPreview = output[:500] + "... (输出被截断)"
		}
		LogBQLQueryDebug(ledgerConfig.Mail, contextTag,
			"bqlRawQuery 执行成功，输出预览: %s", outputPreview)
	}

	if err != nil {
		errorMsg := fmt.Sprintf("BQL:%s - 自定义 BQL 查询失败: %v", contextTag, err)
		LogError(ledgerConfig.Mail, errorMsg)
		return fmt.Errorf("自定义 BQL 查询失败: %v", err)
	}

	// 调试信息：执行parseResult前
	LogBQLQueryDebug(ledgerConfig.Mail, contextTag, "正在调用parseResult解析查询结果...")

	parseErr := parseResult(output, queryResultPtr, false)

	// 调试信息：parseResult执行结果
	if parseErr != nil {
		LogBQLQueryDebug(ledgerConfig.Mail, contextTag, "parseResult 解析失败: %v", parseErr)
	} else {
		resultValue := reflect.ValueOf(queryResultPtr).Elem()
		LogBQLQueryDebug(ledgerConfig.Mail, contextTag,
			"parseResult 解析成功\n结果类型: %s\n种类: %s",
			resultValue.Type().String(), resultValue.Kind().String())

		if resultValue.Kind() == reflect.Slice {
			LogBQLQueryDebug(ledgerConfig.Mail, contextTag,
				"结果包含 %d 个项目", resultValue.Len())
		}
	}

	return parseErr
}

func bqlRawQuery(debugCtx *DebugContext, ledgerConfig *Config, selectBql string, queryParamsPtr *QueryParams, queryResultPtr interface{}) (string, error) {
	if debugCtx != nil {
		debugCtx.LogDebugF(ledgerConfig.Mail, "BQLBuilder",
			"=== 开始构建BQL查询 ===\n输入参数: selectBql='%s'\nqueryParamsPtr=%+v",
			selectBql, queryParamsPtr)
	} else {
		debugCtx.LogDebugF(ledgerConfig.Mail, "BQLBuilder",
			"=== 开始构建BQL查询 ===\n输入参数: selectBql='%s'\nqueryParamsPtr=%+v",
			selectBql, queryParamsPtr)
	}

	var bql strings.Builder

	// 1. SELECT 部分
	if selectBql == "" {
		debugCtx.LogDebugF(ledgerConfig.Mail, "BQLBuilder", "自动生成SELECT字段...")
		bql.WriteString("SELECT ")

		queryResultPtrType := reflect.TypeOf(queryResultPtr)
		queryResultType := queryResultPtrType.Elem()
		if queryResultType.Kind() == reflect.Slice {
			queryResultType = queryResultType.Elem()
		}

		first := true
		for i := 0; i < queryResultType.NumField(); i++ {
			typeField := queryResultType.Field(i)
			if b := typeField.Tag.Get("bql"); b != "" {
				if !first {
					bql.WriteString(", ")
				}
				if strings.Contains(b, "distinct") {
					b = strings.ReplaceAll(b, "distinct", "")
					bql.WriteString("DISTINCT ")
				}
				bql.WriteString(b)
				// 保留反斜线作为分隔符
				bql.WriteString(", '\\'")
				first = false
			}
		}
		debugCtx.LogDebugF(ledgerConfig.Mail, "BQLBuilder", "生成的SELECT部分: %s", bql.String())
	} else {
		bql.WriteString(selectBql)
	}

	// 2. 记录WHERE条件构建
	if queryParamsPtr != nil {
		debugCtx.LogDebugF(ledgerConfig.Mail, "BQLBuilder",
			"正在解析和构建WHERE条件子句...")
		// queryParamsType := reflect.TypeOf(queryParamsPtr).Elem()
		// queryParamsValue := reflect.ValueOf(queryParamsPtr).Elem()

		hasConditions := false
		// firstCondition := true

		// 检查是否有实际条件
		if (queryParamsPtr.Year != 0 || queryParamsPtr.Month != 0) || // 时间条件
			(queryParamsPtr.AccountLike != "" || queryParamsPtr.Account != "" || queryParamsPtr.Tag != "") || // 账户/标签条件
			(queryParamsPtr.ID != "" || queryParamsPtr.Currency != "") { // ID/货币条件
			hasConditions = true
		}

		if hasConditions {
			// 账户匹配方式验证
			if queryParamsPtr.Account != "" && queryParamsPtr.AccountLike != "" {
				LogError(ledgerConfig.Mail,
					fmt.Sprintf("参数冲突: Account='%s' 和 AccountLike='%s' 不能同时指定",
						queryParamsPtr.Account, queryParamsPtr.AccountLike))
				return "", fmt.Errorf("不能同时指定Account和AccountLike参数")
			}

			if queryParamsPtr.StrictAccountMatch && queryParamsPtr.AccountLike != "" {
				LogError(ledgerConfig.Mail,
					fmt.Sprintf("参数冲突: StrictAccountMatch=%v 但指定了AccountLike='%s'",
						queryParamsPtr.StrictAccountMatch, queryParamsPtr.AccountLike))
				return "", fmt.Errorf("StrictAccountMatch模式下不能使用AccountLike")
			}

			if !queryParamsPtr.StrictAccountMatch && queryParamsPtr.Account != "" {
				LogError(ledgerConfig.Mail,
					fmt.Sprintf("参数冲突: StrictAccountMatch=%v 但指定了Account='%s' (应使用AccountLike)",
						queryParamsPtr.StrictAccountMatch, queryParamsPtr.Account))
				return "", fmt.Errorf("非StrictAccountMatch模式下必须指定AccountLike")
			}

			debugCtx.LogDebugF(ledgerConfig.Mail, "BQLBuilder",
				"构建前SQL: %s", bql.String())

			bql.WriteString(" WHERE ")
			firstCondition := true

			// 辅助函数添加条件
			addCondition := func(condition string) {
				if !firstCondition {
					bql.WriteString(" AND ")
				}
				bql.WriteString(condition)
				firstCondition = false
			}

			// 时间条件
			if queryParamsPtr.Year != 0 {
				addCondition(fmt.Sprintf("year = %d", queryParamsPtr.Year))
			}

			if queryParamsPtr.Month != 0 {
				addCondition(fmt.Sprintf("month = %d", queryParamsPtr.Month))
			}

			// 账户条件
			if queryParamsPtr.Account != "" {
				addCondition(fmt.Sprintf("account = '%s'", escapeSQLString(queryParamsPtr.Account)))
			} else if queryParamsPtr.AccountLike != "" {
				addCondition(fmt.Sprintf("account ~ '%s'", escapeSQLString(queryParamsPtr.AccountLike)))
			}

			// 其他条件
			if queryParamsPtr.Tag != "" {
				addCondition("tag in tags")
			}

			if queryParamsPtr.TagNotNull != "" {
				addCondition("tags IS NOT NULL")
			}

			if queryParamsPtr.ID != "" {
				addCondition(fmt.Sprintf("id = '%s'", escapeSQLString(queryParamsPtr.ID)))
			}

			debugCtx.LogDebugF(ledgerConfig.Mail, "BQLBuilder",
				"构建后SQL: %s", bql.String())
		}
	}

	// 在构建GROUP BY子句前添加验证
	if queryParamsPtr != nil && queryParamsPtr.GroupBy != "" {
		// 检查是否有聚合函数
		hasAggregate := strings.Contains(bql.String(), "sum(") ||
			strings.Contains(bql.String(), "count(") ||
			strings.Contains(bql.String(), "avg(") ||
			strings.Contains(bql.String(), "min(") ||
			strings.Contains(bql.String(), "max(")

		if hasAggregate {
			debugCtx.LogDebugF(ledgerConfig.Mail, "BQLBuilder-GroupBy", "BQL查询包含聚合函数，将添加GROUP BY子句")
			bql.WriteString(" GROUP BY ")
			bql.WriteString(queryParamsPtr.GroupBy)
		} else {
			debugCtx.LogDebugF(ledgerConfig.Mail, "BQLBuilder-GroupBy", "BQL查询不包含聚合函数，但是指定了GROUP BY子句，将仍然添加GROUP BY子句")
			bql.WriteString(" GROUP BY ")
			bql.WriteString(queryParamsPtr.GroupBy)
		}
	}

	// 构建 ORDER BY 子句
	if queryParamsPtr != nil && queryParamsPtr.OrderBy != "" {
		bql.WriteString(" ORDER BY ")
		orderBy := strings.ReplaceAll(queryParamsPtr.OrderBy, "'", "")
		orderBy = strings.ReplaceAll(orderBy, "\"", "")
		orderBy = strings.ReplaceAll(strings.ToLower(orderBy), "order by", "")
		orderBy = strings.TrimSpace(orderBy)
		bql.WriteString(orderBy)
	}

	// 构建 LIMIT 子句
	if queryParamsPtr != nil && queryParamsPtr.Limit > 0 {
		bql.WriteString(" LIMIT ")
		bql.WriteString(strconv.Itoa(queryParamsPtr.Limit))
	}

	finalQuery := bql.String()
	debugCtx.LogDebugF(ledgerConfig.Mail, "BQLGenerator",
		"=== 最终生成的BQL ===\n%s", finalQuery)

	if strings.Contains(finalQuery, "WHERE") && strings.Contains(finalQuery, "GROUP BY") {
		if strings.Index(finalQuery, "WHERE") > strings.Index(finalQuery, "GROUP BY") {
			return "", fmt.Errorf("SQL语法错误: WHERE子句必须在GROUP BY之前")
		}
	}

	// 验证生成的SQL
	if strings.Contains(finalQuery, "WHERE  AND") {
		LogError(ledgerConfig.Mail, "!!! 检测到非法WHERE条件: "+finalQuery)
		return "", fmt.Errorf("invalid WHERE clause")
	}
	debugCtx.LogDebugF(ledgerConfig.Mail, "BQLGenerator",
		"=== 最终生成的BQL语句 ===\n%s", finalQuery)
	return queryByBQL(debugCtx, ledgerConfig, finalQuery)
}

// 辅助函数：安全转义 SQL 字符串
func escapeSQLString(s string) string {
	// 只转义单引号，不处理反斜杠
	return strings.ReplaceAll(s, "'", "''")
}

// 修改 BeanReportAllPrices 函数
func BeanReportAllPrices(debugCtx *DebugContext, ledgerConfig *Config) []CommodityPrice {
	// 使用正确的 BQL 查询，不需要 FROM 子句
	output, err := queryByBQL(debugCtx, ledgerConfig,
		"SELECT date, 'price', currency, price WHERE price is not NULL")
	if err != nil {
		LogError(ledgerConfig.Mail, "Failed to query prices: "+err.Error())
		return nil
	}

	// 解析 CSV 输出
	reader := csv.NewReader(strings.NewReader(output))
	records, err := reader.ReadAll()
	if err != nil {
		LogError(ledgerConfig.Mail, "Failed to parse CSV: "+err.Error())
		return nil
	}

	// 将 [][]string 转换为 []string
	var lines []string
	if len(records) > 0 {
		// 跳过标题行
		for _, record := range records[1:] {
			lines = append(lines, strings.Join(record, " "))
		}
	}

	return newCommodityPriceListFromString(lines)
}

// 修改 parseResult 函数中的相关部分
func parseCsvResult(output string, queryResultPtr interface{}, selectOne bool) error {
	queryResultPtrType := reflect.TypeOf(queryResultPtr)
	queryResultType := queryResultPtrType.Elem()

	if queryResultType.Kind() == reflect.Slice {
		queryResultType = queryResultType.Elem()
	}

	// 使用 csv 解析器处理输出
	reader := csv.NewReader(strings.NewReader(output))
	records, err := reader.ReadAll()
	if err != nil {
		return err
	}

	// 跳过标题行
	if len(records) > 0 {
		records = records[1:]
	}

	if selectOne && len(records) > 0 {
		records = records[:1]
	}

	l := make([]map[string]interface{}, 0)
	for _, record := range records {
		if len(record) == 0 {
			continue
		}

		temp := make(map[string]interface{})
		for i, val := range record {
			if i >= queryResultType.NumField() {
				continue
			}

			field := queryResultType.Field(i)
			jsonName := field.Tag.Get("json")
			if jsonName == "" {
				jsonName = field.Name
			}

			val = strings.TrimSpace(val)
			if val == "" {
				continue
			}

			switch field.Type.Kind() {
			case reflect.Int, reflect.Int32:
				if i, err := strconv.Atoi(val); err == nil {
					temp[jsonName] = i
				}
			case reflect.String:
				temp[jsonName] = val
			case reflect.Float32, reflect.Float64:
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					temp[jsonName] = f
				}
			case reflect.Array, reflect.Slice:
				strArray := strings.Split(val, ",")
				notBlanks := make([]string, 0)
				for _, s := range strArray {
					if s = strings.TrimSpace(s); s != "" {
						notBlanks = append(notBlanks, s)
					}
				}
				if len(notBlanks) > 0 {
					temp[jsonName] = notBlanks
				}
			}
		}
		if len(temp) > 0 {
			l = append(l, temp)
		}
	}

	var jsonBytes []byte
	var jsonErr error // 修改变量名，避免重复声明
	if selectOne && len(l) > 0 {
		jsonBytes, jsonErr = json.Marshal(l[0])
	} else {
		jsonBytes, jsonErr = json.Marshal(l)
	}
	if jsonErr != nil {
		return jsonErr
	}
	err = json.Unmarshal(jsonBytes, queryResultPtr) // 使用外层的 err
	if err != nil {
		return err
	}
	return nil
}

// 原v2版本格式数据导入函数
// 主解析函数 - 智能识别格式
func parseResult(output string, queryResultPtr interface{}, selectOne bool) error {

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return nil // 空输出
	}

	// 检测格式类型
	isCustomFormat := false
	for _, line := range lines {
		if strings.Contains(line, "\\") || strings.Contains(line, "'") {
			isCustomFormat = true
			break
		}
		if IsDebugMode() {
			log.Printf("line: %s", line)
			log.Println("isCustomFormat:", isCustomFormat)
		}
	}

	if isCustomFormat {
		if IsDebugMode() {
			log.Println("使用自定义分隔符//解析逻辑")
		}
		// 使用自定义分隔符解析逻辑
		return parseCustomFormat(output, queryResultPtr, selectOne)
	} else {
		if IsDebugMode() {
			log.Println("使用表格格式解析逻辑")
		}
		// 使用表格格式解析逻辑
		return parseTableFormat(output, queryResultPtr, selectOne)
	}
}

// 解析自定义分隔符格式（原v2格式）
func parseCustomFormat(output string, queryResultPtr interface{}, selectOne bool) error {
	queryResultPtrType := reflect.TypeOf(queryResultPtr)
	queryResultType := queryResultPtrType.Elem()

	if queryResultType.Kind() == reflect.Slice {
		queryResultType = queryResultType.Elem()
	}

	lines := strings.Split(output, "\n")

	// 跳过标题行（如果有）
	var dataLines []string
	if len(lines) >= 3 && strings.Contains(lines[1], "-") {
		dataLines = lines[2:] // 跳过前2行（标题和分隔线）
	} else {
		dataLines = lines // 没有标准标题格式
	}

	l := make([]map[string]interface{}, 0)
	for _, line := range dataLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		values := strings.Split(line, "\\")

		// 安全地去除首尾空元素
		var cleanedValues []string
		for _, val := range values {
			trimmed := strings.TrimSpace(val)
			if trimmed != "" {
				cleanedValues = append(cleanedValues, trimmed)
			}
		}

		// 如果cleanedValues为空，跳过这行
		if len(cleanedValues) == 0 {
			continue
		}

		temp := make(map[string]interface{})
		for i, val := range cleanedValues {
			if i >= queryResultType.NumField() {
				break // 跳过多余的字段
			}

			field := queryResultType.Field(i)
			jsonName := field.Tag.Get("json")
			if jsonName == "" {
				jsonName = field.Name
			}

			val = strings.TrimSpace(val)
			if val == "" {
				continue
			}

			// 添加tags/payee字段的专门调试
			if jsonName == "tags" && IsDebugMode() {
				DebugLogWithContext("TAGS", "解析tags字段值: %s", val)
			}
			if jsonName == "payee" && IsDebugMode() {
				// DebugLogWithContext("PAYEE", "解析payee字段值: %s", val)
			}

			switch field.Type.Kind() {
			case reflect.Int, reflect.Int32:
				if intVal, err := strconv.Atoi(val); err != nil {
					handleError(err, fmt.Sprintf("解析整数值 '%s' 失败", val))
				} else {
					temp[jsonName] = intVal
				}
			case reflect.String, reflect.Struct:
				temp[jsonName] = val
			case reflect.Array, reflect.Slice:
				strArray := strings.Split(val, ",")
				notBlanks := make([]string, 0)
				for _, s := range strArray {
					if trimmed := strings.TrimSpace(s); trimmed != "" {
						notBlanks = append(notBlanks, trimmed)
					}
				}
				if len(notBlanks) > 0 {
					temp[jsonName] = notBlanks
				}
			default:
				if IsDebugMode() {
					panic(fmt.Sprintf("Unsupported field type: %s", field.Type.Kind()))
				}
			}
		}

		if len(temp) > 0 {
			l = append(l, temp)
		}
	}

	return marshalAndUnmarshal(l, queryResultPtr, selectOne)
}

// 解析表格格式（bean-query默认格式）
func parseTableFormat(output string, queryResultPtr interface{}, selectOne bool) error {
	queryResultPtrType := reflect.TypeOf(queryResultPtr)
	queryResultType := queryResultPtrType.Elem()

	if queryResultType.Kind() == reflect.Slice {
		queryResultType = queryResultType.Elem()
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")

	// 检测并跳过标题行和分隔线
	var dataLines []string
	if len(lines) >= 3 && strings.Contains(lines[1], "-") {
		dataLines = lines[2:] // 跳过前2行
	} else if len(lines) >= 2 && strings.Contains(lines[0], "|") {
		dataLines = lines[1:] // 跳过标题行
	} else {
		dataLines = lines // 没有标准标题
	}

	l := make([]map[string]interface{}, 0)
	for _, line := range dataLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 按 | 分割表格格式
		values := strings.Split(line, "|")
		var cleanedValues []string
		for _, val := range values {
			trimmed := strings.TrimSpace(val)
			if trimmed != "" {
				cleanedValues = append(cleanedValues, trimmed)
			}
		}

		if len(cleanedValues) == 0 {
			continue
		}

		temp := make(map[string]interface{})
		for i, val := range cleanedValues {
			if i >= queryResultType.NumField() {
				break
			}

			field := queryResultType.Field(i)
			jsonName := field.Tag.Get("json")
			if jsonName == "" {
				jsonName = field.Name
			}

			val = strings.TrimSpace(val)
			if val == "" {
				continue
			}

			switch field.Type.Kind() {
			case reflect.Int, reflect.Int32:
				if intVal, err := strconv.Atoi(val); err != nil {
					handleError(err, fmt.Sprintf("解析整数值 '%s' 失败", val))
				} else {
					temp[jsonName] = intVal
				}
			case reflect.String, reflect.Struct:
				temp[jsonName] = val
			case reflect.Float32, reflect.Float64:
				if floatVal, err := strconv.ParseFloat(val, 64); err != nil {
					handleError(err, fmt.Sprintf("解析浮点数值 '%s' 失败", val))
				} else {
					temp[jsonName] = floatVal
				}
			case reflect.Array, reflect.Slice:
				strArray := strings.Split(val, ",")
				notBlanks := make([]string, 0)
				for _, s := range strArray {
					if trimmed := strings.TrimSpace(s); trimmed != "" {
						notBlanks = append(notBlanks, trimmed)
					}
				}
				if len(notBlanks) > 0 {
					temp[jsonName] = notBlanks
				}
			default:
				if IsDebugMode() {
					panic(fmt.Sprintf("Unsupported field type: %s", field.Type.Kind()))
				}
			}
		}

		if len(temp) > 0 {
			l = append(l, temp)
		}
	}

	return marshalAndUnmarshal(l, queryResultPtr, selectOne)
}

// 通用的JSON序列化和反序列化
// 通用的JSON序列化和反序列化
func marshalAndUnmarshal(data []map[string]interface{}, queryResultPtr interface{}, selectOne bool) error {
	if len(data) == 0 {
		// 对于空结果，设置默认值
		if selectOne {
			if IsDebugMode() {
				panic("selectOne 查询没有找到结果")
			}
			return fmt.Errorf("selectOne 查询没有找到结果")
		}
		// 对于切片，返回空切片是合理的
	}

	var jsonBytes []byte
	var err error

	if selectOne {
		if len(data) == 0 {
			if IsDebugMode() {
				panic("selectOne 查询没有找到结果")
			}
			return fmt.Errorf("selectOne 查询没有找到结果")
		}
		jsonBytes, err = json.Marshal(data[0])
	} else {
		jsonBytes, err = json.Marshal(data)
	}

	if err != nil {
		handleError(err, "JSON 序列化失败")
		return err
	}

	err = json.Unmarshal(jsonBytes, queryResultPtr)
	if err != nil {
		handleError(err, "JSON 反序列化失败")
		return err
	}

	return nil
}

// 直接调用v3版本的bean-query命令，返回beancount专用表格格式
func queryByBQL(debugCtx *DebugContext, ledgerConfig *Config, bql string) (string, error) {
	beanFilePath := ledgerConfig.DataPath + "/index.bean"

	// 使用带请求ID的日志方法
	debugCtx.LogInfo(ledgerConfig.Mail, fmt.Sprintf("[BQLExecution] 查询语句: %s", bql))

	// 获取虚拟环境执行器
	executor := GetVenvExecutor()
	if executor == nil {
		// 降级方案：使用原来的逻辑但修复路径
		debugCtx.LogWarn(ledgerConfig.Mail, "VenvExecutor", "虚拟环境执行器未找到，使用降级方案")
		return queryByBQLFallback(debugCtx, beanFilePath, bql) // 也需要修改 fallback 函数
	}

	// 使用虚拟环境工具执行 bean-query
	debugCtx.LogDebugF(ledgerConfig.Mail, "BeanQuery", "开始执行 bean-query，文件: %s", beanFilePath)
	output, err := executor.BeanQueryStdout(beanFilePath, bql)
	if err != nil {
		errorMsg := fmt.Sprintf("bean-query执行失败 - 错误详情: %v", err)
		debugCtx.LogErrorDetailed(ledgerConfig.Mail, "BeanQuery", errorMsg)
		return "", fmt.Errorf("bean-query 执行失败: %v", err)
	}

	debugCtx.LogDebugF(ledgerConfig.Mail, "BeanQuery", "bean-query执行成功，输出长度: %d", len(output))
	return string(output), nil
}

// 降级方案，确保即使虚拟环境工具有问题也能工作
func queryByBQLFallback(debugCtx *DebugContext, beanFilePath, bql string) (string, error) {
	debugCtx.LogDebugF("System", "BQLFallback", "使用降级方案执行BQL查询，文件: %s", beanFilePath)

	var cmdPath string
	if runtime.GOOS == "windows" {
		cmdPath = ".env_beancount-v3/Scripts/bean-query.exe"
	} else {
		cmdPath = ".env_beancount-v3/bin/bean-query"
	}

	// 检查文件是否存在
	if _, err := os.Stat(cmdPath); os.IsNotExist(err) {
		return "", fmt.Errorf("bean-query 未找到: %s", cmdPath)
	}

	cmd := exec.Command(cmdPath, beanFilePath, bql)
	output, err := cmd.Output()
	if err != nil {
		debugCtx.LogErrorDetailed("System", "BQLFallback", "降级方案执行失败: %v", err)
		return "", err
	}

	debugCtx.LogDebugF("System", "BQLFallback", "降级方案执行成功，输出长度: %d", len(output))
	return string(output), nil
}

// 自定义 使用 Python 脚本执行 BQL 查询 返回csv数据格式
func queryByBQL_byPython(ledgerConfig *Config, bql string) (string, error) {
	beanFilePath := ledgerConfig.DataPath + "/index.bean"
	LogInfo(ledgerConfig.Mail, bql)

	// 创建临时 Python 脚本文件
	tempFile, err := os.CreateTemp("", "query_*.py")
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// 使用三重引号字符串来避免转义问题
	script := fmt.Sprintf(`
import sys
import os
import csv
import io
import traceback
from beancount import loader
from beanquery import query

def main():
	try:
	  # 使用原始路径
	  bean_file = r'%s'
	  print(f"正在加载文件: {bean_file}", file=sys.stderr)
	  
	  # 加载 beancount 文件
	  entries, errors, options_map = loader.load_file(bean_file)
	  
	  if errors:
		print("加载文件时出现错误:", file=sys.stderr)
		for error in errors:
			print(str(error), file=sys.stderr)
		sys.exit(1)
	  
	  # 执行查询
	  query_str = '''%s'''
	  print(f"正在执行查询: {query_str}", file=sys.stderr)
	  
	  result_types, result_rows = query.run_query(entries, options_map, query_str)
	  
	  # 将结果转换为 CSV 格式
	  output = io.StringIO()
	  writer = csv.writer(output)
	  
	  # 写入列名
	  writer.writerow([rt.name for rt in result_types])
	  
	  # 写入数据行
	  for row in result_rows:
		writer.writerow(row)
	  
	  # 输出结果
	  print(output.getvalue(), end='')
	  
	except Exception as e:
	  print("错误跟踪:", file=sys.stderr)
	  print(traceback.format_exc(), file=sys.stderr)
	  print(f"错误信息: {str(e)}", file=sys.stderr)
	  sys.exit(1)

if __name__ == "__main__":
	main()
`, beanFilePath, bql)

	_, err = tempFile.WriteString(script)
	if err != nil {
		return "", fmt.Errorf("写入脚本失败: %v", err)
	}
	tempFile.Close()

	// 尝试不同的 Python 命令
	pythonCommands := [][]string{
		{"python", tempFile.Name()},
		{"python3", tempFile.Name()},
		{"py", tempFile.Name()},
	}

	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("获取当前目录失败: %v", err)
	}

	// 首先尝试直接使用 Python 命令
	for _, pythonCmd := range pythonCommands {
		cmd := exec.Command(pythonCmd[0], pythonCmd[1])
		cmd.Dir = currentDir

		// 捕获标准输出和错误输出
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err == nil {
			return stdout.String(), nil
		}

		// 记录详细的错误信息
		logMessage := fmt.Sprintf("%s 执行失败: %v\n错误输出: %s",
			pythonCmd[0], err, stderr.String())
		LogError(ledgerConfig.Mail, logMessage)
	}

	// 尝试虚拟环境中的 Python
	venvPaths := []string{
		".env_beancount-v3",
		"../.env_beancount-v3",
		"../../.env_beancount-v3",
	}

	for _, venvPath := range venvPaths {
		if !filepath.IsAbs(venvPath) {
			venvPath = filepath.Join(currentDir, venvPath)
		}

		pythonPaths := []string{
			filepath.Join(venvPath, "Scripts", "python.exe"),
			filepath.Join(venvPath, "bin", "python"),
			filepath.Join(venvPath, "bin", "python3"),
		}

		for _, pythonPath := range pythonPaths {
			if _, err := os.Stat(pythonPath); err == nil {
				cmd := exec.Command(pythonPath, tempFile.Name())
				cmd.Dir = currentDir

				// 捕获标准输出和错误输出
				var stdout, stderr bytes.Buffer
				cmd.Stdout = &stdout
				cmd.Stderr = &stderr

				err := cmd.Run()
				if err == nil {
					return stdout.String(), nil
				}

				// 记录详细的错误信息
				LogInfo(ledgerConfig.Mail, fmt.Sprintf("%s 执行失败: %v", pythonPath, err))
				LogInfo(ledgerConfig.Mail, fmt.Sprintf("错误输出: %s", stderr.String()))
			}
		}
	}

	return "", fmt.Errorf("所有 Python 环境执行查询都失败")
}

func assertQueryResultIsPointer(queryResult interface{}) {
	k := reflect.TypeOf(queryResult).Kind()
	if k != reflect.Ptr {
		panic("QueryResult 类型必须是指针，当前是 " + k.String())
	}
}
