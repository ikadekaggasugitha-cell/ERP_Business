// Package repository implements gl-service persistence with strict PostgreSQL RLS scope
// enforcement, atomic outbox writes, account balance accumulation, and idempotent event ingest.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/gl-service/internal/gldb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type JournalRepository struct {
	db *platform.DB
}

func NewJournalRepository(db *platform.DB) *JournalRepository {
	return &JournalRepository{db: db}
}

func (r *JournalRepository) CreateEntryWithLines(
	ctx context.Context,
	scope []uuid.UUID,
	entry gldb.CreateJournalEntryParams,
	lines []gldb.CreateJournalLineParams,
) (gldb.CreateJournalEntryRow, error) {
	var out gldb.CreateJournalEntryRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := gldb.New(tx)
		row, err := q.CreateJournalEntry(ctx, entry)
		if err != nil {
			return err
		}
		for _, l := range lines {
			l.JournalEntryID = row.ID
			l.PostingDate = row.PostingDate
			if err := q.CreateJournalLine(ctx, l); err != nil {
				return err
			}
		}
		out = row
		return nil
	})
	return out, err
}

func (r *JournalRepository) GetByID(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (gldb.GlJournalEntry, []gldb.ListJournalLinesRow, error) {
	var out gldb.GlJournalEntry
	var lines []gldb.ListJournalLinesRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := gldb.New(tx)
		row, err := q.GetJournalEntryByID(ctx, id)
		if err != nil {
			return err
		}
		ls, err := q.ListJournalLines(ctx, gldb.ListJournalLinesParams{
			JournalEntryID: row.ID,
			PostingDate:    row.PostingDate,
		})
		if err != nil {
			return err
		}
		out = row
		lines = ls
		return nil
	})
	return out, lines, err
}

func (r *JournalRepository) ListWithCount(ctx context.Context, scope []uuid.UUID, p gldb.ListJournalEntriesParams) ([]gldb.ListJournalEntriesRow, int64, error) {
	var out []gldb.ListJournalEntriesRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := gldb.New(tx)
		var err error
		out, err = q.ListJournalEntries(ctx, p)
		if err != nil {
			return err
		}
		total, err = q.CountJournalEntries(ctx)
		return err
	})
	return out, total, err
}

func (r *JournalRepository) UpdateDraftEntry(
	ctx context.Context,
	scope []uuid.UUID,
	hdr gldb.UpdateJournalEntryDraftParams,
	lines []gldb.CreateJournalLineParams,
) (int64, error) {
	var n int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := gldb.New(tx)
		var err error
		n, err = q.UpdateJournalEntryDraft(ctx, hdr)
		if err != nil || n == 0 {
			return err
		}
		if len(lines) > 0 {
			if err := q.DeleteJournalLines(ctx, gldb.DeleteJournalLinesParams{
				JournalEntryID: hdr.ID,
				PostingDate:    hdr.PostingDate,
			}); err != nil {
				return err
			}
			for _, l := range lines {
				l.JournalEntryID = hdr.ID
				l.PostingDate = hdr.PostingDate
				if err := q.CreateJournalLine(ctx, l); err != nil {
					return err
				}
			}
		} else {
			if err := q.RecalculateJournalLinesFunctionalAmount(ctx, gldb.RecalculateJournalLinesFunctionalAmountParams{
				JournalEntryID: hdr.ID,
				PostingDate:    hdr.PostingDate,
				UpdatedBy:      hdr.UpdatedBy,
				ExchangeRate:   hdr.ExchangeRate,
			}); err != nil {
				return err
			}
		}
		return nil
	})
	return n, err
}

func (r *JournalRepository) SoftDeleteDraft(ctx context.Context, scope []uuid.UUID, p gldb.SoftDeleteJournalEntryParams) (int64, error) {
	var n int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var err error
		n, err = gldb.New(tx).SoftDeleteJournalEntry(ctx, p)
		return err
	})
	return n, err
}

func (r *JournalRepository) Totals(ctx context.Context, scope []uuid.UUID, p gldb.JournalEntryTotalsParams) (gldb.JournalEntryTotalsRow, error) {
	var out gldb.JournalEntryTotalsRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var err error
		out, err = gldb.New(tx).JournalEntryTotals(ctx, p)
		return err
	})
	return out, err
}

