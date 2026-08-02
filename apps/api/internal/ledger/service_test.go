package ledger

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/beancount-gs/api/internal/beancount"
	"github.com/beancount-gs/api/internal/db"
	"github.com/google/uuid"
)

type fakeEngine struct {
	rows []beancount.Row
	err  error
}

func (f *fakeEngine) QueryCSV(_ context.Context, _, _ string) ([]beancount.Row, error) {
	return f.rows, f.err
}
func (f *fakeEngine) Print(_ context.Context, _, _ string) (string, error) { return "", nil }
func (f *fakeEngine) Check(_ context.Context, _ string) ([]string, error)  { return nil, nil }

func TestGroupTransactions(t *testing.T) {
	rows := []beancount.Row{
		{"id": "txn-1", "date": "2026-08-02", "payee": "盒马鲜生", "narration": "日常采购",
			"tags": "('Food', 'Family')", "account": "Expenses:Food", "number": "-120.00", "currency": "CNY",
			"cost_number": "", "cost_currency": "", "cost_date": "", "price": ""},
		{"id": "txn-1", "date": "2026-08-02", "payee": "盒马鲜生", "narration": "日常采购",
			"tags": "('Food', 'Family')", "account": "Assets:Cash", "number": "120.00", "currency": "CNY",
			"cost_number": "5.61", "cost_currency": "CNY", "cost_date": "2026-08-02", "price": "7.1850 USD"},
	}
	txns := groupTransactions(rows)
	if len(txns) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(txns))
	}
	txn := txns[0]
	if len(txn.Postings) != 2 {
		t.Fatalf("expected 2 postings, got %d", len(txn.Postings))
	}
	if len(txn.Tags) != 2 || txn.Tags[0] != "Food" {
		t.Fatalf("tags parse failed: %v", txn.Tags)
	}
	if txn.Postings[1].Cost == nil || txn.Postings[1].Cost.Number != "5.61" {
		t.Fatalf("cost parse failed: %+v", txn.Postings[1].Cost)
	}
	if txn.Postings[1].Price == nil || txn.Postings[1].Price.Number != "7.1850" {
		t.Fatalf("price parse failed: %+v", txn.Postings[1].Price)
	}
}

func TestValidate(t *testing.T) {
	balanced := Transaction{
		Date: "2026-08-02",
		Postings: []Posting{
			{Account: "Expenses:Food", Units: &Amount{Number: "-120.00", Currency: "CNY"}},
			{Account: "Assets:Cash", Units: &Amount{Number: "120.00", Currency: "CNY"}},
		},
	}
	if err := Validate(balanced); err != nil {
		t.Fatalf("balanced should pass: %v", err)
	}
	unbalanced := balanced
	unbalanced.Postings[1].Units.Number = "100.00"
	if !errors.Is(Validate(unbalanced), ErrNotBalanced) {
		t.Fatal("unbalanced should fail")
	}
	short := balanced
	short.Postings = short.Postings[:1]
	if !errors.Is(Validate(short), ErrNoPostings) {
		t.Fatal("single posting should fail")
	}
}

