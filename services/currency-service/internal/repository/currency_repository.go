// Package repository provides RLS-enforced Currency, Exchange Rate Type,
// Exchange Rate, Triangulation, and Outbox relay persistence.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/currency-service/internal/currencydb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

var systemOrgID = uuid.MustParse("00000000-0000-0000-0000-000000000000")

type CurrencyRepository struct {
	db *platform.DB
	q  *currencydb.Queries
}

func NewCurrencyRepository(db *platform.DB) *CurrencyRepository {
	var q *currencydb.Queries
	if db != nil && db.Pool != nil {
		q = currencydb.New(db.Pool)
	}
	return &CurrencyRepository{db: db, q: q}
}

// withOrgTx validates that orgScope is non-empty, appends the system organization
// UUID (00000000-0000-0000-0000-000000000000) if not already present so global
// ISO-4217 currencies and rate types are visible under RLS, and runs fn inside WithOrgScope.
func (r *CurrencyRepository) withOrgTx(ctx context.Context, orgScope []uuid.UUID, fn func(pgx.Tx) error) error {
	if len(orgScope) == 0 {
		return errors.New("org scope must not be empty")
	}
	scopeWithSystem := ScopeWithSystemOrg(orgScope)
	return r.db.WithOrgScope(ctx, scopeWithSystem, fn)
}

// ScopeWithSystemOrg returns a copy of orgScope guaranteed to include the global
// system organization UUID (00000000-0000-0000-0000-000000000000) when orgScope is non-empty.
func ScopeWithSystemOrg(orgScope []uuid.UUID) []uuid.UUID {
	if len(orgScope) == 0 {
		return nil
	}
	scopeWithSystem := make([]uuid.UUID, len(orgScope), len(orgScope)+1)
	copy(scopeWithSystem, orgScope)
	for _, id := range scopeWithSystem {
		if id == systemOrgID {
			return scopeWithSystem
		}
	}
	return append(scopeWithSystem, systemOrgID)
}

// -----------------------------------------------------------------------------
// Currency
// -----------------------------------------------------------------------------

func (r *CurrencyRepository) CreateCurrency(
	ctx context.Context,
	scope []uuid.UUID,
	arg currencydb.CreateCurrencyParams,
	correlationID uuid.UUID,
) (currencydb.CreateCurrencyRow, error) {
	var out currencydb.CreateCurrencyRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		out, e = q.CreateCurrency(ctx, arg)
		if e != nil {
			return e
		}

		payload, _ := json.Marshal(map[string]any{
			"id":             out.ID.String(),
			"organizationId": arg.OrganizationID.String(),
			"code":           arg.Code,
			"numericCode":    arg.NumericCode,
			"name":           arg.Name,
			"precision":      arg.Precision,
			"isActive":       arg.IsActive,
		})

		return q.InsertOutbox(ctx, currencydb.InsertOutboxParams{
			AggregateType:  "Currency",
			AggregateID:    out.ID,
			EventType:      "currency.created",
			Payload:        payload,
			OrganizationID: arg.OrganizationID,
			CorrelationID:  correlationID,
		})
	})
	if err != nil {
		return out, httpx.MapDBError(err, "currency with this code already exists in organization")
	}
	return out, nil
}

func (r *CurrencyRepository) GetCurrency(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (currencydb.CurCurrency, error) {
	var out currencydb.CurCurrency
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).GetCurrency(ctx, id)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("currency not found")
		}
		return e
	})
	return out, err
}

func (r *CurrencyRepository) GetCurrencyByCode(ctx context.Context, scope []uuid.UUID, code string) (currencydb.CurCurrency, error) {
	var out currencydb.CurCurrency
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).GetCurrencyByCode(ctx, code)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("currency not found")
		}
		return e
	})
	return out, err
}

func (r *CurrencyRepository) ListCurrencies(
	ctx context.Context,
	scope []uuid.UUID,
	isActive *bool,
	limit, offset int32,
) ([]currencydb.ListCurrenciesRow, int64, error) {
	var rows []currencydb.ListCurrenciesRow
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountCurrencies(ctx, isActive); e != nil {
			return e
		}
		rows, e = q.ListCurrencies(ctx, currencydb.ListCurrenciesParams{
			IsActive: isActive,
			Limit:    limit,
			Offset:   offset,
		})
		return e
	})
	return rows, total, err
}

func (r *CurrencyRepository) UpdateCurrency(ctx context.Context, scope []uuid.UUID, arg currencydb.UpdateCurrencyParams) (int32, error) {
	var ver int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		ver, e = r.q.WithTx(tx).UpdateCurrency(ctx, arg)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau currency tidak ditemukan")
		}
		return e
	})
	if err != nil {
		return 0, httpx.MapDBError(err, "")
	}
	return ver, nil
}

func (r *CurrencyRepository) DeleteCurrency(ctx context.Context, scope []uuid.UUID, arg currencydb.DeleteCurrencyParams) (int64, error) {
	var n int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		n, e = r.q.WithTx(tx).DeleteCurrency(ctx, arg)
		if e != nil {
			return e
		}
		if n == 0 {
			return httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau currency tidak ditemukan")
		}
		return nil
	})
	return n, err
}

// -----------------------------------------------------------------------------
// Exchange Rate Type
// -----------------------------------------------------------------------------

func (r *CurrencyRepository) CreateExchangeRateType(
	ctx context.Context,
	scope []uuid.UUID,
	arg currencydb.CreateExchangeRateTypeParams,
) (currencydb.CreateExchangeRateTypeRow, error) {
	var out currencydb.CreateExchangeRateTypeRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateExchangeRateType(ctx, arg)
		return e
	})
	if err != nil {
		return out, httpx.MapDBError(err, "exchange rate type with this code already exists")
	}
	return out, nil
}

