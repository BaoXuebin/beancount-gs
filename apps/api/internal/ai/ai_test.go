package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChatJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("unexpected auth: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"summary\":\"你好\"}"}}]}`))
	}))
	defer srv.Close()

	client := NewClient(Config{
		Provider: "openai", APIKey: "test-key", Model: "gpt-4o-mini", BaseURL: srv.URL,
	})
	var out struct {
		Summary string `json:"summary"`
	}
	if err := client.ChatJSON(context.Background(), "sys", "user", &out); err != nil {
		t.Fatalf("chat json: %v", err)
	}
	if out.Summary != "你好" {
		t.Fatalf("unexpected summary: %+v", out)
	}
	_ = json.Valid
}

func TestNotConfigured(t *testing.T) {
	c := NewClient(Config{})
	if c.Enabled() {
		t.Fatal("empty config should be disabled")
	}
}
