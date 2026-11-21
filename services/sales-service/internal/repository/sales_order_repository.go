package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/sales-service/internal/salesdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

// SalesOrderDetail aggregates a sales order header with its ordered lines.
type SalesOrderDetail struct {
	Header salesdb.GetSalesOrderRow
	Lines  []salesdb.ListSalesOrderLinesRow
}

// DeliveryDetail aggregates a delivery header with its delivered lines.
type DeliveryDetail struct {
	Header salesdb.GetDeliveryRow
	Lines  []salesdb.ListDeliveryLinesRow
}

// CreateSOLineInput carries validated fixed-point line fields for SO creation/update.
type CreateSOLineInput struct {
	LineNo    int32
	ItemID    uuid.UUID
	OrderQty  pgtype.Numeric
	UnitPrice pgtype.Numeric
}

// CreateSOInput creates a Draft sales order header and its lines atomically.
type CreateSOInput struct {
	OrganizationID uuid.UUID
	SoNo           string
	CustomerID     uuid.UUID
	OrderDate      pgtype.Date
	CurrencyCode   string
	TotalAmount    pgtype.Numeric
	Lines          []CreateSOLineInput
	CorrelationID  uuid.UUID
	CreatedBy      uuid.UUID
}

// UpdateDraftSOInput updates a Draft sales order header and replaces its lines.
type UpdateDraftSOInput struct {
	ID           uuid.UUID
	RowVersion   int32
	CustomerID   uuid.UUID
	OrderDate    pgtype.Date
	CurrencyCode string
	TotalAmount  pgtype.Numeric
	Lines        []CreateSOLineInput
	UpdatedBy    uuid.UUID
}

// CreateDeliveryInput posts a full delivery for a Confirmed sales order.
type CreateDeliveryInput struct {
	SalesOrderID  uuid.UUID
	RowVersion    int32
	DeliveryNo    string
	WarehouseID   uuid.UUID
	DeliveryDate  pgtype.Date
	CorrelationID uuid.UUID
	UpdatedBy     uuid.UUID
}

// SalesOrderRepository persists sales orders, deliveries, outbox events, and caches.
type SalesOrderRepository struct {
	db *platform.DB
	q  *salesdb.Queries
}

func NewSalesOrderRepository(db *platform.DB) *SalesOrderRepository {
	var q *salesdb.Queries
	if db != nil && db.Pool != nil {
		q = salesdb.New(db.Pool)
	}
	return &SalesOrderRepository{db: db, q: q}
}

func (r *SalesOrderRepository) CreateWithLines(ctx context.Context, scope []uuid.UUID, in CreateSOInput) (SalesOrderDetail, error) {
	var out SalesOrderDetail
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		qt := r.q.WithTx(tx)
		row, err := qt.CreateSalesOrder(ctx, salesdb.CreateSalesOrderParams{
			OrganizationID: in.OrganizationID,
			SoNo:           in.SoNo,
			CustomerID:     in.CustomerID,
			OrderDate:      in.OrderDate,
			CurrencyCode:   in.CurrencyCode,
			TotalAmount:    in.TotalAmount,
			CorrelationID:  in.CorrelationID,
			CreatedBy:      in.CreatedBy,
		})
		if err != nil {
			return httpx.MapDBError(err, "soNo sudah digunakan")
		}
		for _, l := range in.Lines {
			if err := qt.CreateSalesOrderLine(ctx, salesdb.CreateSalesOrderLineParams{
				OrganizationID: in.OrganizationID,
				SalesOrderID:   row.ID,
				LineNo:         l.LineNo,
				ItemID:         l.ItemID,
				OrderQty:       l.OrderQty,
				UnitPrice:      l.UnitPrice,
				CreatedBy:      in.CreatedBy,
			}); err != nil {
				return httpx.MapDBError(err, "duplikat lineNo pada sales order")
			}
		}
		hdr, err := qt.GetSalesOrder(ctx, row.ID)
		if err != nil {
			return httpx.MapDBError(err, "")
		}
		lines, err := qt.ListSalesOrderLines(ctx, row.ID)
		if err != nil {
			return httpx.MapDBError(err, "")
		}
		out = SalesOrderDetail{Header: hdr, Lines: lines}
		return nil
	})
	return out, err
}

