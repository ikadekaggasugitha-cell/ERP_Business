// Package repository implements RLS-enforced Maintenance Management persistence
// (schedules, work orders, parts, checklists, costs, exact *big.Rat totals, and outbox).
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/maintenance-service/internal/mntdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type MaintenanceRepository struct {
	db *platform.DB
	q  *mntdb.Queries
}

func NewMaintenanceRepository(db *platform.DB) *MaintenanceRepository {
	var q *mntdb.Queries
	if db != nil && db.Pool != nil {
		q = mntdb.New(db.Pool)
	}
	return &MaintenanceRepository{db: db, q: q}
}

func (r *MaintenanceRepository) withOrgTx(ctx context.Context, scope []uuid.UUID, fn func(pgx.Tx) error) error {
	if len(scope) == 0 {
		return errors.New("org scope must not be empty")
	}
	return r.db.WithOrgScope(ctx, platform.ScopeWithSystemOrg(scope), fn)
}

func isEditableWOStatus(status string) bool {
	return status == "Open" || status == "Assigned" || status == "InProgress"
}

func emit(ctx context.Context, q *mntdb.Queries, aggType, eventType string, id, org, corr uuid.UUID, payload map[string]any) error {
	payload["id"] = id.String()
	payload["organizationId"] = org.String()
	b, _ := json.Marshal(payload)
	return q.InsertOutbox(ctx, mntdb.InsertOutboxParams{
		AggregateType: aggType, AggregateID: id, EventType: eventType,
		Payload: b, OrganizationID: org, CorrelationID: corr,
	})
}

func (r *MaintenanceRepository) CreateSchedule(ctx context.Context, scope []uuid.UUID, arg mntdb.CreateMaintenanceScheduleParams) (mntdb.CreateMaintenanceScheduleRow, error) {
	var out mntdb.CreateMaintenanceScheduleRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateMaintenanceSchedule(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "failed to create maintenance schedule")
}

func (r *MaintenanceRepository) ListSchedules(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]mntdb.MntMaintenanceSchedule, int64, error) {
	var rows []mntdb.MntMaintenanceSchedule
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountMaintenanceSchedules(ctx); e != nil {
			return e
		}
		rows, e = q.ListMaintenanceSchedules(ctx, mntdb.ListMaintenanceSchedulesParams{Limit: limit, Offset: offset})
		return e
	})
	return rows, total, err
}

func (r *MaintenanceRepository) UpdateSchedule(ctx context.Context, scope []uuid.UUID, arg mntdb.UpdateMaintenanceScheduleParams) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		cur, e := q.GetMaintenanceSchedule(ctx, arg.ID)
		if e != nil {
			return httpx.NotFound("schedule not found")
		}
		if cur.OrganizationID == platform.SystemOrgID {
			return httpx.Forbidden("system maintenance schedule template is read-only")
		}
		next, e = q.UpdateMaintenanceSchedule(ctx, arg)
		return e
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, httpx.Conflict("VERSION_CONFLICT", "schedule not found or version mismatch")
	}
	return next, httpx.MapDBError(err, "schedule update conflict")
}

func (r *MaintenanceRepository) DeleteSchedule(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) error {
	var n int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		cur, e := q.GetMaintenanceSchedule(ctx, id)
		if e != nil {
			return httpx.NotFound("schedule not found")
		}
		if cur.OrganizationID == platform.SystemOrgID {
			return httpx.Forbidden("system maintenance schedule template cannot be deleted")
		}
		n, e = q.SoftDeleteMaintenanceSchedule(ctx, mntdb.SoftDeleteMaintenanceScheduleParams{ID: id, UpdatedBy: &actor, RowVersion: version})
		return e
	})
	if err != nil {
		return httpx.MapDBError(err, "cannot delete schedule")
	}
	if n == 0 {
		return httpx.Conflict("VERSION_CONFLICT", "schedule not found or version mismatch")
	}
	return nil
}

func (r *MaintenanceRepository) CreateWorkOrder(ctx context.Context, scope []uuid.UUID, arg mntdb.CreateWorkOrderParams) (mntdb.CreateWorkOrderRow, error) {
	var out mntdb.CreateWorkOrderRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateWorkOrder(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "work order number already exists")
}

func (r *MaintenanceRepository) GetWorkOrder(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (mntdb.MntWorkOrder, error) {
	var out mntdb.MntWorkOrder
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).GetWorkOrder(ctx, id)
		return e
	})
	return out, httpx.MapDBError(err, "work order not found")
}

