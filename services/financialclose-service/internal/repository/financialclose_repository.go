// Package repository implements RLS-enforced Financial Close checklists, closing runs,
// mandatory task gates, period status locks, SoD period reopenings, FX revaluation, and outbox relay.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/financialclose-service/internal/fcldb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type FXPositionInput struct {
	AccountRef    uuid.UUID `json:"accountRef"`
	CurrencyCode  string    `json:"currencyCode"`
	ForeignAmount string    `json:"foreignAmount"`
	BookRate      string    `json:"bookRate"`
	ClosingRate   string    `json:"closingRate"`
}

type FXRevaluationLine struct {
	AccountRef     uuid.UUID `json:"accountRef"`
	CurrencyCode   string    `json:"currencyCode"`
	ForeignAmount  string    `json:"foreignAmount"`
	BookRate       string    `json:"bookRate"`
	ClosingRate    string    `json:"closingRate"`
	UnrealizedGain string    `json:"unrealizedGainLoss"`
}

type FXRevaluationResult struct {
	ClosingRunID        uuid.UUID           `json:"closingRunId"`
	FiscalPeriodRef     uuid.UUID           `json:"fiscalPeriodRef"`
	TotalUnrealizedGain string              `json:"totalUnrealizedGainLoss"`
	Lines               []FXRevaluationLine `json:"lines"`
}

type FinancialCloseRepository struct {
	db *platform.DB
	q  *fcldb.Queries
}

func NewFinancialCloseRepository(db *platform.DB) *FinancialCloseRepository {
	var q *fcldb.Queries
	if db != nil && db.Pool != nil {
		q = fcldb.New(db.Pool)
	}
	return &FinancialCloseRepository{db: db, q: q}
}

func (r *FinancialCloseRepository) withOrgTx(ctx context.Context, scope []uuid.UUID, fn func(pgx.Tx) error) error {
	if len(scope) == 0 {
		return errors.New("org scope must not be empty")
	}
	return r.db.WithOrgScope(ctx, platform.ScopeWithSystemOrg(scope), fn)
}

func emit(ctx context.Context, q *fcldb.Queries, aggType, eventType string, id, org, corr uuid.UUID, payload map[string]any) error {
	payload["id"] = id.String()
	payload["organizationId"] = org.String()
	b, _ := json.Marshal(payload)
	return q.InsertOutbox(ctx, fcldb.InsertOutboxParams{
		AggregateType: aggType, AggregateID: id, EventType: eventType,
		Payload: b, OrganizationID: org, CorrelationID: corr,
	})
}

// ComputeFXRevaluation computes exact unrealized FX gain/loss per PSAK 10 / IAS 21:
// gainLoss = foreignAmount * (closingRate - bookRate).
func ComputeFXRevaluation(runID, periodID uuid.UUID, positions []FXPositionInput) (FXRevaluationResult, error) {
	lines := make([]FXRevaluationLine, 0, len(positions))
	total := new(big.Rat)
	for _, pos := range positions {
		if pos.AccountRef == uuid.Nil {
			return FXRevaluationResult{}, httpx.BadRequest("positions[].accountRef is required")
		}
		amtNum, err := platform.ParseNumeric(pos.ForeignAmount)
		if err != nil {
			return FXRevaluationResult{}, httpx.BadRequest("invalid foreignAmount")
		}
		bookNum, err := platform.ParseNumeric(pos.BookRate)
		if err != nil || platform.NumericToRat(bookNum).Sign() <= 0 {
			return FXRevaluationResult{}, httpx.BadRequest("bookRate must be > 0")
		}
		closeNum, err := platform.ParseNumeric(pos.ClosingRate)
		if err != nil || platform.NumericToRat(closeNum).Sign() <= 0 {
			return FXRevaluationResult{}, httpx.BadRequest("closingRate must be > 0")
		}
		diff := new(big.Rat).Sub(platform.NumericToRat(closeNum), platform.NumericToRat(bookNum))
		gain := new(big.Rat).Mul(platform.NumericToRat(amtNum), diff)
		gainNum, _ := platform.RatToNumeric(gain, 4)
		total.Add(total, platform.NumericToRat(gainNum))
		lines = append(lines, FXRevaluationLine{
			AccountRef:     pos.AccountRef,
			CurrencyCode:   pos.CurrencyCode,
			ForeignAmount:  platform.FormatNumeric(amtNum, 4),
			BookRate:       platform.FormatNumeric(bookNum, 8),
			ClosingRate:    platform.FormatNumeric(closeNum, 8),
			UnrealizedGain: platform.FormatNumeric(gainNum, 4),
		})
	}
	totNum, _ := platform.RatToNumeric(total, 4)
	return FXRevaluationResult{
		ClosingRunID:        runID,
		FiscalPeriodRef:     periodID,
		TotalUnrealizedGain: platform.FormatNumeric(totNum, 4),
		Lines:               lines,
	}, nil
}

