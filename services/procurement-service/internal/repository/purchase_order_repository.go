// Package repository encapsulates RLS-scoped procurement_db transactions and
// transactional outbox writes for purchase requisitions, purchase orders, and goods receipts.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/procurement-service/internal/procurementdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

// PurchaseOrderDetail bundles a PO header with its ordered lines.
type PurchaseOrderDetail struct {
	Header procurementdb.GetPurchaseOrderRow
	Lines  []procurementdb.ListPurchaseOrderLinesRow
}

// GoodsReceiptDetail bundles a GR header with its lines.
type GoodsReceiptDetail struct {
	Header procurementdb.GetGoodsReceiptRow
	Lines  []procurementdb.ListGoodsReceiptLinesRow
}

// PurchaseRequisitionDetail bundles a PR header with its lines.
type PurchaseRequisitionDetail struct {
	Header procurementdb.GetPurchaseRequisitionRow
	Lines  []procurementdb.ListPurchaseRequisitionLinesRow
}

// OutboxPayloadFunc builds the outbox row after PO lines are loaded in the receive transaction.
type OutboxPayloadFunc func(lines []procurementdb.ListPurchaseOrderLinesRow, gr procurementdb.CreateGoodsReceiptRow) (procurementdb.InsertOutboxParams, error)

// PurchaseOrderRepository handles persistence for procurement-service.
type PurchaseOrderRepository struct {
	db *platform.DB
	q  *procurementdb.Queries
}

// NewPurchaseOrderRepository constructs a PurchaseOrderRepository backed by db.
func NewPurchaseOrderRepository(db *platform.DB) *PurchaseOrderRepository {
	return &PurchaseOrderRepository{db: db, q: procurementdb.New(db.Pool)}
}

// CreateWithLines creates a Draft purchase order and all of its lines in one RLS-scoped transaction.
func (r *PurchaseOrderRepository) CreateWithLines(ctx context.Context, scope []uuid.UUID, po procurementdb.CreatePurchaseOrderParams, lines []procurementdb.CreatePurchaseOrderLineParams) (procurementdb.CreatePurchaseOrderRow, error) {
	var created procurementdb.CreatePurchaseOrderRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if created, e = q.CreatePurchaseOrder(ctx, po); e != nil {
			return e
		}
		for _, l := range lines {
			l.PurchaseOrderID = created.ID
			if e = q.CreatePurchaseOrderLine(ctx, l); e != nil {
				return e
			}
		}
		return nil
	})
	return created, err
}

// GetByID fetches a purchase order header and its lines inside the caller's org scope.
func (r *PurchaseOrderRepository) GetByID(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (PurchaseOrderDetail, error) {
	var out PurchaseOrderDetail
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		h, e := q.GetPurchaseOrder(ctx, id)
		if e != nil {
			return e
		}
		lines, e := q.ListPurchaseOrderLines(ctx, id)
		if e != nil {
			return e
		}
		out = PurchaseOrderDetail{Header: h, Lines: lines}
		return nil
	})
	return out, err
}

// ListWithCount lists purchase orders (optionally filtered by status) together with the total count.
func (r *PurchaseOrderRepository) ListWithCount(ctx context.Context, scope []uuid.UUID, arg procurementdb.ListPurchaseOrdersParams) ([]procurementdb.ListPurchaseOrdersRow, int64, error) {
	var rows []procurementdb.ListPurchaseOrdersRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if rows, e = q.ListPurchaseOrders(ctx, arg); e != nil {
			return e
		}
		total, e = q.CountPurchaseOrders(ctx, arg.Status)
		return e
	})
	return rows, total, err
}

// UpdateDraftWithLines updates a Draft purchase order and replaces its lines using optimistic locking.
func (r *PurchaseOrderRepository) UpdateDraftWithLines(ctx context.Context, scope []uuid.UUID, arg procurementdb.UpdateDraftPurchaseOrderParams, lines []procurementdb.CreatePurchaseOrderLineParams) (int32, int64, error) {
	var newVer int32
	var affected int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		v, e := q.UpdateDraftPurchaseOrder(ctx, arg)
		if errors.Is(e, pgx.ErrNoRows) {
			return nil
		}
		if e != nil {
			return e
		}
		newVer = v
		affected = 1
		if e = q.DeletePurchaseOrderLines(ctx, arg.ID); e != nil {
			return e
		}
		for _, l := range lines {
			l.PurchaseOrderID = arg.ID
			if e = q.CreatePurchaseOrderLine(ctx, l); e != nil {
				return e
			}
		}
		return nil
	})
	return newVer, affected, err
}

// DeleteDraft soft-deletes a Draft purchase order using optimistic locking.
func (r *PurchaseOrderRepository) DeleteDraft(ctx context.Context, scope []uuid.UUID, arg procurementdb.SoftDeleteDraftPurchaseOrderParams) (int64, error) {
	var affected int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		affected, e = r.q.WithTx(tx).SoftDeleteDraftPurchaseOrder(ctx, arg)
		return e
	})
	return affected, err
}

