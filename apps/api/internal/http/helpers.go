package httpapi

import (
	"time"

	"github.com/beancount-gs/api/internal/db"
	"github.com/beancount-gs/api/internal/http/gen"
	"github.com/oapi-codegen/runtime/types"
)

func strPtr(s string) *string { return &s }

func boolPtr(b bool) *bool { return &b }

func intPtr(i int) *int { return &i }

func parseTime(s string) *time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}

func toGenTeam(t db.Team) gen.Team {
	return gen.Team{
		Id:          t.ID,
		Name:        t.Name,
		Role:        gen.Role(t.Role),
		MemberCount: intPtr(t.MemberCount),
		LedgerCount: intPtr(t.LedgerCount),
		CreatedAt:   parseTime(t.CreatedAt),
	}
}

func toGenLedger(l db.Ledger) gen.Ledger {
	var start *types.Date
	if l.StartDate != "" {
		if t, err := time.Parse("2006-01-02", l.StartDate); err == nil {
			start = &types.Date{Time: t}
		}
	}
	return gen.Ledger{
		Id:                l.ID,
		TeamId:            strPtr(l.TeamID),
		Name:              l.Name,
		DataPath:          strPtr(l.DataPath),
		OperatingCurrency: l.OperatingCurrency,
		StartDate:         start,
		OpeningBalances:   strPtr(l.OpeningBalances),
		IsBak:             boolPtr(l.IsBak),
		Revision:          int(l.Revision),
		Role:              gen.Role(l.Role),
		MemberCount:       intPtr(l.MemberCount),
		CreatedAt:         parseTime(l.CreatedAt),
	}
}
