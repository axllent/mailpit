package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

// fakeIdP is a minimal OIDC provider for tests. It serves
// /.well-known/openid-configuration and /jwks, signs JWTs on demand,
// and counts how many times the JWKS endpoint is hit.
type fakeIdP struct {
	t        *testing.T
	server   *httptest.Server
	signer   jose.Signer
	priv     *rsa.PrivateKey
	kid      string
	jwksHits int64
}

func newFakeIdP(t *testing.T) *fakeIdP {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa key: %v", err)
	}
	kid := "test-key-1"
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: priv},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", kid),
	)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}

	f := &fakeIdP{t: t, priv: priv, kid: kid, signer: signer}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                f.IssuerURL(),
			"jwks_uri":                              f.IssuerURL() + "/jwks",
			"authorization_endpoint":                f.IssuerURL() + "/authorize",
			"token_endpoint":                        f.IssuerURL() + "/token",
			"id_token_signing_alg_values_supported": []string{"RS256"},
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&f.jwksHits, 1)
		jwk := jose.JSONWebKey{Key: &priv.PublicKey, KeyID: kid, Algorithm: "RS256", Use: "sig"}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}})
	})
	f.server = httptest.NewServer(mux)
	return f
}

func (f *fakeIdP) Close()             { f.server.Close() }
func (f *fakeIdP) IssuerURL() string  { return f.server.URL }
func (f *fakeIdP) JWKSHits() int64    { return atomic.LoadInt64(&f.jwksHits) }
func (f *fakeIdP) resetHits()         { atomic.StoreInt64(&f.jwksHits, 0) }

// issue signs a JWT with default claims (iss=this issuer, aud=clientID,
// exp=+1h) and applies any caller-supplied overrides.
func (f *fakeIdP) issue(clientID, sub string, override map[string]any) string {
	f.t.Helper()
	claims := map[string]any{
		"iss": f.IssuerURL(),
		"aud": clientID,
		"sub": sub,
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	maps.Copy(claims, override)
	tok, err := jwt.Signed(f.signer).Claims(claims).Serialize()
	if err != nil {
		f.t.Fatalf("sign jwt: %v", err)
	}
	return tok
}

// resetOIDCState clears the package globals so tests don't bleed into each other.
func resetOIDCState() func() {
	prev := OIDCVerifier
	return func() { OIDCVerifier = prev }
}

func TestInitOIDC_Disabled(t *testing.T) {
	defer resetOIDCState()()
	OIDCVerifier = nil
	if err := InitOIDC(context.Background(), "", ""); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if OIDCVerifier != nil {
		t.Fatalf("expected verifier to remain nil when issuer is empty")
	}
}

func TestInitOIDC_MissingClientID(t *testing.T) {
	defer resetOIDCState()()
	idp := newFakeIdP(t)
	defer idp.Close()
	err := InitOIDC(context.Background(), idp.IssuerURL(), "")
	if err == nil {
		t.Fatalf("expected error when client ID is empty")
	}
	if !strings.Contains(err.Error(), "client ID") {
		t.Fatalf("expected error to mention client ID, got %v", err)
	}
}

func TestInitOIDC_UnreachableIssuer(t *testing.T) {
	defer resetOIDCState()()
	// 127.0.0.1:1 — guaranteed to refuse connections.
	if err := InitOIDC(context.Background(), "http://127.0.0.1:1/realms/x", "mailpit"); err == nil {
		t.Fatalf("expected error for unreachable issuer")
	}
}

func TestInitOIDC_PreloadsJWKS(t *testing.T) {
	defer resetOIDCState()()
	idp := newFakeIdP(t)
	defer idp.Close()
	if err := InitOIDC(context.Background(), idp.IssuerURL(), "mailpit"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if got := idp.JWKSHits(); got != 1 {
		t.Fatalf("expected exactly 1 JWKS hit on boot, got %d", got)
	}
}

func TestVerifyBearer_HappyPath(t *testing.T) {
	defer resetOIDCState()()
	idp := newFakeIdP(t)
	defer idp.Close()
	if err := InitOIDC(context.Background(), idp.IssuerURL(), "mailpit"); err != nil {
		t.Fatalf("init: %v", err)
	}
	tok := idp.issue("mailpit", "alice", nil)
	sub, ok := VerifyBearer(context.Background(), tok)
	if !ok {
		t.Fatalf("expected verify to succeed")
	}
	if sub != "alice" {
		t.Fatalf("expected sub=alice, got %q", sub)
	}
}

func TestVerifyBearer_StripsBearerPrefix(t *testing.T) {
	defer resetOIDCState()()
	idp := newFakeIdP(t)
	defer idp.Close()
	if err := InitOIDC(context.Background(), idp.IssuerURL(), "mailpit"); err != nil {
		t.Fatalf("init: %v", err)
	}
	tok := idp.issue("mailpit", "bob", nil)
	if _, ok := VerifyBearer(context.Background(), "Bearer "+tok); !ok {
		t.Fatalf("expected verify to succeed with Bearer prefix")
	}
	if _, ok := VerifyBearer(context.Background(), tok); !ok {
		t.Fatalf("expected verify to succeed without prefix")
	}
}

func TestVerifyBearer_WrongAudience(t *testing.T) {
	defer resetOIDCState()()
	idp := newFakeIdP(t)
	defer idp.Close()
	if err := InitOIDC(context.Background(), idp.IssuerURL(), "mailpit"); err != nil {
		t.Fatalf("init: %v", err)
	}
	tok := idp.issue("someone-else", "alice", nil)
	if _, ok := VerifyBearer(context.Background(), tok); ok {
		t.Fatalf("expected verify to fail for wrong audience")
	}
}

func TestVerifyBearer_WrongIssuer(t *testing.T) {
	defer resetOIDCState()()
	idp := newFakeIdP(t)
	defer idp.Close()
	if err := InitOIDC(context.Background(), idp.IssuerURL(), "mailpit"); err != nil {
		t.Fatalf("init: %v", err)
	}
	tok := idp.issue("mailpit", "alice", map[string]any{"iss": "https://attacker.example/"})
	if _, ok := VerifyBearer(context.Background(), tok); ok {
		t.Fatalf("expected verify to fail for wrong issuer")
	}
}

func TestVerifyBearer_Expired(t *testing.T) {
	defer resetOIDCState()()
	idp := newFakeIdP(t)
	defer idp.Close()
	if err := InitOIDC(context.Background(), idp.IssuerURL(), "mailpit"); err != nil {
		t.Fatalf("init: %v", err)
	}
	tok := idp.issue("mailpit", "alice", map[string]any{"exp": time.Now().Add(-time.Hour).Unix()})
	if _, ok := VerifyBearer(context.Background(), tok); ok {
		t.Fatalf("expected verify to fail for expired token")
	}
}

func TestVerifyBearer_BadSignature(t *testing.T) {
	defer resetOIDCState()()
	idp := newFakeIdP(t)
	defer idp.Close()
	if err := InitOIDC(context.Background(), idp.IssuerURL(), "mailpit"); err != nil {
		t.Fatalf("init: %v", err)
	}
	// Token signed by a different (unrelated) signer.
	other, _ := rsa.GenerateKey(rand.Reader, 2048)
	otherSigner, _ := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: other},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "rogue"),
	)
	tok, _ := jwt.Signed(otherSigner).Claims(map[string]any{
		"iss": idp.IssuerURL(),
		"aud": "mailpit",
		"sub": "alice",
		"exp": time.Now().Add(time.Hour).Unix(),
	}).Serialize()
	if _, ok := VerifyBearer(context.Background(), tok); ok {
		t.Fatalf("expected verify to fail for unrelated signing key")
	}
}

