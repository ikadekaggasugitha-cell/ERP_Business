// Package repository implements RLS-enforced Consolidation Groups, Members,
// Elimination Rules/Entries, Top-Side Adjustments, double-entry balance checks, and outbox.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/consolidation-service/internal/consdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type ConsolidationRepository struct {
	db *platform.DB
	q  *consdb.Queries
}

func NewConsolidationRepository(db *platform.DB) *ConsolidationRepository {
	var q *consdb.Queries
	if db != nil && db.Pool != nil {
		q = consdb.New(db.Pool)
	}
	return &ConsolidationRepository{db: db, q: q}
}

func (r *ConsolidationRepository) withOrgTx(ctx context.Context, scope []uuid.UUID, fn func(pgx.Tx) error) error {
	if len(scope) == 0 {
		return errors.New("org scope must not be empty")
	}
	return r.db.WithOrgScope(ctx, scope, fn)
}

func emit(ctx context.Context, q *consdb.Queries, aggType, eventType string, id, org, corr uuid.UUID, payload map[string]any) error {
	payload["id"] = id.String()
	payload["organizationId"] = org.String()
	b, _ := json.Marshal(payload)
	return q.InsertOutbox(ctx, consdb.InsertOutboxParams{
		AggregateType: aggType, AggregateID: id, EventType: eventType,
		Payload: b, OrganizationID: org, CorrelationID: corr,
	})
}

func (r *ConsolidationRepository) CreateGroup(ctx context.Context, scope []uuid.UUID, arg consdb.CreateConsolidationGroupParams) (consdb.CreateConsolidationGroupRow, error) {
	var out consdb.CreateConsolidationGroupRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateConsolidationGroup(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "consolidation group code already exists")
}

func (r *ConsolidationRepository) ListGroups(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]consdb.ConsConsolidationGroup, int64, error) {
	var rows []consdb.ConsConsolidationGroup
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountConsolidationGroups(ctx); e != nil {
			return e
		}
		rows, e = q.ListConsolidationGroups(ctx, consdb.ListConsolidationGroupsParams{Limit: limit, Offset: offset})
		return e
	})
	return rows, total, err
}

func (r *ConsolidationRepository) AddMember(ctx context.Context, scope []uuid.UUID, arg consdb.CreateConsolidationMemberParams) (consdb.CreateConsolidationMemberRow, error) {
	var out consdb.CreateConsolidationMemberRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateConsolidationMember(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "company already belongs to consolidation group")
}

func (r *ConsolidationRepository) ListMembers(ctx context.Context, scope []uuid.UUID, groupID uuid.UUID) ([]consdb.ConsConsolidationMember, error) {
	var rows []consdb.ConsConsolidationMember
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		rows, e = r.q.WithTx(tx).ListConsolidationMembers(ctx, groupID)
		return e
	})
	return rows, err
}

func (r *ConsolidationRepository) CreateRule(ctx context.Context, scope []uuid.UUID, arg consdb.CreateEliminationRuleParams) (consdb.CreateEliminationRuleRow, error) {
	var out consdb.CreateEliminationRuleRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateEliminationRule(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "")
}

func (r *ConsolidationRepository) ListRules(ctx context.Context, scope []uuid.UUID, groupID *uuid.UUID, limit, offset int32) ([]consdb.ConsEliminationRule, int64, error) {
	var rows []consdb.ConsEliminationRule
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountEliminationRules(ctx, groupID); e != nil {
			return e
		}
		rows, e = q.ListEliminationRules(ctx, consdb.ListEliminationRulesParams{ConsolidationGroupID: groupID, Limit: limit, Offset: offset})
		return e
	})
	return rows, total, err
}

func (r *ConsolidationRepository) CreateRun(ctx context.Context, scope []uuid.UUID, arg consdb.CreateConsolidationRunParams) (consdb.CreateConsolidationRunRow, error) {
	var out consdb.CreateConsolidationRunRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateConsolidationRun(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "")
}

func (r *ConsolidationRepository) GetRunDetail(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (consdb.ConsConsolidationRun, []consdb.ConsEliminationEntry, []consdb.ConsConsolidationAdjustment, error) {
	var run consdb.ConsConsolidationRun
	var elims []consdb.ConsEliminationEntry
	var adjs []consdb.ConsConsolidationAdjustment
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		run, e = q.GetConsolidationRun(ctx, id)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("consolidation run not found")
		}
		if e != nil {
			return e
		}
		if elims, e = q.ListEliminationEntriesByRun(ctx, id); e != nil {
			return e
		}
		adjs, e = q.ListConsolidationAdjustmentsByRun(ctx, id)
		return e
	})
	return run, elims, adjs, err
}

func (r *ConsolidationRepository) ListRuns(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]consdb.ConsConsolidationRun, int64, error) {
	var rows []consdb.ConsConsolidationRun
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountConsolidationRuns(ctx); e != nil {
			return e
		}
		rows, e = q.ListConsolidationRuns(ctx, consdb.ListConsolidationRunsParams{Limit: limit, Offset: offset})
		return e
	})
	return rows, total, err
}

