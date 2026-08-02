package db

import (
	"context"
	"fmt"
	"time"
)

func (s *Store) CreateTeamWithOwner(ctx context.Context, teamID, name, ownerUserID string) (Team, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Team{}, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO teams (id, name, owner_user_id, created_at) VALUES (?, ?, ?, ?)`,
		teamID, name, ownerUserID, now); err != nil {
		return Team{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO team_members (team_id, user_id, role, status) VALUES (?, ?, 'owner', 'active')`,
		teamID, ownerUserID); err != nil {
		return Team{}, err
	}
	if err := tx.Commit(); err != nil {
		return Team{}, err
	}
	return Team{
		ID:          teamID,
		Name:        name,
		OwnerUserID: ownerUserID,
		Role:        "owner",
		MemberCount: 1,
		LedgerCount: 0,
		CreatedAt:   now,
	}, nil
}

func (s *Store) ListTeamsForUser(ctx context.Context, userID string) ([]Team, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.id, t.name, t.owner_user_id, tm.role, t.created_at,
			(SELECT COUNT(*) FROM team_members tm2 WHERE tm2.team_id = t.id) AS member_count,
			(SELECT COUNT(*) FROM ledgers l WHERE l.team_id = t.id) AS ledger_count
		FROM teams t
		JOIN team_members tm ON tm.team_id = t.id
		WHERE tm.user_id = ?
		ORDER BY t.created_at`, userID)
	if err != nil {
		return nil, fmt.Errorf("list teams: %w", err)
	}
	defer rows.Close()

	teams := make([]Team, 0)
	for rows.Next() {
		var t Team
		if err := rows.Scan(&t.ID, &t.Name, &t.OwnerUserID, &t.Role, &t.CreatedAt, &t.MemberCount, &t.LedgerCount); err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, rows.Err()
}

func (s *Store) TeamRole(ctx context.Context, teamID, userID string) (string, error) {
	var role string
	err := s.db.QueryRowContext(ctx, `
		SELECT role FROM team_members WHERE team_id = ? AND user_id = ?`, teamID, userID).Scan(&role)
	return role, err
}
