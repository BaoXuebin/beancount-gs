package ledger

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/beancount-gs/api/internal/db"
)

// Months 返回账本中有交易的月份列表（"YYYY-MM"），按时间倒序（最近在前）。
// 数据来自账本实际交易，参考 v1 的 /stats/months 实现。
func (s *Service) Months(ctx context.Context, l db.Ledger) ([]string, error) {
	rows, err := s.Engine.QueryCSV(ctx, indexPath(l),
		"SELECT distinct year(date) AS year, month(date) AS month")
	if err != nil {
		return nil, err
	}
	type ym struct {
		year, month int
	}
	seen := make(map[ym]bool)
	list := make([]ym, 0, len(rows))
	for _, r := range rows {
		y, errY := strconv.Atoi(strings.TrimSpace(r["year"]))
		m, errM := strconv.Atoi(strings.TrimSpace(r["month"]))
		if errY != nil || errM != nil || y <= 0 || m < 1 || m > 12 {
			continue
		}
		key := ym{y, m}
		if seen[key] {
			continue
		}
		seen[key] = true
		list = append(list, key)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].year != list[j].year {
			return list[i].year > list[j].year
		}
		return list[i].month > list[j].month
	})
	result := make([]string, 0, len(list))
	for _, v := range list {
		result = append(result, fmt.Sprintf("%d-%02d", v.year, v.month))
	}
	return result, nil
}
