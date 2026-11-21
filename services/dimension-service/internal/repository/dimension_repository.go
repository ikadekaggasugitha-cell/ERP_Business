// Package repository provides RLS-enforced Cost Center, Profit Center,
// Allocation Rule persistence and the transactional outbox relay.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/dimension-service/internal/dimensiondb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type DimensionRepository struct {
	db *platform.DB
	q  *dimensiondb.Queries
}

func NewDimensionRepository(db *platform.DB) *DimensionRepository {
	var q *dimensiondb.Queries
	if db != nil && db.Pool != nil {
		q = dimensiondb.New(db.Pool)
	}
	return &DimensionRepository{db: db, q: q}
}

// withOrgTx fails closed on an empty scope and runs fn inside an RLS-scoped tx.
func (r *DimensionRepository) withOrgTx(ctx context.Context, scope []uuid.UUID, fn func(pgx.Tx) error) error {
	if len(scope) == 0 {
		return errors.New("org scope must not be empty")
	}
	return r.db.WithOrgScope(ctx, scope, fn)
}

func emit(ctx context.Context, q *dimensiondb.Queries, aggType, eventType string, id, org, corr uuid.UUID, payload map[string]any) error {
	payload["id"] = id.String()
	payload["organizationId"] = org.String()
	b, _ := json.Marshal(payload)
	return q.InsertOutbox(ctx, dimensiondb.InsertOutboxParams{
		AggregateType: aggType, AggregateID: id, EventType: eventType,
		Payload: b, OrganizationID: org, CorrelationID: corr,
	})
}

func staleOrNotFound(what string) error {
	return httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau "+what+" tidak ditemukan")
}

// ---------------- Cost Center ----------------

func (r *DimensionRepository) CreateCostCenter(ctx context.Context, scope []uuid.UUID, arg dimensiondb.CreateCostCenterParams, corr uuid.UUID) (dimensiondb.CreateCostCenterRow, error) {
	var out dimensiondb.CreateCostCenterRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if out, e = q.CreateCostCenter(ctx, arg); e != nil {
			return e
		}
		return emit(ctx, q, "CostCenter", "dimension.cost_center.upserted", out.ID, arg.OrganizationID, corr,
			map[string]any{"code": arg.Code, "name": arg.Name, "status": arg.Status})
	})
	if err != nil {
		return out, httpx.MapDBError(err, "cost center with this code already exists in organization")
	}
	return out, nil
}

func (r *DimensionRepository) GetCostCenter(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (dimensiondb.CcCostCenter, error) {
	var out dimensiondb.CcCostCenter
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).GetCostCenter(ctx, id)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("cost center not found")
		}
		return e
	})
	return out, err
}

func (r *DimensionRepository) ListCostCenters(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]dimensiondb.CcCostCenter, int64, error) {
	var rows []dimensiondb.CcCostCenter
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountCostCenters(ctx, status); e != nil {
			return e
		}
		rows, e = q.ListCostCenters(ctx, dimensiondb.ListCostCentersParams{Status: status, Limit: limit, Offset: offset})
		return e
	})
	return rows, total, err
}

func (r *DimensionRepository) UpdateCostCenter(ctx context.Context, scope []uuid.UUID, arg dimensiondb.UpdateCostCenterParams, corr uuid.UUID) (int32, error) {
	var ver int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		if arg.ParentID != nil {
			cur := *arg.ParentID
			for depth := 0; depth < 32; depth++ {
				if cur == arg.ID {
					return httpx.BadRequest("parentId creates a cost center hierarchy cycle")
				}
				p, pe := q.GetCostCenter(ctx, cur)
				if pe != nil {
					if errors.Is(pe, pgx.ErrNoRows) {
						return httpx.BadRequest("parentId cost center not found")
					}
					return pe
				}
				if p.ParentID == nil {
					break
				}
				cur = *p.ParentID
			}
		}
		var e error
		ver, e = q.UpdateCostCenter(ctx, arg)
		if errors.Is(e, pgx.ErrNoRows) {
			return staleOrNotFound("cost center")
		}
		if e != nil {
			return e
		}
		return emit(ctx, q, "CostCenter", "dimension.cost_center.upserted", arg.ID, scope[0], corr,
			map[string]any{"name": arg.Name, "status": arg.Status})
	})
	if err != nil {
		return 0, httpx.MapDBError(err, "")
	}
	return ver, nil
}

