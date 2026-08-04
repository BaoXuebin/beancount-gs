package ledger

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/beancount-gs/api/internal/db"
	"github.com/beancount-gs/api/internal/repository"
)

var accountTypes = []string{"Assets", "Liabilities", "Income", "Expenses", "Equity"}

type Account struct {
	Name          string
	Type          string
	Status        string // open | closed
	OpenedOn      string
	ClosedOn      string
	Currency      string
	Positions     []Position
	MarketNumber  string
	MarketCurrency string
}

type Position struct {
	Number         string
	Currency       string
	CurrencySymbol string
}

type AccountType struct {
	Prefix string
	Name   string
}

type OpenAccount struct {
	Account  string
	OpenedOn string
	Currency string
	Booking  string
}

var positionRe = regexp.MustCompile(`(-?\d+(?:\.\d+)?) ([A-Za-z0-9.]+)`)

func (s *Service) ListAccounts(ctx context.Context, l db.Ledger, includeClosed bool) ([]Account, error) {
	directives, err := repository.ReadAccountFiles(l.DataPath)
	if err != nil {
		return nil, err
	}
	positions, err := s.accountPositions(ctx, l)
	if err != nil {
		return nil, err
	}
	byName := make(map[string]*Account)
	for _, d := range directives {
		acc, ok := byName[d.Account]
		if !ok {
			acc = &Account{
				Name: d.Account,
				Type: accountPrefix(d.Account),
			}
			byName[d.Account] = acc
		}
		if d.Kind == "open" {
			if acc.OpenedOn == "" || d.Date > acc.OpenedOn {
				acc.OpenedOn = d.Date
			}
			if d.Currency != "" {
				acc.Currency = d.Currency
			}
			acc.ClosedOn = ""
			acc.Status = "open"
		} else if d.Kind == "close" {
			acc.ClosedOn = d.Date
			acc.Status = "closed"
		}
	}
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]Account, 0, len(names))
	for _, name := range names {
		acc := byName[name]
		if !includeClosed && acc.Status == "closed" {
			continue
		}
		if p, ok := positions[name]; ok {
			acc.Positions = p.Positions
			acc.MarketNumber = p.MarketNumber
			acc.MarketCurrency = p.MarketCurrency
		}
		result = append(result, *acc)
	}
	return result, nil
}

func (s *Service) GetAccount(ctx context.Context, l db.Ledger, name string) (*Account, error) {
	accounts, err := s.ListAccounts(ctx, l, true)
	if err != nil {
		return nil, err
	}
	for i := range accounts {
		if accounts[i].Name == name {
			return &accounts[i], nil
		}
	}
	return nil, ErrNotFound
}

func (s *Service) OpenAccount(ctx context.Context, l db.Ledger, form OpenAccount, expectedRevision int64, actor Actor) (*Account, error) {
	unlock := s.lockLedger(l.ID)
	defer unlock()
	if err := validateAccountName(form.Account); err != nil {
		return nil, err
	}
	if _, err := time.Parse("2006-01-02", form.OpenedOn); err != nil {
		return nil, ErrInvalidDate
	}
	directives, err := repository.ReadAccountFiles(l.DataPath)
	if err != nil {
		return nil, err
	}
	for _, d := range directives {
		if d.Account == form.Account {
			return nil, ErrDuplicateAccount
		}
	}
	line := form.OpenedOn + " open " + form.Account
	if form.Currency != "" {
		line += " " + form.Currency
	}
	if form.Booking == "fifo" || (form.Currency != "" && form.Currency != l.OperatingCurrency) {
		line += ` "FIFO"`
	}
	if _, err := s.Store.CompareAndBumpRevision(ctx, l.ID, expectedRevision); err != nil {
		return nil, err
	}
	if err := repository.AppendAccountDirective(l.DataPath, accountPrefix(form.Account), line); err != nil {
		return nil, err
	}
	if err := s.audit(ctx, l, actor, "open_account", form.Account); err != nil {
		return nil, err
	}
	return &Account{
		Name: form.Account, Type: accountPrefix(form.Account),
		Status: "open", OpenedOn: form.OpenedOn, Currency: form.Currency,
	}, nil
}

type SkippedAccount struct {
	Account string
	Reason  string
}

type BatchOpenResult struct {
	Created []Account
	Skipped []SkippedAccount
}