func checkFiscalPeriodOpen(ctx context.Context, q *gldb.Queries, orgID, periodID uuid.UUID) error {
	if periodID == uuid.Nil {
		return nil
	}
	st, err := q.GetFiscalPeriodStatus(ctx, gldb.GetFiscalPeriodStatusParams{
		OrganizationID: orgID,
		PeriodID:       periodID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	if strings.EqualFold(strings.TrimSpace(st), "Closed") || strings.EqualFold(strings.TrimSpace(st), "Locked") {
		return httpx.Unprocessable("FISCAL_PERIOD_CLOSED", "fiscal period is closed or locked")
	}
	return nil
}

// ConsumePeriodStatus idempotently records a fiscal period status transition in gl.fiscal_period_status.
func (r *JournalRepository) ConsumePeriodStatus(
	ctx context.Context,
	scope []uuid.UUID,
	mark gldb.MarkEventProcessedParams,
	up gldb.UpsertFiscalPeriodStatusParams,
) (bool, error) {
	var applied bool
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := gldb.New(tx)
		n, err := q.MarkEventProcessed(ctx, mark)
		if err != nil || n == 0 {
			return err
		}
		if err := q.UpsertFiscalPeriodStatus(ctx, up); err != nil {
			return err
		}
		applied = true
		return nil
	})
	return applied, err
}

// PostAndEmit atomically verifies fiscal period status + BR-556 balance, transitions Draft->Posted,
// accumulates gl.account_balance in functional currency, and appends the enriched gl.journal.posted outbox event.
func (r *JournalRepository) PostAndEmit(
	ctx context.Context,
	scope []uuid.UUID,
	entry gldb.GlJournalEntry,
	rowVer int32,
	actor uuid.UUID,
) (int64, error) {
	var n int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := gldb.New(tx)
		if err := checkFiscalPeriodOpen(ctx, q, entry.OrganizationID, entry.FiscalPeriodID); err != nil {
			return err
		}
		tot, err := q.JournalEntryTotals(ctx, gldb.JournalEntryTotalsParams{
			JournalEntryID: entry.ID,
			PostingDate:    entry.PostingDate,
		})
		if err != nil {
			return err
		}
		dr := platform.NumericToRat(tot.TotalDebit)
		cr := platform.NumericToRat(tot.TotalCredit)
		fDr := platform.NumericToRat(tot.TotalFunctionalDebit)
		fCr := platform.NumericToRat(tot.TotalFunctionalCredit)
		if tot.LineCount < 2 || dr.Sign() <= 0 || dr.Cmp(cr) != 0 {
			return httpx.Unprocessable("GL_JOURNAL_UNBALANCED", "total debit tidak sama dengan total kredit",
				httpx.Detail{Field: "lines", Issue: "debit=" + dr.FloatString(4) + ", credit=" + cr.FloatString(4)})
		}
		if (fDr.Sign() > 0 || fCr.Sign() > 0) && fDr.Cmp(fCr) != 0 {
			return httpx.Unprocessable("GL_JOURNAL_UNBALANCED", "total functional debit tidak sama dengan total functional kredit",
				httpx.Detail{Field: "lines", Issue: "functionalDebit=" + fDr.FloatString(4) + ", functionalCredit=" + fCr.FloatString(4)})
		}
		n, err = q.PostJournalEntry(ctx, gldb.PostJournalEntryParams{
			ID:          entry.ID,
			PostingDate: entry.PostingDate,
			RowVersion:  rowVer,
			UpdatedBy:   &actor,
		})
		if err != nil || n == 0 {
			return err
		}
		lines, err := q.ListJournalLines(ctx, gldb.ListJournalLinesParams{
			JournalEntryID: entry.ID,
			PostingDate:    entry.PostingDate,
		})
		if err != nil {
			return err
		}
		return upsertBalancesAndEmitOutbox(ctx, q, entry, lines, actor, "", "")
	})
	return n, err
}

