package httpapi

import (
	"time"

	"github.com/beancount-gs/api/internal/ledger"
	"github.com/beancount-gs/api/internal/http/gen"
	"github.com/oapi-codegen/runtime/types"
)

func toGenTransaction(t ledger.Transaction) gen.Transaction {
	flag := t.Flag
	if flag == "" {
		flag = "*"
	}
	postings := make([]gen.Posting, 0, len(t.Postings))
	for _, p := range t.Postings {
		gp := gen.Posting{Account: p.Account}
		if p.Units != nil {
			gp.Units = &gen.Amount{Number: p.Units.Number, Currency: p.Units.Currency}
		}
		if p.Cost != nil {
			cost := &gen.Cost{}
			if p.Cost.Number != "" {
				cost.Number = strPtr(p.Cost.Number)
			}
			if p.Cost.Currency != "" {
				cost.Currency = strPtr(p.Cost.Currency)
			}
			if p.Cost.Date != "" {
				if d, err := time.Parse("2006-01-02", p.Cost.Date); err == nil {
					cost.Date = &types.Date{Time: d}
				}
			}
			gp.Cost = cost
		}
		if p.Price != nil {
			gp.Price = &gen.Amount{Number: p.Price.Number, Currency: p.Price.Currency}
		}
		postings = append(postings, gp)
	}
	date, _ := time.Parse("2006-01-02", t.Date)
	f := gen.TransactionFlag(flag)
	tags := t.Tags
	links := t.Links
	return gen.Transaction{
		Id:        t.ID,
		Date:      types.Date{Time: date},
		Flag:      &f,
		Payee:     optionalStr(t.Payee),
		Narration: optionalStr(t.Narration),
		Tags:      optionalList(tags),
		Links:     optionalList(links),
		Postings:  postings,
	}
}

func fromGenTransactionCreate(form gen.TransactionCreate) ledger.Transaction {
	t := ledger.Transaction{
		Date: form.Date.String(),
		Flag: "*",
	}
	if form.Flag != nil && string(*form.Flag) != "" {
		t.Flag = string(*form.Flag)
	}
	if form.Payee != nil {
		t.Payee = *form.Payee
	}
	if form.Narration != nil {
		t.Narration = *form.Narration
	}
	if form.Tags != nil {
		t.Tags = *form.Tags
	}
	if form.Links != nil {
		t.Links = *form.Links
	}
	for _, gp := range form.Postings {
		p := ledger.Posting{Account: gp.Account}
		if gp.Units != nil {
			p.Units = &ledger.Amount{Number: gp.Units.Number, Currency: gp.Units.Currency}
		}
		if gp.Cost != nil {
			cost := &ledger.Cost{}
			if gp.Cost.Number != nil {
				cost.Number = *gp.Cost.Number
			}
			if gp.Cost.Currency != nil {
				cost.Currency = *gp.Cost.Currency
			}
			if gp.Cost.Date != nil {
				cost.Date = gp.Cost.Date.String()
			}
			if gp.Cost.Label != nil {
				cost.Label = *gp.Cost.Label
			}
			p.Cost = cost
		}
		if gp.Price != nil {
			p.Price = &ledger.Amount{Number: gp.Price.Number, Currency: gp.Price.Currency}
		}
		t.Postings = append(t.Postings, p)
	}
	return t
}

func optionalStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func optionalList(items []string) *[]string {
	if len(items) == 0 {
		return nil
	}
	return &items
}

func toGenAccount(a ledger.Account) gen.Account {
	var openedOn, closedOn *types.Date
	if a.OpenedOn != "" {
		if d, err := time.Parse("2006-01-02", a.OpenedOn); err == nil {
			openedOn = &types.Date{Time: d}
		}
	}
	if a.ClosedOn != "" {
		if d, err := time.Parse("2006-01-02", a.ClosedOn); err == nil {
			closedOn = &types.Date{Time: d}
		}
	}
	var positions *[]struct {
		Currency       *string `json:"currency,omitempty"`
		CurrencySymbol *string `json:"currency_symbol,omitempty"`
		Number         *string `json:"number,omitempty"`
	}
	if len(a.Positions) > 0 {
		list := make([]struct {
			Currency       *string `json:"currency,omitempty"`
			CurrencySymbol *string `json:"currency_symbol,omitempty"`
			Number         *string `json:"number,omitempty"`
		}, 0, len(a.Positions))
		for _, p := range a.Positions {
			list = append(list, struct {
				Currency       *string `json:"currency,omitempty"`
				CurrencySymbol *string `json:"currency_symbol,omitempty"`
				Number         *string `json:"number,omitempty"`
			}{
				Number:   strPtr(p.Number),
				Currency: strPtr(p.Currency),
			})
		}
		positions = &list
	}
	return gen.Account{
		Account:        a.Name,
		Type:           gen.AccountType(a.Type),
		Status:         gen.AccountStatus(a.Status),
		OpenedOn:       openedOn,
		ClosedOn:       closedOn,
		Currency:       optionalStr(a.Currency),
		Positions:      positions,
		MarketNumber:   optionalStr(a.MarketNumber),
		MarketCurrency: optionalStr(a.MarketCurrency),
	}
}


func toGenCurrencies(items []ledger.Currency) []gen.Currency {
	out := make([]gen.Currency, 0, len(items))
	for _, c := range items {
		gc := gen.Currency{Currency: c.Code, Name: c.Name}
		if c.IsOperating {
			gc.IsOperating = boolPtr(true)
		}
		if c.Symbol != "" {
			gc.Symbol = strPtr(c.Symbol)
		}
		if c.Price != "" {
			gc.Price = strPtr(c.Price)
			if d, err := time.Parse("2006-01-02", c.PriceDate); err == nil {
				gc.PriceDate = &types.Date{Time: d}
			}
		}
		out = append(out, gc)
	}
	return out
}
