package ledger

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/beancount-gs/api/internal/beancount"
	"github.com/beancount-gs/api/internal/db"
	"github.com/beancount-gs/api/internal/repository"
	"github.com/google/uuid"
)

// TestBeanQueryIntegration 使用真实 bean-query 验证「写入 → 查询」全链路。
func TestBeanQueryIntegration(t *testing.T) {
	if _, err := exec.LookPath("bean-query"); err != nil {
		t.Skip("bean-query not installed")
	}
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
	if err := repository.InitLedgerFiles(dataPath, findTemplateDir(t), "2026-08-02", "CNY"); err != nil {
		t.Fatalf("init ledger files: %v", err)
	}
	ledgerRow, err := store.CreateLedgerWithOwner(ctx, db.NewLedgerParams{
		ID: uuid.NewString(), TeamID: team.ID, Name: "家庭账本",
		DataPath: dataPath, OperatingCurrency: "CNY",
	}, user.ID)
	if err != nil {
		t.Fatal(err)
	}

	svc := &Service{Store: store, Engine: beancount.CmdEngine{}}
	txn := Transaction{
		Date:      "2026-08-02",
		Payee:     "盒马鲜生",
		Narration: "日常采购",
		Tags:      []string{"Food"},
		Postings: []Posting{
			{Account: "Expenses:Food", Units: &Amount{Number: "-120.00", Currency: "CNY"}},
			{Account: "Assets:Cash", Units: &Amount{Number: "120.00", Currency: "CNY"}},
		},
	}
	created, err := svc.Create(ctx, ledgerRow, txn, 0, Actor{UserID: user.ID, Login: "alice"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID == "" {
		t.Fatal("bean-query did not return a transaction id")
	}

	got, err := svc.Get(ctx, ledgerRow, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Narration != "日常采购" || len(got.Postings) != 2 {
		t.Fatalf("unexpected transaction: %+v", got)
	}
	if got.Postings[0].Units == nil || got.Postings[0].Units.Number != "-120.00" {
		t.Fatalf("posting units wrong: %+v", got.Postings[0])
	}

	list, err := svc.List(ctx, ledgerRow, Filters{Month: "2026-08"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(list))
	}
}

// findTemplateDir 从测试工作目录向上查找仓库根目录的 template 目录。
func findTemplateDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "template", "index.bean")); err == nil {
			return filepath.Join(dir, "template")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("template directory not found")
		}
		dir = parent
	}
}
