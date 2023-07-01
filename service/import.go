package service

import (
	"bufio"
	"encoding/csv"
	"errors"
	"github.com/beancount-gs/script"
	"github.com/gin-gonic/gin"
	"golang.org/x/text/encoding/simplifiedchinese"
	"io"
	"strconv"
	"strings"
	"time"
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
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			script.LogError(ledgerConfig.Mail, err.Error())
		}
		if len(lines) == 17 {
			transaction, err := importBrowserAliPayCSV(lines, currency, currencySymbol)
			if err != nil {
				script.LogInfo(ledgerConfig.Mail, err.Error())
				continue
			}
			if transaction.Account == "" {
				script.LogInfo(ledgerConfig.Mail, "Invalid transaction")
				continue
			}
			result = append(result, transaction)
		} else if len(lines) == 12 || len(lines) == 13 {
			transaction, err := importMobileAliPayCSV(lines, currency, currencySymbol)
			if err != nil {
				script.LogInfo(ledgerConfig.Mail, err.Error())
				continue
			}
			if transaction.Account == "" {
				script.LogInfo(ledgerConfig.Mail, "Invalid transaction")
				continue
			}
			result = append(result, transaction)
		}
	}

	OK(c, result)
}

func importBrowserAliPayCSV(lines []string, currency string, currencySymbol string) (Transaction, error) {
	dateColumn := strings.Fields(lines[2])
	status := strings.Trim(lines[15], " ")
	account := ""
	if status == "" {
		account = ""
	} else if status == "已收入" {
		account = "Income:"
	} else {
		account = "Expenses:"
	}

	if len(dateColumn) >= 2 {
		return Transaction{
			Id:             strings.Trim(lines[0], " "),
			Date:           strings.Trim(dateColumn[0], " "),
			Payee:          strings.Trim(lines[7], " "),
			Narration:      strings.Trim(lines[8], " "),
			Number:         strings.Trim(lines[9], " "),
			Account:        account,
			Currency:       currency,
			CurrencySymbol: currencySymbol,
		}, nil
	}
	return Transaction{}, errors.New("parse error")
}

func importMobileAliPayCSV(lines []string, currency string, currencySymbol string) (Transaction, error) {
	dateColumn := strings.Fields(lines[0])
	status := strings.Trim(lines[5], " ")
	account := ""
	if status == "" {
		account = ""
	} else if status == "支出" {
		account = "Expenses:"
	} else {
		account = "Income:"
	}

	if len(dateColumn) >= 2 {
		return Transaction{
			Id:             strings.Trim(lines[9], " "),
			Date:           strings.Trim(dateColumn[0], " "),
			Payee:          strings.Trim(lines[2], " "),
			Narration:      strings.Trim(lines[4], " "),
			Number:         strings.Trim(lines[6], " "),
			Account:        account,
			Currency:       currency,
			CurrencySymbol: currencySymbol,
		}, nil
	}
	return Transaction{}, errors.New("parse error")
}

func ImportWxPayCSV(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)

	file, _ := c.FormFile("file")
	f, _ := file.Open()
	reader := csv.NewReader(bufio.NewReader(f))

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
		if len(lines) > 8 {
			fields := strings.Fields(lines[0])
			status := strings.Trim(lines[4], " ")
			account := ""
			if status == "收入" {
				account = "Income:"
			} else if status == "支出" {
				account = "Expenses:"
			} else {
				continue
			}

			if len(fields) >= 2 {
				result = append(result, Transaction{
					Id:             strings.Trim(lines[8], " "),
					Date:           strings.Trim(fields[0], " "),
					Payee:          strings.Trim(lines[2], " "),
					Narration:      strings.Trim(lines[3], " "),
					Number:         strings.Trim(lines[5], "¥"),
					Account:        account,
					Currency:       currency,
					CurrencySymbol: currencySymbol,
				})
			}
		}
	}

	OK(c, result)
}

func ImportICBCCSV(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)

	file, _ := c.FormFile("file")
	f, _ := file.Open()
	reader := csv.NewReader(bufio.NewReader(f))

	result := make([]Transaction, 0)

	currency := "CNY"
	currencySymbol := script.GetCommoditySymbol(currency)

	id := 0
	for {
		lines, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			script.LogError(ledgerConfig.Mail, err.Error())
		}
		if len(lines) >= 13 && lines[0] != "交易日期" {
			incomeAmount := formatStr(lines[8])
			expensesAmount := formatStr(lines[9])
			account := ""
			number := ""
			switch {
			case incomeAmount != "":
				account = "Income:"
				number = strings.ReplaceAll(incomeAmount, ",", "")
			case expensesAmount != "":
				account = "Expenses:"
				number = strings.ReplaceAll(expensesAmount, ",", "")
			default:
				continue
			}

			id++
			result = append(result, Transaction{
				Id:             strconv.Itoa(id),
				Date:           formatStr(lines[0]),
				Payee:          formatStr(lines[12]),
				Narration:      formatStr(lines[1]),
				Number:         number,
				Account:        account,
				Currency:       currency,
				CurrencySymbol: currencySymbol,
			})
		}
	}

	OK(c, result)
}

func ImportABCCSV(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)

	file, _ := c.FormFile("file")
	f, _ := file.Open()
	reader := csv.NewReader(bufio.NewReader(f))

	result := make([]Transaction, 0)

	currency := "CNY"
	currencySymbol := script.GetCommoditySymbol(currency)

	id := 0
	for {
		lines, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			script.LogError(ledgerConfig.Mail, err.Error())
		}
		if len(lines) >= 11 && lines[0] != "交易日期" {
			amount := formatStr(lines[2])
			account := ""
			number := ""
			switch {
			case strings.HasPrefix(amount, "+"):
				account = "Income:"
				number = strings.ReplaceAll(amount, "+", "")
			case strings.HasPrefix(amount, "-"):
				account = "Expenses:"
				number = strings.ReplaceAll(amount, "-", "")
			default:
				continue
			}

			id++
			date, err := time.Parse("20060102", formatStr(lines[0]))
			if err != nil {
				continue
			}
			result = append(result, Transaction{
				Id:             strconv.Itoa(id),
				Date:           date.Format("2006-01-02"),
				Payee:          formatStr(lines[10]),
				Narration:      formatStr(lines[9]),
				Number:         number,
				Account:        account,
				Currency:       currency,
				CurrencySymbol: currencySymbol,
			})
		}
	}

	OK(c, result)
}

func formatStr(str string) string {
	str = strings.Trim(str, "\t")
	return strings.Trim(str, " ")
}
