package mcp

import (
	"context"

	"github.com/beancount-gs/api/internal/db"
)

type ctxKey int

const (
	ctxUser ctxKey = iota
	ctxScope
)

// WithAuthUser 将 MCP 认证信息注入上下文。
func WithAuthUser(ctx context.Context, user db.User, scope string) context.Context {
	ctx = context.WithValue(ctx, ctxUser, user)
	return context.WithValue(ctx, ctxScope, scope)
}

// AuthFrom 从上下文读取 MCP 认证信息（由 RequireApiKey 中间件注入）。
func AuthFrom(ctx context.Context) (db.User, string, bool) {
	user, ok := ctx.Value(ctxUser).(db.User)
	if !ok {
		return db.User{}, "", false
	}
	scope, _ := ctx.Value(ctxScope).(string)
	return user, scope, true
}

func userFrom(ctx context.Context) db.User {
	user, _, _ := AuthFrom(ctx)
	return user
}

func ctxWithUser(ctx context.Context, user db.User) context.Context {
	scope, _ := ctx.Value(ctxScope).(string)
	return WithAuthUser(ctx, user, scope)
}
