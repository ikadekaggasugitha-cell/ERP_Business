// Package repository is the AP data-access layer: every scoped call runs inside
// a WithOrgScope transaction so RLS applies; sqlc provides the type-safe queries.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/ap-service/internal/apdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

// InvoiceRepository encapsulates RLS-scoped AP invoice persistence and outbox relay.
type InvoiceRepository struct {
	db *platform.DB
	q  *apdb.Queries
}

// NewInvoiceRepository constructs an InvoiceRepository bound to db.
func NewInvoiceRepository(db *platform.DB) *InvoiceRepository {
	return &InvoiceRepository{db: db, q: apdb.New(db.Pool)}
}

// ConsumeAndCacheParty applies masterdata.party.upserted idempotently.
func (r *InvoiceRepository) ConsumeAndCacheParty(ctx context.Context, scope []uuid.UUID, mark apdb.MarkEventProcessedParams, up apdb.UpsertMasterdataCacheParams) (applied bool, err error) {
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

// CreateInvoiceWithLines atomically inserts an AP invoice and its lines inside one RLS tx.
func (r *InvoiceRepository) CreateInvoiceWithLines(
	ctx context.Context, scope []uuid.UUID,
	invoice apdb.CreateInvoiceParams, lines []apdb.CreateInvoiceLineParams,
) (apdb.CreateInvoiceRow, error) {
	var created apdb.CreateInvoiceRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if created, e = q.CreateInvoice(ctx, invoice); e != nil {
			return e
		}
		for i := range lines {
			lines[i].ApInvoiceID = created.ID
			lines[i].InvoiceDate = created.InvoiceDate
			if e = q.CreateInvoiceLine(ctx, lines[i]); e != nil {
				return e
			}
		}
		return nil
	})
	return created, err
}

// UpdateDraftWithLines updates a Draft AP invoice header + replaces its lines inside one optimistic-locked tx.
func (r *InvoiceRepository) UpdateDraftWithLines(
	ctx context.Context, scope []uuid.UUID,
	hdr apdb.UpdateDraftInvoiceParams, lines []apdb.CreateInvoiceLineParams,
) (int32, error) {
	var newVer int32
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		v, e := q.UpdateDraftInvoice(ctx, hdr)
		if e != nil {
			if errors.Is(e, pgx.ErrNoRows) {
				return httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi, dokumen tidak ditemukan, atau status bukan Draft")
			}
			return e
		}
		newVer = v
		if e = q.DeleteInvoiceLines(ctx, apdb.DeleteInvoiceLinesParams{
			ApInvoiceID: hdr.ID,
			InvoiceDate: hdr.InvoiceDate,
		}); e != nil {
			return e
		}
		for i := range lines {
			lines[i].ApInvoiceID = hdr.ID
			lines[i].InvoiceDate = hdr.InvoiceDate
			if e = q.CreateInvoiceLine(ctx, lines[i]); e != nil {
				return e
			}
		}
		return nil
	})
	return newVer, err
}

// DeleteDraft soft-deletes a Draft invoice with optimistic locking.
func (r *InvoiceRepository) DeleteDraft(ctx context.Context, scope []uuid.UUID, arg apdb.DeleteDraftInvoiceParams) (int64, error) {
	var n int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		n, e = r.q.WithTx(tx).DeleteDraftInvoice(ctx, arg)
		return e
	})
	return n, err
}

// GetByID fetches one active invoice header by id.
func (r *InvoiceRepository) GetByID(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (apdb.ApApInvoice, error) {
	var out apdb.ApApInvoice
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).GetInvoiceByID(ctx, id)
		return e
	})
	return out, err
}

// GetWithLines fetches an invoice header and its active lines in one RLS tx.
func (r *InvoiceRepository) GetWithLines(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (apdb.ApApInvoice, []apdb.ListInvoiceLinesRow, error) {
	var inv apdb.ApApInvoice
	var lines []apdb.ListInvoiceLinesRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if inv, e = q.GetInvoiceByID(ctx, id); e != nil {
			return e
		}
		lines, e = q.ListInvoiceLines(ctx, apdb.ListInvoiceLinesParams{
			ApInvoiceID: inv.ID,
			InvoiceDate: inv.InvoiceDate,
		})
		return e
	})
	return inv, lines, err
}

// ListWithCount returns paginated invoices and exact matching total count inside one RLS tx.
func (r *InvoiceRepository) ListWithCount(ctx context.Context, scope []uuid.UUID, status string, limit, offset int32) ([]apdb.ListInvoicesRow, int64, error) {
	var rows []apdb.ListInvoicesRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountInvoices(ctx, status); e != nil {
			return e
		}
		rows, e = q.ListInvoices(ctx, apdb.ListInvoicesParams{
			Limit:  limit,
			Offset: offset,
			Status: status,
		})
		return e
	})
	return rows, total, err
}

// ListByStatus returns invoices filtered by organization and status.
func (r *InvoiceRepository) ListByStatus(ctx context.Context, scope []uuid.UUID, arg apdb.ListInvoicesByStatusParams) ([]apdb.ListInvoicesByStatusRow, error) {
	var out []apdb.ListInvoicesByStatusRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).ListInvoicesByStatus(ctx, arg)
		return e
	})
	return out, err
}