// BatchOpenAccounts 批量开户：校验全部表单后一次写入多个 open 指令，修订号只 bump 一次。
// 已存在的账户跳过并记录原因，不视为错误。
func (s *Service) BatchOpenAccounts(ctx context.Context, l db.Ledger, forms []OpenAccount, expectedRevision int64, actor Actor) (BatchOpenResult, error) {
	if len(forms) == 0 {
		return BatchOpenResult{}, errors.New("no accounts to open")
	}
	unlock := s.lockLedger(l.ID)
	defer unlock()

	// 校验并去重
	seen := make(map[string]bool)
	unique := make([]OpenAccount, 0, len(forms))
	for _, form := range forms {
		form.Account = strings.TrimSpace(form.Account)
		if form.OpenedOn == "" {
			form.OpenedOn = time.Now().Format("2006-01-02")
		}
		if err := validateAccountName(form.Account); err != nil {
			return BatchOpenResult{}, err
		}
		if _, err := time.Parse("2006-01-02", form.OpenedOn); err != nil {
			return BatchOpenResult{}, ErrInvalidDate
		}
		if seen[form.Account] {
			continue
		}
		seen[form.Account] = true
		unique = append(unique, form)
	}

	directives, err := repository.ReadAccountFiles(l.DataPath)
	if err != nil {
		return BatchOpenResult{}, err
	}
	existing := make(map[string]bool)
	for _, d := range directives {
		existing[d.Account] = true
	}

	var result BatchOpenResult
	items := make([]repository.AccountDirectiveLine, 0, len(unique))
	createdNames := make([]string, 0, len(unique))
	for _, form := range unique {
		if existing[form.Account] {
			result.Skipped = append(result.Skipped, SkippedAccount{Account: form.Account, Reason: "账户已存在"})
			continue
		}
		line := form.OpenedOn + " open " + form.Account
		if form.Currency != "" {
			line += " " + form.Currency
		}
		if form.Booking == "fifo" || (form.Currency != "" && form.Currency != l.OperatingCurrency) {
			line += ` "FIFO"`
		}
		items = append(items, repository.AccountDirectiveLine{Prefix: accountPrefix(form.Account), Line: line})
		result.Created = append(result.Created, Account{
			Name: form.Account, Type: accountPrefix(form.Account),
			Status: "open", OpenedOn: form.OpenedOn, Currency: form.Currency,
		})
		createdNames = append(createdNames, form.Account)
	}

	if len(items) == 0 {
		return result, nil
	}
	if _, err := s.Store.CompareAndBumpRevision(ctx, l.ID, expectedRevision); err != nil {
		return BatchOpenResult{}, err
	}
	if err := repository.AppendAccountDirectiveBatch(l.DataPath, items); err != nil {
		return BatchOpenResult{}, err
	}
	if err := s.audit(ctx, l, actor, "open_account_batch", strings.Join(createdNames, ", ")); err != nil {
		return BatchOpenResult{}, err
	}
	return result, nil
}
func (s *Service) CloseAccount(ctx context.Context, l db.Ledger, name, closedOn string, expectedRevision int64, actor Actor) (*Account, error) {
	unlock := s.lockLedger(l.ID)
	defer unlock()
	if _, err := time.Parse("2006-01-02", closedOn); err != nil {
		return nil, ErrInvalidDate
	}
	line := closedOn + " close " + name
	if _, err := s.Store.CompareAndBumpRevision(ctx, l.ID, expectedRevision); err != nil {
		return nil, err
	}
	if err := repository.AppendAccountDirective(l.DataPath, accountPrefix(name), line); err != nil {
		return nil, err
	}
	if err := s.audit(ctx, l, actor, "close_account", name); err != nil {
		return nil, err
	}
	return &Account{Name: name, Type: accountPrefix(name), Status: "closed", ClosedOn: closedOn}, nil
}

// ReopenAccount 重新开户：为已关闭的账户追加一条 open 指令恢复使用。
// 校验账户存在且处于关闭状态，且新开户日期不早于最后关闭日期。
func (s *Service) ReopenAccount(ctx context.Context, l db.Ledger, name, openedOn string, expectedRevision int64, actor Actor) (*Account, error) {
	unlock := s.lockLedger(l.ID)
	defer unlock()
	if _, err := time.Parse("2006-01-02", openedOn); err != nil {
		return nil, ErrInvalidDate
	}
	directives, err := repository.ReadAccountFiles(l.DataPath)
	if err != nil {
		return nil, err
	}
	found := false
	lastKind := ""
	lastDate := ""
	lastClose := ""
	currency := ""
	for _, d := range directives {
		if d.Account != name {
			continue
		}
		found = true
		if d.Date >= lastDate {
			lastDate = d.Date
			lastKind = d.Kind
		}
		if d.Kind == "close" && d.Date > lastClose {
			lastClose = d.Date
		}
		if d.Kind == "open" && d.Currency != "" {
			currency = d.Currency
		}
	}
	if !found {
		return nil, ErrNotFound
	}
	if lastKind != "close" {
		return nil, ErrAccountNotClosed
	}
	if openedOn < lastClose {
		return nil, fmt.Errorf("%w: 重新开户日期不能早于关闭日期 %s", ErrInvalidDate, lastClose)
	}
	line := openedOn + " open " + name
	if currency != "" {
		line += " " + currency
	}
	if _, err := s.Store.CompareAndBumpRevision(ctx, l.ID, expectedRevision); err != nil {
		return nil, err
	}
	if err := repository.AppendAccountDirective(l.DataPath, accountPrefix(name), line); err != nil {
		return nil, err
	}
	if err := s.audit(ctx, l, actor, "reopen_account", name); err != nil {
		return nil, err
	}
	return &Account{Name: name, Type: accountPrefix(name), Status: "open", OpenedOn: openedOn, Currency: currency}, nil
}

