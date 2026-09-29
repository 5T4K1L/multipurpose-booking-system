package service

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
	"strings"
	"testing"
	"time"
)

type sessionFake struct {
	user           db.User
	row            db.UserSession
	err, commitErr error
	touched        bool
	hash           string
}

func (f *sessionFake) WithinSession(_ context.Context, fn func(SessionStore) error) error {
	if err := fn(f); err != nil {
		return err
	}
	return f.commitErr
}
func (f *sessionFake) GetUserByID(context.Context, int64) (db.User, error) { return f.user, f.err }
func (f *sessionFake) CreateUserSession(_ context.Context, id int64, hash string, expires pgtype.Timestamp) (db.UserSession, error) {
	f.hash = hash
	f.row = db.UserSession{ID: 1, UserID: id, TokenHash: hash, ExpiresAt: expires}
	return f.row, f.err
}
func (f *sessionFake) GetUserSessionByTokenHash(context.Context, string) (db.UserSession, error) {
	return f.row, f.err
}
func (f *sessionFake) UpdateUserSessionLastSeen(context.Context, int64) error {
	f.touched = true
	return f.err
}
func (f *sessionFake) RevokeUserSession(context.Context, int64) error     { return f.err }
func (f *sessionFake) RevokeAllUserSessions(context.Context, int64) error { return f.err }

func TestSessionLifecyclePolicy(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	f := &sessionFake{user: db.User{ID: 1, Status: "active"}}
	s, err := NewSessionService(f, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	s.now = func() time.Time { return now }
	issued, err := s.Create(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := sessionHash(issued.Token)
	if err != nil || len(hash) != 64 || hash != f.hash || hash == issued.Token {
		t.Fatal("token storage is not hashed")
	}
	if !issued.ExpiresAt.Equal(now.Add(time.Hour)) {
		t.Fatal("wrong expiration")
	}
	serialized, err := json.Marshal(issued)
	if err != nil || strings.Contains(string(serialized), issued.Token) {
		t.Fatal("token exposed by JSON")
	}
	if user, err := s.Validate(ctx, issued.Token); err != nil || user.ID != 1 || !f.touched {
		t.Fatalf("validation: %v", err)
	}
	if !f.row.ExpiresAt.Time.Equal(issued.ExpiresAt) {
		t.Fatal("validation extended expiry")
	}
	for _, tc := range []struct {
		name   string
		change func()
		want   error
	}{
		{"expired exactly", func() { s.now = func() time.Time { return issued.ExpiresAt } }, ErrInvalidSession},
		{"disabled", func() { s.now = func() time.Time { return now }; f.user.Status = "disabled" }, ErrInvalidSession},
		{"unknown status", func() { f.user.Status = "unknown" }, ErrInvalidSession},
		{"revoked", func() { f.user.Status = "active"; f.row.RevokedAt = pgtype.Timestamp{Valid: true} }, ErrInvalidSession},
		{"missing", func() { f.row.RevokedAt.Valid = false; f.err = ErrNotFound }, ErrInvalidSession},
		{"database failure", func() { f.err = errors.New("secret detail") }, ErrSessionUnavailable},
		{"commit failure", func() { f.err = nil; f.commitErr = errors.New("commit detail") }, ErrSessionUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.change()
			f.touched = false
			user, err := s.Validate(ctx, issued.Token)
			if !errors.Is(err, tc.want) || user.ID != 0 {
				t.Fatalf("got user %v, error %v", user.ID, err)
			}
		})
	}
}

func TestSessionCreationFailures(t *testing.T) {
	for _, tc := range []struct {
		name, status         string
		err, commitErr, want error
	}{
		{name: "disabled", status: "disabled", want: ErrAccountDisabled},
		{name: "missing", err: ErrNotFound, want: ErrInvalidSession},
		{name: "storage", err: errors.New("private"), want: ErrSessionUnavailable},
		{name: "commit", status: "active", commitErr: errors.New("private"), want: ErrSessionUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &sessionFake{user: db.User{ID: 1, Status: tc.status}, err: tc.err, commitErr: tc.commitErr}
			s, _ := NewSessionService(f, time.Hour)
			issued, err := s.Create(context.Background(), 1)
			if !errors.Is(err, tc.want) || issued.Token != "" {
				t.Fatalf("error = %v, token returned = %v", err, issued.Token != "")
			}
		})
	}
}

func TestSessionTokensAndRevocation(t *testing.T) {
	f := &sessionFake{user: db.User{ID: 1, Status: "active"}}
	s, _ := NewSessionService(f, time.Hour)
	first, _ := s.Create(context.Background(), 1)
	second, _ := s.Create(context.Background(), 1)
	if first.Token == second.Token {
		t.Fatal("session token reused")
	}
	for _, token := range []string{"", "short", strings.Repeat("!", 43), strings.Repeat("a", 1000)} {
		if _, err := s.Validate(context.Background(), token); !errors.Is(err, ErrInvalidSession) {
			t.Fatal("invalid token accepted")
		}
		if err := s.Revoke(context.Background(), token); !errors.Is(err, ErrInvalidSession) {
			t.Fatal("invalid revocation token accepted")
		}
	}
	f.err = ErrNotFound
	if err := s.Revoke(context.Background(), first.Token); err != nil {
		t.Fatal(err)
	}
	if err := s.RevokeAll(context.Background(), 0); !errors.Is(err, ErrInvalidSession) {
		t.Fatal(err)
	}
	if _, err := NewSessionService(nil, time.Hour); err == nil {
		t.Fatal("nil store accepted")
	}
	if _, err := NewSessionService(f, 0); err == nil {
		t.Fatal("invalid lifetime accepted")
	}
}