func (r *SalesOrderRepository) GetByID(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (SalesOrderDetail, error) {
	var out SalesOrderDetail
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		qt := r.q.WithTx(tx)
		hdr, err := qt.GetSalesOrder(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("sales order tidak ditemukan")
			}
			return httpx.MapDBError(err, "")
		}
		lines, err := qt.ListSalesOrderLines(ctx, id)
		if err != nil {
			return httpx.MapDBError(err, "")
		}
		out = SalesOrderDetail{Header: hdr, Lines: lines}
		return nil
	})
	return out, err
}

func (r *SalesOrderRepository) ListWithCount(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]salesdb.ListSalesOrdersRow, int64, error) {
	var items []salesdb.ListSalesOrdersRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		qt := r.q.WithTx(tx)
		var err error
		total, err = qt.CountSalesOrders(ctx, status)
		if err != nil {
			return httpx.MapDBError(err, "")
		}
		items, err = qt.ListSalesOrders(ctx, salesdb.ListSalesOrdersParams{
			Limit:  limit,
			Offset: offset,
			Status: status,
		})
		if err != nil {
			return httpx.MapDBError(err, "")
		}
		return nil
	})
	return items, total, err
}

func (r *SalesOrderRepository) UpdateDraftWithLines(ctx context.Context, scope []uuid.UUID, in UpdateDraftSOInput) (SalesOrderDetail, error) {
	var out SalesOrderDetail
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		qt := r.q.WithTx(tx)
		cur, err := qt.GetSalesOrder(ctx, in.ID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("sales order tidak ditemukan")
			}
			return httpx.MapDBError(err, "")
		}
		if cur.Status != "Draft" {
			return httpx.Conflict("INVALID_STATE", "hanya sales order berstatus Draft yang dapat diubah")
		}
		if cur.RowVersion != in.RowVersion {
			return httpx.Conflict("VERSION_CONFLICT", "rowVersion tidak cocok (data sudah diubah)")
		}
		updBy := in.UpdatedBy
		if _, err := qt.UpdateDraftSalesOrder(ctx, salesdb.UpdateDraftSalesOrderParams{
			ID:           in.ID,
			RowVersion:   in.RowVersion,
			CustomerID:   in.CustomerID,
			OrderDate:    in.OrderDate,
			CurrencyCode: in.CurrencyCode,
			TotalAmount:  in.TotalAmount,
			UpdatedBy:    &updBy,
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.Conflict("VERSION_CONFLICT", "sales order berubah saat diperbarui")
			}
			return httpx.MapDBError(err, "")
		}
		if err := qt.DeleteSalesOrderLines(ctx, in.ID); err != nil {
			return httpx.MapDBError(err, "")
		}
		for _, l := range in.Lines {
			if err := qt.CreateSalesOrderLine(ctx, salesdb.CreateSalesOrderLineParams{
				OrganizationID: cur.OrganizationID,
				SalesOrderID:   in.ID,
				LineNo:         l.LineNo,
				ItemID:         l.ItemID,
				OrderQty:       l.OrderQty,
				UnitPrice:      l.UnitPrice,
				CreatedBy:      in.UpdatedBy,
			}); err != nil {
				return httpx.MapDBError(err, "duplikat lineNo pada sales order")
			}
		}
		hdr, err := qt.GetSalesOrder(ctx, in.ID)
		if err != nil {
			return httpx.MapDBError(err, "")
		}
		lines, err := qt.ListSalesOrderLines(ctx, in.ID)
		if err != nil {
			return httpx.MapDBError(err, "")
		}
		out = SalesOrderDetail{Header: hdr, Lines: lines}
		return nil
	})
	return out, err
}

func (r *SalesOrderRepository) DeleteDraft(ctx context.Context, scope []uuid.UUID, id uuid.UUID, version int32, updatedBy uuid.UUID) error {
	return r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		qt := r.q.WithTx(tx)
		cur, err := qt.GetSalesOrder(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("sales order tidak ditemukan")
			}
			return httpx.MapDBError(err, "")
		}
		if cur.Status != "Draft" {
			return httpx.Conflict("INVALID_STATE", "hanya sales order berstatus Draft yang dapat dihapus")
		}
		if cur.RowVersion != version {
			return httpx.Conflict("VERSION_CONFLICT", "rowVersion tidak cocok")
		}
		n, err := qt.SoftDeleteDraftSalesOrder(ctx, salesdb.SoftDeleteDraftSalesOrderParams{
			ID:         id,
			RowVersion: version,
			UpdatedBy:  &updatedBy,
		})
		if err != nil {
			return httpx.MapDBError(err, "")
		}
		if n == 0 {
			return httpx.Conflict("VERSION_CONFLICT", "sales order gagal dihapus karena perubahan paralel")
		}
		return nil
	})
}

