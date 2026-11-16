package auth

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSignAndVerifyHS256(t *testing.T) {
	secret := []byte("unit-test-secret")
	now := time.Unix(1700000000, 0).UTC()
	in := Claims{
		Subject:     "00000000-0000-0000-0000-000000000101",
		Username:    "admin.finance",
		OrgID:       "11111111-1111-1111-1111-111111111111",
		OrgScope:    []string{"11111111-1111-1111-1111-111111111111"},
		Roles:       []string{"SYSTEM_ADMIN"},
		Permissions: []string{"gl.read"},
		TokenUse:    "access",
		IssuedAt:    now.Unix(),
		ExpiresAt:   now.Add(time.Hour).Unix(),
	}

	tok, err := SignHS256(in, secret)
	if err != nil {
		t.Fatalf("SignHS256: %v", err)
	}

	got, err := VerifyHS256(tok, secret, now.Add(5*time.Minute))
	if err != nil {
		t.Fatalf("VerifyHS256 valid token: %v", err)
	}
	if got.Subject != in.Subject || got.OrgID != in.OrgID {
		t.Fatalf("claims mismatch: got %+v want %+v", got, in)
	}

	if _, err := VerifyHS256(tok, []byte("wrong-secret"), now); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for wrong secret, got %v", err)
	}

	if _, err := VerifyHS256(tok, secret, now.Add(2*time.Hour)); !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("expected ErrExpiredToken for expired token, got %v", err)
	}

	parts := strings.Split(tok, ".")
	badHdr := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	if _, err := VerifyHS256(badHdr+"."+parts[1]+"."+parts[2], secret, now); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for alg=none, got %v", err)
	}
}

