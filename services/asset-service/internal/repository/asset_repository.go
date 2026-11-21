// Package repository implements RLS-enforced Operational Asset Management
// persistence (assets, location history, movements, verifications, and outbox).
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/asset-service/internal/astdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type AssetRepository struct {
	db *platform.DB
	q  *astdb.Queries
}

func NewAssetRepository(db *platform.DB) *AssetRepository {
	var q *astdb.Queries
	if db != nil && db.Pool != nil {
		q = astdb.New(db.Pool)
	}
	return &AssetRepository{db: db, q: q}
}

func (r *AssetRepository) withOrgTx(ctx context.Context, scope []uuid.UUID, fn func(pgx.Tx) error) error {
	if len(scope) == 0 {
		return errors.New("org scope must not be empty")
	}
	return r.db.WithOrgScope(ctx, platform.ScopeWithSystemOrg(scope), fn)
}

func emit(ctx context.Context, q *astdb.Queries, aggType, eventType string, id, org, corr uuid.UUID, payload map[string]any) error {
	payload["id"] = id.String()
	payload["organizationId"] = org.String()
	b, _ := json.Marshal(payload)
	return q.InsertOutbox(ctx, astdb.InsertOutboxParams{
		AggregateType: aggType, AggregateID: id, EventType: eventType,
		Payload: b, OrganizationID: org, CorrelationID: corr,
	})
}

func (r *AssetRepository) CreateAsset(ctx context.Context, scope []uuid.UUID, arg astdb.CreateAssetParams) (astdb.CreateAssetRow, error) {
	var out astdb.CreateAssetRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateAsset(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "asset code already exists")
}

func (r *AssetRepository) GetAsset(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (astdb.AstAsset, error) {
	var out astdb.AstAsset
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).GetAsset(ctx, id)
		return e
	})
	return out, httpx.MapDBError(err, "asset not found")
}

func (r *AssetRepository) ListAssets(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]astdb.AstAsset, int64, error) {
	var rows []astdb.AstAsset
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountAssets(ctx, status); e != nil {
			return e
		}
		rows, e = q.ListAssets(ctx, astdb.ListAssetsParams{Limit: limit, Offset: offset, Status: status})
		return e
	})
	return rows, total, err
}

func (r *AssetRepository) UpdateAsset(ctx context.Context, scope []uuid.UUID, arg astdb.UpdateAssetParams) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		cur, e := q.GetAsset(ctx, arg.ID)
		if e != nil {
			return httpx.NotFound("asset not found")
		}
		if cur.OrganizationID == platform.SystemOrgID {
			return httpx.Forbidden("system asset template is read-only")
		}
		next, e = q.UpdateAsset(ctx, arg)
		return e
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, httpx.Conflict("VERSION_CONFLICT", "asset not found or version mismatch")
	}
	return next, httpx.MapDBError(err, "asset update conflict")
}

func (r *AssetRepository) DeleteAsset(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) error {
	var n int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		cur, e := q.GetAsset(ctx, id)
		if e != nil {
			return httpx.NotFound("asset not found")
		}
		if cur.OrganizationID == platform.SystemOrgID {
			return httpx.Forbidden("system asset template cannot be deleted")
		}
		n, e = q.SoftDeleteAsset(ctx, astdb.SoftDeleteAssetParams{ID: id, UpdatedBy: &actor, RowVersion: version})
		return e
	})
	if err != nil {
		return httpx.MapDBError(err, "cannot delete asset")
	}
	if n == 0 {
		return httpx.Conflict("VERSION_CONFLICT", "asset not found or version mismatch")
	}
	return nil
}

func (r *AssetRepository) CreateLocationHistory(ctx context.Context, scope []uuid.UUID, arg astdb.CreateLocationHistoryParams) (astdb.CreateLocationHistoryRow, error) {
	var out astdb.CreateLocationHistoryRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		if _, e := r.q.WithTx(tx).GetAsset(ctx, arg.AssetID); e != nil {
			return httpx.NotFound("asset not found")
		}
		var e error
		out, e = r.q.WithTx(tx).CreateLocationHistory(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "failed to record location history")
}

func (r *AssetRepository) ListLocationHistory(ctx context.Context, scope []uuid.UUID, assetID uuid.UUID) ([]astdb.AstAssetLocationHistory, error) {
	var rows []astdb.AstAssetLocationHistory
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		rows, e = r.q.WithTx(tx).ListLocationHistory(ctx, assetID)
		return e
	})
	return rows, err
}

func (r *AssetRepository) CreateMovement(ctx context.Context, scope []uuid.UUID, arg astdb.CreateAssetMovementParams) (astdb.CreateAssetMovementRow, error) {
	var out astdb.CreateAssetMovementRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		asset, e := q.GetAsset(ctx, arg.AssetID)
		if e != nil {
			return httpx.NotFound("asset not found")
		}
		if asset.Status == "Disposed" {
			return httpx.Conflict("ASSET_DISPOSED", "cannot move a Disposed asset")
		}
		if arg.FromOrganizationRef == nil {
			from := asset.OrganizationRef
			arg.FromOrganizationRef = &from
		}
		if *arg.FromOrganizationRef == arg.ToOrganizationRef {
			return httpx.Unprocessable("SAME_ORGANIZATION", "fromOrganizationRef and toOrganizationRef must differ")
		}
		out, e = q.CreateAssetMovement(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "failed to create asset movement")
}