// LineTotal returns the summed line amount for an invoice.
func (r *InvoiceRepository) LineTotal(ctx context.Context, scope []uuid.UUID, arg apdb.InvoiceLineTotalParams) (pgtype.Numeric, error) {
	var out pgtype.Numeric
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).InvoiceLineTotal(ctx, arg)
		return e
	})
	return out, err
}

// ApplyDebitNoteAndEmit reduces an AP invoice's outstanding amount and enqueues ap.debit_note.issued atomically.
func (r *InvoiceRepository) ApplyDebitNoteAndEmit(
	ctx context.Context, scope []uuid.UUID, arg apdb.ApplyPaymentToInvoiceParams, ev apdb.InsertOutboxParams,
) (int64, error) {
	var n int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		n, e = q.ApplyPaymentToInvoice(ctx, arg)
		if e != nil || n == 0 {
			return e
		}
		return q.InsertOutbox(ctx, ev)
	})
	return n, err
}

// SettlementAllocation carries one AP payment allocation from treasury.payment.settled.
type SettlementAllocation struct {
	InvoiceRef      string
	AllocatedAmount pgtype.Numeric
}

// ConsumePaymentSettled idempotently applies treasury.payment.settled allocations to AP vendor invoices.
func (r *InvoiceRepository) ConsumePaymentSettled(
	ctx context.Context, scope []uuid.UUID,
	mark apdb.MarkEventProcessedParams,
	orgID, paymentID, actor uuid.UUID,
	allocs []SettlementAllocation,
) (applied bool, err error) {
	err = r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		n, e := q.MarkEventProcessed(ctx, mark)
		if e != nil || n == 0 {
			return e
		}
		for _, a := range allocs {
			inv, e := q.GetInvoiceByRef(ctx, apdb.GetInvoiceByRefParams{
				OrganizationID: orgID,
				InvoiceRef:     a.InvoiceRef,
			})
			if e != nil {
				if errors.Is(e, pgx.ErrNoRows) {
					continue
				}
				return e
			}
			if e = q.CreatePaymentAllocation(ctx, apdb.CreatePaymentAllocationParams{
				OrganizationID:  orgID,
				ApInvoiceID:     inv.ID,
				InvoiceDate:     inv.InvoiceDate,
				PaymentID:       paymentID,
				AllocatedAmount: a.AllocatedAmount,
				CreatedBy:       actor,
			}); e != nil {
				return e
			}
			if _, e = q.ApplyPaymentToInvoice(ctx, apdb.ApplyPaymentToInvoiceParams{
				ID:          inv.ID,
				InvoiceDate: inv.InvoiceDate,
				UpdatedBy:   &actor,
				PaidAmount:  a.AllocatedAmount,
			}); e != nil {
				return e
			}
		}
		applied = true
		return nil
	})
	return applied, err
}

// ConsumeAndRecordInvoice creates+records an AP invoice idempotently from procurement.goods.received.
func (r *InvoiceRepository) ConsumeAndRecordInvoice(
	ctx context.Context, scope []uuid.UUID,
	mark apdb.MarkEventProcessedParams,
	invoice apdb.CreateInvoiceParams, ev apdb.InsertOutboxParams,
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
		if _, e = q.RecordInvoice(ctx, apdb.RecordInvoiceParams{
			ID: created.ID, InvoiceDate: created.InvoiceDate, RowVersion: created.RowVersion, UpdatedBy: &invoice.CreatedBy,
		}); e != nil {
			return e
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

// RecordAndEmit validates line-sum equality (when lines exist), transitions Draft->Recorded,
// and writes the ap.invoice.recorded outbox row atomically.
func (r *InvoiceRepository) RecordAndEmit(
	ctx context.Context, scope []uuid.UUID,
	rec apdb.RecordInvoiceParams, ev apdb.InsertOutboxParams,
) (int64, error) {
	var n int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		inv, e := q.GetInvoiceByID(ctx, rec.ID)
		if e != nil {
			return e
		}
		cnt, e := q.InvoiceLineCount(ctx, apdb.InvoiceLineCountParams{ApInvoiceID: inv.ID, InvoiceDate: inv.InvoiceDate})
		if e != nil {
			return e
		}
		if cnt > 0 {
			sumNum, e := q.InvoiceLineTotal(ctx, apdb.InvoiceLineTotalParams{ApInvoiceID: inv.ID, InvoiceDate: inv.InvoiceDate})
			if e != nil {
				return e
			}
			if platform.NumericToRat(sumNum).Cmp(platform.NumericToRat(inv.TotalAmount)) != 0 {
				return httpx.Unprocessable("AMOUNT_MISMATCH", "jumlah seluruh baris faktur harus sama dengan totalAmount")
			}
		}
		if n, e = q.RecordInvoice(ctx, rec); e != nil {
			return e
		}
		if n == 1 {
			if e = q.InsertOutbox(ctx, ev); e != nil {
				return e
			}
		}
		return nil
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
		if e := q.MarkOutboxPublished(ctx, apdb.MarkOutboxPublishedParams{ID: row.ID, CreatedAt: row.CreatedAt}); e != nil {
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