// ReverseEntryAndEmit marks a Posted entry Reversed and creates+posts an opposite reversing journal.
func (r *JournalRepository) ReverseEntryAndEmit(
	ctx context.Context,
	scope []uuid.UUID,
	orig gldb.GlJournalEntry,
	rowVer int32,
	actor uuid.UUID,
	corrID uuid.UUID,
) (gldb.CreateJournalEntryRow, int64, error) {
	var revRow gldb.CreateJournalEntryRow
	var n int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := gldb.New(tx)
		if err := checkFiscalPeriodOpen(ctx, q, orig.OrganizationID, orig.FiscalPeriodID); err != nil {
			return err
		}
		var err error
		n, err = q.MarkJournalReversed(ctx, gldb.MarkJournalReversedParams{
			ID:          orig.ID,
			PostingDate: orig.PostingDate,
			RowVersion:  rowVer,
			UpdatedBy:   &actor,
		})
		if err != nil || n == 0 {
			return err
		}
		origLines, err := q.ListJournalLines(ctx, gldb.ListJournalLinesParams{
			JournalEntryID: orig.ID,
			PostingDate:    orig.PostingDate,
		})
		if err != nil {
			return err
		}
		revNo := "REV-" + orig.JournalNo
		if len(revNo) > 50 {
			revNo = revNo[:50]
		}
		revIDOrig := orig.ID
		srcModule := "GL"
		srcDoc := orig.JournalNo
		revRow, err = q.CreateJournalEntry(ctx, gldb.CreateJournalEntryParams{
			OrganizationID:          orig.OrganizationID,
			JournalNo:               revNo,
			FinancialOrganizationID: orig.FinancialOrganizationID,
			LedgerID:                orig.LedgerID,
			FiscalPeriodID:          orig.FiscalPeriodID,
			PostingDate:             orig.PostingDate,
			JournalSource:           "Reversal",
			SourceModule:            &srcModule,
			SourceDocumentRef:       &srcDoc,
			CurrencyCode:            orig.CurrencyCode,
			ExchangeRate:            orig.ExchangeRate,
			ReversedEntryID:         &revIDOrig,
			CorrelationID:           corrID,
			CreatedBy:               actor,
		})
		if err != nil {
			return err
		}
		revLines := make([]gldb.ListJournalLinesRow, 0, len(origLines))
		for _, l := range origLines {
			err := q.CreateJournalLine(ctx, gldb.CreateJournalLineParams{
				OrganizationID:   orig.OrganizationID,
				JournalEntryID:   revRow.ID,
				PostingDate:      revRow.PostingDate,
				LineNo:           l.LineNo,
				AccountID:        l.AccountID,
				DebitAmount:      l.CreditAmount,
				CreditAmount:     l.DebitAmount,
				FunctionalAmount: l.FunctionalAmount,
				Description:      l.Description,
				CreatedBy:        actor,
			})
			if err != nil {
				return err
			}
			revLines = append(revLines, gldb.ListJournalLinesRow{
				LineNo:           l.LineNo,
				AccountID:        l.AccountID,
				DebitAmount:      l.CreditAmount,
				CreditAmount:     l.DebitAmount,
				FunctionalAmount: l.FunctionalAmount,
				Description:      l.Description,
			})
		}
		if _, err := q.PostJournalEntry(ctx, gldb.PostJournalEntryParams{
			ID:          revRow.ID,
			PostingDate: revRow.PostingDate,
			RowVersion:  revRow.RowVersion,
			UpdatedBy:   &actor,
		}); err != nil {
			return err
		}
		revRow.Status = "Posted"
		revRow.RowVersion++
		revEntry := gldb.GlJournalEntry{
			ID:                      revRow.ID,
			OrganizationID:          orig.OrganizationID,
			JournalNo:               revNo,
			FinancialOrganizationID: orig.FinancialOrganizationID,
			LedgerID:                orig.LedgerID,
			FiscalPeriodID:          orig.FiscalPeriodID,
			PostingDate:             revRow.PostingDate,
			CurrencyCode:            orig.CurrencyCode,
			ReversedEntryID:         &revIDOrig,
			CorrelationID:           corrID,
		}
		return upsertBalancesAndEmitOutbox(ctx, q, revEntry, revLines, actor, "gl.journal.reverse", orig.JournalNo)
	})
	return revRow, n, err
}

func (r *JournalRepository) ListAccountBalances(ctx context.Context, scope []uuid.UUID, ledgerID, periodID uuid.UUID, limit, offset int32) ([]gldb.ListAccountBalancesRow, int64, error) {
	var rows []gldb.ListAccountBalancesRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := gldb.New(tx)
		var err error
		rows, err = q.ListAccountBalances(ctx, gldb.ListAccountBalancesParams{
			LedgerID:       ledgerID,
			FiscalPeriodID: periodID,
			Limit:          limit,
			Offset:         offset,
		})
		if err != nil {
			return err
		}
		total, err = q.CountAccountBalances(ctx, gldb.CountAccountBalancesParams{
			LedgerID:       ledgerID,
			FiscalPeriodID: periodID,
		})
		return err
	})
	return rows, total, err
}

