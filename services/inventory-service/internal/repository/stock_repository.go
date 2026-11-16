// Package repository manages RLS-scoped inventory_db transactions across
// warehouses, stock_balance, partitioned stock_movement, stock_reservation,
// transactional outbox, and consumer deduplication.
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

	"github.com/sinergiinvestasiproperti/erp/services/inventory-service/internal/inventorydb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

// ErrInsufficientStock indicates requested stock exceeds available balance.
var ErrInsufficientStock = errors.New("insufficient available stock")

// StockLineRequest identifies an item, quantity, and unit cost for stock adjustments/receipts/issues.
type StockLineRequest struct {
	ItemID    uuid.UUID
	Qty       pgtype.Numeric
	UnitPrice pgtype.Numeric
}

// StockRepository provides data access for inventory-service.
type StockRepository struct {
	db *platform.DB
	q  *inventorydb.Queries
}

// NewStockRepository creates a StockRepository backed by db.
func NewStockRepository(db *platform.DB) *StockRepository {
	return &StockRepository{db: db, q: inventorydb.New(db.Pool)}
}

// ConsumeAndCacheParty deduplicates and refreshes the local party read model.
func (r *StockRepository) ConsumeAndCacheParty(ctx context.Context, scope []uuid.UUID, mark inventorydb.MarkEventProcessedParams, up inventorydb.UpsertMasterdataCacheParams) (bool, error) {
	applied := false
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		n, e := q.MarkEventProcessed(ctx, mark)
		if e != nil || n == 0 {
			return e
		}
		if e = q.UpsertMasterdataCache(ctx, up); e != nil {
			return e
		}
		applied = true
		return nil
	})
	return applied, err
}

// CreateWarehouse inserts a warehouse master record.
func (r *StockRepository) CreateWarehouse(ctx context.Context, scope []uuid.UUID, arg inventorydb.CreateWarehouseParams) (inventorydb.CreateWarehouseRow, error) {
	var out inventorydb.CreateWarehouseRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateWarehouse(ctx, arg)
		return e
	})
	return out, err
}

// ListWarehousesWithCount returns paginated warehouses and total row count.
func (r *StockRepository) ListWarehousesWithCount(ctx context.Context, scope []uuid.UUID, arg inventorydb.ListWarehousesParams) ([]inventorydb.ListWarehousesRow, int64, error) {
	var rows []inventorydb.ListWarehousesRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if rows, e = q.ListWarehouses(ctx, arg); e != nil {
			return e
		}
		total, e = q.CountWarehouses(ctx)
		return e
	})
	return rows, total, err
}

// ListStockBalancesWithCount lists stock balances filtered by warehouse/item.
func (r *StockRepository) ListStockBalancesWithCount(ctx context.Context, scope []uuid.UUID, arg inventorydb.ListStockBalancesParams) ([]inventorydb.ListStockBalancesRow, int64, error) {
	var rows []inventorydb.ListStockBalancesRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if rows, e = q.ListStockBalances(ctx, arg); e != nil {
			return e
		}
		total, e = q.CountStockBalances(ctx, inventorydb.CountStockBalancesParams{
			WarehouseID: arg.WarehouseID,
			ItemID:      arg.ItemID,
		})
		return e
	})
	return rows, total, err
}

// ListStockMovementsWithCount lists stock movements filtered by warehouse/item.
func (r *StockRepository) ListStockMovementsWithCount(ctx context.Context, scope []uuid.UUID, arg inventorydb.ListStockMovementsParams) ([]inventorydb.ListStockMovementsRow, int64, error) {
	var rows []inventorydb.ListStockMovementsRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if rows, e = q.ListStockMovements(ctx, arg); e != nil {
			return e
		}
		total, e = q.CountStockMovements(ctx, inventorydb.CountStockMovementsParams{
			WarehouseID: arg.WarehouseID,
			ItemID:      arg.ItemID,
		})
		return e
	})
	return rows, total, err
}

