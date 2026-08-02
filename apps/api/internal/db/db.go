package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var ErrRevisionConflict = errors.New("revision conflict")

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create db dir: %w", err)
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// SQLite 单写者：限制连接数，避免写锁竞争
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys=ON`); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		github_id TEXT UNIQUE NOT NULL,
		github_login TEXT NOT NULL,
		email TEXT,
		display_name TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS teams (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		owner_user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		created_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS team_members (
		team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		role TEXT NOT NULL CHECK (role IN ('owner','editor','viewer')),
		status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','pending')),
		PRIMARY KEY (team_id, user_id)
	);
	CREATE TABLE IF NOT EXISTS ledgers (
		id TEXT PRIMARY KEY,
		team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		data_path TEXT NOT NULL UNIQUE,
		operating_currency TEXT NOT NULL DEFAULT 'CNY',
		start_date TEXT,
		opening_balances TEXT NOT NULL DEFAULT 'Equity:OpeningBalances',
		is_bak INTEGER NOT NULL DEFAULT 1,
		revision INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS ledger_members (
		ledger_id TEXT NOT NULL REFERENCES ledgers(id) ON DELETE CASCADE,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		role TEXT NOT NULL CHECK (role IN ('owner','editor','viewer')),
		PRIMARY KEY (ledger_id, user_id)
	);
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		token_hash TEXT NOT NULL UNIQUE,
		expires_at TEXT NOT NULL,
		created_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS api_keys (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		secret_hash TEXT NOT NULL,
		scope TEXT NOT NULL CHECK (scope IN ('read-only','read-write','ai')),
		revoked INTEGER NOT NULL DEFAULT 0,
		last_used_at TEXT,
		created_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ledger_id TEXT,
		user_id TEXT,
		actor TEXT,
		action TEXT NOT NULL,
		object TEXT,
		detail TEXT,
		created_at TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_sessions_token_hash ON sessions(token_hash);
	CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_team_members_user ON team_members(user_id);
	CREATE INDEX IF NOT EXISTS idx_ledger_members_user ON ledger_members(user_id);
	CREATE INDEX IF NOT EXISTS idx_ledgers_team ON ledgers(team_id);
	CREATE INDEX IF NOT EXISTS idx_audit_ledger ON audit_logs(ledger_id);`,
}

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}
	var current int
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&current); err != nil {
		return fmt.Errorf("read migration version: %w", err)
	}
	for i, migration := range migrations {
		version := i + 1
		if version <= current {
			continue
		}
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, migration); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %d: %w", version, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES (?)`, version); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %d: %w", version, err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
