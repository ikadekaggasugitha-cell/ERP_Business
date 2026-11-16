// Package repository provides RLS-enforced CoA, Financial Organization,
// Fiscal Calendar/Period, and Outbox relay persistence.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/coa-service/internal/coadb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type AccountRepository struct {
	db *platform.DB
	q  *coadb.Queries
}

func NewAccountRepository(db *platform.DB) *AccountRepository {
	return &AccountRepository{db: db, q: coadb.New(db.Pool)}
}

func (r *AccountRepository) CreateChartOfAccounts(ctx context.Context, scope []uuid.UUID, arg coadb.CreateChartOfAccountsParams) (coadb.CreateChartOfAccountsRow, error) {
	var out coadb.CreateChartOfAccountsRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateChartOfAccounts(ctx, arg)
		return e
	})
	return out, err
}

func (r *AccountRepository) GetChartOfAccounts(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (coadb.CoaChartOfAccount, error) {
	var out coadb.CoaChartOfAccount
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).GetChartOfAccounts(ctx, id)
		return e
	})
	return out, err
}

func (r *AccountRepository) ListChartsWithCount(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]coadb.ListChartsOfAccountsRow, int64, error) {
	var rows []coadb.ListChartsOfAccountsRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountChartsOfAccounts(ctx); e != nil {
			return e
		}
		rows, e = q.ListChartsOfAccounts(ctx, coadb.ListChartsOfAccountsParams{Limit: limit, Offset: offset})
		return e
	})
	return rows, total, err
}

func (r *AccountRepository) UpdateChartOfAccounts(ctx context.Context, scope []uuid.UUID, arg coadb.UpdateChartOfAccountsParams) (int32, error) {
	var ver int32
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		ver, e = r.q.WithTx(tx).UpdateChartOfAccounts(ctx, arg)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau chart of accounts tidak ditemukan")
		}
		return e
	})
	return ver, err
}

func (r *AccountRepository) DeleteChartOfAccounts(ctx context.Context, scope []uuid.UUID, arg coadb.DeleteChartOfAccountsParams) (int64, error) {
	var n int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		n, e = r.q.WithTx(tx).DeleteChartOfAccounts(ctx, arg)
		return e
	})
	return n, err
}

func (r *AccountRepository) CreateAndEmit(ctx context.Context, scope []uuid.UUID, acct coadb.CreateAccountParams, ev coadb.InsertOutboxParams) (coadb.CreateAccountRow, error) {
	var created coadb.CreateAccountRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if created, e = q.CreateAccount(ctx, acct); e != nil {
			return e
		}
		ev.AggregateID = created.ID
		return q.InsertOutbox(ctx, ev)
	})
	return created, err
}

func (r *AccountRepository) UpdateAccountAndEmit(ctx context.Context, scope []uuid.UUID, arg coadb.UpdateAccountParams, ev coadb.InsertOutboxParams) (int32, error) {
	var newVer int32
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		v, e := q.UpdateAccount(ctx, arg)
		if e != nil {
			if errors.Is(e, pgx.ErrNoRows) {
				return httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau account tidak ditemukan")
			}
			return e
		}
		newVer = v
		return q.InsertOutbox(ctx, ev)
	})
	return newVer, err
}

func (r *AccountRepository) DeleteAccount(ctx context.Context, scope []uuid.UUID, arg coadb.DeleteAccountParams) (int64, error) {
	var n int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		n, e = r.q.WithTx(tx).DeleteAccount(ctx, arg)
		return e
	})
	return n, err
}

func (r *AccountRepository) GetByID(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (coadb.CoaAccount, error) {
	var out coadb.CoaAccount
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).GetAccount(ctx, id)
		return e
	})
	return out, err
}

func (r *AccountRepository) ListWithCount(ctx context.Context, scope []uuid.UUID, acctType string, limit, offset int32) ([]coadb.ListAccountsRow, int64, error) {
	var rows []coadb.ListAccountsRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountAccounts(ctx, acctType); e != nil {
			return e
		}
		rows, e = q.ListAccounts(ctx, coadb.ListAccountsParams{
			Limit:       limit,
			Offset:      offset,
			AccountType: acctType,
		})
		return e
	})
	return rows, total, err
}

func (r *AccountRepository) CreateFinancialOrg(ctx context.Context, scope []uuid.UUID, arg coadb.CreateFinancialOrganizationParams) (coadb.CreateFinancialOrganizationRow, error) {
	var out coadb.CreateFinancialOrganizationRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateFinancialOrganization(ctx, arg)
		return e
	})
	return out, err
}

func (r *AccountRepository) ListFinancialOrgsWithCount(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]coadb.ListFinancialOrganizationsRow, int64, error) {
	var rows []coadb.ListFinancialOrganizationsRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountFinancialOrganizations(ctx); e != nil {
			return e
		}
		rows, e = q.ListFinancialOrganizations(ctx, coadb.ListFinancialOrganizationsParams{Limit: limit, Offset: offset})
		return e
	})
	return rows, total, err
}

func (r *AccountRepository) CreateLedger(ctx context.Context, scope []uuid.UUID, arg coadb.CreateLedgerParams) (coadb.CreateLedgerRow, error) {
	var out coadb.CreateLedgerRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateLedger(ctx, arg)
		return e
	})
	return out, err
}

