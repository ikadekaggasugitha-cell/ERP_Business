// Package repository implements RLS-enforced Budget, Line, Control Policy,
// Commitment persistence, exact *big.Rat availability checks, and outbox relay.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/budget-service/internal/budgetdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type BudgetCheckResult struct {
	BudgetID         uuid.UUID `json:"budgetId"`
	TotalBudget      string    `json:"totalBudget"`
	TotalCommitted   string    `json:"totalCommitted"`
	AvailableAmount  string    `json:"availableAmount"`
	RequestedAmount  string    `json:"requestedAmount"`
	ControlType      string    `json:"controlType"`
	Allowed          bool      `json:"allowed"`
	Warning          bool      `json:"warning"`
	RequiresApproval bool      `json:"requiresApproval"`
}

type BudgetRepository struct {
	db *platform.DB
	q  *budgetdb.Queries
}

func NewBudgetRepository(db *platform.DB) *BudgetRepository {
	var q *budgetdb.Queries
	if db != nil && db.Pool != nil {
		q = budgetdb.New(db.Pool)
	}
	return &BudgetRepository{db: db, q: q}
}

func (r *BudgetRepository) withOrgTx(ctx context.Context, scope []uuid.UUID, fn func(pgx.Tx) error) error {
	if len(scope) == 0 {
		return errors.New("org scope must not be empty")
	}
	return r.db.WithOrgScope(ctx, platform.ScopeWithSystemOrg(scope), fn)
}

func emit(ctx context.Context, q *budgetdb.Queries, aggType, eventType string, id, org, corr uuid.UUID, payload map[string]any) error {
	payload["id"] = id.String()
	payload["organizationId"] = org.String()
	b, _ := json.Marshal(payload)
	return q.InsertOutbox(ctx, budgetdb.InsertOutboxParams{
		AggregateType: aggType, AggregateID: id, EventType: eventType,
		Payload: b, OrganizationID: org, CorrelationID: corr,
	})
}

// EvaluateBudgetPolicy evaluates requested vs available budget using exact *big.Rat math.
func EvaluateBudgetPolicy(budgetID uuid.UUID, totalBudget, totalCommitted, requested *big.Rat, controlType string) BudgetCheckResult {
	avail := new(big.Rat).Sub(totalBudget, totalCommitted)
	exceeded := requested.Cmp(avail) > 0
	if controlType == "" {
		controlType = "HardStop"
	}
	res := BudgetCheckResult{
		BudgetID:        budgetID,
		TotalBudget:     totalBudget.FloatString(4),
		TotalCommitted:  totalCommitted.FloatString(4),
		AvailableAmount: avail.FloatString(4),
		RequestedAmount: requested.FloatString(4),
		ControlType:     controlType,
		Allowed:         true,
	}
	if exceeded {
		switch controlType {
		case "HardStop":
			res.Allowed = false
		case "SoftWarning":
			res.Allowed = true
			res.Warning = true
		case "ApprovalOverride":
			res.Allowed = false
			res.RequiresApproval = true
		}
	}
	return res
}

func (r *BudgetRepository) CreateBudget(ctx context.Context, scope []uuid.UUID, arg budgetdb.CreateBudgetParams) (budgetdb.CreateBudgetRow, error) {
	var out budgetdb.CreateBudgetRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateBudget(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "budget code/version already exists")
}

func (r *BudgetRepository) GetBudget(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (budgetdb.BudBudget, []budgetdb.BudBudgetLine, error) {
	var b budgetdb.BudBudget
	var lines []budgetdb.BudBudgetLine
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		b, e = q.GetBudget(ctx, id)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("budget not found")
		}
		if e != nil {
			return e
		}
		lines, e = q.ListBudgetLines(ctx, id)
		return e
	})
	return b, lines, err
}

func (r *BudgetRepository) ListBudgets(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]budgetdb.BudBudget, int64, error) {
	var rows []budgetdb.BudBudget
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountBudgets(ctx, status); e != nil {
			return e
		}
		rows, e = q.ListBudgets(ctx, budgetdb.ListBudgetsParams{Status: status, Limit: limit, Offset: offset})
		return e
	})
	return rows, total, err
}

func (r *BudgetRepository) DeleteDraftBudget(ctx context.Context, scope []uuid.UUID, arg budgetdb.DeleteDraftBudgetParams) error {
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		b, e := q.GetBudget(ctx, arg.ID)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("budget not found")
		}
		if e != nil {
			return e
		}
		if b.Status != "Draft" {
			return httpx.Conflict("INVALID_STATE", "only Draft budget can be deleted")
		}
		n, e := q.DeleteDraftBudget(ctx, arg)
		if e != nil {
			return e
		}
		if n == 0 {
			return httpx.Conflict("STALE_ROW_VERSION", "rowVersion mismatch")
		}
		return nil
	})
	return httpx.MapDBError(err, "")
}

