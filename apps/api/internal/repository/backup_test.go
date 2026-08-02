package repository

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func zipBytes(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range entries {
		f, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestExtractZipOK(t *testing.T) {
	data := zipBytes(t, map[string]string{
		"index.bean":         "option \"operating_currency\" \"CNY\"\n",
		"month/2026-08.bean": "2026-08-01 * \"a\" \"b\"\n  Expenses:Food 1.00 CNY\n  Assets:Cash\n",
	})
	dest := t.TempDir()
	files, err := ExtractZip(bytes.NewReader(data), int64(len(data)), dest)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %v", files)
	}
	if _, err := os.Stat(filepath.Join(dest, "month", "2026-08.bean")); err != nil {
		t.Fatal("nested file missing")
	}
}

func TestExtractZipRejectsTraversal(t *testing.T) {
	data := zipBytes(t, map[string]string{"../evil.txt": "x"})
	if _, err := ExtractZip(bytes.NewReader(data), int64(len(data)), t.TempDir()); err == nil {
		t.Fatal("expected traversal error")
	}
}

func TestExtractZipRejectsAbsolute(t *testing.T) {
	data := zipBytes(t, map[string]string{"/etc/passwd": "x"})
	if _, err := ExtractZip(bytes.NewReader(data), int64(len(data)), t.TempDir()); err == nil {
		t.Fatal("expected absolute path error")
	}
}

func TestSnapshotSkipsBak(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "index.bean"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(src, "bak", "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "bak", "sub", "old.bean"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	backup := filepath.Join(t.TempDir(), "bak")
	n, err := SnapshotTree(src, backup)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected 1 file, got %d", n)
	}
	if _, err := os.Stat(filepath.Join(backup, "index.bean")); err != nil {
		t.Fatal("index.bean snapshot missing")
	}
	if _, err := os.Stat(filepath.Join(backup, "bak", "sub", "old.bean")); !os.IsNotExist(err) {
		t.Fatal("bak dir should be skipped")
	}
}

func TestCopyTree(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "index.bean"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(src, "month"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "month", "2026-08.bean"), []byte("txn"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := t.TempDir()
	n, err := CopyTree(src, dst)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("expected 2 files, got %d", n)
	}
	content, err := os.ReadFile(filepath.Join(dst, "month", "2026-08.bean"))
	if err != nil || string(content) != "txn" {
		t.Fatalf("copy failed: %v %q", err, content)
	}
}
