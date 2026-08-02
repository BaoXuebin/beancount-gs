package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/beancount-gs/api/internal/beancount"
	"github.com/beancount-gs/api/internal/db"
	httpapi "github.com/beancount-gs/api/internal/http"
	"github.com/beancount-gs/api/internal/ledger"
	mcpapi "github.com/beancount-gs/api/internal/mcp"
	"github.com/beancount-gs/api/internal/security"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestMcpHTTPHandshake(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	store, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
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
	if _, err := store.CreateLedgerWithOwner(ctx, db.NewLedgerParams{
		ID: uuid.NewString(), TeamID: team.ID, Name: "家庭账本",
		DataPath: filepath.Join(t.TempDir(), "ledger"), OperatingCurrency: "CNY",
	}, user.ID); err != nil {
		t.Fatal(err)
	}
	secret := "bgsk_http_test"
	if err := store.CreateApiKey(ctx, db.ApiKey{
		ID: uuid.NewString(), UserID: user.ID, Name: "test",
		SecretHash: security.HashToken(secret), Prefix: secret[:12], Scope: "read-write",
	}); err != nil {
		t.Fatal(err)
	}

	svc := &ledger.Service{Store: store, Engine: beancount.CmdEngine{}}
	router := gin.New()
	router.POST("/mcp", httpapi.RequireApiKey(store), gin.WrapH(mcpapi.New(store, svc, t.TempDir())))

	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+secret)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("initialize status %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		JSONRPC string `json:"jsonrpc"`
		Result  struct {
			ServerInfo struct {
				Name string `json:"name"`
			} `json:"serverInfo"`
		} `json:"result"`
		Error *json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse initialize response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("initialize error: %s", *resp.Error)
	}
	if resp.Result.ServerInfo.Name != "beancount-gs" {
		t.Fatalf("unexpected server info: %+v", resp.Result.ServerInfo)
	}

	// 无 Key → 401
	req = httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("missing key should 401, got %d", w.Code)
	}
}
