package httpapi

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/beancount-gs/api/internal/beancount"
	"github.com/beancount-gs/api/internal/db"
	"github.com/beancount-gs/api/internal/http/gen"
	"github.com/beancount-gs/api/internal/ledger"
	"github.com/beancount-gs/api/internal/security"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func buildBackupZip(t *testing.T, entries map[string]string) []byte {
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

func multipartZipRequest(path string, zipData []byte) (*http.Request, *multipart.Writer) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "backup.zip")
	_, _ = fw.Write(zipData)
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, path, &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req, mw
}

func TestBackupImportNewAndInto(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	store, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	user, err := store.UpsertGitHubUser(ctx, "111", "alice", "alice@example.com", "Alice")
	if err != nil {
		t.Fatal(err)
	}
	team, err := store.CreateTeamWithOwner(ctx, uuid.NewString(), "家庭", user.ID)
	if err != nil {
		t.Fatal(err)
	}
	token := "session-token"
	if err := store.CreateSession(ctx, uuid.NewString(), user.ID, security.HashToken(token), time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	tpl := t.TempDir()
	if err := os.WriteFile(filepath.Join(tpl, "index.bean"),
		[]byte("option \"operating_currency\" \"%operatingCurrency%\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dataRoot := t.TempDir()

	router := gin.New()
	authed := router.Group("/api/v2", RequireSession(store, "bgs_session"))
	ledgerH := &LedgerHandlers{Store: store, DataRoot: dataRoot, TemplateDir: tpl}
	authed.POST("/ledgers", ledgerH.Create)
	backupH := &BackupHandlers{
		Store:    store,
		Service:  &ledger.Service{Store: store, Engine: beancount.CmdEngine{}},
		DataRoot: dataRoot,
	}
	authed.POST("/ledgers/import", backupH.ImportAsNew)
	authed.POST("/ledgers/:ledger_id/import", backupH.ImportInto)

	cookie := &http.Cookie{Name: "bgs_session", Value: token}

	// 1) 新建账本导入
	zipData := buildBackupZip(t, map[string]string{
		"index.bean":         "option \"operating_currency\" \"CNY\"\n2021-01-01 open Assets:Cash CNY\n2021-01-01 open Expenses:Food CNY\ninclude \"./month/2026-08.bean\"\n",
		"month/2026-08.bean": "2026-08-01 * \"盒马\" \"采购\"\n  Expenses:Food 100.00 CNY\n  Assets:Cash\n",
	})
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("team_id", team.ID)
	_ = mw.WriteField("name", "导入账本")
	_ = mw.WriteField("operating_currency", "CNY")
	fw, _ := mw.CreateFormFile("file", "backup.zip")
	_, _ = fw.Write(zipData)
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v2/ledgers/import", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("import new status %d body=%s", w.Code, w.Body.String())
	}
	var created struct {
		Ledger gen.Ledger `json:"ledger"`
		Files  []string   `json:"files"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Ledger.Name != "导入账本" || created.Ledger.DataPath == nil {
		t.Fatalf("unexpected ledger: %+v", created.Ledger)
	}
	if len(created.Files) != 2 {
		t.Fatalf("expected 2 files, got %v", created.Files)
	}
	if _, err := os.Stat(filepath.Join(*created.Ledger.DataPath, "month", "2026-08.bean")); err != nil {
		t.Fatalf("imported month file missing: %v", err)
	}

	// 2) 导入已有账本（覆盖 index.bean + 新增 event 目录），修订号 0 -> 1
	newZip := buildBackupZip(t, map[string]string{
		"index.bean":        "option \"operating_currency\" \"CNY\"\n; imported-again\n",
		"event/events.bean": "; events\n",
	})
	req, _ = multipartZipRequest("/api/v2/ledgers/"+created.Ledger.Id+"/import", newZip)
	req.Header.Set("If-Revision-Match", "0")
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("import into status %d body=%s", w.Code, w.Body.String())
	}
	var result struct {
		Revision int64 `json:"revision"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Revision != 1 {
		t.Fatalf("expected revision 1, got %d", result.Revision)
	}
	content, err := os.ReadFile(filepath.Join(*created.Ledger.DataPath, "index.bean"))
	if err != nil || !bytes.Contains(content, []byte("; imported-again")) {
		t.Fatalf("index.bean not replaced: %v %q", err, content)
	}
	backups, err := filepath.Glob(filepath.Join(*created.Ledger.DataPath, "bak", "*", "index.bean"))
	if err != nil || len(backups) == 0 {
		t.Fatalf("expected bak snapshot of index.bean, got %v %v", backups, err)
	}

	// 3) 修订号冲突返回 409
	req, _ = multipartZipRequest("/api/v2/ledgers/"+created.Ledger.Id+"/import", newZip)
	req.Header.Set("If-Revision-Match", "0")
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 on stale revision, got %d body=%s", w.Code, w.Body.String())
	}

	// 4) 缺少 index.bean 返回 422
	badZip := buildBackupZip(t, map[string]string{"readme.txt": "x"})
	var body2 bytes.Buffer
	mw2 := multipart.NewWriter(&body2)
	_ = mw2.WriteField("team_id", team.ID)
	_ = mw2.WriteField("name", "坏账本")
	fw2, _ := mw2.CreateFormFile("file", "bad.zip")
	_, _ = fw2.Write(badZip)
	_ = mw2.Close()
	req = httptest.NewRequest(http.MethodPost, "/api/v2/ledgers/import", &body2)
	req.Header.Set("Content-Type", mw2.FormDataContentType())
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for missing index.bean, got %d body=%s", w.Code, w.Body.String())
	}
}
