// Package repository owns user_db transactions (users, role/org assignments, outbox).
package repository

import (
	"context"
	"encoding/json"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
	"github.com/sinergiinvestasiproperti/erp/services/user-service/internal/userdb"
)

type UserRepository struct {
	db *platform.DB
	q  *userdb.Queries
}

func NewUserRepository(db *platform.DB) *UserRepository {
	return &UserRepository{db: db, q: userdb.New(db.Pool)}
}

func (r *UserRepository) CreateAndEmit(ctx context.Context, scope []uuid.UUID, arg userdb.CreateAppUserParams, ev userdb.InsertOutboxParams) (userdb.CreateAppUserRow, error) {
	var created userdb.CreateAppUserRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if created, e = q.CreateAppUser(ctx, arg); e != nil {
			return e
		}
		ev.AggregateID = created.ID
		return q.InsertOutbox(ctx, ev)
	})
	return created, err
}

func (r *UserRepository) ProvisionAndEmit(ctx context.Context, scope []uuid.UUID, arg userdb.CreateAppUserParams, assignments []userdb.CreateUserOrgAssignmentParams, ev userdb.InsertOutboxParams) (userdb.CreateAppUserRow, []userdb.CreateUserOrgAssignmentRow, error) {
	var created userdb.CreateAppUserRow
	outAssignments := make([]userdb.CreateUserOrgAssignmentRow, 0, len(assignments))
	err := r.db.WithTenantWriteScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if created, e = q.CreateAppUser(ctx, arg); e != nil {
			return e
		}
		for _, a := range assignments {
			a.UserID = created.ID
			row, ae := q.CreateUserOrgAssignment(ctx, a)
			if ae != nil {
				return ae
			}
			outAssignments = append(outAssignments, row)
		}
		ev.AggregateID = created.ID
		return q.InsertOutbox(ctx, ev)
	})
	return created, outAssignments, err
}

func (r *UserRepository) GetByUsernameWithRoles(ctx context.Context, scope []uuid.UUID, username string) (userdb.UsrAppUser, []userdb.ListUserOrgAssignmentsRow, error) {
	var u userdb.UsrAppUser
	var roles []userdb.ListUserOrgAssignmentsRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if u, e = q.GetAppUserByUsername(ctx, username); e != nil {
			return e
		}
		roles, e = q.ListUserOrgAssignments(ctx, u.ID)
		return e
	})
	return u, roles, err
}

func (r *UserRepository) GetWithRoles(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (userdb.UsrAppUser, []userdb.ListUserOrgAssignmentsRow, error) {
	var u userdb.UsrAppUser
	var roles []userdb.ListUserOrgAssignmentsRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if u, e = q.GetAppUser(ctx, id); e != nil {
			return e
		}
		roles, e = q.ListUserOrgAssignments(ctx, id)
		return e
	})
	return u, roles, err
}

func (r *UserRepository) ListWithCount(ctx context.Context, scope []uuid.UUID, arg userdb.ListAppUsersParams) ([]userdb.ListAppUsersRow, int64, error) {
	var rows []userdb.ListAppUsersRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if rows, e = q.ListAppUsers(ctx, arg); e != nil {
			return e
		}
		total, e = q.CountAppUsers(ctx, arg.Status)
		return e
	})
	return rows, total, err
}

func (r *UserRepository) UpdateAndEmit(ctx context.Context, scope []uuid.UUID, arg userdb.UpdateAppUserParams, ev userdb.InsertOutboxParams) (userdb.UpdateAppUserRow, error) {
	var out userdb.UpdateAppUserRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if out, e = q.UpdateAppUser(ctx, arg); e != nil {
			return e
		}
		return q.InsertOutbox(ctx, ev)
	})
	return out, err
}

func (r *UserRepository) ChangeStatusAndEmit(ctx context.Context, scope []uuid.UUID, arg userdb.ChangeUserStatusParams, ev userdb.InsertOutboxParams) (userdb.ChangeUserStatusRow, error) {
	var out userdb.ChangeUserStatusRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if out, e = q.ChangeUserStatus(ctx, arg); e != nil {
			return e
		}
		return q.InsertOutbox(ctx, ev)
	})
	return out, err
}

func (r *UserRepository) DeleteUser(ctx context.Context, scope []uuid.UUID, arg userdb.DeleteAppUserParams) (int64, error) {
	var n int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		n, e = r.q.WithTx(tx).DeleteAppUser(ctx, arg)
		return e
	})
	return n, err
}

func (r *UserRepository) AssignRoleAndEmit(ctx context.Context, scope []uuid.UUID, arg userdb.CreateUserOrgAssignmentParams, ev userdb.InsertOutboxParams) (userdb.CreateUserOrgAssignmentRow, error) {
	var out userdb.CreateUserOrgAssignmentRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		if _, e := q.GetAppUser(ctx, arg.UserID); e != nil {
			return e
		}
		var e error
		if out, e = q.CreateUserOrgAssignment(ctx, arg); e != nil {
			return e
		}
		return q.InsertOutbox(ctx, ev)
	})
	return out, err
}

func (r *UserRepository) RemoveRoleAndEmit(ctx context.Context, scope []uuid.UUID, arg userdb.RemoveUserOrgAssignmentParams, ev userdb.InsertOutboxParams) (int64, error) {
	var n int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if n, e = q.RemoveUserOrgAssignment(ctx, arg); e != nil {
			return e
		}
		if n == 1 {
			return q.InsertOutbox(ctx, ev)
		}
		return nil
	})
	return n, err
}

func (r *UserRepository) DrainAndPublish(ctx context.Context, limit int32, pub *eventbus.Publisher) (int, error) {
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
			log.Printf("user relay publish %s: %v", row.ID, err)
			pubErr = err
			break
		}
		if err := q.MarkOutboxPublished(ctx, userdb.MarkOutboxPublishedParams{
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
