package httpapi

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/beancount-gs/api/internal/ai"
	"github.com/beancount-gs/api/internal/db"
	"github.com/beancount-gs/api/internal/http/gen"
	"github.com/beancount-gs/api/internal/ledger"
	"github.com/gin-gonic/gin"
)

type AIHandlers struct {
	Store   *db.Store
	Service *ledger.Service
}

func (h *AIHandlers) Record(c *gin.Context) {
	l, ok := requireLedger(c, h.Store, "editor")
	if !ok {
		return
	}
	var form struct {
		Text string `json:"text" binding:"required"`
	}
	if err := c.ShouldBindJSON(&form); err != nil {
		BadRequest(c, "参数错误："+err.Error())
		return
	}
	txn, notes, err := h.Service.AiRecord(c.Request.Context(), *l, form.Text)
	if err != nil {
		h.writeAIError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"draft": toGenTransaction(txn), "notes": notes})
}

func (h *AIHandlers) RecordBatch(c *gin.Context) {
	l, ok := requireLedger(c, h.Store, "editor")
	if !ok {
		return
	}
	var form struct {
		Text string `json:"text" binding:"required"`
	}
	if err := c.ShouldBindJSON(&form); err != nil {
		BadRequest(c, "参数错误："+err.Error())
		return
	}
	txns, notes, err := h.Service.AiRecordBatch(c.Request.Context(), *l, form.Text)
	if err != nil {
		h.writeAIError(c, err)
		return
	}
	drafts := make([]gen.Transaction, 0, len(txns))
	for _, t := range txns {
		drafts = append(drafts, toGenTransaction(t))
	}
	c.JSON(http.StatusOK, gin.H{"drafts": drafts, "notes": notes})
}

func (h *AIHandlers) Chat(c *gin.Context) {
	l, ok := requireLedger(c, h.Store, "editor")
	if !ok {
		return
	}
	var form struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		Drafts []gen.Transaction `json:"drafts"`
	}
	if err := c.ShouldBindJSON(&form); err != nil {
		BadRequest(c, "参数错误："+err.Error())
		return
	}
	messages := make([]ledger.ChatMessage, 0, len(form.Messages))
	for _, m := range form.Messages {
		if m.Content == "" {
			continue
		}
		messages = append(messages, ledger.ChatMessage{Role: m.Role, Content: m.Content})
	}
	if len(messages) == 0 {
		BadRequest(c, "对话内容不能为空")
		return
	}
	drafts := make([]ledger.Transaction, 0, len(form.Drafts))
	for _, d := range form.Drafts {
		drafts = append(drafts, fromGenDraft(d))
	}
	txns, notes, err := h.Service.AiRecordChat(c.Request.Context(), *l, messages, drafts)
	if err != nil {
		h.writeAIError(c, err)
		return
	}
	out := make([]gen.Transaction, 0, len(txns))
	for _, t := range txns {
		out = append(out, toGenTransaction(t))
	}
	c.JSON(http.StatusOK, gin.H{"drafts": out, "notes": notes})
}

// fromGenDraft 把 gen 草稿转换回内部模型（用于多轮调整时把当前草稿回传给 AI）。
func fromGenDraft(t gen.Transaction) ledger.Transaction {
	out := ledger.Transaction{Date: t.Date.String()}
	if t.Payee != nil {
		out.Payee = *t.Payee
	}
	if t.Narration != nil {
		out.Narration = *t.Narration
	}
	for _, p := range t.Postings {
		lp := ledger.Posting{Account: p.Account}
		if p.Units != nil {
			lp.Units = &ledger.Amount{Number: p.Units.Number, Currency: p.Units.Currency}
		}
		out.Postings = append(out.Postings, lp)
	}
	return out
}

func (h *AIHandlers) Accounts(c *gin.Context) {
	l, ok := requireLedger(c, h.Store, "editor")
	if !ok {
		return
	}
	var form struct {
		Text string `json:"text" binding:"required"`
	}
	if err := c.ShouldBindJSON(&form); err != nil {
		BadRequest(c, "参数错误："+err.Error())
		return
	}
	accounts, notes, err := h.Service.AiAccounts(c.Request.Context(), *l, form.Text)
	if err != nil {
		h.writeAIError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"accounts": accounts, "notes": notes})
}
func (h *AIHandlers) Summarize(c *gin.Context) {
	l, ok := requireLedger(c, h.Store, "")
	if !ok {
		return
	}
	summary, err := h.Service.AiSummarize(c.Request.Context(), *l, c.Query("month"))
	if err != nil {
		h.writeAIError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"summary": summary})
}

func (h *AIHandlers) Insights(c *gin.Context) {
	l, ok := requireLedger(c, h.Store, "")
	if !ok {
		return
	}
	insights, err := h.Service.AiInsights(c.Request.Context(), *l, c.Query("month"))
	if err != nil {
		slog.Error("ai insights failed", "err", err)
		Error(c, http.StatusInternalServerError, "INTERNAL", "洞察分析失败", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"insights": insights})
}

func (h *AIHandlers) writeAIError(c *gin.Context, err error) {
	if errors.Is(err, ai.ErrNotConfigured) {
		Error(c, http.StatusServiceUnavailable, "AI_NOT_CONFIGURED", err.Error(), nil)
		return
	}
	slog.Error("ai request failed", "err", err)
	Error(c, http.StatusInternalServerError, "INTERNAL", "AI 请求失败："+err.Error(), nil)
}
