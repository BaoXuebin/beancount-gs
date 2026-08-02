package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

type txnFakeEngineWithText struct {
	rows      []beancount.Row
	printText string
}

func (f *txnFakeEngineWithText) QueryCSV(_ context.Context, _, _ string) ([]beancount.Row, error) {
	return f.rows, nil
}
func (f *txnFakeEngineWithText) Print(_ context.Context, _, _ string) (string, error) {
	return f.printText, nil
}
func (f *txnFakeEngineWithText) Check(_ context.Context, _ string) ([]string, error) { return nil, nil }

func TestTransactionFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	store, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	user, err := store.UpsertGitHubUser(ctx, "1", "alice", "", "Alice")
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
	dataPath := filepath.Join(t.TempDir(), "ledger")
	ledgerRow, err := store.CreateLedgerWithOwner(ctx, db.NewLedgerParams{
		ID: uuid.NewString(), TeamID: team.ID, Name: "家庭账本",
		DataPath: dataPath, OperatingCurrency: "CNY",
	}, user.ID)
	if err != nil {
		t.Fatal(err)
	}

	engine := &txnFakeEngineWithText{rows: []beancount.Row{
		{"id": "txn-abc", "date": "2026-08-02", "payee": "盒马鲜生", "narration": "日常采购",
			"account": "Expenses:Food", "number": "-120.00", "currency": "CNY",
			"cost_number": "", "cost_currency": "", "cost_date": "", "price": ""},
		{"id": "txn-abc", "date": "2026-08-02", "payee": "盒马鲜生", "narration": "日常采购",
			"account": "Assets:Cash", "number": "120.00", "currency": "CNY",
			"cost_number": "", "cost_currency": "", "cost_date": "", "price": ""},
	}}
	svc := &ledger.Service{Store: store, Engine: engine}
	router := gin.New()
	authed := router.Group("/api/v2", RequireSession(store, "bgs_session"))
	h := &TransactionHandlers{Store: store, Service: svc}
	authed.POST("/ledgers/:ledger_id/transactions", h.Create)
	authed.GET("/ledgers/:ledger_id/transactions", h.List)
	authed.GET("/ledgers/:ledger_id/transactions/:transaction_id", h.Get)
	authed.PUT("/ledgers/:ledger_id/transactions/:transaction_id", h.Update)
	authed.DELETE("/ledgers/:ledger_id/transactions/:transaction_id", h.Delete)

	body := `{"date":"2026-08-02","payee":"盒马鲜生","narration":"日常采购","tags":["Food"],"postings":[
		{"account":"Expenses:Food","units":{"number":"-120.00","currency":"CNY"}},
		{"account":"Assets:Cash","units":{"number":"120.00","currency":"CNY"}}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/ledgers/"+ledgerRow.ID+"/transactions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Revision-Match", "0")
	req.AddCookie(&http.Cookie{Name: "bgs_session", Value: token})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create txn status %d body=%s", w.Code, w.Body.String())
	}
	var created gen.Transaction
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Narration == nil || *created.Narration != "日常采购" || len(created.Postings) != 2 {
		t.Fatalf("unexpected transaction: %+v", created)
	}

	// 缺少修订号 → 422
	req = httptest.NewRequest(http.MethodPost, "/api/v2/ledgers/"+ledgerRow.ID+"/transactions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "bgs_session", Value: token})
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("missing revision should 422, got %d", w.Code)
	}

	// 列表
	req = httptest.NewRequest(http.MethodGet, "/api/v2/ledgers/"+ledgerRow.ID+"/transactions", nil)
	req.AddCookie(&http.Cookie{Name: "bgs_session", Value: token})
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list status %d", w.Code)
	}
	var listResp struct {
		Items []gen.Transaction `json:"items"`
		Total int               `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil {
		t.Fatal(err)
	}
	if listResp.Total != 1 || len(listResp.Items) != 1 {
		t.Fatalf("unexpected list: %+v", listResp)
	}

	// 详情
	req = httptest.NewRequest(http.MethodGet, "/api/v2/ledgers/"+ledgerRow.ID+"/transactions/txn-abc", nil)
	req.AddCookie(&http.Cookie{Name: "bgs_session", Value: token})
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("detail status %d", w.Code)
	}

	// 更新
	monthFile := filepath.Join(dataPath, "month", "2026-08.bean")
	fileContent, err := os.ReadFile(monthFile)
	if err != nil {
		t.Fatalf("read month file: %v", err)
	}
	engine.printText = string(fileContent)
	engine.rows = []beancount.Row{
		{"id": "txn-def", "date": "2026-08-02", "payee": "盒马鲜生", "narration": "修正",
			"account": "Expenses:Food", "number": "-130.00", "currency": "CNY",
			"cost_number": "", "cost_currency": "", "cost_date": "", "price": ""},
		{"id": "txn-def", "date": "2026-08-02", "payee": "盒马鲜生", "narration": "修正",
			"account": "Assets:Cash", "number": "130.00", "currency": "CNY",
			"cost_number": "", "cost_currency": "", "cost_date": "", "price": ""},
	}
	updateBody := `{"date":"2026-08-02","payee":"盒马鲜生","narration":"修正","postings":[
		{"account":"Expenses:Food","units":{"number":"-130.00","currency":"CNY"}},
		{"account":"Assets:Cash","units":{"number":"130.00","currency":"CNY"}}]}`
	req = httptest.NewRequest(http.MethodPut, "/api/v2/ledgers/"+ledgerRow.ID+"/transactions/txn-abc", bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Revision-Match", "1")
	req.AddCookie(&http.Cookie{Name: "bgs_session", Value: token})
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update status %d body=%s", w.Code, w.Body.String())
	}
	fileContent, err = os.ReadFile(monthFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(fileContent), "修正") || strings.Contains(string(fileContent), "-120.00") {
		t.Fatalf("update not written to file:\n%s", fileContent)
	}

	// 删除
	engine.printText = string(fileContent)
	req = httptest.NewRequest(http.MethodDelete, "/api/v2/ledgers/"+ledgerRow.ID+"/transactions/txn-def", nil)
	req.Header.Set("If-Revision-Match", "2")
	req.AddCookie(&http.Cookie{Name: "bgs_session", Value: token})
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete status %d body=%s", w.Code, w.Body.String())
	}
	fileContent, err = os.ReadFile(monthFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(fileContent)) != "" {
		t.Fatalf("delete not applied:\n%s", fileContent)
	}
}
