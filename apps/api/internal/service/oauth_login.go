package service

import (
	"context"
	"multipurpose-booking-system/api/internal/auth"
)

type OAuthLoginService struct {
	authentication *AuthenticationService
	sessions       *SessionService
	allowCreation  bool
}

func NewOAuthLoginService(authentication *AuthenticationService, sessions *SessionService, allowCreation bool) *OAuthLoginService {
	return &OAuthLoginService{authentication, sessions, allowCreation}
}
func (s *OAuthLoginService) LoginOAuth(ctx context.Context, identity auth.OAuthIdentity) (IssuedSession, error) {
	return s.sessions.AuthenticateOAuth(ctx, s.authentication, identity, s.allowCreation)
}
