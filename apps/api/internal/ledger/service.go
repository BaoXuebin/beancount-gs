package ledger

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/beancount-gs/api/internal/beancount"
	"github.com/beancount-gs/api/internal/db"
	"github.com/beancount-gs/api/internal/repository"
)

var ErrNotFound = errors.New("transaction not found")

const transactionSelect = `id, date, payee, narration, tags, links, account, number, currency, cost_number, cost_currency, cost_date, price`

func indexPath(l db.Ledger) string {
	return filepath.Join(l.DataPath, "index.bean")
}

func (s *Service) List(ctx context.Context, l db.Ledger, f Filters) ([]Transaction, error) {
	query := "SELECT " + transactionSelect + buildWhere(f) + " ORDER BY date " + orderBy(f.Order)
	rows, err := s.Engine.QueryCSV(ctx, indexPath(l), query)
	if err != nil {
		return nil, err
	}
	txns := groupTransactions(rows)
	// 行级查询无法按交易截断（一笔交易含多行分录），分组后再按交易数分页
	if f.Offset > 0 {
		if f.Offset >= len(txns) {
			return []Transaction{}, nil
		}
		txns = txns[f.Offset:]
	}
	if f.Limit > 0 && len(txns) > f.Limit {
		txns = txns[:f.Limit]
	}
	return txns, nil
}

func (s *Service) Get(ctx context.Context, l db.Ledger, id string) (*Transaction, error) {
	query := "SELECT " + transactionSelect + " WHERE id = '" + strings.ReplaceAll(id, "'", "") + "'"
	rows, err := s.Engine.QueryCSV(ctx, indexPath(l), query)
	if err != nil {
		return nil, err
	}
	grouped := groupTransactions(rows)
	if len(grouped) == 0 {
		return nil, ErrNotFound
	}
	return &grouped[0], nil
}

// Create 校验、生成 bean 文本、写月份文件，乐观并发 CAS 修订号后回查交易 id。
func (s *Service) Create(ctx context.Context, l db.Ledger, t Transaction, expectedRevision int64, actor Actor) (*Transaction, error) {
	unlock := s.lockLedger(l.ID)
	defer unlock()
	if err := Validate(t); err != nil {
		return nil, err
	}
	monthStr := t.Date[:7]
	text, priceLines, err := BuildBeanText(t, l.OperatingCurrency)
	if err != nil {
		return nil, err
	}

	// 先 CAS 修订号，再写文件，防止并发丢更新
	_, err = s.Store.CompareAndBumpRevision(ctx, l.ID, expectedRevision)
	if err != nil {
		return nil, err
	}
	if err := repository.AppendMonthTransaction(l.DataPath, monthStr, text); err != nil {
		return nil, fmt.Errorf("append month file: %w", err)
	}
	for _, line := range priceLines {
		if err := repository.AppendPrice(l.DataPath, line); err != nil {
			return nil, fmt.Errorf("append price: %w", err)
		}
	}
	detail, _ := json.Marshal(map[string]string{"date": t.Date, "payee": t.Payee, "narration": t.Narration})
	if err := s.Store.InsertAuditLog(ctx, db.AuditParams{
		LedgerID: l.ID, UserID: actor.UserID, Actor: actor.Login,
		Action: "create_transaction", Object: t.Date, Detail: string(detail),
	}); err != nil {
		return nil, fmt.Errorf("audit log: %w", err)
	}

	created := t
	created.ID = s.findTransactionID(ctx, l, t)
	return &created, nil
}

