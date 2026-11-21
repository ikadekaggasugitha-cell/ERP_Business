package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

type Claims struct {
	Subject     string   `json:"sub"`
	Username    string   `json:"username"`
	FullName    string   `json:"fullName"`
	Email       string   `json:"email"`
	OrgID       string   `json:"org_id"`
	OrgScope    []string `json:"org_scope"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	AdminTier   int      `json:"adminTier,omitempty"`
	TokenUse    string   `json:"use"` // "access" or "stream"
	IssuedAt    int64    `json:"iat"`
	ExpiresAt   int64    `json:"exp"`
}

func SignHS256(claims Claims, secret []byte) (string, error) {
	headerBytes, err := json.Marshal(map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	})
	if err != nil {
		return "", err
	}
	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	header := base64.RawURLEncoding.EncodeToString(headerBytes)
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	signingInput := header + "." + payload

	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(signingInput))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return signingInput + "." + sig, nil
}

func VerifyHS256(token string, secret []byte, now time.Time) (Claims, error) {
	var claims Claims
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return claims, ErrInvalidToken
	}

	var hdr struct {
		Alg string `json:"alg"`
	}
	hdrBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || json.Unmarshal(hdrBytes, &hdr) != nil || hdr.Alg != "HS256" {
		return claims, ErrInvalidToken
	}

	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(signingInput))
	expectedSig := mac.Sum(nil)
	gotSig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(expectedSig, gotSig) {
		return claims, ErrInvalidToken
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return claims, ErrInvalidToken
	}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return claims, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if claims.ExpiresAt <= now.Unix() {
		return claims, ErrExpiredToken
	}
	return claims, nil
}

