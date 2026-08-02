package ledger

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/beancount-gs/api/internal/beancount"
	"github.com/beancount-gs/api/internal/db"
)

func TestAiInsightsDuplicate(t *testing.T) {
	ctx := context.Background()
	rows := []beancount.Row{
		{"id": "t1", "date": "2026-08-01", "payee": "盒马鲜生", "narration": "a",
			"account": "Expenses:Food", "number": "-120.00", "currency": "CNY",
			"cost_number": "", "cost_currency": "", "cost_date": "", "price": ""},
		{"id": "t1", "date": "2026-08-01", "payee": "盒马鲜生", "narration": "a",
			"account": "Assets:Cash", "number": "120.00", "currency": "CNY",
			"cost_number": "", "cost_currency": "", "cost_date": "", "price": ""},
		{"id": "t2", "date": "2026-08-02", "payee": "盒马鲜生", "narration": "a",
			"account": "Expenses:Food", "number": "-120.00", "currency": "CNY",
			"cost_number": "", "cost_currency": "", "cost_date": "", "price": ""},
		{"id": "t2", "date": "2026-08-02", "payee": "盒马鲜生", "narration": "a",
			"account": "Assets:Cash", "number": "120.00", "currency": "CNY",
			"cost_number": "", "cost_currency": "", "cost_date": "", "price": ""},
	}
	svc := &Service{Engine: &fakeEngine{rows: rows}}
	l := db.Ledger{ID: "l1", DataPath: filepath.Join(t.TempDir(), "ledger"), OperatingCurrency: "CNY"}
	insights, err := svc.AiInsights(ctx, l, "2026-08")
	if err != nil {
		t.Fatal(err)
	}
	if len(insights) != 1 || insights[0].Type != "duplicate" {
		t.Fatalf("unexpected insights: %+v", insights)
	}
}
