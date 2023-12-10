package service

import (
	"encoding/json"
	"fmt"
	"github.com/beancount-gs/script"
	"github.com/gin-gonic/gin"
	"regexp"
	"sort"
	"strings"
	"time"
)

func QueryValidAccount(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	allAccounts := script.GetLedgerAccounts(ledgerConfig.Id)
	currencyMap := script.GetLedgerCurrencyMap(ledgerConfig.Id)
	result := make([]script.Account, 0)
	for _, account := range allAccounts {
		if account.EndDate == "" {
			// 货币实时汇率（忽略账本主货币）
			if account.Currency != ledgerConfig.OperatingCurrency && account.Currency != "" {
				// 从 map 中获取对应货币的实时汇率和符号
				currency, ok := currencyMap[account.Currency]
				if ok {
					account.CurrencySymbol = currency.Symbol
					account.Price = currency.Price
					account.PriceDate = currency.PriceDate
					account.IsAnotherCurrency = true
				}
			}
			result = append(result, account)
		}
	}
	OK(c, result)
}

type accountPosition struct {
	Account        string `json:"account"`
	MarketPosition string `json:"market_position"`
	Position       string `json:"position"`
}

func QueryAllAccount(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)

	bql := fmt.Sprintf("select '\\', account, '\\', sum(convert(value(position), '%s')) as market_position, '\\', sum(convert(value(position), currency)) as position, '\\'", ledgerConfig.OperatingCurrency)
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

	currencyMap := script.GetLedgerCurrencyMap(ledgerConfig.Id)
	accounts := script.GetLedgerAccounts(ledgerConfig.Id)
	result := make([]script.Account, 0, len(accounts))
	for i := 0; i < len(accounts); i++ {
		account := accounts[i]
		// 过滤已结束的账户
		if account.EndDate != "" {
			continue
		}
		// 货币实时汇率（忽略账本主货币）
		if account.Currency != ledgerConfig.OperatingCurrency && account.Currency != "" {
			// 从 map 中获取对应货币的实时汇率和符号
			currency, ok := currencyMap[account.Currency]
			if ok {
				account.CurrencySymbol = currency.Symbol
				account.Price = currency.Price
				account.PriceDate = currency.PriceDate
				account.IsAnotherCurrency = true
			}
		}
		key := account.Acc
		typ := script.GetAccountType(ledgerConfig.Id, key)
		account.Type = &typ
		marketPosition := strings.Trim(accountPositionMap[key].MarketPosition, " ")
		if marketPosition != "" {
			fields := strings.Fields(marketPosition)
			account.MarketNumber = fields[0]
			account.MarketCurrency = fields[1]
			account.MarketCurrencySymbol = script.GetCommoditySymbol(ledgerConfig.Id, fields[1])
		}
		position := strings.Trim(accountPositionMap[key].Position, " ")
		if position != "" {
			account.Positions = parseAccountPositions(ledgerConfig.Id, position)
		}
		result = append(result, account)
	}
	OK(c, result)
}

