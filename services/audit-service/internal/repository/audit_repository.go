// Package repository wraps sqlc-generated auditdb queries with the
// PostgreSQL RLS scope boundary (platform.DB.WithOrgScope).
package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/audit-service/internal/auditdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

// AuditRepository provides RLS-enforced read and append operations over audit_db.
type AuditRepository struct {
	db *platform.DB
	q  *auditdb.Queries
}

// NewAuditRepository constructs an AuditRepository bound to the given pool.
func NewAuditRepository(db *platform.DB) *AuditRepository {
	return &AuditRepository{db: db, q: auditdb.New(db.Pool)}
}

// Get retrieves a single audit_log entry by composite PK (id, occurred_at).
func (r *AuditRepository) Get(ctx context.Context, scope []uuid.UUID, id uuid.UUID, occurredAt pgtype.Timestamptz) (auditdb.AuditAuditLog, error) {
	var out auditdb.AuditAuditLog
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var err error
		out, err = r.q.WithTx(tx).GetAuditLog(ctx, auditdb.GetAuditLogParams{ID: id, OccurredAt: occurredAt})
		return err
	})
	return out, err
}

// GetByID retrieves the latest audit_log entry matching id inside the caller's scope.
func (r *AuditRepository) GetByID(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (auditdb.AuditAuditLog, error) {
	var out auditdb.AuditAuditLog
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var err error
		out, err = r.q.WithTx(tx).GetAuditLogByID(ctx, id)
		return err
	})
	return out, err
}

// List returns recent audit entries within the caller's RLS scope.
func (r *AuditRepository) List(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]auditdb.ListAuditLogsRow, error) {
	var out []auditdb.ListAuditLogsRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var err error
		out, err = r.q.WithTx(tx).ListAuditLogs(ctx, auditdb.ListAuditLogsParams{Limit: limit, Offset: offset})
		return err
	})
	return out, err
}

// ListFilter defines optional predicates and pagination for listing audit logs.
type ListFilter struct {
	EntityType    string
	Action        string
	From          pgtype.Timestamptz
	To            pgtype.Timestamptz
	EntityID      *uuid.UUID
	CorrelationID *uuid.UUID
	Limit         int32
	Offset        int32
}

// ListFiltered returns paginated audit entries plus the exact total count inside one RLS tx.
func (r *AuditRepository) ListFiltered(ctx context.Context, scope []uuid.UUID, filter ListFilter) ([]auditdb.ListAuditLogsFilteredRow, int64, error) {
	var items []auditdb.ListAuditLogsFilteredRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var err error
		total, err = q.CountAuditLogsFiltered(ctx, auditdb.CountAuditLogsFilteredParams{
			EntityType:    filter.EntityType,
			Action:        filter.Action,
			FromTs:        filter.From,
			ToTs:          filter.To,
			EntityID:      filter.EntityID,
			CorrelationID: filter.CorrelationID,
		})
		if err != nil {
			return err
		}
		items, err = q.ListAuditLogsFiltered(ctx, auditdb.ListAuditLogsFilteredParams{
			Limit:         filter.Limit,
			Offset:        filter.Offset,
			EntityType:    filter.EntityType,
			Action:        filter.Action,
			FromTs:        filter.From,
			ToTs:          filter.To,
			EntityID:      filter.EntityID,
			CorrelationID: filter.CorrelationID,
		})
		return err
	})
	return items, total, err
}

// ListByEntity returns recent audit entries for one org and entity_type.
func (r *AuditRepository) ListByEntity(ctx context.Context, scope []uuid.UUID, orgID uuid.UUID, entityType string, limit, offset int32) ([]auditdb.ListAuditLogsByEntityRow, error) {
	var out []auditdb.ListAuditLogsByEntityRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var err error
		out, err = r.q.WithTx(tx).ListAuditLogsByEntity(ctx, auditdb.ListAuditLogsByEntityParams{
			OrganizationID: orgID,
			EntityType:     entityType,
			Limit:          limit,
			Offset:         offset,
		})
		return err
	})
	return out, err
}

// Record appends one entry to the immutable audit_log partition table.
func (r *AuditRepository) Record(ctx context.Context, scope []uuid.UUID, arg auditdb.RecordAuditLogParams) (auditdb.RecordAuditLogRow, error) {
	var out auditdb.RecordAuditLogRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var err error
		out, err = r.q.WithTx(tx).RecordAuditLog(ctx, arg)
		return err
	})
	return out, err
}

// RecordFromEvent atomically de-duplicates by event_id and appends an audit_log row.
// Returns (true, nil) if already processed, (false, nil) on fresh append.
func (r *AuditRepository) RecordFromEvent(ctx context.Context, scope []uuid.UUID, proc auditdb.MarkEventProcessedParams, entry auditdb.RecordAuditLogParams) (bool, error) {
	alreadyProcessed := false
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		n, err := q.MarkEventProcessed(ctx, proc)
		if err != nil {
			return err
		}
		if n == 0 {
			alreadyProcessed = true
			return nil
		}
		_, err = q.RecordAuditLog(ctx, entry)
		return err
	})
	return alreadyProcessed, err
}

// MarkEventProcessed records event_id in the technical processed_event ledger.
func (r *AuditRepository) MarkEventProcessed(ctx context.Context, arg auditdb.MarkEventProcessedParams) (int64, error) {
	return r.q.MarkEventProcessed(ctx, arg)
}
