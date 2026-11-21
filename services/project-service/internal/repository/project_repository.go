// Package repository implements RLS-enforced Project Management persistence
// (projects, WBS hierarchy with cycle detection, budgets, costs, progress,
// billings, change orders, and transactional outbox).
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/project-service/internal/prjdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type ProjectRepository struct {
	db *platform.DB
	q  *prjdb.Queries
}

func NewProjectRepository(db *platform.DB) *ProjectRepository {
	var q *prjdb.Queries
	if db != nil && db.Pool != nil {
		q = prjdb.New(db.Pool)
	}
	return &ProjectRepository{db: db, q: q}
}

func (r *ProjectRepository) withOrgTx(ctx context.Context, scope []uuid.UUID, fn func(pgx.Tx) error) error {
	if len(scope) == 0 {
		return errors.New("org scope must not be empty")
	}
	return r.db.WithOrgScope(ctx, scope, fn)
}

func emit(ctx context.Context, q *prjdb.Queries, aggType, eventType string, id, org, corr uuid.UUID, payload map[string]any) error {
	payload["id"] = id.String()
	payload["organizationId"] = org.String()
	b, _ := json.Marshal(payload)
	return q.InsertOutbox(ctx, prjdb.InsertOutboxParams{
		AggregateType: aggType, AggregateID: id, EventType: eventType,
		Payload: b, OrganizationID: org, CorrelationID: corr,
	})
}

func (r *ProjectRepository) CreateProject(ctx context.Context, scope []uuid.UUID, arg prjdb.CreateProjectParams) (prjdb.CreateProjectRow, error) {
	var out prjdb.CreateProjectRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateProject(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "project code already exists")
}

func (r *ProjectRepository) GetProject(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (prjdb.PrjProject, error) {
	var out prjdb.PrjProject
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).GetProject(ctx, id)
		return e
	})
	return out, httpx.MapDBError(err, "project not found")
}

func (r *ProjectRepository) ListProjects(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]prjdb.PrjProject, int64, error) {
	var rows []prjdb.PrjProject
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountProjects(ctx, status); e != nil {
			return e
		}
		rows, e = q.ListProjects(ctx, prjdb.ListProjectsParams{Limit: limit, Offset: offset, Status: status})
		return e
	})
	return rows, total, err
}

func (r *ProjectRepository) UpdateProject(ctx context.Context, scope []uuid.UUID, arg prjdb.UpdateProjectParams) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		next, e = r.q.WithTx(tx).UpdateProject(ctx, arg)
		return e
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, httpx.Conflict("VERSION_CONFLICT", "project not found or version mismatch")
	}
	return next, httpx.MapDBError(err, "project update conflict")
}

func (r *ProjectRepository) TransitionProject(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32, fromStatus, toStatus string) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		next, e = r.q.WithTx(tx).TransitionProjectStatus(ctx, prjdb.TransitionProjectStatusParams{
			ID: id, Status: toStatus, UpdatedBy: &actor, RowVersion: version, Status_2: fromStatus,
		})
		return e
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, httpx.Conflict("INVALID_STATE", fmt.Sprintf("project must be %s with matching rowVersion", fromStatus))
	}
	return next, httpx.MapDBError(err, "project transition failed")
}

func (r *ProjectRepository) DeleteProject(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) error {
	var n int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		n, e = r.q.WithTx(tx).SoftDeleteProject(ctx, prjdb.SoftDeleteProjectParams{ID: id, UpdatedBy: &actor, RowVersion: version})
		return e
	})
	if err != nil {
		return httpx.MapDBError(err, "cannot delete project")
	}
	if n == 0 {
		return httpx.Conflict("INVALID_STATE", "only Draft project with matching rowVersion can be deleted")
	}
	return nil
}

func (r *ProjectRepository) CreateWBSElement(ctx context.Context, scope []uuid.UUID, arg prjdb.CreateWBSElementParams) (prjdb.CreateWBSElementRow, error) {
	var out prjdb.CreateWBSElementRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		if _, e := q.GetProject(ctx, arg.ProjectID); e != nil {
			return httpx.NotFound("project not found")
		}
		if arg.ParentID != nil {
			parent, e := q.GetWBSElement(ctx, *arg.ParentID)
			if e != nil {
				return httpx.NotFound("parent WBS element not found")
			}
			if parent.ProjectID != arg.ProjectID {
				return httpx.Unprocessable("PARENT_PROJECT_MISMATCH", "parent WBS element belongs to another project")
			}
		}
		var e error
		out, e = q.CreateWBSElement(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "WBS code already exists for project")
}

