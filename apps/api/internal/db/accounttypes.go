package db

import (
	"context"
	"fmt"
)

type AccountType struct {
	Prefix string
	Name   string
}

func (s *Store) ListAccountTypes(ctx context.Context, ledgerID string) ([]AccountType, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT prefix, name FROM account_types WHERE ledger_id = ? ORDER BY prefix`, ledgerID)
	if err != nil {
		return nil, fmt.Errorf("list account types: %w", err)
	}
	defer rows.Close()
	result := make([]AccountType, 0)
	for rows.Next() {
		var t AccountType
		if err := rows.Scan(&t.Prefix, &t.Name); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

func (s *Store) UpsertAccountType(ctx context.Context, ledgerID, prefix, name string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO account_types (ledger_id, prefix, name) VALUES (?, ?, ?)
		ON CONFLICT(ledger_id, prefix) DO UPDATE SET name = excluded.name`,
		ledgerID, prefix, name)
	return err
}
