package ledger

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/beancount-gs/api/internal/beancount"
	"github.com/beancount-gs/api/internal/db"
)

func TestAiInsightsDuplicate(t *testing.T) {
	ctx := context.Background()
	rows := []beancount.Row{
		// 同日两笔同收款方同金额：疑似重复扣款
		{"id": "t1", "date": "2026-08-01", "payee": "盒马鲜生", "narration": "a",
			"account": "Expenses:Food", "number": "-120.00", "currency": "CNY",
			"cost_number": "", "cost_currency": "", "cost_date": "", "price": ""},
		{"id": "t1", "date": "2026-08-01", "payee": "盒马鲜生", "narration": "a",
			"account": "Assets:Cash", "number": "120.00", "currency": "CNY",
			"cost_number": "", "cost_currency": "", "cost_date": "", "price": ""},
		{"id": "t2", "date": "2026-08-01", "payee": "盒马鲜生", "narration": "a",
			"account": "Expenses:Food", "number": "-120.00", "currency": "CNY",
			"cost_number": "", "cost_currency": "", "cost_date": "", "price": ""},
		{"id": "t2", "date": "2026-08-01", "payee": "盒马鲜生", "narration": "a",
			"account": "Assets:Cash", "number": "120.00", "currency": "CNY",
			"cost_number": "", "cost_currency": "", "cost_date": "", "price": ""},
		// 不同日期同金额：正常重复消费，不应报告
		{"id": "t3", "date": "2026-08-02", "payee": "盒马鲜生", "narration": "a",
			"account": "Expenses:Food", "number": "-120.00", "currency": "CNY",
			"cost_number": "", "cost_currency": "", "cost_date": "", "price": ""},
		{"id": "t3", "date": "2026-08-02", "payee": "盒马鲜生", "narration": "a",
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
	if !strings.Contains(insights[0].Message, "2026-08-01") || strings.Contains(insights[0].Message, "2026-08-02") {
		t.Fatalf("message should only cover 2026-08-01: %s", insights[0].Message)
	}
}
