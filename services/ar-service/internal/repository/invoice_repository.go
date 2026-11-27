// Package repository is the AR data-access layer: every scoped call runs inside a
// WithOrgScope transaction so RLS applies; sqlc provides the type-safe queries.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/ar-service/internal/ardb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

// InvoiceRepository provides RLS-enforced AR invoice persistence and outbox relay.
type InvoiceRepository struct {
	db *platform.DB
	q  *ardb.Queries
}

// NewInvoiceRepository constructs an InvoiceRepository bound to db.
func NewInvoiceRepository(db *platform.DB) *InvoiceRepository {
	return &InvoiceRepository{db: db, q: ardb.New(db.Pool)}
}

// ConsumeAndCacheParty applies masterdata.party.upserted idempotently.
func (r *InvoiceRepository) ConsumeAndCacheParty(ctx context.Context, scope []uuid.UUID, mark ardb.MarkEventProcessedParams, up ardb.UpsertMasterdataCacheParams) (applied bool, err error) {
	err = r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		n, e := q.MarkEventProcessed(ctx, mark)
		if e != nil {
			return e
		}
		if n == 0 {
			return nil
		}
		if e = q.UpsertMasterdataCache(ctx, up); e != nil {
			return e
		}
		applied = true
		return nil
	})
	return applied, err
}

// CreateInvoiceWithLinesAndEmit atomically inserts an AR invoice, its lines,
// and the ar.invoice.issued outbox row in one org-scoped transaction.
func (r *InvoiceRepository) CreateInvoiceWithLinesAndEmit(
	ctx context.Context, scope []uuid.UUID,
	invoice ardb.CreateInvoiceParams, lines []ardb.CreateInvoiceLineParams, ev ardb.InsertOutboxParams,
) (ardb.CreateInvoiceRow, error) {
	var created ardb.CreateInvoiceRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if created, e = q.CreateInvoice(ctx, invoice); e != nil {
			return e
		}
		for i := range lines {
			lines[i].ArInvoiceID = created.ID
			lines[i].InvoiceDate = created.InvoiceDate
			if e = q.CreateInvoiceLine(ctx, lines[i]); e != nil {
				return e
			}
		}
		ev.AggregateID = created.ID
		return q.InsertOutbox(ctx, ev)
	})
	return created, err
}

// SettlementAllocation carries one AR receipt allocation from treasury.payment.settled.
type SettlementAllocation struct {
	InvoiceRef      string
	AllocatedAmount pgtype.Numeric
}

// UpdateDraftWithLines updates an unpaid Draft/Issued AR invoice and optionally replaces its lines.
func (r *InvoiceRepository) UpdateDraftWithLines(
	ctx context.Context, scope []uuid.UUID,
	hdr ardb.UpdateDraftInvoiceParams, lines []ardb.CreateInvoiceLineParams,
) (int32, error) {
	var nextVer int32
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		nextVer, e = q.UpdateDraftInvoice(ctx, hdr)
		if e != nil {
			return e
		}
		if len(lines) > 0 {
			if e = q.DeleteInvoiceLines(ctx, ardb.DeleteInvoiceLinesParams{
				ArInvoiceID: hdr.ID,
				InvoiceDate: hdr.InvoiceDate,
			}); e != nil {
				return e
			}
			for i := range lines {
				lines[i].ArInvoiceID = hdr.ID
				lines[i].InvoiceDate = hdr.InvoiceDate
				if e = q.CreateInvoiceLine(ctx, lines[i]); e != nil {
					return e
				}
			}
		}
		return nil
	})
	return nextVer, err
}

// ApplyCreditNoteAndEmit reduces an AR invoice's outstanding amount and enqueues ar.credit_note.issued atomically.
func (r *InvoiceRepository) ApplyCreditNoteAndEmit(
	ctx context.Context, scope []uuid.UUID, arg ardb.ApplyReceiptToInvoiceParams, ev ardb.InsertOutboxParams,
) (ardb.ApplyReceiptToInvoiceRow, error) {
	var out ardb.ApplyReceiptToInvoiceRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		out, e = q.ApplyReceiptToInvoice(ctx, arg)
		if e != nil {
			return e
		}
		return q.InsertOutbox(ctx, ev)
	})
	return out, err
}

// WriteOffAndEmit writes off an unpaid AR invoice and enqueues ar.invoice.written_off atomically.
func (r *InvoiceRepository) WriteOffAndEmit(
	ctx context.Context, scope []uuid.UUID, arg ardb.WriteOffInvoiceParams, ev ardb.InsertOutboxParams,
) (int64, error) {
	var n int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		n, e = q.WriteOffInvoice(ctx, arg)
		if e != nil || n == 0 {
			return e
		}
		return q.InsertOutbox(ctx, ev)
	})
	return n, err
}

