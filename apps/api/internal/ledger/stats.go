package ledger

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/beancount-gs/api/internal/db"
	"github.com/shopspring/decimal"
)

type PayeeStat struct {
	Payee  string
	Count  int
	Amount string
}

type TrendPoint struct {
	Date     string
	Amount   string
	Currency string
}

type FlowLink struct {
	Source int
	Target int
	Value  string
}

type FlowResult struct {
	Nodes []string
	Links []FlowLink
}

func statsWhere(month, account string) string {
	wheres := make([]string, 0, 2)
	if from, to, ok := monthRange(month); ok {
		wheres = append(wheres, "date >= "+from+" AND date <= "+to)
	}
	if account != "" {
		wheres = append(wheres, "account ~ '^"+strings.ReplaceAll(account, "'", "")+"'")
	}
	if len(wheres) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(wheres, " AND ")
}

// StatsTotal 按账户根类型（Assets/Liabilities/...）汇总金额。
func (s *Service) StatsTotal(ctx context.Context, l db.Ledger, month, account string) (map[string]string, error) {
	query := "SELECT root(account, 1) AS account_type, sum(convert(value(position), '" + l.OperatingCurrency +
		"')) AS total" + statsWhere(month, account)
	rows, err := s.Engine.QueryCSV(ctx, indexPath(l), query)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string)
	for _, r := range rows {
		typ := strings.TrimSpace(r["account_type"])
		total := strings.TrimSpace(r["total"])
		if typ == "" || total == "" {
			continue
		}
		fields := strings.Fields(total)
		result[typ] = fields[0]
	}
	return result, nil
}

// StatsPayee 收款方统计；kind: total | count | avg。
func (s *Service) StatsPayee(ctx context.Context, l db.Ledger, month, account, kind string) ([]PayeeStat, error) {
	query := "SELECT payee, count(payee) AS cnt, sum(convert(value(position), '" + l.OperatingCurrency +
		"')) AS total" + statsWhere(month, account)
	rows, err := s.Engine.QueryCSV(ctx, indexPath(l), query)
	if err != nil {
		return nil, err
	}
	result := make([]PayeeStat, 0)
	for _, r := range rows {
		payee := strings.TrimSpace(r["payee"])
		if payee == "" {
			continue
		}
		count, _ := strconv.Atoi(strings.TrimSpace(r["cnt"]))
		amount := ""
		totalFields := strings.Fields(strings.TrimSpace(r["total"]))
		switch kind {
		case "count":
			amount = strconv.Itoa(count)
		case "avg":
			if len(totalFields) > 0 && count > 0 {
				if total, err := decimal.NewFromString(totalFields[0]); err == nil {
					amount = total.Div(decimal.NewFromInt(int64(count))).Round(2).StringFixed(2)
				}
			}
		default:
			if len(totalFields) > 0 {
				amount = totalFields[0]
			}
		}
		result = append(result, PayeeStat{Payee: payee, Count: count, Amount: amount})
	}
	sort.Slice(result, func(i, j int) bool {
		a, _ := decimal.NewFromString(result[i].Amount)
		b, _ := decimal.NewFromString(result[j].Amount)
		return a.GreaterThan(b)
	})
	return result, nil
}

// StatsTrend 账户余额趋势；kind: day | month | year | sum。
func (s *Service) StatsTrend(ctx context.Context, l db.Ledger, month, account, kind string) ([]TrendPoint, error) {
	var selectCols string
	dateFromRow := func(r map[string]string) string { return r["date"] }
	switch kind {
	case "day":
		selectCols = "SELECT date, sum(convert(value(position), '" + l.OperatingCurrency + "')) AS amount"
	case "month":
		selectCols = "SELECT year AS y, month AS m, sum(convert(value(position), '" + l.OperatingCurrency + "')) AS amount"
		dateFromRow = func(r map[string]string) string {
			return fmt.Sprintf("%04d-%02d", atoi(r["y"]), atoi(r["m"]))
		}
	case "year":
		selectCols = "SELECT year AS y, sum(convert(value(position), '" + l.OperatingCurrency + "')) AS amount"
		dateFromRow = func(r map[string]string) string { return r["y"] }
	case "sum":
		selectCols = "SELECT date, convert(balance, '" + l.OperatingCurrency + "') AS amount"
	default:
		return []TrendPoint{}, nil
	}
	rows, err := s.Engine.QueryCSV(ctx, indexPath(l), selectCols+statsWhere(month, account))
	if err != nil {
		return nil, err
	}
	result := make([]TrendPoint, 0, len(rows))
	for _, r := range rows {
		amountField := strings.TrimSpace(r["amount"])
		if amountField == "" {
			continue
		}
		fields := strings.Fields(amountField)
		currency := l.OperatingCurrency
		if len(fields) >= 2 {
			currency = fields[1]
		}
		result = append(result, TrendPoint{
			Date:     dateFromRow(r),
			Amount:   fields[0],
			Currency: currency,
		})
	}
	return result, nil
}

