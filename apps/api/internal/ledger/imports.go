package ledger

import (
	"context"
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/beancount-gs/api/internal/db"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type ImportRow struct {
	Index            int     `json:"index"`
	Date             string  `json:"date"`
	Payee            string  `json:"payee"`
	Narration        string  `json:"narration"`
	Number           string  `json:"number"`
	Currency         string  `json:"currency"`
	SuggestedAccount string  `json:"suggested_account"`
	Confidence       float64 `json:"confidence"`
	Status           string  `json:"status"`
}

var sourceCounterAccounts = map[string]string{
	"alipay": "Assets:Flow:EBank:AliPay:支付宝",
	"wechat": "Assets:Flow:EBank:WxPay:微信支付",
	"icbc":   "Assets:Flow:Bank:ICBC:工商银行",
	"abc":    "Assets:Flow:Bank:ABC:农业银行",
}

// ImportPreview 解析账单 CSV 并生成预览行（不落账）。
func (s *Service) ImportPreview(ctx context.Context, l db.Ledger, source string, r io.Reader) ([]ImportRow, error) {
	rows, err := parseImportCSV(source, r)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].Index = i
		rows[i].Currency = "CNY"
		rows[i].SuggestedAccount, rows[i].Status = suggestImportAccount(rows[i].Number)
	}
	return rows, nil
}

// ImportConfirm 按用户确认的账户批量写入交易。
func (s *Service) ImportConfirm(ctx context.Context, l db.Ledger, source string, rows []ImportRow, actor Actor) (map[string]any, error) {
	counter := sourceCounterAccounts[source]
	if counter == "" {
		return nil, errors.New("未知的导入来源")
	}
	created := 0
	failed := make([]map[string]any, 0)
	for _, row := range rows {
		account := strings.TrimSpace(row.SuggestedAccount)
		if account == "" {
			failed = append(failed, map[string]any{"index": row.Index, "reason": "账户未选择"})
			continue
		}
		number := strings.TrimSpace(row.Number)
		if number == "" {
			failed = append(failed, map[string]any{"index": row.Index, "reason": "金额为空"})
			continue
		}
		revision, err := s.currentRevision(ctx, l)
		if err != nil {
			return nil, err
		}
		_, err = s.Create(ctx, l, Transaction{
			Date:      row.Date,
			Payee:     row.Payee,
			Narration: row.Narration,
			Postings: []Posting{
				{Account: account, Units: &Amount{Number: number, Currency: "CNY"}},
				{Account: counter, Units: &Amount{Number: negate(number), Currency: "CNY"}},
			},
		}, revision, actor)
		if err != nil {
			failed = append(failed, map[string]any{"index": row.Index, "reason": err.Error()})
			continue
		}
		created++
	}
	return map[string]any{"created": created, "failed": failed}, nil
}

func (s *Service) currentRevision(ctx context.Context, l db.Ledger) (int64, error) {
	got, err := s.Store.GetLedger(ctx, l.ID)
	if err != nil {
		return 0, err
	}
	return got.Revision, nil
}

func negate(number string) string {
	number = strings.TrimSpace(number)
	if strings.HasPrefix(number, "-") {
		return strings.TrimPrefix(number, "-")
	}
	return "-" + number
}

func suggestImportAccount(number string) (string, string) {
	if strings.HasPrefix(strings.TrimSpace(number), "-") {
		return "Expenses:", "suggested"
	}
	return "Income:", "suggested"
}

func parseImportCSV(source string, r io.Reader) ([]ImportRow, error) {
	switch source {
	case "alipay":
		return parseAlipayCSV(r)
	case "wechat":
		return parseWechatCSV(r)
	case "icbc":
		return parseIcbcCSV(r)
	case "abc":
		return parseAbcCSV(r)
	default:
		return nil, fmt.Errorf("不支持的导入来源：%s", source)
	}
}

// decodeCSVInput 优先按 UTF-8 解析，无效时按 GBK 解码（支付宝等导出的旧编码）。
func decodeCSVInput(r io.Reader) (io.Reader, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if !utf8.Valid(data) {
		decoded, _, err := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), data)
		if err != nil {
			return nil, err
		}
		data = decoded
	}
	return bytes.NewReader(data), nil
}

