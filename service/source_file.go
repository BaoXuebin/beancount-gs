package service

import (
	"fmt"
	"github.com/beancount-gs/script"
	"github.com/gin-gonic/gin"
	"os"
	"strings"
	"time"
)

func QueryLedgerSourceFileDir(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	result, err := dirs(ledgerConfig.DataPath, ledgerConfig.DataPath)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	OK(c, result)
}

func dirs(parent string, dirPath string) ([]string, error) {
	result := make([]string, 0)
	rd, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	for _, dir := range rd {
		parentDir := dirPath + "/" + dir.Name()
		if dir.IsDir() {
			// 跳过备份文件夹
			if dir.Name() == "bak" {
				continue
			}
			files, err := dirs(parent, parentDir)
			if err != nil {
				return nil, err
			}
			result = append(result, files...)
		} else {
			fmt.Println(parentDir)
			result = append(result, strings.ReplaceAll(parentDir, parent+"/", ""))
		}
	}
	return result, nil
}

func QueryLedgerSourceFileContent(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	queryParams := script.GetQueryParams(c)
	if queryParams.Path == "" {
		BadRequest(c, "params must not be blank")
		return
	}
	bytes, err := script.ReadFile(ledgerConfig.DataPath + "/" + queryParams.Path)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	OK(c, string(bytes))
}

type UpdateSourceFileForm struct {
	Path    string `form:"path" binding:"required"`
	Content string `form:"content"`
}

func UpdateLedgerSourceFileContent(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)

	var updateSourceFileForm UpdateSourceFileForm
	if err := c.ShouldBindJSON(&updateSourceFileForm); err != nil {
		BadRequest(c, err.Error())
		return
	}

	sourceFilePath := ledgerConfig.DataPath + "/" + updateSourceFileForm.Path
	targetFilePath := ledgerConfig.DataPath + "/bak/" + time.Now().Format("20060102150405") + "_" + strings.ReplaceAll(updateSourceFileForm.Path, "/", "_")
	// 备份数据
	if ledgerConfig.IsBak {
		err := script.CopyFile(sourceFilePath, targetFilePath)
		if err != nil {
			InternalError(c, err.Error())
			return
		}
	}

	err := script.WriteFile(sourceFilePath, updateSourceFileForm.Content)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	OK(c, nil)
}
