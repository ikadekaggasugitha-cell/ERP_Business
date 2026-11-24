package auth_test

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sinergiinvestasiproperti/erp/services/api-gateway/internal/auth"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func signRS256(t *testing.T, priv *rsa.PrivateKey, kid string, payload map[string]any) string {
	t.Helper()
	hdr, _ := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT", "kid": kid})
	body, _ := json.Marshal(payload)
	input := base64.RawURLEncoding.EncodeToString(hdr) + "." + base64.RawURLEncoding.EncodeToString(body)
	h := sha256.Sum256([]byte(input))
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, h[:])
	if err != nil {
		t.Fatalf("sign rs256: %v", err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func signES256(t *testing.T, priv *ecdsa.PrivateKey, kid string, payload map[string]any) string {
	t.Helper()
	hdr, _ := json.Marshal(map[string]string{"alg": "ES256", "typ": "JWT", "kid": kid})
	body, _ := json.Marshal(payload)
	input := base64.RawURLEncoding.EncodeToString(hdr) + "." + base64.RawURLEncoding.EncodeToString(body)
	h := sha256.Sum256([]byte(input))
	r, s, err := ecdsa.Sign(rand.Reader, priv, h[:])
	if err != nil {
		t.Fatalf("sign es256: %v", err)
	}
	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	s.FillBytes(sig[32:])
	return input + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func rsaToJWK(pub *rsa.PublicKey, kid string) map[string]any {
	return map[string]any{
		"kty": "RSA",
		"kid": kid,
		"use": "sig",
		"alg": "RS256",
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}
}

func ecToJWK(pub *ecdsa.PublicKey, kid string) map[string]any {
	xBytes := make([]byte, 32)
	yBytes := make([]byte, 32)
	pub.X.FillBytes(xBytes)
	pub.Y.FillBytes(yBytes)
	return map[string]any{
		"kty": "EC",
		"kid": kid,
		"use": "sig",
		"alg": "ES256",
		"crv": "P-256",
		"x":   base64.RawURLEncoding.EncodeToString(xBytes),
		"y":   base64.RawURLEncoding.EncodeToString(yBytes),
	}
}

func TestOIDCVerifierRS256ES256AndKidRotation(t *testing.T) {
	t.Parallel()

	rsaKey1, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	rsaKey2, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	var rotated atomic.Bool
	var jwksFetches atomic.Int32
	issuer := "https://idp.sinergi.example.com"

	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			var body []byte
			switch r.URL.Path {
			case "/.well-known/openid-configuration":
				body, _ = json.Marshal(map[string]string{
					"issuer":   issuer,
					"jwks_uri": issuer + "/protocol/openid-connect/certs",
				})
			case "/protocol/openid-connect/certs":
				jwksFetches.Add(1)
				keys := []map[string]any{
					rsaToJWK(&rsaKey1.PublicKey, "rsa-1"),
					ecToJWK(&ecKey.PublicKey, "ec-1"),
				}
				if rotated.Load() {
					keys = append(keys, rsaToJWK(&rsaKey2.PublicKey, "rsa-2"))
				}
				body, _ = json.Marshal(map[string]any{"keys": keys})
			default:
				return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(""))}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(string(body))),
			}, nil
		}),
	}

	verifier, err := auth.NewOIDCVerifier(auth.OIDCConfig{
		IssuerURL:       issuer,
		Audience:        "erp-api",
		HTTPClient:      client,
		RefreshCooldown: 10 * time.Millisecond,
		ClockSkew:       5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewOIDCVerifier: %v", err)
	}

	now := time.Now().UTC()

	// 1. Valid RS256 token with Keycloak realm_access.roles
	rsTok := signRS256(t, rsaKey1, "rsa-1", map[string]any{
		"jti":                "jti-rs-1",
		"iss":                issuer,
		"aud":                []string{"erp-api", "account"},
		"sub":                "00000000-0000-0000-0000-000000000201",
		"preferred_username": "cfo.holding",
		"name":               "CFO Holding",
		"email":              "cfo@sinergi.example.com",
		"org_id":             "11111111-1111-1111-1111-111111111111",
		"realm_access":       map[string]any{"roles": []string{"HOLDING_ADMIN"}},
		"admin_tier":         1,
		"iat":                now.Add(-time.Minute).Unix(),
		"exp":                now.Add(time.Hour).Unix(),
	})
	claims, err := verifier.Verify(context.Background(), rsTok, now)
	if err != nil {
		t.Fatalf("Verify RS256 failed: %v", err)
	}
	if claims.JTI != "jti-rs-1" || claims.Username != "cfo.holding" || len(claims.Roles) != 1 || claims.Roles[0] != "HOLDING_ADMIN" {
		t.Fatalf("unexpected RS256 claims: %+v", claims)
	}

	// 2. Valid ES256 token
	ecTok := signES256(t, ecKey, "ec-1", map[string]any{
		"jti":    "jti-ec-1",
		"iss":    issuer,
		"aud":    "erp-api",
		"sub":    "00000000-0000-0000-0000-000000000202",
		"roles":  []string{"FINANCE_MANAGER"},
		"org_id": "22222222-2222-2222-2222-222222222222",
		"iat":    now.Add(-time.Minute).Unix(),
		"exp":    now.Add(time.Hour).Unix(),
	})
	ecClaims, err := verifier.Verify(context.Background(), ecTok, now)
	if err != nil {
		t.Fatalf("Verify ES256 failed: %v", err)
	}
	if ecClaims.Subject != "00000000-0000-0000-0000-000000000202" || ecClaims.OrgID != "22222222-2222-2222-2222-222222222222" {
		t.Fatalf("unexpected ES256 claims: %+v", ecClaims)
	}

	// 3. Key rotation: unknown kid triggers JWKS refresh after cooldown
	rotated.Store(true)
	rsTok2 := signRS256(t, rsaKey2, "rsa-2", map[string]any{
		"iss": issuer,
		"aud": "erp-api",
		"sub": "00000000-0000-0000-0000-000000000203",
		"exp": now.Add(time.Hour).Unix(),
	})
	if _, err := verifier.Verify(context.Background(), rsTok2, now.Add(50*time.Millisecond)); err != nil {
		t.Fatalf("Verify after kid rotation failed: %v", err)
	}
	if jwksFetches.Load() < 2 {
		t.Fatalf("expected JWKS refresh on kid rotation, fetches=%d", jwksFetches.Load())
	}

	// 4. Reject bad issuer / bad audience / expired token
	badIss := signRS256(t, rsaKey1, "rsa-1", map[string]any{
		"iss": "https://evil.example.com",
		"aud": "erp-api",
		"sub": "u1",
		"exp": now.Add(time.Hour).Unix(),
	})
	if _, err := verifier.Verify(context.Background(), badIss, now); !errors.Is(err, auth.ErrInvalidIssuer) {
		t.Fatalf("expected ErrInvalidIssuer, got %v", err)
	}

	badAud := signRS256(t, rsaKey1, "rsa-1", map[string]any{
		"iss": issuer,
		"aud": "other-client",
		"sub": "u1",
		"exp": now.Add(time.Hour).Unix(),
	})
	if _, err := verifier.Verify(context.Background(), badAud, now); !errors.Is(err, auth.ErrInvalidAud) {
		t.Fatalf("expected ErrInvalidAud, got %v", err)
	}

	expired := signRS256(t, rsaKey1, "rsa-1", map[string]any{
		"iss": issuer,
		"aud": "erp-api",
		"sub": "u1",
		"exp": now.Add(-time.Minute).Unix(),
	})
	if _, err := verifier.Verify(context.Background(), expired, now); !errors.Is(err, auth.ErrExpiredToken) {
		t.Fatalf("expected ErrExpiredToken, got %v", err)
	}
}

