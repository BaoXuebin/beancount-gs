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
