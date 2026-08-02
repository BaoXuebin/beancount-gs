package db

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func TestApiKeys(t *testing.T) {
	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	user, err := store.UpsertGitHubUser(ctx, "1", "alice", "", "Alice")
	if err != nil {
		t.Fatal(err)
	}

	key := ApiKey{
		ID: uuid.NewString(), UserID: user.ID, Name: "claude-code",
		SecretHash: "hash-1", Prefix: "bgsk_abc", Scope: "read-write",
	}
	if err := store.CreateApiKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetApiKeyByHash(ctx, "hash-1")
	if err != nil || got.Name != "claude-code" || got.Revoked {
		t.Fatalf("lookup: %v %+v", err, got)
	}
	keys, err := store.ListApiKeys(ctx, user.ID)
	if err != nil || len(keys) != 1 {
		t.Fatalf("list: %v %d", err, len(keys))
	}
	if err := store.TouchApiKey(ctx, key.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.RevokeApiKey(ctx, key.ID, user.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetApiKeyByHash(ctx, "hash-1"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("revoked key should not resolve, got %v", err)
	}
}