func (r *DimensionRepository) DeleteCostCenter(ctx context.Context, scope []uuid.UUID, arg dimensiondb.DeleteCostCenterParams) error {
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var childCount, ruleCount int64
		if e := tx.QueryRow(ctx, "SELECT COUNT(*) FROM cc.cost_center WHERE parent_id = $1 AND deleted_at IS NULL", arg.ID).Scan(&childCount); e != nil {
			return e
		}
		if childCount > 0 {
			return httpx.Conflict("HAS_CHILDREN", "cannot delete cost center with active child cost centers")
		}
		if e := tx.QueryRow(ctx, "SELECT COUNT(*) FROM cc.cost_allocation_rule WHERE source_cost_center_id = $1 AND deleted_at IS NULL", arg.ID).Scan(&ruleCount); e != nil {
			return e
		}
		if ruleCount > 0 {
			return httpx.Conflict("HAS_CHILDREN", "cannot delete cost center referenced by active allocation rules")
		}
		n, e := r.q.WithTx(tx).DeleteCostCenter(ctx, arg)
		if e != nil {
			return e
		}
		if n == 0 {
			return staleOrNotFound("cost center")
		}
		return nil
	})
	if err != nil {
		return httpx.MapDBError(err, "")
	}
	return nil
}

// ---------------- Profit Center ----------------

func (r *DimensionRepository) CreateProfitCenter(ctx context.Context, scope []uuid.UUID, arg dimensiondb.CreateProfitCenterParams, corr uuid.UUID) (dimensiondb.CreateProfitCenterRow, error) {
	var out dimensiondb.CreateProfitCenterRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if out, e = q.CreateProfitCenter(ctx, arg); e != nil {
			return e
		}
		return emit(ctx, q, "ProfitCenter", "dimension.profit_center.upserted", out.ID, arg.OrganizationID, corr,
			map[string]any{"code": arg.Code, "name": arg.Name, "status": arg.Status})
	})
	if err != nil {
		return out, httpx.MapDBError(err, "profit center with this code already exists in organization")
	}
	return out, nil
}

func (r *DimensionRepository) GetProfitCenter(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (dimensiondb.PcProfitCenter, error) {
	var out dimensiondb.PcProfitCenter
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).GetProfitCenter(ctx, id)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("profit center not found")
		}
		return e
	})
	return out, err
}

func (r *DimensionRepository) ListProfitCenters(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]dimensiondb.PcProfitCenter, int64, error) {
	var rows []dimensiondb.PcProfitCenter
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountProfitCenters(ctx, status); e != nil {
			return e
		}
		rows, e = q.ListProfitCenters(ctx, dimensiondb.ListProfitCentersParams{Status: status, Limit: limit, Offset: offset})
		return e
	})
	return rows, total, err
}

func (r *DimensionRepository) UpdateProfitCenter(ctx context.Context, scope []uuid.UUID, arg dimensiondb.UpdateProfitCenterParams, corr uuid.UUID) (int32, error) {
	var ver int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		if arg.ParentID != nil {
			cur := *arg.ParentID
			for depth := 0; depth < 32; depth++ {
				if cur == arg.ID {
					return httpx.BadRequest("parentId creates a profit center hierarchy cycle")
				}
				p, pe := q.GetProfitCenter(ctx, cur)
				if pe != nil {
					if errors.Is(pe, pgx.ErrNoRows) {
						return httpx.BadRequest("parentId profit center not found")
					}
					return pe
				}
				if p.ParentID == nil {
					break
				}
				cur = *p.ParentID
			}
		}
		var e error
		ver, e = q.UpdateProfitCenter(ctx, arg)
		if errors.Is(e, pgx.ErrNoRows) {
			return staleOrNotFound("profit center")
		}
		if e != nil {
			return e
		}
		return emit(ctx, q, "ProfitCenter", "dimension.profit_center.upserted", arg.ID, scope[0], corr,
			map[string]any{"name": arg.Name, "status": arg.Status})
	})
	if err != nil {
		return 0, httpx.MapDBError(err, "")
	}
	return ver, nil
}

