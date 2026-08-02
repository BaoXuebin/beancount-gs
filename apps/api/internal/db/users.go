package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (s *Store) UpsertGitHubUser(ctx context.Context, githubID, githubLogin, email, displayName string) (User, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback()

	var u User
	err = tx.QueryRowContext(ctx, `
		SELECT id, github_id, github_login, COALESCE(email, ''), display_name, created_at
		FROM users WHERE github_id = ?`, githubID).
		Scan(&u.ID, &u.GitHubID, &u.GitHubLogin, &u.Email, &u.DisplayName, &u.CreatedAt)
	switch {
	case err == nil:
		if _, err := tx.ExecContext(ctx, `
			UPDATE users SET github_login = ?, email = COALESCE(?, email), display_name = COALESCE(?, display_name)
			WHERE id = ?`, githubLogin, email, displayName, u.ID); err != nil {
			return User{}, err
		}
		u.GitHubLogin = githubLogin
		if email != "" {
			u.Email = email
		}
		if displayName != "" {
			u.DisplayName = displayName
		}
	case errors.Is(err, sql.ErrNoRows):
		u = User{
			ID:          uuid.NewString(),
			GitHubID:    githubID,
			GitHubLogin: githubLogin,
			Email:       email,
			DisplayName: displayName,
			CreatedAt:   now,
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO users (id, github_id, github_login, email, display_name, created_at)
			VALUES (?, ?, ?, ?, ?, ?)`,
			u.ID, u.GitHubID, u.GitHubLogin, u.Email, u.DisplayName, u.CreatedAt); err != nil {
			return User{}, err
		}
	default:
		return User{}, err
	}
	if err := tx.Commit(); err != nil {
		return User{}, err
	}
	return u, nil
}

func (s *Store) GetUserByID(ctx context.Context, id string) (*User, error) {
	var u User
	err := s.db.QueryRowContext(ctx, `
		SELECT id, github_id, github_login, COALESCE(email, ''), display_name, created_at
		FROM users WHERE id = ?`, id).
		Scan(&u.ID, &u.GitHubID, &u.GitHubLogin, &u.Email, &u.DisplayName, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &u, nil
}
