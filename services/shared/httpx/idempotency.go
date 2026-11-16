package httpx

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

// IdemStore is a dev-grade in-process idempotency cache with payload-hash
// conflict protection and bounded eviction that never evicts in-flight reservations.
type IdemStore struct {
	mu  sync.Mutex
	m   map[string]*idemEntry
	ttl time.Duration
	max int
}

type idemEntry struct {
	done    bool
	reqHash string
	status  int
	body    []byte
	at      time.Time
}

type ReserveResult int

const (
	ReserveAcquired ReserveResult = iota
	ReserveInFlight
	ReserveCompleted
	ReserveHashMismatch
)

func NewIdemStore() *IdemStore {
	return &IdemStore{m: make(map[string]*idemEntry), ttl: time.Hour, max: 10000}
}

func (s *IdemStore) effectiveTTL() time.Duration {
	if s.ttl <= 0 {
		return time.Hour
	}
	return s.ttl
}

func HashBody(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (s *IdemStore) k(scopeKey, key string) string { return scopeKey + "|" + key }

// Replay writes a cached COMPLETED response for (scope,key) if present.
func (s *IdemStore) Replay(w http.ResponseWriter, scopeKey, key string) bool {
	return s.ReplayWithHash(w, scopeKey, key, "")
}

// ReplayWithHash replays a cached COMPLETED response if present. When reqHash is
// provided and differs from the original request hash, it writes 409 and returns true.
func (s *IdemStore) ReplayWithHash(w http.ResponseWriter, scopeKey, key, reqHash string) bool {
	if key == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.m[s.k(scopeKey, key)]
	if !ok || !e.done || time.Since(e.at) > s.effectiveTTL() {
		return false
	}
	if reqHash != "" && e.reqHash != "" && e.reqHash != reqHash {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":{"code":"IDEMPOTENCY_KEY_REUSED","message":"Idempotency-Key reused with different request payload"}}`))
		return true
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Idempotency-Replayed", "true")
	w.WriteHeader(e.status)
	_, _ = w.Write(e.body)
	return true
}

// Reserve marks a key in-flight. Returns false if already in-flight or completed.
func (s *IdemStore) Reserve(scopeKey, key string) bool {
	return s.ReserveWithHash(scopeKey, key, "") == ReserveAcquired
}

// ReserveWithHash marks a key in-flight and stores its request payload hash.
func (s *IdemStore) ReserveWithHash(scopeKey, key, reqHash string) ReserveResult {
	if key == "" {
		return ReserveAcquired
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	k := s.k(scopeKey, key)
	if e, ok := s.m[k]; ok && time.Since(e.at) <= s.effectiveTTL() {
		if reqHash != "" && e.reqHash != "" && e.reqHash != reqHash {
			return ReserveHashMismatch
		}
		if e.done {
			return ReserveCompleted
		}
		return ReserveInFlight
	}
	s.evictLocked()
	s.m[k] = &idemEntry{reqHash: reqHash, at: time.Now()}
	return ReserveAcquired
}

// Save records the final response and marks the key completed.
func (s *IdemStore) Save(scopeKey, key string, status int, body []byte) {
	s.SaveWithHash(scopeKey, key, "", status, body)
}

func (s *IdemStore) SaveWithHash(scopeKey, key, reqHash string, status int, body []byte) {
	if key == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	k := s.k(scopeKey, key)
	if reqHash == "" {
		if existing, ok := s.m[k]; ok {
			reqHash = existing.reqHash
		}
	}
	s.m[k] = &idemEntry{done: true, reqHash: reqHash, status: status, body: body, at: time.Now()}
}

// Unreserve drops an in-flight (not-yet-completed) reservation so a FAILED
// request can be retried. No-op once Save has marked the key done.
func (s *IdemStore) Unreserve(scopeKey, key string) {
	if key == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	k := s.k(scopeKey, key)
	if e, ok := s.m[k]; ok && !e.done {
		delete(s.m, k)
	}
}

// evictLocked drops expired entries first; if still at cap, drops only completed
// entries so in-flight concurrent guards are never evicted.
func (s *IdemStore) evictLocked() {
	if len(s.m) < s.max {
		return
	}
	for k, e := range s.m {
		if e.done && time.Since(e.at) > s.effectiveTTL() {
			delete(s.m, k)
		}
	}
	for k, e := range s.m {
		if len(s.m) < s.max {
			break
		}
		if !e.done {
			continue
		}
		delete(s.m, k)
	}
}
