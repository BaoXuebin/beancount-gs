package repository

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitLedgerFiles(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "index.bean"),
		[]byte("option \"operating_currency\" \"%operatingCurrency%\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(src, "month"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "month", "months.bean"),
		[]byte("; months\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(t.TempDir(), "ledger")
	if err := InitLedgerFiles(dest, src, "2026-08-02", "CNY"); err != nil {
		t.Fatalf("init ledger files: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dest, "index.bean"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "\"CNY\"") {
		t.Fatalf("placeholder not replaced: %s", content)
	}
	if _, err := os.Stat(filepath.Join(dest, "month", "months.bean")); err != nil {
		t.Fatalf("nested file missing: %v", err)
	}
}