func (r *AssetRepository) ListMovements(ctx context.Context, scope []uuid.UUID, assetID *uuid.UUID, limit, offset int32) ([]astdb.AstAssetMovement, int64, error) {
	var rows []astdb.AstAssetMovement
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountAssetMovements(ctx, assetID); e != nil {
			return e
		}
		rows, e = q.ListAssetMovements(ctx, astdb.ListAssetMovementsParams{Limit: limit, Offset: offset, AssetID: assetID})
		return e
	})
	return rows, total, err
}

func (r *AssetRepository) ApproveMovement(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		next, e = r.q.WithTx(tx).ApproveAssetMovement(ctx, astdb.ApproveAssetMovementParams{ID: id, ApprovedBy: &actor, RowVersion: version})
		return e
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, httpx.Conflict("INVALID_STATE", "asset movement must be Draft with matching rowVersion")
	}
	return next, err
}

func (r *AssetRepository) CompleteMovementAndEmit(ctx context.Context, scope []uuid.UUID, id, org, actor, corr uuid.UUID, version int32) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		mv, e := q.GetAssetMovement(ctx, id)
		if e != nil {
			return httpx.NotFound("asset movement not found")
		}
		if mv.Status != "Approved" {
			return httpx.Conflict("INVALID_STATE", "asset movement must be Approved before completion")
		}
		asset, e := q.GetAsset(ctx, mv.AssetID)
		if e != nil {
			return httpx.NotFound("asset not found")
		}
		if asset.OrganizationID == platform.SystemOrgID {
			return httpx.Forbidden("system asset template cannot be moved")
		}
		if asset.Status == "Disposed" {
			return httpx.Conflict("ASSET_DISPOSED", "cannot complete movement for a Disposed asset")
		}
		if mv.FromOrganizationRef != nil && *mv.FromOrganizationRef == mv.ToOrganizationRef {
			return httpx.Unprocessable("SAME_ORGANIZATION", "fromOrganizationRef and toOrganizationRef must differ")
		}
		next, e = q.CompleteAssetMovement(ctx, astdb.CompleteAssetMovementParams{ID: id, UpdatedBy: &actor, RowVersion: version})
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.Conflict("VERSION_CONFLICT", "asset movement version mismatch")
		}
		if e != nil {
			return e
		}
		if e = q.UpdateAssetOrganizationRef(ctx, astdb.UpdateAssetOrganizationRefParams{
			ID: mv.AssetID, OrganizationRef: mv.ToOrganizationRef, UpdatedBy: &actor,
		}); e != nil {
			return e
		}
		note := "Completed asset movement " + id.String()
		if _, e = q.CreateLocationHistory(ctx, astdb.CreateLocationHistoryParams{
			OrganizationID: org, AssetID: mv.AssetID, OrganizationRef: mv.ToOrganizationRef,
			LocationNote: &note, EffectiveFrom: pgtype.Date{Time: time.Now().UTC(), Valid: true}, CreatedBy: actor,
		}); e != nil {
			return e
		}
		payload := map[string]any{
			"movementId":        id.String(),
			"assetId":           mv.AssetID.String(),
			"toOrganizationRef": mv.ToOrganizationRef.String(),
		}
		if mv.FromOrganizationRef != nil {
			payload["fromOrganizationRef"] = mv.FromOrganizationRef.String()
		}
		return emit(ctx, q, "AssetMovement", "asset.movement.completed", id, org, corr, payload)
	})
	return next, err
}

func (r *AssetRepository) RecordVerificationAndEmit(ctx context.Context, scope []uuid.UUID, arg astdb.CreateAssetVerificationParams, corr uuid.UUID) (astdb.CreateAssetVerificationRow, error) {
	var out astdb.CreateAssetVerificationRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		asset, e := q.GetAsset(ctx, arg.AssetID)
		if e != nil {
			return httpx.NotFound("asset not found")
		}
		if arg.SystemStatus == nil {
			s := asset.Status
			arg.SystemStatus = &s
		}
		out, e = q.CreateAssetVerification(ctx, arg)
		if e != nil {
			return e
		}
		payload := map[string]any{
			"verificationId": out.ID.String(),
			"assetId":        arg.AssetID.String(),
			"assetCode":      asset.AssetCode,
		}
		if arg.SystemStatus != nil {
			payload["systemStatus"] = *arg.SystemStatus
		}
		if arg.PhysicalStatus != nil {
			payload["physicalStatus"] = *arg.PhysicalStatus
		}
		return emit(ctx, q, "AssetVerification", "asset.verification.recorded", out.ID, arg.OrganizationID, corr, payload)
	})
	return out, httpx.MapDBError(err, "failed to record verification")
}

func (r *AssetRepository) ListVerifications(ctx context.Context, scope []uuid.UUID, assetID *uuid.UUID, limit, offset int32) ([]astdb.AstAssetVerification, int64, error) {
	var rows []astdb.AstAssetVerification
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountAssetVerifications(ctx, assetID); e != nil {
			return e
		}
		rows, e = q.ListAssetVerifications(ctx, astdb.ListAssetVerificationsParams{Limit: limit, Offset: offset, AssetID: assetID})
		return e
	})
	return rows, total, err
}

func (r *AssetRepository) DrainAndPublish(ctx context.Context, batch int32, pub *eventbus.Publisher) (int, error) {
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
		if err := q.MarkOutboxPublished(ctx, astdb.MarkOutboxPublishedParams{ID: row.ID, CreatedAt: row.CreatedAt}); err != nil {
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
