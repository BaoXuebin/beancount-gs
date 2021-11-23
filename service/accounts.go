package service

import (
	"github.com/beancount-gs/script"
	"github.com/gin-gonic/gin"
	"sort"
)

func QueryValidAccount(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	OK(c, script.GetLedgerAccounts(ledgerConfig.Id))
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
