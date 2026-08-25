package ledger

import (
	"context"
	"encoding/json"
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
	var draft aiTxnDraft
	if err := s.AI.ChatJSON(ctx, system, text, &draft); err != nil {
		return Transaction{}, "", err
	}
	txn, ok := buildAiTransaction(l, draft)
	if !ok {
		return Transaction{}, "", errors.New("AI 生成的交易缺少完整分录")
	}
	return txn, "AI 生成草稿，请确认后调用创建接口", nil
}

// aiTxnDraft AI 生成交易草稿的原始结构（单条与批量共用）。
type aiTxnDraft struct {
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

// buildAiTransaction 把 AI 草稿转换为内部交易；分录不足 2 条时返回 ok=false。
func buildAiTransaction(l db.Ledger, d aiTxnDraft) (Transaction, bool) {
	date := d.Date
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	txn := Transaction{Date: date, Payee: d.Payee, Narration: d.Narration}
	for _, p := range d.Postings {
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
	return txn, len(txn.Postings) >= 2
}

// AiRecordBatch 自然语言 → 多条待确认交易草稿（一次可描述多笔交易）。
func (s *Service) AiRecordBatch(ctx context.Context, l db.Ledger, text string) ([]Transaction, string, error) {
	if s.AI == nil || !s.AI.Enabled() {
		return nil, "", ai.ErrNotConfigured
	}
	system := "你是 beancount 记账助手。把用户自然语言转换为 json 对象，字段：drafts（交易数组，每笔含 date(YYYY-MM-DD，未给则用今天)、payee、narration、postings（数组，每项 account 和 units{number,currency}））、notes（提示文字）。" +
		"用户可能一次描述多笔交易（如多行），请为每一笔生成一个草稿。每笔交易借贷必须平衡，number 是字符串，账户以 Assets:/Liabilities:/Income:/Expenses:/Equity: 开头。" +
		"只输出 json，不要 Markdown。示例：{\"drafts\":[{\"date\":\"2026-08-04\",\"payee\":\"星巴克\",\"narration\":\"咖啡\",\"postings\":[{\"account\":\"Expenses:Food:Coffee\",\"units\":{\"number\":\"38\",\"currency\":\"CNY\"}},{\"account\":\"Assets:Cash\",\"units\":{\"number\":\"-38\",\"currency\":\"CNY\"}}]}],\"notes\":\"已生成 1 笔\"}"
	var out struct {
		Drafts []aiTxnDraft `json:"drafts"`
		Notes  string       `json:"notes,omitempty"`
	}
	if err := s.AI.ChatJSON(ctx, system, text, &out); err != nil {
		return nil, "", err
	}
	result := make([]Transaction, 0, len(out.Drafts))
	for _, d := range out.Drafts {
		if txn, ok := buildAiTransaction(l, d); ok {
			result = append(result, txn)
		}
	}
	if len(result) == 0 {
		return nil, "", errors.New("AI 未生成有效交易，请调整描述后重试")
	}
	return result, out.Notes, nil
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
	system := "你是 beancount 账户设计助手。根据用户描述整理成 beancount 账户列表。" +
		"账户名必须以 Assets:/Liabilities:/Income:/Expenses:/Equity: 开头且至少两级（如 Assets:Bank:招商银行）。" +
		"多币种账户在 currency 字段注明（逗号分隔多个币种），普通账户省略 currency。" +
		"只输出需要新增的账户：已出现在现有账户列表中的账户不要输出，用户需求未提到的账户也不要输出。" +
		"只输出 json 对象，不要输出任何其他文字、解释或 Markdown。示例：{\"accounts\":[{\"account\":\"Assets:Bank:招商银行\",\"currency\":\"CNY\"}],\"notes\":\"...\"}"
	prompt := fmt.Sprintf("现有账户：%s\n用户需求：%s", strings.Join(names, ", "), text)
	var out struct {
		Accounts []AccountSuggestion `json:"accounts"`
		Notes    string              `json:"notes,omitempty"`
	}
	if err := s.AI.ChatJSON(ctx, system, prompt, &out); err != nil {
		return nil, "", err
	}
	// 双保险过滤：跳过已存在账户，并对返回列表自身去重（不依赖 AI 服从指令）
	existingSet := make(map[string]struct{}, len(existing))
	for _, a := range existing {
		existingSet[a.Name] = struct{}{}
	}
	filtered := make([]AccountSuggestion, 0, len(out.Accounts))
	seen := make(map[string]struct{}, len(out.Accounts))
	for _, a := range out.Accounts {
		name := strings.TrimSpace(a.Account)
		if name == "" {
			continue
		}
		if _, dup := existingSet[name]; dup {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		filtered = append(filtered, AccountSuggestion{Account: name, Currency: a.Currency})
	}
	if len(filtered) == 0 {
		return nil, "", errors.New("AI 未返回需要新增的账户（可能都已存在），请调整描述后重试")
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

// AiInsights 规则式洞察：同日同收款方同金额的重复扣款检测（无需 LLM）。
func (s *Service) AiInsights(ctx context.Context, l db.Ledger, month string) ([]Insight, error) {
	txns, err := s.List(ctx, l, Filters{Month: month})
	if err != nil {
		return nil, err
	}
	// 按日期+收款方+金额+币种聚合；不同日期的同额消费属正常重复消费，不算重复扣款
	type dupKey struct {
		date, payee, number, currency string
	}
	counts := make(map[dupKey][]string) // key -> 去重后的交易 id
	for _, t := range txns {
		for _, p := range t.Postings {
			if p.Units == nil || !strings.HasPrefix(p.Units.Number, "-") {
				continue // 只看支出腿
			}
			k := dupKey{t.Date, t.Payee, p.Units.Number, p.Units.Currency}
			ids := counts[k]
			if len(ids) == 0 || ids[len(ids)-1] != t.ID {
				found := false
				for _, id := range ids {
					if id == t.ID {
						found = true
						break
					}
				}
				if !found {
					counts[k] = append(ids, t.ID)
				}
			}
		}
	}
	insights := make([]Insight, 0)
	for k, ids := range counts {
		if len(ids) < 2 {
			continue
		}
		insights = append(insights, Insight{
			Type: "duplicate",
			Message: fmt.Sprintf("%s 有 %d 笔「%s」金额 %s %s，疑似重复扣款",
				k.date, len(ids), k.payee, k.number, k.currency),
		})
	}
	sort.Slice(insights, func(i, j int) bool { return insights[i].Message < insights[j].Message })
	return insights, nil
}

// ChatMessage 对话消息（多轮 AI 记录）。
type ChatMessage struct {
	Role    string `json:"role"`    // user | assistant
	Content string `json:"content"`
}

// AiRecordChat 对话式批量记账：结合对话历史与当前草稿，返回调整后的完整草稿数组。
// 前端勾选确认后逐笔调用创建接口。
func (s *Service) AiRecordChat(ctx context.Context, l db.Ledger, messages []ChatMessage, drafts []Transaction) ([]Transaction, string, error) {
	if s.AI == nil || !s.AI.Enabled() {
		return nil, "", ai.ErrNotConfigured
	}
	var b strings.Builder
	b.WriteString("你是 beancount 记账助手，与用户多轮对话生成交易草稿。\n")
	b.WriteString("把用户最新的自然语言转换为 json 对象，字段：drafts（完整交易数组，每笔含 date(YYYY-MM-DD，未给则用今天)、payee、narration、postings（数组，每项 account 和 units{number,currency}））、notes（提示文字）。\n")
	b.WriteString("用户可能一次描述多笔交易。每次都必须输出当前全部草稿（包含之前已生成且未删除的），并按最新对话要求增删改。\n")
	b.WriteString("每笔交易借贷必须平衡，number 是字符串，账户以 Assets:/Liabilities:/Income:/Expenses:/Equity: 开头。\n")
	b.WriteString("只输出 json，不要 Markdown。\n")
	if len(drafts) > 0 {
		if data, err := json.Marshal(drafts); err == nil {
			b.WriteString("当前草稿（尽量保留，仅按用户要求修改）：\n")
			b.WriteString(string(data))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n对话历史：\n")
	for _, m := range messages {
		role := "用户"
		if m.Role == "assistant" {
			role = "助手"
		}
		b.WriteString(role + "：" + m.Content + "\n")
	}
	prompt := b.String()
	var out struct {
		Drafts []aiTxnDraft `json:"drafts"`
		Notes  string       `json:"notes,omitempty"`
	}
	if err := s.AI.ChatJSON(ctx, "你是 beancount 记账助手，只输出 json，不要 Markdown", prompt, &out); err != nil {
		return nil, "", err
	}
	result := make([]Transaction, 0, len(out.Drafts))
	for _, d := range out.Drafts {
		if txn, ok := buildAiTransaction(l, d); ok {
			result = append(result, txn)
		}
	}
	if len(result) == 0 {
		return nil, "", errors.New("AI 未生成有效交易，请调整描述后重试")
	}
	return result, out.Notes, nil
}