type confirmedSOLinePayload struct {
	LineNo    int32  `json:"lineNo"`
	ItemID    string `json:"itemId"`
	OrderQty  string `json:"orderQty"`
	UnitPrice string `json:"unitPrice"`
}

func (r *SalesOrderRepository) ConfirmAndEmit(ctx context.Context, scope []uuid.UUID, id uuid.UUID, version int32, warehouseID *uuid.UUID, updatedBy, corrID uuid.UUID) (SalesOrderDetail, error) {
	var out SalesOrderDetail
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		qt := r.q.WithTx(tx)
		hdr, err := qt.GetSalesOrder(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("sales order tidak ditemukan")
			}
			return httpx.MapDBError(err, "")
		}
		if hdr.Status != "Draft" {
			return httpx.Conflict("INVALID_STATE", "hanya sales order berstatus Draft yang dapat dikonfirmasi")
		}
		if hdr.RowVersion != version {
			return httpx.Conflict("VERSION_CONFLICT", "rowVersion tidak cocok")
		}
		lines, err := qt.ListSalesOrderLines(ctx, id)
		if err != nil {
			return httpx.MapDBError(err, "")
		}
		if len(lines) == 0 {
			return httpx.Unprocessable("EMPTY_LINES", "sales order wajib memiliki minimal satu baris item sebelum dikonfirmasi")
		}
		newVer, err := qt.ConfirmSalesOrder(ctx, salesdb.ConfirmSalesOrderParams{
			ID:         id,
			RowVersion: version,
			UpdatedBy:  &updatedBy,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.Conflict("VERSION_CONFLICT", "sales order gagal dikonfirmasi karena perubahan paralel")
			}
			return httpx.MapDBError(err, "")
		}
		hdr.Status = "Confirmed"
		hdr.RowVersion = newVer

		evLines := make([]confirmedSOLinePayload, 0, len(lines))
		for _, l := range lines {
			evLines = append(evLines, confirmedSOLinePayload{
				LineNo:    l.LineNo,
				ItemID:    l.ItemID.String(),
				OrderQty:  platform.FormatNumeric(l.OrderQty, 4),
				UnitPrice: platform.FormatNumeric(l.UnitPrice, 4),
			})
		}
		payloadMap := map[string]any{
			"soNo":         hdr.SoNo,
			"customerId":   hdr.CustomerID.String(),
			"orderDate":    platform.FormatDate(hdr.OrderDate),
			"currencyCode": hdr.CurrencyCode,
			"totalAmount":  platform.FormatNumeric(hdr.TotalAmount, 4),
			"status":       "Confirmed",
			"lines":        evLines,
		}
		if warehouseID != nil && *warehouseID != uuid.Nil {
			payloadMap["warehouseId"] = warehouseID.String()
		}
		payload, err := json.Marshal(payloadMap)
		if err != nil {
			return err
		}
		if err := qt.InsertOutbox(ctx, salesdb.InsertOutboxParams{
			AggregateType:  "sales_order",
			AggregateID:    id,
			EventType:      "sales.order.confirmed",
			Payload:        payload,
			OrganizationID: hdr.OrganizationID,
			CorrelationID:  corrID,
		}); err != nil {
			return httpx.MapDBError(err, "")
		}
		out = SalesOrderDetail{Header: hdr, Lines: lines}
		return nil
	})
	return out, err
}

