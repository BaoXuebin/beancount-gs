package ledger

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/beancount-gs/api/internal/ai"
	"github.com/beancount-gs/api/internal/db"
)

type Insight struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// AiRecord 自然语言 → 待确认交易草稿（AI 不直接写账）。
func (s *Service) AiRecord(ctx context.Context, l db.Ledger, text string) (Transaction, string, error) {
	if s.AI == nil || !s.AI.Enabled() {
		return Transaction{}, "", ai.ErrNotConfigured
	}
	system := "你是 beancount 记账助手。把用户自然语言转换为 json 交易对象，严格使用字段：" +
		"date(YYYY-MM-DD，未给则用今天)、payee、narration、postings（数组，每项 account 和 units{number,currency}）。" +
		"借贷必须平衡，number 是字符串，账户以 Assets:/Liabilities:/Income:/Expenses:/Equity: 开头。"
	var draft struct {
		Date      string `json:"date"`
		Payee     string `json:"payee"`
		Narration string `json:"narration"`
		Postings  []struct {
			Account string `json:"account"`
			Units   struct {
				Number   string `json:"number"`
				Currency string `json:"currency"`
			} `json:"units"`
		} `json:"postings"`
	}
	if err := s.AI.ChatJSON(ctx, system, text, &draft); err != nil {
		return Transaction{}, "", err
	}
	if draft.Date == "" {
		draft.Date = time.Now().Format("2006-01-02")
	}
	txn := Transaction{Date: draft.Date, Payee: draft.Payee, Narration: draft.Narration}
	for _, p := range draft.Postings {
		if strings.TrimSpace(p.Account) == "" {
			continue
		}
		currency := p.Units.Currency
		if currency == "" {
			currency = l.OperatingCurrency
		}
		txn.Postings = append(txn.Postings, Posting{
			Account: p.Account,
			Units:   &Amount{Number: p.Units.Number, Currency: currency},
		})
	}
	if len(txn.Postings) < 2 {
		return Transaction{}, "", errors.New("AI 生成的交易缺少完整分录")
	}
	return txn, "AI 生成草稿，请确认后调用创建接口", nil
}

type AccountSuggestion struct {
	Account  string `json:"account"`
	Currency string `json:"currency,omitempty"`
}

// AiAccounts 自然语言 → 建议账户列表（AI 不直接写账，由调用方确认后批量开户）。
func (s *Service) AiAccounts(ctx context.Context, l db.Ledger, text string) ([]AccountSuggestion, string, error) {
	if s.AI == nil || !s.AI.Enabled() {
		return nil, "", ai.ErrNotConfigured
	}
	existing, err := s.ListAccounts(ctx, l, true)
	if err != nil {
		return nil, "", err
	}
	names := make([]string, 0, len(existing))
	for _, a := range existing {
		names = append(names, a.Name)
	}
	sort.Strings(names)
	system := "你是 beancount 账户设计助手。根据用户描述整理成不重复的 beancount 账户列表。" +
		"账户名必须以 Assets:/Liabilities:/Income:/Expenses:/Equity: 开头且至少两级（如 Assets:Bank:招商银行）。" +
		"多币种账户在 currency 字段注明（逗号分隔多个币种），普通账户省略 currency。" +
		"只输出 json 对象，不要输出任何其他文字、解释或 Markdown。示例：{\"accounts\":[{\"account\":\"Assets:Bank:招商银行\",\"currency\":\"CNY\"}],\"notes\":\"...\"}"
	prompt := fmt.Sprintf("现有账户：%s\n用户需求：%s", strings.Join(names, ", "), text)
	var out struct {
		Accounts []AccountSuggestion `json:"accounts"`
		Notes    string              `json:"notes,omitempty"`
	}
	if err := s.AI.ChatJSON(ctx, system, prompt, &out); err != nil {
		return nil, "", err
	}
	// 过滤空账户名
	filtered := out.Accounts[:0]
	for _, a := range out.Accounts {
		if strings.TrimSpace(a.Account) != "" {
			filtered = append(filtered, a)
		}
	}
	if len(filtered) == 0 {
		return nil, "", errors.New("AI 未返回有效账户，请调整描述后重试")
	}
	return filtered, out.Notes, nil
}

// AiSummarize 生成月度财务总结。
func (s *Service) AiSummarize(ctx context.Context, l db.Ledger, month string) (string, error) {
	if s.AI == nil || !s.AI.Enabled() {
		return "", ai.ErrNotConfigured
	}
	total, _ := s.StatsTotal(ctx, l, month, "")
	trend, _ := s.StatsTrend(ctx, l, month, "", "month")
	prompt := fmt.Sprintf("账本数据：\n总览：%v\n月度趋势：%v\n请用中文生成一段简明财务总结（收入、支出、结余、主要变化）。", total, trend)
	var out struct {
		Summary string `json:"summary"`
	}
	if err := s.AI.ChatJSON(ctx, "你是财务分析助手，只输出 JSON {\"summary\": \"...\"}", prompt, &out); err != nil {
		return "", err
	}
	return out.Summary, nil
}

// AiInsights 规则式洞察：重复扣款检测（无需 LLM）。
func (s *Service) AiInsights(ctx context.Context, l db.Ledger, month string) ([]Insight, error) {
	txns, err := s.List(ctx, l, Filters{Month: month})
	if err != nil {
		return nil, err
	}
	byKey := make(map[string][]string)
	for _, t := range txns {
		for _, p := range t.Postings {
			if p.Units == nil || !strings.HasPrefix(p.Units.Number, "-") {
				continue // 只看支出腿
			}
			key := t.Payee + "|" + p.Units.Number + "|" + p.Units.Currency
			byKey[key] = append(byKey[key], t.Date)
		}
	}
	insights := make([]Insight, 0)
	for key, dates := range byKey {
		if len(dates) < 2 {
			continue
		}
		parts := strings.Split(key, "|")
		insights = append(insights, Insight{
			Type: "duplicate",
			Message: fmt.Sprintf("检测到 %d 笔「%s」金额 %s %s，疑似重复扣款：%s",
				len(dates), parts[0], parts[1], parts[2], strings.Join(dates, ", ")),
		})
	}
	return insights, nil
}
