// Package platform wires the pgx connection pool, PostgreSQL RLS scope
// boundary, exact fixed-point decimals, field-level encryption, and SQLSTATE classification.
package platform

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB owns the connection pool. The pool MUST authenticate as a non-owner role
// (erp_app_rw / erp_app_ro) so PostgreSQL Row-Level Security is enforced.
type DB struct{ Pool *pgxpool.Pool }

// New opens and verifies a pool using env-tunable connection limits.
func New(ctx context.Context, dsn string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	cfg.MaxConns = envInt32("DATABASE_MAX_OPEN_CONNS", 15)
	cfg.MinConns = envInt32("DATABASE_MIN_CONNS", 2)
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
	return &DB{Pool: pool}, nil
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
