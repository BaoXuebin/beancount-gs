package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/beancount-gs/api/internal/db"
	"github.com/beancount-gs/api/internal/ledger"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

type Server struct {
	Store    *db.Store
	Service  *ledger.Service
	DataRoot string
	stream   *mcpserver.StreamableHTTPServer
}

func New(store *db.Store, service *ledger.Service, dataRoot string) *Server {
	s := &Server{Store: store, Service: service, DataRoot: dataRoot}
	ms := mcpserver.NewMCPServer("beancount-gs", "2.0.0")

	ms.AddTool(mcp.NewTool("list_ledgers", mcp.WithDescription("列出当前 API Key 用户可访问的账本")),
		s.mcpHandler(s.listLedgers, "read"))
	ms.AddTool(mcp.NewTool("query_transactions",
		mcp.WithDescription("按条件查询交易（beancount 术语：narration / postings / units / cost）"),
		mcp.WithString("ledger_id", mcp.Required(), mcp.Description("账本 ID")),
		mcp.WithString("month", mcp.Description("YYYY-MM")),
		mcp.WithString("account", mcp.Description("账户精确匹配")),
		mcp.WithString("tag", mcp.Description("标签")),
		mcp.WithString("q", mcp.Description("收款方 / 描述模糊搜索")),
		mcp.WithString("from", mcp.Description("起始日期 YYYY-MM-DD")),
		mcp.WithString("to", mcp.Description("结束日期 YYYY-MM-DD"))),
		s.mcpHandler(s.queryTransactions, "read"))
	ms.AddTool(mcp.NewTool("query_accounts",
		mcp.WithDescription("查询账本账户列表（含持仓与市值）"),
		mcp.WithString("ledger_id", mcp.Required(), mcp.Description("账本 ID"))),
		s.mcpHandler(s.queryAccounts, "read"))
	ms.AddTool(mcp.NewTool("query_stats",
		mcp.WithDescription("统计查询：total / payee / trend / flow"),
		mcp.WithString("ledger_id", mcp.Required(), mcp.Description("账本 ID")),
		mcp.WithString("kind", mcp.Required(), mcp.WithStringEnumItems([]string{"total", "payee", "trend", "flow"})),
		mcp.WithString("month", mcp.Description("YYYY-MM")),
		mcp.WithString("account", mcp.Description("账户前缀过滤，如 Expenses")),
		mcp.WithString("type", mcp.Description("payee: total|count|avg；trend: day|month|year|sum"))),
		s.mcpHandler(s.queryStats, "read"))
	ms.AddTool(mcp.NewTool("read_source_file",
		mcp.WithDescription("读取账本 beancount 源文件"),
		mcp.WithString("ledger_id", mcp.Required(), mcp.Description("账本 ID")),
		mcp.WithString("path", mcp.Required(), mcp.Description("相对路径，如 month/2026-08.bean"))),
		s.mcpHandler(s.readSourceFile, "read"))

	ms.AddTool(mcp.NewTool("create_transaction",
		mcp.WithDescription("新建交易（借贷平衡校验，postings 为 JSON 字符串）"),
		mcp.WithString("ledger_id", mcp.Required()),
		mcp.WithString("date", mcp.Required(), mcp.Description("YYYY-MM-DD")),
		mcp.WithString("payee", mcp.Description("收款方")),
		mcp.WithString("narration", mcp.Description("描述")),
		mcp.WithString("postings", mcp.Required(), mcp.Description(
			`JSON 数组：[{"account":"Expenses:Food","units":{"number":"-120.00","currency":"CNY"}},{"account":"Assets:Cash","units":{"number":"120.00","currency":"CNY"}}]`)),
		mcp.WithString("tags", mcp.WithStringItems())),
		s.mcpHandler(s.createTransaction, "write"))
	ms.AddTool(mcp.NewTool("update_transaction",
		mcp.WithDescription("整体更新交易（内容变化后交易 id 会重新生成）"),
		mcp.WithString("ledger_id", mcp.Required()),
		mcp.WithString("transaction_id", mcp.Required()),
		mcp.WithString("date", mcp.Required()),
		mcp.WithString("payee"),
		mcp.WithString("narration"),
		mcp.WithString("postings", mcp.Required()),
		mcp.WithString("tags", mcp.WithStringItems())),
		s.mcpHandler(s.updateTransaction, "write"))
	ms.AddTool(mcp.NewTool("delete_transaction",
		mcp.WithDescription("删除交易"),
		mcp.WithString("ledger_id", mcp.Required()),
		mcp.WithString("transaction_id", mcp.Required())),
		s.mcpHandler(s.deleteTransaction, "write"))
	ms.AddTool(mcp.NewTool("write_source_file",
		mcp.WithDescription("写入账本 beancount 源文件（带修订号校验）"),
		mcp.WithString("ledger_id", mcp.Required()),
		mcp.WithString("path", mcp.Required()),
		mcp.WithString("content", mcp.Required())),
		s.mcpHandler(s.writeSourceFile, "write"))

	ms.AddTool(mcp.NewTool("ai_record",
		mcp.WithDescription("自然语言记账（AI 生成待确认草稿）"),
		mcp.WithString("ledger_id", mcp.Required()),
		mcp.WithString("text", mcp.Required(), mcp.Description("自然语言描述，如：昨天星巴克咖啡 38 元"))),
		s.mcpHandler(s.aiRecord, "ai"))
	ms.AddTool(mcp.NewTool("ai_summarize",
		mcp.WithDescription("生成账本财务总结"),
		mcp.WithString("ledger_id", mcp.Required()),
		mcp.WithString("month", mcp.Description("YYYY-MM"))),
		s.mcpHandler(s.aiSummarize, "ai"))

	s.stream = mcpserver.NewStreamableHTTPServer(ms)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.stream.ServeHTTP(w, r)
}