// RecordStockMovement posts a manual Receipt, Issue, or Adjustment and atomically updates stock_balance.
func (r *StockRepository) RecordStockMovement(ctx context.Context, scope []uuid.UUID, arg inventorydb.CreateStockMovementParams, unitCost pgtype.Numeric) (inventorydb.CreateStockMovementRow, error) {
	var created inventorydb.CreateStockMovementRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		qtyRat := platform.NumericToRat(arg.Quantity)
		if qtyRat == nil || qtyRat.Sign() == 0 {
			return errors.New("quantity cannot be zero")
		}
		switch arg.MovementType {
		case "Receipt":
			if qtyRat.Sign() < 0 {
				return errors.New("receipt quantity must be positive")
			}
			if _, e := q.UpsertReceiptBalance(ctx, inventorydb.UpsertReceiptBalanceParams{
				OrganizationID: arg.OrganizationID,
				WarehouseID:    arg.WarehouseID,
				ItemID:         arg.ItemID,
				OnHandQty:      arg.Quantity,
				AvgCost:        unitCost,
				CreatedBy:      arg.CreatedBy,
			}); e != nil {
				return e
			}
		case "Issue":
			absQty := arg.Quantity
			if qtyRat.Sign() < 0 {
				absNum, e := platform.RatToNumeric(new(big.Rat).Abs(qtyRat), 4)
				if e != nil {
					return e
				}
				absQty = absNum
			} else {
				negNum, e := platform.RatToNumeric(new(big.Rat).Neg(qtyRat), 4)
				if e != nil {
					return e
				}
				arg.Quantity = negNum
			}
			bal, e := q.LockStockBalanceForItemWarehouse(ctx, inventorydb.LockStockBalanceForItemWarehouseParams{
				OrganizationID: arg.OrganizationID,
				WarehouseID:    arg.WarehouseID,
				ItemID:         arg.ItemID,
			})
			if errors.Is(e, pgx.ErrNoRows) {
				return ErrInsufficientStock
			}
			if e != nil {
				return e
			}
			if _, e := q.DeductAvailableStockBalance(ctx, inventorydb.DeductAvailableStockBalanceParams{
				ID:        bal.ID,
				OnHandQty: absQty,
				UpdatedBy: &arg.CreatedBy,
			}); errors.Is(e, pgx.ErrNoRows) {
				return ErrInsufficientStock
			} else if e != nil {
				return e
			}
		case "Adjustment":
			if qtyRat.Sign() > 0 {
				if _, e := q.UpsertReceiptBalance(ctx, inventorydb.UpsertReceiptBalanceParams{
					OrganizationID: arg.OrganizationID,
					WarehouseID:    arg.WarehouseID,
					ItemID:         arg.ItemID,
					OnHandQty:      arg.Quantity,
					AvgCost:        unitCost,
					CreatedBy:      arg.CreatedBy,
				}); e != nil {
					return e
				}
			} else {
				absNum, e := platform.RatToNumeric(new(big.Rat).Abs(qtyRat), 4)
				if e != nil {
					return e
				}
				bal, e := q.LockStockBalanceForItemWarehouse(ctx, inventorydb.LockStockBalanceForItemWarehouseParams{
					OrganizationID: arg.OrganizationID,
					WarehouseID:    arg.WarehouseID,
					ItemID:         arg.ItemID,
				})
				if errors.Is(e, pgx.ErrNoRows) {
					return ErrInsufficientStock
				}
				if e != nil {
					return e
				}
				if _, e := q.DeductAvailableStockBalance(ctx, inventorydb.DeductAvailableStockBalanceParams{
					ID:        bal.ID,
					OnHandQty: absNum,
					UpdatedBy: &arg.CreatedBy,
				}); errors.Is(e, pgx.ErrNoRows) {
					return ErrInsufficientStock
				} else if e != nil {
					return e
				}
			}
		}
		var e error
		created, e = q.CreateStockMovement(ctx, arg)
		return e
	})
	return created, err
}

