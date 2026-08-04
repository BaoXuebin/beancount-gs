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

// ErrEmptyResponse 模型返回了空内容（DeepSeek json_object 模式已知偶发问题）。
var ErrEmptyResponse = errors.New("AI 模型返回为空，请重试或调整输入")

type Config struct {
	Provider   string // openai | compatible | ollama | deepseek
	APIKey     string
	Model      string
	BaseURL    string
	HTTPClient *http.Client
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
	case "deepseek":
		if base == "" {
			base = "https://api.deepseek.com/v1"
		}
	default:
		if base == "" {
			base = "https://api.openai.com/v1"
		}
	}
	cfg.BaseURL = base
	c := &Client{cfg: cfg}
	if cfg.HTTPClient != nil {
		c.http = cfg.HTTPClient
	} else {
		c.http = &http.Client{Timeout: 60 * time.Second}
	}
	return c
}

func (c *Client) Enabled() bool {
	return c.cfg.Enabled()
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatJSON 调用 OpenAI 兼容的 /chat/completions，要求模型返回 JSON。
// DeepSeek 的 json_object 模式存在偶发返回空内容的问题，首次失败会自动去掉
// response_format 重试一次（仅靠 prompt 引导输出 JSON，兼容更多模型 / 网关）。
func (c *Client) ChatJSON(ctx context.Context, system, user string, out any) error {
	err := c.chatJSONOnce(ctx, system, user, out, true)
	if err == nil {
		return nil
	}
	// 空内容 / JSON 解析失败时才重试，避免重复无意义的网络请求
	if errors.Is(err, ErrEmptyResponse) || isJSONParseError(err) {
		if retryErr := c.chatJSONOnce(ctx, system, user, out, false); retryErr == nil {
			return nil
		}
	}
	return err
}

func (c *Client) chatJSONOnce(ctx context.Context, system, user string, out any, useJSONMode bool) error {
	payload := map[string]any{
		"model": c.cfg.Model,
		"messages": []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		"temperature": 0.2,
	}
	if useJSONMode {
		payload["response_format"] = map[string]string{"type": "json_object"}
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
	if strings.TrimSpace(content) == "" {
		return ErrEmptyResponse
	}
	return json.Unmarshal([]byte(content), out)
}

func isJSONParseError(err error) bool {
	if err == nil {
		return false
	}
	var syntaxErr *json.SyntaxError
	return errors.As(err, &syntaxErr) || strings.Contains(err.Error(), "unexpected end of JSON input")
}
