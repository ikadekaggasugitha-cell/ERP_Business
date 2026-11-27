package httpapi_test

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/currency-service/internal/currencydb"
	"github.com/sinergiinvestasiproperti/erp/services/currency-service/internal/httpapi"
	"github.com/sinergiinvestasiproperti/erp/services/currency-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type mockStore struct {
	currencies          []currencydb.ListCurrenciesRow
	rateTypes           []currencydb.ListExchangeRateTypesRow
	rates               []currencydb.ListExchangeRatesRow
	currency            *currencydb.CurCurrency
	rateType            *currencydb.CurExchangeRateType
	rate                *currencydb.GetExchangeRateRow
	convertRes          *repository.ConvertResult
	lastListRatesParams currencydb.ListExchangeRatesParams
	err                 error
}

func (m *mockStore) CreateCurrency(_ context.Context, _ []uuid.UUID, _ currencydb.CreateCurrencyParams, _ uuid.UUID) (currencydb.CreateCurrencyRow, error) {
	if m.err != nil {
		return currencydb.CreateCurrencyRow{}, m.err
	}
	return currencydb.CreateCurrencyRow{ID: uuid.New(), RowVersion: 1}, nil
}

func (m *mockStore) GetCurrency(_ context.Context, _ []uuid.UUID, id uuid.UUID) (currencydb.CurCurrency, error) {
	if m.err != nil {
		return currencydb.CurCurrency{}, m.err
	}
	if m.currency != nil {
		return *m.currency, nil
	}
	return currencydb.CurCurrency{
		ID:        id,
		Code:      "USD",
		Name:      "US Dollar",
		Precision: 2,
		IsActive:  true,
	}, nil
}

func (m *mockStore) GetCurrencyByCode(_ context.Context, _ []uuid.UUID, code string) (currencydb.CurCurrency, error) {
	if m.err != nil {
		return currencydb.CurCurrency{}, m.err
	}
	return currencydb.CurCurrency{
		ID:        uuid.New(),
		Code:      code,
		Name:      code + " Currency",
		Precision: 2,
		IsActive:  true,
	}, nil
}

func (m *mockStore) ListCurrencies(_ context.Context, _ []uuid.UUID, _ *bool, _, _ int32) ([]currencydb.ListCurrenciesRow, int64, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	return m.currencies, int64(len(m.currencies)), nil
}

func (m *mockStore) UpdateCurrency(_ context.Context, _ []uuid.UUID, _ currencydb.UpdateCurrencyParams) (int32, error) {
	if m.err != nil {
		return 0, m.err
	}
	return 2, nil
}

func (m *mockStore) DeleteCurrency(_ context.Context, _ []uuid.UUID, _ currencydb.DeleteCurrencyParams) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return 1, nil
}

func (m *mockStore) CreateExchangeRateType(_ context.Context, _ []uuid.UUID, _ currencydb.CreateExchangeRateTypeParams) (currencydb.CreateExchangeRateTypeRow, error) {
	if m.err != nil {
		return currencydb.CreateExchangeRateTypeRow{}, m.err
	}
	return currencydb.CreateExchangeRateTypeRow{ID: uuid.New(), RowVersion: 1}, nil
}

func (m *mockStore) GetExchangeRateType(_ context.Context, _ []uuid.UUID, id uuid.UUID) (currencydb.CurExchangeRateType, error) {
	if m.err != nil {
		return currencydb.CurExchangeRateType{}, m.err
	}
	return currencydb.CurExchangeRateType{ID: id, Code: "SPOT", Name: "Spot Rate", RowVersion: 1}, nil
}

func (m *mockStore) GetExchangeRateTypeByCode(_ context.Context, _ []uuid.UUID, code string) (currencydb.CurExchangeRateType, error) {
	if m.err != nil {
		return currencydb.CurExchangeRateType{}, m.err
	}
	return currencydb.CurExchangeRateType{ID: uuid.New(), Code: code, Name: code + " Rate", RowVersion: 1}, nil
}

func (m *mockStore) ListExchangeRateTypes(_ context.Context, _ []uuid.UUID, _, _ int32) ([]currencydb.ListExchangeRateTypesRow, int64, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	return m.rateTypes, int64(len(m.rateTypes)), nil
}

func (m *mockStore) CreateExchangeRate(_ context.Context, _ []uuid.UUID, _ currencydb.CreateExchangeRateParams, _ uuid.UUID) (currencydb.CreateExchangeRateRow, error) {
	if m.err != nil {
		return currencydb.CreateExchangeRateRow{}, m.err
	}
	return currencydb.CreateExchangeRateRow{ID: uuid.New(), RowVersion: 1}, nil
}

