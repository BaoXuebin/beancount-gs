package script

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var serverSecret string
var serverConfig Config
var serverCurrencies []LedgerCurrency
var ledgerConfigMap map[string]Config
var ledgerAccountsMap map[string][]Account
var ledgerAccountTypesMap map[string]map[string]string
var ledgerCurrencyMap map[string][]LedgerCurrency
var whiteList []string

type Config struct {
	Id                string `json:"id,omitempty"`
	Mail              string `json:"mail,omitempty"`
	Title             string `json:"title,omitempty"`
	DataPath          string `json:"dataPath,omitempty"`
	OperatingCurrency string `json:"operatingCurrency"`
	StartDate         string `json:"startDate"`
	IsBak             bool   `json:"isBak"`
	OpeningBalances   string `json:"openingBalances"`
	CreateDate        string `json:"createDate,omitempty"`
}

type Account struct {
	Acc                  string            `json:"account"`
	StartDate            string            `json:"startDate"`
	Currency             string            `json:"currency,omitempty"`          // 货币
	CurrencySymbol       string            `json:"currencySymbol,omitempty"`    // 货币符号
	Price                string            `json:"price,omitempty"`             // 汇率
	PriceDate            string            `json:"priceDate,omitempty"`         // 汇率日期
	IsAnotherCurrency    bool              `json:"isAnotherCurrency,omitempty"` // 其他币种标识
	IsCurrent            bool              `json:"isCurrent,omitempty"`
	Positions            []AccountPosition `json:"positions,omitempty"`
	MarketNumber         string            `json:"marketNumber,omitempty"`
	MarketCurrency       string            `json:"marketCurrency,omitempty"`
	MarketCurrencySymbol string            `json:"marketCurrencySymbol,omitempty"`
	EndDate              string            `json:"endDate,omitempty"`
	Type                 *AccountType      `json:"type,omitempty"`
}

type AccountPosition struct {
	Number         string `json:"number,omitempty"`
	Currency       string `json:"currency,omitempty"`
	CurrencySymbol string `json:"currencySymbol,omitempty"`
}

type AccountType struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type LedgerCurrency struct {
	Name      string `json:"name"`
	Currency  string `json:"currency"`
	Symbol    string `json:"symbol"`
	Current   bool   `json:"current,omitempty"`
	Price     string `json:"price,omitempty"`
	PriceDate string `json:"priceDate,omitempty"`
}

func GetServerConfig() Config {
	return serverConfig
}

func LoadServerConfig() error {
	filePath := GetServerConfigFilePath()
	if !FileIfExist(filePath) {
		serverConfig = Config{
			OpeningBalances:   "Equity:OpeningBalances",
			OperatingCurrency: "CNY",
			StartDate:         "1970-01-01",
			IsBak:             true,
		}
		return nil
	}
	fileContent, err := ReadFile(filePath)
	if err != nil {
		return err
	}
	err = json.Unmarshal(fileContent, &serverConfig)
	if err != nil {
		LogSystemError("Failed unmarshall config file (" + filePath + ")")
		return err
	}
	LogSystemInfo("Success load config file (" + filePath + ")")
	// load white list
	whiteListFilePath := GetServerWhiteListFilePath()
	if FileIfExist(whiteListFilePath) {
		fileContent, err = ReadFile(whiteListFilePath)
		if err != nil {
			return err
		}
		err = json.Unmarshal(fileContent, &whiteList)
		if err != nil {
			LogSystemError("Failed unmarshal whitelist file (" + whiteListFilePath + ")")
			return err
		}
	} else {
		err = CreateFile(whiteListFilePath)
		if err != nil {
			return err
		}
		err = WriteFile(whiteListFilePath, "[]")
		if err != nil {
			return err
		}
		whiteList = make([]string, 0)
	}
	LogSystemInfo("Success load whitelist file (" + whiteListFilePath + ")")
	return nil
}

func UpdateServerConfig(config Config) error {
	bytes, err := json.Marshal(config)
	if err != nil {
		return err
	}
	err = WriteFile(GetServerConfigFilePath(), string(bytes))
	if err != nil {
		return err
	}
	serverConfig = config
	return nil
}

func GetLedgerConfigMap() map[string]Config {
	return ledgerConfigMap
}

func GetLedgerConfig(ledgerId string) *Config {
	for k, v := range ledgerConfigMap {
		if k == ledgerId {
			return &v
		}
	}
	return nil
}

