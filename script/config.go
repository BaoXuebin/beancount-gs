package script

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"os"
	"sort"
	"strings"
)

var serverConfig Config
var ledgerConfigMap map[string]Config
var ledgerAccountsMap map[string][]Account
var ledgerAccountTypesMap map[string]map[string]string
var whiteList []string

type Config struct {
	Id                string `json:"id"`
	Mail              string `json:"mail"`
	Title             string `json:"title"`
	DataPath          string `json:"dataPath"`
	OperatingCurrency string `json:"operatingCurrency"`
	StartDate         string `json:"startDate"`
	IsBak             bool   `json:"isBak"`
}

type Account struct {
	Acc       string `json:"account"`
	Commodity string `json:"commodity"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

type AccountType struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

func GetServerConfig() Config {
	return serverConfig
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

func GetLedgerAccountTypes(ledgerId string) map[string]string {
	return ledgerAccountTypesMap[ledgerId]
}

func IsInWhiteList(ledgerId string) bool {
	// ledger white list is empty, return true
	if whiteList == nil || len(whiteList) == 0 {
		return true
	}
	for i := range whiteList {
		if whiteList[i] == ledgerId {
			return true
		}
	}
	return false
}

func LoadServerConfig() error {
	fileContent, err := ReadFile("./config/config.json")
	if err != nil {
		return err
	}
	err = json.Unmarshal(fileContent, &serverConfig)
	if err != nil {
		LogSystemError("Failed unmarshall config file (/config/config.json)")
		return err
	}
	LogSystemInfo("Success load config file (/config/config.json)")
	// load white list
	fileContent, err = ReadFile("./config/white_list.json")
	if err != nil {
		return err
	}
	err = json.Unmarshal(fileContent, &whiteList)
	if err != nil {
		LogSystemError("Failed unmarshal whitelist file (/config/white_list.json)")
		return err
	}
	LogSystemInfo("Success load whitelist file (/config/white_list.json)")
	return nil
}

func LoadLedgerConfigMap() error {
	path := GetServerLedgerConfigFilePath()
	fileContent, err := ReadFile(path)
	if err != nil {
		return err
	}
	err = json.Unmarshal(fileContent, &ledgerConfigMap)
	if err != nil {
		LogSystemError("Failed unmarshal config file (" + path + ")")
		return err
	}
	LogSystemInfo("Success load ledger_config file (" + path + ")")
	return nil
}

func LoadLedgerAccountsMap() error {
	if ledgerAccountsMap == nil {
		ledgerAccountsMap = make(map[string][]Account)
	}
	for _, config := range ledgerConfigMap {
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
						account := Account{Acc: key}
						// 货币单位
						if len(words) >= 4 {
							account.Commodity = words[3]
						}
						if words[1] == "open" {
							account.StartDate = words[0]
						} else if words[1] == "close" {
							account.EndDate = words[0]
						}
						if temp.StartDate != "" {
							account.StartDate = temp.StartDate
						}
						if temp.EndDate != "" {
							account.EndDate = temp.EndDate
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
	}
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
	LogSystemInfo("Success load ledger_config file (" + path + ")")
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

func GetCommoditySymbol(commodity string) string {
	switch commodity {
	case "CNY":
		return "￥"
	case "USD":
		return "$"
	}
	return ""
}
