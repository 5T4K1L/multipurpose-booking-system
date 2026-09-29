package service

import (
	"context"
	"errors"
	"testing"

	"multipurpose-booking-system/api/internal/auth"
	"multipurpose-booking-system/api/internal/db"
)

type fakeStore struct {
	user                                                                            db.User
	credential                                                                      db.UserCredential
	identity                                                                        db.UserOauthIdentity
	userErr, credentialErr, identityErr, providerErr, createErr, linkErr, commitErr error
	created, linked, committed                                                      bool
}

func (f *fakeStore) WithinAuthentication(_ context.Context, fn func(AuthenticationStore) error) error {
	err := fn(f)
	if err == nil {
		err = f.commitErr
	}
	f.committed = err == nil
	if err != nil {
		f.created = false
		f.linked = false
	}
	return err
}
func (f *fakeStore) GetUserByID(context.Context, int64) (db.User, error) { return f.user, f.userErr }
func (f *fakeStore) GetUserByEmail(context.Context, string) (db.User, error) {
	return f.user, f.userErr
}
func (f *fakeStore) GetUserCredential(context.Context, int64) (db.UserCredential, error) {
	return f.credential, f.credentialErr
}
func (f *fakeStore) GetOAuthIdentity(context.Context, string, string) (db.UserOauthIdentity, error) {
	return f.identity, f.identityErr
}
func (f *fakeStore) GetUserOAuthIdentity(context.Context, int64, string) (db.UserOauthIdentity, error) {
	return f.identity, f.providerErr
}
func (f *fakeStore) CreateUser(_ context.Context, email, name string) (db.User, error) {
	f.created = true
	return db.User{ID: 2, Email: email, Username: name, Status: "active"}, f.createErr
}
func (f *fakeStore) CreateOAuthIdentity(context.Context, int64, string, string) (db.UserOauthIdentity, error) {
	f.linked = true
	return f.identity, f.linkErr
}

func TestPasswordAuthentication(t *testing.T) {
	hash, err := auth.HashPassword("correct")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, password, status       string
		userErr, credentialErr, want error
	}{
		{name: "success", password: "correct", status: "active"},
		{name: "wrong password", password: "wrong", status: "active", want: ErrInvalidCredentials},
		{name: "missing user", password: "correct", userErr: ErrNotFound, want: ErrInvalidCredentials},
		{name: "OAuth only user", password: "correct", credentialErr: ErrNotFound, want: ErrInvalidCredentials},
		{name: "disabled", password: "correct", status: "disabled", want: ErrAccountDisabled},
		{name: "disabled wrong password", password: "wrong", status: "disabled", want: ErrInvalidCredentials},
		{name: "unknown status", password: "correct", status: "unknown", want: ErrAccountDisabled},
		{name: "storage error", password: "correct", userErr: errors.New("secret database detail"), want: ErrAuthenticationUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeStore{user: db.User{ID: 1, Status: tc.status}, credential: db.UserCredential{PasswordHash: hash}, userErr: tc.userErr, credentialErr: tc.credentialErr}
			user, err := NewAuthenticationService(f).AuthenticatePassword(context.Background(), "user@example.com", tc.password)
			if !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
			if err != nil && user.ID != 0 {
				t.Fatal("failed authentication returned user")
			}
		})
	}
}

