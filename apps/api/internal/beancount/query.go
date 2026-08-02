package beancount

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// Row 是 bean-query 一行结果（列名 → 值）。
type Row = map[string]string

// QueryEngine 抽象 beancount CLI 查询能力，便于测试替换。
type QueryEngine interface {
	QueryCSV(ctx context.Context, indexBeanPath, query string) ([]Row, error)
	Print(ctx context.Context, indexBeanPath, transactionID string) (string, error)
	Check(ctx context.Context, indexBeanPath string) ([]string, error)
}

type CmdEngine struct{}

func (CmdEngine) QueryCSV(ctx context.Context, indexBeanPath, query string) ([]Row, error) {
	out, err := run(ctx, "bean-query", "-f", "csv", indexBeanPath, query)
	if err != nil {
		return nil, err
	}
	rows, err := parseCSV(string(out))
	if err != nil {
		return nil, fmt.Errorf("parse bean-query csv: %w", err)
	}
	return rows, nil
}

func (CmdEngine) Print(ctx context.Context, indexBeanPath, transactionID string) (string, error) {
	out, err := run(ctx, "bean-query", indexBeanPath, "PRINT FROM id = '"+transactionID+"'")
	if err != nil {
		return "", err
	}
	return decodeOutput(out), nil
}

func (CmdEngine) Check(ctx context.Context, indexBeanPath string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "bean-check", indexBeanPath)
	cmd.Env = envWithUTF8()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		lines := strings.Split(stderr.String(), "\n")
		result := make([]string, 0, len(lines))
		for _, l := range lines {
			if l = strings.TrimSpace(l); l != "" {
				result = append(result, l)
			}
		}
		return result, nil
	}
	return []string{}, nil
}

func run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = envWithUTF8()
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s failed: %w", name, err)
	}
	return out, nil
}

func envWithUTF8() []string {
	return append(os.Environ(), "PYTHONIOENCODING=utf-8")
}

func parseCSV(s string) ([]Row, error) {
	r := csv.NewReader(strings.NewReader(s))
	r.TrimLeadingSpace = true
	header, err := r.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return []Row{}, nil
		}
		return nil, err
	}
	rows := make([]Row, 0)
	for {
		record, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		row := make(Row, len(header))
		for i, col := range header {
			value := ""
			if i < len(record) {
				value = strings.TrimSpace(record[i])
			}
			if value == "None" {
				value = ""
			}
			row[strings.TrimSpace(col)] = value
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// decodeOutput 优先按 UTF-8 解析；无效时按 GBK 兜底（旧版 bean-query 在 Windows 的控制台编码）。
func decodeOutput(b []byte) string {
	if utf8.Valid(b) {
		return string(b)
	}
	decoded, _, err := transform.String(simplifiedchinese.GBK.NewDecoder(), string(b))
	if err != nil {
		return string(b)
	}
	return decoded
}
