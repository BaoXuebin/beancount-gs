package httpapi

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/beancount-gs/api/internal/ai"
	"github.com/beancount-gs/api/internal/db"
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
