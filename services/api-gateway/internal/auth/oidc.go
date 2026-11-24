package auth

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	ErrUnsupportedAlg = errors.New("unsupported jwt signing algorithm")
	ErrUnknownKeyID   = errors.New("unknown jwks key id")
	ErrInvalidIssuer  = errors.New("invalid token issuer")
	ErrInvalidAud     = errors.New("invalid token audience")
	ErrTokenNotActive = errors.New("token not yet valid (nbf)")
	ErrTokenRevoked   = errors.New("token has been revoked")
)

// OIDCConfig configures OIDC Discovery and JWKS verification.
type OIDCConfig struct {
	IssuerURL       string
	Audience        string
	JWKSURL         string // Optional override; discovered via /.well-known/openid-configuration when empty
	HTTPClient      *http.Client
	CacheTTL        time.Duration
	RefreshCooldown time.Duration
	ClockSkew       time.Duration
}

// OIDCVerifier validates RS256 and ES256 JWTs issued by an OIDC Identity Provider.
type OIDCVerifier struct {
	issuer          string
	audience        string
	jwksURL         string
	client          *http.Client
	cacheTTL        time.Duration
	refreshCooldown time.Duration
	clockSkew       time.Duration

	mu          sync.RWMutex
	keys        map[string]crypto.PublicKey
	fetchedAt   time.Time
	lastRefresh time.Time
}

// NewOIDCVerifier creates an OIDCVerifier for the given issuer and audience.
func NewOIDCVerifier(cfg OIDCConfig) (*OIDCVerifier, error) {
	issuer := strings.TrimRight(strings.TrimSpace(cfg.IssuerURL), "/")
	if issuer == "" && strings.TrimSpace(cfg.JWKSURL) == "" {
		return nil, errors.New("OIDC_ISSUER_URL or OIDC_JWKS_URL is required")
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	cacheTTL := cfg.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = 15 * time.Minute
	}
	cooldown := cfg.RefreshCooldown
	if cooldown <= 0 {
		cooldown = 30 * time.Second
	}
	skew := cfg.ClockSkew
	if skew < 0 {
		skew = 0
	}
	if skew == 0 {
		skew = 30 * time.Second
	}
	return &OIDCVerifier{
		issuer:          issuer,
		audience:        strings.TrimSpace(cfg.Audience),
		jwksURL:         strings.TrimSpace(cfg.JWKSURL),
		client:          client,
		cacheTTL:        cacheTTL,
		refreshCooldown: cooldown,
		clockSkew:       skew,
		keys:            make(map[string]crypto.PublicKey),
	}, nil
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	Typ string `json:"typ"`
}