func (r *ConsolidationRepository) PostEliminations(ctx context.Context, scope []uuid.UUID, runID uuid.UUID, rowVer int32, entries []consdb.CreateEliminationEntryParams, actor uuid.UUID) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		run, e := q.GetConsolidationRun(ctx, runID)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("consolidation run not found")
		}
		if e != nil {
			return e
		}
		if run.Status == "Finalized" {
			return httpx.Conflict("INVALID_STATE", "finalized consolidation run cannot be modified")
		}
		for _, ent := range entries {
			ent.ConsolidationRunID = runID
			if _, e = q.CreateEliminationEntry(ctx, ent); e != nil {
				return e
			}
		}
		next, e = q.TransitionConsolidationRun(ctx, consdb.TransitionConsolidationRunParams{
			ID: runID, RowVersion: rowVer, Status: "Eliminated", UpdatedBy: &actor,
		})
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.Conflict("STALE_ROW_VERSION", "rowVersion mismatch")
		}
		return e
	})
	return next, httpx.MapDBError(err, "")
}

func (r *ConsolidationRepository) PostAdjustments(ctx context.Context, scope []uuid.UUID, runID uuid.UUID, rowVer int32, adjs []consdb.CreateConsolidationAdjustmentParams, actor uuid.UUID) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		run, e := q.GetConsolidationRun(ctx, runID)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("consolidation run not found")
		}
		if e != nil {
			return e
		}
		if run.Status == "Finalized" {
			return httpx.Conflict("INVALID_STATE", "finalized consolidation run cannot be modified")
		}
		for _, adj := range adjs {
			adj.ConsolidationRunID = runID
			if _, e = q.CreateConsolidationAdjustment(ctx, adj); e != nil {
				return e
			}
		}
		next, e = q.TransitionConsolidationRun(ctx, consdb.TransitionConsolidationRunParams{
			ID: runID, RowVersion: rowVer, Status: "Adjusted", UpdatedBy: &actor,
		})
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.Conflict("STALE_ROW_VERSION", "rowVersion mismatch")
		}
		return e
	})
	return next, httpx.MapDBError(err, "")
}

func (r *ConsolidationRepository) FinalizeRunAndEmit(ctx context.Context, scope []uuid.UUID, runID uuid.UUID, rowVer int32, actor, corr uuid.UUID) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		run, e := q.GetConsolidationRun(ctx, runID)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("consolidation run not found")
		}
		if e != nil {
			return e
		}
		if run.Status == "Finalized" {
			return httpx.Conflict("INVALID_STATE", "consolidation run is already finalized")
		}
		// Verify double-entry balance across eliminations + adjustments before finalization.
		elims, e := q.ListEliminationEntriesByRun(ctx, runID)
		if e != nil {
			return e
		}
		adjs, e := q.ListConsolidationAdjustmentsByRun(ctx, runID)
		if e != nil {
			return e
		}
		dr, cr := new(big.Rat), new(big.Rat)
		for _, el := range elims {
			dr.Add(dr, platform.NumericToRat(el.DebitAmount))
			cr.Add(cr, platform.NumericToRat(el.CreditAmount))
		}
		for _, ad := range adjs {
			dr.Add(dr, platform.NumericToRat(ad.DebitAmount))
			cr.Add(cr, platform.NumericToRat(ad.CreditAmount))
		}
		if dr.Cmp(cr) != 0 {
			return httpx.Unprocessable("CONSOLIDATION_UNBALANCED", "elimination and adjustment debits must equal credits")
		}
		next, e = q.TransitionConsolidationRun(ctx, consdb.TransitionConsolidationRunParams{
			ID: runID, RowVersion: rowVer, Status: "Finalized", UpdatedBy: &actor,
		})
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.Conflict("STALE_ROW_VERSION", "rowVersion mismatch")
		}
		if e != nil {
			return e
		}
		return emit(ctx, q, "ConsolidationRun", "consolidation.run.finalized", run.ID, run.OrganizationID, corr, map[string]any{
			"consolidationGroupId": run.ConsolidationGroupID.String(),
			"fiscalPeriodRef":      run.FiscalPeriodRef.String(),
			"version":              run.Version,
			"totalDebit":           dr.FloatString(4),
			"totalCredit":          cr.FloatString(4),
			"status":               "Finalized",
		})
	})
	return next, httpx.MapDBError(err, "")
}

func (r *ConsolidationRepository) DrainAndPublish(ctx context.Context, limit int32, pub *eventbus.Publisher) (int, error) {
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
	if err != nil || len(rows) == 0 {
		return 0, err
	}
	published := 0
	var pubErr error
	for _, row := range rows {
		if e := pub.Publish(ctx, eventbus.Envelope{
			EventID: row.ID.String(), EventType: row.EventType, AggregateType: row.AggregateType,
			AggregateID: row.AggregateID.String(), OccurredAt: row.CreatedAt.Time,
			CorrelationID: row.CorrelationID.String(), OrganizationID: row.OrganizationID.String(),
			SchemaVersion: "v1", Data: json.RawMessage(row.Payload),
		}); e != nil {
			pubErr = e
			break
		}
		if e := q.MarkOutboxPublished(ctx, consdb.MarkOutboxPublishedParams{ID: row.ID, CreatedAt: row.CreatedAt}); e != nil {
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