func (r *ProjectRepository) ListWBSElements(ctx context.Context, scope []uuid.UUID, projectID uuid.UUID) ([]prjdb.PrjWbsElement, error) {
	var rows []prjdb.PrjWbsElement
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		rows, e = r.q.WithTx(tx).ListWBSElementsByProject(ctx, projectID)
		return e
	})
	return rows, err
}

func (r *ProjectRepository) UpdateWBSElement(ctx context.Context, scope []uuid.UUID, arg prjdb.UpdateWBSElementParams) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		cur, e := q.GetWBSElement(ctx, arg.ID)
		if e != nil {
			return httpx.NotFound("WBS element not found")
		}
		if arg.ParentID != nil {
			if *arg.ParentID == arg.ID {
				return httpx.Unprocessable("CYCLE_DETECTED", "WBS element cannot be its own parent")
			}
			visited := map[uuid.UUID]struct{}{arg.ID: {}}
			cursor := arg.ParentID
			for cursor != nil {
				if _, seen := visited[*cursor]; seen {
					return httpx.Unprocessable("CYCLE_DETECTED", "WBS hierarchy cycle detected")
				}
				visited[*cursor] = struct{}{}
				p, pe := q.GetWBSElement(ctx, *cursor)
				if pe != nil {
					return httpx.NotFound("parent WBS element not found")
				}
				if p.ProjectID != cur.ProjectID {
					return httpx.Unprocessable("PARENT_PROJECT_MISMATCH", "parent WBS element belongs to another project")
				}
				cursor = p.ParentID
			}
		}
		next, e = q.UpdateWBSElement(ctx, arg)
		return e
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, httpx.Conflict("VERSION_CONFLICT", "WBS element version mismatch")
	}
	return next, httpx.MapDBError(err, "WBS update conflict")
}

func (r *ProjectRepository) DeleteWBSElement(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) error {
	return r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		children, e := q.CountWBSChildren(ctx, &id)
		if e != nil {
			return e
		}
		if children > 0 {
			return httpx.Conflict("HAS_CHILDREN", "cannot delete WBS element with child elements")
		}
		n, e := q.SoftDeleteWBSElement(ctx, prjdb.SoftDeleteWBSElementParams{ID: id, UpdatedBy: &actor, RowVersion: version})
		if e != nil {
			return httpx.MapDBError(e, "cannot delete WBS element")
		}
		if n == 0 {
			return httpx.Conflict("VERSION_CONFLICT", "WBS element not found or version mismatch")
		}
		return nil
	})
}

func (r *ProjectRepository) CreateBudget(ctx context.Context, scope []uuid.UUID, arg prjdb.CreateProjectBudgetParams) (prjdb.CreateProjectBudgetRow, error) {
	var out prjdb.CreateProjectBudgetRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateProjectBudget(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "failed to create project budget")
}

func (r *ProjectRepository) ListBudgets(ctx context.Context, scope []uuid.UUID, projectID uuid.UUID) ([]prjdb.PrjProjectBudget, error) {
	var rows []prjdb.PrjProjectBudget
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		rows, e = r.q.WithTx(tx).ListProjectBudgets(ctx, projectID)
		return e
	})
	return rows, err
}

func (r *ProjectRepository) ApproveBudget(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		next, e = r.q.WithTx(tx).ApproveProjectBudget(ctx, prjdb.ApproveProjectBudgetParams{ID: id, UpdatedBy: &actor, RowVersion: version})
		return e
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, httpx.Conflict("INVALID_STATE", "project budget must be Draft with matching rowVersion")
	}
	return next, err
}

func (r *ProjectRepository) CreateCost(ctx context.Context, scope []uuid.UUID, arg prjdb.CreateProjectCostParams) (prjdb.CreateProjectCostRow, error) {
	var out prjdb.CreateProjectCostRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateProjectCost(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "failed to record project cost")
}

func (r *ProjectRepository) ListCosts(ctx context.Context, scope []uuid.UUID, projectID uuid.UUID) ([]prjdb.PrjProjectCost, error) {
	var rows []prjdb.PrjProjectCost
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		rows, e = r.q.WithTx(tx).ListProjectCosts(ctx, projectID)
		return e
	})
	return rows, err
}

func (r *ProjectRepository) UpsertProgress(ctx context.Context, scope []uuid.UUID, arg prjdb.UpsertProjectProgressParams) (prjdb.UpsertProjectProgressRow, error) {
	var out prjdb.UpsertProjectProgressRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).UpsertProjectProgress(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "failed to record project progress")
}