// Verify validates signature (RS256 or ES256), exp/nbf/iss/aud, and maps claims.
func (v *OIDCVerifier) Verify(ctx context.Context, rawToken string, now time.Time) (Claims, error) {
	parts := strings.Split(strings.TrimSpace(rawToken), ".")
	if len(parts) != 3 {
		return Claims{}, ErrInvalidToken
	}

	hdrBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	var hdr jwtHeader
	if err := json.Unmarshal(hdrBytes, &hdr); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if hdr.Alg != "RS256" && hdr.Alg != "ES256" {
		return Claims{}, fmt.Errorf("%w: %s", ErrUnsupportedAlg, hdr.Alg)
	}

	pubKey, err := v.resolveKey(ctx, hdr.Kid, now)
	if err != nil {
		return Claims{}, err
	}

	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	signingInput := []byte(parts[0] + "." + parts[1])
	digest := sha256.Sum256(signingInput)

	switch hdr.Alg {
	case "RS256":
		rsaKey, ok := pubKey.(*rsa.PublicKey)
		if !ok {
			return Claims{}, ErrInvalidToken
		}
		if err := rsa.VerifyPKCS1v15(rsaKey, crypto.SHA256, digest[:], sigBytes); err != nil {
			return Claims{}, ErrInvalidToken
		}
	case "ES256":
		ecKey, ok := pubKey.(*ecdsa.PublicKey)
		if !ok || len(sigBytes) != 64 {
			return Claims{}, ErrInvalidToken
		}
		r := new(big.Int).SetBytes(sigBytes[:32])
		s := new(big.Int).SetBytes(sigBytes[32:])
		if !ecdsa.Verify(ecKey, digest[:], r, s) {
			return Claims{}, ErrInvalidToken
		}
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payloadBytes, &raw); err != nil {
		return Claims{}, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	return v.mapAndValidateClaims(raw, now)
}

func (v *OIDCVerifier) mapAndValidateClaims(raw map[string]json.RawMessage, now time.Time) (Claims, error) {
	exp, hasExp := jsonInt64(raw["exp"])
	if !hasExp || now.Unix() >= exp+int64(v.clockSkew.Seconds()) {
		return Claims{}, ErrExpiredToken
	}
	if nbf, hasNbf := jsonInt64(raw["nbf"]); hasNbf {
		if now.Unix()+int64(v.clockSkew.Seconds()) < nbf {
			return Claims{}, ErrTokenNotActive
		}
	}
	if v.issuer != "" {
		iss := strings.TrimRight(jsonString(raw["iss"]), "/")
		if iss != v.issuer {
			return Claims{}, ErrInvalidIssuer
		}
	}
	if v.audience != "" {
		if !audContains(raw["aud"], v.audience) {
			return Claims{}, ErrInvalidAud
		}
	}

	sub := jsonString(raw["sub"])
	if sub == "" {
		return Claims{}, ErrInvalidToken
	}
	iat, _ := jsonInt64(raw["iat"])
	username := firstNonEmpty(jsonString(raw["preferred_username"]), jsonString(raw["username"]), jsonString(raw["email"]), sub)
	fullName := firstNonEmpty(jsonString(raw["name"]), jsonString(raw["fullName"]), username)
	email := jsonString(raw["email"])
	orgID := firstNonEmpty(jsonString(raw["org_id"]), jsonString(raw["orgId"]))
	orgScope := jsonStringSlice(raw["org_scope"])
	if len(orgScope) == 0 {
		orgScope = jsonStringSlice(raw["orgScope"])
	}
	if orgID != "" && !containsStr(orgScope, orgID) {
		orgScope = append([]string{orgID}, orgScope...)
	}

	roles := jsonStringSlice(raw["roles"])
	if len(roles) == 0 {
		// Support Keycloak realm_access.roles convention.
		if ra, ok := raw["realm_access"]; ok {
			var realmAccess struct {
				Roles []string `json:"roles"`
			}
			if json.Unmarshal(ra, &realmAccess) == nil {
				roles = realmAccess.Roles
			}
		}
	}
	perms := jsonStringSlice(raw["permissions"])
	adminTier, _ := jsonInt64(raw["admin_tier"])
	if adminTier == 0 {
		adminTier, _ = jsonInt64(raw["adminTier"])
	}

	return Claims{
		JTI:         jsonString(raw["jti"]),
		Subject:     sub,
		Username:    username,
		FullName:    fullName,
		Email:       email,
		OrgID:       orgID,
		OrgScope:    orgScope,
		Roles:       roles,
		Permissions: perms,
		AdminTier:   int(adminTier),
		TokenUse:    "access",
		IssuedAt:    iat,
		ExpiresAt:   exp,
	}, nil
}

func (v *OIDCVerifier) resolveKey(ctx context.Context, kid string, now time.Time) (crypto.PublicKey, error) {
	v.mu.RLock()
	fresh := !v.fetchedAt.IsZero() && now.Sub(v.fetchedAt) < v.cacheTTL
	key, ok := v.lookupKeyLocked(kid)
	canRefresh := v.lastRefresh.IsZero() || now.Sub(v.lastRefresh) >= v.refreshCooldown
	v.mu.RUnlock()

	if ok && fresh {
		return key, nil
	}
	if !ok && !canRefresh {
		return nil, ErrUnknownKeyID
	}
	if err := v.refreshJWKS(ctx, now); err != nil {
		// If we still had a cached key (cache expired, IdP transiently unreachable), use it.
		if ok {
			return key, nil
		}
		return nil, err
	}

	v.mu.RLock()
	defer v.mu.RUnlock()
	key, ok = v.lookupKeyLocked(kid)
	if !ok {
		return nil, ErrUnknownKeyID
	}
	return key, nil
}

func (v *OIDCVerifier) lookupKeyLocked(kid string) (crypto.PublicKey, bool) {
	if kid != "" {
		k, ok := v.keys[kid]
		return k, ok
	}
	if len(v.keys) == 1 {
		for _, k := range v.keys {
			return k, true
		}
	}
	return nil, false
}

func (v *OIDCVerifier) refreshJWKS(ctx context.Context, now time.Time) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Double-check cooldown under write lock.
	if !v.lastRefresh.IsZero() && now.Sub(v.lastRefresh) < v.refreshCooldown && len(v.keys) > 0 && now.Sub(v.fetchedAt) < v.cacheTTL {
		return nil
	}
	v.lastRefresh = now

	jwksURL := v.jwksURL
	if jwksURL == "" {
		discovered, err := v.discoverJWKSURL(ctx)
		if err != nil {
			return err
		}
		jwksURL = discovered
		v.jwksURL = discovered
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return err
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch jwks: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read jwks: %w", err)
	}

	var doc struct {
		Keys []jwkKey `json:"keys"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return fmt.Errorf("decode jwks: %w", err)
	}

	parsed := make(map[string]crypto.PublicKey, len(doc.Keys))
	for _, k := range doc.Keys {
		if k.Use != "" && k.Use != "sig" {
			continue
		}
		pub, err := k.toPublicKey()
		if err != nil {
			continue
		}
		id := k.Kid
		if id == "" {
			id = fmt.Sprintf("default-%d", len(parsed))
		}
		parsed[id] = pub
	}
	if len(parsed) == 0 {
		return errors.New("jwks contained no usable signing keys")
	}
	v.keys = parsed
	v.fetchedAt = now
	return nil
}

func (v *OIDCVerifier) discoverJWKSURL(ctx context.Context) (string, error) {
	url := v.issuer + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("oidc discovery: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("oidc discovery: status %d", resp.StatusCode)
	}
	var doc struct {
		JWKSURI string `json:"jwks_uri"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 256<<10)).Decode(&doc); err != nil {
		return "", fmt.Errorf("decode oidc discovery: %w", err)
	}
	if strings.TrimSpace(doc.JWKSURI) == "" {
		return "", errors.New("oidc discovery missing jwks_uri")
	}
	return strings.TrimSpace(doc.JWKSURI), nil
}

type jwkKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	// RSA fields
	N string `json:"n"`
	E string `json:"e"`
	// EC fields
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

func (k jwkKey) toPublicKey() (crypto.PublicKey, error) {
	switch k.Kty {
	case "RSA":
		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil || len(nBytes) == 0 {
			return nil, errors.New("invalid rsa modulus")
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil || len(eBytes) == 0 {
			return nil, errors.New("invalid rsa exponent")
		}
		e := int(new(big.Int).SetBytes(eBytes).Int64())
		if e < 3 {
			return nil, errors.New("invalid rsa exponent value")
		}
		return &rsa.PublicKey{
			N: new(big.Int).SetBytes(nBytes),
			E: e,
		}, nil
	case "EC":
		if k.Crv != "P-256" {
			return nil, fmt.Errorf("unsupported ec curve %q", k.Crv)
		}
		xBytes, err := base64.RawURLEncoding.DecodeString(k.X)
		if err != nil || len(xBytes) == 0 {
			return nil, errors.New("invalid ec x")
		}
		yBytes, err := base64.RawURLEncoding.DecodeString(k.Y)
		if err != nil || len(yBytes) == 0 {
			return nil, errors.New("invalid ec y")
		}
		x := new(big.Int).SetBytes(xBytes)
		y := new(big.Int).SetBytes(yBytes)
		curve := elliptic.P256()
		if !curve.IsOnCurve(x, y) {
			return nil, errors.New("ec key not on P-256 curve")
		}
		return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}, nil
	default:
		return nil, fmt.Errorf("unsupported jwk kty %q", k.Kty)
	}
}

func jsonString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return strings.TrimSpace(s)
	}
	return ""
}

func jsonInt64(raw json.RawMessage) (int64, bool) {
	if len(raw) == 0 {
		return 0, false
	}
	var n float64
	if json.Unmarshal(raw, &n) == nil {
		return int64(n), true
	}
	return 0, false
}

func jsonStringSlice(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var out []string
	if json.Unmarshal(raw, &out) == nil {
		return out
	}
	var single string
	if json.Unmarshal(raw, &single) == nil && strings.TrimSpace(single) != "" {
		return []string{strings.TrimSpace(single)}
	}
	return nil
}

func audContains(raw json.RawMessage, want string) bool {
	for _, a := range jsonStringSlice(raw) {
		if a == want {
			return true
		}
	}
	return false
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func containsStr(list []string, target string) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}