func GetLedgerConfigByMail(mail string) *Config {
	for _, v := range ledgerConfigMap {
		if v.Mail == mail {
			return &v
		}
	}
	return nil
}

func GetLedgerConfigFromContext(c *gin.Context) *Config {
	ledgerConfig, _ := c.Get("LedgerConfig")
	t, _ := ledgerConfig.(*Config)
	return t
}

func GetLedgerAccounts(ledgerId string) []Account {
	return ledgerAccountsMap[ledgerId]
}

func GetLedgerAccount(ledgerId string, account string) Account {
	accounts := ledgerAccountsMap[ledgerId]
	for _, acc := range accounts {
		if acc.Acc == account {
			return acc
		}
	}
	panic("Invalid account")
}

func UpdateLedgerAccounts(ledgerId string, accounts []Account) {
	ledgerAccountsMap[ledgerId] = accounts
}

func ClearLedgerAccounts(ledgerId string) {
	delete(ledgerAccountsMap, ledgerId)
}

func GetLedgerAccountTypes(ledgerId string) map[string]string {
	return ledgerAccountTypesMap[ledgerId]
}

func UpdateLedgerAccountTypes(ledgerId string, accountTypesMap map[string]string) {
	ledgerAccountTypesMap[ledgerId] = accountTypesMap
}

func ClearLedgerAccountTypes(ledgerId string) {
	delete(ledgerAccountTypesMap, ledgerId)
}

func GetAccountType(ledgerId string, acc string) AccountType {
	accountTypes := ledgerAccountTypesMap[ledgerId]
	accNodes := strings.Split(acc, ":")
	accountType := AccountType{
		Key: acc,
		// 默认取最后一个节点
		Name: accNodes[len(accNodes)-1],
	}
	var matchKey = ""
	for key, name := range accountTypes {
		if strings.Contains(acc, key) && len(matchKey) < len(key) {
			matchKey = key
			accountType = AccountType{Key: key, Name: name}
		}
	}
	return accountType
}

func IsInWhiteList(ledgerId string) bool {
	// ledger white list is empty, return true
	if len(whiteList) == 0 {
		return true
	}
	for i := range whiteList {
		if whiteList[i] == ledgerId {
			return true
		}
	}
	return false
}

func LoadLedgerConfigMap() error {
	path := GetServerLedgerConfigFilePath()
	// 文件不存在，则创建 ledger_config.json
	if !FileIfExist(path) {
		err := CreateFile(path)
		if err != nil {
			return err
		}
		ledgerConfigMap = make(map[string]Config)
	} else {
		// 文件存在，将文件内容加载到缓存
		fileContent, err := ReadFile(path)
		if err != nil {
			return err
		}
		if string(fileContent) != "" {
			err = json.Unmarshal(fileContent, &ledgerConfigMap)
			if err != nil {
				LogSystemError("Failed unmarshal ledger_config file (" + path + ")")
				return err
			}
		} else {
			ledgerConfigMap = make(map[string]Config)
		}
		LogSystemInfo("Success load ledger_config file (" + path + ")")
	}
	return nil
}

func LoadLedgerAccountsMap() error {
	if ledgerAccountsMap == nil {
		ledgerAccountsMap = make(map[string][]Account)
	}
	for _, config := range ledgerConfigMap {
		// 兼容性处理
		err := handleCompatible(config)
		if err != nil {
			return err
		}
		err = LoadLedgerAccounts(config.Id)
		if err != nil {
			return err
		}
		err = LoadLedgerCurrencyMap(&config)
		if err != nil {
			return err
		}
	}
	return nil
}