func TestBuildBeanText(t *testing.T) {
	txn := Transaction{
		Date:      "2026-08-02",
		Payee:     "Starbucks",
		Narration: "咖啡",
		Tags:      []string{"Daily"},
		Postings: []Posting{
			{Account: "Expenses:Food:Coffee", Units: &Amount{Number: "-38.00", Currency: "CNY"}},
			{Account: "Assets:Cash", Units: &Amount{Number: "38.00", Currency: "CNY"}},
		},
	}
	text, prices, err := BuildBeanText(txn, "CNY")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`2026-08-02 * "Starbucks" "咖啡" #Daily`,
		"  Expenses:Food:Coffee  -38.00 CNY",
		"  Assets:Cash  38.00 CNY",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("bean text missing %q:\n%s", want, text)
		}
	}
	if len(prices) != 0 {
		t.Fatalf("CNY txn should not produce price lines: %v", prices)
	}

	// 外币 + 汇率
	foreign := Transaction{
		Date:  "2026-08-02",
		Payee: "Apple",
		Postings: []Posting{
			{Account: "Assets:Stock", Units: &Amount{Number: "10", Currency: "USD"}, Price: &Amount{Number: "7.185", Currency: "CNY"}},
			{Account: "Assets:Cash", Units: &Amount{Number: "-71.85", Currency: "CNY"}},
		},
	}
	text, prices, err = BuildBeanText(foreign, "CNY")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, " @ 7.19 CNY") {
		t.Errorf("price not rendered:\n%s", text)
	}
	if len(prices) != 1 || !strings.Contains(prices[0], "2026-08-02 price USD 7.19 CNY") {
		t.Fatalf("price lines unexpected: %v", prices)
	}
}

func TestCreateWritesMonthFile(t *testing.T) {
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

	svc := &Service{
		Store: store,
		Engine: &fakeEngine{rows: []beancount.Row{
			{"id": "txn-abc", "date": "2026-08-02", "payee": "盒马鲜生", "narration": "日常采购"},
		}},
		Now: time.Now,
	}
	created, err := svc.Create(ctx, ledgerRow, Transaction{
		Date:      "2026-08-02",
		Payee:     "盒马鲜生",
		Narration: "日常采购",
		Tags:      []string{"Food"},
		Postings: []Posting{
			{Account: "Expenses:Food", Units: &Amount{Number: "-120.00", Currency: "CNY"}},
			{Account: "Assets:Cash", Units: &Amount{Number: "120.00", Currency: "CNY"}},
		},
	}, 0, Actor{UserID: user.ID, Login: "alice"})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != "txn-abc" {
		t.Fatalf("unexpected created id: %s", created.ID)
	}
	content, err := os.ReadFile(filepath.Join(dataPath, "month", "2026-08.bean"))
	if err != nil {
		t.Fatalf("month file missing: %v", err)
	}
	if !strings.Contains(string(content), "2026-08-02 * \"盒马鲜生\" \"日常采购\" #Food") {
		t.Fatalf("month file content wrong:\n%s", content)
	}
	got, err := store.GetLedger(ctx, ledgerRow.ID)
	if err != nil || got.Revision != 1 {
		t.Fatalf("revision should be 1: %v %+v", err, got)
	}
	logs, err := store.ListAuditLogs(ctx, ledgerRow.ID, 10)
	if err != nil || len(logs) != 1 || logs[0].Action != "create_transaction" {
		t.Fatalf("audit log missing: %v %+v", err, logs)
	}
}

func TestStaleRevisionFails(t *testing.T) {
	ctx := context.Background()
	store, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	user, _ := store.UpsertGitHubUser(ctx, "1", "alice", "", "Alice")
	team, _ := store.CreateTeamWithOwner(ctx, uuid.NewString(), "家庭", user.ID)
	ledgerRow, _ := store.CreateLedgerWithOwner(ctx, db.NewLedgerParams{
		ID: uuid.NewString(), TeamID: team.ID, Name: "账本",
		DataPath: filepath.Join(t.TempDir(), "ledger"), OperatingCurrency: "CNY",
	}, user.ID)
	svc := &Service{Store: store, Engine: &fakeEngine{}, Now: time.Now}
	txn := Transaction{
		Date: "2026-08-02",
		Postings: []Posting{
			{Account: "Expenses:Food", Units: &Amount{Number: "-10", Currency: "CNY"}},
			{Account: "Assets:Cash", Units: &Amount{Number: "10", Currency: "CNY"}},
		},
	}
	if _, err := svc.Create(ctx, ledgerRow, txn, 5, Actor{UserID: user.ID}); !errors.Is(err, db.ErrRevisionConflict) {
		t.Fatalf("stale revision should conflict, got %v", err)
	}
}
