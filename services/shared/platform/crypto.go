package platform

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"os"
	"strings"
)

const defaultDevPIIKeyHex = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

// LoadPIIKey reads a 32-byte AES-256 key from PII_ENCRYPTION_KEY (64-char hex or base64).
// Uses an explicit dev key only when PII_ENCRYPTION_KEY is unset and APP_ENV != production.
func LoadPIIKey() ([]byte, error) {
	raw := strings.TrimSpace(os.Getenv("PII_ENCRYPTION_KEY"))
	if raw == "" {
		if strings.EqualFold(os.Getenv("APP_ENV"), "production") {
			return nil, errors.New("PII_ENCRYPTION_KEY is required in production")
		}
		log.Println("WARN: PII_ENCRYPTION_KEY unset; using ephemeral dev AES-256 key")
		b, _ := hex.DecodeString(defaultDevPIIKeyHex)
		return b, nil
	}
	if len(raw) == 64 {
		if b, err := hex.DecodeString(raw); err == nil && len(b) == 32 {
			return b, nil
		}
	}
	if b, err := base64.StdEncoding.DecodeString(raw); err == nil && len(b) == 32 {
		return b, nil
	}
	sum := sha256.Sum256([]byte(raw))
	return sum[:], nil
}

// EncryptField encrypts plaintext using AES-256-GCM and encodes as base64.
func EncryptField(key []byte, plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// DecryptField decrypts a base64 AES-256-GCM field payload.
func DecryptField(key []byte, encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce, ct := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

// HashLookup computes a keyed HMAC-SHA256 hex digest for blind-index PII lookups (e.g. national_id_hash).
func HashLookup(key []byte, value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(trimmed))
	return hex.EncodeToString(mac.Sum(nil))
}

// MaskNationalID leaves only the last 4 digits readable for response DTOs.
func MaskNationalID(raw string) string {
	s := strings.TrimSpace(raw)
	if len(s) <= 4 {
		return strings.Repeat("*", len(s))
	}
	return strings.Repeat("*", len(s)-4) + s[len(s)-4:]
}