func (r *SalesOrderRepository) CreateDelivery(ctx context.Context, scope []uuid.UUID, in CreateDeliveryInput) (DeliveryDetail, error) {
	var out DeliveryDetail
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		qt := r.q.WithTx(tx)
		hdr, err := qt.GetSalesOrder(ctx, in.SalesOrderID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("sales order tidak ditemukan")
			}
			return httpx.MapDBError(err, "")
		}
		if hdr.Status != "Confirmed" {
			return httpx.Conflict("INVALID_STATE", "hanya sales order berstatus Confirmed yang dapat dikirim")
		}
		if hdr.RowVersion != in.RowVersion {
			return httpx.Conflict("VERSION_CONFLICT", "rowVersion tidak cocok")
		}
		lines, err := qt.ListSalesOrderLines(ctx, in.SalesOrderID)
		if err != nil {
			return httpx.MapDBError(err, "")
		}
		if len(lines) == 0 {
			return httpx.Unprocessable("EMPTY_LINES", "sales order tidak memiliki baris untuk dikirim")
		}

		if _, err := qt.MarkSalesOrderDelivered(ctx, salesdb.MarkSalesOrderDeliveredParams{
			ID:         in.SalesOrderID,
			RowVersion: in.RowVersion,
			UpdatedBy:  &in.UpdatedBy,
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.Conflict("VERSION_CONFLICT", "sales order gagal diperbarui menjadi Delivered")
			}
			return httpx.MapDBError(err, "")
		}
		if err := qt.MarkSOLinesDelivered(ctx, salesdb.MarkSOLinesDeliveredParams{
			SalesOrderID: in.SalesOrderID,
			UpdatedBy:    &in.UpdatedBy,
		}); err != nil {
			return httpx.MapDBError(err, "")
		}

		delNo := in.DeliveryNo
		if delNo == "" {
			delNo = "DO-" + hdr.SoNo
		}
		whID := in.WarehouseID
		if whID == uuid.Nil {
			whID = hdr.OrganizationID
		}
		delDate := in.DeliveryDate
		if !delDate.Valid {
			now := time.Now().UTC()
			delDate = pgtype.Date{Time: time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC), Valid: true}
		}
		dRow, err := qt.CreateDelivery(ctx, salesdb.CreateDeliveryParams{
			OrganizationID: hdr.OrganizationID,
			DeliveryNo:     delNo,
			SalesOrderID:   in.SalesOrderID,
			WarehouseID:    whID,
			DeliveryDate:   delDate,
			CorrelationID:  in.CorrelationID,
			CreatedBy:      in.UpdatedBy,
		})
		if err != nil {
			return httpx.MapDBError(err, "deliveryNo sudah digunakan")
		}
		for _, l := range lines {
			if err := qt.CreateDeliveryLine(ctx, salesdb.CreateDeliveryLineParams{
				OrganizationID: hdr.OrganizationID,
				DeliveryID:     dRow.ID,
				SoLineID:       l.ID,
				LineNo:         l.LineNo,
				ItemID:         l.ItemID,
				DeliveredQty:   l.OrderQty,
				CreatedBy:      in.UpdatedBy,
			}); err != nil {
				return httpx.MapDBError(err, "duplikat lineNo pada delivery")
			}
		}

		payload, err := json.Marshal(map[string]any{
			"deliveryId":   dRow.ID.String(),
			"deliveryNo":   dRow.DeliveryNo,
			"salesOrderId": in.SalesOrderID.String(),
			"soNo":         hdr.SoNo,
			"customerId":   hdr.CustomerID.String(),
			"warehouseId":  whID.String(),
			"status":       "Delivered",
		})
		if err != nil {
			return err
		}
		if err := qt.InsertOutbox(ctx, salesdb.InsertOutboxParams{
			AggregateType:  "delivery",
			AggregateID:    dRow.ID,
			EventType:      "sales.delivery.completed",
			Payload:        payload,
			OrganizationID: hdr.OrganizationID,
			CorrelationID:  in.CorrelationID,
		}); err != nil {
			return httpx.MapDBError(err, "")
		}

		dHdr, err := qt.GetDelivery(ctx, dRow.ID)
		if err != nil {
			return httpx.MapDBError(err, "")
		}
		dLines, err := qt.ListDeliveryLines(ctx, dRow.ID)
		if err != nil {
			return httpx.MapDBError(err, "")
		}
		out = DeliveryDetail{Header: dHdr, Lines: dLines}
		return nil
	})
	return out, err
}

func (r *SalesOrderRepository) GetDelivery(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (DeliveryDetail, error) {
	var out DeliveryDetail
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		qt := r.q.WithTx(tx)
		hdr, err := qt.GetDelivery(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("delivery tidak ditemukan")
			}
			return httpx.MapDBError(err, "")
		}
		lines, err := qt.ListDeliveryLines(ctx, id)
		if err != nil {
			return httpx.MapDBError(err, "")
		}
		out = DeliveryDetail{Header: hdr, Lines: lines}
		return nil
	})
	return out, err
}

