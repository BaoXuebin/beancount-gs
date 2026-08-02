package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := `port: 8080
db_path: /data/test.db
github_client_id: cid
github_client_secret: secret
ai_provider: ollama
ai_model: qwen2.5
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 8080 || cfg.DBPath != "/data/test.db" {
		t.Fatalf("file values not applied: %+v", cfg)
	}
	if cfg.GitHubClientID != "cid" || cfg.AIProvider != "ollama" || cfg.AIModel != "qwen2.5" {
		t.Fatalf("secrets not applied: %+v", cfg)
	}
	// 未填字段补默认
	if cfg.DataRoot != "data" || cfg.PublicURL != "http://localhost:8080" {
		t.Fatalf("defaults not applied: %+v", cfg)
	}
	if cfg.FrontendURL != "http://localhost:8080" {
		t.Fatalf("frontend_url should default to public_url: %+v", cfg)
	}
}

func TestLoadCreatesDefaultFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 10000 || cfg.SessionCookie != "bgs_session" {
		t.Fatalf("defaults wrong: %+v", cfg)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("default file should be created: %v", err)
	}
}