func newCSVReader(r io.Reader) *csv.Reader {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	cr.TrimLeadingSpace = true
	return cr
}

func parseAlipayCSV(r io.Reader) ([]ImportRow, error) {
	decoded, err := decodeCSVInput(r)
	if err != nil {
		return nil, err
	}
	reader := newCSVReader(decoded)
	rows := make([]ImportRow, 0)
	for {
		line, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		switch {
		case len(line) == 17: // 浏览器导出
			status := strings.TrimSpace(line[15])
			if status == "" {
				continue
			}
			dateField := strings.Fields(line[2])
			if len(dateField) < 1 {
				continue
			}
			number := strings.TrimSpace(line[9])
			if status != "已收入" {
				number = "-" + number
			}
			rows = append(rows, ImportRow{
				Date: dateField[0], Payee: strings.TrimSpace(line[7]),
				Narration: strings.TrimSpace(line[8]), Number: number,
			})
		case len(line) == 12 || len(line) == 13: // 移动端导出
			dateField := strings.Fields(line[0])
			if len(dateField) < 1 {
				continue
			}
			status := strings.TrimSpace(line[5])
			number := strings.TrimSpace(line[6])
			if status == "支出" {
				number = "-" + number
			}
			rows = append(rows, ImportRow{
				Date: dateField[0], Payee: strings.TrimSpace(line[2]),
				Narration: strings.TrimSpace(line[4]), Number: number,
			})
		}
	}
	return rows, nil
}

func parseWechatCSV(r io.Reader) ([]ImportRow, error) {
	decoded, err := decodeCSVInput(r)
	if err != nil {
		return nil, err
	}
	reader := newCSVReader(decoded)
	rows := make([]ImportRow, 0)
	for {
		line, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(line) <= 8 {
			continue
		}
		status := strings.TrimSpace(line[4])
		if status != "收入" && status != "支出" {
			continue
		}
		dateField := strings.Fields(line[0])
		if len(dateField) < 1 {
			continue
		}
		number := strings.Trim(strings.TrimSpace(line[5]), "¥")
		if status == "支出" && !strings.HasPrefix(number, "-") {
			number = "-" + number
		}
		rows = append(rows, ImportRow{
			Date: dateField[0], Payee: strings.TrimSpace(line[2]),
			Narration: strings.TrimSpace(line[3]), Number: number,
		})
	}
	return rows, nil
}

func parseIcbcCSV(r io.Reader) ([]ImportRow, error) {
	decoded, err := decodeCSVInput(r)
	if err != nil {
		return nil, err
	}
	reader := newCSVReader(decoded)
	rows := make([]ImportRow, 0)
	for {
		line, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(line) < 13 || strings.TrimSpace(line[0]) == "交易日期" {
			continue
		}
		income := strings.TrimSpace(line[8])
		expense := strings.TrimSpace(line[9])
		number := ""
		switch {
		case income != "":
			number = strings.ReplaceAll(income, ",", "")
		case expense != "":
			number = "-" + strings.ReplaceAll(expense, ",", "")
		default:
			continue
		}
		rows = append(rows, ImportRow{
			Date: strings.TrimSpace(line[0]), Payee: strings.TrimSpace(line[12]),
			Narration: strings.TrimSpace(line[1]), Number: number,
		})
	}
	return rows, nil
}

func parseAbcCSV(r io.Reader) ([]ImportRow, error) {
	decoded, err := decodeCSVInput(r)
	if err != nil {
		return nil, err
	}
	reader := newCSVReader(decoded)
	rows := make([]ImportRow, 0)
	for {
		line, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(line) < 11 || strings.TrimSpace(line[0]) == "交易日期" {
			continue
		}
		amount := strings.TrimSpace(line[2])
		number := ""
		switch {
		case strings.HasPrefix(amount, "+"):
			number = strings.TrimPrefix(amount, "+")
		case strings.HasPrefix(amount, "-"):
			number = "-" + strings.TrimPrefix(amount, "-")
		default:
			continue
		}
		date, err := time.Parse("20060102", strings.TrimSpace(line[0]))
		if err != nil {
			continue
		}
		rows = append(rows, ImportRow{
			Date: date.Format("2006-01-02"), Payee: strings.TrimSpace(line[10]),
			Narration: strings.TrimSpace(line[9]), Number: number,
		})
	}
	return rows, nil
}