func (s *Server) mcpHandler(fn func(context.Context, map[string]any) (any, error), minScope string) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		user, scope, ok := AuthFrom(ctx)
		if !ok {
			return mcp.NewToolResultError("未授权：缺少有效的 API Key（Bearer）"), nil
		}
		if !scopeAllowed(scope, minScope) {
			return mcp.NewToolResultErrorf("权限不足：当前 Key 范围 %s，需要 %s", scope, minScope), nil
		}
		args, _ := req.Params.Arguments.(map[string]any)
		data, err := fn(ctxWithUser(ctx, user), args)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, err := json.Marshal(data)
		if err != nil {
			return mcp.NewToolResultError("序列化失败：" + err.Error()), nil
		}
		return mcp.NewToolResultText(string(b)), nil
	}
}

func (s *Server) listLedgers(ctx context.Context, _ map[string]any) (any, error) {
	user := userFrom(ctx)
	ledgers, err := s.Store.ListLedgersForUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return ledgers, nil
}

func (s *Server) queryTransactions(ctx context.Context, args map[string]any) (any, error) {
	l, _, err := s.requireLedger(ctx, strArg(args, "ledger_id"), false)
	if err != nil {
		return nil, err
	}
	txns, err := s.Service.List(ctx, l, ledger.Filters{
		Month:   strArg(args, "month"),
		Account: strArg(args, "account"),
		Tag:     strArg(args, "tag"),
		Q:       strArg(args, "q"),
		From:    strArg(args, "from"),
		To:      strArg(args, "to"),
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"items": txns, "total": len(txns)}, nil
}

func (s *Server) queryAccounts(ctx context.Context, args map[string]any) (any, error) {
	l, _, err := s.requireLedger(ctx, strArg(args, "ledger_id"), false)
	if err != nil {
		return nil, err
	}
	return s.Service.ListAccounts(ctx, l, true)
}

func (s *Server) queryStats(ctx context.Context, args map[string]any) (any, error) {
	l, _, err := s.requireLedger(ctx, strArg(args, "ledger_id"), false)
	if err != nil {
		return nil, err
	}
	month := strArg(args, "month")
	account := strArg(args, "account")
	switch strArg(args, "kind") {
	case "total":
		return s.Service.StatsTotal(ctx, l, month, account)
	case "payee":
		return s.Service.StatsPayee(ctx, l, month, account, strArg(args, "type"))
	case "trend":
		return s.Service.StatsTrend(ctx, l, month, account, strArg(args, "type"))
	case "flow":
		return s.Service.StatsFlow(ctx, l, month, account)
	default:
		return nil, errors.New("kind 必须是 total / payee / trend / flow")
	}
}

func (s *Server) readSourceFile(ctx context.Context, args map[string]any) (any, error) {
	l, _, err := s.requireLedger(ctx, strArg(args, "ledger_id"), false)
	if err != nil {
		return nil, err
	}
	path, err := safeJoin(l.DataPath, strArg(args, "path"))
	if err != nil {
		return nil, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return map[string]string{"path": strArg(args, "path"), "content": string(content)}, nil
}

func (s *Server) createTransaction(ctx context.Context, args map[string]any) (any, error) {
	l, user, err := s.requireLedger(ctx, strArg(args, "ledger_id"), true)
	if err != nil {
		return nil, err
	}
	txn, err := transactionFromArgs(args)
	if err != nil {
		return nil, err
	}
	created, err := s.Service.Create(ctx, l, txn, l.Revision, ledger.Actor{UserID: user.ID, Login: user.GitHubLogin})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *Server) updateTransaction(ctx context.Context, args map[string]any) (any, error) {
	l, user, err := s.requireLedger(ctx, strArg(args, "ledger_id"), true)
	if err != nil {
		return nil, err
	}
	txn, err := transactionFromArgs(args)
	if err != nil {
		return nil, err
	}
	updated, err := s.Service.Update(ctx, l, strArg(args, "transaction_id"), txn, l.Revision,
		ledger.Actor{UserID: user.ID, Login: user.GitHubLogin})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Server) deleteTransaction(ctx context.Context, args map[string]any) (any, error) {
	l, user, err := s.requireLedger(ctx, strArg(args, "ledger_id"), true)
	if err != nil {
		return nil, err
	}
	if err := s.Service.Delete(ctx, l, strArg(args, "transaction_id"), l.Revision,
		ledger.Actor{UserID: user.ID, Login: user.GitHubLogin}); err != nil {
		return nil, err
	}
	return map[string]bool{"deleted": true}, nil
}

func (s *Server) writeSourceFile(ctx context.Context, args map[string]any) (any, error) {
	l, user, err := s.requireLedger(ctx, strArg(args, "ledger_id"), true)
	if err != nil {
		return nil, err
	}
	path, err := safeJoin(l.DataPath, strArg(args, "path"))
	if err != nil {
		return nil, err
	}
	if _, err := s.Store.CompareAndBumpRevision(ctx, l.ID, l.Revision); err != nil {
		return nil, fmt.Errorf("修订号过期：%w", err)
	}
	if err := os.WriteFile(path, []byte(strArg(args, "content")), 0o644); err != nil {
		return nil, err
	}
	if err := s.Store.InsertAuditLog(ctx, db.AuditParams{
		LedgerID: l.ID, UserID: user.ID, Actor: "mcp:" + user.GitHubLogin,
		Action: "write_source_file", Object: strArg(args, "path"),
	}); err != nil {
		return nil, err
	}
	return map[string]bool{"written": true}, nil
}

func (s *Server) aiRecord(ctx context.Context, args map[string]any) (any, error) {
	l, _, err := s.requireLedger(ctx, strArg(args, "ledger_id"), false)
	if err != nil {
		return nil, err
	}
	txn, notes, err := s.Service.AiRecord(ctx, l, strArg(args, "text"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"draft": txn, "notes": notes}, nil
}

func (s *Server) aiSummarize(ctx context.Context, args map[string]any) (any, error) {
	l, _, err := s.requireLedger(ctx, strArg(args, "ledger_id"), false)
	if err != nil {
		return nil, err
	}
	summary, err := s.Service.AiSummarize(ctx, l, strArg(args, "month"))
	if err != nil {
		return nil, err
	}
	return map[string]string{"summary": summary}, nil
}

func (s *Server) requireLedger(ctx context.Context, ledgerID string, write bool) (db.Ledger, db.User, error) {
	user := userFrom(ctx)
	l, err := s.Store.GetLedgerForUser(ctx, ledgerID, user.ID)
	if err != nil {
		return db.Ledger{}, db.User{}, errors.New("账本不存在或无权限")
	}
	if write && l.Role == "viewer" {
		return db.Ledger{}, db.User{}, errors.New("viewer 无写权限")
	}
	return *l, user, nil
}

func transactionFromArgs(args map[string]any) (ledger.Transaction, error) {
	var postings []ledger.Posting
	if err := json.Unmarshal([]byte(strArg(args, "postings")), &postings); err != nil {
		return ledger.Transaction{}, fmt.Errorf("postings 解析失败：%w", err)
	}
	t := ledger.Transaction{
		Date:      strArg(args, "date"),
		Payee:     strArg(args, "payee"),
		Narration: strArg(args, "narration"),
		Tags:      strSliceArg(args, "tags"),
		Postings:  postings,
	}
	if len(t.Postings) == 0 {
		return t, errors.New("postings 不能为空")
	}
	return t, nil
}

func safeJoin(root, rel string) (string, error) {
	if strings.TrimSpace(rel) == "" {
		return "", errors.New("path 不能为空")
	}
	cleanRoot := filepath.Clean(root)
	joined := filepath.Clean(filepath.Join(cleanRoot, rel))
	if joined != cleanRoot && !strings.HasPrefix(joined, cleanRoot+string(filepath.Separator)) {
		return "", errors.New("path 越界")
	}
	return joined, nil
}

func strArg(args map[string]any, key string) string {
	if v, ok := args[key].(string); ok {
		return v
	}
	return ""
}

func strSliceArg(args map[string]any, key string) []string {
	raw, ok := args[key].([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func scopeAllowed(scope, minScope string) bool {
	switch minScope {
	case "read":
		return scope != ""
	case "write":
		return scope == "read-write" || scope == "ai"
	case "ai":
		return scope == "ai"
	}
	return false
}
