package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type NewLedgerParams struct {
	ID                string
	TeamID            string
	Name              string
	DataPath          string
	OperatingCurrency string
	StartDate         string
	OpeningBalances   string
	IsBak             bool
}

const ledgerSelect = `
	SELECT l.id, l.team_id, l.name, l.data_path, l.operating_currency,
		COALESCE(l.start_date, ''), l.opening_balances, l.is_bak, l.revision, l.created_at,
		lm.role,
		(SELECT COUNT(*) FROM ledger_members lm2 WHERE lm2.ledger_id = l.id) AS member_count
	FROM ledgers l
	JOIN ledger_members lm ON lm.ledger_id = l.id`

func (s *Store) CreateLedgerWithOwner(ctx context.Context, p NewLedgerParams, ownerUserID string) (Ledger, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	if p.OperatingCurrency == "" {
		p.OperatingCurrency = "CNY"
	}
	if p.OpeningBalances == "" {
		p.OpeningBalances = "Equity:OpeningBalances"
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Ledger{}, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO ledgers (id, team_id, name, data_path, operating_currency, start_date, opening_balances, is_bak, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.TeamID, p.Name, p.DataPath, p.OperatingCurrency, p.StartDate, p.OpeningBalances, p.IsBak, now); err != nil {
		return Ledger{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO ledger_members (ledger_id, user_id, role) VALUES (?, ?, 'owner')`,
		p.ID, ownerUserID); err != nil {
		return Ledger{}, err
	}
	if err := tx.Commit(); err != nil {
		return Ledger{}, err
	}
	return Ledger{
		ID:                p.ID,
		TeamID:            p.TeamID,
		Name:              p.Name,
		DataPath:          p.DataPath,
		OperatingCurrency: p.OperatingCurrency,
		StartDate:         p.StartDate,
		OpeningBalances:   p.OpeningBalances,
		IsBak:             p.IsBak,
		Revision:          0,
		Role:              "owner",
		MemberCount:       1,
		CreatedAt:         now,
	}, nil
}

func (s *Store) ListLedgersForUser(ctx context.Context, userID string) ([]Ledger, error) {
	rows, err := s.db.QueryContext(ctx, ledgerSelect+` WHERE lm.user_id = ? ORDER BY l.created_at`, userID)
	if err != nil {
		return nil, fmt.Errorf("list ledgers: %w", err)
	}
	defer rows.Close()
	ledgers, err := scanLedgers(rows)
	if err != nil {
		return nil, err
	}
	return ledgers, nil
}

func (s *Store) GetLedgerForUser(ctx context.Context, ledgerID, userID string) (*Ledger, error) {
	row := s.db.QueryRowContext(ctx, ledgerSelect+` WHERE l.id = ? AND lm.user_id = ?`, ledgerID, userID)
	ledger, err := scanLedger(row)
	if err != nil {
		return nil, err
	}
	return ledger, nil
}

func (s *Store) GetLedger(ctx context.Context, ledgerID string) (*Ledger, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT l.id, l.team_id, l.name, l.data_path, l.operating_currency,
			COALESCE(l.start_date, ''), l.opening_balances, l.is_bak, l.revision, l.created_at,
			'', 0
		FROM ledgers l WHERE l.id = ?`, ledgerID)
	ledger, err := scanLedger(row)
	if err != nil {
		return nil, err
	}
	return ledger, nil
}

// CompareAndBumpRevision 乐观并发：仅当当前修订号等于 expected 时 +1 并返回新值。
func (s *Store) CompareAndBumpRevision(ctx context.Context, ledgerID string, expected int64) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE ledgers SET revision = revision + 1 WHERE id = ? AND revision = ?`, ledgerID, expected)
	if err != nil {
		return 0, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if affected == 0 {
		return 0, ErrRevisionConflict
	}
	var rev int64
	if err := s.db.QueryRowContext(ctx, `SELECT revision FROM ledgers WHERE id = ?`, ledgerID).Scan(&rev); err != nil {
		return 0, err
	}
	return rev, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanLedgers(rows *sql.Rows) ([]Ledger, error) {
	ledgers := make([]Ledger, 0)
	for rows.Next() {
		var l Ledger
		if err := rows.Scan(&l.ID, &l.TeamID, &l.Name, &l.DataPath, &l.OperatingCurrency,
			&l.StartDate, &l.OpeningBalances, &l.IsBak, &l.Revision, &l.CreatedAt,
			&l.Role, &l.MemberCount); err != nil {
			return nil, err
		}
		ledgers = append(ledgers, l)
	}
	return ledgers, rows.Err()
}

func scanLedger(row rowScanner) (*Ledger, error) {
	var l Ledger
	if err := row.Scan(&l.ID, &l.TeamID, &l.Name, &l.DataPath, &l.OperatingCurrency,
		&l.StartDate, &l.OpeningBalances, &l.IsBak, &l.Revision, &l.CreatedAt,
		&l.Role, &l.MemberCount); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &l, nil
}
