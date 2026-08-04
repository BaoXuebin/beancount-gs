package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/beancount-gs/api/internal/db"
	"github.com/beancount-gs/api/internal/repository"
)

// Currency 币种与最新汇率（内部模型）。
type Currency struct {
	Code        string
	Name        string
	Symbol      string
	IsOperating bool
	Price       string
	PriceDate   string
}

// FXProvider 拉取以 base 为本位币的最新汇率（币种 → 汇率）。
type FXProvider func(ctx context.Context, base string) (map[string]float64, error)

var (
	priceLineRe    = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\s+price\s+(\S+)\s+([0-9.]+)\s+(\S+)$`)
	currencyCodeRe = regexp.MustCompile(`^[A-Za-z0-9]{1,8}$`)
)

type currencyMeta struct {
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
}

type currencyMetaFile struct {
	Name     string `json:"name"`
	Currency string `json:"currency"`
	Symbol   string `json:"symbol"`
}

// ListCurrencies 汇总账本币种：本位币 + 元数据(currency.json) + price 指令 + 账户 open 指令中的币种。
// 每个币种附带相对本位币的最新汇率（取 price/prices.bean 中日期最新的一条）。
func (s *Service) ListCurrencies(ctx context.Context, l db.Ledger) ([]Currency, error) {
	operating := strings.ToUpper(strings.TrimSpace(l.OperatingCurrency))
	if operating == "" {
		operating = "CNY"
	}
	meta, err := readCurrencyMeta(l.DataPath)
	if err != nil {
		return nil, err
	}
	prices, err := readLatestPrices(l.DataPath, operating)
	if err != nil {
		return nil, err
	}
	directives, err := repository.ReadAccountFiles(l.DataPath)
	if err != nil {
		return nil, err
	}
	codes := map[string]bool{operating: true}
	for _, d := range directives {
		for _, c := range strings.Split(d.Currency, ",") {
			c = strings.ToUpper(strings.TrimSpace(c))
			if c != "" {
				codes[c] = true
			}
		}
	}
	for c := range meta {
		codes[c] = true
	}
	for c := range prices {
		codes[c] = true
	}
	list := make([]Currency, 0, len(codes))
	for code := range codes {
		m := meta[code]
		p := prices[code]
		list = append(list, Currency{
			Code:        code,
			Name:        m.Name,
			Symbol:      m.Symbol,
			IsOperating: code == operating,
			Price:       p.Price,
			PriceDate:   p.PriceDate,
		})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Code < list[j].Code })
	return list, nil
}

// AddCurrency 新增币种到 .beancount-gs/currency.json（名称 / 符号元数据）。
func (s *Service) AddCurrency(ctx context.Context, l db.Ledger, code, name, symbol string, expectedRevision int64, actor Actor) error {
	unlock := s.lockLedger(l.ID)
	defer unlock()
	code = strings.ToUpper(strings.TrimSpace(code))
	name = strings.TrimSpace(name)
	symbol = strings.TrimSpace(symbol)
	if code == "" || name == "" {
		return errors.New("币种代码和名称不能为空")
	}
	if !currencyCodeRe.MatchString(code) {
		return errors.New("币种代码须为 1-8 位字母或数字")
	}
	if _, err := s.Store.CompareAndBumpRevision(ctx, l.ID, expectedRevision); err != nil {
		return err
	}
	if err := writeCurrencyMeta(l.DataPath, code, name, symbol); err != nil {
		return err
	}
	return s.audit(ctx, l, actor, "add_currency", code)
}

// SyncCurrencies 拉取最新汇率并写入 price/prices.bean（仅更新账本已有币种）。
// 若没有任何新价格需要写入，则不 bump 修订号。
func (s *Service) SyncCurrencies(ctx context.Context, l db.Ledger, expectedRevision int64, actor Actor) ([]Currency, error) {
	unlock := s.lockLedger(l.ID)
	defer unlock()
	operating := strings.ToUpper(strings.TrimSpace(l.OperatingCurrency))
	if operating == "" {
		operating = "CNY"
	}
	fetch := s.FX
	if fetch == nil {
		fetch = defaultFetchRates
	}
	rates, err := fetch(ctx, operating)
	if err != nil {
		return nil, fmt.Errorf("拉取汇率失败: %w", err)
	}
	currencies, err := s.ListCurrencies(ctx, l)
	if err != nil {
		return nil, err
	}
	date := s.nowDate()
	lines := make([]string, 0, len(currencies))
	for _, c := range currencies {
		if c.IsOperating {
			continue
		}
		rate, ok := rates[c.Code]
		if !ok || rate <= 0 {
			continue
		}
		price := formatRate(rate)
		if c.PriceDate == date && c.Price == price {
			continue // 当日已是最新价格，跳过
		}
		lines = append(lines, fmt.Sprintf("%s price %s %s %s", date, c.Code, price, operating))
	}
	if len(lines) == 0 {
		return currencies, nil
	}
	if _, err := s.Store.CompareAndBumpRevision(ctx, l.ID, expectedRevision); err != nil {
		return nil, err
	}
	for _, line := range lines {
		if err := repository.AppendPrice(l.DataPath, line); err != nil {
			return nil, err
		}
	}
	if err := s.audit(ctx, l, actor, "sync_currencies", fmt.Sprintf("%d prices", len(lines))); err != nil {
		return nil, err
	}
	return s.ListCurrencies(ctx, l)
}

func (s *Service) nowDate() string {
	if s.Now != nil {
		return s.Now().Format("2006-01-02")
	}
	return time.Now().Format("2006-01-02")
}

func formatRate(r float64) string {
	return fmt.Sprintf("%.6f", r)
}

// readCurrencyMeta 读取 .beancount-gs/currency.json（不存在则返回空表）。
func readCurrencyMeta(dataPath string) (map[string]currencyMeta, error) {
	result := map[string]currencyMeta{}
	path := filepath.Join(dataPath, ".beancount-gs", "currency.json")
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return nil, err
	}
	var list []currencyMetaFile
	if err := json.Unmarshal(content, &list); err != nil {
		return nil, fmt.Errorf("parse currency.json: %w", err)
	}
	for _, item := range list {
		code := strings.ToUpper(strings.TrimSpace(item.Currency))
		if code != "" {
			result[code] = currencyMeta{Name: item.Name, Symbol: item.Symbol}
		}
	}
	return result, nil
}

// writeCurrencyMeta 合并写入单个币种元数据（相同代码覆盖）。
func writeCurrencyMeta(dataPath, code, name, symbol string) error {
	meta, err := readCurrencyMeta(dataPath)
	if err != nil {
		return err
	}
	meta[code] = currencyMeta{Name: name, Symbol: symbol}
	list := make([]currencyMetaFile, 0, len(meta))
	for c, m := range meta {
		list = append(list, currencyMetaFile{Currency: c, Name: m.Name, Symbol: m.Symbol})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Currency < list[j].Currency })
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Join(dataPath, ".beancount-gs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "currency.json"), data, 0o644)
}

// readLatestPrices 解析 price/prices.bean，返回每个币种相对 operating 的最新价格。
func readLatestPrices(dataPath, operating string) (map[string]struct{ Price, PriceDate string }, error) {
	result := map[string]struct{ Price, PriceDate string }{}
	path := filepath.Join(dataPath, "price", "prices.bean")
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return nil, err
	}
	for _, line := range strings.Split(string(content), "\n") {
		m := priceLineRe.FindStringSubmatch(strings.TrimSpace(line))
		if len(m) != 5 || m[4] != operating {
			continue
		}
		code := m[2]
		date := m[1]
		cur, ok := result[code]
		if !ok || date > cur.PriceDate {
			result[code] = struct{ Price, PriceDate string }{Price: m[3], PriceDate: date}
		}
	}
	return result, nil
}

// defaultFetchRates 从公开汇率接口拉取以 base 为本位币的汇率：
// 优先 open.er-api.com（覆盖面广），失败时回退 api.frankfurter.app（ECB）。
func defaultFetchRates(ctx context.Context, base string) (map[string]float64, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	if rates, err := fetchERAPI(ctx, client, base); err == nil {
		return rates, nil
	}
	return fetchFrankfurter(ctx, client, base)
}

func fetchERAPI(ctx context.Context, client *http.Client, base string) (map[string]float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://open.er-api.com/v6/latest/"+base, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var body struct {
		Result string             `json:"result"`
		Rates  map[string]float64 `json:"rates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	if body.Result != "success" || len(body.Rates) == 0 {
		return nil, errors.New("er-api: unexpected response")
	}
	return body.Rates, nil
}

func fetchFrankfurter(ctx context.Context, client *http.Client, base string) (map[string]float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://api.frankfurter.app/latest?from="+base, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var body struct {
		Rates map[string]float64 `json:"rates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	if len(body.Rates) == 0 {
		return nil, errors.New("frankfurter: unexpected response")
	}
	return body.Rates, nil
}
