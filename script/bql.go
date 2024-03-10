package script

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"reflect"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type QueryParams struct {
	From        bool   `bql:"From"`
	FromYear    int    `bql:"year ="`
	FromMonth   int    `bql:"month ="`
	Where       bool   `bql:"where"`
	Currency    string `bql:"currency ="`
	Year        int    `bql:"year ="`
	Month       int    `bql:"month ="`
	Tag         string `bql:"in tags"`
	Account     string `bql:"account ="`
	AccountLike string `bql:"account ~"`
	GroupBy     string `bql:"group by"`
	OrderBy     string `bql:"order by"`
	Limit       int    `bql:"limit"`
	Path        string
}

func GetQueryParams(c *gin.Context) QueryParams {
	var queryParams QueryParams
	var hasWhere bool
	if c.Query("year") != "" {
		val, err := strconv.Atoi(c.Query("year"))
		if err == nil {
			queryParams.Year = val
			hasWhere = true
		}
	}
	if c.Query("month") != "" {
		val, err := strconv.Atoi(c.Query("month"))
		if err == nil {
			queryParams.Month = val
			hasWhere = true
		}
	}
	if c.Query("tag") != "" {
		queryParams.Tag = c.Query("tag")
		hasWhere = true
	}
	if c.Query("type") != "" {
		queryParams.AccountLike = c.Query("type")
		hasWhere = true
	}
	if c.Query("account") != "" {
		queryParams.Account = c.Query("account")
		queryParams.Limit = 100
		hasWhere = true
	}
	queryParams.Where = hasWhere
	if c.Query("path") != "" {
		queryParams.Path = c.Query("path")
	}
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

func BQLQueryList(ledgerConfig *Config, queryParams *QueryParams, queryResultPtr interface{}) error {
	assertQueryResultIsPointer(queryResultPtr)
	output, err := bqlRawQuery(ledgerConfig, "", queryParams, queryResultPtr)
	if err != nil {
		return err
	}
	err = parseResult(output, queryResultPtr, false)
	if err != nil {
		return err
	}
	return nil
}

func BQLQueryListByCustomSelect(ledgerConfig *Config, selectBql string, queryParams *QueryParams, queryResultPtr interface{}) error {
	assertQueryResultIsPointer(queryResultPtr)
	output, err := bqlRawQuery(ledgerConfig, selectBql, queryParams, queryResultPtr)
	if err != nil {
		return err
	}
	err = parseResult(output, queryResultPtr, false)
	if err != nil {
		return err
	}
	return nil
}

func BeanReportAllPrices(ledgerConfig *Config) string {
	beanFilePath := GetLedgerPriceFilePath(ledgerConfig.DataPath)

	LogInfo(ledgerConfig.Mail, "bean-report "+beanFilePath+" all_prices")
	cmd := exec.Command("bean-report", beanFilePath, "all_prices")
	output, _ := cmd.Output()
	return string(output)
}

func bqlRawQuery(ledgerConfig *Config, selectBql string, queryParamsPtr *QueryParams, queryResultPtr interface{}) (string, error) {
	var bql string
	if selectBql == "" {
		bql = "select"
		queryResultPtrType := reflect.TypeOf(queryResultPtr)
		queryResultType := queryResultPtrType.Elem()

		if queryResultType.Kind() == reflect.Slice {
			queryResultType = queryResultType.Elem()
		}

		for i := 0; i < queryResultType.NumField(); i++ {
			typeField := queryResultType.Field(i)
			// 字段的 tag 不带 bql 的不进行拼接
			b := typeField.Tag.Get("bql")
			if b != "" {
				if strings.Contains(b, "distinct") {
					b = strings.ReplaceAll(b, "distinct", "")
					bql = fmt.Sprintf("%s distinct '\\', %s, ", bql, b)
				} else {
					bql = fmt.Sprintf("%s '\\', %s, ", bql, typeField.Tag.Get("bql"))
				}
			}
		}
		bql += " '\\'"
	} else {
		bql = selectBql
	}

	// 查询条件不为空时，拼接查询条件
	if queryParamsPtr != nil {
		queryParamsType := reflect.TypeOf(queryParamsPtr).Elem()
		queryParamsValue := reflect.ValueOf(queryParamsPtr).Elem()
		for i := 0; i < queryParamsType.NumField(); i++ {
			typeField := queryParamsType.Field(i)
			valueField := queryParamsValue.Field(i)
			switch valueField.Kind() {
			case reflect.String:
				val := valueField.String()
				if val != "" {
					if typeField.Name == "OrderBy" || typeField.Name == "GroupBy" {
						// 去除上一个条件后缀的 AND 关键字
						bql = strings.Trim(bql, " AND")
						bql = fmt.Sprintf("%s %s %s", bql, typeField.Tag.Get("bql"), val)
					} else if typeField.Name == "Tag" {
						bql = fmt.Sprintf("%s '%s' %s", bql, strings.Trim(val, " "), typeField.Tag.Get("bql"))
					} else {
						bql = fmt.Sprintf("%s %s '%s' AND", bql, typeField.Tag.Get("bql"), val)
					}
				}
			case reflect.Int:
				val := valueField.Int()
				if val != 0 {
					bql = fmt.Sprintf("%s %s %d AND", bql, typeField.Tag.Get("bql"), val)
				}
			case reflect.Bool:
				val := valueField.Bool()
				// where 前的 from 可能会带有 and
				if typeField.Name == "Where" {
					bql = strings.Trim(bql, " AND")
				}
				if val {
					bql = fmt.Sprintf("%s %s ", bql, typeField.Tag.Get("bql"))
				}
			}
		}
		bql = strings.TrimRight(bql, " AND")
	}
	return queryByBQL(ledgerConfig, bql)
}

func parseResult(output string, queryResultPtr interface{}, selectOne bool) error {
	queryResultPtrType := reflect.TypeOf(queryResultPtr)
	queryResultType := queryResultPtrType.Elem()

	if queryResultType.Kind() == reflect.Slice {
		queryResultType = queryResultType.Elem()
	}

	lines := strings.Split(output, "\n")[2:]
	if selectOne && len(lines) >= 3 {
		lines = lines[2:3]
	}

	l := make([]map[string]interface{}, 0)
	for _, line := range lines {
		if line != "" {
			values := strings.Split(line, "\\")
			// 去除 '\' 分割带来的空字符串
			values = values[1 : len(values)-1]
			temp := make(map[string]interface{})
			for i, val := range values {
				field := queryResultType.Field(i)
				jsonName := field.Tag.Get("json")
				if jsonName == "" {
					jsonName = field.Name
				}
				switch field.Type.Kind() {
				case reflect.Int, reflect.Int32:
					i, err := strconv.Atoi(strings.Trim(val, " "))
					if err != nil {
						panic(err)
					}
					temp[jsonName] = i
				// decimal
				case reflect.String, reflect.Struct:
					v := strings.Trim(val, " ")
					if v != "" {
						temp[jsonName] = v
					}
				case reflect.Array, reflect.Slice:
					// 去除空格
					strArray := strings.Split(val, ",")
					notBlanks := make([]string, 0)
					for _, s := range strArray {
						if strings.Trim(s, " ") != "" {
							notBlanks = append(notBlanks, s)
						}
					}
					temp[jsonName] = notBlanks
				default:
					panic("Unsupported field type")
				}
			}
			l = append(l, temp)
		}
	}

	var jsonBytes []byte
	var err error
	if selectOne {
		jsonBytes, err = json.Marshal(l[0])
	} else {
		jsonBytes, err = json.Marshal(l)
	}
	if err != nil {
		return err
	}
	err = json.Unmarshal(jsonBytes, queryResultPtr)
	if err != nil {
		return err
	}
	return nil
}

func queryByBQL(ledgerConfig *Config, bql string) (string, error) {
	beanFilePath := ledgerConfig.DataPath + "/index.bean"
	LogInfo(ledgerConfig.Mail, bql)
	cmd := exec.Command("bean-query", beanFilePath, bql)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func assertQueryResultIsPointer(queryResult interface{}) {
	k := reflect.TypeOf(queryResult).Kind()
	if k != reflect.Ptr {
		panic("QueryResult type must be pointer, it's " + k.String())
	}
}
