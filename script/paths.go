package script

import "os"

func GetServerLedgerConfigFilePath() string {
	return GetServerConfig().DataPath + "/ledger_config.json"
}

func GetExampleLedgerConfigDirPath() string {
	currentPath, err := os.Getwd()
	if err != nil {
		return ""
	}
	return currentPath + "/example"
}
