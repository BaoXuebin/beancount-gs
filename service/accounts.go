package service

import (
	"fmt"
	"github.com/beancount-gs/script"
	"github.com/gin-gonic/gin"
	"sort"
	"strings"
)

func QueryValidAccount(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	OK(c, script.GetLedgerAccounts(ledgerConfig.Id))
}

type accountPosition struct {
	Account  string `json:"account"`
	Position string `json:"position"`
}

func QueryAllAccount(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)

	bql := fmt.Sprintf("select '\\', account, '\\', sum(convert(value(position), '%s')), '\\'", ledgerConfig.OperatingCurrency)
	accountPositions := make([]accountPosition, 0)
	err := script.BQLQueryListByCustomSelect(ledgerConfig, bql, nil, &accountPositions)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	// 将查询结果放入 map 中方便查询账户金额
	accountPositionMap := make(map[string]accountPosition)
	for _, ap := range accountPositions {
		accountPositionMap[ap.Account] = ap
	}

	accounts := script.GetLedgerAccounts(ledgerConfig.Id)
	result := make([]script.Account, 0, len(accounts))
	for i := 0; i < len(accounts); i++ {
		// 过滤已结束的账户
		if accounts[i].EndDate != "" {
			continue
		}
		key := accounts[i].Acc
		typ := script.GetAccountType(ledgerConfig.Id, key)
		accounts[i].Type = &typ
		position := strings.Trim(accountPositionMap[key].Position, " ")
		if position != "" {
			fields := strings.Fields(position)
			accounts[i].PriceAmount = fields[0]
			accounts[i].PriceCommodity = fields[1]
			accounts[i].PriceCommoditySymbol = script.GetCommoditySymbol(fields[1])
		}
		result = append(result, accounts[i])
	}
	OK(c, result)
}

func QueryAccountType(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	accountTypes := script.GetLedgerAccountTypes(ledgerConfig.Id)

	result := make([]script.AccountType, 0)
	for k, v := range accountTypes {
		result = append(result, script.AccountType{Key: k, Name: v})
	}
	sort.Sort(script.AccountTypeSort(result))
	OK(c, result)
}
