package service

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/beancount-gs/script"
	"github.com/gin-gonic/gin"
)

func CheckBeancount(c *gin.Context) {
	cmd := exec.Command("bean-query", "--version")
	output, err := cmd.Output()
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	OK(c, string(output))
}

func QueryServerConfig(c *gin.Context) {
	OK(c, script.GetServerConfig())
}

type QueryLedgerResult struct {
	Mail              string `json:"mail"`
	Title             string `json:"title"`
	CreateDate        string `json:"createDate"`
	OperatingCurrency string `json:"operatingCurrency"`
}

type LedgerSort []QueryLedgerResult

func (s LedgerSort) Len() int {
	return len(s)
}

func (s LedgerSort) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s LedgerSort) Less(i, j int) bool {
	return s[i].CreateDate <= s[j].CreateDate && s[i].Mail <= s[j].Mail
}

func QueryLedgerList(c *gin.Context) {
	result := make([]QueryLedgerResult, 0)
	for _, config := range script.GetLedgerConfigMap() {
		result = append(result, QueryLedgerResult{
			Title:             config.Title,
			Mail:              config.Mail,
			CreateDate:        config.CreateDate,
			OperatingCurrency: config.OperatingCurrency,
		})
	}
	sort.Sort(LedgerSort(result))
	OK(c, result)
}

type UpdateConfigForm struct {
	Secret            string `form:"secret" binding:"required"`
	StartDate         string `form:"startDate" binding:"required"`
	DataPath          string `form:"dataPath" binding:"required"`
	OperatingCurrency string `form:"operatingCurrency" binding:"required"`
	OpeningBalances   string `form:"openingBalances" binding:"required"`
	IsBak             bool   `form:"isBak" binding:"required"`
}

func UpdateServerConfig(c *gin.Context) {
	var updateConfigForm UpdateConfigForm
	if err := c.ShouldBindJSON(&updateConfigForm); err != nil {
		BadRequest(c, err.Error())
		return
	}
	if !script.EqualServerSecret(updateConfigForm.Secret) {
		ServerSecretNotMatch(c)
		return
	}
	var serverConfig = script.Config{
		OperatingCurrency: updateConfigForm.OperatingCurrency,
		DataPath:          updateConfigForm.DataPath,
		StartDate:         updateConfigForm.StartDate,
		OpeningBalances:   updateConfigForm.OpeningBalances,
		IsBak:             updateConfigForm.IsBak,
	}
	// 更新配置
	err := script.UpdateServerConfig(serverConfig)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	// 账本目录不存在，则创建
	dataPath := serverConfig.DataPath
	if !script.FileIfExist(dataPath) {
		err = script.MkDir(dataPath)
		if err != nil {
			InternalError(c, err.Error())
			return
		}
	}
	// 加载账户缓存
	err = script.LoadLedgerConfigMap()
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	err = script.LoadLedgerAccountsMap()
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	QueryServerConfig(c)
}

type LoginForm struct {
	LedgerName        string `form:"ledgerName" binding:"required"`
	Secret            string `form:"secret"`
	OperatingCurrency string `form:"operatingCurrency"`
	StartDate         string `form:"startDate"`
	OpeningBalances   string `form:"openingBalances"`
	IsBak             bool   `form:"isBak"`
}

func OpenOrCreateLedger(c *gin.Context) {
	var loginForm LoginForm
	if err := c.ShouldBindJSON(&loginForm); err != nil {
		BadRequest(c, err.Error())
		return
	}
	// is mail exist white list
	if !script.IsInWhiteList(loginForm.LedgerName) {
		LedgerIsNotExist(c)
		return
	}

	t := sha1.New()
	_, err := io.WriteString(t, loginForm.LedgerName+loginForm.Secret)
	if err != nil {
		LedgerIsNotAllowAccess(c)
		return
	}

	ledgerId := hex.EncodeToString(t.Sum(nil))
	userLedger := script.GetLedgerConfigByMail(loginForm.LedgerName)
	if userLedger != nil {
		if ledgerId != userLedger.Id {
			LedgerIsNotAllowAccess(c)
			return
		}
		// 账本已存在，返回账本信息
		resultMap := make(map[string]string)
		resultMap["ledgerId"] = ledgerId
		resultMap["title"] = userLedger.Title
		resultMap["currency"] = userLedger.OperatingCurrency
		resultMap["currencySymbol"] = script.GetServerCommoditySymbol(userLedger.OperatingCurrency)
		resultMap["createDate"] = userLedger.CreateDate
		OK(c, resultMap)
		return
	}

	userLedger, err = createNewLedger(loginForm, ledgerId)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	resultMap := make(map[string]string)
	resultMap["ledgerId"] = ledgerId
	resultMap["title"] = userLedger.Title
	resultMap["currency"] = userLedger.OperatingCurrency
	resultMap["currencySymbol"] = script.GetCommoditySymbol(ledgerId, userLedger.OperatingCurrency)
	resultMap["createDate"] = userLedger.CreateDate
	OK(c, resultMap)
}

