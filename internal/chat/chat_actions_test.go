package chat

import (
	"context"
	"errors"
	"testing"

	"github.com/ml8s/liki-web/internal/product"
)

func TestRenameSessionUpdatesTitleWithoutTouchingActivity(t *testing.T) {
	service, chats, _ := newAgentModeTestService(t)
	ctx := context.Background()
	session, err := service.CreateSession(ctx, "user-1", string(product.SkillConsultation), "原名", "")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	renamed, err := service.RenameSession(ctx, "user-1", session.ID, "新名字")
	if err != nil {
		t.Fatalf("rename session: %v", err)
	}
	if renamed.Title != "新名字" {
		t.Fatalf("title = %q", renamed.Title)
	}
	if !renamed.UpdatedAt.Equal(session.UpdatedAt) {
		t.Fatalf("rename bumped updated_at: %v -> %v", session.UpdatedAt, renamed.UpdatedAt)
	}

	if _, err := chats.RenameSession(ctx, "user-2", session.ID, "他人的名字"); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("foreign rename err = %v, want ErrSessionNotFound", err)
	}
	if _, err := service.RenameSession(ctx, "user-1", "missing", "x"); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("missing rename err = %v, want ErrSessionNotFound", err)
	}
}

func TestSoftDeleteSessionHidesFromListAndLookup(t *testing.T) {
	service, chats, _ := newAgentModeTestService(t)
	ctx := context.Background()
	kept, err := service.CreateSession(ctx, "user-1", string(product.SkillConsultation), "保留", "")
	if err != nil {
		t.Fatalf("create kept: %v", err)
	}
	removed, err := service.CreateSession(ctx, "user-1", string(product.SkillConsultation), "删除", "")
	if err != nil {
		t.Fatalf("create removed: %v", err)
	}

	if err := service.DeleteSession(ctx, "user-1", removed.ID); err != nil {
		t.Fatalf("delete session: %v", err)
	}
	sessions, err := service.store.ListSessions(ctx, "user-1")
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].ID != kept.ID {
		t.Fatalf("sessions after delete = %+v", sessions)
	}
	if _, err := service.store.GetSession(ctx, "user-1", removed.ID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("get deleted err = %v, want ErrSessionNotFound", err)
	}
	if _, err := chats.RenameSession(ctx, "user-1", removed.ID, "x"); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("rename deleted err = %v, want ErrSessionNotFound", err)
	}
	if err := service.DeleteSession(ctx, "user-1", removed.ID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("double delete err = %v, want ErrSessionNotFound", err)
	}
	if err := service.DeleteSession(ctx, "user-2", kept.ID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("foreign delete err = %v, want ErrSessionNotFound", err)
	}
}
