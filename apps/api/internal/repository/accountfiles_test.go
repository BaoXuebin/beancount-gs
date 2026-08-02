package repository

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadAccountFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "account")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets.bean"), []byte(
		"2021-01-01 open Assets:Cash CNY\n; 注释\n2026-08-02 open Assets:Bank:招商银行 CNY \"FIFO\"\n2026-08-02 close Assets:Cash\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	directives, err := ReadAccountFiles(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_ = directives
	directives, err = ReadAccountFiles(filepath.Dir(dir))
	if err != nil {
		t.Fatalf("read account files: %v", err)
	}
	if len(directives) != 3 {
		t.Fatalf("expected 3 directives, got %d: %+v", len(directives), directives)
	}
	if directives[1].Kind != "open" || directives[1].Booking != "fifo" || directives[1].Currency != "CNY" {
		t.Fatalf("unexpected directive: %+v", directives[1])
	}
	if directives[2].Kind != "close" || directives[2].Account != "Assets:Cash" {
		t.Fatalf("unexpected close directive: %+v", directives[2])
	}
}