func (r *SalesOrderRepository) ListDeliveriesWithCount(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]salesdb.ListDeliveriesRow, int64, error) {
	var items []salesdb.ListDeliveriesRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		qt := r.q.WithTx(tx)
		var err error
		total, err = qt.CountDeliveries(ctx)
		if err != nil {
			return httpx.MapDBError(err, "")
		}
		items, err = qt.ListDeliveries(ctx, salesdb.ListDeliveriesParams{
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			return httpx.MapDBError(err, "")
		}
		return nil
	})
	return items, total, err
}

func (r *SalesOrderRepository) ConsumeAndCancel(ctx context.Context, orgID, eventID, soID uuid.UUID, eventType string) error {
	return r.db.WithOrgScope(ctx, []uuid.UUID{orgID}, func(tx pgx.Tx) error {
		qt := r.q.WithTx(tx)
		n, err := qt.MarkEventProcessed(ctx, salesdb.MarkEventProcessedParams{
			EventID:   eventID,
			EventType: eventType,
		})
		if err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
		_, err = qt.CancelSalesOrder(ctx, salesdb.CancelSalesOrderParams{
			ID:        soID,
			UpdatedBy: nil,
		})
		return err
	})
}

func (r *SalesOrderRepository) ConsumeAndCreateDraftSO(ctx context.Context, orgID, eventID uuid.UUID, eventType string, in CreateSOInput) error {
	return r.db.WithOrgScope(ctx, []uuid.UUID{orgID}, func(tx pgx.Tx) error {
		qt := r.q.WithTx(tx)
		n, err := qt.MarkEventProcessed(ctx, salesdb.MarkEventProcessedParams{
			EventID:   eventID,
			EventType: eventType,
		})
		if err != nil || n == 0 {
			return err
		}
		created, err := qt.CreateSalesOrder(ctx, salesdb.CreateSalesOrderParams{
			OrganizationID: in.OrganizationID,
			SoNo:           in.SoNo,
			CustomerID:     in.CustomerID,
			OrderDate:      in.OrderDate,
			CurrencyCode:   in.CurrencyCode,
			TotalAmount:    in.TotalAmount,
			CorrelationID:  in.CorrelationID,
			CreatedBy:      in.CreatedBy,
		})
		if err != nil {
			return err
		}
		for _, l := range in.Lines {
			if err := qt.CreateSalesOrderLine(ctx, salesdb.CreateSalesOrderLineParams{
				OrganizationID: in.OrganizationID,
				SalesOrderID:   created.ID,
				LineNo:         l.LineNo,
				ItemID:         l.ItemID,
				OrderQty:       l.OrderQty,
				UnitPrice:      l.UnitPrice,
				CreatedBy:      in.CreatedBy,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *SalesOrderRepository) ConsumeAndCacheParty(ctx context.Context, orgID, eventID, partyID uuid.UUID, eventType, code, name, partyType string) error {
	return r.db.WithOrgScope(ctx, []uuid.UUID{orgID}, func(tx pgx.Tx) error {
		qt := r.q.WithTx(tx)
		n, err := qt.MarkEventProcessed(ctx, salesdb.MarkEventProcessedParams{
			EventID:   eventID,
			EventType: eventType,
		})
		if err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
		return qt.UpsertMasterdataCache(ctx, salesdb.UpsertMasterdataCacheParams{
			OrganizationID: orgID,
			PartyID:        partyID,
			Code:           code,
			Name:           name,
			PartyType:      partyType,
		})
	})
}

func (r *SalesOrderRepository) DrainAndPublish(ctx context.Context, batch int32, pub *eventbus.Publisher) (int, error) {
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
	qt := r.q.WithTx(tx)
	rows, err := qt.DrainOutbox(ctx, batch)
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	published := 0
	var pubErr error
	for _, row := range rows {
		if e := pub.Publish(ctx, eventbus.Envelope{
			EventID:        row.ID.String(),
			EventType:      row.EventType,
			AggregateType:  row.AggregateType,
			AggregateID:    row.AggregateID.String(),
			OccurredAt:     row.CreatedAt.Time,
			CorrelationID:  row.CorrelationID.String(),
			OrganizationID: row.OrganizationID.String(),
			SchemaVersion:  "v1",
			Data:           json.RawMessage(row.Payload),
		}); e != nil {
			pubErr = e
			break
		}
		if e := qt.MarkOutboxPublished(ctx, salesdb.MarkOutboxPublishedParams{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
		}); e != nil {
			pubErr = e
			break
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