// ApproveAndEmit transitions a PO Draft->Approved and inserts procurement.po.approved into pr.outbox.
func (r *PurchaseOrderRepository) ApproveAndEmit(ctx context.Context, scope []uuid.UUID, arg procurementdb.ApprovePurchaseOrderParams, ev procurementdb.InsertOutboxParams) (int32, int64, error) {
	var newVer int32
	var affected int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		v, e := q.ApprovePurchaseOrder(ctx, arg)
		if errors.Is(e, pgx.ErrNoRows) {
			return nil
		}
		if e != nil {
			return e
		}
		newVer = v
		affected = 1
		return q.InsertOutbox(ctx, ev)
	})
	return newVer, affected, err
}

// ReceiveAndEmit transitions a PO Approved->Received, creates gr.goods_receipt + gr.gr_line records,
// updates received_qty on po.po_line, and emits procurement.goods.received in the same transaction.
func (r *PurchaseOrderRepository) ReceiveAndEmit(ctx context.Context, scope []uuid.UUID, arg procurementdb.ReceivePurchaseOrderParams, grArg procurementdb.CreateGoodsReceiptParams, buildOutbox OutboxPayloadFunc) (int32, procurementdb.CreateGoodsReceiptRow, int64, error) {
	var newVer int32
	var grRow procurementdb.CreateGoodsReceiptRow
	var affected int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		v, e := q.ReceivePurchaseOrder(ctx, arg)
		if errors.Is(e, pgx.ErrNoRows) {
			return nil
		}
		if e != nil {
			return e
		}
		newVer = v
		affected = 1

		lines, e := q.ListPurchaseOrderLines(ctx, arg.ID)
		if e != nil {
			return e
		}
		if e = q.MarkPOLinesReceived(ctx, procurementdb.MarkPOLinesReceivedParams{PurchaseOrderID: arg.ID, UpdatedBy: arg.UpdatedBy}); e != nil {
			return e
		}
		grRow, e = q.CreateGoodsReceipt(ctx, grArg)
		if e != nil {
			return e
		}
		for _, l := range lines {
			if e = q.CreateGoodsReceiptLine(ctx, procurementdb.CreateGoodsReceiptLineParams{
				OrganizationID: grArg.OrganizationID,
				GoodsReceiptID: grRow.ID,
				PoLineID:       l.ID,
				LineNo:         l.LineNo,
				ReceivedQty:    l.OrderQty,
				CreatedBy:      grArg.CreatedBy,
			}); e != nil {
				return e
			}
		}
		ev, e := buildOutbox(lines, grRow)
		if e != nil {
			return e
		}
		return q.InsertOutbox(ctx, ev)
	})
	return newVer, grRow, affected, err
}

// GetGoodsReceipt fetches a goods receipt header and its lines inside org scope.
func (r *PurchaseOrderRepository) GetGoodsReceipt(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (GoodsReceiptDetail, error) {
	var out GoodsReceiptDetail
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		h, e := q.GetGoodsReceipt(ctx, id)
		if e != nil {
			return e
		}
		lines, e := q.ListGoodsReceiptLines(ctx, id)
		if e != nil {
			return e
		}
		out = GoodsReceiptDetail{Header: h, Lines: lines}
		return nil
	})
	return out, err
}

// ListGoodsReceiptsWithCount lists goods receipts and returns the total row count.
func (r *PurchaseOrderRepository) ListGoodsReceiptsWithCount(ctx context.Context, scope []uuid.UUID, arg procurementdb.ListGoodsReceiptsParams) ([]procurementdb.ListGoodsReceiptsRow, int64, error) {
	var rows []procurementdb.ListGoodsReceiptsRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if rows, e = q.ListGoodsReceipts(ctx, arg); e != nil {
			return e
		}
		total, e = q.CountGoodsReceipts(ctx)
		return e
	})
	return rows, total, err
}

// CreateRequisitionWithLines creates a Draft purchase requisition and its lines.
func (r *PurchaseOrderRepository) CreateRequisitionWithLines(ctx context.Context, scope []uuid.UUID, pr procurementdb.CreatePurchaseRequisitionParams, lines []procurementdb.CreatePurchaseRequisitionLineParams) (procurementdb.CreatePurchaseRequisitionRow, error) {
	var created procurementdb.CreatePurchaseRequisitionRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if created, e = q.CreatePurchaseRequisition(ctx, pr); e != nil {
			return e
		}
		for _, l := range lines {
			l.PurchaseRequisitionID = created.ID
			if e = q.CreatePurchaseRequisitionLine(ctx, l); e != nil {
				return e
			}
		}
		return nil
	})
	return created, err
}

// GetRequisition fetches a purchase requisition header and its lines inside org scope.
func (r *PurchaseOrderRepository) GetRequisition(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (PurchaseRequisitionDetail, error) {
	var out PurchaseRequisitionDetail
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		h, e := q.GetPurchaseRequisition(ctx, id)
		if e != nil {
			return e
		}
		lines, e := q.ListPurchaseRequisitionLines(ctx, id)
		if e != nil {
			return e
		}
		out = PurchaseRequisitionDetail{Header: h, Lines: lines}
		return nil
	})
	return out, err
}

