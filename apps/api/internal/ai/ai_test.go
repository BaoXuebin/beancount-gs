package ai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
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

func TestDeepseekDefaultBaseURL(t *testing.T) {
	c := NewClient(Config{Provider: "deepseek", APIKey: "sk-test", Model: "deepseek-chat"})
	if c.cfg.BaseURL != "https://api.deepseek.com/v1" {
		t.Fatalf("deepseek base url wrong: %s", c.cfg.BaseURL)
	}
}

func TestChatJSONEmptyContentRetryWithoutJSONMode(t *testing.T) {
	var calls int
	var secondBodyHasJSONMode bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		body := readBody(r)
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			// DeepSeek json_object 模式偶发空 content
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":""}}]}`))
			return
		}
		// 第二次请求不应再带 response_format
		secondBodyHasJSONMode = strings.Contains(body, "response_format")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"ok\":true}"}}]}`))
	}))
	defer srv.Close()

	client := NewClient(Config{
		Provider: "openai", APIKey: "test-key", Model: "gpt-4o-mini", BaseURL: srv.URL,
	})
	var out struct {
		Ok bool `json:"ok"`
	}
	if err := client.ChatJSON(context.Background(), "sys", "user", &out); err != nil {
		t.Fatalf("chat json: %v", err)
	}
	if !out.Ok {
		t.Fatalf("unexpected out: %+v", out)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
	if secondBodyHasJSONMode {
		t.Fatal("second attempt should not include response_format")
	}
}

func TestChatJSONEmptyContentBothAttempts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":""}}]}`))
	}))
	defer srv.Close()

	client := NewClient(Config{
		Provider: "openai", APIKey: "test-key", Model: "gpt-4o-mini", BaseURL: srv.URL,
	})
	var out struct {
		Ok bool `json:"ok"`
	}
	err := client.ChatJSON(context.Background(), "sys", "user", &out)
	if !errors.Is(err, ErrEmptyResponse) {
		t.Fatalf("expected ErrEmptyResponse, got %v", err)
	}
}

func readBody(r *http.Request) string {
	b, _ := io.ReadAll(r.Body)
	r.Body = http.NoBody
	return string(b)
}
