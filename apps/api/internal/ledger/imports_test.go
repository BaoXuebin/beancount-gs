package ledger

import (
	"strings"
	"testing"
)

func TestParseAlipayCSV(t *testing.T) {
	// 移动端格式（13 列）
	csv := "2026-08-02 12:30:00,分类,星巴克,方式,咖啡,支出,38.00,余额,CNY,单号,商户号,x,x"
	rows, err := parseAlipayCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Date != "2026-08-02" || rows[0].Number != "-38.00" {
		t.Fatalf("unexpected alipay row: %+v", rows)
	}
}

func TestParseWechatCSV(t *testing.T) {
	csv := "2026-08-01 08:00:00,商户,盒马鲜生,水果,支出,¥86.00,0.00,CNY,202608011234"
	rows, err := parseWechatCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Number != "-86.00" {
		t.Fatalf("unexpected wechat row: %+v", rows)
	}
}

func TestParseIcbcCSV(t *testing.T) {
	csv := "20260802,工资,a,b,c,d,e,f,18000.00,,g,h,公司"
	rows, err := parseIcbcCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Number != "18000.00" || rows[0].Payee != "公司" {
		t.Fatalf("unexpected icbc row: %+v", rows)
	}
}

func TestParseAbcCSV(t *testing.T) {
	csv := "20260802,消费,-120.00,1,2,3,4,5,6,7,8,盒马"
	rows, err := parseAbcCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Date != "2026-08-02" || rows[0].Number != "-120.00" {
		t.Fatalf("unexpected abc row: %+v", rows)
	}
}

func TestSuggestImportAccount(t *testing.T) {
	if acc, _ := suggestImportAccount("-38.00"); acc != "Expenses:" {
		t.Fatalf("expense should map to Expenses:, got %s", acc)
	}
	if acc, _ := suggestImportAccount("18000.00"); acc != "Income:" {
		t.Fatalf("income should map to Income:, got %s", acc)
	}
}