// CreateReservationAndEmit locks available stock_balance, increments reserved_qty,
// creates inv.stock_reservation, and appends inventory.stock.reserved to inv.outbox.
func (r *StockRepository) CreateReservationAndEmit(ctx context.Context, scope []uuid.UUID, arg inventorydb.CreateReservationParams, ev inventorydb.InsertOutboxParams) (inventorydb.CreateReservationRow, error) {
	var created inventorydb.CreateReservationRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		bal, e := q.LockStockBalanceForItemWarehouse(ctx, inventorydb.LockStockBalanceForItemWarehouseParams{
			OrganizationID: arg.OrganizationID,
			WarehouseID:    arg.WarehouseID,
			ItemID:         arg.ItemID,
		})
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrInsufficientStock
		}
		if e != nil {
			return e
		}
		if _, e := q.ReserveStockBalance(ctx, inventorydb.ReserveStockBalanceParams{
			ID:          bal.ID,
			ReservedQty: arg.ReservedQty,
			UpdatedBy:   &arg.CreatedBy,
		}); errors.Is(e, pgx.ErrNoRows) {
			return ErrInsufficientStock
		} else if e != nil {
			return e
		}
		if created, e = q.CreateReservation(ctx, arg); e != nil {
			return e
		}
		ev.AggregateID = created.ID
		return q.InsertOutbox(ctx, ev)
	})
	return created, err
}

// IssueAndEmit consumes a reservation Reserved->Consumed, decrements stock_balance on_hand/reserved,
// records an Issue movement, and writes inventory.goods.issued in the same transaction.
func (r *StockRepository) IssueAndEmit(ctx context.Context, scope []uuid.UUID, arg inventorydb.IssueReservationParams, corrID uuid.UUID, moveDate pgtype.Date, ev inventorydb.InsertOutboxParams) (int32, int64, error) {
	var newVer int32
	var affected int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		res, e := q.GetReservation(ctx, arg.ID)
		if errors.Is(e, pgx.ErrNoRows) {
			return nil
		}
		if e != nil {
			return e
		}
		v, e := q.IssueReservation(ctx, arg)
		if errors.Is(e, pgx.ErrNoRows) {
			return nil
		}
		if e != nil {
			return e
		}
		newVer = v
		affected = 1

		bal, e := q.LockStockBalanceForItemWarehouse(ctx, inventorydb.LockStockBalanceForItemWarehouseParams{
			OrganizationID: res.OrganizationID,
			WarehouseID:    res.WarehouseID,
			ItemID:         res.ItemID,
		})
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrInsufficientStock
		}
		if e != nil {
			return e
		}
		if _, e := q.ConsumeReservedStockBalance(ctx, inventorydb.ConsumeReservedStockBalanceParams{
			ID:        bal.ID,
			OnHandQty: res.ReservedQty,
			UpdatedBy: arg.UpdatedBy,
		}); errors.Is(e, pgx.ErrNoRows) {
			return ErrInsufficientStock
		} else if e != nil {
			return e
		}
		qtyRat := platform.NumericToRat(res.ReservedQty)
		if qtyRat == nil {
			qtyRat = new(big.Rat)
		}
		negNum, e := platform.RatToNumeric(new(big.Rat).Neg(qtyRat), 4)
		if e != nil {
			return e
		}
		actor := res.OrganizationID
		if arg.UpdatedBy != nil {
			actor = *arg.UpdatedBy
		}
		ref := res.SalesOrderRef
		if _, e := q.CreateStockMovement(ctx, inventorydb.CreateStockMovementParams{
			OrganizationID:    res.OrganizationID,
			MovementDate:      moveDate,
			WarehouseID:       res.WarehouseID,
			ItemID:            res.ItemID,
			MovementType:      "Issue",
			Quantity:          negNum,
			SourceDocumentRef: &ref,
			CorrelationID:     corrID,
			CreatedBy:         actor,
		}); e != nil {
			return e
		}
		return q.InsertOutbox(ctx, ev)
	})
	return newVer, affected, err
}

// ReleaseReservation releases a Reserved reservation back to available balance.
func (r *StockRepository) ReleaseReservation(ctx context.Context, scope []uuid.UUID, arg inventorydb.ReleaseReservationParams) (int32, int64, error) {
	var newVer int32
	var affected int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		res, e := q.GetReservation(ctx, arg.ID)
		if errors.Is(e, pgx.ErrNoRows) {
			return nil
		}
		if e != nil {
			return e
		}
		v, e := q.ReleaseReservation(ctx, arg)
		if errors.Is(e, pgx.ErrNoRows) {
			return nil
		}
		if e != nil {
			return e
		}
		newVer = v
		affected = 1
		bal, e := q.LockStockBalanceForItemWarehouse(ctx, inventorydb.LockStockBalanceForItemWarehouseParams{
			OrganizationID: res.OrganizationID,
			WarehouseID:    res.WarehouseID,
			ItemID:         res.ItemID,
		})
		if errors.Is(e, pgx.ErrNoRows) {
			return nil
		}
		if e != nil {
			return e
		}
		_, e = q.ReleaseStockBalance(ctx, inventorydb.ReleaseStockBalanceParams{
			ID:          bal.ID,
			ReservedQty: res.ReservedQty,
			UpdatedBy:   arg.UpdatedBy,
		})
		return e
	})
	return newVer, affected, err
}