func DeleteLedger(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	// remove from ledger_config.json
	ledgerConfigMap := script.GetLedgerConfigMap()
	delete(ledgerConfigMap, ledgerConfig.Id)
	err := script.WriteLedgerConfigMap(ledgerConfigMap)
	if err != nil {
		InternalError(c, "Failed to update ledger_config.json")
		return
	}
	// remove from account cache
	script.ClearLedgerAccounts(ledgerConfig.Id)
	script.LogInfo(ledgerConfig.Mail, "Success clear ledger account cache "+ledgerConfig.Id)
	// remove from account types cache
	script.ClearLedgerAccountTypes(ledgerConfig.Id)
	script.LogInfo(ledgerConfig.Mail, "Success clear ledger account types cache "+ledgerConfig.Id)
	// delete source file
	err = os.RemoveAll(ledgerConfig.DataPath)
	if err != nil {
		script.LogError(ledgerConfig.Mail, "Failed to delete ledger, cause by "+err.Error())
		InternalError(c, "Failed to delete ledger")
		return
	}
	script.LogInfo(ledgerConfig.Mail, "Success delete "+ledgerConfig.DataPath)
	OK(c, "OK")
}

func CheckLedger(c *gin.Context) {
	var stderr bytes.Buffer
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	cmd := exec.Command("bean-check", script.GetLedgerIndexFilePath(ledgerConfig.DataPath))
	cmd.Stderr = &stderr
	_, err := cmd.Output()
	result := make([]string, 0)
	if err != nil {
		errors := strings.Split(stderr.String(), "\r\n")
		for _, e := range errors {
			if e == "" {
				continue
			}
			result = append(result, e)
		}
	}
	OK(c, result)
}

func createNewLedger(loginForm LoginForm, ledgerId string) (*script.Config, error) {
	// create new ledger
	serverConfig := script.GetServerConfig()
	ledgerConfigMap := script.GetLedgerConfigMap()

	currency := loginForm.OperatingCurrency
	if currency == "" {
		currency = serverConfig.OperatingCurrency
	}
	startDate := loginForm.StartDate
	if startDate == "" {
		startDate = serverConfig.StartDate
	}
	openingBalances := loginForm.OpeningBalances
	if openingBalances == "" {
		openingBalances = serverConfig.OpeningBalances
	}

	ledgerConfig := script.Config{
		Id:                ledgerId,
		Mail:              loginForm.LedgerName,
		Title:             loginForm.LedgerName,
		DataPath:          serverConfig.DataPath + "/" + ledgerId,
		OperatingCurrency: currency,
		StartDate:         startDate,
		OpeningBalances:   openingBalances,
		IsBak:             loginForm.IsBak,
		CreateDate:        time.Now().Format("2006-01-02"),
	}
	// init ledger files
	err := initLedgerFiles(script.GetTemplateLedgerConfigDirPath(), ledgerConfig.DataPath, ledgerConfig)
	if err != nil {
		return nil, err
	}
	// add ledger config to ledger_config.json
	ledgerConfigMap[ledgerId] = ledgerConfig
	err = script.WriteLedgerConfigMap(ledgerConfigMap)
	if err != nil {
		return nil, err
	}
	// add accounts cache
	err = script.LoadLedgerAccounts(ledgerId)
	if err != nil {
		return nil, err
	}
	// add currency cache
	err = script.LoadLedgerCurrencyMap(&ledgerConfig)
	if err != nil {
		return nil, err
	}
	return &ledgerConfig, nil
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
			if err == nil {
				err = copyFile(newSourceFilePath, newTargetFilePath, ledgerConfig)
			}
		} else if !script.FileIfExist(newTargetFilePath) {
			var fileContent, err = script.ReadFile(newSourceFilePath)
			if err != nil {
				return err
			}
			err = script.WriteFile(newTargetFilePath, strings.ReplaceAll(strings.ReplaceAll(string(fileContent), "%startDate%", ledgerConfig.StartDate), "%operatingCurrency%", ledgerConfig.OperatingCurrency))
			if err != nil {
				return err
			}
			script.LogInfo(ledgerConfig.Mail, "Success create file "+newTargetFilePath)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
