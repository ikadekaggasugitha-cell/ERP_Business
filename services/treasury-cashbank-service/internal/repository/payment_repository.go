// Package repository owns treasury_db transactions (bank accounts,
// payments, bank statement lines, reconciliation, outbox, processed_event).
package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
	"github.com/sinergiinvestasiproperti/erp/services/treasury-cashbank-service/internal/treasurydb"
)

type PaymentRepository struct {
	db *platform.DB
	q  *treasurydb.Queries
}

func NewPaymentRepository(db *platform.DB) *PaymentRepository {
	return &PaymentRepository{db: db, q: treasurydb.New(db.Pool)}
}

type ReconcileMatch struct {
	StatementLineID uuid.UUID `json:"statementLineId"`
	PaymentID       uuid.UUID `json:"paymentId"`
	PaymentNo       string    `json:"paymentNo"`
	Amount          string    `json:"amount"`
}

// ConsumeAndSchedulePayment idempotently schedules a Draft payment from an
// inbound event (ap.invoice.recorded / payroll.run.completed) inside one tx.
func (r *PaymentRepository) ConsumeAndSchedulePayment(
	ctx context.Context, scope []uuid.UUID,
	evMark treasurydb.MarkEventProcessedParams,
	glClear, sysActor uuid.UUID,
	pay treasurydb.CreatePaymentParams,
) (bool, uuid.UUID, error) {
	var created bool
	var out uuid.UUID
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		n, err := q.MarkEventProcessed(ctx, evMark)
		if err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
		bankID, err := q.EnsureSystemBankAccount(ctx, treasurydb.EnsureSystemBankAccountParams{
			OrganizationID: scope[0],
			GlAccountID:    glClear,
			CreatedBy:      sysActor,
		})
		if err != nil {
			return err
		}
		pay.BankAccountID = bankID
		row, err := q.CreatePayment(ctx, pay)
		if err != nil {
			return err
		}
		created = true
		out = row.ID
		return nil
	})
	return created, out, err
}

func (r *PaymentRepository) CreateBankAccount(ctx context.Context, scope []uuid.UUID, arg treasurydb.CreateBankAccountParams) (treasurydb.CreateBankAccountRow, error) {
	var out treasurydb.CreateBankAccountRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateBankAccount(ctx, arg)
		return e
	})
	return out, err
}

func (r *PaymentRepository) GetBankAccount(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (treasurydb.TrsBankAccount, error) {
	var out treasurydb.TrsBankAccount
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).GetBankAccount(ctx, id)
		return e
	})
	return out, err
}

func (r *PaymentRepository) ListBanksWithCount(ctx context.Context, scope []uuid.UUID, arg treasurydb.ListBankAccountsParams) ([]treasurydb.ListBankAccountsRow, int64, error) {
	var rows []treasurydb.ListBankAccountsRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if rows, e = q.ListBankAccounts(ctx, arg); e != nil {
			return e
		}
		total, e = q.CountBankAccounts(ctx, arg.CurrencyCode)
		return e
	})
	return rows, total, err
}

func (r *PaymentRepository) CreatePayment(ctx context.Context, scope []uuid.UUID, arg treasurydb.CreatePaymentParams, allocs []treasurydb.CreatePaymentAllocationParams) (treasurydb.CreatePaymentRow, error) {
	var out treasurydb.CreatePaymentRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		if _, err := q.GetBankAccount(ctx, arg.BankAccountID); err != nil {
			return err
		}
		var e error
		if out, e = q.CreatePayment(ctx, arg); e != nil {
			return e
		}
		for _, a := range allocs {
			a.PaymentID = out.ID
			a.OrganizationID = arg.OrganizationID
			a.CreatedBy = arg.CreatedBy
			if e = q.CreatePaymentAllocation(ctx, a); e != nil {
				return e
			}
		}
		return nil
	})
	return out, err
}

func (r *PaymentRepository) GetWithAllocations(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (treasurydb.TrsPayment, []treasurydb.ListPaymentAllocationsRow, error) {
	var p treasurydb.TrsPayment
	var allocs []treasurydb.ListPaymentAllocationsRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if p, e = q.GetPayment(ctx, id); e != nil {
			return e
		}
		allocs, e = q.ListPaymentAllocations(ctx, id)
		return e
	})
	return p, allocs, err
}