// ConsumePaymentSettled idempotently allocates a settled treasury receipt to AR invoices
// and emits ar.invoice.paid when an invoice becomes Paid.
func (r *InvoiceRepository) ConsumePaymentSettled(
	ctx context.Context, scope []uuid.UUID,
	mark ardb.MarkEventProcessedParams,
	orgID, receiptID, actor, corrID uuid.UUID,
	allocs []SettlementAllocation,
) (applied bool, err error) {
	err = r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		n, e := q.MarkEventProcessed(ctx, mark)
		if e != nil || n == 0 {
			return e
		}
		for _, a := range allocs {
			inv, e := q.GetInvoiceByRef(ctx, ardb.GetInvoiceByRefParams{
				OrganizationID: orgID,
				InvoiceRef:     a.InvoiceRef,
			})
			if e != nil {
				if errors.Is(e, pgx.ErrNoRows) {
					continue
				}
				return e
			}
			if e = q.CreateReceiptAllocation(ctx, ardb.CreateReceiptAllocationParams{
				OrganizationID:  orgID,
				ArInvoiceID:     inv.ID,
				InvoiceDate:     inv.InvoiceDate,
				ReceiptID:       receiptID,
				AllocatedAmount: a.AllocatedAmount,
				CreatedBy:       actor,
			}); e != nil {
				return e
			}
			updated, e := q.ApplyReceiptToInvoice(ctx, ardb.ApplyReceiptToInvoiceParams{
				ID:          inv.ID,
				InvoiceDate: inv.InvoiceDate,
				UpdatedBy:   &actor,
				PaidAmount:  a.AllocatedAmount,
			})
			if e != nil {
				if errors.Is(e, pgx.ErrNoRows) {
					continue
				}
				return e
			}
			if updated.Status == "Paid" {
				payPayload, _ := json.Marshal(map[string]any{
					"invoiceNo":    updated.InvoiceNo,
					"customerId":   updated.CustomerID.String(),
					"currencyCode": updated.CurrencyCode,
					"paidAmount":   platform.FormatNumeric(a.AllocatedAmount, 4),
					"totalAmount":  platform.FormatNumeric(updated.TotalAmount, 4),
					"status":       "Paid",
				})
				if e = q.InsertOutbox(ctx, ardb.InsertOutboxParams{
					AggregateType:  "ARInvoice",
					AggregateID:    updated.ID,
					EventType:      "ar.invoice.paid",
					Payload:        payPayload,
					OrganizationID: orgID,
					CorrelationID:  corrID,
				}); e != nil {
					return e
				}
			}
		}
		applied = true
		return nil
	})
	return applied, err
}

// ConsumeAndCreateInvoice creates an AR invoice idempotently from inventory.goods.issued.
func (r *InvoiceRepository) ConsumeAndCreateInvoice(
	ctx context.Context, scope []uuid.UUID,
	mark ardb.MarkEventProcessedParams,
	invoice ardb.CreateInvoiceParams, lines []ardb.CreateInvoiceLineParams, ev ardb.InsertOutboxParams,
) (applied bool, err error) {
	err = r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		n, e := q.MarkEventProcessed(ctx, mark)
		if e != nil {
			return e
		}
		if n == 0 {
			return nil
		}
		created, e := q.CreateInvoice(ctx, invoice)
		if e != nil {
			return e
		}
		for i := range lines {
			lines[i].ArInvoiceID = created.ID
			lines[i].InvoiceDate = created.InvoiceDate
			if e = q.CreateInvoiceLine(ctx, lines[i]); e != nil {
				return e
			}
		}
		ev.AggregateID = created.ID
		if e = q.InsertOutbox(ctx, ev); e != nil {
			return e
		}
		applied = true
		return nil
	})
	return applied, err
}

// GetWithLines fetches an active invoice and its active lines in one RLS tx.
func (r *InvoiceRepository) GetWithLines(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (ardb.ArArInvoice, []ardb.ListInvoiceLinesRow, error) {
	var inv ardb.ArArInvoice
	var lines []ardb.ListInvoiceLinesRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if inv, e = q.GetInvoiceByID(ctx, id); e != nil {
			return e
		}
		lines, e = q.ListInvoiceLines(ctx, ardb.ListInvoiceLinesParams{
			ArInvoiceID: inv.ID,
			InvoiceDate: inv.InvoiceDate,
		})
		return e
	})
	return inv, lines, err
}

// GetByID fetches one active invoice by id.
func (r *InvoiceRepository) GetByID(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (ardb.ArArInvoice, error) {
	var out ardb.ArArInvoice
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).GetInvoiceByID(ctx, id)
		return e
	})
	return out, err
}

// ListWithCount returns paginated AR invoices plus exact total count inside one RLS tx.
func (r *InvoiceRepository) ListWithCount(ctx context.Context, scope []uuid.UUID, status string, limit, offset int32) ([]ardb.ListInvoicesRow, int64, error) {
	var rows []ardb.ListInvoicesRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountInvoices(ctx, status); e != nil {
			return e
		}
		rows, e = q.ListInvoices(ctx, ardb.ListInvoicesParams{
			Limit:  limit,
			Offset: offset,
			Status: status,
		})
		return e
	})
	return rows, total, err
}

// MarkPaid transitions an Issued invoice to Paid with optimistic locking and emits ar.invoice.paid atomically.
func (r *InvoiceRepository) MarkPaid(ctx context.Context, scope []uuid.UUID, arg ardb.MarkInvoicePaidParams, ev ardb.InsertOutboxParams) (int64, error) {
	var n int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		n, e = q.MarkInvoicePaid(ctx, arg)
		if e != nil || n == 0 {
			return e
		}
		return q.InsertOutbox(ctx, ev)
	})
	return n, err
}

// DeleteDraft soft-deletes a Draft AR invoice with optimistic locking.
func (r *InvoiceRepository) DeleteDraft(ctx context.Context, scope []uuid.UUID, arg ardb.DeleteDraftInvoiceParams) (int64, error) {
	var n int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		n, e = r.q.WithTx(tx).DeleteDraftInvoice(ctx, arg)
		return e
	})
	return n, err
}

// DrainAndPublish drains unpublished outbox rows under FOR UPDATE SKIP LOCKED,
// publishes each to Kafka, marks published inside the same tx, and commits.
func (r *InvoiceRepository) DrainAndPublish(ctx context.Context, limit int32, pub *eventbus.Publisher) (int, error) {
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
		if e := q.MarkOutboxPublished(ctx, ardb.MarkOutboxPublishedParams{ID: row.ID, CreatedAt: row.CreatedAt}); e != nil {
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
