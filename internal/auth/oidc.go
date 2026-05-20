// OIDC verification for the web UI.
//
// When configured (issuer + client ID), Mailpit verifies incoming
// `Authorization: Bearer <jwt>` headers against the IdP's published
// signing keys. The JWKS is fetched once at startup and cached in
// memory for jwksTTL (24h by default) or until restart — never
// re-fetched per request.

package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	jose "github.com/go-jose/go-jose/v4"
)

// jwksTTL is overridable in tests via init via SetJWKSTTLForTests.
var jwksTTL = 24 * time.Hour

// OIDCVerifier is nil when OIDC is disabled.
var OIDCVerifier *oidc.IDTokenVerifier

var oidcSupportedAlgs = []jose.SignatureAlgorithm{
	jose.RS256, jose.RS384, jose.RS512,
	jose.ES256, jose.ES384, jose.ES512,
	jose.PS256, jose.PS384, jose.PS512,
}

// cachedKeySet implements oidc.KeySet. JWKS is held in memory for jwksTTL.
type cachedKeySet struct {
	jwksURL string

	mu      sync.RWMutex
	keys    *jose.JSONWebKeySet
	fetched time.Time
}

func (c *cachedKeySet) VerifySignature(ctx context.Context, raw string) ([]byte, error) {
	if err := c.ensureFresh(ctx); err != nil {
		return nil, err
	}
	jws, err := jose.ParseSigned(raw, oidcSupportedAlgs)
	if err != nil {
		return nil, fmt.Errorf("oidc: parse jwt: %w", err)
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.keys == nil {
		return nil, errors.New("oidc: jwks not loaded")
	}
	for _, k := range c.keys.Keys {
		if payload, err := jws.Verify(k); err == nil {
			return payload, nil
		}
	}
	return nil, errors.New("oidc: no matching signing key")
}

func (c *cachedKeySet) ensureFresh(ctx context.Context) error {
	c.mu.RLock()
	fresh := c.keys != nil && time.Since(c.fetched) < jwksTTL
	c.mu.RUnlock()
	if fresh {
		return nil
	}
	return c.fetch(ctx)
}

func (c *cachedKeySet) fetch(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.jwksURL, nil)
	if err != nil {
		return fmt.Errorf("oidc: jwks request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("oidc: jwks fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("oidc: jwks fetch: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("oidc: jwks read: %w", err)
	}
	var ks jose.JSONWebKeySet
	if err := json.Unmarshal(body, &ks); err != nil {
		return fmt.Errorf("oidc: jwks parse: %w", err)
	}
	c.mu.Lock()
	c.keys = &ks
	c.fetched = time.Now()
	c.mu.Unlock()
	return nil
}

// InitOIDC configures the OIDC verifier from the issuer URL and client ID.
// When issuer is empty, OIDC is disabled and the verifier remains nil.
// JWKS is fetched once here so an unreachable IdP makes Mailpit fail to start.
func InitOIDC(ctx context.Context, issuer, clientID string) error {
	if issuer == "" {
		OIDCVerifier = nil
		return nil
	}
	if clientID == "" {
		return errors.New("OIDC client ID is required when issuer is set")
	}
	p, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return fmt.Errorf("oidc discovery: %w", err)
	}
	var claims struct {
		JWKSURL string `json:"jwks_uri"`
	}
	if err := p.Claims(&claims); err != nil {
		return fmt.Errorf("oidc claims: %w", err)
	}
	if claims.JWKSURL == "" {
		return errors.New("oidc: provider discovery returned no jwks_uri")
	}
	keys := &cachedKeySet{jwksURL: claims.JWKSURL}
	if err := keys.fetch(ctx); err != nil {
		return err
	}
	OIDCVerifier = oidc.NewVerifier(issuer, keys, &oidc.Config{ClientID: clientID})
	return nil
}

// VerifyBearer accepts a raw JWT (with or without a "Bearer " prefix) and
// returns (subject, true) when the token verifies against the configured IdP.
func VerifyBearer(ctx context.Context, raw string) (string, bool) {
	if OIDCVerifier == nil || raw == "" {
		return "", false
	}
	raw = strings.TrimPrefix(raw, "Bearer ")
	tok, err := OIDCVerifier.Verify(ctx, raw)
	if err != nil {
		return "", false
	}
	return tok.Subject, true
}

// SetJWKSTTLForTests overrides the cache TTL. Test-only.
func SetJWKSTTLForTests(d time.Duration) func() {
	prev := jwksTTL
	jwksTTL = d
	return func() { jwksTTL = prev }
}
