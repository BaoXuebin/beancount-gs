package db

import (
	"context"
	"fmt"
	"time"
)

type AuditParams struct {
	LedgerID string
	UserID   string
	Actor    string
	Action   string
	Object   string
	Detail   string
}

func (s *Store) InsertAuditLog(ctx context.Context, p AuditParams) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO audit_logs (ledger_id, user_id, actor, action, object, detail, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		p.LedgerID, p.UserID, p.Actor, p.Action, p.Object, p.Detail, time.Now().UTC().Format(time.RFC3339))
	return err
}

func (s *Store) ListAuditLogs(ctx context.Context, ledgerID string, limit int) ([]AuditLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, COALESCE(ledger_id, ''), COALESCE(user_id, ''), COALESCE(actor, ''),
			action, COALESCE(object, ''), COALESCE(detail, ''), created_at
		FROM audit_logs
		WHERE (? = '' OR ledger_id = ?)
		ORDER BY created_at DESC
		LIMIT ?`, ledgerID, ledgerID, limit)
	if err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	logs := make([]AuditLog, 0)
	for rows.Next() {
		var l AuditLog
		if err := rows.Scan(&l.ID, &l.LedgerID, &l.UserID, &l.Actor, &l.Action, &l.Object, &l.Detail, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}
