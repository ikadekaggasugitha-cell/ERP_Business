// Package repository implements RLS-enforced Fixed Asset, Depreciation, and
// Asset Transaction persistence with exact *big.Rat math and transactional outbox.
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

	"github.com/sinergiinvestasiproperti/erp/services/fixedasset-service/internal/fadb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type FixedAssetRepository struct {
	db *platform.DB
	q  *fadb.Queries
}

func NewFixedAssetRepository(db *platform.DB) *FixedAssetRepository {
	var q *fadb.Queries
	if db != nil && db.Pool != nil {
		q = fadb.New(db.Pool)
	}
	return &FixedAssetRepository{db: db, q: q}
}

func (r *FixedAssetRepository) withOrgTx(ctx context.Context, scope []uuid.UUID, fn func(pgx.Tx) error) error {
	if len(scope) == 0 {
		return errors.New("org scope must not be empty")
	}
	return r.db.WithOrgScope(ctx, platform.ScopeWithSystemOrg(scope), fn)
}

func emit(ctx context.Context, q *fadb.Queries, aggType, eventType string, id, org, corr uuid.UUID, payload map[string]any) error {
	payload["id"] = id.String()
	payload["organizationId"] = org.String()
	b, _ := json.Marshal(payload)
	return q.InsertOutbox(ctx, fadb.InsertOutboxParams{
		AggregateType: aggType, AggregateID: id, EventType: eventType,
		Payload: b, OrganizationID: org, CorrelationID: corr,
	})
}

// ComputeMonthlyDepreciation calculates exact monthly depreciation using *big.Rat,
// capped so netBookValue never drops below salvageValue.
func ComputeMonthlyDepreciation(method string, cost, salvage, currentNBV *big.Rat, usefulLifeMonths int32) (*big.Rat, *big.Rat, error) {
	return ComputeMonthlyDepreciationWithUnits(method, cost, salvage, currentNBV, usefulLifeMonths, nil, nil)
}

// ComputeMonthlyDepreciationWithUnits calculates exact depreciation for StraightLine,
// DecliningBalance, or UnitsOfProduction (using unitsProduced / totalEstimatedUnits when provided).
func ComputeMonthlyDepreciationWithUnits(method string, cost, salvage, currentNBV *big.Rat, usefulLifeMonths int32, unitsProduced, totalEstimatedUnits *big.Rat) (*big.Rat, *big.Rat, error) {
	if usefulLifeMonths <= 0 {
		return nil, nil, httpx.Unprocessable("INVALID_LIFE", "usefulLifeMonths must be > 0")
	}
	if cost == nil || salvage == nil || currentNBV == nil {
		return nil, nil, httpx.Unprocessable("INVALID_AMOUNT", "asset monetary amounts must be valid")
	}
	remaining := new(big.Rat).Sub(currentNBV, salvage)
	if remaining.Sign() <= 0 {
		return big.NewRat(0, 1), new(big.Rat).Set(currentNBV), nil
	}
	life := big.NewRat(int64(usefulLifeMonths), 1)
	var dep *big.Rat
	switch method {
	case "StraightLine":
		depreciable := new(big.Rat).Sub(cost, salvage)
		if depreciable.Sign() <= 0 {
			return big.NewRat(0, 1), new(big.Rat).Set(currentNBV), nil
		}
		dep = new(big.Rat).Quo(depreciable, life)
	case "DecliningBalance":
		// Double-declining monthly rate = 2 / usefulLifeMonths
		dep = new(big.Rat).Quo(new(big.Rat).Mul(currentNBV, big.NewRat(2, 1)), life)
	case "UnitsOfProduction":
		depreciable := new(big.Rat).Sub(cost, salvage)
		if depreciable.Sign() <= 0 {
			return big.NewRat(0, 1), new(big.Rat).Set(currentNBV), nil
		}
		if unitsProduced != nil && totalEstimatedUnits != nil && totalEstimatedUnits.Sign() > 0 {
			if unitsProduced.Sign() < 0 {
				return nil, nil, httpx.Unprocessable("INVALID_UNITS", "unitsProduced must be >= 0")
			}
			dep = new(big.Rat).Quo(new(big.Rat).Mul(depreciable, unitsProduced), totalEstimatedUnits)
		} else {
			dep = new(big.Rat).Quo(depreciable, life)
		}
	default:
		return nil, nil, httpx.Unprocessable("INVALID_METHOD", "unsupported depreciation method")
	}
	if dep.Cmp(remaining) > 0 {
		dep = remaining
	}
	newNBV := new(big.Rat).Sub(currentNBV, dep)
	return dep, newNBV, nil
}

