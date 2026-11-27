// Package platform wires the pgx connection pool, PostgreSQL RLS scope
// boundary, exact fixed-point decimals, field-level encryption, and SQLSTATE classification.
package platform

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SystemOrgID is the global reference organization UUID (00000000-0000-0000-0000-000000000000).
// Reference data seeded under SystemOrgID is read-only for tenants.
var SystemOrgID = uuid.Nil

var defaultDB atomic.Pointer[DB]

// DefaultDB returns the most recently initialized process-wide DB pool (if any).
func DefaultDB() *DB {
	return defaultDB.Load()
}

// ScopeWithSystemOrg appends SystemOrgID to a non-empty tenant scope so global
// reference rows remain visible under RLS while empty scopes still fail closed.
func ScopeWithSystemOrg(scope []uuid.UUID) []uuid.UUID {
	if len(scope) == 0 {
		return nil
	}
	if slices.Contains(scope, SystemOrgID) {
		return slices.Clone(scope)
	}
	out := make([]uuid.UUID, 0, len(scope)+1)
	out = append(out, scope...)
	out = append(out, SystemOrgID)
	return out
}

// TenantOnlyScope strips SystemOrgID (00000000-0000-0000-0000-000000000000)
// from an org scope slice so tenant mutations cannot target global system rows.
func TenantOnlyScope(scope []uuid.UUID) []uuid.UUID {
	if len(scope) == 0 {
		return nil
	}
	out := make([]uuid.UUID, 0, len(scope))
	for _, u := range scope {
		if u != SystemOrgID {
			out = append(out, u)
		}
	}
	return out
}

// DB owns the connection pool. The pool MUST authenticate as a non-owner role
// (erp_app_rw / erp_app_ro) so PostgreSQL Row-Level Security is enforced.
type DB struct{ Pool *pgxpool.Pool }

// New opens and verifies a pool using env-tunable connection limits.
// Default MaxConns=5 ensures 30 microservices (150 total) fit safely within Postgres max_connections=250..300.
func New(ctx context.Context, dsn string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	if err := requireTLSInProduction(cfg); err != nil {
		return nil, err
	}
	cfg.MaxConns = envInt32("DATABASE_MAX_OPEN_CONNS", 5)
	cfg.MinConns = envInt32("DATABASE_MIN_CONNS", 1)
	cfg.MaxConnLifetime = envDuration("DATABASE_CONN_MAX_LIFETIME", time.Hour)
	cfg.MaxConnIdleTime = envDuration("DATABASE_CONN_MAX_IDLE_TIME", 5*time.Minute)
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	db := &DB{Pool: pool}
	defaultDB.Store(db)
	return db, nil
}

// requireTLSInProduction makes a production service refuse to start on a plaintext
// database connection. The localhost sslmode=disable fallback baked into each
// service main.go is therefore only usable outside APP_ENV=production.
func requireTLSInProduction(cfg *pgxpool.Config) error {
	if !strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production") {
		return nil
	}
	if cfg.ConnConfig.TLSConfig == nil {
		return errors.New("refusing plaintext database connection in production: set DATABASE_URL with sslmode=require or stricter")
	}
	for _, fb := range cfg.ConnConfig.Fallbacks {
		if fb.TLSConfig == nil {
			return errors.New("refusing database sslmode that allows plaintext fallback in production (use sslmode=require, verify-ca or verify-full)")
		}
	}
	return nil
}

func envInt32(key string, fallback int32) int32 {
	if raw := strings.TrimSpace(os.Getenv(key)); raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 32); err == nil && n > 0 {
			return int32(n)
		}
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if raw := strings.TrimSpace(os.Getenv(key)); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			return d
		}
	}
	return fallback
}

// Close releases the pool.
func (d *DB) Close() {
	if d != nil && d.Pool != nil {
		d.Pool.Close()
	}
}

// Ping verifies pool connectivity for /healthz and /readyz probes.
func (d *DB) Ping(ctx context.Context) error {
	if d == nil || d.Pool == nil {
		return errors.New("database pool not initialized")
	}
	return d.Pool.Ping(ctx)
}

func scopeLiteral(scope []uuid.UUID) string {
	parts := make([]string, len(scope))
	for i, u := range scope {
		parts[i] = u.String()
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// WithOrgScope runs fn inside a transaction with transaction-local PostgreSQL RLS
// scope set via set_config('app.org_scope', ..., true). Rejects empty scopes and
// performs panic-safe rollback unless Commit succeeds.
func (d *DB) WithOrgScope(ctx context.Context, scope []uuid.UUID, fn func(pgx.Tx) error) (err error) {
	if len(scope) == 0 {
		return errors.New("org scope must not be empty")
	}
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.org_scope', $1, true)", scopeLiteral(scope)); err != nil {
		return fmt.Errorf("set org scope: %w", err)
	}
	if err = fn(tx); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	committed = true
	return nil
}

// WithOrgReadScope runs fn inside a transaction with SystemOrgID included in app.org_scope
// (for non-empty caller scopes) so global reference rows remain visible on read paths.
func (d *DB) WithOrgReadScope(ctx context.Context, scope []uuid.UUID, fn func(pgx.Tx) error) error {
	if len(scope) == 0 {
		return errors.New("org scope must not be empty")
	}
	return d.WithOrgScope(ctx, ScopeWithSystemOrg(scope), fn)
}

// WithTenantWriteScope runs fn inside a transaction with SystemOrgID stripped from app.org_scope,
// rejecting requests whose scope contains no active tenant organization.
func (d *DB) WithTenantWriteScope(ctx context.Context, scope []uuid.UUID, fn func(pgx.Tx) error) error {
	ts := TenantOnlyScope(scope)
	if len(ts) == 0 {
		return errors.New("active tenant organization required (system org is read-only)")
	}
	return d.WithOrgScope(ctx, ts, fn)
}

type PGErrorKind int

const (
	PGUnknown PGErrorKind = iota
	PGNotFound
	PGUnique
	PGForeignKey
	PGCheck
	PGRLS
	PGInvalidInput
)

// ClassifyPGError maps PostgreSQL SQLSTATE codes cleanly without brittle string matching.
func ClassifyPGError(err error) PGErrorKind {
	if err == nil {
		return PGUnknown
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return PGNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return PGUnique
		case "23503":
			return PGForeignKey
		case "23514":
			return PGCheck
		case "42501":
			return PGRLS
		case "22P02", "22003":
			return PGInvalidInput
		}
	}
	return PGUnknown
}
