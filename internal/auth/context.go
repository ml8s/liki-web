package auth

import (
	"context"
)

type contextKey string

const (
	userContextKey    contextKey = "auth.user"
	sessionContextKey contextKey = "auth.session"
	tokenContextKey   contextKey = "auth.token"
)

func ContextUser(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(userContextKey).(User)
	return user, ok
}

func ContextSession(ctx context.Context) (AuthSession, bool) {
	session, ok := ctx.Value(sessionContextKey).(AuthSession)
	return session, ok
}

func ContextSessionToken(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(tokenContextKey).(string)
	return token, ok
}

func WithAuth(ctx context.Context, user User, session AuthSession, token string) context.Context {
	ctx = context.WithValue(ctx, userContextKey, user)
	ctx = context.WithValue(ctx, sessionContextKey, session)
	return context.WithValue(ctx, tokenContextKey, token)
}

func SessionTokenFromContext(ctx context.Context) (string, bool) {
	return ContextSessionToken(ctx)
}

func UserFromContext(ctx context.Context) (User, bool) {
	return ContextUser(ctx)
}
