package service

import (
	"context"
	"errors"
	"testing"
)

type authorizationStoreStub struct {
	state AuthorizationState
	err   error
	calls int
}

func (s *authorizationStoreStub) GetAuthorizationState(context.Context, int64) (AuthorizationState, error) {
	s.calls++
	return s.state, s.err
}
func TestAuthorizationPolicy(t *testing.T) {
	for _, tc := range []struct {
		name            string
		id              int64
		active          bool
		grants, allowed []Role
		storeErr, want  error
	}{
		{"customer", 1, true, nil, []Role{RoleCustomer}, nil, nil},
		{"no implicit admin", 1, true, nil, []Role{RolePlatformAdmin}, nil, ErrForbidden},
		{"admin", 1, true, []Role{RolePlatformAdmin}, []Role{RolePlatformAdmin}, nil, nil},
		{"operator", 1, true, []Role{RolePlatformOperator}, []Role{RolePlatformOperator}, nil, nil},
		{"no hierarchy", 1, true, []Role{RolePlatformAdmin}, []Role{RolePlatformOperator}, nil, ErrForbidden},
		{"any role", 1, true, []Role{RolePlatformOperator}, []Role{RolePlatformAdmin, RolePlatformOperator}, nil, nil},
		{"disabled", 1, false, []Role{RolePlatformAdmin}, []Role{RolePlatformAdmin}, nil, ErrForbidden},
		{"anonymous", 0, true, nil, []Role{RoleCustomer}, nil, ErrForbidden},
		{"empty policy", 1, true, nil, nil, nil, ErrForbidden},
		{"unknown policy", 1, true, nil, []Role{"ROOT"}, nil, ErrForbidden},
		{"unknown grant", 1, true, []Role{"ROOT"}, []Role{RoleCustomer}, nil, ErrForbidden},
		{"store failure", 1, true, nil, []Role{RoleCustomer}, errors.New("private"), ErrAuthorizationUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &authorizationStoreStub{state: AuthorizationState{Active: tc.active, PlatformRoles: tc.grants}, err: tc.storeErr}
			s := NewAuthorizationService(store)
			if err := s.RequireAnyRole(context.Background(), tc.id, tc.allowed...); !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
		})
	}
	for _, role := range []Role{RoleBusinessOwner, RoleBusinessAdmin, RoleManager, RoleStaff} {
		s := NewAuthorizationService(&authorizationStoreStub{state: AuthorizationState{Active: true, PlatformRoles: []Role{RolePlatformAdmin}}})
		if err := s.RequireAnyRole(context.Background(), 1, RoleCustomer, role); !errors.Is(err, ErrForbidden) {
			t.Fatal("business policy allowed without membership")
		}
	}
	if err := NewAuthorizationService(nil).RequireAnyRole(context.Background(), 1, RoleCustomer); !errors.Is(err, ErrAuthorizationUnavailable) {
		t.Fatal(err)
	}
}
func TestResourceOwnership(t *testing.T) {
	for _, tc := range []struct {
		user, owner int64
		allow       bool
	}{{1, 1, true}, {1, 2, false}, {0, 0, false}, {1, 0, false}, {-1, -1, false}} {
		err := RequireOwner(tc.user, tc.owner)
		if (err == nil) != tc.allow {
			t.Fatal("incorrect ownership decision")
		}
	}
}
