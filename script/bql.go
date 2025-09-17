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

	/* "regexp"*/
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// 调试模式下的错误处理
// 修改 handleError 函数中的提示
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
	From        bool   `bql:"From"`             // 是否包含 From 子句
	FromYear    int    `bql:"year ="`           // From 子句中的年份条件
	FromMonth   int    `bql:"month ="`          // From 子句中的月份条件
	Where       bool   `bql:"where"`            // 是否包含 Where 子句
	ID          string `bql:"id ="`             // ID 等于条件
	IDList      string `bql:"id in"`            // ID 列表条件
	Currency    string `bql:"currency ="`       // 货币等于条件
	Year        int    `bql:"year ="`           // 年份等于条件
	Month       int    `bql:"month ="`          // 月份等于条件
	Tag         string `bql:"in tags"`          // 用于 tag in tags 条件
	TagNotNull  string `bql:"tags IS NOT NULL"` // 标签非空条件
	Account     string `bql:"account ="`        // 账户等于条件
	AccountLike string `bql:"account ~"`        // 账户模糊匹配条件
	GroupBy     string `bql:"group by"`         // 分组条件
	OrderBy     string `bql:"order by"`         // 排序条件
	Limit       int    `bql:"limit"`            // 限制结果数量
	Path        string // 查询路径
}

func (queryParams *QueryParams) HasConditions() bool {
	return queryParams.Year != 0 || queryParams.Month != 0 ||
		queryParams.AccountLike != "" || queryParams.Tag != "" ||
		queryParams.ID != "" || queryParams.Currency != ""
}