func (r *CurrencyRepository) GetExchangeRateType(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (currencydb.CurExchangeRateType, error) {
	var out currencydb.CurExchangeRateType
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).GetExchangeRateType(ctx, id)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("exchange rate type not found")
		}
		return e
	})
	return out, err
}

func (r *CurrencyRepository) GetExchangeRateTypeByCode(ctx context.Context, scope []uuid.UUID, code string) (currencydb.CurExchangeRateType, error) {
	var out currencydb.CurExchangeRateType
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).GetExchangeRateTypeByCode(ctx, code)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("exchange rate type not found")
		}
		return e
	})
	return out, err
}

func (r *CurrencyRepository) ListExchangeRateTypes(
	ctx context.Context,
	scope []uuid.UUID,
	limit, offset int32,
) ([]currencydb.ListExchangeRateTypesRow, int64, error) {
	var rows []currencydb.ListExchangeRateTypesRow
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountExchangeRateTypes(ctx); e != nil {
			return e
		}
		rows, e = q.ListExchangeRateTypes(ctx, currencydb.ListExchangeRateTypesParams{
			Limit:  limit,
			Offset: offset,
		})
		return e
	})
	return rows, total, err
}

// -----------------------------------------------------------------------------
// Exchange Rate
// -----------------------------------------------------------------------------

func (r *CurrencyRepository) CreateExchangeRate(
	ctx context.Context,
	scope []uuid.UUID,
	arg currencydb.CreateExchangeRateParams,
	correlationID uuid.UUID,
) (currencydb.CreateExchangeRateRow, error) {
	var out currencydb.CreateExchangeRateRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		out, e = q.CreateExchangeRate(ctx, arg)
		if e != nil {
			return e
		}

		payload, _ := json.Marshal(map[string]any{
			"id":             out.ID.String(),
			"organizationId": arg.OrganizationID.String(),
			"fromCurrency":   arg.FromCurrency,
			"toCurrency":     arg.ToCurrency,
			"rateTypeId":     arg.RateTypeID.String(),
			"rate":           platform.FormatNumeric(arg.Rate, 8),
			"validFrom":      platform.FormatDate(arg.ValidFrom),
			"validTo":        platform.FormatDate(arg.ValidTo),
		})

		return q.InsertOutbox(ctx, currencydb.InsertOutboxParams{
			AggregateType:  "ExchangeRate",
			AggregateID:    out.ID,
			EventType:      "currency.rate.created",
			Payload:        payload,
			OrganizationID: arg.OrganizationID,
			CorrelationID:  correlationID,
		})
	})
	if err != nil {
		return out, httpx.MapDBError(err, "exchange rate for this pair, type, and date already exists")
	}
	return out, nil
}

func (r *CurrencyRepository) GetExchangeRate(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (currencydb.GetExchangeRateRow, error) {
	var out currencydb.GetExchangeRateRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).GetExchangeRate(ctx, id)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("exchange rate not found")
		}
		return e
	})
	return out, err
}

func (r *CurrencyRepository) ListExchangeRates(
	ctx context.Context,
	scope []uuid.UUID,
	params currencydb.ListExchangeRatesParams,
) ([]currencydb.ListExchangeRatesRow, int64, error) {
	var rows []currencydb.ListExchangeRatesRow
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountExchangeRates(ctx, currencydb.CountExchangeRatesParams{
			FromCurrency: params.FromCurrency,
			ToCurrency:   params.ToCurrency,
			RateTypeID:   params.RateTypeID,
			OnDate:       params.OnDate,
		}); e != nil {
			return e
		}
		rows, e = q.ListExchangeRates(ctx, params)
		return e
	})
	return rows, total, err
}

func (r *CurrencyRepository) UpdateExchangeRate(
	ctx context.Context,
	scope []uuid.UUID,
	arg currencydb.UpdateExchangeRateParams,
	correlationID uuid.UUID,
) (int32, error) {
	var ver int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		ver, e = q.UpdateExchangeRate(ctx, arg)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau exchange rate tidak ditemukan")
		}
		if e != nil {
			return e
		}

		payload, _ := json.Marshal(map[string]any{
			"id":      arg.ID.String(),
			"rate":    platform.FormatNumeric(arg.Rate, 8),
			"validTo": platform.FormatDate(arg.ValidTo),
		})

		return q.InsertOutbox(ctx, currencydb.InsertOutboxParams{
			AggregateType:  "ExchangeRate",
			AggregateID:    arg.ID,
			EventType:      "currency.rate.updated",
			Payload:        payload,
			OrganizationID: scope[0],
			CorrelationID:  correlationID,
		})
	})
	if err != nil {
		return 0, httpx.MapDBError(err, "")
	}
	return ver, nil
}

func (r *CurrencyRepository) DeleteExchangeRate(ctx context.Context, scope []uuid.UUID, arg currencydb.DeleteExchangeRateParams) (int64, error) {
	var n int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		n, e = r.q.WithTx(tx).DeleteExchangeRate(ctx, arg)
		if e != nil {
			return e
		}
		if n == 0 {
			return httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau exchange rate tidak ditemukan")
		}
		return nil
	})
	return n, err
}

// -----------------------------------------------------------------------------
// Outbox Relay Drain & Publish
// -----------------------------------------------------------------------------

// DrainAndPublish drains cur.outbox under FOR UPDATE SKIP LOCKED inside a tx.
func (r *CurrencyRepository) DrainAndPublish(ctx context.Context, limit int32, pub *eventbus.Publisher) (int, error) {
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
		if e := q.MarkOutboxPublished(ctx, currencydb.MarkOutboxPublishedParams{ID: row.ID, CreatedAt: row.CreatedAt}); e != nil {
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