func (r *FixedAssetRepository) CreateCategory(ctx context.Context, scope []uuid.UUID, arg fadb.CreateAssetCategoryParams) (fadb.CreateAssetCategoryRow, error) {
	var out fadb.CreateAssetCategoryRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateAssetCategory(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "asset category code already exists")
}

func (r *FixedAssetRepository) GetCategory(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (fadb.FaAssetCategory, error) {
	var out fadb.FaAssetCategory
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).GetAssetCategory(ctx, id)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("asset category not found")
		}
		return e
	})
	return out, err
}

func (r *FixedAssetRepository) ListCategories(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]fadb.FaAssetCategory, int64, error) {
	var rows []fadb.FaAssetCategory
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountAssetCategories(ctx); e != nil {
			return e
		}
		rows, e = q.ListAssetCategories(ctx, fadb.ListAssetCategoriesParams{Limit: limit, Offset: offset})
		return e
	})
	return rows, total, err
}

func (r *FixedAssetRepository) UpdateCategory(ctx context.Context, scope []uuid.UUID, arg fadb.UpdateAssetCategoryParams) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		cur, e := q.GetAssetCategory(ctx, arg.ID)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("asset category not found")
		}
		if e != nil {
			return e
		}
		if cur.OrganizationID == platform.SystemOrgID {
			return httpx.Forbidden("system reference data cannot be modified")
		}
		next, e = q.UpdateAssetCategory(ctx, arg)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.Conflict("STALE_ROW_VERSION", "rowVersion mismatch")
		}
		return e
	})
	return next, httpx.MapDBError(err, "")
}

func (r *FixedAssetRepository) DeleteCategory(ctx context.Context, scope []uuid.UUID, arg fadb.DeleteAssetCategoryParams) error {
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		cur, e := q.GetAssetCategory(ctx, arg.ID)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("asset category not found")
		}
		if e != nil {
			return e
		}
		if cur.OrganizationID == platform.SystemOrgID {
			return httpx.Forbidden("system reference data cannot be modified")
		}
		n, e := q.DeleteAssetCategory(ctx, arg)
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

func (r *FixedAssetRepository) CreateAsset(ctx context.Context, scope []uuid.UUID, arg fadb.CreateFixedAssetParams) (fadb.CreateFixedAssetRow, error) {
	var out fadb.CreateFixedAssetRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateFixedAsset(ctx, arg)
		return e
	})
	return out, httpx.MapDBError(err, "assetNo already exists")
}

func (r *FixedAssetRepository) GetAsset(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (fadb.FaFixedAsset, error) {
	var out fadb.FaFixedAsset
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).GetFixedAsset(ctx, id)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("fixed asset not found")
		}
		return e
	})
	return out, err
}

func (r *FixedAssetRepository) ListAssets(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]fadb.FaFixedAsset, int64, error) {
	var rows []fadb.FaFixedAsset
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountFixedAssets(ctx, status); e != nil {
			return e
		}
		rows, e = q.ListFixedAssets(ctx, fadb.ListFixedAssetsParams{Status: status, Limit: limit, Offset: offset})
		return e
	})
	return rows, total, err
}

