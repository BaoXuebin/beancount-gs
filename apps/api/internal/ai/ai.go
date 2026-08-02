package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var ErrNotConfigured = errors.New("AI 未配置：请设置 AI_PROVIDER / AI_API_KEY / AI_MODEL")

type Config struct {
	Provider string // openai | compatible | ollama
	APIKey   string
	Model    string
	BaseURL  string
}

func (c Config) Enabled() bool {
	return c.Provider != "" && c.Model != ""
}

type Client struct {
	cfg  Config
	http *http.Client
}

func NewClient(cfg Config) *Client {
	base := cfg.BaseURL
	switch cfg.Provider {
	case "ollama":
		if base == "" {
			base = "http://localhost:11434/v1"
		}
	default:
		if base == "" {
			base = "https://api.openai.com/v1"
		}
	}
	return &Client{cfg: cfg, http: &http.Client{Timeout: 60 * time.Second}}
}

func (c *Client) Enabled() bool {
	return c.cfg.Enabled()
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatJSON 调用 OpenAI 兼容的 /chat/completions，要求模型返回 JSON。
func (c *Client) ChatJSON(ctx context.Context, system, user string, out any) error {
	payload := map[string]any{
		"model": c.cfg.Model,
		"messages": []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		"temperature":     0.2,
		"response_format": map[string]string{"type": "json_object"},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" && c.cfg.Provider != "ollama" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode >= 400 {
		return fmt.Errorf("LLM 请求失败 %d: %s", resp.StatusCode, raw)
	}
	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &chatResp); err != nil {
		return err
	}
	if len(chatResp.Choices) == 0 {
		return errors.New("LLM 返回空结果")
	}
	content := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	return json.Unmarshal([]byte(content), out)
}