/*
GetQueryParams 从 gin.Context 中解析查询参数并填充到 QueryParams 结构体中。
根据请求中的参数值，设置相应的查询条件，并返回填充好的 QueryParams 结构体。
*/
func GetQueryParams(c *gin.Context) QueryParams {
	var queryParams QueryParams

	// 数值型条件
	if val, err := strconv.Atoi(c.Query("year")); err == nil {
		queryParams.Year = val
	}

	if val, err := strconv.Atoi(c.Query("month")); err == nil {
		queryParams.Month = val
	}

	// 字符串条件
	if tag := c.Query("tag"); tag != "" {
		queryParams.Tag = tag
	}

	if accType := c.Query("type"); accType != "" {
		queryParams.AccountLike = accType
	}

	if account := c.Query("account"); account != "" {
		queryParams.Account = account
		queryParams.Limit = 100
	}

	if id := c.Query("id"); id != "" {
		queryParams.ID = id
	}

	queryParams.Where = queryParams.HasConditions() // 使用HasConditions方法

	if path := c.Query("path"); path != "" {
		queryParams.Path = path
	}

	// // 设置默认的 OrderBy 和 Limit（如果需要）
	// if c.Query("orderBy") != "" {
	//     queryParams.OrderBy = c.Query("orderBy")
	// } else {
	//     queryParams.OrderBy = "date desc"  // 默认排序
	// }

	// if c.Query("limit") != "" {
	//     if limit, err := strconv.Atoi(c.Query("limit")); err == nil {
	//       queryParams.Limit = limit
	//     }
	// } else if queryParams.Limit == 0 {
	//     queryParams.Limit = 100  // 默认限制
	// }

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

func BQLPrint(ledgerConfig *Config, transactionId string) (string, error) {
	// PRINT FROM id = 'xxx'
	output, err := queryByBQL(ledgerConfig, "PRINT FROM id = '"+transactionId+"'")
	if err != nil {
		return "", err
	}
	utf8, err := ConvertGBKToUTF8(output)
	if err != nil {
		return "", err
	}
	return utf8, nil
}

func BQLQueryList(ledgerConfig *Config, queryParams *QueryParams, queryResultPtr interface{}) error {
	// 调试模式设置，默认为true（开启调试模式）

	// 调试信息：函数开始执行
	if IsDebugMode() {
		LogInfo(ledgerConfig.Mail, "DEBUG: BQLQueryList 函数开始执行")
		LogInfo(ledgerConfig.Mail, fmt.Sprintf("DEBUG: 输入查询参数: %+v", queryParams))
		LogInfo(ledgerConfig.Mail, fmt.Sprintf("queryResultPtr 类型: %T", queryResultPtr))
	}

	assertQueryResultIsPointer(queryResultPtr)

	// 调试信息：执行bqlRawQuery前
	if IsDebugMode() {
		LogInfo(ledgerConfig.Mail, "DEBUG: 正在执行 bqlRawQuery...")
	}

	output, err := bqlRawQuery(ledgerConfig, "", queryParams, queryResultPtr)

	// 调试信息：bqlRawQuery执行结果
	if IsDebugMode() {
		if err != nil {
			LogInfo(ledgerConfig.Mail, fmt.Sprintf("DEBUG: bqlRawQuery 执行失败: %v", err))
		} else {
			// 限制输出长度，避免日志过大
			outputPreview := output
			if len(output) > 500 {
				outputPreview = output[:500] + "... (输出被截断)"
			}
			LogInfo(ledgerConfig.Mail, fmt.Sprintf("DEBUG: bqlRawQuery 执行成功，输出预览: %s", outputPreview))
		}
	}

	if err != nil {
		return fmt.Errorf("BQL 查询失败: %v", err)
	}

	// 调试信息：执行parseResult前
	if IsDebugMode() {
		LogInfo(ledgerConfig.Mail, "DEBUG: 正在使用parseResult解析查询结果...")
	}

	parseErr := parseResult(output, queryResultPtr, false)

	// 调试信息：parseResult执行结果
	if IsDebugMode() {
		if parseErr != nil {
			LogInfo(ledgerConfig.Mail, fmt.Sprintf("DEBUG: parseResult 解析失败: %v", parseErr))
		} else {
			LogInfo(ledgerConfig.Mail, "DEBUG: parseResult 解析成功")
			// 尝试打印结果的简要信息
			resultValue := reflect.ValueOf(queryResultPtr).Elem()
			LogInfo(ledgerConfig.Mail, fmt.Sprintf("DEBUG: 结果类型: %s, 种类: %s",
				resultValue.Type().String(), resultValue.Kind().String()))

			if resultValue.Kind() == reflect.Slice {
				LogInfo(ledgerConfig.Mail, fmt.Sprintf("DEBUG: 结果包含 %d 个项目", resultValue.Len()))
			}
		}
	}

	return parseErr
}

func BQLQueryListByCustomSelect(ledgerConfig *Config, selectBql string, queryParams *QueryParams, queryResultPtr interface{}) error {
	// 调试模式设置，默认为true（开启调试模式）

	// 调试信息：函数开始执行
	if IsDebugMode() {
		LogInfo(ledgerConfig.Mail, "DEBUG: BQLQueryListByCustomSelect 函数开始执行")
		LogInfo(ledgerConfig.Mail, fmt.Sprintf("DEBUG: 自定义 selectBql: %s", selectBql))
		LogInfo(ledgerConfig.Mail, fmt.Sprintf("DEBUG: 输入查询参数: %+v", queryParams))
		LogInfo(ledgerConfig.Mail, fmt.Sprintf("queryResultPtr 类型: %T", queryResultPtr))
	}

	assertQueryResultIsPointer(queryResultPtr)

	// 调试信息：执行bqlRawQuery前
	if IsDebugMode() {
		LogInfo(ledgerConfig.Mail, "DEBUG: 正在执行 bqlRawQuery...")
	}

	output, err := bqlRawQuery(ledgerConfig, selectBql, queryParams, queryResultPtr)

	// 调试信息：bqlRawQuery执行结果
	if IsDebugMode() {
		if err != nil {
			LogInfo(ledgerConfig.Mail, fmt.Sprintf("DEBUG: bqlRawQuery 执行失败: %v", err))
		} else {
			// 限制输出长度，避免日志过大
			outputPreview := output
			if len(output) > 500 {
				outputPreview = output[:500] + "... (输出被截断)"
			}
			LogInfo(ledgerConfig.Mail, fmt.Sprintf("DEBUG: bqlRawQuery 执行成功，输出预览: %s", outputPreview))
		}
	}

	if err != nil {
		return fmt.Errorf("自定义 BQL 查询失败: %v", err)
	}

	// 调试信息：执行parseResult前
	if IsDebugMode() {
		LogInfo(ledgerConfig.Mail, "DEBUG: 正在调用parseResult解析查询结果...")
	}

	parseErr := parseResult(output, queryResultPtr, false)

	// 调试信息：parseResult执行结果
	if IsDebugMode() {
		if parseErr != nil {
			LogInfo(ledgerConfig.Mail, fmt.Sprintf("DEBUG: parseResult 解析失败: %v", parseErr))
		} else {
			LogInfo(ledgerConfig.Mail, "DEBUG: parseResult 解析成功")
			// 尝试打印结果的简要信息
			resultValue := reflect.ValueOf(queryResultPtr).Elem()
			LogInfo(ledgerConfig.Mail, fmt.Sprintf("DEBUG: 结果类型: %s, 种类: %s",
				resultValue.Type().String(), resultValue.Kind().String()))

			if resultValue.Kind() == reflect.Slice {
				LogInfo(ledgerConfig.Mail, fmt.Sprintf("DEBUG: 结果包含 %d 个项目", resultValue.Len()))
			}
		}
	}

	return parseErr
}

func bqlRawQuery(ledgerConfig *Config, selectBql string, queryParamsPtr *QueryParams, queryResultPtr interface{}) (string, error) {
	if IsDebugMode() {
		LogInfo(ledgerConfig.Mail, "[DEBUG]=== 开始构建BQL查询 ===")
		LogInfo(ledgerConfig.Mail, fmt.Sprintf("[DEBUG]输入参数: selectBql='%s', queryParamsPtr=%+v", selectBql, queryParamsPtr))
	}

	var bql strings.Builder

	// 1. 记录SELECT部分构建
	if selectBql == "" {
		if IsDebugMode() {
			LogInfo(ledgerConfig.Mail, "[DEBUG]自动生成SELECT字段...")
		}
		bql.WriteString("SELECT ")
		queryResultPtrType := reflect.TypeOf(queryResultPtr)
		queryResultType := queryResultPtrType.Elem()

		if queryResultType.Kind() == reflect.Slice {
			queryResultType = queryResultType.Elem()
		}

		first := true
		for i := 0; i < queryResultType.NumField(); i++ {
			typeField := queryResultType.Field(i)
			b := typeField.Tag.Get("bql")
			if b != "" {
				if !first {
					bql.WriteString(", ")
				}
				if strings.Contains(b, "distinct") {
					b = strings.ReplaceAll(b, "distinct", "")
					bql.WriteString("DISTINCT ")
				}
				bql.WriteString(b)
				bql.WriteString(", '\\'")
				first = false
			}
		}
		if IsDebugMode() {
			LogInfo(ledgerConfig.Mail, fmt.Sprintf("[DEBUG]生成的SELECT部分: %s", bql.String()))
		}
	} else {
		bql.WriteString(selectBql)
	}

	// 2. 记录WHERE条件构建
	if queryParamsPtr != nil {
		if IsDebugMode() {
			LogInfo(ledgerConfig.Mail, "[DEBUG]开始处理WHERE条件...")
		}
		// queryParamsType := reflect.TypeOf(queryParamsPtr).Elem()
		// queryParamsValue := reflect.ValueOf(queryParamsPtr).Elem()

		hasConditions := false
		firstCondition := true

		// 检查是否有实际条件
		if (queryParamsPtr.Year != 0 || queryParamsPtr.Month != 0) || // 时间条件
			(queryParamsPtr.AccountLike != "" || queryParamsPtr.Tag != "") || // 账户/标签条件
			(queryParamsPtr.ID != "" || queryParamsPtr.Currency != "") { // ID/货币条件
			hasConditions = true
		}

		if hasConditions {
			if IsDebugMode() {
				LogInfo(ledgerConfig.Mail, "[DEBUG]构建前SQL: "+bql.String())
			}

			bql.WriteString(" WHERE ")

			if queryParamsPtr.Year != 0 {
				bql.WriteString("year = ")
				bql.WriteString(strconv.Itoa(queryParamsPtr.Year))
				firstCondition = false
			}

			if queryParamsPtr.Month != 0 {
				if !firstCondition {
					bql.WriteString(" AND ")
				}
				bql.WriteString("month = ")
				bql.WriteString(strconv.Itoa(queryParamsPtr.Month))
				firstCondition = false
			}

			if queryParamsPtr.AccountLike != "" {
				if !firstCondition {
					bql.WriteString(" AND ")
				}
				bql.WriteString("account ~ '")
				bql.WriteString(escapeSQLString(queryParamsPtr.AccountLike))
				bql.WriteString("'")
				firstCondition = false
			}

			if queryParamsPtr.Tag != "" {
				if !firstCondition {
					bql.WriteString(" AND ")
				}
				bql.WriteString("tag = '")
				bql.WriteString(escapeSQLString(queryParamsPtr.Tag))
				bql.WriteString("'")
				firstCondition = false
			}

			if queryParamsPtr.ID != "" {
				if !firstCondition {
					bql.WriteString(" AND ")
				}
				bql.WriteString("id = '")
				bql.WriteString(escapeSQLString(queryParamsPtr.ID))
				bql.WriteString("'")
			}
			if IsDebugMode() {
				LogInfo(ledgerConfig.Mail, fmt.Sprintf("[DEBUG]构建后SQL: %s", bql.String()))
			}
		}
	}

	// 构建 GROUP BY 子句
	if queryParamsPtr != nil && queryParamsPtr.GroupBy != "" {
		bql.WriteString(" GROUP BY ")
		bql.WriteString(queryParamsPtr.GroupBy)
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
	if IsDebugMode() {
		LogInfo(ledgerConfig.Mail, "[DEBUG]Generated BQL: "+finalQuery)
	}

	// 验证生成的SQL
	if strings.Contains(finalQuery, "WHERE  AND") {
		LogError(ledgerConfig.Mail, "!!! 检测到非法WHERE条件: "+finalQuery)
		return "", fmt.Errorf("invalid WHERE clause")
	}
	LogInfo(ledgerConfig.Mail, fmt.Sprintf("[DEBUG]最终BQL语句: %s", finalQuery))
	return queryByBQL(ledgerConfig, finalQuery)
}

// 辅助函数：安全转义 SQL 字符串
func escapeSQLString(s string) string {
	// 只转义单引号，不处理反斜杠
	return strings.ReplaceAll(s, "'", "''")
}

// 修改 BeanReportAllPrices 函数
func BeanReportAllPrices(ledgerConfig *Config) []CommodityPrice {
	// 使用正确的 BQL 查询，不需要 FROM 子句
	output, err := queryByBQL(ledgerConfig,
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
func queryByBQL(ledgerConfig *Config, bql string) (string, error) {
	beanFilePath := ledgerConfig.DataPath + "/index.bean"
	LogInfo(ledgerConfig.Mail, bql)
	cmd := exec.Command(".env_beancount-v3/bin/bean-query", beanFilePath, bql)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
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
		LogInfo(ledgerConfig.Mail, fmt.Sprintf("%s 执行失败: %v", pythonCmd[0], err))
		LogInfo(ledgerConfig.Mail, fmt.Sprintf("错误输出: %s", stderr.String()))
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