func TestVerifyBearer_EmptyAndDisabled(t *testing.T) {
	defer resetOIDCState()()
	OIDCVerifier = nil
	if _, ok := VerifyBearer(context.Background(), "anything"); ok {
		t.Fatalf("expected false when verifier is nil")
	}
	idp := newFakeIdP(t)
	defer idp.Close()
	if err := InitOIDC(context.Background(), idp.IssuerURL(), "mailpit"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, ok := VerifyBearer(context.Background(), ""); ok {
		t.Fatalf("expected false for empty token")
	}
}

func TestJWKSCachedAcrossVerifies(t *testing.T) {
	defer resetOIDCState()()
	idp := newFakeIdP(t)
	defer idp.Close()
	if err := InitOIDC(context.Background(), idp.IssuerURL(), "mailpit"); err != nil {
		t.Fatalf("init: %v", err)
	}
	idp.resetHits() // discount the boot fetch
	for i := range 50 {
		tok := idp.issue("mailpit", "alice", nil)
		if _, ok := VerifyBearer(context.Background(), tok); !ok {
			t.Fatalf("iter %d: expected verify to succeed", i)
		}
	}
	if got := idp.JWKSHits(); got != 0 {
		t.Fatalf("expected 0 JWKS hits across cached verifies, got %d", got)
	}
}

func TestJWKSRefreshAfterTTL(t *testing.T) {
	defer resetOIDCState()()
	restore := SetJWKSTTLForTests(50 * time.Millisecond)
	defer restore()

	idp := newFakeIdP(t)
	defer idp.Close()
	if err := InitOIDC(context.Background(), idp.IssuerURL(), "mailpit"); err != nil {
		t.Fatalf("init: %v", err)
	}
	idp.resetHits()
	// First verify uses cached keys (within TTL).
	tok := idp.issue("mailpit", "alice", nil)
	if _, ok := VerifyBearer(context.Background(), tok); !ok {
		t.Fatalf("expected verify to succeed (cache)")
	}
	if got := idp.JWKSHits(); got != 0 {
		t.Fatalf("expected 0 JWKS hits before TTL, got %d", got)
	}
	time.Sleep(80 * time.Millisecond)
	// Now TTL elapsed — next verify should trigger a refresh.
	tok2 := idp.issue("mailpit", "alice", nil)
	if _, ok := VerifyBearer(context.Background(), tok2); !ok {
		t.Fatalf("expected verify to succeed (refresh)")
	}
	if got := idp.JWKSHits(); got != 1 {
		t.Fatalf("expected 1 JWKS hit after TTL refresh, got %d", got)
	}
}
