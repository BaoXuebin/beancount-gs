package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type ApiKey struct {
	ID         string
	UserID     string
	Name       string
	SecretHash string
	Prefix     string
	Scope      string
	Revoked    bool
	LastUsedAt string
	CreatedAt  string
}

func (s *Store) CreateApiKey(ctx context.Context, key ApiKey) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO api_keys (id, user_id, name, secret_hash, prefix, scope, revoked, created_at)
		VALUES (?, ?, ?, ?, ?, ?, 0, ?)`,
		key.ID, key.UserID, key.Name, key.SecretHash, key.Prefix, key.Scope,
		time.Now().UTC().Format(time.RFC3339))
	return err
}

func (s *Store) ListApiKeys(ctx context.Context, userID string) ([]ApiKey, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, name, secret_hash, prefix, scope, revoked,
			COALESCE(last_used_at, ''), created_at
		FROM api_keys WHERE user_id = ? ORDER BY created_at`, userID)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()
	keys := make([]ApiKey, 0)
	for rows.Next() {
		var k ApiKey
		if err := rows.Scan(&k.ID, &k.UserID, &k.Name, &k.SecretHash, &k.Prefix, &k.Scope,
			&k.Revoked, &k.LastUsedAt, &k.CreatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (s *Store) GetApiKeyByHash(ctx context.Context, secretHash string) (*ApiKey, error) {
	var k ApiKey
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, name, secret_hash, prefix, scope, revoked,
			COALESCE(last_used_at, ''), created_at
		FROM api_keys WHERE secret_hash = ? AND revoked = 0`, secretHash).
		Scan(&k.ID, &k.UserID, &k.Name, &k.SecretHash, &k.Prefix, &k.Scope,
			&k.Revoked, &k.LastUsedAt, &k.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func (s *Store) RevokeApiKey(ctx context.Context, id, userID string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE api_keys SET revoked = 1 WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) TouchApiKey(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE api_keys SET last_used_at = ? WHERE id = ?`,
		time.Now().UTC().Format(time.RFC3339), id)
	return err
}
