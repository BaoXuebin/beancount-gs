package ledger

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

var (
	ErrNotBalanced = errors.New("transaction not balanced")
	ErrInvalidDate = errors.New("invalid transaction date")
	ErrNoPostings  = errors.New("transaction requires at least two postings")
)

// Validate 校验交易结构：日期、分录数量、借贷平衡。
func Validate(t Transaction) error {
	if _, err := time.Parse("2006-01-02", t.Date); err != nil {
		return ErrInvalidDate
	}
	if len(t.Postings) < 2 {
		return ErrNoPostings
	}
	for _, p := range t.Postings {
		if strings.TrimSpace(p.Account) == "" {
			return errors.New("posting account must not be empty")
		}
	}
	if !isBalanced(t) {
		return ErrNotBalanced
	}
	return nil
}

func isBalanced(t Transaction) bool {
	sum := decimal.Zero
	for _, p := range t.Postings {
		if p.Units == nil || p.Units.Number == "" {
			continue
		}
		num, err := decimal.NewFromString(strings.ReplaceAll(p.Units.Number, ",", ""))
		if err != nil {
			return false
		}
		if p.Price != nil && p.Price.Number != "" {
			price, err := decimal.NewFromString(strings.ReplaceAll(p.Price.Number, ",", ""))
			if err != nil {
				return false
			}
			sum = sum.Add(num.Mul(price))
		} else {
			sum = sum.Add(num)
		}
	}
	return sum.Abs().LessThanOrEqual(decimal.NewFromFloat(0.01))
}

// BuildBeanText 生成交易块文本，并返回需要写入 price 文件的指令行。
func BuildBeanText(t Transaction, operatingCurrency string) (string, []string, error) {
	if err := Validate(t); err != nil {
		return "", nil, err
	}
	flag := t.Flag
	if flag == "" {
		flag = "*"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s", t.Date, flag)
	if t.Payee != "" || t.Narration != "" {
		fmt.Fprintf(&b, " %q %q", cleanString(t.Payee), cleanString(t.Narration))
	}
	for _, tag := range t.Tags {
		if tag = strings.TrimSpace(tag); tag != "" {
			b.WriteString(" #" + tag)
		}
	}
	for _, link := range t.Links {
		if link = strings.TrimSpace(link); link != "" {
			b.WriteString(" ^" + link)
		}
	}
	for _, p := range t.Postings {
		b.WriteString("\n  " + p.Account)
		if p.Units != nil && p.Units.Number != "" {
			b.WriteString("  " + round2(p.Units.Number) + " " + p.Units.Currency)
		}
		if p.Cost != nil && p.Cost.Number != "" {
			b.WriteString(" {" + round2(p.Cost.Number) + " " + p.Cost.Currency)
			if p.Cost.Date != "" {
				b.WriteString(", " + p.Cost.Date)
			}
			b.WriteString("}")
		} else if p.Price != nil && p.Price.Number != "" {
			b.WriteString(" @ " + round2(p.Price.Number) + " " + p.Price.Currency)
		}
	}

	priceLines := make([]string, 0)
	for _, p := range t.Postings {
		if p.Units != nil && p.Price != nil && p.Price.Number != "" &&
			p.Units.Currency != "" && p.Units.Currency != operatingCurrency {
			priceLines = append(priceLines, fmt.Sprintf("%s price %s %s %s",
				t.Date, p.Units.Currency, round2(p.Price.Number), p.Price.Currency))
		}
	}
	return b.String(), priceLines, nil
}

func cleanString(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), `"`, "")
}

func round2(s string) string {
	d, err := decimal.NewFromString(strings.ReplaceAll(s, ",", ""))
	if err != nil {
		return s
	}
	return d.Round(2).StringFixedBank(2)
}
