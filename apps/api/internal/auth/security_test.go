package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestSecuritySignedOIDCClaims(t *testing.T) {
	key := mustGenerateRSAKey(t)
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, &jose.SignerOptions{ExtraHeaders: map[jose.HeaderKey]interface{}{"kid": "security-key"}})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: key.Public(), KeyID: "security-key", Algorithm: "RS256", Use: "sig"}}})
	}))
	defer server.Close()
	verifier, err := NewOIDCVerifier(server.Client(), server.URL, "https://issuer.example", "client", "nonce", []jose.SignatureAlgorithm{jose.RS256})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name   string
		change func(*OIDCClaims)
	}{
		{"issuer", func(c *OIDCClaims) { c.Issuer = "https://attacker.example" }},
		{"audience", func(c *OIDCClaims) { c.Audience = OIDCAudience{"other-client"} }},
		{"nonce", func(c *OIDCClaims) { c.Nonce = "attacker-nonce" }},
		{"missing nonce", func(c *OIDCClaims) { c.Nonce = "" }},
		{"expired boundary", func(c *OIDCClaims) { c.ExpiresAt = now.Unix() }},
		{"future issued", func(c *OIDCClaims) { c.IssuedAt = now.Add(time.Second).Unix() }},
		{"not before", func(c *OIDCClaims) { v := now.Add(time.Second).Unix(); c.NotBefore = &v }},
		{"missing subject", func(c *OIDCClaims) { c.Subject = "" }},
		{"multiple audience no azp", func(c *OIDCClaims) { c.Audience = OIDCAudience{"client", "other"} }},
		{"single audience wrong azp", func(c *OIDCClaims) { c.AuthorizedParty = "other-client" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := OIDCClaims{Issuer: "https://issuer.example", Audience: OIDCAudience{"client"}, Subject: "subject", Nonce: "nonce", IssuedAt: now.Add(-time.Minute).Unix(), ExpiresAt: now.Add(time.Minute).Unix()}
			tc.change(&c)
			token, err := jwt.Signed(signer).Claims(c).Serialize()
			if err != nil {
				t.Fatal(err)
			}
			got, err := verifier.Verify(context.Background(), token, now)
			if err == nil || got.Subject != "" {
				t.Fatal("unsafe signed claims accepted")
			}
		})
	}
}

func TestSecurityPKCEChallengeAndFreshness(t *testing.T) {
	cfg := GoogleOAuthConfig("client", "secret", "https://app.example/callback")
	seen := map[string]bool{}
	for i := 0; i < 8; i++ {
		a, err := BuildOAuthAuthorization(context.Background(), cfg, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := url.Parse(a.URL)
		if err != nil {
			t.Fatal(err)
		}
		q := parsed.Query()
		sum := sha256.Sum256([]byte(a.PKCEVerifier))
		if q.Get("code_challenge") != base64.RawURLEncoding.EncodeToString(sum[:]) || q.Get("code_challenge_method") != "S256" || q.Get("code_verifier") != "" {
			t.Fatal("PKCE binding or verifier disclosure")
		}
		for _, value := range []string{a.State.State, a.State.Nonce, a.PKCEVerifier} {
			if len(value) < 32 || seen[value] {
				t.Fatal("reused or weak transaction secret")
			}
			seen[value] = true
		}
	}
}

func TestSecuritySingleAudienceAuthorizedParty(t *testing.T) {
	now := time.Now()
	claims := OIDCClaims{Issuer: "issuer", Subject: "subject", Audience: OIDCAudience{"client"}, AuthorizedParty: "client", Nonce: "nonce", IssuedAt: now.Add(-time.Minute).Unix(), ExpiresAt: now.Add(time.Minute).Unix()}
	if err := claims.Validate("issuer", "client", "nonce", now); err != nil {
		t.Fatal(err)
	}
	claims.AuthorizedParty = ""
	if err := claims.Validate("issuer", "client", "nonce", now); err != nil {
		t.Fatal("optional azp rejected", err)
	}
}
func FuzzOAuthTransactionDecode(f *testing.F) {
	codec, err := NewOAuthTransactionCodec(make([]byte, 32))
	if err != nil {
		f.Fatal(err)
	}
	original := validOAuthTransaction()
	encoded, err := codec.Encode(original)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(encoded)
	f.Add("")
	f.Add("not-a-transaction")
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 8192 {
			t.Skip()
		}
		got, err := codec.Decode(input, time.Unix(original.CreatedAtUnix, 0).Add(time.Minute))
		if err != nil && got != (OAuthTransaction{}) {
			t.Fatal("invalid encrypted cookie returned transaction data")
		}
		if err == nil && got != original {
			t.Fatal("unauthenticated transaction accepted")
		}
	})
}
