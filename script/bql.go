package script

import (
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

func BQLQuery(ledgerConfig *Config, queryParams QueryParams, queryResult interface{}) error {
	bql := "SELECT '\\', id, '\\', date, '\\', payee, '\\', narration, '\\', account, '\\', position, '\\', tags, '\\' WHERE"
	queryParamsType := reflect.TypeOf(queryParams)
	queryParamsValue := reflect.ValueOf(queryParams)
	for i := 0; i < queryParamsValue.NumField(); i++ {
		typeField := queryParamsType.Field(i)
		valueField := queryParamsValue.Field(i)
		switch valueField.Kind() {
		case reflect.String:
			val := valueField.String()
			if val != "" {
				bql = fmt.Sprintf("%s %s '%s' AND", bql, typeField.Tag.Get("bql"), val)
			}
		case reflect.Int:
			val := valueField.Int()
			if val != 0 {
				bql = fmt.Sprintf("%s %s %d AND", bql, typeField.Tag.Get("bql"), val)
			}
		}
	}
	bql = strings.TrimRight(bql, " AND")

	output, err := queryByBQL(ledgerConfig, bql)
	if err != nil {
		return err
	}

	fmt.Println(output)
	//panic("Unsupported result type")
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
