package auth

import (
	"context"
	"testing"
)

func TestWithAuthRoundTrip(t *testing.T) {
	user := User{ID: "user-1", Email: "user@example.com", Status: UserActive}
	session := AuthSession{ID: "session-1"}
	const token = "session-token-value"

	ctx := WithAuth(context.Background(), user, session, token)

	gotUser, ok := ContextUser(ctx)
	if !ok || gotUser.ID != user.ID || gotUser.Email != user.Email {
		t.Fatalf("ContextUser() = %+v, %v", gotUser, ok)
	}
	gotSession, ok := ContextSession(ctx)
	if !ok || gotSession.ID != session.ID {
		t.Fatalf("ContextSession() = %+v, %v", gotSession, ok)
	}
	gotToken, ok := ContextSessionToken(ctx)
	if !ok || gotToken != token {
		t.Fatalf("ContextSessionToken() = %q, %v", gotToken, ok)
	}

	if u, ok := UserFromContext(ctx); !ok || u.ID != user.ID {
		t.Fatalf("UserFromContext() = %+v, %v", u, ok)
	}
	if s, ok := SessionTokenFromContext(ctx); !ok || s != token {
		t.Fatalf("SessionTokenFromContext() = %q, %v", s, ok)
	}
}

func TestAuthContextMissingValues(t *testing.T) {
	if _, ok := ContextUser(context.Background()); ok {
		t.Error("ContextUser() reported presence on empty context")
	}
	if _, ok := ContextSession(context.Background()); ok {
		t.Error("ContextSession() reported presence on empty context")
	}
	if _, ok := ContextSessionToken(context.Background()); ok {
		t.Error("ContextSessionToken() reported presence on empty context")
	}
}

func TestAuthContextRejectsMismatchedTypes(t *testing.T) {
	wrong := context.WithValue(context.Background(), userContextKey, "not-a-user")
	if _, ok := ContextUser(wrong); ok {
		t.Error("ContextUser() accepted a non-User value")
	}
	wrongToken := context.WithValue(context.Background(), tokenContextKey, 42)
	if _, ok := ContextSessionToken(wrongToken); ok {
		t.Error("ContextSessionToken() accepted a non-string value")
	}
}
