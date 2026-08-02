package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/beancount-gs/api/internal/db"
	"github.com/beancount-gs/api/internal/http/gen"
	"github.com/beancount-gs/api/internal/security"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestLedgerFlow(t *testing.T) {
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

	router := gin.New()
	authed := router.Group("/api/v2", RequireSession(store, "bgs_session"))
	ledgerH := &LedgerHandlers{Store: store, DataRoot: t.TempDir(), TemplateDir: tpl}
	authed.POST("/ledgers", ledgerH.Create)
	authed.GET("/ledgers/:ledger_id", ledgerH.Get)
	authed.GET("/ledgers/:ledger_id/revision", ledgerH.Revision)

	body, _ := json.Marshal(gen.LedgerCreate{TeamId: team.ID, Name: "家庭账本", OperatingCurrency: "CNY"})
	req := httptest.NewRequest(http.MethodPost, "/api/v2/ledgers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "bgs_session", Value: token})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create ledger status %d body=%s", w.Code, w.Body.String())
	}
	var created gen.Ledger
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Name != "家庭账本" || created.Role != gen.Role("owner") || created.Revision != 0 {
		t.Fatalf("unexpected ledger: %+v", created)
	}
	if created.DataPath == nil {
		t.Fatal("data_path missing")
	}
	content, err := os.ReadFile(filepath.Join(*created.DataPath, "index.bean"))
	if err != nil {
		t.Fatalf("ledger files not initialized: %v", err)
	}
	if !bytes.Contains(content, []byte(`"CNY"`)) {
		t.Fatalf("placeholder not replaced: %s", content)
	}

	// 详情与修订号
	req = httptest.NewRequest(http.MethodGet, "/api/v2/ledgers/"+created.Id, nil)
	req.AddCookie(&http.Cookie{Name: "bgs_session", Value: token})
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get ledger status %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v2/ledgers/"+created.Id+"/revision", nil)
	req.AddCookie(&http.Cookie{Name: "bgs_session", Value: token})
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("revision status %d", w.Code)
	}

	// 非成员访问 → 404
	other, err := store.UpsertGitHubUser(ctx, "222", "bob", "", "Bob")
	if err != nil {
		t.Fatal(err)
	}
	token2 := "session-token-2"
	if err := store.CreateSession(ctx, uuid.NewString(), other.ID, security.HashToken(token2), time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v2/ledgers/"+created.Id, nil)
	req.AddCookie(&http.Cookie{Name: "bgs_session", Value: token2})
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("other user should get 404, got %d", w.Code)
	}
}
