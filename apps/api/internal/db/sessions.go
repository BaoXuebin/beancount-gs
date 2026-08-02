package db

import (
	"context"
	"fmt"
	"time"
)

func (s *Store) CreateSession(ctx context.Context, sessionID, userID, tokenHash string, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, token_hash, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		sessionID, userID, tokenHash, expiresAt.UTC().Format(time.RFC3339), time.Now().UTC().Format(time.RFC3339))
	return err
}

func (s *Store) UserBySession(ctx context.Context, tokenHash string) (*User, error) {
	var u User
	err := s.db.QueryRowContext(ctx, `
		SELECT u.id, u.github_id, u.github_login, COALESCE(u.email, ''), u.display_name, u.created_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = ? AND s.expires_at > ?`,
		tokenHash, time.Now().UTC().Format(time.RFC3339)).
		Scan(&u.ID, &u.GitHubID, &u.GitHubLogin, &u.Email, &u.DisplayName, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("session lookup: %w", err)
	}
	return &u, nil
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, tokenHash)
	return err
}
