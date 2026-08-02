package repository

import (
	"errors"
	"os"
	"path/filepath"
)

// AppendMonthTransaction 将交易文本追加到对应月份的 bean 文件；
// 文件不存在时创建并登记到 month/months.bean 的 include。
func AppendMonthTransaction(dataPath, month, content string) error {
	dir := filepath.Join(dataPath, "month")
	file := filepath.Join(dir, month+".bean")
	if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(file, nil, 0o644); err != nil {
			return err
		}
		if err := appendLine(filepath.Join(dir, "months.bean"), `include "./`+month+`.bean"`); err != nil {
			return err
		}
	}
	return appendLine(file, content)
}

// AppendPrice 追加一条 price 指令到 price/prices.bean。
func AppendPrice(dataPath, line string) error {
	dir := filepath.Join(dataPath, "price")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return appendLine(filepath.Join(dir, "prices.bean"), line)
}

func appendLine(path, content string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString("\n" + content); err != nil {
		return err
	}
	return nil
}
