// Package repository owns org_db transactions (organization types,
// organizations, materialized path + closure table, outbox).
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/org-service/internal/orgdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

var ErrHasChildren = errors.New("organization still has active child units")

type OrganizationRepository struct {
	db *platform.DB
	q  *orgdb.Queries
}

func NewOrganizationRepository(db *platform.DB) *OrganizationRepository {
	return &OrganizationRepository{db: db, q: orgdb.New(db.Pool)}
}

func (r *OrganizationRepository) CreateOrganizationType(ctx context.Context, scope []uuid.UUID, arg orgdb.CreateOrganizationTypeParams) (orgdb.CreateOrganizationTypeRow, error) {
	var out orgdb.CreateOrganizationTypeRow
	err := r.db.WithTenantWriteScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateOrganizationType(ctx, arg)
		return e
	})
	return out, err
}

func (r *OrganizationRepository) ListOrganizationTypesWithCount(ctx context.Context, scope []uuid.UUID, arg orgdb.ListOrganizationTypesParams) ([]orgdb.ListOrganizationTypesRow, int64, error) {
	var rows []orgdb.ListOrganizationTypesRow
	var total int64
	err := r.db.WithOrgReadScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if rows, e = q.ListOrganizationTypes(ctx, arg); e != nil {
			return e
		}
		total, e = q.CountOrganizationTypes(ctx)
		return e
	})
	return rows, total, err
}

func (r *OrganizationRepository) GetOrganizationTypeByID(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (orgdb.GetOrganizationTypeByIDRow, error) {
	var out orgdb.GetOrganizationTypeByIDRow
	err := r.db.WithOrgReadScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).GetOrganizationTypeByID(ctx, id)
		return e
	})
	return out, err
}

func (r *OrganizationRepository) ListAttributes(ctx context.Context, scope []uuid.UUID, orgRefID uuid.UUID) (map[string]any, error) {
	out := make(map[string]any)
	err := r.db.WithOrgReadScope(ctx, scope, func(tx pgx.Tx) error {
		rows, e := r.q.WithTx(tx).ListOrganizationAttributes(ctx, orgRefID)
		if e != nil {
			return e
		}
		for _, row := range rows {
			var v any
			if err := json.Unmarshal(row.AttrValue, &v); err == nil {
				out[row.AttrKey] = v
			}
		}
		return nil
	})
	return out, err
}

func (r *OrganizationRepository) UpsertAttributes(ctx context.Context, scope []uuid.UUID, orgID, orgRefID uuid.UUID, attrs map[string]any, actor uuid.UUID) error {
	if len(attrs) == 0 {
		return nil
	}
	return r.db.WithTenantWriteScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		for k, v := range attrs {
			raw, err := json.Marshal(v)
			if err != nil {
				return err
			}
			if err := q.UpsertOrganizationAttribute(ctx, orgdb.UpsertOrganizationAttributeParams{
				OrganizationID:    orgID,
				OrganizationRefID: orgRefID,
				AttrKey:           k,
				AttrValue:         raw,
				CreatedBy:         actor,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *OrganizationRepository) CreateAndEmit(ctx context.Context, scope []uuid.UUID, arg orgdb.CreateOrganizationParams, ev orgdb.InsertOutboxParams) (orgdb.CreateOrganizationRow, error) {
	var created orgdb.CreateOrganizationRow
	err := r.db.WithTenantWriteScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		arg.Path = "/" + arg.Code
		if arg.ParentID != nil {
			parent, err := q.GetOrganization(ctx, *arg.ParentID)
			if err != nil {
				return err
			}
			arg.Path = parent.Path + "/" + arg.Code
		}
		var e error
		if created, e = q.CreateOrganization(ctx, arg); e != nil {
			return e
		}
		if e = q.InsertSelfClosure(ctx, orgdb.InsertSelfClosureParams{
			OrganizationID: arg.OrganizationID,
			AncestorID:     created.ID,
			CreatedBy:      arg.CreatedBy,
		}); e != nil {
			return e
		}
		if arg.ParentID != nil {
			if e = q.InsertAncestorClosures(ctx, orgdb.InsertAncestorClosuresParams{
				OrganizationID: arg.OrganizationID,
				DescendantID:   created.ID,
				CreatedBy:      arg.CreatedBy,
				DescendantID_2: *arg.ParentID,
			}); e != nil {
				return e
			}
		}
		ev.AggregateID = created.ID
		return q.InsertOutbox(ctx, ev)
	})
	return created, err
}

func (r *OrganizationRepository) GetWithHierarchy(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (orgdb.OrgOrganization, []orgdb.ListOrganizationDescendantsRow, error) {
	var o orgdb.OrgOrganization
	var desc []orgdb.ListOrganizationDescendantsRow
	err := r.db.WithOrgReadScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if o, e = q.GetOrganization(ctx, id); e != nil {
			return e
		}
		desc, e = q.ListOrganizationDescendants(ctx, id)
		return e
	})
	return o, desc, err
}

func (r *OrganizationRepository) ListWithCount(ctx context.Context, scope []uuid.UUID, arg orgdb.ListOrganizationsParams) ([]orgdb.ListOrganizationsRow, int64, error) {
	var rows []orgdb.ListOrganizationsRow
	var total int64
	err := r.db.WithOrgReadScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if rows, e = q.ListOrganizations(ctx, arg); e != nil {
			return e
		}
		total, e = q.CountOrganizations(ctx, orgdb.CountOrganizationsParams{
			Status:   arg.Status,
			ParentID: arg.ParentID,
		})
		return e
	})
	return rows, total, err
}

func (r *OrganizationRepository) UpdateAndEmit(ctx context.Context, scope []uuid.UUID, arg orgdb.UpdateOrganizationParams, ev orgdb.InsertOutboxParams) (orgdb.UpdateOrganizationRow, error) {
	var out orgdb.UpdateOrganizationRow
	err := r.db.WithTenantWriteScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if out, e = q.UpdateOrganization(ctx, arg); e != nil {
			return e
		}
		return q.InsertOutbox(ctx, ev)
	})
	return out, err
}

func (r *OrganizationRepository) ActivateAndEmit(ctx context.Context, scope []uuid.UUID, arg orgdb.ActivateOrganizationParams, ev orgdb.InsertOutboxParams) (int32, error) {
	var ver int32
	err := r.db.WithTenantWriteScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if ver, e = q.ActivateOrganization(ctx, arg); e != nil {
			return e
		}
		return q.InsertOutbox(ctx, ev)
	})
	return ver, err
}

func (r *OrganizationRepository) DeleteOrganization(ctx context.Context, scope []uuid.UUID, arg orgdb.DeleteOrganizationParams) (int64, error) {
	var n int64
	err := r.db.WithTenantWriteScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		children, err := q.CountChildOrganizations(ctx, &arg.ID)
		if err != nil {
			return err
		}
		if children > 0 {
			return ErrHasChildren
		}
		var e error
		n, e = q.DeleteOrganization(ctx, arg)
		return e
	})
	return n, err
}

func (r *OrganizationRepository) DrainAndPublish(ctx context.Context, limit int32, pub *eventbus.Publisher) (int, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

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
		if err := pub.Publish(ctx, env); err != nil {
			log.Printf("org relay publish %s: %v", row.ID, err)
			pubErr = err
			break
		}
		if err := q.MarkOutboxPublished(ctx, orgdb.MarkOutboxPublishedParams{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
		}); err != nil {
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
