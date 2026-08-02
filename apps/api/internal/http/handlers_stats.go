package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/beancount-gs/api/internal/db"
	"github.com/beancount-gs/api/internal/http/gen"
	"github.com/beancount-gs/api/internal/ledger"
	"github.com/gin-gonic/gin"
)

type StatsHandlers struct {
	Store   *db.Store
	Service *ledger.Service
}

func (h *StatsHandlers) Total(c *gin.Context) {
	l, ok := requireLedger(c, h.Store, "")
	if !ok {
		return
	}
	result, err := h.Service.StatsTotal(c.Request.Context(), *l, c.Query("month"), c.Query("account"))
	if err != nil {
		slog.Error("stats total failed", "err", err)
		Error(c, http.StatusInternalServerError, "INTERNAL", "统计失败", nil)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *StatsHandlers) Payee(c *gin.Context) {
	l, ok := requireLedger(c, h.Store, "")
	if !ok {
		return
	}
	stats, err := h.Service.StatsPayee(c.Request.Context(), *l,
		c.Query("month"), c.Query("account"), c.DefaultQuery("type", "total"))
	if err != nil {
		slog.Error("stats payee failed", "err", err)
		Error(c, http.StatusInternalServerError, "INTERNAL", "统计失败", nil)
		return
	}
	result := make([]gen.StatsPayee, 0, len(stats))
	for _, s := range stats {
		result = append(result, gen.StatsPayee{
			Payee:  strPtr(s.Payee),
			Count:  intPtr(s.Count),
			Amount: strPtr(s.Amount),
		})
	}
	c.JSON(http.StatusOK, result)
}

func (h *StatsHandlers) Trend(c *gin.Context) {
	l, ok := requireLedger(c, h.Store, "")
	if !ok {
		return
	}
	points, err := h.Service.StatsTrend(c.Request.Context(), *l,
		c.Query("month"), c.Query("account"), c.DefaultQuery("type", "month"))
	if err != nil {
		slog.Error("stats trend failed", "err", err)
		Error(c, http.StatusInternalServerError, "INTERNAL", "统计失败", nil)
		return
	}
	result := make([]gen.StatsPoint, 0, len(points))
	for _, p := range points {
		result = append(result, gen.StatsPoint{
			Date:              strPtr(p.Date),
			Amount:            strPtr(p.Amount),
			OperatingCurrency: strPtr(p.Currency),
		})
	}
	c.JSON(http.StatusOK, result)
}

func (h *StatsHandlers) Flow(c *gin.Context) {
	l, ok := requireLedger(c, h.Store, "")
	if !ok {
		return
	}
	flow, err := h.Service.StatsFlow(c.Request.Context(), *l, c.Query("month"), c.Query("account"))
	if err != nil {
		slog.Error("stats flow failed", "err", err)
		Error(c, http.StatusInternalServerError, "INTERNAL", "统计失败", nil)
		return
	}
	nodes := make([]gin.H, 0, len(flow.Nodes))
	for _, n := range flow.Nodes {
		nodes = append(nodes, gin.H{"name": n})
	}
	links := make([]gin.H, 0, len(flow.Links))
	for _, lk := range flow.Links {
		links = append(links, gin.H{"source": lk.Source, "target": lk.Target, "value": lk.Value})
	}
	c.JSON(http.StatusOK, gin.H{"nodes": nodes, "links": links})
}
