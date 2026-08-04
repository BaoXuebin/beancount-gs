package ledger

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/beancount-gs/api/internal/db"
	"github.com/beancount-gs/api/internal/repository"
	"github.com/google/uuid"
)

func newTestLedger(t *testing.T, operating string) (context.Context, db.Ledger, Actor, *Service) {
	t.Helper()
	ctx := context.Background()
	store, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	user, err := store.UpsertGitHubUser(ctx, "cur", "cur", "", "Cur")
	if err != nil {
		t.Fatal(err)
	}
	team, err := store.CreateTeamWithOwner(ctx, uuid.NewString(), "test", user.ID)
	if err != nil {
		t.Fatal(err)
	}
	dataPath := filepath.Join(t.TempDir(), "ledger")
	if err := os.MkdirAll(filepath.Join(dataPath, ".beancount-gs"), 0o755); err != nil {
		t.Fatal(err)
	}
	ledgerRow, err := store.CreateLedgerWithOwner(ctx, db.NewLedgerParams{
		ID: uuid.NewString(), TeamID: team.ID, Name: "test",
		DataPath: dataPath, OperatingCurrency: operating,
	}, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	svc := &Service{Store: store, Engine: &fakeEngine{}}
	return ctx, ledgerRow, Actor{UserID: user.ID, Login: "cur"}, svc
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestListCurrencies(t *testing.T) {
	ctx, ledgerRow, _, svc := newTestLedger(t, "CNY")
	writeTestFile(t, filepath.Join(ledgerRow.DataPath, ".beancount-gs", "currency.json"),
		`[{"name":"美元","currency":"USD","symbol":"$"},{"name":"京东京豆","currency":"JDB","symbol":"JDB¥"}]` + "\n")
	writeTestFile(t, filepath.Join(ledgerRow.DataPath, "price", "prices.bean"),
		`2026-01-01 price USD 7.10 CNY` + "\n" + `2026-08-01 price USD 7.20 CNY` + "\n" + `2026-08-01 price JDB 0.01 CNY` + "\n")
	if err := repository.AppendAccountDirective(ledgerRow.DataPath, "Assets", "2026-08-01 open Assets:Bank:CMB CNY,USD"); err != nil {
		t.Fatal(err)
	}

	list, err := svc.ListCurrencies(ctx, ledgerRow)
	if err != nil {
		t.Fatal(err)
	}
	byCode := map[string]Currency{}
	for _, c := range list {
		byCode[c.Code] = c
	}
	if len(byCode) != 3 {
		t.Fatalf("expected 3 currencies, got %d: %+v", len(byCode), list)
	}
	if !byCode["CNY"].IsOperating {
		t.Fatalf("CNY should be operating: %+v", byCode["CNY"])
	}
	if byCode["USD"].Name != "美元" || byCode["USD"].Price != "7.20" || byCode["USD"].PriceDate != "2026-08-01" {
		t.Fatalf("USD unexpected: %+v", byCode["USD"])
	}
	if byCode["JDB"].Symbol != "JDB¥" || byCode["JDB"].Price != "0.01" {
		t.Fatalf("JDB unexpected: %+v", byCode["JDB"])
	}
}

func TestAddCurrency(t *testing.T) {
	ctx, ledgerRow, actor, svc := newTestLedger(t, "CNY")
	if err := svc.AddCurrency(ctx, ledgerRow, "eur", "欧元", "€", 0, actor); err != nil {
		t.Fatal(err)
	}
	meta, err := readCurrencyMeta(ledgerRow.DataPath)
	if err != nil {
		t.Fatal(err)
	}
	if meta["EUR"].Name != "欧元" || meta["EUR"].Symbol != "€" {
		t.Fatalf("unexpected meta: %+v", meta["EUR"])
	}
	if err := svc.AddCurrency(ctx, ledgerRow, "BAD CODE!", "x", "", 1, actor); err == nil {
		t.Fatal("invalid code should fail")
	}
}

func TestSyncCurrencies(t *testing.T) {
	ctx, ledgerRow, actor, svc := newTestLedger(t, "CNY")
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.Local)
	svc.Now = func() time.Time { return now }
	svc.FX = func(ctx context.Context, base string) (map[string]float64, error) {
		return map[string]float64{"USD": 7.25, "JPY": 0.049}, nil
	}
	writeTestFile(t, filepath.Join(ledgerRow.DataPath, ".beancount-gs", "currency.json"),
		`[{"name":"美元","currency":"USD","symbol":"$"}]` + "\n")

	// 首次同步：USD/JPY 均写入
	list, err := svc.SyncCurrencies(ctx, ledgerRow, 0, actor)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(ledgerRow.DataPath, "price", "prices.bean"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "2026-08-04 price USD 7.250000 CNY") {
		t.Fatalf("USD price line not written:\n%s", content)
	}
	if strings.Contains(string(content), "JPY") {
		t.Fatalf("unknown currency JPY should not be synced:\n%s", content)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 currencies after sync, got %d", len(list))
	}

	// 同日同价再次同步：不重复写入
	_, err = svc.SyncCurrencies(ctx, ledgerRow, 1, actor)
	if err != nil {
		t.Fatal(err)
	}
	content2, err := os.ReadFile(filepath.Join(ledgerRow.DataPath, "price", "prices.bean"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(content2), "price USD") != 1 {
		t.Fatalf("duplicate price written:\n%s", content2)
	}
}

func TestListCurrenciesNoFiles(t *testing.T) {
	ctx, ledgerRow, _, svc := newTestLedger(t, "CNY")
	list, err := svc.ListCurrencies(ctx, ledgerRow)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Code != "CNY" || !list[0].IsOperating {
		t.Fatalf("expected only operating currency: %+v", list)
	}
}
