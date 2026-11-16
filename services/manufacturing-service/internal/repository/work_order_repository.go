// Package repository manages RLS-scoped manufacturing_db transactions for
// BOMs, work orders, and transactional outbox event emission.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/manufacturing-service/internal/manufacturingdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

// BOMDetail bundles a bill of materials header with its component lines.
type BOMDetail struct {
	Header manufacturingdb.GetBillOfMaterialRow
	Lines  []manufacturingdb.ListBomLinesRow
}

// WorkOrderRepository provides persistence for manufacturing-service.
type WorkOrderRepository struct {
	db *platform.DB
	q  *manufacturingdb.Queries
}

// NewWorkOrderRepository constructs a WorkOrderRepository backed by db.
func NewWorkOrderRepository(db *platform.DB) *WorkOrderRepository {
	return &WorkOrderRepository{db: db, q: manufacturingdb.New(db.Pool)}
}

// CreateBOMWithLines creates a BOM header and optional component lines in one transaction.
func (r *WorkOrderRepository) CreateBOMWithLines(ctx context.Context, scope []uuid.UUID, bom manufacturingdb.CreateBillOfMaterialParams, lines []manufacturingdb.CreateBomLineParams) (manufacturingdb.CreateBillOfMaterialRow, error) {
	var created manufacturingdb.CreateBillOfMaterialRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if created, e = q.CreateBillOfMaterial(ctx, bom); e != nil {
			return e
		}
		for _, l := range lines {
			l.BomID = created.ID
			if e = q.CreateBomLine(ctx, l); e != nil {
				return e
			}
		}
		return nil
	})
	return created, err
}

// GetBOM fetches a BOM header and its lines inside org scope.
func (r *WorkOrderRepository) GetBOM(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (BOMDetail, error) {
	var out BOMDetail
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		h, e := q.GetBillOfMaterial(ctx, id)
		if e != nil {
			return e
		}
		lines, e := q.ListBomLines(ctx, id)
		if e != nil {
			return e
		}
		out = BOMDetail{Header: h, Lines: lines}
		return nil
	})
	return out, err
}

// ListBOMsWithCount lists bills of materials and returns the total row count.
func (r *WorkOrderRepository) ListBOMsWithCount(ctx context.Context, scope []uuid.UUID, arg manufacturingdb.ListBillsOfMaterialParams) ([]manufacturingdb.ListBillsOfMaterialRow, int64, error) {
	var rows []manufacturingdb.ListBillsOfMaterialRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if rows, e = q.ListBillsOfMaterial(ctx, arg); e != nil {
			return e
		}
		total, e = q.CountBillsOfMaterial(ctx)
		return e
	})
	return rows, total, err
}

// CreateWorkOrder inserts a new Planned work order.
func (r *WorkOrderRepository) CreateWorkOrder(ctx context.Context, scope []uuid.UUID, arg manufacturingdb.CreateWorkOrderParams) (manufacturingdb.CreateWorkOrderRow, error) {
	var out manufacturingdb.CreateWorkOrderRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateWorkOrder(ctx, arg)
		return e
	})
	return out, err
}

// GetByID returns a work order inside org scope.
func (r *WorkOrderRepository) GetByID(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (manufacturingdb.GetWorkOrderRow, error) {
	var out manufacturingdb.GetWorkOrderRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).GetWorkOrder(ctx, id)
		return e
	})
	return out, err
}

// ListWithCount returns paginated work orders (optionally filtered by status) and total count.
func (r *WorkOrderRepository) ListWithCount(ctx context.Context, scope []uuid.UUID, arg manufacturingdb.ListWorkOrdersParams) ([]manufacturingdb.ListWorkOrdersRow, int64, error) {
	var rows []manufacturingdb.ListWorkOrdersRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if rows, e = q.ListWorkOrders(ctx, arg); e != nil {
			return e
		}
		total, e = q.CountWorkOrders(ctx, arg.Status)
		return e
	})
	return rows, total, err
}