func (r *FixedAssetRepository) UpdateAssetDraft(ctx context.Context, scope []uuid.UUID, arg fadb.UpdateFixedAssetDraftParams) (int32, error) {
	var next int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		a, e := q.GetFixedAsset(ctx, arg.ID)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("fixed asset not found")
		}
		if e != nil {
			return e
		}
		if a.Status != "Draft" {
			return httpx.Conflict("INVALID_STATE", "only Draft fixed asset can be updated")
		}
		if platform.NumericToRat(arg.SalvageValue).Cmp(platform.NumericToRat(a.AcquisitionCost)) > 0 {
			return httpx.Unprocessable("INVALID_SALVAGE", "salvageValue cannot exceed acquisitionCost")
		}
		next, e = q.UpdateFixedAssetDraft(ctx, arg)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.Conflict("STALE_ROW_VERSION", "rowVersion mismatch")
		}
		return e
	})
	return next, httpx.MapDBError(err, "")
}

func (r *FixedAssetRepository) DeleteAssetDraft(ctx context.Context, scope []uuid.UUID, arg fadb.DeleteFixedAssetDraftParams) error {
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		a, e := q.GetFixedAsset(ctx, arg.ID)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("fixed asset not found")
		}
		if e != nil {
			return e
		}
		if a.Status != "Draft" {
			return httpx.Conflict("INVALID_STATE", "only Draft fixed asset can be deleted")
		}
		n, e := q.DeleteFixedAssetDraft(ctx, arg)
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

func (r *FixedAssetRepository) CapitalizeAndEmit(ctx context.Context, scope []uuid.UUID, id uuid.UUID, rowVer int32, txDate pgtype.Date, actor, corr uuid.UUID) (int32, error) {
	var newVer int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		a, e := q.GetFixedAsset(ctx, id)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("fixed asset not found")
		}
		if e != nil {
			return e
		}
		if a.Status != "Draft" {
			return httpx.Conflict("INVALID_STATE", "only Draft fixed asset can be capitalized")
		}
		newVer, e = q.TransitionFixedAsset(ctx, fadb.TransitionFixedAssetParams{
			ID: id, RowVersion: rowVer, Status: "Active", NetBookValue: a.AcquisitionCost, UpdatedBy: &actor,
		})
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.Conflict("STALE_ROW_VERSION", "rowVersion mismatch")
		}
		if e != nil {
			return e
		}
		if _, e = q.CreateAssetTransaction(ctx, fadb.CreateAssetTransactionParams{
			OrganizationID: a.OrganizationID, FixedAssetID: a.ID, TransactionType: "Capitalization",
			TransactionDate: txDate, Amount: a.AcquisitionCost, CreatedBy: actor,
		}); e != nil {
			return e
		}
		return emit(ctx, q, "FixedAsset", "fixedasset.asset.acquired", a.ID, a.OrganizationID, corr, map[string]any{
			"assetNo": a.AssetNo, "reference": a.AssetNo, "amount": platform.FormatNumeric(a.AcquisitionCost, 4),
			"assetAccountId": a.AssetAccountID.String(),
		})
	})
	return newVer, httpx.MapDBError(err, "")
}

