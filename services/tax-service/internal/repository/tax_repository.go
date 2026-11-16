// Package repository provides RLS-enforced tax code/transaction persistence and outbox relay.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
	"github.com/sinergiinvestasiproperti/erp/services/tax-service/internal/taxdb"
)

type TaxRepository struct {
	db *platform.DB
	q  *taxdb.Queries
}

func NewTaxRepository(db *platform.DB) *TaxRepository {
	return &TaxRepository{db: db, q: taxdb.New(db.Pool)}
}

func (r *TaxRepository) CreateTaxCode(ctx context.Context, scope []uuid.UUID, arg taxdb.CreateTaxCodeParams) (taxdb.CreateTaxCodeRow, error) {
	var out taxdb.CreateTaxCodeRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateTaxCode(ctx, arg)
		return e
	})
	return out, err
}

func (r *TaxRepository) GetTaxCode(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (taxdb.TaxTaxCode, error) {
	var out taxdb.TaxTaxCode
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).GetTaxCode(ctx, id)
		return e
	})
	return out, err
}

func (r *TaxRepository) ListTaxCodesWithCount(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]taxdb.ListTaxCodesRow, int64, error) {
	var rows []taxdb.ListTaxCodesRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountTaxCodes(ctx); e != nil {
			return e
		}
		rows, e = q.ListTaxCodes(ctx, taxdb.ListTaxCodesParams{Limit: limit, Offset: offset})
		return e
	})
	return rows, total, err
}

func (r *TaxRepository) CreateTransactionAndEmit(
	ctx context.Context, scope []uuid.UUID,
	txn taxdb.CreateTaxTransactionParams, ev taxdb.InsertOutboxParams,
) (taxdb.CreateTaxTransactionRow, error) {
	var created taxdb.CreateTaxTransactionRow
	err := r.db.WithOrgScope(ctx, scope, func(dbtx pgx.Tx) error {
		q := r.q.WithTx(dbtx)
		var e error
		if created, e = q.CreateTaxTransaction(ctx, txn); e != nil {
			return e
		}
		ev.AggregateID = created.ID
		return q.InsertOutbox(ctx, ev)
	})
	return created, err
}

func (r *TaxRepository) ReportEfakturAndEmit(
	ctx context.Context, scope []uuid.UUID,
	arg taxdb.ReportTaxTransactionParams, ev taxdb.InsertOutboxParams,
) (int32, error) {
	var newVer int32
	err := r.db.WithOrgScope(ctx, scope, func(dbtx pgx.Tx) error {
		q := r.q.WithTx(dbtx)
		v, e := q.ReportTaxTransaction(ctx, arg)
		if e != nil {
			if errors.Is(e, pgx.ErrNoRows) {
				return httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau tax transaction tidak ditemukan")
			}
			return e
		}
		newVer = v
		return q.InsertOutbox(ctx, ev)
	})
	return newVer, err
}

func (r *TaxRepository) ConsumeAndCreateTaxInvoice(
	ctx context.Context, scope []uuid.UUID,
	mark taxdb.MarkEventProcessedParams, sysActor uuid.UUID,
	txn taxdb.CreateTaxTransactionParams, ev taxdb.InsertOutboxParams,
) (applied bool, err error) {
	err = r.db.WithOrgScope(ctx, scope, func(dbtx pgx.Tx) error {
		q := r.q.WithTx(dbtx)
		n, e := q.MarkEventProcessed(ctx, mark)
		if e != nil {
			return e
		}
		if n == 0 {
			return nil
		}
		codeID, e := q.EnsureSystemTaxCode(ctx, taxdb.EnsureSystemTaxCodeParams{OrganizationID: txn.OrganizationID, CreatedBy: sysActor})
		if e != nil {
			return e
		}
		txn.TaxCodeID = codeID
		created, e := q.CreateTaxTransaction(ctx, txn)
		if e != nil {
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

func (r *TaxRepository) GetByID(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (taxdb.TaxTaxTransaction, error) {
	var out taxdb.TaxTaxTransaction
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).GetTaxTransaction(ctx, id)
		return e
	})
	return out, err
}

func (r *TaxRepository) ListTxnWithCount(ctx context.Context, scope []uuid.UUID, status string, limit, offset int32) ([]taxdb.ListTaxTransactionsRow, int64, error) {
	var rows []taxdb.ListTaxTransactionsRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountTaxTransactions(ctx, status); e != nil {
			return e
		}
		rows, e = q.ListTaxTransactions(ctx, taxdb.ListTaxTransactionsParams{
			Limit:         limit,
			Offset:        offset,
			EfakturStatus: status,
		})
		return e
	})
	return rows, total, err
}

// DrainAndPublish drains tax.outbox under FOR UPDATE SKIP LOCKED inside a tx.
func (r *TaxRepository) DrainAndPublish(ctx context.Context, limit int32, pub *eventbus.Publisher) (int, error) {
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
		if e := q.MarkOutboxPublished(ctx, taxdb.MarkOutboxPublishedParams{ID: row.ID, CreatedAt: row.CreatedAt}); e != nil {
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