func (m *mockStore) GetExchangeRate(_ context.Context, _ []uuid.UUID, id uuid.UUID) (currencydb.GetExchangeRateRow, error) {
	if m.err != nil {
		return currencydb.GetExchangeRateRow{}, m.err
	}
	if m.rate != nil {
		return *m.rate, nil
	}
	r, _ := platform.ParseNumeric("16000.00000000")
	return currencydb.GetExchangeRateRow{
		ID:           id,
		FromCurrency: "USD",
		ToCurrency:   "IDR",
		Rate:         r,
		ValidFrom:    pgtype.Date{Valid: true},
		RowVersion:   1,
	}, nil
}

func (m *mockStore) ListExchangeRates(_ context.Context, _ []uuid.UUID, p currencydb.ListExchangeRatesParams) ([]currencydb.ListExchangeRatesRow, int64, error) {
	m.lastListRatesParams = p
	if m.err != nil {
		return nil, 0, m.err
	}
	return m.rates, int64(len(m.rates)), nil
}

func (m *mockStore) UpdateExchangeRate(_ context.Context, _ []uuid.UUID, _ currencydb.UpdateExchangeRateParams, _ uuid.UUID) (int32, error) {
	if m.err != nil {
		return 0, m.err
	}
	return 2, nil
}

func (m *mockStore) DeleteExchangeRate(_ context.Context, _ []uuid.UUID, _ currencydb.DeleteExchangeRateParams) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return 1, nil
}

func (m *mockStore) ResolveExchangeRate(_ context.Context, _ []uuid.UUID, from, to string, rt uuid.UUID, _ pgtype.Date, _ string) (*repository.RateLookupResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &repository.RateLookupResult{
		FromCurrency:  from,
		ToCurrency:    to,
		RateTypeID:    rt,
		Rate:          big.NewRat(16000, 1),
		RateString:    "16000.00000000",
		Source:        repository.RateSourceDirect,
		EffectiveDate: "2026-11-13",
	}, nil
}

func (m *mockStore) ConvertAmount(_ context.Context, _ []uuid.UUID, from, to string, rt uuid.UUID, _ pgtype.Date, _ string, _ string) (*repository.ConvertResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.convertRes != nil {
		return m.convertRes, nil
	}
	return &repository.ConvertResult{
		FromCurrency:    from,
		ToCurrency:      to,
		RateTypeID:      rt,
		OriginalAmount:  "100.00",
		ConvertedAmount: "1600000.00",
		Rate:            "16000.00000000",
		RateSource:      repository.RateSourceDirect,
		EffectiveDate:   "2026-11-13",
	}, nil
}

func setupTestServer(store httpapi.Store) http.Handler {
	mux := http.NewServeMux()
	httpapi.NewServer(store).Register(mux)
	return httpx.Middleware(mux)
}