func (r *ProjectRepository) ListProgress(ctx context.Context, scope []uuid.UUID, wbsID uuid.UUID) ([]prjdb.PrjProjectProgress, error) {
	var rows []prjdb.PrjProjectProgress
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		rows, e = r.q.WithTx(tx).ListProjectProgress(ctx, wbsID)
		return e
	})
	return rows, err
}

func (r *ProjectRepository) CreateBilling(ctx context.Context, scope []uuid.UUID, arg prjdb.CreateProjectBillingParams) (prjdb.CreateProjectBillingRow, error) {
	var out prjdb.CreateProjectBillingRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		if _, e := r.q.WithTx(tx).GetProject(ctx, arg.ProjectID); e != nil {
			return httpx.NotFound("project not found")
		}
		var e error
		out, e = r.q.WithTx(tx).CreateProjectBilling(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "failed to create project billing")
}

func (r *ProjectRepository) ListBillings(ctx context.Context, scope []uuid.UUID, projectID *uuid.UUID, limit, offset int32) ([]prjdb.PrjProjectBilling, int64, error) {
	var rows []prjdb.PrjProjectBilling
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountProjectBillings(ctx, projectID); e != nil {
			return e
		}
		rows, e = q.ListProjectBillings(ctx, prjdb.ListProjectBillingsParams{Limit: limit, Offset: offset, ProjectID: projectID})
		return e
	})
	return rows, total, err
}

func (r *ProjectRepository) ApproveBillingAndEmit(ctx context.Context, scope []uuid.UUID, id, org, actor, corr uuid.UUID, version int32, customerID *uuid.UUID, arRef *string) (int32, string, error) {
	var next int32
	var invoiceRef string
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		billing, e := q.GetProjectBilling(ctx, id)
		if e != nil {
			return httpx.NotFound("project billing not found")
		}
		if billing.Status != "Draft" {
			return httpx.Conflict("INVALID_STATE", "only Draft billing can be approved")
		}
		proj, e := q.GetProject(ctx, billing.ProjectID)
		if e != nil {
			return httpx.NotFound("project not found")
		}
		if arRef != nil && *arRef != "" {
			invoiceRef = *arRef
		} else if billing.ArInvoiceRef != nil && *billing.ArInvoiceRef != "" {
			invoiceRef = *billing.ArInvoiceRef
		} else {
			invoiceRef = fmt.Sprintf("PB-%s-%s", proj.Code, id.String()[:8])
		}
		next, e = q.InvoiceProjectBilling(ctx, prjdb.InvoiceProjectBillingParams{
			ID: id, ArInvoiceRef: &invoiceRef, UpdatedBy: &actor, RowVersion: version,
		})
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.Conflict("VERSION_CONFLICT", "project billing version mismatch")
		}
		if e != nil {
			return e
		}
		payload := map[string]any{
			"billingId":    id.String(),
			"projectId":    proj.ID.String(),
			"projectCode":  proj.Code,
			"billingType":  billing.BillingType,
			"amount":       platform.FormatNumeric(billing.Amount, 4),
			"currencyCode": proj.CurrencyCode,
			"arInvoiceRef": invoiceRef,
		}
		if customerID != nil {
			payload["customerId"] = customerID.String()
		}
		return emit(ctx, q, "ProjectBilling", "project.billing.approved", id, org, corr, payload)
	})
	return next, invoiceRef, err
}

func (r *ProjectRepository) CreateChangeOrder(ctx context.Context, scope []uuid.UUID, arg prjdb.CreateChangeOrderParams) (prjdb.CreateChangeOrderRow, error) {
	var out prjdb.CreateChangeOrderRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateChangeOrder(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "failed to create change order")
}

func (r *ProjectRepository) ListChangeOrders(ctx context.Context, scope []uuid.UUID, projectID uuid.UUID) ([]prjdb.PrjChangeOrder, error) {
	var rows []prjdb.PrjChangeOrder
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		rows, e = r.q.WithTx(tx).ListChangeOrders(ctx, projectID)
		return e
	})
	return rows, err
}

func (r *ProjectRepository) DecideChangeOrder(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32, status string) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		next, e = r.q.WithTx(tx).DecideChangeOrder(ctx, prjdb.DecideChangeOrderParams{ID: id, Status: status, UpdatedBy: &actor, RowVersion: version})
		return e
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, httpx.Conflict("INVALID_STATE", "change order must be Draft with matching rowVersion")
	}
	return next, err
}

func (r *ProjectRepository) DrainAndPublish(ctx context.Context, batch int32, pub *eventbus.Publisher) (int, error) {
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
		if err := q.MarkOutboxPublished(ctx, prjdb.MarkOutboxPublishedParams{ID: row.ID, CreatedAt: row.CreatedAt}); err != nil {
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