// Update 整体替换交易；跨月份时自动迁移到新月份文件。
func (s *Service) Update(ctx context.Context, l db.Ledger, id string, t Transaction, expectedRevision int64, actor Actor) (*Transaction, error) {
	unlock := s.lockLedger(l.ID)
	defer unlock()
	if err := Validate(t); err != nil {
		return nil, err
	}
	old, err := s.Get(ctx, l, id)
	if err != nil {
		return nil, err
	}
	oldText, err := s.Engine.Print(ctx, indexPath(l), id)
	if err != nil {
		return nil, fmt.Errorf("print transaction: %w", err)
	}
	text, priceLines, err := BuildBeanText(t, l.OperatingCurrency)
	if err != nil {
		return nil, err
	}
	if _, err := s.Store.CompareAndBumpRevision(ctx, l.ID, expectedRevision); err != nil {
		return nil, err
	}

	oldFile, err := s.monthFilePath(l, old.Date)
	if err != nil {
		return nil, err
	}
	oldLines := splitTextLines(oldText)
	newLines := splitTextLines(text)
	if t.Date[:7] == old.Date[:7] {
		if err := repository.ReplaceLinesByNormalizedMatch(oldFile, oldLines, newLines); err != nil {
			return nil, fmt.Errorf("replace transaction: %w", err)
		}
	} else {
		if err := repository.ReplaceLinesByNormalizedMatch(oldFile, oldLines, nil); err != nil {
			return nil, fmt.Errorf("remove old transaction: %w", err)
		}
		if err := repository.AppendMonthTransaction(l.DataPath, t.Date[:7], text); err != nil {
			return nil, fmt.Errorf("append new month: %w", err)
		}
	}
	for _, line := range priceLines {
		if err := repository.AppendPrice(l.DataPath, line); err != nil {
			return nil, fmt.Errorf("append price: %w", err)
		}
	}
	if err := s.audit(ctx, l, actor, "update_transaction", id); err != nil {
		return nil, err
	}
	created := t
	created.ID = s.findTransactionID(ctx, l, t)
	return &created, nil
}

func (s *Service) Delete(ctx context.Context, l db.Ledger, id string, expectedRevision int64, actor Actor) error {
	unlock := s.lockLedger(l.ID)
	defer unlock()
	old, err := s.Get(ctx, l, id)
	if err != nil {
		return err
	}
	oldText, err := s.Engine.Print(ctx, indexPath(l), id)
	if err != nil {
		return fmt.Errorf("print transaction: %w", err)
	}
	if _, err := s.Store.CompareAndBumpRevision(ctx, l.ID, expectedRevision); err != nil {
		return err
	}
	oldFile, err := s.monthFilePath(l, old.Date)
	if err != nil {
		return err
	}
	if err := repository.ReplaceLinesByNormalizedMatch(oldFile, splitTextLines(oldText), nil); err != nil {
		return fmt.Errorf("remove transaction: %w", err)
	}
	return s.audit(ctx, l, actor, "delete_transaction", id)
}

func (s *Service) RawText(ctx context.Context, l db.Ledger, id string) (string, error) {
	return s.Engine.Print(ctx, indexPath(l), id)
}

func (s *Service) UpdateRawText(ctx context.Context, l db.Ledger, id, raw string, expectedRevision int64, actor Actor) error {
	unlock := s.lockLedger(l.ID)
	defer unlock()
	old, err := s.Get(ctx, l, id)
	if err != nil {
		return err
	}
	oldText, err := s.Engine.Print(ctx, indexPath(l), id)
	if err != nil {
		return fmt.Errorf("print transaction: %w", err)
	}
	if _, err := s.Store.CompareAndBumpRevision(ctx, l.ID, expectedRevision); err != nil {
		return err
	}
	oldFile, err := s.monthFilePath(l, old.Date)
	if err != nil {
		return err
	}
	newLines := splitTextLines(raw)
	if err := repository.ReplaceLinesByNormalizedMatch(oldFile, splitTextLines(oldText), newLines); err != nil {
		return fmt.Errorf("replace raw text: %w", err)
	}
	return s.audit(ctx, l, actor, "update_raw_text", id)
}

func (s *Service) monthFilePath(l db.Ledger, date string) (string, error) {
	m, err := time.Parse("2006-01-02", date)
	if err != nil {
		return "", ErrInvalidDate
	}
	return filepath.Join(l.DataPath, "month", m.Format("2006-01")+".bean"), nil
}

func (s *Service) audit(ctx context.Context, l db.Ledger, actor Actor, action, object string) error {
	return s.Store.InsertAuditLog(ctx, db.AuditParams{
		LedgerID: l.ID, UserID: actor.UserID, Actor: actor.Login,
		Action: action, Object: object,
	})
}