// GetByID fetches one stock reservation.
func (r *StockRepository) GetByID(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (inventorydb.GetReservationRow, error) {
	var out inventorydb.GetReservationRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).GetReservation(ctx, id)
		return e
	})
	return out, err
}

// ListWithCount returns paginated reservations and total count.
func (r *StockRepository) ListWithCount(ctx context.Context, scope []uuid.UUID, arg inventorydb.ListReservationsParams) ([]inventorydb.ListReservationsRow, int64, error) {
	var rows []inventorydb.ListReservationsRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if rows, e = q.ListReservations(ctx, arg); e != nil {
			return e
		}
		total, e = q.CountReservations(ctx)
		return e
	})
	return rows, total, err
}

// FulfillSalesOrder consumes stock balances for sales.order.confirmed idempotently;
// emits inventory.goods.issued on success, or inventory.reservation.failed when stock is insufficient.
func (r *StockRepository) FulfillSalesOrder(ctx context.Context, org uuid.UUID, corr uuid.UUID, actor uuid.UUID, moveDate pgtype.Date, ref string, warehouseID *uuid.UUID, lines []StockLineRequest, mark inventorydb.MarkEventProcessedParams, evOK, evFail inventorydb.InsertOutboxParams) (bool, error) {
	applied := false
	err := r.db.WithOrgScope(ctx, []uuid.UUID{org}, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		n, e := q.MarkEventProcessed(ctx, mark)
		if e != nil || n == 0 {
			return e
		}
		applied = true

		type deduction struct {
			balID       uuid.UUID
			warehouseID uuid.UUID
			itemID      uuid.UUID
			qty         pgtype.Numeric
		}
		if len(lines) == 0 {
			return q.InsertOutbox(ctx, evFail)
		}

		remaining := make(map[uuid.UUID]*big.Rat, len(lines))
		plan := make([]deduction, 0, len(lines))
		for _, l := range lines {
			reqRat := platform.NumericToRat(l.Qty)
			if reqRat == nil || reqRat.Sign() <= 0 {
				return q.InsertOutbox(ctx, evFail)
			}
			var balID, whID uuid.UUID
			var availNum pgtype.Numeric
			if warehouseID != nil {
				row, errLock := q.LockStockBalanceForItemWarehouse(ctx, inventorydb.LockStockBalanceForItemWarehouseParams{
					OrganizationID: org,
					WarehouseID:    *warehouseID,
					ItemID:         l.ItemID,
				})
				if errors.Is(errLock, pgx.ErrNoRows) {
					return q.InsertOutbox(ctx, evFail)
				}
				if errLock != nil {
					return errLock
				}
				balID, whID, availNum = row.ID, row.WarehouseID, row.AvailableQty
			} else {
				row, errLock := q.LockBestStockBalanceForItem(ctx, inventorydb.LockBestStockBalanceForItemParams{
					OrganizationID: org,
					ItemID:         l.ItemID,
				})
				if errors.Is(errLock, pgx.ErrNoRows) {
					return q.InsertOutbox(ctx, evFail)
				}
				if errLock != nil {
					return errLock
				}
				balID, whID, availNum = row.ID, row.WarehouseID, row.AvailableQty
			}

			rem, ok := remaining[balID]
			if !ok {
				availRat := platform.NumericToRat(availNum)
				if availRat == nil {
					availRat = new(big.Rat)
				}
				rem = new(big.Rat).Set(availRat)
				remaining[balID] = rem
			}
			if rem.Cmp(reqRat) < 0 {
				return q.InsertOutbox(ctx, evFail)
			}
			rem.Sub(rem, reqRat)
			plan = append(plan, deduction{balID: balID, warehouseID: whID, itemID: l.ItemID, qty: l.Qty})
		}

		for _, d := range plan {
			if _, e := q.DeductAvailableStockBalance(ctx, inventorydb.DeductAvailableStockBalanceParams{
				ID:        d.balID,
				OnHandQty: d.qty,
				UpdatedBy: &actor,
			}); errors.Is(e, pgx.ErrNoRows) {
				return ErrInsufficientStock
			} else if e != nil {
				return e
			}
			reqRat := platform.NumericToRat(d.qty)
			negNum, e := platform.RatToNumeric(new(big.Rat).Neg(reqRat), 4)
			if e != nil {
				return e
			}
			if _, e := q.CreateStockMovement(ctx, inventorydb.CreateStockMovementParams{
				OrganizationID:    org,
				MovementDate:      moveDate,
				WarehouseID:       d.warehouseID,
				ItemID:            d.itemID,
				MovementType:      "Issue",
				Quantity:          negNum,
				SourceDocumentRef: &ref,
				CorrelationID:     corr,
				CreatedBy:         actor,
			}); e != nil {
				return e
			}
		}
		return q.InsertOutbox(ctx, evOK)
	})
	return applied, err
}

