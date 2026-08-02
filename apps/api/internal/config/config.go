package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Port               int    `yaml:"port"`
	DBPath             string `yaml:"db_path"`
	DataRoot           string `yaml:"data_root"`
	TemplateDir        string `yaml:"template_dir"`
	PublicURL          string `yaml:"public_url"`
	SessionCookie      string `yaml:"session_cookie"`
	StateCookie        string `yaml:"oauth_state_cookie"`
	GitHubClientID     string `yaml:"github_client_id"`
	GitHubClientSecret string `yaml:"github_client_secret"`
	AIProvider         string `yaml:"ai_provider"`
	AIAPIKey           string `yaml:"ai_api_key"`
	AIModel            string `yaml:"ai_model"`
	AIBaseURL          string `yaml:"ai_base_url"`
}

// Defaults 返回默认配置。
func Defaults() Config {
	return Config{
		Port:          10000,
		DBPath:        "data/beancount-gs.db",
		DataRoot:      "data",
		TemplateDir:   "../../template",
		PublicURL:     "http://localhost:10000",
		SessionCookie: "bgs_session",
		StateCookie:   "bgs_oauth_state",
	}
}

const defaultFile = `# beancount-gs 服务配置
port: 10000
db_path: data/beancount-gs.db
data_root: data
template_dir: ../../template
public_url: http://localhost:10000
session_cookie: bgs_session
oauth_state_cookie: bgs_oauth_state

# GitHub OAuth（在 https://github.com/settings/developers 创建 OAuth App，
# 回调地址为 public_url + /api/v2/auth/github/callback）
github_client_id: ""
github_client_secret: ""

# AI（可选；provider: openai | compatible | ollama）
ai_provider: ""
ai_api_key: ""
ai_model: ""
ai_base_url: ""
`

// Load 读取配置文件；文件不存在时自动生成默认配置并返回默认值。
func Load(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if werr := os.WriteFile(path, []byte(defaultFile), 0o644); werr != nil {
				return cfg, fmt.Errorf("创建配置文件 %s 失败: %w", path, werr)
			}
			return Defaults(), nil
		}
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("解析配置文件 %s 失败: %w", path, err)
	}
	cfg.applyDefaults()
	return cfg, nil
}

// applyDefaults 为空字段补默认值，允许配置文件只写需要覆盖的项。
func (c *Config) applyDefaults() {
	def := Defaults()
	if c.Port == 0 {
		c.Port = def.Port
	}
	if c.DBPath == "" {
		c.DBPath = def.DBPath
	}
	if c.DataRoot == "" {
		c.DataRoot = def.DataRoot
	}
	if c.TemplateDir == "" {
		c.TemplateDir = def.TemplateDir
	}
	if c.PublicURL == "" {
		c.PublicURL = fmt.Sprintf("http://localhost:%d", c.Port)
	}
	if c.SessionCookie == "" {
		c.SessionCookie = def.SessionCookie
	}
	if c.StateCookie == "" {
		c.StateCookie = def.StateCookie
	}
}
