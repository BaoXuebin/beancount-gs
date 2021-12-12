package service

import (
	"bufio"
	"encoding/csv"
	"github.com/beancount-gs/script"
	"github.com/gin-gonic/gin"
	"golang.org/x/text/encoding/simplifiedchinese"
	"io"
	"strings"
)

func ImportAliPayCSV(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)

	file, _ := c.FormFile("file")
	f, _ := file.Open()
	reader := csv.NewReader(simplifiedchinese.GBK.NewDecoder().Reader(bufio.NewReader(f)))

	result := make([]Transaction, 0)

	currency := "CNY"
	currencySymbol := script.GetCommoditySymbol(currency)

	for {
		lines, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			script.LogError(ledgerConfig.Mail, err.Error())
		}
		if len(lines) > 11 {
			fields := strings.Fields(lines[2])
			status := strings.Trim(lines[15], " ")
			account := ""
			if status == "已收入" {
				account = "Income:"
			} else if status == "已支出" {
				account = "Expenses:"
			} else {
				continue
			}

			if len(fields) >= 2 {
				result = append(result, Transaction{
					Id:             strings.Trim(lines[0], " "),
					Date:           strings.Trim(fields[0], " "),
					Payee:          strings.Trim(lines[7], " "),
					Narration:      strings.Trim(lines[8], " "),
					Number:         strings.Trim(lines[9], " "),
					Account:        account,
					Currency:       currency,
					CurrencySymbol: currencySymbol,
				})
			}
		}
	}

	OK(c, result)
}
