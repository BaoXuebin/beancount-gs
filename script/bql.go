package script

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"reflect"
	"strings"
)

type QueryParams struct {
	Year        int    `bql:"year ="`
	Month       int    `bql:"month ="`
	AccountType string `bql:"account ~"`
}

func BQLQueryOne(ledgerConfig *Config, queryParams *QueryParams, queryResultPtr interface{}) error {
	assertQueryResultIsPointer(queryResultPtr)
	output, err := bqlRawQuery(ledgerConfig, queryParams, queryResultPtr)
	if err != nil {
		return err
	}
	err = parseResult(output, queryResultPtr, true)
	if err != nil {
		return err
	}
	return nil
}

func BQLQueryList(ledgerConfig *Config, queryParams *QueryParams, queryResultPtr interface{}) error {
	assertQueryResultIsPointer(queryResultPtr)
	output, err := bqlRawQuery(ledgerConfig, queryParams, queryResultPtr)
	if err != nil {
		return err
	}
	err = parseResult(output, queryResultPtr, false)
	if err != nil {
		return err
	}
	return nil
}

func bqlRawQuery(ledgerConfig *Config, queryParamsPtr *QueryParams, queryResultPtr interface{}) (string, error) {
	bql := "SELECT"
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
	// 查询条件不为空时，拼接查询条件
	if queryParamsPtr != nil {
		bql += " '\\' WHERE"
		queryParamsType := reflect.TypeOf(queryParamsPtr).Elem()
		queryParamsValue := reflect.ValueOf(queryParamsPtr).Elem()
		for i := 0; i < queryParamsType.NumField(); i++ {
			typeField := queryParamsType.Field(i)
			valueField := queryParamsValue.Field(i)
			switch valueField.Kind() {
			case reflect.String:
				val := valueField.String()
				if val != "" {
					bql = fmt.Sprintf("%s %s '%s' AND", bql, typeField.Tag.Get("bql"), val)
				}
				break
			case reflect.Int:
				val := valueField.Int()
				if val != 0 {
					bql = fmt.Sprintf("%s %s %d AND", bql, typeField.Tag.Get("bql"), val)
				}
				break
			}
		}
		bql = strings.TrimRight(bql, " AND")
	} else {
		bql += " '\\'"
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
				case reflect.String:
					temp[jsonName] = strings.Trim(val, " ")
					break
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
					break
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
