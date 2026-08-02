package ledger

import (
	"context"
	"testing"

	"github.com/beancount-gs/api/internal/beancount"
	"github.com/beancount-gs/api/internal/db"
	"github.com/shopspring/decimal"
)

func TestBuildFlowLinks(t *testing.T) {
	nodeIndex := map[string]int{"A": 0, "B": 1, "C": 2}
	postings := []flowPosting{
		{account: "A", amount: decimal.NewFromInt(-100)},
		{account: "B", amount: decimal.NewFromInt(60)},
		{account: "C", amount: decimal.NewFromInt(40)},
	}
	links := buildFlowLinks(postings, nodeIndex)
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %+v", links)
	}
	total := decimal.Zero
	for _, l := range links {
		v, err := decimal.NewFromString(l.Value)
		if err != nil {
			t.Fatal(err)
		}
		total = total.Add(v)
	}
	if !total.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("link total should be 100, got %s", total)
	}
}

func TestBreakFlowCycles(t *testing.T) {
	nodes := []string{"A", "B"}
	links := []FlowLink{
		{Source: 0, Target: 1, Value: "10.00"},
		{Source: 1, Target: 0, Value: "5.00"},
	}
	nodes, links = breakFlowCycles(nodes, links)
	if len(nodes) != 3 {
		t.Fatalf("cycle should clone a node: %v", nodes)
	}
	if _, _, ok := findBackEdge(links); ok {
		t.Fatalf("cycle should be broken: %+v", links)
	}
}

func TestStatsMappings(t *testing.T) {
	ctx := context.Background()
	engine := &fakeEngine{rows: []beancount.Row{
		{"account_type": "Expenses", "total": "-120.00 CNY"},
		{"account_type": "Income", "total": "100.00 CNY"},
	}}
	svc := &Service{Engine: engine}
	l := db.Ledger{ID: "l1", DataPath: t.TempDir(), OperatingCurrency: "CNY"}

	total, err := svc.StatsTotal(ctx, l, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if total["Expenses"] != "-120.00" || total["Income"] != "100.00" {
		t.Fatalf("unexpected total: %+v", total)
	}

	engine.rows = []beancount.Row{
		{"payee": "盒马鲜生", "cnt": "2", "total": "-80.00 CNY"},
	}
	payees, err := svc.StatsPayee(ctx, l, "", "Expenses", "avg")
	if err != nil {
		t.Fatal(err)
	}
	if len(payees) != 1 || payees[0].Amount != "-40.00" || payees[0].Count != 2 {
		t.Fatalf("unexpected payee stat: %+v", payees)
	}

	engine.rows = []beancount.Row{
		{"date": "2026-08-02", "amount": "-120.00 CNY"},
		{"date": "2026-08-03", "amount": "20.00 CNY"},
	}
	points, err := svc.StatsTrend(ctx, l, "", "Expenses", "day")
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 2 || points[0].Date != "2026-08-02" || points[0].Amount != "-120.00" ||
		points[0].Currency != "CNY" {
		t.Fatalf("unexpected trend: %+v", points)
	}
}