// UpdatePlanned updates a Planned work order using optimistic locking.
func (r *WorkOrderRepository) UpdatePlanned(ctx context.Context, scope []uuid.UUID, arg manufacturingdb.UpdatePlannedWorkOrderParams) (int32, int64, error) {
	var newVer int32
	var affected int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		v, e := r.q.WithTx(tx).UpdatePlannedWorkOrder(ctx, arg)
		if errors.Is(e, pgx.ErrNoRows) {
			return nil
		}
		if e != nil {
			return e
		}
		newVer = v
		affected = 1
		return nil
	})
	return newVer, affected, err
}

// DeletePlanned soft-deletes a Planned work order using optimistic locking.
func (r *WorkOrderRepository) DeletePlanned(ctx context.Context, scope []uuid.UUID, arg manufacturingdb.SoftDeletePlannedWorkOrderParams) (int64, error) {
	var affected int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		affected, e = r.q.WithTx(tx).SoftDeletePlannedWorkOrder(ctx, arg)
		return e
	})
	return affected, err
}

// ReleaseAndEmit transitions Planned->Released and emits manufacturing.workorder.released atomically.
func (r *WorkOrderRepository) ReleaseAndEmit(ctx context.Context, scope []uuid.UUID, arg manufacturingdb.ReleaseWorkOrderParams, ev manufacturingdb.InsertOutboxParams) (int32, int64, error) {
	var newVer int32
	var affected int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		v, e := q.ReleaseWorkOrder(ctx, arg)
		if errors.Is(e, pgx.ErrNoRows) {
			return nil
		}
		if e != nil {
			return e
		}
		newVer = v
		affected = 1
		return q.InsertOutbox(ctx, ev)
	})
	return newVer, affected, err
}

// CompleteAndEmit transitions Released/InProgress->Completed and emits manufacturing.workorder.completed atomically.
func (r *WorkOrderRepository) CompleteAndEmit(ctx context.Context, scope []uuid.UUID, arg manufacturingdb.CompleteWorkOrderParams, ev manufacturingdb.InsertOutboxParams) (int32, int64, error) {
	var newVer int32
	var affected int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		v, e := q.CompleteWorkOrder(ctx, arg)
		if errors.Is(e, pgx.ErrNoRows) {
			return nil
		}
		if e != nil {
			return e
		}
		newVer = v
		affected = 1
		return q.InsertOutbox(ctx, ev)
	})
	return newVer, affected, err
}

// DrainAndPublish drains unpublished outbox events under row locks and publishes them to Kafka.
func (r *WorkOrderRepository) DrainAndPublish(ctx context.Context, pub *eventbus.Publisher, limit int32) (int, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin outbox tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	q := r.q.WithTx(tx)
	rows, err := q.DrainOutbox(ctx, limit)
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	published := 0
	var pubErr error
	for _, row := range rows {
		env := eventbus.Envelope{
			EventID:        row.ID.String(),
			EventType:      row.EventType,
			AggregateType:  row.AggregateType,
			AggregateID:    row.AggregateID.String(),
			OccurredAt:     row.CreatedAt.Time,
			CorrelationID:  row.CorrelationID.String(),
			OrganizationID: row.OrganizationID.String(),
			SchemaVersion:  "v1",
			Data:           json.RawMessage(row.Payload),
		}
		if e := pub.Publish(ctx, env); e != nil {
			pubErr = e
			break
		}
		if e := q.MarkOutboxPublished(ctx, manufacturingdb.MarkOutboxPublishedParams{ID: row.ID, CreatedAt: row.CreatedAt}); e != nil {
			return published, e
		}
		published++
	}
	if published > 0 {
		if err := tx.Commit(ctx); err != nil {
			return 0, fmt.Errorf("commit outbox tx: %w", err)
		}
		committed = true
	}
	return published, pubErr
}
