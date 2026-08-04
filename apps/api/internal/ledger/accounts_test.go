package ledger

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/beancount-gs/api/internal/beancount"
	"github.com/beancount-gs/api/internal/db"
	"github.com/google/uuid"
)

func TestOpenAccountAndList(t *testing.T) {
	ctx := context.Background()
	store, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	user, err := store.UpsertGitHubUser(ctx, "1", "alice", "", "Alice")
	if err != nil {
		t.Fatal(err)
	}
	team, err := store.CreateTeamWithOwner(ctx, uuid.NewString(), "家庭", user.ID)
	if err != nil {
		t.Fatal(err)
	}
	dataPath := filepath.Join(t.TempDir(), "ledger")
	ledgerRow, err := store.CreateLedgerWithOwner(ctx, db.NewLedgerParams{
		ID: uuid.NewString(), TeamID: team.ID, Name: "家庭账本",
		DataPath: dataPath, OperatingCurrency: "CNY",
	}, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	actor := Actor{UserID: user.ID, Login: "alice"}
	engine := &fakeEngine{rows: []beancount.Row{
		{"account": "Assets:Cash", "market_position": "12345.00 CNY", "position": "12345.00 CNY"},
	}}
	svc := &Service{Store: store, Engine: engine}

	acc, err := svc.OpenAccount(ctx, ledgerRow, OpenAccount{
		Account: "Assets:Cash", OpenedOn: "2026-08-02", Currency: "CNY",
	}, 0, actor)
	if err != nil {
		t.Fatalf("open account: %v", err)
	}
	if acc.Status != "open" || acc.Type != "Assets" {
		t.Fatalf("unexpected account: %+v", acc)
	}
	content, err := os.ReadFile(filepath.Join(dataPath, "account", "assets.bean"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "2026-08-02 open Assets:Cash CNY") {
		t.Fatalf("open directive not written:\n%s", content)
	}

	// 重复开户 → 冲突
	if _, err := svc.OpenAccount(ctx, ledgerRow, OpenAccount{
		Account: "Assets:Cash", OpenedOn: "2026-08-02",
	}, 1, actor); !errors.Is(err, ErrDuplicateAccount) {
		t.Fatalf("duplicate should conflict, got %v", err)
	}

	// 列表（含持仓）
	accounts, err := svc.ListAccounts(ctx, ledgerRow, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) != 1 || accounts[0].Name != "Assets:Cash" {
		t.Fatalf("unexpected list: %+v", accounts)
	}
	if accounts[0].MarketNumber != "12345.00" || len(accounts[0].Positions) != 1 {
		t.Fatalf("positions missing: %+v", accounts[0])
	}

	// 关闭
	closed, err := svc.CloseAccount(ctx, ledgerRow, "Assets:Cash", "2026-09-01", 1, actor)
	if err != nil {
		t.Fatalf("close account: %v", err)
	}
	if closed.Status != "closed" {
		t.Fatalf("unexpected closed: %+v", closed)
	}
	accounts, err = svc.ListAccounts(ctx, ledgerRow, false)
	if err != nil || len(accounts) != 0 {
		t.Fatalf("closed account should be filtered: %v %d", err, len(accounts))
	}
	accounts, err = svc.ListAccounts(ctx, ledgerRow, true)
	if err != nil || len(accounts) != 1 || accounts[0].Status != "closed" {
		t.Fatalf("closed list wrong: %v %+v", err, accounts)
	}
}

func TestBatchOpenAccounts(t *testing.T) {
	ctx := context.Background()
	store, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	user, err := store.UpsertGitHubUser(ctx, "2", "bob", "", "Bob")
	if err != nil {
		t.Fatal(err)
	}
	team, err := store.CreateTeamWithOwner(ctx, uuid.NewString(), "test", user.ID)
	if err != nil {
		t.Fatal(err)
	}
	dataPath := filepath.Join(t.TempDir(), "ledger")
	ledgerRow, err := store.CreateLedgerWithOwner(ctx, db.NewLedgerParams{
		ID: uuid.NewString(), TeamID: team.ID, Name: "test", DataPath: dataPath, OperatingCurrency: "CNY",
	}, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	actor := Actor{UserID: user.ID, Login: "bob"}
	svc := &Service{Store: store, Engine: &fakeEngine{}}

	// 先开一个账户，验证批量创建会跳过已存在的
	if _, err := svc.OpenAccount(ctx, ledgerRow, OpenAccount{Account: "Assets:Cash", OpenedOn: "2026-08-01"}, 0, actor); err != nil {
		t.Fatal(err)
	}

	result, err := svc.BatchOpenAccounts(ctx, ledgerRow, []OpenAccount{
		{Account: "Assets:Cash", OpenedOn: "2026-08-02"}, // 已存在 → 跳过
		{Account: "Assets:Bank:CMB", OpenedOn: "2026-08-02", Currency: "CNY"},
		{Account: "Expenses:Food:Coffee", OpenedOn: "2026-08-02"}, // 未给日期 → 默认今天
		{Account: "Assets:Bank:CMB", OpenedOn: "2026-08-03"},      // 列表内重复 → 去重
	}, 1, actor)
	if err != nil {
		t.Fatalf("batch open: %v", err)
	}
	if len(result.Created) != 2 {
		t.Fatalf("expected 2 created, got %d (%+v)", len(result.Created), result.Created)
	}
	if len(result.Skipped) != 1 || result.Skipped[0].Account != "Assets:Cash" {
		t.Fatalf("expected 1 skipped for Assets:Cash, got %+v", result.Skipped)
	}

	content, err := os.ReadFile(filepath.Join(dataPath, "account", "assets.bean"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "2026-08-02 open Assets:Bank:CMB CNY") {
		t.Fatalf("assets directive not written:\n%s", content)
	}
	expenses, err := os.ReadFile(filepath.Join(dataPath, "account", "expenses.bean"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(expenses), " open Expenses:Food:Coffee") {
		t.Fatalf("expenses directive not written:\n%s", expenses)
	}

	// 全部已存在 → 不报错、不 bump 修订号
	all, err := svc.BatchOpenAccounts(ctx, ledgerRow, []OpenAccount{{Account: "Assets:Cash", OpenedOn: "2026-08-03"}}, 2, actor)
	if err != nil {
		t.Fatal(err)
	}
	if len(all.Created) != 0 || len(all.Skipped) != 1 {
		t.Fatalf("unexpected all-duplicate result: %+v", all)
	}

	// 非法账户名 → 报错
	if _, err := svc.BatchOpenAccounts(ctx, ledgerRow, []OpenAccount{{Account: "Bad:Name", OpenedOn: "2026-08-03"}}, 3, actor); err == nil {
		t.Fatal("expected validation error for bad prefix")
	}
}


func TestReopenAccount(t *testing.T) {
	ctx := context.Background()
	store, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	user, err := store.UpsertGitHubUser(ctx, "3", "carol", "", "Carol")
	if err != nil {
		t.Fatal(err)
	}
	team, err := store.CreateTeamWithOwner(ctx, uuid.NewString(), "test", user.ID)
	if err != nil {
		t.Fatal(err)
	}
	dataPath := filepath.Join(t.TempDir(), "ledger")
	ledgerRow, err := store.CreateLedgerWithOwner(ctx, db.NewLedgerParams{
		ID: uuid.NewString(), TeamID: team.ID, Name: "test", DataPath: dataPath, OperatingCurrency: "CNY",
	}, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	actor := Actor{UserID: user.ID, Login: "carol"}
	svc := &Service{Store: store, Engine: &fakeEngine{}}

	if _, err := svc.OpenAccount(ctx, ledgerRow, OpenAccount{
		Account: "Assets:Bank:CMB", OpenedOn: "2026-08-01", Currency: "CNY",
	}, 0, actor); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CloseAccount(ctx, ledgerRow, "Assets:Bank:CMB", "2026-09-01", 1, actor); err != nil {
		t.Fatal(err)
	}

	// 重新开户日期早于关闭日期 → 422 类错误
	if _, err := svc.ReopenAccount(ctx, ledgerRow, "Assets:Bank:CMB", "2026-08-15", 2, actor); !errors.Is(err, ErrInvalidDate) {
		t.Fatalf("reopen before close should fail, got %v", err)
	}

	// 正常重新开户
	acc, err := svc.ReopenAccount(ctx, ledgerRow, "Assets:Bank:CMB", "2026-10-01", 2, actor)
	if err != nil {
		t.Fatalf("reopen account: %v", err)
	}
	if acc.Status != "open" || acc.OpenedOn != "2026-10-01" || acc.Currency != "CNY" {
		t.Fatalf("unexpected reopened account: %+v", acc)
	}
	content, err := os.ReadFile(filepath.Join(dataPath, "account", "assets.bean"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "2026-10-01 open Assets:Bank:CMB CNY") {
		t.Fatalf("reopen directive not written:\n%s", content)
	}
	accounts, err := svc.ListAccounts(ctx, ledgerRow, false)
	if err != nil || len(accounts) != 1 || accounts[0].Status != "open" {
		t.Fatalf("reopened account should be open in list: %v %+v", err, accounts)
	}

	// 已重新开户的账户再次开户 → 冲突
	if _, err := svc.ReopenAccount(ctx, ledgerRow, "Assets:Bank:CMB", "2026-11-01", 3, actor); !errors.Is(err, ErrAccountNotClosed) {
		t.Fatalf("reopen open account should fail, got %v", err)
	}
}