func (r *FinancialCloseRepository) CreateTemplate(ctx context.Context, scope []uuid.UUID, arg fcldb.CreateChecklistTemplateParams, tasks []fcldb.CreateTaskTemplateParams) (fcldb.CreateChecklistTemplateRow, error) {
	var out fcldb.CreateChecklistTemplateRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if out, e = q.CreateChecklistTemplate(ctx, arg); e != nil {
			return e
		}
		for _, t := range tasks {
			t.TemplateID = out.ID
			if _, e = q.CreateTaskTemplate(ctx, t); e != nil {
				return e
			}
		}
		return nil
	})
	return out, httpx.MapDBError(err, "closing checklist template code already exists")
}

func (r *FinancialCloseRepository) ListTemplates(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]fcldb.FclClosingChecklistTemplate, int64, error) {
	var rows []fcldb.FclClosingChecklistTemplate
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountChecklistTemplates(ctx); e != nil {
			return e
		}
		rows, e = q.ListChecklistTemplates(ctx, fcldb.ListChecklistTemplatesParams{Limit: limit, Offset: offset})
		return e
	})
	return rows, total, err
}

func (r *FinancialCloseRepository) CreateClosingRun(ctx context.Context, scope []uuid.UUID, arg fcldb.CreateClosingRunParams, templateID *uuid.UUID) (fcldb.CreateClosingRunRow, error) {
	var out fcldb.CreateClosingRunRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		if templateID != nil && *templateID != uuid.Nil {
			var found uuid.UUID
			if e := tx.QueryRow(ctx, `SELECT id FROM fcl.closing_checklist_template WHERE id = $1 AND deleted_at IS NULL`, *templateID).Scan(&found); errors.Is(e, pgx.ErrNoRows) {
				return httpx.NotFound("closing checklist template not found")
			} else if e != nil {
				return e
			}
		}
		var e error
		if out, e = q.CreateClosingRun(ctx, arg); e != nil {
			return e
		}
		if templateID != nil && *templateID != uuid.Nil {
			tplTasks, e := q.ListTaskTemplatesByTemplate(ctx, *templateID)
			if e != nil {
				return e
			}
			for _, tt := range tplTasks {
				if _, e = q.CreateClosingTask(ctx, fcldb.CreateClosingTaskParams{
					OrganizationID: arg.OrganizationID, ClosingRunID: out.ID,
					TaskName: tt.TaskName, IsMandatory: tt.IsMandatory, CreatedBy: arg.CreatedBy,
				}); e != nil {
					return e
				}
			}
		}
		return nil
	})
	return out, httpx.MapDBError(err, "closing run already exists for period/type")
}

func (r *FinancialCloseRepository) GetClosingRun(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (fcldb.FclClosingRun, []fcldb.FclClosingTask, error) {
	var run fcldb.FclClosingRun
	var tasks []fcldb.FclClosingTask
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		run, e = q.GetClosingRun(ctx, id)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("closing run not found")
		}
		if e != nil {
			return e
		}
		tasks, e = q.ListClosingTasksByRun(ctx, id)
		return e
	})
	return run, tasks, err
}

func (r *FinancialCloseRepository) ListClosingRuns(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]fcldb.FclClosingRun, int64, error) {
	var rows []fcldb.FclClosingRun
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountClosingRuns(ctx, status); e != nil {
			return e
		}
		rows, e = q.ListClosingRuns(ctx, fcldb.ListClosingRunsParams{Status: status, Limit: limit, Offset: offset})
		return e
	})
	return rows, total, err
}

