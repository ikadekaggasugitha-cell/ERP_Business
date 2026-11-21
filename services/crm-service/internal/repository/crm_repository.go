// Package repository implements RLS-enforced CRM persistence (leads, opportunities,
// customer segments, interactions, support cases, activity tasks, and outbox).
package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/crm-service/internal/crmdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type CRMRepository struct {
	db *platform.DB
	q  *crmdb.Queries
}

func NewCRMRepository(db *platform.DB) *CRMRepository {
	var q *crmdb.Queries
	if db != nil && db.Pool != nil {
		q = crmdb.New(db.Pool)
	}
	return &CRMRepository{db: db, q: q}
}

func (r *CRMRepository) withOrgTx(ctx context.Context, scope []uuid.UUID, fn func(pgx.Tx) error) error {
	if len(scope) == 0 {
		return errors.New("org scope must not be empty")
	}
	return r.db.WithOrgScope(ctx, platform.ScopeWithSystemOrg(scope), fn)
}

func emit(ctx context.Context, q *crmdb.Queries, aggType, eventType string, id, org, corr uuid.UUID, payload map[string]any) error {
	payload["id"] = id.String()
	payload["organizationId"] = org.String()
	b, _ := json.Marshal(payload)
	return q.InsertOutbox(ctx, crmdb.InsertOutboxParams{
		AggregateType: aggType, AggregateID: id, EventType: eventType,
		Payload: b, OrganizationID: org, CorrelationID: corr,
	})
}

func (r *CRMRepository) CreateLead(ctx context.Context, scope []uuid.UUID, arg crmdb.CreateLeadParams) (crmdb.CreateLeadRow, error) {
	var out crmdb.CreateLeadRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateLead(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "lead code already exists")
}

func (r *CRMRepository) GetLead(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (crmdb.CrmLead, error) {
	var out crmdb.CrmLead
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).GetLead(ctx, id)
		return e
	})
	return out, httpx.MapDBError(err, "lead not found")
}

func (r *CRMRepository) ListLeads(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]crmdb.CrmLead, int64, error) {
	var rows []crmdb.CrmLead
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountLeads(ctx, status); e != nil {
			return e
		}
		rows, e = q.ListLeads(ctx, crmdb.ListLeadsParams{Limit: limit, Offset: offset, Status: status})
		return e
	})
	return rows, total, err
}

func (r *CRMRepository) UpdateLead(ctx context.Context, scope []uuid.UUID, arg crmdb.UpdateLeadParams) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		next, e = r.q.WithTx(tx).UpdateLead(ctx, arg)
		return e
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, httpx.Conflict("VERSION_CONFLICT", "lead not found or version mismatch")
	}
	return next, httpx.MapDBError(err, "lead update conflict")
}

func (r *CRMRepository) TransitionLead(ctx context.Context, scope []uuid.UUID, id, org, actor, corr uuid.UUID, version int32, fromStatus, toStatus string, customerRef *uuid.UUID) (int32, *uuid.UUID, error) {
	var next int32
	var createdOppID *uuid.UUID
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		lead, e := q.GetLead(ctx, id)
		if e != nil {
			return httpx.NotFound("lead not found")
		}
		next, e = q.TransitionLeadStatus(ctx, crmdb.TransitionLeadStatusParams{
			ID: id, Status: toStatus, CustomerRef: customerRef, UpdatedBy: &actor, RowVersion: version, Status_2: fromStatus,
		})
		if e != nil {
			return e
		}
		if toStatus == "Qualified" {
			return emit(ctx, q, "Lead", "crm.lead.qualified", id, org, corr, map[string]any{
				"leadId":         id.String(),
				"code":           lead.Code,
				"name":           lead.Name,
				"estimatedValue": platform.FormatNumeric(lead.EstimatedValue, 4),
			})
		}
		if toStatus == "Converted" {
			effCust := lead.CustomerRef
			if customerRef != nil {
				effCust = customerRef
			}
			prob, _ := platform.ParseNumeric("50.00")
			opp, oe := q.CreateOpportunity(ctx, crmdb.CreateOpportunityParams{
				OrganizationID: org,
				Code:           "OPP-" + lead.Code,
				LeadID:         &lead.ID,
				CustomerRef:    effCust,
				Stage:          "Qualification",
				ProbabilityPct: prob,
				EstimatedValue: lead.EstimatedValue,
				CreatedBy:      actor,
			})
			if oe != nil {
				return oe
			}
			oppID := opp.ID
			createdOppID = &oppID
			payload := map[string]any{
				"leadId":        id.String(),
				"opportunityId": opp.ID.String(),
				"code":          lead.Code,
			}
			if effCust != nil {
				payload["customerId"] = effCust.String()
			}
			return emit(ctx, q, "Lead", "crm.lead.converted", id, org, corr, payload)
		}
		return nil
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil, httpx.Conflict("INVALID_STATE", "lead state or version mismatch")
	}
	return next, createdOppID, httpx.MapDBError(err, "lead transition conflict")
}