func TestSessionStoreRefreshRotationAndJTIRevocation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := auth.NewSessionStore("")

	// JTI revocation
	if store.IsRevoked(ctx, "jti-123") {
		t.Fatal("fresh jti should not be revoked")
	}
	if err := store.RevokeJTI(ctx, "jti-123", time.Minute); err != nil {
		t.Fatalf("RevokeJTI: %v", err)
	}
	if !store.IsRevoked(ctx, "jti-123") {
		t.Fatal("expected jti-123 to be revoked")
	}

	// Refresh rotation
	baseClaims := auth.Claims{
		Subject:  "00000000-0000-0000-0000-000000000101",
		Username: "admin.ho",
		OrgID:    "11111111-1111-1111-1111-111111111111",
		Roles:    []string{"HOLDING_ADMIN"},
	}
	rt1, err := store.IssueRefreshToken(ctx, baseClaims, time.Hour)
	if err != nil || rt1 == "" {
		t.Fatalf("IssueRefreshToken: %v", err)
	}

	gotClaims, rt2, err := store.RotateRefreshToken(ctx, rt1, time.Hour)
	if err != nil {
		t.Fatalf("RotateRefreshToken(rt1): %v", err)
	}
	if gotClaims.Username != "admin.ho" || rt2 == "" || rt2 == rt1 {
		t.Fatalf("unexpected rotated result: claims=%+v rt2=%q", gotClaims, rt2)
	}

	// Replaying rt1 must fail with ErrRefreshTokenReused AND revoke rt2 (same family).
	if _, _, err := store.RotateRefreshToken(ctx, rt1, time.Hour); !errors.Is(err, auth.ErrRefreshTokenReused) {
		t.Fatalf("expected ErrRefreshTokenReused on replay, got %v", err)
	}
	if _, _, err := store.RotateRefreshToken(ctx, rt2, time.Hour); !errors.Is(err, auth.ErrInvalidRefreshToken) {
		t.Fatalf("expected rt2 invalidated after family revocation, got %v", err)
	}
}
