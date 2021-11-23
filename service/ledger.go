package service

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"github.com/beancount-gs/script"
	"github.com/gin-gonic/gin"
	"io"
	"io/ioutil"
	"strings"
)

type LoginForm struct {
	Mail   string `form:"mail" binding:"required"`
	Secret string `form:"secret" binding:"required"`
}

func OpenOrCreateLedger(c *gin.Context) {
	var loginForm LoginForm
	if err := c.ShouldBindJSON(&loginForm); err != nil {
		BadRequest(c, err.Error())
		return
	}
	// is mail exist white list
	if !script.IsInWhiteList(loginForm.Mail) {
		LedgerIsNotExist(c)
		return
	}

	t := sha1.New()
	_, err := io.WriteString(t, loginForm.Mail+loginForm.Secret)
	if err != nil {
		LedgerIsNotAllowAccess(c)
		return
	}

	ledgerId := hex.EncodeToString(t.Sum(nil))
	fmt.Println(ledgerId)
	userLedger := script.GetLedgerConfigByMail(loginForm.Mail)
	if userLedger != nil {
		if ledgerId != userLedger.Id {
			LedgerIsNotAllowAccess(c)
			return
		}
	}
	// create new ledger
	serverConfig := script.GetServerConfig()
	ledgerConfigMap := script.GetLedgerConfigMap()
	ledgerConfig := script.Config{
		Id:                ledgerId,
		Mail:              loginForm.Mail,
		Title:             serverConfig.Title,
		DataPath:          serverConfig.DataPath + "/" + ledgerId,
		OperatingCurrency: serverConfig.OperatingCurrency,
		StartDate:         serverConfig.StartDate,
		IsBak:             serverConfig.IsBak,
	}
	// init ledger files
	err = initLedgerFiles(script.GetExampleLedgerConfigDirPath(), ledgerConfig.DataPath, ledgerConfig)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	// add ledger config to ledger_config.json
	ledgerConfigMap[ledgerId] = ledgerConfig
	err = script.WriteLedgerConfigMap(ledgerConfigMap)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	OK(c, ledgerId)
}

func initLedgerFiles(sourceFilePath string, targetFilePath string, ledgerConfig script.Config) error {
	return copyFile(sourceFilePath, targetFilePath, ledgerConfig)
}

func copyFile(sourceFilePath string, targetFilePath string, ledgerConfig script.Config) error {
	rd, err := ioutil.ReadDir(sourceFilePath)
	if err != nil {
		return err
	}
	for _, fi := range rd {
		newSourceFilePath := sourceFilePath + "/" + fi.Name()
		newTargetFilePath := targetFilePath + "/" + fi.Name()
		if fi.IsDir() {
			err = script.MkDir(newTargetFilePath)
			err = copyFile(newSourceFilePath, newTargetFilePath, ledgerConfig)
		} else if !script.FileIfExist(newTargetFilePath) {
			fileContent, err := script.ReadFile(newSourceFilePath)
			if err != nil {
				return err
			}
			err = script.WriteFile(newTargetFilePath, strings.ReplaceAll(strings.ReplaceAll(string(fileContent), "%startDate%", ledgerConfig.StartDate), "%operatingCurrency%", ledgerConfig.OperatingCurrency))
			script.LogInfo(ledgerConfig.Mail, "Success create file " + newTargetFilePath)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
