package repository

import (
	"os"
	"path/filepath"
	"strings"
)

type AccountDirective struct {
	Date     string
	Kind     string // open | close
	Account  string
	Currency string
	Booking  string
}

// ReadAccountFiles 解析 account/*.bean 中的 open / close 指令。
func ReadAccountFiles(dataPath string) ([]AccountDirective, error) {
	dir := filepath.Join(dataPath, "account")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []AccountDirective{}, nil
		}
		return nil, err
	}
	directives := make([]AccountDirective, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".bean") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		for _, line := range strings.Split(string(content), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, ";") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) < 3 || (fields[1] != "open" && fields[1] != "close") {
				continue
			}
			d := AccountDirective{Date: fields[0], Kind: fields[1], Account: fields[2]}
			if fields[1] == "open" {
				if len(fields) >= 4 {
					d.Currency = fields[3]
				}
				if len(fields) > 4 {
					for _, f := range fields[4:] {
						if strings.Contains(f, "FIFO") {
							d.Booking = "fifo"
						}
					}
				}
			}
			directives = append(directives, d)
		}
	}
	return directives, nil
}

// AppendAccountDirective 追加 open / close 指令到对应账户类型的文件（account/<prefix>.bean）。
func AppendAccountDirective(dataPath, prefix, line string) error {
	dir := filepath.Join(dataPath, "account")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return appendLine(filepath.Join(dir, strings.ToLower(prefix)+".bean"), line)
}
