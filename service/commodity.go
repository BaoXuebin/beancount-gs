package service

import (
	"fmt"
	"github.com/beancount-gs/script"
	"github.com/gin-gonic/gin"
	"strings"
	"time"
)

type SyncCommodityPriceForm struct {
	Commodity string `form:"commodity" binding:"required" json:"commodity"`
	Date      string `form:"date" binding:"required" json:"date"`
	Price     string `form:"price" binding:"required" json:"price"`
}

func SyncCommodityPrice(c *gin.Context) {
	var syncCommodityPriceForm SyncCommodityPriceForm
	if err := c.ShouldBindJSON(&syncCommodityPriceForm); err != nil {
		BadRequest(c, err.Error())
		return
	}

	ledgerConfig := script.GetLedgerConfigFromContext(c)
	filePath := script.GetLedgerPriceFilePath(ledgerConfig.DataPath)
	line := fmt.Sprintf("%s price %s %s %s", syncCommodityPriceForm.Date, syncCommodityPriceForm.Commodity, syncCommodityPriceForm.Price, ledgerConfig.OperatingCurrency)
	// 写入文件
	err := script.AppendFileInNewLine(filePath, line)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	OK(c, syncCommodityPriceForm)
}

type CommodityCurrency struct {
	Name     string `json:"name"`
	Currency string `json:"currency"`
	Symbol   string `json:"symbol"`
	Current  bool   `json:"current"`
	ExRate   string `json:"exRate"`
	Date     string `json:"date"`
}

func QueryAllCurrencies(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)

	// 查询货币获取当前汇率
	output := script.BeanReportAllPrices(ledgerConfig)
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
	// statsPricesResultList 转为 map
	existCurrencyMap := make(map[string]StatsPricesResult)
	for _, statsPricesResult := range statsPricesResultList {
		existCurrencyMap[statsPricesResult.Commodity] = statsPricesResult
	}

	result := make([]CommodityCurrency, 0)
	currencies := script.GetLedgerCurrency(ledgerConfig.Id)
	for _, c := range currencies {
		current := c.Currency == ledgerConfig.OperatingCurrency
		var exRate string
		var date string
		if current {
			exRate = "1"
			date = time.Now().Format("2006-01-02")
		} else {
			value, exists := existCurrencyMap[c.Currency]
			if exists {
				exRate = value.Value
				date = value.Date
			}
		}
		result = append(result, CommodityCurrency{
			Name:     c.Name,
			Currency: c.Currency,
			Symbol:   c.Symbol,
			Current:  current,
			ExRate:   exRate,
			Date:     date,
		})
	}

	OK(c, result)
}