func TestCurrenciesEndpoints(t *testing.T) {
	t.Parallel()

	orgID := uuid.NewString()
	userID := uuid.NewString()

	t.Run("GET /currencies with pagination", func(t *testing.T) {
		t.Parallel()
		store := &mockStore{
			currencies: []currencydb.ListCurrenciesRow{
				{ID: uuid.New(), Code: "IDR", Name: "Indonesian Rupiah", Precision: 2, IsActive: true, RowVersion: 1},
				{ID: uuid.New(), Code: "USD", Name: "US Dollar", Precision: 2, IsActive: true, RowVersion: 1},
			},
		}
		srv := setupTestServer(store)
		req := httptest.NewRequest(http.MethodGet, "/currencies?page=1&size=10", nil)
		req.Header.Set("X-Organization-Id", orgID)

		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}
		var env httpx.ListEnvelope
		if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if env.Page.TotalElements != 2 {
			t.Fatalf("expected 2 elements, got %d", env.Page.TotalElements)
		}
	})

	t.Run("POST /currencies - valid", func(t *testing.T) {
		t.Parallel()
		store := &mockStore{}
		srv := setupTestServer(store)
		body := `{"code":"SGD","name":"Singapore Dollar","precision":2,"isActive":true}`
		req := httptest.NewRequest(http.MethodPost, "/currencies", strings.NewReader(body))
		req.Header.Set("X-Organization-Id", orgID)
		req.Header.Set("X-User-Id", userID)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("POST /currencies - invalid code (400)", func(t *testing.T) {
		t.Parallel()
		store := &mockStore{}
		srv := setupTestServer(store)
		body := `{"code":"TOOLONG","name":"Invalid","precision":2}`
		req := httptest.NewRequest(http.MethodPost, "/currencies", strings.NewReader(body))
		req.Header.Set("X-Organization-Id", orgID)
		req.Header.Set("X-User-Id", userID)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("POST /currencies - negative precision (422)", func(t *testing.T) {
		t.Parallel()
		store := &mockStore{}
		srv := setupTestServer(store)
		body := `{"code":"EUR","name":"Euro","precision":-1}`
		req := httptest.NewRequest(http.MethodPost, "/currencies", strings.NewReader(body))
		req.Header.Set("X-Organization-Id", orgID)
		req.Header.Set("X-User-Id", userID)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422 Unprocessable, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("POST /currencies - duplicate code (409)", func(t *testing.T) {
		t.Parallel()
		store := &mockStore{err: httpx.Conflict("DUPLICATE_RESOURCE", "already exists")}
		srv := setupTestServer(store)
		body := `{"code":"USD","name":"US Dollar","precision":2}`
		req := httptest.NewRequest(http.MethodPost, "/currencies", strings.NewReader(body))
		req.Header.Set("X-Organization-Id", orgID)
		req.Header.Set("X-User-Id", userID)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("GET /currencies/{id} - not found (404)", func(t *testing.T) {
		t.Parallel()
		store := &mockStore{err: httpx.NotFound("currency not found")}
		srv := setupTestServer(store)
		req := httptest.NewRequest(http.MethodGet, "/currencies/"+uuid.NewString(), nil)
		req.Header.Set("X-Organization-Id", orgID)

		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404 Not Found, got %d", rec.Code)
		}
	})

	t.Run("PUT /currencies/{id} - missing If-Match (428)", func(t *testing.T) {
		t.Parallel()
		store := &mockStore{}
		srv := setupTestServer(store)
		body := `{"name":"Updated US Dollar"}`
		req := httptest.NewRequest(http.MethodPut, "/currencies/"+uuid.NewString(), strings.NewReader(body))
		req.Header.Set("X-Organization-Id", orgID)
		req.Header.Set("X-User-Id", userID)

		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusPreconditionRequired {
			t.Fatalf("expected 428 Precondition Required, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("DELETE /currencies/{id} - valid with If-Match (204)", func(t *testing.T) {
		t.Parallel()
		store := &mockStore{}
		srv := setupTestServer(store)
		req := httptest.NewRequest(http.MethodDelete, "/currencies/"+uuid.NewString(), nil)
		req.Header.Set("X-Organization-Id", orgID)
		req.Header.Set("X-User-Id", userID)
		req.Header.Set("If-Match", `"1"`)

		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204 No Content, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestExchangeRatesEndpoints(t *testing.T) {
	t.Parallel()

	orgID := uuid.NewString()
	userID := uuid.NewString()
	rateTypeID := uuid.NewString()

	t.Run("POST /exchange-rates - valid (201)", func(t *testing.T) {
		t.Parallel()
		store := &mockStore{}
		srv := setupTestServer(store)
		body := `{"fromCurrency":"USD","toCurrency":"IDR","rateTypeId":"` + rateTypeID + `","rate":"16200.50000000","validFrom":"2026-11-01"}`
		req := httptest.NewRequest(http.MethodPost, "/exchange-rates", strings.NewReader(body))
		req.Header.Set("X-Organization-Id", orgID)
		req.Header.Set("X-User-Id", userID)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("POST /exchange-rates - non-positive rate (422)", func(t *testing.T) {
		t.Parallel()
		store := &mockStore{}
		srv := setupTestServer(store)
		body := `{"fromCurrency":"USD","toCurrency":"IDR","rateTypeId":"` + rateTypeID + `","rate":"0.00000000","validFrom":"2026-11-01"}`
		req := httptest.NewRequest(http.MethodPost, "/exchange-rates", strings.NewReader(body))
		req.Header.Set("X-Organization-Id", orgID)
		req.Header.Set("X-User-Id", userID)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422 Unprocessable, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("POST /exchange-rates - identical currencies (422)", func(t *testing.T) {
		t.Parallel()
		store := &mockStore{}
		srv := setupTestServer(store)
		body := `{"fromCurrency":"USD","toCurrency":"USD","rateTypeId":"` + rateTypeID + `","rate":"1.00000000","validFrom":"2026-11-01"}`
		req := httptest.NewRequest(http.MethodPost, "/exchange-rates", strings.NewReader(body))
		req.Header.Set("X-Organization-Id", orgID)
		req.Header.Set("X-User-Id", userID)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422 Unprocessable, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("POST /exchange-rates - invalid date range (422)", func(t *testing.T) {
		t.Parallel()
		store := &mockStore{}
		srv := setupTestServer(store)
		body := `{"fromCurrency":"USD","toCurrency":"IDR","rateTypeId":"` + rateTypeID + `","rate":"16000.00000000","validFrom":"2026-11-10","validTo":"2026-11-01"}`
		req := httptest.NewRequest(http.MethodPost, "/exchange-rates", strings.NewReader(body))
		req.Header.Set("X-Organization-Id", orgID)
		req.Header.Set("X-User-Id", userID)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422 Unprocessable, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("POST /exchange-rates/convert - valid direct (200)", func(t *testing.T) {
		t.Parallel()
		store := &mockStore{
			convertRes: &repository.ConvertResult{
				FromCurrency:    "USD",
				ToCurrency:      "IDR",
				RateTypeID:      uuid.MustParse(rateTypeID),
				OriginalAmount:  "100.00",
				ConvertedAmount: "1620000.00",
				Rate:            "16200.00000000",
				RateSource:      repository.RateSourceDirect,
				EffectiveDate:   "2026-11-13",
			},
		}
		srv := setupTestServer(store)
		body := `{"fromCurrency":"USD","toCurrency":"IDR","rateTypeId":"` + rateTypeID + `","amount":"100.00"}`
		req := httptest.NewRequest(http.MethodPost, "/exchange-rates/convert", strings.NewReader(body))
		req.Header.Set("X-Organization-Id", orgID)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}
		var res repository.ConvertResult
		if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
			t.Fatalf("decode convert response: %v", err)
		}
		if res.ConvertedAmount != "1620000.00" {
			t.Fatalf("expected 1620000.00, got %s", res.ConvertedAmount)
		}
		if res.RateSource != repository.RateSourceDirect {
			t.Fatalf("expected direct rate source, got %s", res.RateSource)
		}
	})

	t.Run("POST /exchange-rates/convert - rate not found (404)", func(t *testing.T) {
		t.Parallel()
		store := &mockStore{err: httpx.NotFound("exchange rate not found")}
		srv := setupTestServer(store)
		body := `{"fromCurrency":"SEK","toCurrency":"IDR","rateTypeId":"` + rateTypeID + `","amount":"100.00"}`
		req := httptest.NewRequest(http.MethodPost, "/exchange-rates/convert", strings.NewReader(body))
		req.Header.Set("X-Organization-Id", orgID)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404 Not Found, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("GET single resources return ETag", func(t *testing.T) {
		t.Parallel()
		store := &mockStore{
			currency: &currencydb.CurCurrency{ID: uuid.New(), Code: "USD", Name: "US Dollar", Precision: 2, IsActive: true, RowVersion: 3},
		}
		srv := setupTestServer(store)

		for _, tc := range []struct {
			path string
			etag string
		}{
			{"/currencies/" + store.currency.ID.String(), `"3"`},
			{"/exchange-rate-types/" + uuid.NewString(), `"1"`},
			{"/exchange-rates/" + uuid.NewString(), `"1"`},
		} {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("X-Organization-Id", orgID)
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("GET %s: expected 200, got %d: %s", tc.path, rec.Code, rec.Body.String())
			}
			if got := rec.Header().Get("ETag"); got != tc.etag {
				t.Fatalf("GET %s: expected ETag %s, got %q", tc.path, tc.etag, got)
			}
		}
	})

	t.Run("POST /currencies - precision > 8 rejected (422)", func(t *testing.T) {
		t.Parallel()
		store := &mockStore{}
		srv := setupTestServer(store)
		req := httptest.NewRequest(http.MethodPost, "/currencies", strings.NewReader(`{"code":"BTC","name":"Bitcoin","precision":9}`))
		req.Header.Set("X-Organization-Id", orgID)
		req.Header.Set("X-User-Id", userID)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("GET /exchange-rates?type=SPOT forwards rate type filter", func(t *testing.T) {
		t.Parallel()
		store := &mockStore{}
		srv := setupTestServer(store)
		req := httptest.NewRequest(http.MethodGet, "/exchange-rates?type=SPOT", nil)
		req.Header.Set("X-Organization-Id", orgID)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		if store.lastListRatesParams.RateTypeID != "SPOT" {
			t.Fatalf("expected RateTypeID=SPOT, got %q", store.lastListRatesParams.RateTypeID)
		}
	})
}