func (r *PaymentRepository) ListWithCount(ctx context.Context, scope []uuid.UUID, arg treasurydb.ListPaymentsParams) ([]treasurydb.ListPaymentsRow, int64, error) {
	var rows []treasurydb.ListPaymentsRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if rows, e = q.ListPayments(ctx, arg); e != nil {
			return e
		}
		total, e = q.CountPayments(ctx, treasurydb.CountPaymentsParams{
			Status:        arg.Status,
			BankAccountID: arg.BankAccountID,
		})
		return e
	})
	return rows, total, err
}

func (r *PaymentRepository) UpdateDraft(ctx context.Context, scope []uuid.UUID, arg treasurydb.UpdateDraftPaymentParams) (int32, error) {
	var ver int32
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		if _, err := q.GetBankAccount(ctx, arg.BankAccountID); err != nil {
			return err
		}
		var e error
		ver, e = q.UpdateDraftPayment(ctx, arg)
		return e
	})
	return ver, err
}

func (r *PaymentRepository) DeleteDraft(ctx context.Context, scope []uuid.UUID, arg treasurydb.DeleteDraftPaymentParams) (int64, error) {
	var n int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		n, e = r.q.WithTx(tx).DeleteDraftPayment(ctx, arg)
		return e
	})
	return n, err
}

// SettleAndEmit transitions Draft->Settled (optimistic-locked) AND writes
// treasury.payment.settled + cashbank.payment.settled outbox events in one tx.
func (r *PaymentRepository) SettleAndEmit(
	ctx context.Context, scope []uuid.UUID,
	settle treasurydb.SettlePaymentParams, events ...treasurydb.InsertOutboxParams,
) (int32, error) {
	var ver int32
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if ver, e = q.SettlePayment(ctx, settle); e != nil {
			return e
		}
		for _, ev := range events {
			if e = q.InsertOutbox(ctx, ev); e != nil {
				return e
			}
		}
		return nil
	})
	return ver, err
}

func (r *PaymentRepository) CreateStatementLine(ctx context.Context, scope []uuid.UUID, arg treasurydb.CreateBankStatementLineParams) (treasurydb.CreateBankStatementLineRow, error) {
	var out treasurydb.CreateBankStatementLineRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		if _, err := q.GetBankAccount(ctx, arg.BankAccountID); err != nil {
			return err
		}
		var e error
		out, e = q.CreateBankStatementLine(ctx, arg)
		return e
	})
	return out, err
}

func (r *PaymentRepository) ListStatementLinesWithCount(ctx context.Context, scope []uuid.UUID, arg treasurydb.ListBankStatementLinesParams) ([]treasurydb.ListBankStatementLinesRow, int64, error) {
	var rows []treasurydb.ListBankStatementLinesRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if rows, e = q.ListBankStatementLines(ctx, arg); e != nil {
			return e
		}
		total, e = q.CountBankStatementLines(ctx, treasurydb.CountBankStatementLinesParams{
			BankAccountID: arg.BankAccountID,
			IsReconciled:  arg.IsReconciled,
		})
		return e
	})
	return rows, total, err
}

// ReconcileBankAccountAndEmit auto-matches unreconciled statement lines against
// Settled payments on the same bank account and emits treasury.payment.reconciled.
func (r *PaymentRepository) ReconcileBankAccountAndEmit(
	ctx context.Context, scope []uuid.UUID,
	bankAccountID, actorID, correlationID uuid.UUID,
) ([]ReconcileMatch, error) {
	var matches []ReconcileMatch
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		acct, err := q.GetBankAccount(ctx, bankAccountID)
		if err != nil {
			return err
		}
		lines, err := q.ListUnreconciledStatementLines(ctx, bankAccountID)
		if err != nil {
			return err
		}
		payments, err := q.ListSettledPaymentsForBank(ctx, bankAccountID)
		if err != nil {
			return err
		}
		pairs := MatchStatementLines(lines, payments)
		for _, pair := range pairs {
			n, err := q.MarkStatementLineReconciled(ctx, treasurydb.MarkStatementLineReconciledParams{
				ID:               pair.line.ID,
				RowVersion:       pair.line.RowVersion,
				MatchedPaymentID: &pair.payment.ID,
				UpdatedBy:        &actorID,
			})
			if err != nil {
				return err
			}
			if n != 1 {
				return fmt.Errorf("stale statement line %s", pair.line.ID)
			}
			if _, err := q.ReconcilePayment(ctx, treasurydb.ReconcilePaymentParams{
				ID:         pair.payment.ID,
				RowVersion: pair.payment.RowVersion,
				UpdatedBy:  &actorID,
			}); err != nil {
				return err
			}
			amtStr := platform.FormatNumeric(pair.payment.Amount, 4)
			payload, _ := json.Marshal(map[string]any{
				"paymentId":       pair.payment.ID,
				"paymentNo":       pair.payment.PaymentNo,
				"bankAccountId":   acct.ID,
				"statementLineId": pair.line.ID,
				"amount":          amtStr,
				"statementDate":   platform.FormatDate(pair.line.StatementDate),
			})
			if err := q.InsertOutbox(ctx, treasurydb.InsertOutboxParams{
				AggregateType:  "payment",
				AggregateID:    pair.payment.ID,
				EventType:      "treasury.payment.reconciled",
				Payload:        payload,
				OrganizationID: acct.OrganizationID,
				CorrelationID:  correlationID,
			}); err != nil {
				return err
			}
			matches = append(matches, ReconcileMatch{
				StatementLineID: pair.line.ID,
				PaymentID:       pair.payment.ID,
				PaymentNo:       pair.payment.PaymentNo,
				Amount:          amtStr,
			})
		}
		return nil
	})
	return matches, err
}