func (r *MaintenanceRepository) ListWorkOrders(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]mntdb.MntWorkOrder, int64, error) {
	var rows []mntdb.MntWorkOrder
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountWorkOrders(ctx, status); e != nil {
			return e
		}
		rows, e = q.ListWorkOrders(ctx, mntdb.ListWorkOrdersParams{Limit: limit, Offset: offset, Status: status})
		return e
	})
	return rows, total, err
}

func (r *MaintenanceRepository) AssignWorkOrder(ctx context.Context, scope []uuid.UUID, arg mntdb.AssignWorkOrderParams) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		next, e = r.q.WithTx(tx).AssignWorkOrder(ctx, arg)
		return e
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, httpx.Conflict("INVALID_STATE", "work order must be Open with matching rowVersion")
	}
	return next, err
}

func (r *MaintenanceRepository) StartWorkOrder(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		next, e = r.q.WithTx(tx).StartWorkOrder(ctx, mntdb.StartWorkOrderParams{ID: id, UpdatedBy: &actor, RowVersion: version})
		return e
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, httpx.Conflict("INVALID_STATE", "work order must be Assigned with matching rowVersion")
	}
	return next, err
}

func (r *MaintenanceRepository) CompleteWorkOrderAndEmit(ctx context.Context, scope []uuid.UUID, id, org, actor, corr uuid.UUID, version int32) (int32, string, error) {
	var next int32
	var totalCostStr string
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		wo, e := q.GetWorkOrder(ctx, id)
		if e != nil {
			return httpx.NotFound("work order not found")
		}
		if wo.Status != "InProgress" {
			return httpx.Conflict("INVALID_STATE", "work order must be InProgress before completion")
		}
		checklist, e := q.ListWorkOrderChecklist(ctx, id)
		if e != nil {
			return e
		}
		for _, item := range checklist {
			if !item.IsDone {
				return httpx.Unprocessable("CHECKLIST_INCOMPLETE", "all work order checklist steps must be completed before completing the work order")
			}
		}
		next, e = q.CompleteWorkOrder(ctx, mntdb.CompleteWorkOrderParams{ID: id, UpdatedBy: &actor, RowVersion: version})
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.Conflict("VERSION_CONFLICT", "work order version mismatch")
		}
		if e != nil {
			return e
		}
		costs, e := q.ListMaintenanceCosts(ctx, id)
		if e != nil {
			return e
		}
		parts, e := q.ListWorkOrderParts(ctx, id)
		if e != nil {
			return e
		}
		total := big.NewRat(0, 1)
		for _, c := range costs {
			if rat := platform.NumericToRat(c.Amount); rat != nil {
				total.Add(total, rat)
			}
		}
		for _, p := range parts {
			qty := platform.NumericToRat(p.Quantity)
			uc := platform.NumericToRat(p.UnitCost)
			if qty != nil && uc != nil {
				total.Add(total, new(big.Rat).Mul(qty, uc))
			}
		}
		if wo.MaintenanceType == "Preventive" {
			if e = q.AdvancePreventiveScheduleDueDate(ctx, mntdb.AdvancePreventiveScheduleDueDateParams{
				AssetRef: wo.AssetRef, UpdatedBy: &actor,
			}); e != nil {
				return e
			}
		}
		totalCostStr = total.FloatString(4)
		return emit(ctx, q, "WorkOrder", "maintenance.workorder.completed", id, org, corr, map[string]any{
			"workOrderId":     id.String(),
			"woNo":            wo.WoNo,
			"assetRef":        wo.AssetRef.String(),
			"maintenanceType": wo.MaintenanceType,
			"totalCost":       totalCostStr,
		})
	})
	return next, totalCostStr, err
}

func (r *MaintenanceRepository) CloseWorkOrder(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		next, e = r.q.WithTx(tx).CloseWorkOrder(ctx, mntdb.CloseWorkOrderParams{ID: id, UpdatedBy: &actor, RowVersion: version})
		return e
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, httpx.Conflict("INVALID_STATE", "work order must be Completed with matching rowVersion")
	}
	return next, err
}

func (r *MaintenanceRepository) AddPartAndEmit(ctx context.Context, scope []uuid.UUID, arg mntdb.CreateWorkOrderPartParams, warehouseID *uuid.UUID, corr uuid.UUID) (mntdb.CreateWorkOrderPartRow, error) {
	var out mntdb.CreateWorkOrderPartRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		wo, e := q.GetWorkOrder(ctx, arg.WorkOrderID)
		if e != nil {
			return httpx.NotFound("work order not found")
		}
		if !isEditableWOStatus(wo.Status) {
			return httpx.Conflict("INVALID_STATE", "cannot add parts to a Completed or Closed work order")
		}
		out, e = q.CreateWorkOrderPart(ctx, arg)
		if e != nil {
			return e
		}
		payload := map[string]any{
			"partId":      out.ID.String(),
			"workOrderId": wo.ID.String(),
			"woNo":        wo.WoNo,
			"itemId":      arg.ItemRef.String(),
			"quantity":    platform.FormatNumeric(arg.Quantity, 4),
			"unitCost":    platform.FormatNumeric(arg.UnitCost, 4),
		}
		if warehouseID != nil {
			payload["warehouseId"] = warehouseID.String()
		}
		return emit(ctx, q, "WorkOrderPart", "maintenance.workorder.parts_requested", out.ID, arg.OrganizationID, corr, payload)
	})
	return out, httpx.MapDBError(err, "failed to add work order part")
}