func (r *BudgetRepository) TransitionBudgetAndEmit(ctx context.Context, scope []uuid.UUID, id uuid.UUID, rowVer int32, targetStatus string, actor, corr uuid.UUID) (int32, error) {
	var newVer int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		b, e := q.GetBudget(ctx, id)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("budget not found")
		}
		if e != nil {
			return e
		}
		valid := (targetStatus == "Submitted" && b.Status == "Draft") ||
			(targetStatus == "Approved" && (b.Status == "Draft" || b.Status == "Submitted")) ||
			(targetStatus == "Closed" && b.Status == "Approved")
		if !valid {
			return httpx.Conflict("INVALID_STATE", fmt.Sprintf("cannot transition budget from %s to %s", b.Status, targetStatus))
		}
		active := targetStatus == "Approved"
		if active {
			if _, e = tx.Exec(ctx, `
				UPDATE bud.budget
				SET is_active = FALSE, updated_by = $4
				WHERE organization_id = $1 AND financial_organization_ref = $2 AND fiscal_year_ref = $3
				  AND id <> $5 AND is_active = TRUE AND deleted_at IS NULL
			`, b.OrganizationID, b.FinancialOrganizationRef, b.FiscalYearRef, actor, b.ID); e != nil {
				return e
			}
		}
		newVer, e = q.TransitionBudget(ctx, budgetdb.TransitionBudgetParams{
			ID: id, RowVersion: rowVer, Status: targetStatus, IsActive: active, UpdatedBy: &actor,
		})
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.Conflict("STALE_ROW_VERSION", "rowVersion mismatch")
		}
		if e != nil {
			return e
		}
		if targetStatus == "Approved" {
			return emit(ctx, q, "Budget", "budget.approved", b.ID, b.OrganizationID, corr, map[string]any{
				"code": b.Code, "version": b.Version, "status": targetStatus,
			})
		}
		return nil
	})
	return newVer, httpx.MapDBError(err, "")
}

func (r *BudgetRepository) AddBudgetLine(ctx context.Context, scope []uuid.UUID, arg budgetdb.CreateBudgetLineParams) (budgetdb.CreateBudgetLineRow, error) {
	var out budgetdb.CreateBudgetLineRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateBudgetLine(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "")
}

func (r *BudgetRepository) CreatePolicy(ctx context.Context, scope []uuid.UUID, arg budgetdb.CreateBudgetPolicyParams) (budgetdb.CreateBudgetPolicyRow, error) {
	var out budgetdb.CreateBudgetPolicyRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateBudgetPolicy(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "")
}

func (r *BudgetRepository) ListPolicies(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]budgetdb.BudBudgetControlPolicy, int64, error) {
	var rows []budgetdb.BudBudgetControlPolicy
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountBudgetPolicies(ctx); e != nil {
			return e
		}
		rows, e = q.ListBudgetPolicies(ctx, budgetdb.ListBudgetPoliciesParams{Limit: limit, Offset: offset})
		return e
	})
	return rows, total, err
}

func (r *BudgetRepository) CheckAvailability(ctx context.Context, scope []uuid.UUID, budgetID uuid.UUID, accountRef *uuid.UUID, periodNo *int32, txType *string, requested *big.Rat) (BudgetCheckResult, error) {
	var res BudgetCheckResult
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		b, e := q.GetBudget(ctx, budgetID)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("budget not found")
		}
		if e != nil {
			return e
		}
		totBud, e := q.SumBudgetLines(ctx, budgetdb.SumBudgetLinesParams{BudgetID: b.ID, AccountRef: accountRef, PeriodNo: periodNo})
		if e != nil {
			return e
		}
		totCom, e := q.SumActiveCommitments(ctx, b.ID)
		if e != nil {
			return e
		}
		ctrl, ce := q.ResolveBudgetControlType(ctx, budgetdb.ResolveBudgetControlTypeParams{
			OrganizationID: b.OrganizationID, AccountRef: accountRef, TransactionType: txType,
		})
		if ce != nil && !errors.Is(ce, pgx.ErrNoRows) {
			return ce
		}
		res = EvaluateBudgetPolicy(b.ID, platform.NumericToRat(totBud), platform.NumericToRat(totCom), requested, ctrl)
		return nil
	})
	return res, err
}

func (r *BudgetRepository) ListCommitments(ctx context.Context, scope []uuid.UUID, budgetID *uuid.UUID, limit, offset int32) ([]budgetdb.BudBudgetCommitment, int64, error) {
	var rows []budgetdb.BudBudgetCommitment
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountBudgetCommitments(ctx, budgetID); e != nil {
			return e
		}
		rows, e = q.ListBudgetCommitments(ctx, budgetdb.ListBudgetCommitmentsParams{
			BudgetID: budgetID, Limit: limit, Offset: offset,
		})
		return e
	})
	return rows, total, err
}