func (r *CRMRepository) LoseLead(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		next, e = r.q.WithTx(tx).LoseLead(ctx, crmdb.LoseLeadParams{ID: id, UpdatedBy: &actor, RowVersion: version})
		return e
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, httpx.Conflict("INVALID_STATE", "lead must be New or Qualified with matching rowVersion")
	}
	return next, err
}

func (r *CRMRepository) DeleteLead(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) error {
	var n int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		n, e = r.q.WithTx(tx).SoftDeleteLead(ctx, crmdb.SoftDeleteLeadParams{ID: id, UpdatedBy: &actor, RowVersion: version})
		return e
	})
	if err != nil {
		return httpx.MapDBError(err, "cannot delete lead")
	}
	if n == 0 {
		return httpx.Conflict("VERSION_CONFLICT", "lead not found or version mismatch")
	}
	return nil
}

func (r *CRMRepository) CreateOpportunity(ctx context.Context, scope []uuid.UUID, arg crmdb.CreateOpportunityParams) (crmdb.CreateOpportunityRow, error) {
	var out crmdb.CreateOpportunityRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateOpportunity(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "opportunity code already exists")
}

func (r *CRMRepository) GetOpportunity(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (crmdb.CrmOpportunity, error) {
	var out crmdb.CrmOpportunity
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).GetOpportunity(ctx, id)
		return e
	})
	return out, httpx.MapDBError(err, "opportunity not found")
}

func (r *CRMRepository) ListOpportunities(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]crmdb.CrmOpportunity, int64, error) {
	var rows []crmdb.CrmOpportunity
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountOpportunities(ctx, status); e != nil {
			return e
		}
		rows, e = q.ListOpportunities(ctx, crmdb.ListOpportunitiesParams{Limit: limit, Offset: offset, Status: status})
		return e
	})
	return rows, total, err
}

func (r *CRMRepository) UpdateOpportunity(ctx context.Context, scope []uuid.UUID, arg crmdb.UpdateOpportunityParams) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		next, e = r.q.WithTx(tx).UpdateOpportunity(ctx, arg)
		return e
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, httpx.Conflict("INVALID_STATE", "only Open opportunity with matching rowVersion can be updated")
	}
	return next, httpx.MapDBError(err, "opportunity update conflict")
}

