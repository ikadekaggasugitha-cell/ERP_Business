package auth

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
	ErrRefreshTokenReused  = errors.New("refresh token reuse detected; session family revoked")
)

type refreshRecord struct {
	Claims    Claims    `json:"claims"`
	FamilyID  string    `json:"familyId"`
	Used      bool      `json:"used"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// SessionStore manages JTI revocation, refresh token rotation, and optional
// distributed rate limiting. Uses an in-memory store by default and optionally
// syncs to Redis via lightweight stdlib RESP2 when REDIS_URL is configured.
type SessionStore struct {
	mu       sync.Mutex
	revoked  map[string]time.Time
	refresh  map[string]*refreshRecord
	families map[string][]string // familyID -> tokenHashes
	redis    *redisRESPClient
}

// NewSessionStore constructs a SessionStore; redisURL may be empty for in-memory mode.
func NewSessionStore(redisURL string) *SessionStore {
	s := &SessionStore{
		revoked:  make(map[string]time.Time),
		refresh:  make(map[string]*refreshRecord),
		families: make(map[string][]string),
	}
	if strings.TrimSpace(redisURL) != "" {
		if rc, err := newRedisRESPClient(redisURL); err == nil {
			s.redis = rc
		}
	}
	return s
}

// RevokeJTI marks a token's jti as revoked until ttl elapses.
func (s *SessionStore) RevokeJTI(ctx context.Context, jti string, ttl time.Duration) error {
	jti = strings.TrimSpace(jti)
	if jti == "" {
		return nil
	}
	if ttl <= 0 {
		ttl = 8 * time.Hour
	}
	if s.redis != nil {
		if err := s.redis.setEX(ctx, "erp:revoked:"+jti, "1", ttl); err == nil {
			return nil
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gcLocked(time.Now().UTC())
	s.revoked[jti] = time.Now().UTC().Add(ttl)
	return nil
}

// IsRevoked reports whether jti is currently in the revocation denylist.
func (s *SessionStore) IsRevoked(ctx context.Context, jti string) bool {
	jti = strings.TrimSpace(jti)
	if jti == "" {
		return false
	}
	if s.redis != nil {
		if val, ok, err := s.redis.get(ctx, "erp:revoked:"+jti); err == nil {
			if ok && val != "" {
				return true
			}
			// Check local fallback cache as well in case Redis had a transient outage during revoke.
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.revoked[jti]
	if !ok {
		return false
	}
	if time.Now().UTC().After(exp) {
		delete(s.revoked, jti)
		return false
	}
	return true
}

// IssueRefreshToken creates a new opaque refresh token bound to claims.
func (s *SessionStore) IssueRefreshToken(ctx context.Context, claims Claims, ttl time.Duration) (string, error) {
	return s.issueWithFamily(ctx, claims, uuid.NewString(), ttl)
}

func (s *SessionStore) issueWithFamily(ctx context.Context, claims Claims, familyID string, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		ttl = 7 * 24 * time.Hour
	}
	raw, err := generateOpaqueToken()
	if err != nil {
		return "", err
	}
	hash := hashToken(raw)
	rec := &refreshRecord{
		Claims:    claims,
		FamilyID:  familyID,
		Used:      false,
		ExpiresAt: time.Now().UTC().Add(ttl),
	}
	if s.redis != nil {
		if b, err := json.Marshal(rec); err == nil {
			if err := s.redis.setEX(ctx, "erp:refresh:"+hash, string(b), ttl); err == nil {
				return raw, nil
			}
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gcLocked(time.Now().UTC())
	s.refresh[hash] = rec
	s.families[familyID] = append(s.families[familyID], hash)
	return raw, nil
}

// RotateRefreshToken validates rawToken, marks it used, and issues a replacement token.
// Replaying an already-consumed refresh token revokes the entire token family.
func (s *SessionStore) RotateRefreshToken(ctx context.Context, rawToken string, ttl time.Duration) (Claims, string, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return Claims{}, "", ErrInvalidRefreshToken
	}
	hash := hashToken(rawToken)
	now := time.Now().UTC()

	if s.redis != nil {
		if val, ok, err := s.redis.get(ctx, "erp:refresh:"+hash); err == nil && ok {
			var rec refreshRecord
			if json.Unmarshal([]byte(val), &rec) == nil {
				if now.After(rec.ExpiresAt) {
					_ = s.redis.del(ctx, "erp:refresh:"+hash)
					return Claims{}, "", ErrInvalidRefreshToken
				}
				if rec.Used {
					_ = s.redis.del(ctx, "erp:refresh:"+hash)
					return Claims{}, "", ErrRefreshTokenReused
				}
				rec.Used = true
				if b, err := json.Marshal(&rec); err == nil {
					rem := time.Until(rec.ExpiresAt)
					if rem < time.Minute {
						rem = time.Minute
					}
					_ = s.redis.setEX(ctx, "erp:refresh:"+hash, string(b), rem)
				}
				nextToken, err := s.issueWithFamily(ctx, rec.Claims, rec.FamilyID, ttl)
				if err != nil {
					return Claims{}, "", err
				}
				return rec.Claims, nextToken, nil
			}
		}
	}

	s.mu.Lock()
	rec, ok := s.refresh[hash]
	if !ok || now.After(rec.ExpiresAt) {
		delete(s.refresh, hash)
		s.mu.Unlock()
		return Claims{}, "", ErrInvalidRefreshToken
	}
	if rec.Used {
		// Replay detected: revoke entire family.
		for _, h := range s.families[rec.FamilyID] {
			delete(s.refresh, h)
		}
		delete(s.families, rec.FamilyID)
		s.mu.Unlock()
		return Claims{}, "", ErrRefreshTokenReused
	}
	rec.Used = true
	claims := rec.Claims
	familyID := rec.FamilyID
	s.mu.Unlock()

	nextToken, err := s.issueWithFamily(ctx, claims, familyID, ttl)
	if err != nil {
		return Claims{}, "", err
	}
	return claims, nextToken, nil
}

// AllowRateLimit performs a fixed-window distributed rate limit check in Redis when configured.
// Returns (allowed, usedRedis). When usedRedis is false, callers fall back to local token bucket.
func (s *SessionStore) AllowRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, bool) {
	if s.redis == nil || limit <= 0 || window <= 0 {
		return true, false
	}
	bucket := time.Now().UTC().Unix() / int64(window.Seconds())
	rKey := fmt.Sprintf("erp:rl:%s:%d", key, bucket)
	count, err := s.redis.incrWithExpire(ctx, rKey, window)
	if err != nil {
		return true, false
	}
	return count <= int64(limit), true
}

func (s *SessionStore) gcLocked(now time.Time) {
	if len(s.revoked) > 10000 {
		for k, exp := range s.revoked {
			if now.After(exp) {
				delete(s.revoked, k)
			}
		}
	}
	if len(s.refresh) > 10000 {
		for k, rec := range s.refresh {
			if now.After(rec.ExpiresAt) {
				delete(s.refresh, k)
			}
		}
	}
}

func generateOpaqueToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// ---- Lightweight stdlib Redis RESP2 client ---------------------------------

type redisRESPClient struct {
	addr     string
	password string
	db       int
	timeout  time.Duration
}

func newRedisRESPClient(rawURL string) (*redisRESPClient, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, err
	}
	addr := u.Host
	if addr == "" {
		return nil, errors.New("redis url missing host")
	}
	if !strings.Contains(addr, ":") {
		addr += ":6379"
	}
	var pass string
	if u.User != nil {
		pass, _ = u.User.Password()
	}
	db := 0
	if p := strings.TrimPrefix(u.Path, "/"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n >= 0 {
			db = n
		}
	}
	return &redisRESPClient{
		addr:     addr,
		password: pass,
		db:       db,
		timeout:  500 * time.Millisecond,
	}, nil
}

func (c *redisRESPClient) do(ctx context.Context, args ...string) (string, bool, error) {
	d := net.Dialer{Timeout: c.timeout}
	conn, err := d.DialContext(ctx, "tcp", c.addr)
	if err != nil {
		return "", false, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(c.timeout))
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	if c.password != "" {
		if _, _, err := writeAndReadRESP(rw, "AUTH", c.password); err != nil {
			return "", false, err
		}
	}
	if c.db > 0 {
		if _, _, err := writeAndReadRESP(rw, "SELECT", strconv.Itoa(c.db)); err != nil {
			return "", false, err
		}
	}
	return writeAndReadRESP(rw, args...)
}

func (c *redisRESPClient) setEX(ctx context.Context, key, val string, ttl time.Duration) error {
	secs := int(ttl.Seconds())
	if secs < 1 {
		secs = 1
	}
	_, _, err := c.do(ctx, "SET", key, val, "EX", strconv.Itoa(secs))
	return err
}

func (c *redisRESPClient) get(ctx context.Context, key string) (string, bool, error) {
	return c.do(ctx, "GET", key)
}

func (c *redisRESPClient) del(ctx context.Context, key string) error {
	_, _, err := c.do(ctx, "DEL", key)
	return err
}

func (c *redisRESPClient) incrWithExpire(ctx context.Context, key string, window time.Duration) (int64, error) {
	val, _, err := c.do(ctx, "INCR", key)
	if err != nil {
		return 0, err
	}
	n, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, err
	}
	if n == 1 {
		secs := int(window.Seconds())
		if secs < 1 {
			secs = 1
		}
		_, _, _ = c.do(ctx, "EXPIRE", key, strconv.Itoa(secs))
	}
	return n, nil
}

func writeAndReadRESP(rw *bufio.ReadWriter, args ...string) (string, bool, error) {
	fmt.Fprintf(rw, "*%d\r\n", len(args))
	for _, a := range args {
		fmt.Fprintf(rw, "$%d\r\n%s\r\n", len(a), a)
	}
	if err := rw.Flush(); err != nil {
		return "", false, err
	}
	line, err := rw.ReadString('\n')
	if err != nil {
		return "", false, err
	}
	line = strings.TrimRight(line, "\r\n")
	if len(line) == 0 {
		return "", false, errors.New("empty redis response")
	}
	switch line[0] {
	case '+', ':':
		return line[1:], true, nil
	case '-':
		return "", false, errors.New(line[1:])
	case '$':
		n, err := strconv.Atoi(line[1:])
		if err != nil {
			return "", false, err
		}
		if n < 0 {
			return "", false, nil // Nil bulk string
		}
		buf := make([]byte, n+2)
		if _, err := io.ReadFull(rw, buf); err != nil {
			return "", false, err
		}
		return string(buf[:n]), true, nil
	default:
		return "", false, fmt.Errorf("unexpected resp prefix %q", line[0])
	}
}