func (r *AccountRepository) ListLedgersWithCount(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]coadb.ListLedgersRow, int64, error) {
	var rows []coadb.ListLedgersRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountLedgers(ctx); e != nil {
			return e
		}
		rows, e = q.ListLedgers(ctx, coadb.ListLedgersParams{Limit: limit, Offset: offset})
		return e
	})
	return rows, total, err
}

func (r *AccountRepository) CreateFiscalCalendar(ctx context.Context, scope []uuid.UUID, arg coadb.CreateFiscalCalendarParams) (coadb.CreateFiscalCalendarRow, error) {
	var out coadb.CreateFiscalCalendarRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateFiscalCalendar(ctx, arg)
		return e
	})
	return out, err
}

func (r *AccountRepository) ListFiscalCalendarsWithCount(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]coadb.ListFiscalCalendarsRow, int64, error) {
	var rows []coadb.ListFiscalCalendarsRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountFiscalCalendars(ctx); e != nil {
			return e
		}
		rows, e = q.ListFiscalCalendars(ctx, coadb.ListFiscalCalendarsParams{Limit: limit, Offset: offset})
		return e
	})
	return rows, total, err
}

func (r *AccountRepository) CreateFiscalYear(ctx context.Context, scope []uuid.UUID, arg coadb.CreateFiscalYearParams) (coadb.CreateFiscalYearRow, error) {
	var out coadb.CreateFiscalYearRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateFiscalYear(ctx, arg)
		return e
	})
	return out, err
}

func (r *AccountRepository) ListFiscalYearsWithCount(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]coadb.ListFiscalYearsRow, int64, error) {
	var rows []coadb.ListFiscalYearsRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountFiscalYears(ctx); e != nil {
			return e
		}
		rows, e = q.ListFiscalYears(ctx, coadb.ListFiscalYearsParams{Limit: limit, Offset: offset})
		return e
	})
	return rows, total, err
}

func (r *AccountRepository) CreateFiscalPeriod(ctx context.Context, scope []uuid.UUID, arg coadb.CreateFiscalPeriodParams) (coadb.CreateFiscalPeriodRow, error) {
	var out coadb.CreateFiscalPeriodRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateFiscalPeriod(ctx, arg)
		return e
	})
	return out, err
}

func (r *AccountRepository) ListFiscalPeriodsWithCount(ctx context.Context, scope []uuid.UUID, fiscalYearID string, limit, offset int32) ([]coadb.ListFiscalPeriodsRow, int64, error) {
	var rows []coadb.ListFiscalPeriodsRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountFiscalPeriods(ctx, fiscalYearID); e != nil {
			return e
		}
		rows, e = q.ListFiscalPeriods(ctx, coadb.ListFiscalPeriodsParams{
			Limit:        limit,
			Offset:       offset,
			FiscalYearID: fiscalYearID,
		})
		return e
	})
	return rows, total, err
}

func (r *AccountRepository) TransitionFiscalPeriodAndEmit(
	ctx context.Context, scope []uuid.UUID,
	id uuid.UUID, rowVer int32, expectedStatus, newStatus string, user uuid.UUID, correlationID uuid.UUID,
) (coadb.FinFiscalPeriod, int32, error) {
	var period coadb.FinFiscalPeriod
	var nextVer int32
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		period, e = q.GetFiscalPeriod(ctx, id)
		if e != nil {
			return e
		}
		if period.Status != expectedStatus {
			return httpx.Conflict("INVALID_PERIOD_TRANSITION", fmt.Sprintf("fiscal period status is %s, expected %s", period.Status, expectedStatus))
		}
		nextVer, e = q.UpdateFiscalPeriodStatus(ctx, coadb.UpdateFiscalPeriodStatusParams{
			ID:             id,
			RowVersion:     rowVer,
			UpdatedBy:      &user,
			NewStatus:      newStatus,
			ExpectedStatus: expectedStatus,
		})
		if e != nil {
			if errors.Is(e, pgx.ErrNoRows) {
				return httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau status berubah bersamaan")
			}
			return e
		}
		payload, _ := json.Marshal(map[string]any{
			"periodId":     period.ID.String(),
			"fiscalYearId": period.FiscalYearID.String(),
			"periodNo":     period.PeriodNo,
			"startDate":    platform.FormatDate(period.StartDate),
			"endDate":      platform.FormatDate(period.EndDate),
			"status":       newStatus,
		})
		return q.InsertOutbox(ctx, coadb.InsertOutboxParams{
			AggregateType:  "FiscalPeriod",
			AggregateID:    period.ID,
			EventType:      "coa.period.updated",
			Payload:        payload,
			OrganizationID: period.OrganizationID,
			CorrelationID:  correlationID,
		})
	})
	period.Status = newStatus
	period.RowVersion = nextVer
	return period, nextVer, err
}

// DrainAndPublish drains coa.outbox under FOR UPDATE SKIP LOCKED inside a tx.
func (r *AccountRepository) DrainAndPublish(ctx context.Context, limit int32, pub *eventbus.Publisher) (int, error) {
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
		if e := q.MarkOutboxPublished(ctx, coadb.MarkOutboxPublishedParams{ID: row.ID, CreatedAt: row.CreatedAt}); e != nil {
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