func (r *DimensionRepository) DeleteProfitCenter(ctx context.Context, scope []uuid.UUID, arg dimensiondb.DeleteProfitCenterParams) error {
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var childCount int64
		if e := tx.QueryRow(ctx, "SELECT COUNT(*) FROM pc.profit_center WHERE parent_id = $1 AND deleted_at IS NULL", arg.ID).Scan(&childCount); e != nil {
			return e
		}
		if childCount > 0 {
			return httpx.Conflict("HAS_CHILDREN", "cannot delete profit center with active child profit centers")
		}
		n, e := r.q.WithTx(tx).DeleteProfitCenter(ctx, arg)
		if e != nil {
			return e
		}
		if n == 0 {
			return staleOrNotFound("profit center")
		}
		return nil
	})
	if err != nil {
		return httpx.MapDBError(err, "")
	}
	return nil
}

// ---------------- Allocation Rule ----------------

type rawTarget struct {
	CostCenterID uuid.UUID `json:"costCenterId"`
	Percentage   string    `json:"percentage"`
}

func validateTargetCostCentersExist(ctx context.Context, q *dimensiondb.Queries, rawTargets []byte) error {
	var targets []rawTarget
	if err := json.Unmarshal(rawTargets, &targets); err != nil {
		return httpx.BadRequest("invalid allocation targets JSON")
	}
	for _, t := range targets {
		cc, e := q.GetCostCenter(ctx, t.CostCenterID)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.Unprocessable("TARGET_NOT_FOUND", "target cost center "+t.CostCenterID.String()+" not found")
		}
		if e != nil {
			return e
		}
		if cc.Status == "Closed" {
			return httpx.Unprocessable("TARGET_CLOSED", "target cost center "+cc.Code+" is closed")
		}
	}
	return nil
}

// ComputeDistributions splits totalAmount across targets using exact *big.Rat math
// and assigns any rounding remainder to the final target so sum(shares) == totalAmount.
func ComputeDistributions(rawTargets []byte, totalAmount string) []map[string]any {
	amtNum, err := platform.ParseNumeric(totalAmount)
	if err != nil {
		return []map[string]any{}
	}
	amtRat := platform.NumericToRat(amtNum)
	var targets []rawTarget
	_ = json.Unmarshal(rawTargets, &targets)
	if len(targets) == 0 {
		return []map[string]any{}
	}
	hundred := big.NewRat(100, 1)
	distributions := make([]map[string]any, 0, len(targets))
	allocatedSum := new(big.Rat)
	for i, t := range targets {
		var shareNumStr string
		if i == len(targets)-1 {
			rem := new(big.Rat).Sub(amtRat, allocatedSum)
			remNum, _ := platform.RatToNumeric(rem, 4)
			shareNumStr = platform.FormatNumeric(remNum, 4)
		} else {
			pct, _ := new(big.Rat).SetString(strings.TrimSpace(t.Percentage))
			if pct == nil {
				pct = new(big.Rat)
			}
			share := new(big.Rat).Quo(new(big.Rat).Mul(amtRat, pct), hundred)
			shareNum, _ := platform.RatToNumeric(share, 4)
			allocatedSum.Add(allocatedSum, platform.NumericToRat(shareNum))
			shareNumStr = platform.FormatNumeric(shareNum, 4)
		}
		distributions = append(distributions, map[string]any{
			"costCenterId": t.CostCenterID,
			"percentage":   t.Percentage,
			"amount":       shareNumStr,
		})
	}
	return distributions
}