func parseAccountPositions(ledgerId string, input string) []script.AccountPosition {
	// 使用正则表达式提取数字、货币代码和金额
	re := regexp.MustCompile(`(-?\d+\.\d+) (\w+)`)
	matches := re.FindAllStringSubmatch(input, -1)

	var positions []script.AccountPosition

	// 遍历匹配项并创建 AccountPosition
	for _, match := range matches {
		number := match[1]
		currency := match[2]

		// 获取货币符号
		symbol := script.GetCommoditySymbol(ledgerId, currency)

		// 创建 AccountPosition
		position := script.AccountPosition{
			Number:         number,
			Currency:       currency,
			CurrencySymbol: symbol,
		}

		// 添加到切片中
		positions = append(positions, position)
	}

	return positions
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

type AddAccountForm struct {
	Date    string `form:"date" binding:"required"`
	Account string `form:"account" binding:"required"`
	// 账户计量单位可以为空
	Currency string `form:"currency"`
}

func AddAccount(c *gin.Context) {
	var accountForm AddAccountForm
	if err := c.ShouldBindJSON(&accountForm); err != nil {
		BadRequest(c, err.Error())
		return
	}
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	// 判断账户是否已存在
	accounts := script.GetLedgerAccounts(ledgerConfig.Id)
	for _, acc := range accounts {
		if acc.Acc == accountForm.Account {
			DuplicateAccount(c)
			return
		}
	}
	line := fmt.Sprintf("%s open %s %s", accountForm.Date, accountForm.Account, accountForm.Currency)
	if accountForm.Currency != "" && accountForm.Currency != ledgerConfig.OperatingCurrency {
		line += " \"FIFO\""
	}
	// 写入文件
	filePath := ledgerConfig.DataPath + "/account/" + strings.ToLower(script.GetAccountPrefix(accountForm.Account)) + ".bean"
	err := script.AppendFileInNewLine(filePath, line)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	// 更新缓存
	typ := script.GetAccountType(ledgerConfig.Id, accountForm.Account)
	account := script.Account{Acc: accountForm.Account, StartDate: accountForm.Date, Currency: accountForm.Currency, Type: &typ}
	accounts = append(accounts, account)
	script.UpdateLedgerAccounts(ledgerConfig.Id, accounts)
	OK(c, account)
}

type AddAccountTypeForm struct {
	Type string `form:"type" binding:"required"`
	Name string `form:"name" binding:"required"`
}

func AddAccountType(c *gin.Context) {
	var addAccountTypeForm AddAccountTypeForm
	if err := c.ShouldBindJSON(&addAccountTypeForm); err != nil {
		BadRequest(c, err.Error())
		return
	}
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	accountTypesMap := script.GetLedgerAccountTypes(ledgerConfig.Id)
	typ := addAccountTypeForm.Type
	accountTypesMap[typ] = addAccountTypeForm.Name
	// 更新文件
	pathFile := script.GetLedgerAccountTypeFilePath(ledgerConfig.DataPath)
	bytes, err := json.Marshal(accountTypesMap)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	err = script.WriteFile(pathFile, string(bytes))
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	// 更新缓存
	script.UpdateLedgerAccountTypes(ledgerConfig.Id, accountTypesMap)
	OK(c, script.AccountType{
		Key:  addAccountTypeForm.Type,
		Name: addAccountTypeForm.Name,
	})
}

type CloseAccountForm struct {
	Date    string `form:"date" binding:"required"`
	Account string `form:"account" binding:"required"`
}

func CloseAccount(c *gin.Context) {
	var accountForm CloseAccountForm
	if err := c.ShouldBindJSON(&accountForm); err != nil {
		BadRequest(c, err.Error())
		return
	}
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	line := fmt.Sprintf("%s close %s", accountForm.Date, accountForm.Account)
	// 写入文件
	filePath := ledgerConfig.DataPath + "/account/" + strings.ToLower(script.GetAccountPrefix(accountForm.Account)) + ".bean"
	err := script.AppendFileInNewLine(filePath, line)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	// 更新缓存
	accounts := script.GetLedgerAccounts(ledgerConfig.Id)
	for i := 0; i < len(accounts); i++ {
		if accounts[i].Acc == accountForm.Account {
			accounts[i].EndDate = accountForm.Date
		}
	}
	script.UpdateLedgerAccounts(ledgerConfig.Id, accounts)
	OK(c, script.Account{
		Acc: accountForm.Account, EndDate: accountForm.Date,
	})
}

func ChangeAccountIcon(c *gin.Context) {
	account := c.Query("account")
	if account == "" {
		BadRequest(c, "account is not blank")
		return
	}
	file, _ := c.FormFile("file")
	filePath := "./public/icons/" + script.GetAccountIconName(account) + ".png"
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		InternalError(c, err.Error())
		// 自己完成信息提示
		return
	}
	var result = make(map[string]string)
	result["filename"] = filePath
	OK(c, result)
}

type BalanceAccountForm struct {
	Date    string `form:"date" binding:"required" json:"date"`
	Account string `form:"account" binding:"required" json:"account"`
	Number  string `form:"number" binding:"required" json:"number"`
}

func BalanceAccount(c *gin.Context) {
	var accountForm BalanceAccountForm
	if err := c.ShouldBindJSON(&accountForm); err != nil {
		BadRequest(c, err.Error())
		return
	}
	ledgerConfig := script.GetLedgerConfigFromContext(c)

	// 获取当前账户信息
	var acc script.Account
	accounts := script.GetLedgerAccounts(ledgerConfig.Id)
	for _, account := range accounts {
		if account.Acc == accountForm.Account {
			acc = account
		}
	}

	today, err := time.Parse("2006-01-02", accountForm.Date)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	todayStr := today.Format("2006-01-02")
	yesterdayStr := today.AddDate(0, 0, -1).Format("2006-01-02")
	month := today.Format("2006-01")
	line := fmt.Sprintf("\r\n%s pad %s Equity:OpeningBalances", yesterdayStr, accountForm.Account)
	line += fmt.Sprintf("\r\n%s balance %s %s %s", todayStr, accountForm.Account, accountForm.Number, acc.Currency)

	// check month bean file exist
	err = CreateMonthBeanFileIfNotExist(ledgerConfig.DataPath, month)
	if err != nil {
		if c != nil {
			InternalError(c, err.Error())
		}
		return
	}

	// append padding content to month bean file
	err = script.AppendFileInNewLine(script.GetLedgerMonthFilePath(ledgerConfig.DataPath, month), line)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	result := make(map[string]string)
	result["account"] = accountForm.Account
	result["date"] = accountForm.Date
	result["marketNumber"] = accountForm.Number
	result["marketCurrency"] = ledgerConfig.OperatingCurrency
	result["marketCurrencySymbol"] = script.GetCommoditySymbol(ledgerConfig.Id, ledgerConfig.OperatingCurrency)
	OK(c, result)
}

func RefreshAccountCache(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	// 加载账户缓存
	err := script.LoadLedgerAccounts(ledgerConfig.Id)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	// 加载货币缓存
	err = script.LoadLedgerCurrencyMap(ledgerConfig)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	OK(c, nil)
}
