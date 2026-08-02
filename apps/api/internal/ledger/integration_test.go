package ledger

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	// 更新（bean-query 内容哈希变化 → id 变化）
	updatedTxn := Transaction{
		Date: "2026-08-02", Payee: "盒马鲜生", Narration: "日常采购（修正）",
		Tags: []string{"Food"},
		Postings: []Posting{
			{Account: "Expenses:Food", Units: &Amount{Number: "-130.00", Currency: "CNY"}},
			{Account: "Assets:Cash", Units: &Amount{Number: "130.00", Currency: "CNY"}},
		},
	}
	updated, err := svc.Update(ctx, ledgerRow, created.ID, updatedTxn, 1, Actor{UserID: user.ID, Login: "alice"})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.ID == "" || updated.ID == created.ID {
		t.Fatalf("update should change id: old=%s new=%s", created.ID, updated.ID)
	}
	got2, err := svc.Get(ctx, ledgerRow, updated.ID)
	if err != nil || got2.Narration != "日常采购（修正）" || got2.Postings[0].Units.Number != "-130.00" {
		t.Fatalf("updated transaction wrong: %v %+v", err, got2)
	}

	// 原始文本读取与替换
	raw, err := svc.RawText(ctx, ledgerRow, updated.ID)
	if err != nil || !strings.Contains(raw, "日常采购（修正）") {
		t.Fatalf("raw text: %v %q", err, raw)
	}
	rawText := "2026-08-02 * \"盒马鲜生\" \"早餐\"\n  Expenses:Food  -15.00 CNY\n  Assets:Cash  15.00 CNY"
	if err := svc.UpdateRawText(ctx, ledgerRow, updated.ID, rawText, 2, Actor{UserID: user.ID, Login: "alice"}); err != nil {
		t.Fatalf("update raw: %v", err)
	}
	list2, err := svc.List(ctx, ledgerRow, Filters{Month: "2026-08"})
	if err != nil || len(list2) != 1 {
		t.Fatalf("list after raw: %v %d", err, len(list2))
	}
	afterRaw, err := svc.Get(ctx, ledgerRow, list2[0].ID)
	if err != nil || afterRaw.Narration != "早餐" {
		t.Fatalf("raw update not applied: %v %+v", err, afterRaw)
	}

	// 删除
	if err := svc.Delete(ctx, ledgerRow, list2[0].ID, 3, Actor{UserID: user.ID, Login: "alice"}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	list3, err := svc.List(ctx, ledgerRow, Filters{Month: "2026-08"})
	if err != nil || len(list3) != 0 {
		t.Fatalf("list after delete: %v %d", err, len(list3))
	}

	// 账户：开户 → 列表（含持仓）→ 关闭
	if _, err := svc.OpenAccount(ctx, ledgerRow, OpenAccount{
		Account: "Assets:Cash", OpenedOn: "2026-08-02", Currency: "CNY",
	}, 4, Actor{UserID: user.ID, Login: "alice"}); err != nil {
		t.Fatalf("open account: %v", err)
	}
	accounts, err := svc.ListAccounts(ctx, ledgerRow, false)
	if err != nil {
		t.Fatalf("list accounts: %v", err)
	}
	openFound := false
	for _, a := range accounts {
		if a.Name == "Assets:Cash" {
			openFound = true
			if a.Status != "open" {
				t.Fatalf("Assets:Cash should be open: %+v", a)
			}
		}
	}
	if !openFound {
		t.Fatalf("Assets:Cash not in open accounts")
	}
	closed, err := svc.CloseAccount(ctx, ledgerRow, "Assets:Cash", "2026-09-01", 5, Actor{UserID: user.ID, Login: "alice"})
	if err != nil || closed.Status != "closed" {
		t.Fatalf("close account: %v %+v", err, closed)
	}
	closedList, err := svc.ListAccounts(ctx, ledgerRow, true)
	if err != nil {
		t.Fatal(err)
	}
	closedFound := false
	for _, a := range closedList {
		if a.Name == "Assets:Cash" {
			closedFound = true
			if a.Status != "closed" {
				t.Fatalf("Assets:Cash should be closed: %+v", a)
			}
		}
	}
	if !closedFound {
		t.Fatalf("Assets:Cash not in closed accounts")
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