func (r *MaintenanceRepository) ListParts(ctx context.Context, scope []uuid.UUID, woID uuid.UUID) ([]mntdb.MntWorkOrderPart, error) {
	var rows []mntdb.MntWorkOrderPart
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		rows, e = r.q.WithTx(tx).ListWorkOrderParts(ctx, woID)
		return e
	})
	return rows, err
}

func (r *MaintenanceRepository) AddChecklist(ctx context.Context, scope []uuid.UUID, arg mntdb.CreateWorkOrderChecklistParams) (mntdb.CreateWorkOrderChecklistRow, error) {
	var out mntdb.CreateWorkOrderChecklistRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		wo, e := q.GetWorkOrder(ctx, arg.WorkOrderID)
		if e != nil {
			return httpx.NotFound("work order not found")
		}
		if !isEditableWOStatus(wo.Status) {
			return httpx.Conflict("INVALID_STATE", "cannot add checklist to a Completed or Closed work order")
		}
		out, e = q.CreateWorkOrderChecklist(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "checklist step already exists")
}

func (r *MaintenanceRepository) ListChecklist(ctx context.Context, scope []uuid.UUID, woID uuid.UUID) ([]mntdb.MntWorkOrderChecklist, error) {
	var rows []mntdb.MntWorkOrderChecklist
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		rows, e = r.q.WithTx(tx).ListWorkOrderChecklist(ctx, woID)
		return e
	})
	return rows, err
}

func (r *MaintenanceRepository) UpdateChecklist(ctx context.Context, scope []uuid.UUID, arg mntdb.UpdateWorkOrderChecklistParams) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		next, e = r.q.WithTx(tx).UpdateWorkOrderChecklist(ctx, arg)
		return e
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, httpx.Conflict("VERSION_CONFLICT", "checklist item not found or version mismatch")
	}
	return next, err
}

func (r *MaintenanceRepository) AddCost(ctx context.Context, scope []uuid.UUID, arg mntdb.CreateMaintenanceCostParams) (mntdb.CreateMaintenanceCostRow, error) {
	var out mntdb.CreateMaintenanceCostRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		wo, e := q.GetWorkOrder(ctx, arg.WorkOrderID)
		if e != nil {
			return httpx.NotFound("work order not found")
		}
		if !isEditableWOStatus(wo.Status) {
			return httpx.Conflict("INVALID_STATE", "cannot add cost to a Completed or Closed work order")
		}
		out, e = q.CreateMaintenanceCost(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "failed to record maintenance cost")
}

func (r *MaintenanceRepository) ListCosts(ctx context.Context, scope []uuid.UUID, woID uuid.UUID) ([]mntdb.MntMaintenanceCost, error) {
	var rows []mntdb.MntMaintenanceCost
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		rows, e = r.q.WithTx(tx).ListMaintenanceCosts(ctx, woID)
		return e
	})
	return rows, err
}

func (r *MaintenanceRepository) DrainAndPublish(ctx context.Context, batch int32, pub *eventbus.Publisher) (int, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := r.q.WithTx(tx)
	rows, err := q.ClaimUnpublishedOutbox(ctx, batch)
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
			EventID: row.ID.String(), EventType: row.EventType, AggregateType: row.AggregateType,
			AggregateID: row.AggregateID.String(), OccurredAt: row.CreatedAt.Time,
			CorrelationID: row.CorrelationID.String(), OrganizationID: row.OrganizationID.String(),
			SchemaVersion: "v1", Data: json.RawMessage(row.Payload),
		}
		if err := pub.Publish(ctx, env); err != nil {
			pubErr = err
			break
		}
		if err := q.MarkOutboxPublished(ctx, mntdb.MarkOutboxPublishedParams{ID: row.ID, CreatedAt: row.CreatedAt}); err != nil {
			pubErr = err
			break
		}
		published++
	}
	if published > 0 {
		if err := tx.Commit(ctx); err != nil {
			return 0, err
		}
	}
	return published, pubErr
}