func (r *FinancialCloseRepository) CompleteTask(ctx context.Context, scope []uuid.UUID, id uuid.UUID, rowVer int32, actor uuid.UUID) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		next, e = r.q.WithTx(tx).CompleteClosingTask(ctx, fcldb.CompleteClosingTaskParams{ID: id, RowVersion: rowVer, UpdatedBy: &actor})
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.Conflict("STALE_ROW_VERSION", "task already completed or rowVersion mismatch")
		}
		return e
	})
	return next, httpx.MapDBError(err, "")
}

func (r *FinancialCloseRepository) RunFXRevaluationAndEmit(ctx context.Context, scope []uuid.UUID, runID uuid.UUID, positions []FXPositionInput, corr uuid.UUID) (FXRevaluationResult, error) {
	var res FXRevaluationResult
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		run, e := q.GetClosingRun(ctx, runID)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("closing run not found")
		}
		if e != nil {
			return e
		}
		if run.Status == "Locked" {
			return httpx.Conflict("PERIOD_LOCKED", "cannot run FX revaluation on a locked closing run")
		}
		res, e = ComputeFXRevaluation(run.ID, run.FiscalPeriodRef, positions)
		if e != nil {
			return e
		}
		return emit(ctx, q, "ClosingRun", "financialclose.fx_revaluation.completed", run.ID, run.OrganizationID, corr, map[string]any{
			"fiscalPeriodId":          run.FiscalPeriodRef.String(),
			"totalUnrealizedGainLoss": res.TotalUnrealizedGain,
			"lineCount":               len(res.Lines),
		})
	})
	return res, httpx.MapDBError(err, "")
}

func (r *FinancialCloseRepository) TransitionClosingRunAndEmit(ctx context.Context, scope []uuid.UUID, id uuid.UUID, rowVer int32, targetStatus string, actor, corr uuid.UUID) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		run, e := q.GetClosingRun(ctx, id)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("closing run not found")
		}
		if e != nil {
			return e
		}
		valid := (targetStatus == "Closed" && (run.Status == "Open" || run.Status == "PendingClose")) ||
			(targetStatus == "Locked" && run.Status == "Closed")
		if !valid {
			return httpx.Conflict("INVALID_STATE", fmt.Sprintf("cannot transition closing run from %s to %s", run.Status, targetStatus))
		}
		incomplete, e := q.CountIncompleteMandatoryTasks(ctx, id)
		if e != nil {
			return e
		}
		if incomplete > 0 {
			return httpx.Unprocessable("MANDATORY_TASKS_INCOMPLETE", fmt.Sprintf("%d mandatory closing tasks remain incomplete", incomplete))
		}
		next, e = q.TransitionClosingRun(ctx, fcldb.TransitionClosingRunParams{
			ID: id, RowVersion: rowVer, Status: targetStatus, UpdatedBy: &actor,
		})
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.Conflict("STALE_ROW_VERSION", "rowVersion mismatch")
		}
		if e != nil {
			return e
		}
		// Sync local fcl.period_status atomically.
		if _, e = q.UpsertPeriodStatus(ctx, fcldb.UpsertPeriodStatusParams{
			OrganizationID: run.OrganizationID, FinancialOrganizationRef: run.FinancialOrganizationRef,
			FiscalPeriodRef: run.FiscalPeriodRef, Module: "GL", Status: targetStatus, CreatedBy: actor,
		}); e != nil {
			return e
		}
		evType := "financialclose.period.closed"
		if targetStatus == "Locked" {
			evType = "financialclose.period.locked"
		}
		return emit(ctx, q, "ClosingRun", evType, run.ID, run.OrganizationID, corr, map[string]any{
			"fiscalPeriodId":           run.FiscalPeriodRef.String(),
			"financialOrganizationRef": run.FinancialOrganizationRef.String(),
			"runType":                  run.RunType,
			"status":                   targetStatus,
		})
	})
	return next, httpx.MapDBError(err, "")
}