func (s *Service) BalanceAccount(ctx context.Context, l db.Ledger, name, date, number string, expectedRevision int64, actor Actor) error {
	unlock := s.lockLedger(l.ID)
	defer unlock()
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return ErrInvalidDate
	}
	month := date[:7]
	text := date + " pad " + name + " Equity:OpeningBalances\n" +
		date + " balance " + name + " " + number + " " + l.OperatingCurrency
	if _, err := s.Store.CompareAndBumpRevision(ctx, l.ID, expectedRevision); err != nil {
		return err
	}
	if err := repository.AppendMonthTransaction(l.DataPath, month, text); err != nil {
		return err
	}
	return s.audit(ctx, l, actor, "balance_account", name)
}

func (s *Service) ListAccountTypes(ctx context.Context, l db.Ledger) ([]AccountType, error) {
	rows, err := s.Store.ListAccountTypes(ctx, l.ID)
	if err != nil {
		return nil, err
	}
	result := make([]AccountType, 0, len(rows))
	for _, r := range rows {
		result = append(result, AccountType{Prefix: r.Prefix, Name: r.Name})
	}
	return result, nil
}

func (s *Service) UpsertAccountType(ctx context.Context, l db.Ledger, t AccountType, expectedRevision int64, actor Actor) error {
	unlock := s.lockLedger(l.ID)
	defer unlock()
	if t.Prefix == "" || t.Name == "" {
		return errors.New("prefix and name are required")
	}
	if _, err := s.Store.CompareAndBumpRevision(ctx, l.ID, expectedRevision); err != nil {
		return err
	}
	if err := s.Store.UpsertAccountType(ctx, l.ID, t.Prefix, t.Name); err != nil {
		return err
	}
	return s.audit(ctx, l, actor, "upsert_account_type", t.Prefix)
}

type accountPosition struct {
	Positions     []Position
	MarketNumber  string
	MarketCurrency string
}

func (s *Service) accountPositions(ctx context.Context, l db.Ledger) (map[string]accountPosition, error) {
	query := "SELECT account, sum(convert(value(position), '" + l.OperatingCurrency +
		"')) AS market_position, sum(convert(value(position), currency)) AS position"
	rows, err := s.Engine.QueryCSV(ctx, indexPath(l), query)
	if err != nil {
		return nil, fmt.Errorf("query account positions: %w", err)
	}
	result := make(map[string]accountPosition)
	for _, r := range rows {
		account := r["account"]
		if account == "" {
			continue
		}
		ap := result[account]
		if market := strings.TrimSpace(r["market_position"]); market != "" {
			fields := strings.Fields(market)
			if len(fields) >= 2 {
				ap.MarketNumber = fields[0]
				ap.MarketCurrency = fields[1]
			}
		}
		if position := strings.TrimSpace(r["position"]); position != "" {
			ap.Positions = parsePositions(position)
		}
		result[account] = ap
	}
	return result, nil
}

func parsePositions(s string) []Position {
	matches := positionRe.FindAllStringSubmatch(s, -1)
	positions := make([]Position, 0, len(matches))
	for _, m := range matches {
		positions = append(positions, Position{Number: m[1], Currency: m[2]})
	}
	return positions
}

func accountPrefix(account string) string {
	nodes := strings.Split(account, ":")
	return nodes[0]
}

func validateAccountName(account string) error {
	nodes := strings.Split(account, ":")
	if len(nodes) < 2 {
		return errors.New("account must contain at least two segments, e.g. Assets:Cash")
	}
	valid := false
	for _, t := range accountTypes {
		if nodes[0] == t {
			valid = true
			break
		}
	}
	if !valid {
		return errors.New("account must start with Assets / Liabilities / Income / Expenses / Equity")
	}
	return nil
}