// ConsumeAndPost executes auto-posting from subledger events idempotently within a single tx.
func (r *JournalRepository) ConsumeAndPost(
	ctx context.Context,
	scope []uuid.UUID,
	mark gldb.MarkEventProcessedParams,
	entry gldb.CreateJournalEntryParams,
	lines []gldb.CreateJournalLineParams,
	srcEventType, srcDocRef string,
) (bool, error) {
	var applied bool
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := gldb.New(tx)
		if err := checkFiscalPeriodOpen(ctx, q, entry.OrganizationID, entry.FiscalPeriodID); err != nil {
			return err
		}
		n, err := q.MarkEventProcessed(ctx, mark)
		if err != nil || n == 0 {
			return err
		}
		row, err := q.CreateJournalEntry(ctx, entry)
		if err != nil {
			return err
		}
		listLines := make([]gldb.ListJournalLinesRow, 0, len(lines))
		for _, l := range lines {
			l.JournalEntryID = row.ID
			l.PostingDate = row.PostingDate
			if err := q.CreateJournalLine(ctx, l); err != nil {
				return err
			}
			listLines = append(listLines, gldb.ListJournalLinesRow{
				LineNo:           l.LineNo,
				AccountID:        l.AccountID,
				DebitAmount:      l.DebitAmount,
				CreditAmount:     l.CreditAmount,
				FunctionalAmount: l.FunctionalAmount,
				Description:      l.Description,
			})
		}
		if _, err := q.PostJournalEntry(ctx, gldb.PostJournalEntryParams{
			ID:          row.ID,
			PostingDate: row.PostingDate,
			RowVersion:  row.RowVersion,
			UpdatedBy:   &entry.CreatedBy,
		}); err != nil {
			return err
		}
		fullEntry := gldb.GlJournalEntry{
			ID:                      row.ID,
			OrganizationID:          entry.OrganizationID,
			JournalNo:               entry.JournalNo,
			FinancialOrganizationID: entry.FinancialOrganizationID,
			LedgerID:                entry.LedgerID,
			FiscalPeriodID:          entry.FiscalPeriodID,
			PostingDate:             row.PostingDate,
			CurrencyCode:            entry.CurrencyCode,
			ExchangeRate:            entry.ExchangeRate,
			CorrelationID:           entry.CorrelationID,
		}
		if err := upsertBalancesAndEmitOutbox(ctx, q, fullEntry, listLines, entry.CreatedBy, srcEventType, srcDocRef); err != nil {
			return err
		}
		applied = true
		return nil
	})
	return applied, err
}

var wellKnownAccountCodes = map[string]string{
	"10000000-0000-0000-0000-0000000000f1": "1120", // AR Control (Asset)
	"10000000-0000-0000-0000-0000000000f2": "4100", // Revenue
	"10000000-0000-0000-0000-0000000000f3": "5100", // Operating Expense
	"10000000-0000-0000-0000-0000000000f4": "2110", // AP Control (Liability)
	"10000000-0000-0000-0000-0000000000f5": "5200", // COGS (Expense)
	"10000000-0000-0000-0000-0000000000f6": "1140", // Inventory (Asset)
	"10000000-0000-0000-0000-0000000000f7": "5300", // Payroll Expense
	"10000000-0000-0000-0000-0000000000f8": "2130", // Payroll Payable (Liability)
	"10000000-0000-0000-0000-0000000000f9": "1210", // Fixed Assets (Asset)
	"10000000-0000-0000-0000-0000000000fa": "1219", // Accumulated Depreciation (Contra-Asset)
	"10000000-0000-0000-0000-0000000000fb": "5400", // Depreciation Expense
	"10000000-0000-0000-0000-0000000000fc": "1110", // Cash / Asset Clearing
}

func functionalDebitCredit(l gldb.ListJournalLinesRow, exRate pgtype.Numeric) (pgtype.Numeric, pgtype.Numeric) {
	zeroNum, _ := platform.ParseNumeric("0")
	drRat := platform.NumericToRat(l.DebitAmount)
	crRat := platform.NumericToRat(l.CreditAmount)
	fn := l.FunctionalAmount
	fnRat := platform.NumericToRat(fn)
	if !fn.Valid || fnRat == nil || fnRat.Sign() <= 0 {
		rateRat := platform.NumericToRat(exRate)
		if rateRat == nil || rateRat.Sign() <= 0 {
			rateRat = big.NewRat(1, 1)
		}
		base := drRat
		if crRat != nil && crRat.Sign() > 0 {
			base = crRat
		}
		if base != nil && base.Sign() > 0 {
			if computed, err := platform.RatToNumeric(new(big.Rat).Mul(base, rateRat), 4); err == nil {
				fn = computed
			}
		} else {
			fn = zeroNum
		}
	}
	if drRat != nil && drRat.Sign() > 0 {
		return fn, zeroNum
	}
	if crRat != nil && crRat.Sign() > 0 {
		return zeroNum, fn
	}
	return zeroNum, zeroNum
}

