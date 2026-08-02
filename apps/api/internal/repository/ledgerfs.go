package repository

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// InitLedgerFiles 从模板目录初始化账本文件结构，并替换 %startDate% / %operatingCurrency% 占位符。
// 目标已存在的文件不会被覆盖。
func InitLedgerFiles(dataPath, templateDir, startDate, operatingCurrency string) error {
	if err := os.MkdirAll(dataPath, 0o755); err != nil {
		return fmt.Errorf("create ledger dir: %w", err)
	}
	if _, err := os.Stat(templateDir); err != nil {
		return fmt.Errorf("template dir %q: %w", templateDir, err)
	}
	return filepath.WalkDir(templateDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == templateDir {
			return nil
		}
		rel, err := filepath.Rel(templateDir, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dataPath, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if _, err := os.Stat(target); err == nil {
			return nil // 不覆盖已有文件
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(content)
		text = strings.ReplaceAll(text, "%startDate%", startDate)
		text = strings.ReplaceAll(text, "%operatingCurrency%", operatingCurrency)
		if err := os.WriteFile(target, []byte(text), 0o644); err != nil {
			return err
		}
		return nil
	})
}

func EnsureDataRoot(root string) error {
	if strings.TrimSpace(root) == "" {
		return errors.New("data root is empty")
	}
	return os.MkdirAll(root, 0o755)
}
