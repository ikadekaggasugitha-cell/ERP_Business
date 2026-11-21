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
	"strconv"
	"strings"
)

const defaultDevPIIKeyHex = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func decodeAES256Key(raw string, isProd bool) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if isProd && strings.EqualFold(raw, defaultDevPIIKeyHex) {
		return nil, errors.New("PII_ENCRYPTION_KEY must not use default dev key in production")
	}
	if len(raw) == 64 {
		if b, err := hex.DecodeString(raw); err == nil && len(b) == 32 {
			return b, nil
		}
	}
	if b, err := base64.StdEncoding.DecodeString(raw); err == nil && len(b) == 32 {
		return b, nil
	}
	if len(raw) == 32 {
		return []byte(raw), nil
	}
	if isProd {
		return nil, errors.New("PII_ENCRYPTION_KEY in production must be a 32-byte key (64-char hex, 44-char base64, or 32-byte raw)")
	}
	sum := sha256.Sum256([]byte(raw))
	return sum[:], nil
}

// LoadPIIKey reads a 32-byte AES-256 key from PII_ENCRYPTION_KEY (64-char hex, base64, or 32-byte raw).
// Uses an explicit dev key only when PII_ENCRYPTION_KEY is unset and APP_ENV != production.
func LoadPIIKey() ([]byte, error) {
	isProd := strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production")
	raw := strings.TrimSpace(os.Getenv("PII_ENCRYPTION_KEY"))
	if raw == "" {
		if isProd {
			return nil, errors.New("PII_ENCRYPTION_KEY is required in production")
		}
		log.Println("WARN: PII_ENCRYPTION_KEY unset; using ephemeral dev AES-256 key")
		b, _ := hex.DecodeString(defaultDevPIIKeyHex)
		return b, nil
	}
	return decodeAES256Key(raw, isProd)
}

// PIIKeyring supports versioned AES-256-GCM keys (PII_ENCRYPTION_KEY_V1, V2, ...) for zero-downtime key rotation.
type PIIKeyring struct {
	CurrentVersion int32
	Keys           map[int32][]byte
}

// LoadPIIKeyring loads the active key and any versioned keys (PII_ENCRYPTION_KEY_V1..V9).
func LoadPIIKeyring() (*PIIKeyring, error) {
	isProd := strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production")
	baseKey, err := LoadPIIKey()
	if err != nil {
		return nil, err
	}
	curVer := int32(1)
	if vRaw := strings.TrimSpace(os.Getenv("PII_ENCRYPTION_KEY_CURRENT_VERSION")); vRaw != "" {
		if n, err := strconv.ParseInt(vRaw, 10, 32); err == nil && n > 0 {
			curVer = int32(n)
		}
	}
	kr := &PIIKeyring{
		CurrentVersion: curVer,
		Keys:           map[int32][]byte{1: baseKey, curVer: baseKey},
	}
	for v := int32(1); v <= 9; v++ {
		envName := "PII_ENCRYPTION_KEY_V" + strconv.Itoa(int(v))
		if raw := strings.TrimSpace(os.Getenv(envName)); raw != "" {
			k, err := decodeAES256Key(raw, isProd)
			if err != nil {
				return nil, err
			}
			kr.Keys[v] = k
		}
	}
	return kr, nil
}

// KeyFor returns the key for version (falling back to CurrentVersion if version <= 0).
func (kr *PIIKeyring) KeyFor(version int32) ([]byte, error) {
	if kr == nil || len(kr.Keys) == 0 {
		return nil, errors.New("pii keyring not initialized")
	}
	if version <= 0 {
		version = kr.CurrentVersion
	}
	if k, ok := kr.Keys[version]; ok && len(k) == 32 {
		return k, nil
	}
	if k, ok := kr.Keys[kr.CurrentVersion]; ok && len(k) == 32 {
		return k, nil
	}
	return nil, errors.New("pii encryption key version not found")
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

