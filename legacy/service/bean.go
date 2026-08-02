package service

import (
	"errors"
	"fmt"
	"github.com/beancount-gs/script"
)

// CreateMonthBeanFileIfNotExist create month bean file if not exist, otherwise return.
func CreateMonthBeanFileIfNotExist(ledgerDataPath string, month string) error {
	// 文件不存在，则创建
	filePath := fmt.Sprintf("%s/month/%s.bean", ledgerDataPath, month)
	if !script.FileIfExist(filePath) {
		err := script.CreateFile(filePath)
		if err != nil {
			return errors.New("failed to create file")
		}
		// include ./2021-11.bean
		err = script.AppendFileInNewLine(script.GetLedgerMonthsFilePath(ledgerDataPath), fmt.Sprintf("include \"./%s.bean\"", month))
		if err != nil {
			return errors.New("failed to append content to months.bean")
		}
	}
	return nil
}
