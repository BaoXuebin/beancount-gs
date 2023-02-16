package script

import "os"

func GetServerConfigFilePath() string {
	currentPath, _ := os.Getwd()
	return currentPath + "/config/config.json"
}

func GetServerWhiteListFilePath() string {
	currentPath, _ := os.Getwd()
	return currentPath + "/config/white_list.json"
}

func GetServerLedgerConfigFilePath() string {
	return GetServerConfig().DataPath + "/ledger_config.json"
}

func GetTemplateLedgerConfigDirPath() string {
	currentPath, err := os.Getwd()
	if err != nil {
		return ""
	}
	return currentPath + "/template"
}

func GetLedgerConfigDocument(dataPath string) string {
	return dataPath + "/.beancount-gs"
}

func GetCompatibleLedgerConfigDocument(dataPath string) string {
	return dataPath + "/.beancount-ns"
}

func GetLedgerTransactionsTemplateFilePath(dataPath string) string {
	return dataPath + "/.beancount-gs/transaction_template.json"
}

func GetLedgerAccountTypeFilePath(dataPath string) string {
	return dataPath + "/.beancount-gs/account_type.json"
}

func GetLedgerPriceFilePath(dataPath string) string {
	return dataPath + "/price/prices.bean"
}

func GetLedgerMonthsFilePath(dataPath string) string {
	return dataPath + "/month/months.bean"
}

func GetLedgerMonthFilePath(dataPath string, month string) string {
	return dataPath + "/month/" + month + ".bean"
}

func GetLedgerIndexFilePath(dataPath string) string {
	LogInfo(dataPath, dataPath+"/index.bean")
	return dataPath + "/index.bean"
}
