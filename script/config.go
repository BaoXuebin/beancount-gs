package script

import (
	"encoding/json"
)

var serverConfig Config
var ledgerConfigMap ConfigMap
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

type ConfigMap map[string]Config

func GetServerConfig() Config {
	return serverConfig
}

func GetLedgerConfigMap() map[string]Config {
	return ledgerConfigMap
}

func GetLedgerConfig(ledgerId string) Config {
	return ledgerConfigMap[ledgerId]
}

func GetLedgerConfigByMail(mail string) *Config {
	for _, v := range ledgerConfigMap {
		if v.Mail == mail {
			return &v
		}
	}
	return nil
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
		LogError("Failed unmarshall config file (/config/config.json)")
		return err
	}
	LogInfo("Success load config file (/config/config.json)")
	// load white list
	fileContent, err = ReadFile("./config/white_list.json")
	if err != nil {
		return err
	}
	err = json.Unmarshal(fileContent, &whiteList)
	if err != nil {
		LogError("Failed unmarshal whitelist file (/config/white_list.json)")
		return err
	}
	LogInfo("Success load whitelist file (/config/white_list.json)")
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
		LogError("Failed unmarshal config file (" + path + ")")
		return err
	}
	LogInfo("Success load ledger_config file (" + path + ")")
	return nil
}

func WriteLedgerConfigMap(newLedgerConfigMap ConfigMap) error {
	path := GetServerLedgerConfigFilePath()
	mapBytes, err := json.Marshal(ledgerConfigMap)
	if err != nil {
		LogError("Failed marshal ConfigMap")
		return err
	}
	err = WriteFile(path, string(mapBytes))
	ledgerConfigMap = newLedgerConfigMap
	LogInfo("Success write ledger_config file (" + path + ")")
	return err
}