func (r *CRMRepository) WinOpportunityAndEmit(ctx context.Context, scope []uuid.UUID, id, org, actor, corr uuid.UUID, version int32, currencyCode string) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		opp, e := q.GetOpportunity(ctx, id)
		if e != nil {
			return httpx.NotFound("opportunity not found")
		}
		if opp.Status != "Open" {
			return httpx.Conflict("INVALID_STATE", "only Open opportunity can be marked Won")
		}
		if opp.CustomerRef == nil {
			return httpx.Unprocessable("CUSTOMER_REQUIRED", "opportunity must have customerRef before marking Won")
		}
		if ev := platform.NumericToRat(opp.EstimatedValue); ev == nil || ev.Sign() <= 0 {
			return httpx.Unprocessable("POSITIVE_VALUE_REQUIRED", "opportunity must have positive estimatedValue before marking Won")
		}
		next, e = q.CloseOpportunity(ctx, crmdb.CloseOpportunityParams{ID: id, Status: "Won", UpdatedBy: &actor, RowVersion: version})
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.Conflict("VERSION_CONFLICT", "opportunity version mismatch")
		}
		if e != nil {
			return e
		}
		if currencyCode == "" {
			currencyCode = "IDR"
		}
		payload := map[string]any{
			"opportunityId":  id.String(),
			"code":           opp.Code,
			"stage":          opp.Stage,
			"estimatedValue": platform.FormatNumeric(opp.EstimatedValue, 4),
			"currencyCode":   currencyCode,
		}
		if opp.CustomerRef != nil {
			payload["customerId"] = opp.CustomerRef.String()
		}
		return emit(ctx, q, "Opportunity", "crm.opportunity.won", id, org, corr, payload)
	})
	return next, err
}

func (r *CRMRepository) LoseOpportunity(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		next, e = r.q.WithTx(tx).CloseOpportunity(ctx, crmdb.CloseOpportunityParams{ID: id, Status: "Lost", UpdatedBy: &actor, RowVersion: version})
		return e
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, httpx.Conflict("INVALID_STATE", "only Open opportunity with matching rowVersion can be marked Lost")
	}
	return next, err
}

func (r *CRMRepository) CreateSegment(ctx context.Context, scope []uuid.UUID, arg crmdb.CreateCustomerSegmentParams) (crmdb.CreateCustomerSegmentRow, error) {
	var out crmdb.CreateCustomerSegmentRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateCustomerSegment(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "segment code already exists")
}

func (r *CRMRepository) ListSegments(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]crmdb.CrmCustomerSegment, int64, error) {
	var rows []crmdb.CrmCustomerSegment
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountCustomerSegments(ctx); e != nil {
			return e
		}
		rows, e = q.ListCustomerSegments(ctx, crmdb.ListCustomerSegmentsParams{Limit: limit, Offset: offset})
		return e
	})
	return rows, total, err
}

func (r *CRMRepository) UpdateSegment(ctx context.Context, scope []uuid.UUID, arg crmdb.UpdateCustomerSegmentParams) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		cur, e := q.GetCustomerSegment(ctx, arg.ID)
		if e != nil {
			return httpx.NotFound("segment not found")
		}
		if cur.OrganizationID == platform.SystemOrgID {
			return httpx.Forbidden("system customer segment is read-only")
		}
		next, e = q.UpdateCustomerSegment(ctx, arg)
		return e
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, httpx.Conflict("VERSION_CONFLICT", "segment not found or version mismatch")
	}
	return next, err
}

func (r *CRMRepository) DeleteSegment(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) error {
	var n int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		cur, e := q.GetCustomerSegment(ctx, id)
		if e != nil {
			return httpx.NotFound("segment not found")
		}
		if cur.OrganizationID == platform.SystemOrgID {
			return httpx.Forbidden("system customer segment cannot be deleted")
		}
		n, e = q.SoftDeleteCustomerSegment(ctx, crmdb.SoftDeleteCustomerSegmentParams{ID: id, UpdatedBy: &actor, RowVersion: version})
		return e
	})
	if err != nil {
		return httpx.MapDBError(err, "cannot delete segment")
	}
	if n == 0 {
		return httpx.Conflict("VERSION_CONFLICT", "segment not found or version mismatch")
	}
	return nil
}

func (r *CRMRepository) CreateInteraction(ctx context.Context, scope []uuid.UUID, arg crmdb.CreateInteractionParams) (crmdb.CreateInteractionRow, error) {
	var out crmdb.CreateInteractionRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateInteraction(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "failed to create interaction")
}