func upsertBalancesAndEmitOutbox(
	ctx context.Context,
	q *gldb.Queries,
	entry gldb.GlJournalEntry,
	lines []gldb.ListJournalLinesRow,
	actor uuid.UUID,
	srcEventType, srcDocRef string,
) error {
	totDr := new(big.Rat)
	totCr := new(big.Rat)
	linePayloads := make([]map[string]any, 0, len(lines))
	for _, l := range lines {
		totDr.Add(totDr, platform.NumericToRat(l.DebitAmount))
		totCr.Add(totCr, platform.NumericToRat(l.CreditAmount))
		fnDr, fnCr := functionalDebitCredit(l, entry.ExchangeRate)
		if err := q.UpsertAccountBalance(ctx, gldb.UpsertAccountBalanceParams{
			OrganizationID: entry.OrganizationID,
			AccountID:      l.AccountID,
			FiscalPeriodID: entry.FiscalPeriodID,
			LedgerID:       entry.LedgerID,
			PeriodDebit:    fnDr,
			PeriodCredit:   fnCr,
			CreatedBy:      actor,
		}); err != nil {
			return err
		}
		lp := map[string]any{
			"lineNo":           l.LineNo,
			"accountId":        l.AccountID.String(),
			"debitAmount":      platform.FormatNumeric(l.DebitAmount, 4),
			"creditAmount":     platform.FormatNumeric(l.CreditAmount, 4),
			"functionalAmount": platform.FormatNumeric(l.FunctionalAmount, 4),
		}
		if code, ok := wellKnownAccountCodes[l.AccountID.String()]; ok {
			lp["accountCode"] = code
		}
		linePayloads = append(linePayloads, lp)
	}
	payloadMap := map[string]any{
		"journalNo":               entry.JournalNo,
		"postingDate":             entry.PostingDate.Time.Format("2006-01-02"),
		"currencyCode":            entry.CurrencyCode,
		"status":                  "Posted",
		"financialOrganizationId": entry.FinancialOrganizationID.String(),
		"ledgerId":                entry.LedgerID.String(),
		"fiscalPeriodId":          entry.FiscalPeriodID.String(),
		"totalDebit":              totDr.FloatString(4),
		"totalCredit":             totCr.FloatString(4),
		"lines":                   linePayloads,
	}
	if entry.ReversedEntryID != nil {
		payloadMap["reversedEntryId"] = entry.ReversedEntryID.String()
	}
	if srcEventType != "" {
		payloadMap["sourceEventType"] = srcEventType
	}
	if srcDocRef != "" {
		payloadMap["sourceDocumentRef"] = srcDocRef
	}
	payload, err := json.Marshal(payloadMap)
	if err != nil {
		return err
	}
	if err := q.InsertOutbox(ctx, gldb.InsertOutboxParams{
		AggregateType:  "JournalEntry",
		AggregateID:    entry.ID,
		EventType:      "gl.journal.posted",
		Payload:        payload,
		OrganizationID: entry.OrganizationID,
		CorrelationID:  entry.CorrelationID,
	}); err != nil {
		return err
	}
	// Reversal entries additionally emit gl.journal.reversed for audit/SSE;
	// the gl.journal.posted row above keeps feeding the reporting balance projector.
	if entry.ReversedEntryID == nil {
		return nil
	}
	return q.InsertOutbox(ctx, gldb.InsertOutboxParams{
		AggregateType:  "JournalEntry",
		AggregateID:    entry.ID,
		EventType:      "gl.journal.reversed",
		Payload:        payload,
		OrganizationID: entry.OrganizationID,
		CorrelationID:  entry.CorrelationID,
	})
}

// DrainAndPublish runs inside a DB transaction holding FOR UPDATE SKIP LOCKED locks
// until Kafka Publish succeeds and published_at is marked.
func (r *JournalRepository) DrainAndPublish(ctx context.Context, limit int32, pub *eventbus.Publisher) (int, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin outbox: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	q := gldb.New(tx)
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
		if err := pub.Publish(ctx, eventbus.Envelope{
			EventID:        row.ID.String(),
			EventType:      row.EventType,
			AggregateType:  row.AggregateType,
			AggregateID:    row.AggregateID.String(),
			OccurredAt:     row.CreatedAt.Time,
			CorrelationID:  row.CorrelationID.String(),
			OrganizationID: row.OrganizationID.String(),
			SchemaVersion:  "v1",
			Data:           row.Payload,
		}); err != nil {
			pubErr = err
			break
		}
		if err := q.MarkOutboxPublished(ctx, gldb.MarkOutboxPublishedParams{
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
			return 0, fmt.Errorf("commit outbox: %w", err)
		}
		committed = true
	}
	return published, pubErr
}
