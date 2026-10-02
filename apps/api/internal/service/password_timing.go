package service

import "multipurpose-booking-system/api/internal/auth"

// A public dummy hash performs the same Argon2id work as current credentials for
// unknown and OAuth-only accounts. Its result never grants authentication.
// This reduces the obvious timing gap; it does not promise constant-time database access.
func verifyMissingCredential(password string) {
	const dummy = "$argon2id$v=19$m=65536,t=3,p=4$MDEyMzQ1Njc4OWFiY2RlZg$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	_, _ = auth.VerifyPassword(password, dummy)
}