func handleCompatible(config Config) error {
	// 兼容性处理，.beancount-ns -> .beancount-gs
	beancountGsConfigPath := GetLedgerConfigDocument(config.DataPath)
	beancountNsConfigPath := GetCompatibleLedgerConfigDocument(config.DataPath)
	if FileIfExist(beancountNsConfigPath) && !FileIfExist(beancountGsConfigPath) {
		err := CopyDir(beancountNsConfigPath, beancountGsConfigPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func LoadLedgerAccounts(ledgerId string) error {
	config := ledgerConfigMap[ledgerId]
	// 加载 account_type.json 到缓存（内存）
	loadErr := LoadLedgerAccountTypesMap(config)
	if loadErr != nil {
		LogSystemError("Failed to load account types")
		return loadErr
	}
	accountDirPath := config.DataPath + "/account"
	dirs, err := os.ReadDir(accountDirPath)
	if err != nil {
		return err
	}
	accountMap := make(map[string]Account)
	for _, dir := range dirs {
		bytes, err := ReadFile(accountDirPath + "/" + dir.Name())
		if err != nil {
			return err
		}
		lines := strings.Split(string(bytes), "\n")
		var temp Account
		for _, line := range lines {
			if line != "" {
				words := strings.Fields(line)
				if len(words) >= 3 {
					key := words[2]
					temp = accountMap[key]
					account := Account{Acc: key, Type: nil}

					if words[1] == "open" {
						account.StartDate = words[0]
						if account.StartDate != "" && temp.StartDate != "" && strings.Compare(account.StartDate, temp.StartDate) < 0 {
							// 重复定义的账户，取最早的开始时间为准
							account.StartDate = temp.StartDate
						}
						// 货币单位
						if len(words) >= 4 {
							account.Currency = words[3]
						}
					} else if words[1] == "close" {
						account.EndDate = words[0]
						if account.EndDate != "" && temp.EndDate != "" && strings.Compare(account.EndDate, temp.EndDate) > 0 {
							// 重复定义的账户，取最晚的开始时间为准
							account.EndDate = temp.EndDate
						}
					}

					if account.StartDate == "" {
						account.StartDate = temp.StartDate
					}
					if account.EndDate == "" {
						account.EndDate = temp.EndDate
					}
					if account.Currency == "" {
						account.Currency = temp.Currency
					}

					// 如果结束时间小于开始时间，则结束时间为空
					if account.EndDate != "" && strings.Compare(account.StartDate, account.EndDate) > 0 {
						account.EndDate = ""
					}
					accountMap[key] = account
				}
			}
		}
	}

	accounts := make([]Account, 0)
	for _, account := range accountMap {
		accounts = append(accounts, account)
	}
	// 账户按字母排序
	sort.Sort(AccountSort(accounts))
	ledgerAccountsMap[config.Id] = accounts
	LogSystemInfo(fmt.Sprintf("Success load [%s] accounts cache", config.Mail))
	return nil
}

func LoadLedgerAccountTypesMap(config Config) error {
	path := GetLedgerAccountTypeFilePath(config.DataPath)
	fileContent, err := ReadFile(path)
	if err != nil {
		return err
	}
	accountTypes := make(map[string]string)
	err = json.Unmarshal(fileContent, &accountTypes)
	if err != nil {
		LogSystemError("Failed unmarshal config file (" + path + ")")
		return err
	}
	if ledgerAccountTypesMap == nil {
		ledgerAccountTypesMap = make(map[string]map[string]string)
	}
	ledgerAccountTypesMap[config.Id] = accountTypes
	LogSystemInfo(fmt.Sprintf("Success load [%s] account type cache", config.Mail))
	return nil
}

func WriteLedgerConfigMap(newLedgerConfigMap map[string]Config) error {
	path := GetServerLedgerConfigFilePath()
	mapBytes, err := json.Marshal(ledgerConfigMap)
	if err != nil {
		LogSystemError("Failed marshal ConfigMap")
		return err
	}
	err = WriteFile(path, string(mapBytes))
	ledgerConfigMap = newLedgerConfigMap
	LogSystemInfo("Success write ledger_config file (" + path + ")")
	return err
}

func GenerateServerSecret(secret string) string {
	if secret == "" {
		serverSecret = RandChar(16)
	} else {
		serverSecret = secret
	}
	return serverSecret
}

func EqualServerSecret(secret string) bool {
	return serverSecret == secret
}

func LoadServerCurrencyMap() {
	if serverCurrencies == nil {
		serverCurrencies = make([]LedgerCurrency, 0)
	}
	serverCurrencies = append(serverCurrencies, LedgerCurrency{Name: "人民币", Currency: "CNY", Symbol: "¥"})
	serverCurrencies = append(serverCurrencies, LedgerCurrency{Name: "美元", Currency: "USD", Symbol: "$"})
	serverCurrencies = append(serverCurrencies, LedgerCurrency{Name: "欧元", Currency: "EUR", Symbol: "€"})
	serverCurrencies = append(serverCurrencies, LedgerCurrency{Name: "日元", Currency: "JPY", Symbol: "¥"})
	serverCurrencies = append(serverCurrencies, LedgerCurrency{Name: "加拿大元", Currency: "CAD", Symbol: "$"})
	serverCurrencies = append(serverCurrencies, LedgerCurrency{Name: "俄罗斯卢布", Currency: "RUB", Symbol: "₽"})
}

func LoadLedgerCurrencyMap(config *Config) error {
	LoadServerCurrencyMap()
	path := GetLedgerCurrenciesFilePath(config.DataPath)
	if !FileIfExist(path) {
		err := CreateFile(path)
		if err != nil {
			return err
		}

		bytes, err := json.Marshal(serverCurrencies)
		if err != nil {
			return err
		}
		err = WriteFile(path, string(bytes))
		if err != nil {
			return err
		}
	}

	fileContent, err := ReadFile(path)
	if err != nil {
		return err
	}
	var currencies []LedgerCurrency
	err = json.Unmarshal(fileContent, &currencies)
	if err != nil {
		LogSystemError("Failed unmarshal config file (" + path + ")")
		return err
	}
	if ledgerCurrencyMap == nil {
		ledgerCurrencyMap = make(map[string][]LedgerCurrency)
	}
	ledgerCurrencyMap[config.Id] = currencies
	LogSystemInfo(fmt.Sprintf("Success load [%s] account type cache", config.Mail))
	// 刷新汇率
	RefreshLedgerCurrency(config)
	return nil
}

func GetLedgerCurrency(ledgerId string) []LedgerCurrency {
	return ledgerCurrencyMap[ledgerId]
}

type CommodityPrice struct {
	Date      string `json:"date"`
	Commodity string `json:"commodity"`
	Currency  string `json:"operatingCurrency"`
	Value     string `json:"value"`
}

func RefreshLedgerCurrency(ledgerConfig *Config) []LedgerCurrency {
	// 查询货币获取当前汇率
	output := BeanReportAllPrices(ledgerConfig)
	statsPricesResultList := make([]CommodityPrice, 0)
	lines := strings.Split(output, "\n")
	// foreach lines
	for _, line := range lines {
		if strings.Trim(line, " ") == "" {
			continue
		}
		// split line by " "
		words := strings.Fields(line)
		statsPricesResultList = append(statsPricesResultList, CommodityPrice{
			Date:      words[0],
			Commodity: words[2],
			Value:     words[3],
			Currency:  words[4],
		})
	}

	// statsPricesResultList 转为 map
	existCurrencyMap := make(map[string]CommodityPrice)
	for _, statsPricesResult := range statsPricesResultList {
		existCurrencyMap[statsPricesResult.Commodity] = statsPricesResult
	}

	result := make([]LedgerCurrency, 0)
	currencies := GetLedgerCurrency(ledgerConfig.Id)
	for _, c := range currencies {
		current := c.Currency == ledgerConfig.OperatingCurrency
		var price string
		var date string
		if current {
			price = "1"
			date = time.Now().Format("2006-01-02")
		} else {
			value, exists := existCurrencyMap[c.Currency]
			if exists {
				price = value.Value
				date = value.Date
			}
		}
		result = append(result, LedgerCurrency{
			Name:      c.Name,
			Currency:  c.Currency,
			Symbol:    c.Symbol,
			Current:   current,
			Price:     price,
			PriceDate: date,
		})
	}
	// 刷新账本货币缓存
	ledgerCurrencyMap[ledgerConfig.Id] = result
	return result
}

func GetLedgerCurrencyMap(ledgerId string) map[string]LedgerCurrency {
	currencyMap := make(map[string]LedgerCurrency)
	currencies := GetLedgerCurrency(ledgerId)
	if currencies == nil {
		return currencyMap
	}
	for _, currency := range currencies {
		currencyMap[currency.Currency] = currency
	}
	return currencyMap
}

func GetCommoditySymbol(ledgerId string, commodity string) string {
	currencyMap := GetLedgerCurrencyMap(ledgerId)
	if currencyMap == nil {
		return commodity
	}
	if _, ok := currencyMap[commodity]; !ok {
		return commodity
	}
	return currencyMap[commodity].Symbol
}

func GetServerCommoditySymbol(commodity string) string {
	for _, currency := range serverCurrencies {
		if currency.Currency == commodity {
			return currency.Symbol
		}
	}
	return commodity
}

func GetAccountPrefix(account string) string {
	nodes := strings.Split(account, ":")
	return nodes[0]
}

func GetAccountIconName(account string) string {
	nodes := strings.Split(account, ":")
	return strings.Join(nodes, "_")
}