// ListRequisitionsWithCount lists purchase requisitions and returns the total count.
func (r *PurchaseOrderRepository) ListRequisitionsWithCount(ctx context.Context, scope []uuid.UUID, arg procurementdb.ListPurchaseRequisitionsParams) ([]procurementdb.ListPurchaseRequisitionsRow, int64, error) {
	var rows []procurementdb.ListPurchaseRequisitionsRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if rows, e = q.ListPurchaseRequisitions(ctx, arg); e != nil {
			return e
		}
		total, e = q.CountPurchaseRequisitions(ctx, arg.Status)
		return e
	})
	return rows, total, err
}

// TransitionRequisition moves a purchase requisition between workflow states using optimistic locking.
func (r *PurchaseOrderRepository) TransitionRequisition(ctx context.Context, scope []uuid.UUID, arg procurementdb.TransitionPurchaseRequisitionParams) (procurementdb.TransitionPurchaseRequisitionRow, int64, error) {
	var row procurementdb.TransitionPurchaseRequisitionRow
	var affected int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		row, e = r.q.WithTx(tx).TransitionPurchaseRequisition(ctx, arg)
		if errors.Is(e, pgx.ErrNoRows) {
			return nil
		}
		if e != nil {
			return e
		}
		affected = 1
		return nil
	})
	return row, affected, err
}

// ConsumeAndCacheParty deduplicates processed events and refreshes the local party read model.
func (r *PurchaseOrderRepository) ConsumeAndCacheParty(ctx context.Context, scope []uuid.UUID, ev procurementdb.MarkEventProcessedParams, c procurementdb.UpsertMasterdataCacheParams) (int64, error) {
	var inserted int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		n, e := q.MarkEventProcessed(ctx, ev)
		if e != nil || n == 0 {
			return e
		}
		inserted = n
		return q.UpsertMasterdataCache(ctx, c)
	})
	return inserted, err
}

// ConsumeWorkflowApproved handles workflow.approval.completed idempotently,
// auto-transitioning matching Draft POs or Draft/Submitted PRs to Approved.
func (r *PurchaseOrderRepository) ConsumeWorkflowApproved(
	ctx context.Context,
	scope []uuid.UUID,
	mark procurementdb.MarkEventProcessedParams,
	docType, docRef string,
	actor, corr uuid.UUID,
) (bool, error) {
	applied := false
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		n, e := q.MarkEventProcessed(ctx, mark)
		if e != nil || n == 0 {
			return e
		}
		if docRef == "" {
			applied = true
			return nil
		}
		switch docType {
		case "PurchaseOrder", "PO":
			var poID, orgID, vendorID uuid.UUID
			var poNo, ccy string
			var totalAmt pgtype.Numeric
			errScan := tx.QueryRow(ctx, `
				UPDATE prc.purchase_order
				SET status = 'Approved', updated_by = $2
				WHERE (id::text = $1 OR po_no = $1)
				  AND status = 'Draft'
				  AND deleted_at IS NULL
				RETURNING id, organization_id, po_no, vendor_id, currency_code, total_amount
			`, docRef, actor).Scan(&poID, &orgID, &poNo, &vendorID, &ccy, &totalAmt)
			if errScan == nil {
				payload, _ := json.Marshal(map[string]any{
					"poNo":         poNo,
					"vendorId":     vendorID.String(),
					"currencyCode": ccy,
					"totalAmount":  platform.FormatNumeric(totalAmt, 4),
					"status":       "Approved",
					"source":       "workflow.approval.completed",
				})
				if e := q.InsertOutbox(ctx, procurementdb.InsertOutboxParams{
					AggregateType:  "PurchaseOrder",
					AggregateID:    poID,
					EventType:      "procurement.po.approved",
					Payload:        payload,
					OrganizationID: orgID,
					CorrelationID:  corr,
				}); e != nil {
					return e
				}
			} else if !errors.Is(errScan, pgx.ErrNoRows) {
				return errScan
			}
		case "PurchaseRequisition", "PR":
			if _, e := tx.Exec(ctx, `
				UPDATE prc.purchase_requisition
				SET status = 'Approved', updated_by = $2
				WHERE (id::text = $1 OR pr_no = $1)
				  AND status IN ('Draft', 'Submitted')
				  AND deleted_at IS NULL
			`, docRef, actor); e != nil {
				return e
			}
		}
		applied = true
		return nil
	})
	return applied, err
}

// DrainAndPublish drains unpublished outbox events under row locks and publishes them to Kafka.
func (r *PurchaseOrderRepository) DrainAndPublish(ctx context.Context, pub *eventbus.Publisher, limit int32) (int, error) {
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
		if e := q.MarkOutboxPublished(ctx, procurementdb.MarkOutboxPublishedParams{ID: row.ID, CreatedAt: row.CreatedAt}); e != nil {
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