func (r *FixedAssetRepository) DepreciateAndEmit(ctx context.Context, scope []uuid.UUID, id uuid.UUID, rowVer int32, fiscalPeriodRef uuid.UUID, runDate pgtype.Date, unitsProduced, totalEstimatedUnits *big.Rat, actor, corr uuid.UUID) (fadb.CreateDepreciationEntryRow, string, string, int32, error) {
	var entryRow fadb.CreateDepreciationEntryRow
	var depStr, nbvStr string
	var newVer int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		a, e := q.GetFixedAsset(ctx, id)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("fixed asset not found")
		}
		if e != nil {
			return e
		}
		if a.Status != "Active" {
			return httpx.Conflict("INVALID_STATE", "only Active fixed asset can be depreciated")
		}
		costRat := platform.NumericToRat(a.AcquisitionCost)
		salvRat := platform.NumericToRat(a.SalvageValue)
		nbvRat := platform.NumericToRat(a.NetBookValue)
		depRat, newNBVRat, e := ComputeMonthlyDepreciationWithUnits(a.DepreciationMethod, costRat, salvRat, nbvRat, a.UsefulLifeMonths, unitsProduced, totalEstimatedUnits)
		if e != nil {
			return e
		}
		if depRat.Sign() <= 0 {
			return httpx.Unprocessable("FULLY_DEPRECIATED", "fixed asset is already fully depreciated")
		}
		accumRat := new(big.Rat).Sub(costRat, newNBVRat)
		depNum, _ := platform.RatToNumeric(depRat, 4)
		accumNum, _ := platform.RatToNumeric(accumRat, 4)
		newNBVNum, _ := platform.RatToNumeric(newNBVRat, 4)
		depStr = platform.FormatNumeric(depNum, 4)
		nbvStr = platform.FormatNumeric(newNBVNum, 4)

		newVer, e = q.TransitionFixedAsset(ctx, fadb.TransitionFixedAssetParams{
			ID: a.ID, RowVersion: rowVer, Status: "Active", NetBookValue: newNBVNum, UpdatedBy: &actor,
		})
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.Conflict("STALE_ROW_VERSION", "rowVersion mismatch")
		}
		if e != nil {
			return e
		}
		glRef := "FA-DEP-" + a.AssetNo
		entryRow, e = q.CreateDepreciationEntry(ctx, fadb.CreateDepreciationEntryParams{
			OrganizationID: a.OrganizationID, FixedAssetID: a.ID, FiscalPeriodRef: fiscalPeriodRef,
			DepreciationAmount: depNum, AccumulatedDepreciation: accumNum, NetBookValue: newNBVNum,
			IsPosted: true, GlJournalRef: &glRef, RunDate: runDate, CreatedBy: actor,
		})
		if e != nil {
			return e
		}
		return emit(ctx, q, "FixedAsset", "fixedasset.depreciation.posted", a.ID, a.OrganizationID, corr, map[string]any{
			"assetNo": a.AssetNo, "reference": glRef, "fiscalPeriodId": fiscalPeriodRef.String(),
			"amount": depStr, "netBookValue": nbvStr,
			"expenseAccountId": a.DepreciationExpenseAccountID.String(),
			"accumAccountId":   a.AccumDepreciationAccountID.String(),
		})
	})
	return entryRow, depStr, nbvStr, newVer, httpx.MapDBError(err, "depreciation already posted for this period")
}

func (r *FixedAssetRepository) DisposeAndEmit(ctx context.Context, scope []uuid.UUID, id uuid.UUID, rowVer int32, proceeds pgtype.Numeric, txDate pgtype.Date, notes *string, actor, corr uuid.UUID) (int32, string, error) {
	var newVer int32
	var gainLossStr string
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		a, e := q.GetFixedAsset(ctx, id)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("fixed asset not found")
		}
		if e != nil {
			return e
		}
		if a.Status != "Active" {
			return httpx.Conflict("INVALID_STATE", "only Active fixed asset can be disposed")
		}
		zeroNum, _ := platform.ParseNumeric("0.0000")
		glRat := new(big.Rat).Sub(platform.NumericToRat(proceeds), platform.NumericToRat(a.NetBookValue))
		glNum, _ := platform.RatToNumeric(glRat, 4)
		gainLossStr = platform.FormatNumeric(glNum, 4)

		newVer, e = q.TransitionFixedAsset(ctx, fadb.TransitionFixedAssetParams{
			ID: a.ID, RowVersion: rowVer, Status: "Disposed", NetBookValue: zeroNum, UpdatedBy: &actor,
		})
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.Conflict("STALE_ROW_VERSION", "rowVersion mismatch")
		}
		if e != nil {
			return e
		}
		if _, e = q.CreateAssetTransaction(ctx, fadb.CreateAssetTransactionParams{
			OrganizationID: a.OrganizationID, FixedAssetID: a.ID, TransactionType: "Disposal",
			TransactionDate: txDate, Amount: proceeds, GainLossAmount: glNum, Notes: notes, CreatedBy: actor,
		}); e != nil {
			return e
		}
		proceedsStr := platform.FormatNumeric(proceeds, 4)
		nbvStr := platform.FormatNumeric(a.NetBookValue, 4)
		costStr := platform.FormatNumeric(a.AcquisitionCost, 4)
		amtStr := proceedsStr
		if platform.NumericToRat(proceeds).Sign() <= 0 {
			if platform.NumericToRat(a.NetBookValue).Sign() > 0 {
				amtStr = nbvStr
			} else {
				amtStr = costStr
			}
		}
		return emit(ctx, q, "FixedAsset", "fixedasset.asset.disposed", a.ID, a.OrganizationID, corr, map[string]any{
			"assetNo":         a.AssetNo,
			"reference":       "FA-DISP-" + a.AssetNo,
			"amount":          amtStr,
			"proceedsAmount":  proceedsStr,
			"acquisitionCost": costStr,
			"netBookValue":    nbvStr,
			"gainLossAmount":  gainLossStr,
			"assetAccountId":  a.AssetAccountID.String(),
			"accumAccountId":  a.AccumDepreciationAccountID.String(),
		})
	})
	return newVer, gainLossStr, httpx.MapDBError(err, "")
}

