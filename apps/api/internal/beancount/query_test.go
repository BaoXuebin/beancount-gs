package beancount

import (
	"strings"
	"testing"
)

func TestParseCSV(t *testing.T) {
	in := `id,date,payee,narration,tags,account,number,currency,cost_number
txn-1,2026-08-02,盒马鲜生,日常采购,"('Food', 'Family')",Expenses:Food,-120.00,CNY,
txn-1,2026-08-02,盒马鲜生,日常采购,"('Food', 'Family')",Assets:Cash,120.00,CNY,
txn-2,2026-08-01,None,工资,,Income:Salary,18200.00,CNY,`
	rows, err := parseCSV(in)
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if rows[0]["id"] != "txn-1" || rows[0]["account"] != "Expenses:Food" || rows[0]["number"] != "-120.00" {
		t.Fatalf("unexpected row: %+v", rows[0])
	}
	if rows[2]["payee"] != "" || rows[2]["narration"] != "工资" {
		t.Fatalf("None should map to empty: %+v", rows[2])
	}
}

func TestParseCSVEmpty(t *testing.T) {
	rows, err := parseCSV("")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected empty rows")
	}
}

func TestParseCSVLazyQuotes(t *testing.T) {
	// bean-query 偶发输出不闭合引号的字段（如账户名/金额含引号），LazyQuotes 应容忍
	in := "account,market_position,position\n\"Assets:\"Cash\",\"1234.00 CNY\",\"1234.00 CNY\"\n"
	rows, err := parseCSV(in)
	if err != nil {
		t.Fatalf("lazy quotes parse failed: %v", err)
	}
	if len(rows) != 1 || rows[0]["account"] != "Assets:\"Cash" || rows[0]["market_position"] != "1234.00 CNY" {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}

func TestDecodeOutput(t *testing.T) {
	if got := decodeOutput([]byte("hello")); got != "hello" {
		t.Fatalf("utf8 passthrough failed: %q", got)
	}
	// GBK 编码的中文应被转换
	if !strings.Contains(decodeOutput([]byte{0xc4, 0xe3, 0xba, 0xc3}), "你好") {
		t.Fatal("gbk decode failed")
	}
}
