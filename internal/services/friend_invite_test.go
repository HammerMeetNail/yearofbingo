package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestFriendInviteService_CreateInvite_Success(t *testing.T) {
	inviterID := uuid.New()
	inviteID := uuid.New()
	now := time.Now()

	var gotArgs []any
	db := &fakeDB{
		QueryRowFunc: func(ctx context.Context, sql string, args ...any) Row {
			if !strings.Contains(sql, "INSERT INTO friend_invites") {
				t.Fatalf("unexpected sql: %q", sql)
			}
			gotArgs = args
			return rowFromValues(inviteID, inviterID, &now, nil, nil, nil, now)
		},
	}

	svc := NewFriendInviteService(db)
	invite, token, err := svc.CreateInvite(context.Background(), inviterID, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected invite token")
	}
	if invite.ID != inviteID || invite.InviterUserID != inviterID {
		t.Fatalf("unexpected invite: %+v", invite)
	}
	if len(gotArgs) != 3 {
		t.Fatalf("expected 3 args, got %d", len(gotArgs))
	}
	if gotArgs[0] != inviterID {
		t.Fatalf("expected inviterID arg, got %v", gotArgs[0])
	}
	if gotArgs[1] == "" {
		t.Fatal("expected token hash arg")
	}
	if gotArgs[2] == nil {
		t.Fatal("expected expires_at arg")
	}
}

func TestFriendInviteService_ListInvites_ReturnsRows(t *testing.T) {
	inviterID := uuid.New()
	inviteID := uuid.New()
	now := time.Now()
	db := &fakeDB{
		QueryFunc: func(ctx context.Context, sql string, args ...any) (Rows, error) {
			return &fakeRows{rows: [][]any{
				{inviteID, inviterID, &now, nil, nil, nil, now},
			}}, nil
		},
	}

	svc := NewFriendInviteService(db)
	invites, err := svc.ListInvites(context.Background(), inviterID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(invites) != 1 {
		t.Fatalf("expected 1 invite, got %d", len(invites))
	}
	if invites[0].ID != inviteID {
		t.Fatalf("unexpected invite: %+v", invites[0])
	}
}

func TestFriendInviteService_RevokeInvite_NotFound(t *testing.T) {
	db := &fakeDB{
		ExecFunc: func(ctx context.Context, sql string, args ...any) (CommandTag, error) {
			return fakeCommandTag{rowsAffected: 0}, nil
		},
	}
	svc := NewFriendInviteService(db)
	if err := svc.RevokeInvite(context.Background(), uuid.New(), uuid.New()); !errors.Is(err, ErrInviteNotFound) {
		t.Fatalf("expected ErrInviteNotFound, got %v", err)
	}
}

func TestFriendInviteService_AcceptInvite_NotFound(t *testing.T) {
	tx := &fakeTx{
		QueryRowFunc: func(ctx context.Context, sql string, args ...any) Row {
			return fakeRow{scanFunc: func(dest ...any) error {
				return pgx.ErrNoRows
			}}
		},
	}
	db := &fakeDB{
		BeginFunc: func(ctx context.Context) (Tx, error) {
			return tx, nil
		},
	}
	svc := NewFriendInviteService(db)
	_, err := svc.AcceptInvite(context.Background(), uuid.New(), "token")
	if !errors.Is(err, ErrInviteNotFound) {
		t.Fatalf("expected ErrInviteNotFound, got %v", err)
	}
}

func TestFriendInviteService_AcceptInvite_Self(t *testing.T) {
	userID := uuid.New()
	tx := &fakeTx{
		QueryRowFunc: func(ctx context.Context, sql string, args ...any) Row {
			return rowFromValues(uuid.New(), userID, "self")
		},
	}
	db := &fakeDB{
		BeginFunc: func(ctx context.Context) (Tx, error) {
			return tx, nil
		},
	}
	svc := NewFriendInviteService(db)
	_, err := svc.AcceptInvite(context.Background(), userID, "token")
	if !errors.Is(err, ErrCannotFriendSelf) {
		t.Fatalf("expected ErrCannotFriendSelf, got %v", err)
	}
}

func TestFriendInviteService_AcceptInvite_Blocked(t *testing.T) {
	userID := uuid.New()
	inviterID := uuid.New()
	calls := 0
	tx := &fakeTx{
		QueryRowFunc: func(ctx context.Context, sql string, args ...any) Row {
			calls++
			switch calls {
			case 1:
				return rowFromValues(uuid.New(), inviterID, "inviter")
			case 2:
				return rowFromValues(true)
			default:
				t.Fatalf("unexpected query call %d", calls)
				return rowFromValues(false)
			}
		},
	}
	db := &fakeDB{
		BeginFunc: func(ctx context.Context) (Tx, error) {
			return tx, nil
		},
	}
	svc := NewFriendInviteService(db)
	_, err := svc.AcceptInvite(context.Background(), userID, "token")
	if !errors.Is(err, ErrUserBlocked) {
		t.Fatalf("expected ErrUserBlocked, got %v", err)
	}
}

func TestFriendInviteService_AcceptInvite_Success(t *testing.T) {
	userID := uuid.New()
	inviterID := uuid.New()
	inviteID := uuid.New()
	calls := 0
	var execCalls int
	tx := &fakeTx{
		QueryRowFunc: func(ctx context.Context, sql string, args ...any) Row {
			calls++
			switch calls {
			case 1:
				return rowFromValues(inviteID, inviterID, "inviter")
			case 2:
				return rowFromValues(false)
			case 3:
				return rowFromValues(false)
			default:
				t.Fatalf("unexpected query call %d", calls)
				return rowFromValues(false)
			}
		},
		ExecFunc: func(ctx context.Context, sql string, args ...any) (CommandTag, error) {
			execCalls++
			return fakeCommandTag{rowsAffected: 1}, nil
		},
		CommitFunc: func(ctx context.Context) error {
			return nil
		},
	}
	db := &fakeDB{
		BeginFunc: func(ctx context.Context) (Tx, error) {
			return tx, nil
		},
	}
	svc := NewFriendInviteService(db)
	inviter, err := svc.AcceptInvite(context.Background(), userID, "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inviter.ID != inviterID || inviter.Username != "inviter" {
		t.Fatalf("unexpected inviter: %+v", inviter)
	}
	if execCalls != 2 {
		t.Fatalf("expected 2 exec calls, got %d", execCalls)
	}
}