func (r *CRMRepository) ListInteractions(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]crmdb.CrmInteraction, int64, error) {
	var rows []crmdb.CrmInteraction
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountInteractions(ctx); e != nil {
			return e
		}
		rows, e = q.ListInteractions(ctx, crmdb.ListInteractionsParams{Limit: limit, Offset: offset})
		return e
	})
	return rows, total, err
}

func (r *CRMRepository) CreateSupportCase(ctx context.Context, scope []uuid.UUID, arg crmdb.CreateSupportCaseParams) (crmdb.CreateSupportCaseRow, error) {
	var out crmdb.CreateSupportCaseRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateSupportCase(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "support case number already exists")
}

func (r *CRMRepository) GetSupportCase(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (crmdb.CrmSupportCase, error) {
	var out crmdb.CrmSupportCase
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).GetSupportCase(ctx, id)
		return e
	})
	return out, httpx.MapDBError(err, "support case not found")
}

func (r *CRMRepository) ListSupportCases(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]crmdb.CrmSupportCase, int64, error) {
	var rows []crmdb.CrmSupportCase
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountSupportCases(ctx, status); e != nil {
			return e
		}
		rows, e = q.ListSupportCases(ctx, crmdb.ListSupportCasesParams{Limit: limit, Offset: offset, Status: status})
		return e
	})
	return rows, total, err
}

func (r *CRMRepository) EscalateSupportCase(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		next, e = r.q.WithTx(tx).EscalateSupportCase(ctx, crmdb.EscalateSupportCaseParams{ID: id, UpdatedBy: &actor, RowVersion: version})
		return e
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, httpx.Conflict("INVALID_STATE", "support case must be Open with matching rowVersion")
	}
	return next, err
}

func (r *CRMRepository) ResolveSupportCase(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32, resolution *string) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		next, e = r.q.WithTx(tx).ResolveSupportCase(ctx, crmdb.ResolveSupportCaseParams{ID: id, Resolution: resolution, UpdatedBy: &actor, RowVersion: version})
		return e
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, httpx.Conflict("INVALID_STATE", "support case must be Open or Escalated with matching rowVersion")
	}
	return next, err
}

func (r *CRMRepository) CloseSupportCase(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		next, e = r.q.WithTx(tx).CloseSupportCase(ctx, crmdb.CloseSupportCaseParams{ID: id, UpdatedBy: &actor, RowVersion: version})
		return e
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, httpx.Conflict("INVALID_STATE", "support case must be Resolved with matching rowVersion")
	}
	return next, err
}

func (r *CRMRepository) CreateActivityTask(ctx context.Context, scope []uuid.UUID, arg crmdb.CreateActivityTaskParams) (crmdb.CreateActivityTaskRow, error) {
	var out crmdb.CreateActivityTaskRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateActivityTask(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "failed to create activity task")
}

func (r *CRMRepository) ListActivityTasks(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]crmdb.CrmActivityTask, int64, error) {
	var rows []crmdb.CrmActivityTask
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountActivityTasks(ctx, status); e != nil {
			return e
		}
		rows, e = q.ListActivityTasks(ctx, crmdb.ListActivityTasksParams{Limit: limit, Offset: offset, Status: status})
		return e
	})
	return rows, total, err
}

func (r *CRMRepository) CompleteActivityTask(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		next, e = r.q.WithTx(tx).CompleteActivityTask(ctx, crmdb.CompleteActivityTaskParams{ID: id, UpdatedBy: &actor, RowVersion: version})
		return e
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, httpx.Conflict("INVALID_STATE", "activity task must be Pending with matching rowVersion")
	}
	return next, err
}

func (r *CRMRepository) DrainAndPublish(ctx context.Context, batch int32, pub *eventbus.Publisher) (int, error) {
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
		if err := q.MarkOutboxPublished(ctx, crmdb.MarkOutboxPublishedParams{ID: row.ID, CreatedAt: row.CreatedAt}); err != nil {
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