// ReceiveProcurementGoods processes procurement.goods.received idempotently, incrementing
// stock_balance and inserting Receipt stock movements when a warehouse exists.
func (r *StockRepository) ReceiveProcurementGoods(ctx context.Context, org, corr, actor uuid.UUID, moveDate pgtype.Date, ref string, warehouseID *uuid.UUID, lines []StockLineRequest, mark inventorydb.MarkEventProcessedParams) (bool, error) {
	applied := false
	err := r.db.WithOrgScope(ctx, []uuid.UUID{org}, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		n, e := q.MarkEventProcessed(ctx, mark)
		if e != nil || n == 0 {
			return e
		}
		applied = true
		if len(lines) == 0 {
			return nil
		}

		var targetWH uuid.UUID
		if warehouseID != nil {
			if wh, e := q.GetWarehouse(ctx, *warehouseID); e == nil {
				targetWH = wh.ID
			}
		}
		if targetWH == uuid.Nil {
			wh, e := q.GetDefaultWarehouse(ctx)
			if errors.Is(e, pgx.ErrNoRows) {
				return nil // no warehouse provisioned yet in org; event marked processed
			}
			if e != nil {
				return e
			}
			targetWH = wh.ID
		}

		for _, l := range lines {
			reqRat := platform.NumericToRat(l.Qty)
			if reqRat == nil || reqRat.Sign() <= 0 {
				continue
			}
			if _, e := q.UpsertReceiptBalance(ctx, inventorydb.UpsertReceiptBalanceParams{
				OrganizationID: org,
				WarehouseID:    targetWH,
				ItemID:         l.ItemID,
				OnHandQty:      l.Qty,
				AvgCost:        l.UnitPrice,
				CreatedBy:      actor,
			}); e != nil {
				return e
			}
			if _, e := q.CreateStockMovement(ctx, inventorydb.CreateStockMovementParams{
				OrganizationID:    org,
				MovementDate:      moveDate,
				WarehouseID:       targetWH,
				ItemID:            l.ItemID,
				MovementType:      "Receipt",
				Quantity:          l.Qty,
				SourceDocumentRef: &ref,
				CorrelationID:     corr,
				CreatedBy:         actor,
			}); e != nil {
				return e
			}
		}
		return nil
	})
	return applied, err
}

// DrainAndPublish drains unpublished outbox events under row locks and publishes them to Kafka.
func (r *StockRepository) DrainAndPublish(ctx context.Context, pub *eventbus.Publisher, limit int32) (int, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin outbox tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
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
		if e := pub.Publish(ctx, env); e != nil {
			pubErr = e
			break
		}
		if e := q.MarkOutboxPublished(ctx, inventorydb.MarkOutboxPublishedParams{ID: row.ID, CreatedAt: row.CreatedAt}); e != nil {
			return published, e
		}
		published++
	}
	if published > 0 {
		if err := tx.Commit(ctx); err != nil {
			return 0, fmt.Errorf("commit outbox tx: %w", err)
		}
		committed = true
	}
	return published, pubErr
}