func TestOAuthResolution(t *testing.T) {
	for _, tc := range []struct {
		name    string
		change  func(*fakeStore, *auth.OAuthIdentity, *bool)
		want    error
		created bool
	}{
		{name: "existing subject ignores changed email", change: func(f *fakeStore, i *auth.OAuthIdentity, _ *bool) {
			f.identityErr = nil
			i.Email = ""
			i.EmailVerified = false
		}},
		{name: "creation denied", change: func(_ *fakeStore, _ *auth.OAuthIdentity, b *bool) { *b = false }, want: ErrAccountCreationRequired},
		{name: "new verified user", change: func(f *fakeStore, _ *auth.OAuthIdentity, _ *bool) { f.userErr = ErrNotFound }, created: true},
		{name: "matching email never links", want: ErrLinkRequired},
		{name: "unverified email", change: func(_ *fakeStore, i *auth.OAuthIdentity, _ *bool) { i.EmailVerified = false }, want: ErrInvalidIdentity},
		{name: "missing email", change: func(_ *fakeStore, i *auth.OAuthIdentity, _ *bool) { i.Email = "" }, want: ErrInvalidIdentity},
		{name: "unsupported provider", change: func(_ *fakeStore, i *auth.OAuthIdentity, _ *bool) { i.Provider = "other" }, want: ErrInvalidIdentity},
		{name: "empty subject", change: func(_ *fakeStore, i *auth.OAuthIdentity, _ *bool) { i.ProviderSubject = " " }, want: ErrInvalidIdentity},
		{name: "disabled existing user", change: func(f *fakeStore, _ *auth.OAuthIdentity, _ *bool) { f.identityErr = nil; f.user.Status = "disabled" }, want: ErrAccountDisabled},
		{name: "identity insert rollback", change: func(f *fakeStore, _ *auth.OAuthIdentity, _ *bool) { f.userErr = ErrNotFound; f.linkErr = ErrConflict }, want: ErrLinkConflict},
		{name: "email race", change: func(f *fakeStore, _ *auth.OAuthIdentity, _ *bool) { f.userErr = ErrNotFound; f.createErr = ErrConflict }, want: ErrLinkRequired},
		{name: "commit failure", change: func(f *fakeStore, _ *auth.OAuthIdentity, _ *bool) {
			f.userErr = ErrNotFound
			f.commitErr = errors.New("private detail")
		}, want: ErrAuthenticationUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeStore{user: db.User{ID: 1, Status: "active"}, identity: db.UserOauthIdentity{UserID: 1}, identityErr: ErrNotFound}
			i := auth.OAuthIdentity{Provider: auth.OAuthProviderApple, ProviderSubject: "stable", Email: "relay@example.com", EmailVerified: true}
			allow := true
			if tc.change != nil {
				tc.change(f, &i, &allow)
			}
			user, err := NewAuthenticationService(f).ResolveOAuth(context.Background(), i, allow)
			if !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
			if f.created != tc.created || f.linked != tc.created {
				t.Fatal("unexpected account mutation")
			}
			if err != nil && user.ID != 0 {
				t.Fatal("failed authentication returned user")
			}
		})
	}
}

func TestExplicitLinking(t *testing.T) {
	for _, tc := range []struct {
		name                              string
		id                                int64
		identityErr, providerErr, linkErr error
		owner                             int64
		status                            string
		want                              error
		linked                            bool
	}{
		{name: "new link", id: 1, identityErr: ErrNotFound, providerErr: ErrNotFound, status: "active", linked: true},
		{name: "idempotent link", id: 1, owner: 1, status: "active"},
		{name: "other owner", id: 1, owner: 2, status: "active", want: ErrLinkConflict},
		{name: "provider already linked", id: 1, identityErr: ErrNotFound, status: "active", want: ErrLinkConflict},
		{name: "unauthenticated", want: ErrInvalidCredentials},
		{name: "disabled", id: 1, status: "disabled", want: ErrAccountDisabled},
		{name: "concurrent conflict", id: 1, identityErr: ErrNotFound, providerErr: ErrNotFound, linkErr: ErrConflict, status: "active", want: ErrLinkConflict},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeStore{user: db.User{ID: tc.id, Status: tc.status}, identity: db.UserOauthIdentity{UserID: tc.owner}, identityErr: tc.identityErr, providerErr: tc.providerErr, linkErr: tc.linkErr}
			_, err := NewAuthenticationService(f).LinkOAuth(context.Background(), tc.id, auth.OAuthIdentity{Provider: auth.OAuthProviderGoogle, ProviderSubject: "subject"})
			if !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
			if f.created || f.linked != tc.linked {
				t.Fatal("unexpected mutation")
			}
		})
	}
}

func TestLocalUserResolution(t *testing.T) {
	for _, status := range []string{"active", "disabled", ""} {
		f := &fakeStore{user: db.User{ID: 1, Status: status}}
		_, err := NewAuthenticationService(f).ResolveLocalUser(context.Background(), 1)
		if (err == nil) != (status == "active") {
			t.Fatalf("status %q: %v", status, err)
		}
	}
}
