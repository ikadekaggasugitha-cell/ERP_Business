// Package repository owns security_db transactions (roles, permissions,
// role-permission grants, outbox).
package repository

import (
	"context"
	"encoding/json"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/security-service/internal/securitydb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type RoleRepository struct {
	db *platform.DB
	q  *securitydb.Queries
}

func NewRoleRepository(db *platform.DB) *RoleRepository {
	return &RoleRepository{db: db, q: securitydb.New(db.Pool)}
}

func (r *RoleRepository) CreateAndEmit(ctx context.Context, scope []uuid.UUID, arg securitydb.CreateRoleParams, ev securitydb.InsertOutboxParams) (securitydb.CreateRoleRow, error) {
	var created securitydb.CreateRoleRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if created, e = q.CreateRole(ctx, arg); e != nil {
			return e
		}
		ev.AggregateID = created.ID
		return q.InsertOutbox(ctx, ev)
	})
	return created, err
}

func (r *RoleRepository) GetWithPermissions(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (securitydb.SecRole, []securitydb.ListRolePermissionsRow, error) {
	var role securitydb.SecRole
	var perms []securitydb.ListRolePermissionsRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if role, e = q.GetRole(ctx, id); e != nil {
			return e
		}
		perms, e = q.ListRolePermissions(ctx, id)
		return e
	})
	return role, perms, err
}

func (r *RoleRepository) ListWithCount(ctx context.Context, scope []uuid.UUID, arg securitydb.ListRolesParams) ([]securitydb.ListRolesRow, int64, error) {
	var rows []securitydb.ListRolesRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if rows, e = q.ListRoles(ctx, arg); e != nil {
			return e
		}
		total, e = q.CountRoles(ctx, arg.IsSystem)
		return e
	})
	return rows, total, err
}

func (r *RoleRepository) UpdateAndEmit(ctx context.Context, scope []uuid.UUID, arg securitydb.UpdateRoleParams, ev securitydb.InsertOutboxParams) (int32, error) {
	var ver int32
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if ver, e = q.UpdateRole(ctx, arg); e != nil {
			return e
		}
		return q.InsertOutbox(ctx, ev)
	})
	return ver, err
}

func (r *RoleRepository) DeleteRole(ctx context.Context, scope []uuid.UUID, arg securitydb.DeleteRoleParams) (int64, error) {
	var n int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		n, e = r.q.WithTx(tx).DeleteRole(ctx, arg)
		return e
	})
	return n, err
}

func (r *RoleRepository) CreatePermission(ctx context.Context, scope []uuid.UUID, arg securitydb.CreatePermissionParams) (securitydb.CreatePermissionRow, error) {
	var out securitydb.CreatePermissionRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreatePermission(ctx, arg)
		return e
	})
	return out, err
}

func (r *RoleRepository) ListPermissionsWithCount(ctx context.Context, scope []uuid.UUID, arg securitydb.ListPermissionsParams) ([]securitydb.ListPermissionsRow, int64, error) {
	var rows []securitydb.ListPermissionsRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if rows, e = q.ListPermissions(ctx, arg); e != nil {
			return e
		}
		total, e = q.CountPermissions(ctx, arg.Module)
		return e
	})
	return rows, total, err
}

func (r *RoleRepository) GrantPermissionAndEmit(ctx context.Context, scope []uuid.UUID, arg securitydb.CreateRolePermissionParams, ev securitydb.InsertOutboxParams) (int64, error) {
	var n int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		if _, e := q.GetRole(ctx, arg.RoleID); e != nil {
			return e
		}
		var e error
		if n, e = q.CreateRolePermission(ctx, arg); e != nil {
			return e
		}
		return q.InsertOutbox(ctx, ev)
	})
	return n, err
}

func (r *RoleRepository) RevokePermissionAndEmit(ctx context.Context, scope []uuid.UUID, arg securitydb.DeleteRolePermissionParams, ev securitydb.InsertOutboxParams) (int64, error) {
	var n int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if n, e = q.DeleteRolePermission(ctx, arg); e != nil {
			return e
		}
		if n == 1 {
			return q.InsertOutbox(ctx, ev)
		}
		return nil
	})
	return n, err
}

func (r *RoleRepository) DrainAndPublish(ctx context.Context, limit int32, pub *eventbus.Publisher) (int, error) {
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
			log.Printf("security relay publish %s: %v", row.ID, err)
			break
		}
		if err := q.MarkOutboxPublished(ctx, securitydb.MarkOutboxPublishedParams{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
		}); err != nil {
			return published, err
		}
		published++
	}
	if err := tx.Commit(ctx); err != nil {
		return published, err
	}
	return published, nil
}
