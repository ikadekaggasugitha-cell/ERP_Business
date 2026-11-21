// Package repository is the Reporting (CQRS read model) data-access layer.
package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/reporting-service/internal/reportingdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

// GLAccountDelta carries one posted GL journal line delta to accumulate into rpt.gl_balance_snapshot.
type GLAccountDelta struct {
	OrganizationID uuid.UUID
	FiscalPeriodID uuid.UUID
	AccountID      *uuid.UUID
	AccountCode    string
	ClosingBalance pgtype.Numeric
}

type SnapshotRepository struct {
	db *platform.DB
	q  *reportingdb.Queries
}

func NewSnapshotRepository(db *platform.DB) *SnapshotRepository {
	return &SnapshotRepository{db: db, q: reportingdb.New(db.Pool)}
}

func (r *SnapshotRepository) ConsumeGL(
	ctx context.Context, scope []uuid.UUID,
	mark reportingdb.MarkEventProcessedParams,
	accountDeltas []GLAccountDelta,
) (applied bool, err error) {
	err = r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		n, e := q.MarkEventProcessed(ctx, mark)
		if e != nil || n == 0 {
			return e
		}
		for _, delta := range accountDeltas {
			code := strings.TrimSpace(delta.AccountCode)
			if code == "" && delta.AccountID != nil {
				cached, ce := q.GetCOACacheCode(ctx, reportingdb.GetCOACacheCodeParams{
					OrganizationID: delta.OrganizationID,
					AccountID:      *delta.AccountID,
				})
				if ce == nil && strings.TrimSpace(cached) != "" {
					code = strings.TrimSpace(cached)
				} else if ce != nil && !errors.Is(ce, pgx.ErrNoRows) {
					return ce
				} else {
					code = "ACCT:" + delta.AccountID.String()
				}
			}
			if code == "" {
				continue
			}
			if len(code) > 50 {
				code = code[:50]
			}
			if _, e = q.AccumulateGLBalanceSnapshot(ctx, reportingdb.AccumulateGLBalanceSnapshotParams{
				OrganizationID: delta.OrganizationID,
				FiscalPeriodID: delta.FiscalPeriodID,
				AccountCode:    code,
				ClosingBalance: delta.ClosingBalance,
			}); e != nil {
				return e
			}
		}
		applied = true
		return nil
	})
	return applied, err
}

func (r *SnapshotRepository) ConsumeCOAAccount(
	ctx context.Context, scope []uuid.UUID,
	mark reportingdb.MarkEventProcessedParams,
	up reportingdb.UpsertCOACacheParams,
) (applied bool, err error) {
	err = r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		n, e := q.MarkEventProcessed(ctx, mark)
		if e != nil || n == 0 {
			return e
		}
		if e = q.UpsertCOACache(ctx, up); e != nil {
			return e
		}
		applied = true
		return nil
	})
	return applied, err
}

func (r *SnapshotRepository) ConsumeSalesSummary(
	ctx context.Context, scope []uuid.UUID,
	mark reportingdb.MarkEventProcessedParams,
	up reportingdb.UpsertSalesSummaryParams,
) (applied bool, err error) {
	err = r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		n, e := q.MarkEventProcessed(ctx, mark)
		if e != nil || n == 0 {
			return e
		}
		if _, e = q.UpsertSalesSummary(ctx, up); e != nil {
			return e
		}
		applied = true
		return nil
	})
	return applied, err
}

func (r *SnapshotRepository) ConsumeInventoryPosition(
	ctx context.Context, scope []uuid.UUID,
	mark reportingdb.MarkEventProcessedParams,
	up reportingdb.UpsertInventoryPositionParams,
) (applied bool, err error) {
	err = r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		n, e := q.MarkEventProcessed(ctx, mark)
		if e != nil || n == 0 {
			return e
		}
		if _, e = q.UpsertInventoryPosition(ctx, up); e != nil {
			return e
		}
		applied = true
		return nil
	})
	return applied, err
}

func (r *SnapshotRepository) ListGLWithCount(ctx context.Context, scope []uuid.UUID, arg reportingdb.ListGLBalanceSnapshotsParams) ([]reportingdb.ListGLBalanceSnapshotsRow, int64, error) {
	var out []reportingdb.ListGLBalanceSnapshotsRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		out, e = q.ListGLBalanceSnapshots(ctx, arg)
		if e != nil {
			return e
		}
		total, e = q.CountGLBalanceSnapshots(ctx)
		return e
	})
	return out, total, err
}

func (r *SnapshotRepository) ListSalesWithCount(ctx context.Context, scope []uuid.UUID, arg reportingdb.ListSalesSummariesParams) ([]reportingdb.RptSalesSummary, int64, error) {
	var out []reportingdb.RptSalesSummary
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		out, e = q.ListSalesSummaries(ctx, arg)
		if e != nil {
			return e
		}
		total, e = q.CountSalesSummaries(ctx)
		return e
	})
	return out, total, err
}

func (r *SnapshotRepository) ListInventoryWithCount(ctx context.Context, scope []uuid.UUID, arg reportingdb.ListInventoryPositionsParams) ([]reportingdb.RptInventoryPosition, int64, error) {
	var out []reportingdb.RptInventoryPosition
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		out, e = q.ListInventoryPositions(ctx, arg)
		if e != nil {
			return e
		}
		total, e = q.CountInventoryPositions(ctx)
		return e
	})
	return out, total, err
}