func (r *DimensionRepository) CreateAllocationRule(ctx context.Context, scope []uuid.UUID, arg dimensiondb.CreateAllocationRuleParams, corr uuid.UUID) (dimensiondb.CreateAllocationRuleRow, error) {
	var out dimensiondb.CreateAllocationRuleRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		if _, e := q.GetCostCenter(ctx, arg.SourceCostCenterID); errors.Is(e, pgx.ErrNoRows) {
			return httpx.Unprocessable("SOURCE_NOT_FOUND", "source cost center not found")
		} else if e != nil {
			return e
		}
		if e := validateTargetCostCentersExist(ctx, q, arg.Targets); e != nil {
			return e
		}
		var e error
		if out, e = q.CreateAllocationRule(ctx, arg); e != nil {
			return e
		}
		return emit(ctx, q, "AllocationRule", "dimension.allocation_rule.upserted", out.ID, arg.OrganizationID, corr,
			map[string]any{"name": arg.Name, "sourceCostCenterId": arg.SourceCostCenterID.String(), "basis": arg.AllocationBasis})
	})
	if err != nil {
		return out, httpx.MapDBError(err, "")
	}
	return out, nil
}

func (r *DimensionRepository) GetAllocationRule(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (dimensiondb.CcCostAllocationRule, error) {
	var out dimensiondb.CcCostAllocationRule
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).GetAllocationRule(ctx, id)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("allocation rule not found")
		}
		return e
	})
	return out, err
}

func (r *DimensionRepository) ListAllocationRules(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]dimensiondb.CcCostAllocationRule, int64, error) {
	var rows []dimensiondb.CcCostAllocationRule
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountAllocationRules(ctx); e != nil {
			return e
		}
		rows, e = q.ListAllocationRules(ctx, dimensiondb.ListAllocationRulesParams{Limit: limit, Offset: offset})
		return e
	})
	return rows, total, err
}

func (r *DimensionRepository) UpdateAllocationRule(ctx context.Context, scope []uuid.UUID, arg dimensiondb.UpdateAllocationRuleParams, corr uuid.UUID) (int32, error) {
	var ver int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		if e := validateTargetCostCentersExist(ctx, q, arg.Targets); e != nil {
			return e
		}
		var e error
		ver, e = q.UpdateAllocationRule(ctx, arg)
		if errors.Is(e, pgx.ErrNoRows) {
			return staleOrNotFound("allocation rule")
		}
		if e != nil {
			return e
		}
		return emit(ctx, q, "AllocationRule", "dimension.allocation_rule.upserted", arg.ID, scope[0], corr,
			map[string]any{"name": arg.Name, "basis": arg.AllocationBasis, "isActive": arg.IsActive})
	})
	if err != nil {
		return 0, httpx.MapDBError(err, "")
	}
	return ver, nil
}

func (r *DimensionRepository) DeleteAllocationRule(ctx context.Context, scope []uuid.UUID, arg dimensiondb.DeleteAllocationRuleParams) error {
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		n, e := r.q.WithTx(tx).DeleteAllocationRule(ctx, arg)
		if e != nil {
			return e
		}
		if n == 0 {
			return staleOrNotFound("allocation rule")
		}
		return nil
	})
	if err != nil {
		return httpx.MapDBError(err, "")
	}
	return nil
}

func (r *DimensionRepository) ExecuteAllocationRule(ctx context.Context, scope []uuid.UUID, ruleID uuid.UUID, amount string, fiscalPeriodID *uuid.UUID, corr uuid.UUID) (dimensiondb.CcCostAllocationRule, error) {
	var rule dimensiondb.CcCostAllocationRule
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		rule, e = q.GetAllocationRule(ctx, ruleID)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("allocation rule not found")
		}
		if e != nil {
			return e
		}
		if !rule.IsActive {
			return httpx.Conflict("RULE_INACTIVE", "allocation rule is inactive")
		}
		distributions := ComputeDistributions(rule.Targets, amount)
		payload := map[string]any{
			"ruleId":             rule.ID.String(),
			"name":               rule.Name,
			"sourceCostCenterId": rule.SourceCostCenterID.String(),
			"allocationBasis":    rule.AllocationBasis,
			"amount":             amount,
			"targets":            json.RawMessage(rule.Targets),
			"distributions":      distributions,
		}
		if fiscalPeriodID != nil {
			payload["fiscalPeriodId"] = fiscalPeriodID.String()
		}
		return emit(ctx, q, "AllocationRule", "dimension.allocation_rule.executed", rule.ID, rule.OrganizationID, corr, payload)
	})
	if err != nil {
		return rule, httpx.MapDBError(err, "")
	}
	return rule, nil
}

