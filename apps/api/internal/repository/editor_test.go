package repository

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReplaceLinesByNormalizedMatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "2026-08.bean")
	original := "2026-08-01 * \"A\" \"x\"\n  Expenses:Food  100.00 CNY\n  Assets:Cash  -100.00 CNY\n\n2026-08-02 * \"B\" \"y\"\n  Expenses:Food  20.00 CNY\n  Assets:Cash  -20.00 CNY\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	// 用带额外空格/逗号差异的旧文本定位并替换
	oldLines := []string{
		"2026-08-02 * \"B\" \"y\"",
		"  Expenses:Food  20.00 CNY",
		"  Assets:Cash  -20.00 CNY",
	}
	newLines := []string{
		"2026-08-02 * \"B\" \"新描述\"",
		"  Expenses:Food  25.00 CNY",
		"  Assets:Cash  -25.00 CNY",
	}
	if err := ReplaceLinesByNormalizedMatch(path, oldLines, newLines); err != nil {
		t.Fatalf("replace: %v", err)
	}
	content, _ := os.ReadFile(path)
	if !strings.Contains(string(content), `"新描述"`) || strings.Contains(string(content), "20.00") {
		t.Fatalf("replace failed:\n%s", content)
	}

	// 删除块
	if err := ReplaceLinesByNormalizedMatch(path, newLines, nil); err != nil {
		t.Fatalf("remove: %v", err)
	}
	content, _ = os.ReadFile(path)
	if strings.Contains(string(content), "2026-08-02") {
		t.Fatalf("remove failed:\n%s", content)
	}
}