func (r *BudgetRepository) CreateCommitmentAndEmit(ctx context.Context, scope []uuid.UUID, arg budgetdb.CreateBudgetCommitmentParams, corr uuid.UUID) (budgetdb.CreateBudgetCommitmentRow, error) {
	var out budgetdb.CreateBudgetCommitmentRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		b, e := q.GetBudget(ctx, arg.BudgetID)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("budget not found")
		}
		if e != nil {
			return e
		}
		if b.Status != "Approved" {
			return httpx.Conflict("BUDGET_NOT_APPROVED", "commitments can only be created against Approved budgets")
		}
		totBud, e := q.SumBudgetLines(ctx, budgetdb.SumBudgetLinesParams{BudgetID: b.ID})
		if e != nil {
			return e
		}
		totCom, e := q.SumActiveCommitments(ctx, b.ID)
		if e != nil {
			return e
		}
		if platform.NumericToRat(totBud).Sign() > 0 {
			ctrl, ce := q.ResolveBudgetControlType(ctx, budgetdb.ResolveBudgetControlTypeParams{
				OrganizationID: b.OrganizationID,
			})
			if ce != nil && !errors.Is(ce, pgx.ErrNoRows) {
				return ce
			}
			eval := EvaluateBudgetPolicy(b.ID, platform.NumericToRat(totBud), platform.NumericToRat(totCom), platform.NumericToRat(arg.CommittedAmount), ctrl)
			if !eval.Allowed {
				return httpx.Unprocessable("BUDGET_EXCEEDED", "requested commitment exceeds available budget ("+eval.ControlType+")")
			}
		}
		out, e = q.CreateBudgetCommitment(ctx, arg)
		if e != nil {
			return e
		}
		return emit(ctx, q, "BudgetCommitment", "budget.commitment.created", out.ID, arg.OrganizationID, corr, map[string]any{
			"budgetId": arg.BudgetID.String(), "sourceDocumentRef": arg.SourceDocumentRef,
			"amount": platform.FormatNumeric(arg.CommittedAmount, 4),
		})
	})
	return out, httpx.MapDBError(err, "")
}

func (r *BudgetRepository) TransitionCommitment(ctx context.Context, scope []uuid.UUID, id uuid.UUID, rowVer int32, status string, actor uuid.UUID) (int32, error) {
	var newVer int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		newVer, e = r.q.WithTx(tx).TransitionBudgetCommitment(ctx, budgetdb.TransitionBudgetCommitmentParams{
			ID: id, RowVersion: rowVer, Status: status, UpdatedBy: &actor,
		})
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.Conflict("STALE_ROW_VERSION", "commitment not Open or rowVersion mismatch")
		}
		return e
	})
	return newVer, httpx.MapDBError(err, "")
}

func (r *BudgetRepository) ConsumePOApproved(ctx context.Context, scope []uuid.UUID, mark budgetdb.MarkEventProcessedParams, orgID, actor, corr uuid.UUID, docRef string, amount pgtype.Numeric) (bool, error) {
	applied := false
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		n, e := q.MarkEventProcessed(ctx, mark)
		if e != nil || n == 0 {
			return e
		}
		b, be := q.GetActiveApprovedBudget(ctx, orgID)
		if errors.Is(be, pgx.ErrNoRows) {
			// No active approved budget configured for this organization yet; record event idempotently.
			applied = true
			return nil
		}
		if be != nil {
			return be
		}
		row, e := q.CreateBudgetCommitment(ctx, budgetdb.CreateBudgetCommitmentParams{
			OrganizationID: orgID, BudgetID: b.ID, SourceDocumentRef: docRef, CommittedAmount: amount, CreatedBy: actor,
		})
		if e != nil {
			return e
		}
		if e = emit(ctx, q, "BudgetCommitment", "budget.commitment.created", row.ID, orgID, corr, map[string]any{
			"budgetId": b.ID.String(), "sourceDocumentRef": docRef, "amount": platform.FormatNumeric(amount, 4),
		}); e != nil {
			return e
		}
		applied = true
		return nil
	})
	return applied, err
}

func (r *BudgetRepository) ConsumeAPInvoiceRecorded(ctx context.Context, scope []uuid.UUID, mark budgetdb.MarkEventProcessedParams, orgID, actor uuid.UUID, sourceRef string) (bool, error) {
	applied := false
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		n, e := q.MarkEventProcessed(ctx, mark)
		if e != nil || n == 0 {
			return e
		}
		if sourceRef != "" {
			if _, e = tx.Exec(ctx, `
				UPDATE bud.budget_commitment
				SET status = 'Consumed', updated_by = $3
				WHERE organization_id = $1 AND source_document_ref = $2 AND status = 'Open' AND deleted_at IS NULL
			`, orgID, sourceRef, actor); e != nil {
				return e
			}
		}
		applied = true
		return nil
	})
	return applied, err
}

func (r *BudgetRepository) DrainAndPublish(ctx context.Context, limit int32, pub *eventbus.Publisher) (int, error) {
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
		if e := q.MarkOutboxPublished(ctx, budgetdb.MarkOutboxPublishedParams{ID: row.ID, CreatedAt: row.CreatedAt}); e != nil {
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