func (r *FinancialCloseRepository) UpsertPeriodStatus(ctx context.Context, scope []uuid.UUID, arg fcldb.UpsertPeriodStatusParams) (fcldb.UpsertPeriodStatusRow, error) {
	var out fcldb.UpsertPeriodStatusRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var curStatus string
		err := tx.QueryRow(ctx, `
			SELECT status FROM fcl.period_status
			WHERE organization_id = $1 AND financial_organization_ref = $2 AND fiscal_period_ref = $3 AND module = $4 AND deleted_at IS NULL
		`, arg.OrganizationID, arg.FinancialOrganizationRef, arg.FiscalPeriodRef, arg.Module).Scan(&curStatus)
		if err == nil && curStatus == "Locked" && arg.Status != "Locked" {
			return httpx.Conflict("PERIOD_LOCKED", "locked period status cannot be downgraded without an approved reopening")
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		var e error
		out, e = r.q.WithTx(tx).UpsertPeriodStatus(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "")
}

func (r *FinancialCloseRepository) ListPeriodStatuses(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]fcldb.FclPeriodStatus, int64, error) {
	var rows []fcldb.FclPeriodStatus
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountPeriodStatuses(ctx); e != nil {
			return e
		}
		rows, e = q.ListPeriodStatuses(ctx, fcldb.ListPeriodStatusesParams{Limit: limit, Offset: offset})
		return e
	})
	return rows, total, err
}

func (r *FinancialCloseRepository) ListReopenings(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]fcldb.FclPeriodReopening, int64, error) {
	var rows []fcldb.FclPeriodReopening
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountPeriodReopenings(ctx); e != nil {
			return e
		}
		rows, e = q.ListPeriodReopenings(ctx, fcldb.ListPeriodReopeningsParams{Limit: limit, Offset: offset})
		return e
	})
	return rows, total, err
}

func (r *FinancialCloseRepository) RequestReopening(ctx context.Context, scope []uuid.UUID, arg fcldb.CreatePeriodReopeningParams) (fcldb.CreatePeriodReopeningRow, error) {
	var out fcldb.CreatePeriodReopeningRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreatePeriodReopening(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "")
}

func (r *FinancialCloseRepository) ApproveReopeningAndEmit(ctx context.Context, scope []uuid.UUID, id uuid.UUID, rowVer int32, approver, corr uuid.UUID) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		req, e := q.GetPeriodReopening(ctx, id)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("period reopening request not found")
		}
		if e != nil {
			return e
		}
		// Enforce Segregation of Duties (BR-664): requester cannot approve own reopening.
		if req.RequestedBy == approver {
			return httpx.Forbidden("segregation of duties: requester cannot approve own period reopening")
		}
		next, e = q.ApprovePeriodReopening(ctx, fcldb.ApprovePeriodReopeningParams{
			ID: id, RowVersion: rowVer, ApprovedBy: &approver,
		})
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.Conflict("STALE_ROW_VERSION", "already approved or rowVersion mismatch")
		}
		if e != nil {
			return e
		}
		// Sync local fcl.period_status back to Open.
		if _, e = tx.Exec(ctx, `
			UPDATE fcl.period_status
			SET status = 'Open', updated_by = $4
			WHERE organization_id = $1 AND financial_organization_ref = $2 AND fiscal_period_ref = $3 AND deleted_at IS NULL
		`, req.OrganizationID, req.FinancialOrganizationRef, req.FiscalPeriodRef, approver); e != nil {
			return e
		}
		return emit(ctx, q, "PeriodReopening", "financialclose.period.reopened", req.ID, req.OrganizationID, corr, map[string]any{
			"fiscalPeriodId":           req.FiscalPeriodRef.String(),
			"financialOrganizationRef": req.FinancialOrganizationRef.String(),
			"reason":                   req.Reason,
			"status":                   "Open",
		})
	})
	return next, httpx.MapDBError(err, "")
}

func (r *FinancialCloseRepository) DrainAndPublish(ctx context.Context, limit int32, pub *eventbus.Publisher) (int, error) {
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
		if e := q.MarkOutboxPublished(ctx, fcldb.MarkOutboxPublishedParams{ID: row.ID, CreatedAt: row.CreatedAt}); e != nil {
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
