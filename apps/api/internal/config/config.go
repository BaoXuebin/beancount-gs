package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port               int
	DBPath             string
	DataRoot           string
	TemplateDir        string
	GitHubClientID     string
	GitHubClientSecret string
	AppPublicURL       string
	SessionCookie      string
	StateCookie        string
}

func Load() Config {
	port := envInt("PORT", 10000)
	return Config{
		Port:               port,
		DBPath:             envStr("DB_PATH", "data/beancount-gs.db"),
		DataRoot:           envStr("DATA_ROOT", "data"),
		TemplateDir:        envStr("TEMPLATE_DIR", "../../template"),
		GitHubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		AppPublicURL:       envStr("APP_PUBLIC_URL", "http://localhost:"+strconv.Itoa(port)),
		SessionCookie:      envStr("SESSION_COOKIE", "bgs_session"),
		StateCookie:        envStr("OAUTH_STATE_COOKIE", "bgs_oauth_state"),
	}
}

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