func (s *Service) lockLedger(ledgerID string) func() {
	mu, _ := s.locks.LoadOrStore(ledgerID, &sync.Mutex{})
	lock := mu.(*sync.Mutex)
	lock.Lock()
	return lock.Unlock
}

// splitTextLines 拆分行并去除首尾空行。
func splitTextLines(s string) []string {
	lines := strings.Split(s, "\n")
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	return lines
}


// findTransactionID 通过日期与收款方/描述回查新交易的 id（bean-query 生成）。
func (s *Service) findTransactionID(ctx context.Context, l db.Ledger, t Transaction) string {
	where := "date >= " + t.Date + " AND date <= " + t.Date
	if t.Payee != "" {
		where += " AND payee = '" + strings.ReplaceAll(t.Payee, "'", "") + "'"
	}
	if t.Narration != "" {
		where += " AND narration = '" + strings.ReplaceAll(t.Narration, "'", "") + "'"
	}
	rows, err := s.Engine.QueryCSV(ctx, indexPath(l),
		"SELECT id WHERE "+where+" ORDER BY date desc")
	if err != nil || len(rows) == 0 {
		return ""
	}
	return rows[len(rows)-1]["id"]
}

func buildWhere(f Filters) string {
	wheres := make([]string, 0, 6)
	if f.From != "" {
		wheres = append(wheres, "date >= "+f.From)
	}
	if f.To != "" {
		wheres = append(wheres, "date <= "+f.To)
	}
	if f.Month != "" {
		if from, to, ok := monthRange(f.Month); ok {
			wheres = append(wheres, "date >= "+from+" AND date <= "+to)
		}
	}
	if f.Account != "" {
		wheres = append(wheres, "account = '"+f.Account+"'")
	}
	if f.AccountType != "" {
		wheres = append(wheres, "account ~ '^"+strings.ReplaceAll(f.AccountType, "'", "")+"'")
	}
	if f.Tag != "" {
		wheres = append(wheres, "'"+f.Tag+"' in tags")
	}
	if f.Q != "" {
		q := strings.ReplaceAll(f.Q, "'", "")
		wheres = append(wheres, "(payee ~ '"+q+"' OR narration ~ '"+q+"')")
	}
	if len(wheres) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(wheres, " AND ")
}

func orderBy(order string) string {
	if order != "asc" {
		return "desc"
	}
	return "asc"
}

func monthRange(month string) (string, string, bool) {
	start, err := time.Parse("2006-01", month)
	if err != nil {
		return "", "", false
	}
	from := start.Format("2006-01-02")
	to := start.AddDate(0, 1, -1).Format("2006-01-02")
	return from, to, true
}

func groupTransactions(rows []beancount.Row) []Transaction {
	byID := make(map[string]*Transaction)
	order := make([]string, 0)
	for _, r := range rows {
		id := r["id"]
		t, ok := byID[id]
		if !ok {
			t = &Transaction{
				ID:        id,
				Date:      r["date"],
				Payee:     r["payee"],
				Narration: r["narration"],
				Tags:      parseList(r["tags"]),
				Links:     parseList(r["links"]),
			}
			byID[id] = t
			order = append(order, id)
		}
		p := Posting{Account: r["account"]}
		if num := r["number"]; num != "" && num != "0" {
			p.Units = &Amount{Number: num, Currency: r["currency"]}
		}
		if cn := r["cost_number"]; cn != "" && cn != "0" {
			p.Cost = &Cost{Number: cn, Currency: r["cost_currency"], Date: r["cost_date"]}
		}
		if pr := r["price"]; pr != "" {
			fields := strings.Fields(pr)
			if len(fields) > 0 {
				p.Price = &Amount{Number: fields[0], Currency: strings.Join(fields[1:], " ")}
			}
		}
		t.Postings = append(t.Postings, p)
	}
	result := make([]Transaction, 0, len(order))
	for _, id := range order {
		result = append(result, *byID[id])
	}
	return result
}

func parseList(s string) []string {
	s = strings.Trim(s, "()[] ")
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.Trim(strings.TrimSpace(p), `"' `)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// IsNotFound 判断是否为“未找到”错误。
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound) || errors.Is(err, sql.ErrNoRows)
}