func (r *FixedAssetRepository) ListDepreciationEntries(ctx context.Context, scope []uuid.UUID, assetID uuid.UUID) ([]fadb.FaDepreciationEntry, error) {
	var rows []fadb.FaDepreciationEntry
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		rows, e = r.q.WithTx(tx).ListDepreciationEntriesByAsset(ctx, assetID)
		return e
	})
	return rows, err
}

func (r *FixedAssetRepository) ListTransactions(ctx context.Context, scope []uuid.UUID, assetID uuid.UUID) ([]fadb.FaAssetTransaction, error) {
	var rows []fadb.FaAssetTransaction
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		rows, e = r.q.WithTx(tx).ListAssetTransactionsByAsset(ctx, assetID)
		return e
	})
	return rows, err
}

func (r *FixedAssetRepository) DrainAndPublish(ctx context.Context, limit int32, pub *eventbus.Publisher) (int, error) {
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
		if e := q.MarkOutboxPublished(ctx, fadb.MarkOutboxPublishedParams{ID: row.ID, CreatedAt: row.CreatedAt}); e != nil {
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

// ConsumeCapExAsset idempotently registers a Draft fixed asset from AP/Procurement CapEx events.
func (r *FixedAssetRepository) ConsumeCapExAsset(
	ctx context.Context, scope []uuid.UUID,
	mark fadb.MarkEventProcessedParams,
	asset fadb.CreateFixedAssetParams,
) (bool, error) {
	var created bool
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		n, e := q.MarkEventProcessed(ctx, mark)
		if e != nil || n == 0 {
			return e
		}
		if asset.AssetCategoryID == uuid.Nil {
			cats, e := q.ListAssetCategories(ctx, fadb.ListAssetCategoriesParams{Limit: 1, Offset: 0})
			if e != nil {
				return e
			}
			if len(cats) == 0 {
				return nil
			}
			asset.AssetCategoryID = cats[0].ID
			if asset.UsefulLifeMonths <= 0 && cats[0].DefaultUsefulLifeMonths != nil && *cats[0].DefaultUsefulLifeMonths > 0 {
				asset.UsefulLifeMonths = *cats[0].DefaultUsefulLifeMonths
			}
			if asset.DepreciationMethod == "" && cats[0].DefaultDepreciationMethod != "" {
				asset.DepreciationMethod = cats[0].DefaultDepreciationMethod
			}
		}
		if _, e := q.CreateFixedAsset(ctx, asset); e != nil {
			return e
		}
		created = true
		return nil
	})
	return created, err
}
