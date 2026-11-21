package httpx

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

// IdemStore provides payload-hash-validated idempotency storage backed by PostgreSQL
// (public.idempotency_key) when a DB pool is available, with an in-process cache
// and deterministic oldest-first eviction that never evicts in-flight reservations.
type IdemStore struct {
	mu  sync.Mutex
	m   map[string]*idemEntry
	db  *platform.DB
	ttl time.Duration
	max int
}

type idemEntry struct {
	done    bool
	reqHash string
	status  int
	etag    string
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

func NewIdemStore(dbs ...*platform.DB) *IdemStore {
	var db *platform.DB
	if len(dbs) > 0 && dbs[0] != nil {
		db = dbs[0]
	}
	return &IdemStore{m: make(map[string]*idemEntry), db: db, ttl: time.Hour, max: 10000}
}

func (s *IdemStore) activeDB() *platform.DB {
	if s.db != nil && s.db.Pool != nil {
		return s.db
	}
	if d := platform.DefaultDB(); d != nil && d.Pool != nil {
		return d
	}
	return nil
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

func (s *IdemStore) loadFromDB(scopeKey, key string) (*idemEntry, bool) {
	db := s.activeDB()
	if db == nil {
		return nil, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	var e idemEntry
	err := db.Pool.QueryRow(ctx, `
		SELECT req_hash, done, status, etag, body, created_at
		FROM public.idempotency_key
		WHERE scope_key = $1 AND idem_key = $2
	`, scopeKey, key).Scan(&e.reqHash, &e.done, &e.status, &e.etag, &e.body, &e.at)
	if err != nil {
		return nil, false
	}
	if time.Since(e.at) > s.effectiveTTL() {
		_, _ = db.Pool.Exec(ctx, `DELETE FROM public.idempotency_key WHERE scope_key = $1 AND idem_key = $2`, scopeKey, key)
		return nil, false
	}
	return &e, true
}

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
	e, ok := s.m[s.k(scopeKey, key)]
	if (!ok || time.Since(e.at) > s.effectiveTTL()) && s.activeDB() != nil {
		if dbEntry, dbOK := s.loadFromDB(scopeKey, key); dbOK {
			e = dbEntry
			ok = true
			s.m[s.k(scopeKey, key)] = e
		}
	}
	s.mu.Unlock()
	if !ok || !e.done || time.Since(e.at) > s.effectiveTTL() {
		return false
	}
	if reqHash != "" && e.reqHash != "" && e.reqHash != reqHash {
		cid := w.Header().Get("X-Correlation-Id")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusConflict)
		payload, _ := json.Marshal(map[string]any{
			"error": map[string]any{
				"code":          "IDEMPOTENCY_KEY_REUSED",
				"message":       "Idempotency-Key reused with different request payload",
				"correlationId": cid,
			},
		})
		_, _ = w.Write(payload)
		return true
	}
	if e.etag != "" {
		w.Header().Set("ETag", e.etag)
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
	e, ok := s.m[k]
	if (!ok || time.Since(e.at) > s.effectiveTTL()) && s.activeDB() != nil {
		if dbEntry, dbOK := s.loadFromDB(scopeKey, key); dbOK {
			e = dbEntry
			ok = true
			s.m[k] = e
		}
	}
	if ok && time.Since(e.at) <= s.effectiveTTL() {
		if reqHash != "" && e.reqHash != "" && e.reqHash != reqHash {
			return ReserveHashMismatch
		}
		if e.done {
			return ReserveCompleted
		}
		return ReserveInFlight
	}
	if db := s.activeDB(); db != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		_, _ = db.Pool.Exec(ctx, `
			INSERT INTO public.idempotency_key (scope_key, idem_key, req_hash, done, status, etag, created_at)
			VALUES ($1, $2, $3, FALSE, 0, '', now())
			ON CONFLICT (scope_key, idem_key) DO UPDATE
			SET req_hash = EXCLUDED.req_hash, done = FALSE, status = 0, etag = '', body = NULL, created_at = now()
			WHERE public.idempotency_key.created_at < now() - interval '1 hour'
		`, scopeKey, key, reqHash)
	}
	s.evictLocked()
	s.m[k] = &idemEntry{reqHash: reqHash, at: time.Now()}
	return ReserveAcquired
}

// Save records the final response and marks the key completed.
func (s *IdemStore) Save(scopeKey, key string, status int, body []byte) {
	s.SaveWithHashAndETag(scopeKey, key, "", "", status, body)
}

func (s *IdemStore) SaveWithHash(scopeKey, key, reqHash string, status int, body []byte) {
	s.SaveWithHashAndETag(scopeKey, key, reqHash, "", status, body)
}

func (s *IdemStore) SaveWithHashAndETag(scopeKey, key, reqHash, etag string, status int, body []byte) {
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
	s.m[k] = &idemEntry{done: true, reqHash: reqHash, status: status, etag: etag, body: body, at: time.Now()}
	if db := s.activeDB(); db != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		_, _ = db.Pool.Exec(ctx, `
			INSERT INTO public.idempotency_key (scope_key, idem_key, req_hash, done, status, etag, body, created_at)
			VALUES ($1, $2, $3, TRUE, $4, $5, $6, now())
			ON CONFLICT (scope_key, idem_key) DO UPDATE
			SET req_hash = CASE WHEN EXCLUDED.req_hash <> '' THEN EXCLUDED.req_hash ELSE public.idempotency_key.req_hash END,
			    done = TRUE,
			    status = EXCLUDED.status,
			    etag = EXCLUDED.etag,
			    body = EXCLUDED.body,
			    created_at = now()
		`, scopeKey, key, reqHash, status, etag, body)
	}
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
	if db := s.activeDB(); db != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		_, _ = db.Pool.Exec(ctx, `DELETE FROM public.idempotency_key WHERE scope_key = $1 AND idem_key = $2 AND done = FALSE`, scopeKey, key)
	}
}

// evictLocked drops expired entries first; if still at cap, drops the oldest
// completed entries (FIFO by timestamp) so in-flight concurrent guards are never evicted.
func (s *IdemStore) evictLocked() {
	if len(s.m) < s.max {
		return
	}
	ttl := s.effectiveTTL()
	for k, e := range s.m {
		if e.done && time.Since(e.at) > ttl {
			delete(s.m, k)
		}
	}
	for len(s.m) >= s.max {
		var oldestKey string
		var oldestTime time.Time
		found := false
		for k, e := range s.m {
			if !e.done {
				continue
			}
			if !found || e.at.Before(oldestTime) {
				oldestKey = k
				oldestTime = e.at
				found = true
			}
		}
		if !found {
			break
		}
		delete(s.m, oldestKey)
	}
}
