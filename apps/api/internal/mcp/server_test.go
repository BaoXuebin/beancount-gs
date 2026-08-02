package mcp

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/beancount-gs/api/internal/beancount"
	"github.com/beancount-gs/api/internal/db"
	"github.com/beancount-gs/api/internal/ledger"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
)

type fakeEngine struct {
	rows []beancount.Row
}

func (f *fakeEngine) QueryCSV(_ context.Context, _, _ string) ([]beancount.Row, error) {
	return f.rows, nil
}
func (f *fakeEngine) Print(_ context.Context, _, _ string) (string, error) { return "", nil }
func (f *fakeEngine) Check(_ context.Context, _ string) ([]string, error)  { return nil, nil }

func setup(t *testing.T) (*Server, db.User, db.Ledger) {
	t.Helper()
	ctx := context.Background()
	store, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	user, err := store.UpsertGitHubUser(ctx, "1", "alice", "", "Alice")
	if err != nil {
		t.Fatal(err)
	}
	team, err := store.CreateTeamWithOwner(ctx, uuid.NewString(), "家庭", user.ID)
	if err != nil {
		t.Fatal(err)
	}
	ledgerRow, err := store.CreateLedgerWithOwner(ctx, db.NewLedgerParams{
		ID: uuid.NewString(), TeamID: team.ID, Name: "家庭账本",
		DataPath: filepath.Join(t.TempDir(), "ledger"), OperatingCurrency: "CNY",
	}, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	engine := &fakeEngine{rows: []beancount.Row{
		{"id": "txn-mcp", "date": "2026-08-02", "payee": "盒马鲜生", "narration": "日常采购"},
	}}
	svc := &ledger.Service{Store: store, Engine: engine}
	return New(store, svc, t.TempDir()), user, ledgerRow
}

func TestListLedgersTool(t *testing.T) {
	s, user, _ := setup(t)
	ctx := WithAuthUser(context.Background(), user, "read-only")
	data, err := s.listLedgers(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	ledgers, ok := data.([]db.Ledger)
	if !ok || len(ledgers) != 1 {
		t.Fatalf("unexpected ledgers: %+v", data)
	}
}

func TestCreateTransactionTool(t *testing.T) {
	s, user, ledgerRow := setup(t)
	ctx := WithAuthUser(context.Background(), user, "read-write")
	args := map[string]any{
		"ledger_id": ledgerRow.ID,
		"date":      "2026-08-02",
		"payee":     "盒马鲜生",
		"narration": "日常采购",
		"postings":  `[{"account":"Expenses:Food","units":{"number":"-120.00","currency":"CNY"}},{"account":"Assets:Cash","units":{"number":"120.00","currency":"CNY"}}]`,
	}
	created, err := s.createTransaction(ctx, args)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	txn, ok := created.(*ledger.Transaction)
	if !ok || txn.ID != "txn-mcp" || txn.Narration != "日常采购" {
		t.Fatalf("unexpected created: %+v", created)
	}
}

func TestScopeDenied(t *testing.T) {
	s, user, _ := setup(t)
	ctx := WithAuthUser(context.Background(), user, "read-only")
	handler := s.mcpHandler(s.listLedgers, "write")
	result, _ := handler(ctx, mcp.CallToolRequest{})
	if result == nil || !result.IsError {
		t.Fatalf("read-only key should be denied for write tool")
	}
}

func TestSafeJoin(t *testing.T) {
	if _, err := safeJoin(t.TempDir(), "../secret"); err == nil {
		t.Fatal("path traversal should be rejected")
	}
	if _, err := safeJoin(t.TempDir(), "month/2026-08.bean"); err != nil {
		t.Fatalf("valid path rejected: %v", err)
	}
}
