package db

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestStoreLifecycle(t *testing.T) {
	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	user, err := store.UpsertGitHubUser(ctx, "12345", "octocat", "octo@example.com", "Octo")
	if err != nil {
		t.Fatalf("upsert user: %v", err)
	}
	if user.ID == "" || user.GitHubLogin != "octocat" {
		t.Fatalf("unexpected user: %+v", user)
	}
	// 再次 upsert 应更新而非新建
	updated, err := store.UpsertGitHubUser(ctx, "12345", "octocat2", "", "OctoCat")
	if err != nil {
		t.Fatalf("upsert user again: %v", err)
	}
	if updated.ID != user.ID || updated.GitHubLogin != "octocat2" {
		t.Fatalf("upsert should update same user: %+v", updated)
	}

	team, err := store.CreateTeamWithOwner(ctx, uuid.NewString(), "家庭", user.ID)
	if err != nil {
		t.Fatalf("create team: %v", err)
	}
	teams, err := store.ListTeamsForUser(ctx, user.ID)
	if err != nil || len(teams) != 1 || teams[0].ID != team.ID {
		t.Fatalf("list teams: %v %+v", err, teams)
	}

	ledger, err := store.CreateLedgerWithOwner(ctx, NewLedgerParams{
		ID:                uuid.NewString(),
		TeamID:            team.ID,
		Name:              "家庭账本",
		DataPath:          filepath.Join(t.TempDir(), "ledger"),
		OperatingCurrency: "CNY",
		IsBak:             true,
	}, user.ID)
	if err != nil {
		t.Fatalf("create ledger: %v", err)
	}
	got, err := store.GetLedgerForUser(ctx, ledger.ID, user.ID)
	if err != nil || got.Role != "owner" || got.Revision != 0 {
		t.Fatalf("get ledger: %v %+v", err, got)
	}

	// 乐观并发 CAS
	rev, err := store.CompareAndBumpRevision(ctx, ledger.ID, 0)
	if err != nil || rev != 1 {
		t.Fatalf("bump revision: %v %d", err, rev)
	}
	if _, err := store.CompareAndBumpRevision(ctx, ledger.ID, 0); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale revision should conflict, got %v", err)
	}

	// 会话
	tokenHash := "tok-hash-1"
	if err := store.CreateSession(ctx, uuid.NewString(), user.ID, tokenHash, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("create session: %v", err)
	}
	sessUser, err := store.UserBySession(ctx, tokenHash)
	if err != nil || sessUser.ID != user.ID {
		t.Fatalf("session user: %v %+v", err, sessUser)
	}
	if err := store.DeleteSession(ctx, tokenHash); err != nil {
		t.Fatalf("delete session: %v", err)
	}
	if _, err := store.UserBySession(ctx, tokenHash); err == nil {
		t.Fatal("expired session should fail")
	}

	// 审计
	if err := store.InsertAuditLog(ctx, AuditParams{LedgerID: ledger.ID, UserID: user.ID, Actor: "octocat2", Action: "create_ledger", Object: ledger.ID}); err != nil {
		t.Fatalf("insert audit: %v", err)
	}
	logs, err := store.ListAuditLogs(ctx, ledger.ID, 10)
	if err != nil || len(logs) != 1 || logs[0].Action != "create_ledger" {
		t.Fatalf("list audit: %v %+v", err, logs)
	}
}
