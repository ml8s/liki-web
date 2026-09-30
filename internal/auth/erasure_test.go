package auth

import (
	"context"
	"errors"
	"testing"
)

func TestEraseUserDestroysIdentityAndKeepsPseudonymousRecord(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	service, mailer := newTestService(t, store)
	email := "erase@example.com"
	if _, err := service.RequestEmailCode(ctx, email, "192.0.2.10", "erase-agent", "zh-Hans"); err != nil {
		t.Fatal(err)
	}
	result := login(t, service, email, mailer.codes[0])

	res, err := store.EraseUser(ctx, result.User.ID, "erased-test@liki.invalid")
	if err != nil {
		t.Fatal(err)
	}
	if res.AlreadyErased || res.Identities != 1 || res.Sessions != 1 || res.Codes != 1 {
		t.Fatalf("erasure=%+v", res)
	}

	if _, _, err := store.GetSessionUser(ctx, result.SessionToken); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("session lookup err=%v, want ErrSessionNotFound", err)
	}
	if _, err := store.GetUser(ctx, result.User.ID); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("get user err=%v, want ErrUserNotFound", err)
	}
	if identities, err := store.ListIdentities(ctx, result.User.ID); err != nil || len(identities) != 0 {
		t.Fatalf("identities=%+v err=%v", identities, err)
	}

	var storedEmail, status string
	var deletedAt *string
	err = store.db.QueryRowContext(ctx, `SELECT email,status,deleted_at FROM users WHERE id=?`, result.User.ID).
		Scan(&storedEmail, &status, &deletedAt)
	if err != nil {
		t.Fatal(err)
	}
	if status != string(UserDeleted) || deletedAt == nil || storedEmail != "erased-test@liki.invalid" {
		t.Fatalf("tombstone email=%q status=%q deleted=%v", storedEmail, status, deletedAt)
	}

	var audits int
	var pii int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM auth_audit_logs WHERE user_id=?`, result.User.ID).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(ctx, `
SELECT COUNT(1) FROM auth_audit_logs
WHERE user_id=? AND (email=? OR ip_hash IS NOT NULL OR user_agent IS NOT NULL)`,
		result.User.ID, email).Scan(&pii); err != nil {
		t.Fatal(err)
	}
	if audits < 2 || pii != 0 {
		t.Fatalf("audits=%d pii=%d", audits, pii)
	}

	// Idempotency is important for retry-safe orchestration.
	repeat, err := store.EraseUser(ctx, result.User.ID, "erased-test@liki.invalid")
	if err != nil || !repeat.AlreadyErased {
		t.Fatalf("repeat=%+v err=%v", repeat, err)
	}
}

func TestEraseUserMissing(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.EraseUser(context.Background(), "missing", ""); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("err=%v", err)
	}
}
