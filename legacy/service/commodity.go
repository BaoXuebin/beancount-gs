package service

import (
	"fmt"
	"github.com/beancount-gs/script"
	"github.com/gin-gonic/gin"
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

	// 刷新货币最新汇率值
	err = script.LoadLedgerCurrencyMap(ledgerConfig)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	OK(c, syncCommodityPriceForm)
}

func QueryAllCurrencies(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	// 查询货币获取当前汇率
	currency := script.RefreshLedgerCurrency(ledgerConfig)
	OK(c, currency)
}