// StatsFlow 账户资金流向（桑基图数据：节点 + 边，含环处理）。
func (s *Service) StatsFlow(ctx context.Context, l db.Ledger, month, account string) (FlowResult, error) {
	query := "SELECT id, account, sum(convert(value(position), '" + l.OperatingCurrency +
		"')) AS position" + statsWhere(month, account)
	rows, err := s.Engine.QueryCSV(ctx, indexPath(l), query)
	if err != nil {
		return FlowResult{}, err
	}
	nodeIndex := make(map[string]int)
	nodes := make([]string, 0)
	txnPostings := make(map[string][]flowPosting)
	for _, r := range rows {
		id := r["id"]
		account := r["account"]
		position := strings.TrimSpace(r["position"])
		if id == "" || account == "" || position == "" {
			continue
		}
		if _, ok := nodeIndex[account]; !ok {
			nodeIndex[account] = len(nodes)
			nodes = append(nodes, account)
		}
		amount, err := decimal.NewFromString(strings.Fields(position)[0])
		if err != nil {
			continue
		}
		txnPostings[id] = append(txnPostings[id], flowPosting{account: account, amount: amount})
	}
	links := make([]FlowLink, 0)
	for _, postings := range txnPostings {
		links = append(links, buildFlowLinks(postings, nodeIndex)...)
	}
	links = aggregateFlowLinks(links)
	nodes, links = breakFlowCycles(nodes, links)
	return FlowResult{Nodes: nodes, Links: links}, nil
}

type flowPosting struct {
	account string
	amount  decimal.Decimal
}

func buildFlowLinks(postings []flowPosting, nodeIndex map[string]int) []FlowLink {
	var negatives, positives []flowPosting
	for _, p := range postings {
		if p.amount.IsNegative() {
			negatives = append(negatives, flowPosting{account: p.account, amount: p.amount.Abs()})
		} else if p.amount.IsPositive() {
			positives = append(positives, p)
		}
	}
	links := make([]FlowLink, 0)
	guard := len(negatives) + len(positives) + 1
	for len(negatives) > 0 && len(positives) > 0 && guard > 0 {
		n := negatives[0]
		p := positives[0]
		negatives = negatives[1:]
		positives = positives[1:]
		delta := n.amount.Sub(p.amount)
		value := p.amount
		if delta.IsNegative() {
			value = n.amount
			positives = append(positives, flowPosting{account: p.account, amount: p.amount.Sub(n.amount)})
		} else if delta.IsPositive() {
			value = p.amount
			negatives = append(negatives, flowPosting{account: n.account, amount: n.amount.Sub(p.amount)})
		}
		src, dst := nodeIndex[n.account], nodeIndex[p.account]
		if src != dst {
			links = append(links, FlowLink{Source: src, Target: dst, Value: value.Round(2).String()})
		}
		guard--
	}
	return links
}

func aggregateFlowLinks(links []FlowLink) []FlowLink {
	sums := make(map[[2]int]decimal.Decimal)
	for _, l := range links {
		key := [2]int{l.Source, l.Target}
		if v, err := decimal.NewFromString(l.Value); err == nil {
			sums[key] = sums[key].Add(v)
		}
	}
	result := make([]FlowLink, 0, len(sums))
	for key, v := range sums {
		result = append(result, FlowLink{Source: key[0], Target: key[1], Value: v.Round(2).String()})
	}
	return result
}

// breakFlowCycles 通过克隆节点打断循环，保证桑基图可渲染（至多 10 轮）。
func breakFlowCycles(nodes []string, links []FlowLink) ([]string, []FlowLink) {
	for i := 0; i < 10; i++ {
		back, idx, ok := findBackEdge(links)
		if !ok {
			break
		}
		clone := len(nodes)
		nodes = append(nodes, nodes[back.Target]+"(循环)")
		links[idx].Target = clone
	}
	return nodes, links
}

func findBackEdge(links []FlowLink) (FlowLink, int, bool) {
	adj := make(map[int][]int)
	edgeIdx := make(map[[2]int]int)
	for i, l := range links {
		adj[l.Source] = append(adj[l.Source], l.Target)
		edgeIdx[[2]int{l.Source, l.Target}] = i
	}
	visited := make(map[int]bool)
	recStack := make(map[int]bool)
	var dfs func(int) (int, int, bool)
	dfs = func(n int) (int, int, bool) {
		visited[n] = true
		recStack[n] = true
		for _, t := range adj[n] {
			if recStack[t] {
				return n, t, true
			}
			if !visited[t] {
				if s, e, ok := dfs(t); ok {
					return s, e, true
				}
			}
		}
		delete(recStack, n)
		return 0, 0, false
	}
	for _, l := range links {
		if !visited[l.Source] {
			if s, t, ok := dfs(l.Source); ok {
				idx := edgeIdx[[2]int{s, t}]
				return links[idx], idx, true
			}
		}
	}
	return FlowLink{}, -1, false
}

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}