type matchedPair struct {
	line    treasurydb.ListUnreconciledStatementLinesRow
	payment treasurydb.ListSettledPaymentsForBankRow
}

// MatchStatementLines pairs unreconciled bank statement lines with settled payments
// using reference+amount matching first, then sign-aware amount matching, then
// unsigned magnitude fallback for positive-only bank exports.
func MatchStatementLines(lines []treasurydb.ListUnreconciledStatementLinesRow, payments []treasurydb.ListSettledPaymentsForBankRow) []matchedPair {
	usedLines := make([]bool, len(lines))
	usedPay := make([]bool, len(payments))
	var out []matchedPair

	lineAbs := make([]*big.Rat, len(lines))
	lineRaw := make([]*big.Rat, len(lines))
	for i, l := range lines {
		if r := platform.NumericToRat(l.Amount); r != nil {
			lineRaw[i] = r
			lineAbs[i] = new(big.Rat).Abs(r)
		}
	}
	payAbs := make([]*big.Rat, len(payments))
	for j, p := range payments {
		if r := platform.NumericToRat(p.Amount); r != nil {
			payAbs[j] = new(big.Rat).Abs(r)
		}
	}

	assign := func(i, j int) {
		usedLines[i] = true
		usedPay[j] = true
		out = append(out, matchedPair{line: lines[i], payment: payments[j]})
	}

	// Pass 1: ExternalRef matches PaymentNo or payment.ExternalRef AND exact magnitude.
	for i, l := range lines {
		if lineAbs[i] == nil || l.ExternalRef == nil || *l.ExternalRef == "" {
			continue
		}
		for j, p := range payments {
			if usedPay[j] || payAbs[j] == nil || lineAbs[i].Cmp(payAbs[j]) != 0 {
				continue
			}
			if *l.ExternalRef == p.PaymentNo || (p.ExternalRef != nil && *l.ExternalRef == *p.ExternalRef) {
				assign(i, j)
				break
			}
		}
	}

	// Pass 2: Signed direction + exact magnitude (Outgoing <= 0, Incoming >= 0).
	for i := range lines {
		if usedLines[i] || lineAbs[i] == nil {
			continue
		}
		for j, p := range payments {
			if usedPay[j] || payAbs[j] == nil || lineAbs[i].Cmp(payAbs[j]) != 0 {
				continue
			}
			s := lineRaw[i].Sign()
			if (p.PaymentType == "Outgoing" && s < 0) || (p.PaymentType == "Incoming" && s > 0) {
				assign(i, j)
				break
			}
		}
	}

	// Pass 3: Positive-only bank exports matching payment magnitude on the same bank account.
	for i := range lines {
		if usedLines[i] || lineAbs[i] == nil {
			continue
		}
		for j := range payments {
			if usedPay[j] || payAbs[j] == nil || lineAbs[i].Cmp(payAbs[j]) != 0 {
				continue
			}
			assign(i, j)
			break
		}
	}

	return out
}

// DrainAndPublish relays unpublished outbox rows to Kafka inside one DB tx
// using FOR UPDATE SKIP LOCKED so concurrent replicas never double-publish.
func (r *PaymentRepository) DrainAndPublish(ctx context.Context, limit int32, pub *eventbus.Publisher) (int, error) {
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
			log.Printf("treasury relay publish %s: %v", row.ID, err)
			break
		}
		if err := q.MarkOutboxPublished(ctx, treasurydb.MarkOutboxPublishedParams{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
		}); err != nil {
			pubErr = err
			break
		}
		published++
	}
	if err := tx.Commit(ctx); err != nil {
		return published, err
	}
	return published, pubErr
}