// ---------------- Assignments ----------------

func (r *DimensionRepository) CreateCostCenterAssignment(ctx context.Context, scope []uuid.UUID, arg dimensiondb.CreateCostCenterAssignmentParams) (dimensiondb.CreateCostCenterAssignmentRow, error) {
	var out dimensiondb.CreateCostCenterAssignmentRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateCostCenterAssignment(ctx, arg)
		return e
	})
	if err != nil {
		return out, httpx.MapDBError(err, "cost center assignment conflict")
	}
	return out, nil
}

func (r *DimensionRepository) ListCostCenterAssignments(ctx context.Context, scope []uuid.UUID, costCenterID *uuid.UUID, limit, offset int32) ([]dimensiondb.CcCostCenterAssignment, int64, error) {
	var rows []dimensiondb.CcCostCenterAssignment
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountCostCenterAssignments(ctx, costCenterID); e != nil {
			return e
		}
		rows, e = q.ListCostCenterAssignments(ctx, dimensiondb.ListCostCenterAssignmentsParams{
			CostCenterID: costCenterID, Limit: limit, Offset: offset,
		})
		return e
	})
	return rows, total, err
}

func (r *DimensionRepository) DeleteCostCenterAssignment(ctx context.Context, scope []uuid.UUID, arg dimensiondb.DeleteCostCenterAssignmentParams) error {
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		n, e := r.q.WithTx(tx).DeleteCostCenterAssignment(ctx, arg)
		if e != nil {
			return e
		}
		if n == 0 {
			return staleOrNotFound("cost center assignment")
		}
		return nil
	})
	return httpx.MapDBError(err, "")
}

func (r *DimensionRepository) CreateProfitCenterAssignment(ctx context.Context, scope []uuid.UUID, arg dimensiondb.CreateProfitCenterAssignmentParams) (dimensiondb.CreateProfitCenterAssignmentRow, error) {
	var out dimensiondb.CreateProfitCenterAssignmentRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateProfitCenterAssignment(ctx, arg)
		return e
	})
	if err != nil {
		return out, httpx.MapDBError(err, "profit center assignment conflict")
	}
	return out, nil
}

func (r *DimensionRepository) ListProfitCenterAssignments(ctx context.Context, scope []uuid.UUID, profitCenterID *uuid.UUID, limit, offset int32) ([]dimensiondb.PcProfitCenterAssignment, int64, error) {
	var rows []dimensiondb.PcProfitCenterAssignment
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountProfitCenterAssignments(ctx, profitCenterID); e != nil {
			return e
		}
		rows, e = q.ListProfitCenterAssignments(ctx, dimensiondb.ListProfitCenterAssignmentsParams{
			ProfitCenterID: profitCenterID, Limit: limit, Offset: offset,
		})
		return e
	})
	return rows, total, err
}

func (r *DimensionRepository) DeleteProfitCenterAssignment(ctx context.Context, scope []uuid.UUID, arg dimensiondb.DeleteProfitCenterAssignmentParams) error {
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		n, e := r.q.WithTx(tx).DeleteProfitCenterAssignment(ctx, arg)
		if e != nil {
			return e
		}
		if n == 0 {
			return staleOrNotFound("profit center assignment")
		}
		return nil
	})
	return httpx.MapDBError(err, "")
}

// DrainAndPublish drains cc.outbox under FOR UPDATE SKIP LOCKED inside a tx.
// Partial publish failures commit already-published rows (never roll back the batch).
func (r *DimensionRepository) DrainAndPublish(ctx context.Context, limit int32, pub *eventbus.Publisher) (int, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin outbox drain: %w", err)
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
		if e := q.MarkOutboxPublished(ctx, dimensiondb.MarkOutboxPublishedParams{ID: row.ID, CreatedAt: row.CreatedAt}); e != nil {
			pubErr = e
			break
		}
		published++
	}
	if published > 0 {
		if err := tx.Commit(ctx); err != nil {
			return 0, fmt.Errorf("commit outbox drain: %w", err)
		}
		committed = true
	}
	return published, pubErr
}

