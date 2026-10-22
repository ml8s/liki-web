package auth

import (
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestDecideLoginCode(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	record := LoginCodeRecord{ID: "code-1", CodeHash: hash, Attempts: 2, MaxTry: 5, ExpiresAt: now.Add(time.Minute)}

	decision := decideLoginCode(now, "123456", record)
	if decision.Err != nil || !decision.Consume || decision.Attempts != 2 {
		t.Fatalf("success decision = %+v", decision)
	}

	decision = decideLoginCode(now, "654321", record)
	if !errors.Is(decision.Err, ErrCodeInvalid) || decision.Consume || decision.Attempts != 3 {
		t.Fatalf("invalid decision = %+v", decision)
	}

	expired := record
	expired.ExpiresAt = now.Add(-time.Second)
	decision = decideLoginCode(now, "123456", expired)
	if !errors.Is(decision.Err, ErrCodeExpired) || !decision.Consume {
		t.Fatalf("expired decision = %+v", decision)
	}

	locked := record
	locked.Attempts = locked.MaxTry
	decision = decideLoginCode(now, "123456", locked)
	if !errors.Is(decision.Err, ErrCodeLocked) || !decision.Consume {
		t.Fatalf("locked decision = %+v", decision)
	}
}
