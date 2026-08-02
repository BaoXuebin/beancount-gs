package ledger

import (
	"context"
	"testing"

	"github.com/beancount-gs/api/internal/beancount"
	"github.com/beancount-gs/api/internal/db"
)

func TestMonths(t *testing.T) {
	engine := &fakeEngine{rows: []beancount.Row{
		{"year": "2026", "month": "8"},
		{"year": "2026", "month": "7"},
		{"year": "2025", "month": "12"},
		{"year": "2026", "month": "8"}, // 去重
		{"year": "bad", "month": "1"},  // 忽略非法行
	}}
	svc := &Service{Engine: engine}
	months, err := svc.Months(context.Background(), db.Ledger{DataPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"2026-08", "2026-07", "2025-12"}
	if len(months) != len(want) {
		t.Fatalf("expected %v, got %v", want, months)
	}
	for i := range want {
		if months[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, months)
		}
	}
}

func TestMonthsEmpty(t *testing.T) {
	svc := &Service{Engine: &fakeEngine{rows: []beancount.Row{}}}
	months, err := svc.Months(context.Background(), db.Ledger{DataPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if len(months) != 0 {
		t.Fatalf("expected empty, got %v", months)
	}
}
