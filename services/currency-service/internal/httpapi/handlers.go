package httpapi

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/currency-service/internal/currencydb"
	"github.com/sinergiinvestasiproperti/erp/services/currency-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

// Store abstracts currency repository operations for handler unit testing.
type Store interface {
	CreateCurrency(ctx context.Context, scope []uuid.UUID, arg currencydb.CreateCurrencyParams, correlationID uuid.UUID) (currencydb.CreateCurrencyRow, error)
	GetCurrency(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (currencydb.CurCurrency, error)
	GetCurrencyByCode(ctx context.Context, scope []uuid.UUID, code string) (currencydb.CurCurrency, error)
	ListCurrencies(ctx context.Context, scope []uuid.UUID, isActive *bool, limit, offset int32) ([]currencydb.ListCurrenciesRow, int64, error)
	UpdateCurrency(ctx context.Context, scope []uuid.UUID, arg currencydb.UpdateCurrencyParams) (int32, error)
	DeleteCurrency(ctx context.Context, scope []uuid.UUID, arg currencydb.DeleteCurrencyParams) (int64, error)

	CreateExchangeRateType(ctx context.Context, scope []uuid.UUID, arg currencydb.CreateExchangeRateTypeParams) (currencydb.CreateExchangeRateTypeRow, error)
	GetExchangeRateType(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (currencydb.CurExchangeRateType, error)
	GetExchangeRateTypeByCode(ctx context.Context, scope []uuid.UUID, code string) (currencydb.CurExchangeRateType, error)
	ListExchangeRateTypes(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]currencydb.ListExchangeRateTypesRow, int64, error)

	CreateExchangeRate(ctx context.Context, scope []uuid.UUID, arg currencydb.CreateExchangeRateParams, correlationID uuid.UUID) (currencydb.CreateExchangeRateRow, error)
	GetExchangeRate(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (currencydb.GetExchangeRateRow, error)
	ListExchangeRates(ctx context.Context, scope []uuid.UUID, params currencydb.ListExchangeRatesParams) ([]currencydb.ListExchangeRatesRow, int64, error)
	UpdateExchangeRate(ctx context.Context, scope []uuid.UUID, arg currencydb.UpdateExchangeRateParams, correlationID uuid.UUID) (int32, error)
	DeleteExchangeRate(ctx context.Context, scope []uuid.UUID, arg currencydb.DeleteExchangeRateParams) (int64, error)

	ResolveExchangeRate(ctx context.Context, scope []uuid.UUID, fromCurr, toCurr string, rateTypeID uuid.UUID, onDate pgtype.Date, anchorOverride string) (*repository.RateLookupResult, error)
	ConvertAmount(ctx context.Context, scope []uuid.UUID, fromCurr, toCurr string, rateTypeID uuid.UUID, onDate pgtype.Date, amountStr string, anchorOverride string) (*repository.ConvertResult, error)
}

type Server struct {
	repo Store
	idem *httpx.IdemStore
}

func NewServer(repo Store) *Server {
	return &Server{repo: repo, idem: httpx.NewIdemStore()}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /currencies", s.listCurrencies)
	mux.HandleFunc("POST /currencies", s.createCurrency)
	mux.HandleFunc("GET /currencies/{id}", s.getCurrency)
	mux.HandleFunc("PUT /currencies/{id}", s.updateCurrency)
	mux.HandleFunc("DELETE /currencies/{id}", s.deleteCurrency)

	mux.HandleFunc("GET /exchange-rate-types", s.listRateTypes)
	mux.HandleFunc("POST /exchange-rate-types", s.createRateType)
	mux.HandleFunc("GET /exchange-rate-types/{id}", s.getRateType)

	mux.HandleFunc("GET /exchange-rates", s.listRates)
	mux.HandleFunc("POST /exchange-rates", s.createRate)
	mux.HandleFunc("GET /exchange-rates/{id}", s.getRate)
	mux.HandleFunc("PUT /exchange-rates/{id}", s.updateRate)
	mux.HandleFunc("DELETE /exchange-rates/{id}", s.deleteRate)
	mux.HandleFunc("POST /exchange-rates/convert", s.convertRate)
}

func requireWrite(w http.ResponseWriter, r *http.Request) ([]uuid.UUID, uuid.UUID, bool) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return nil, uuid.Nil, false
	}
	if scope[0] == uuid.Nil {
		httpx.WriteError(w, r, httpx.Forbidden("system organization cannot be used as active tenant"))
		return nil, uuid.Nil, false
	}
	user, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return nil, uuid.Nil, false
	}
	if err := httpx.RequirePermission(r, "currency:write"); err != nil {
		httpx.WriteError(w, r, err)
		return nil, uuid.Nil, false
	}
	return scope, user, true
}

func requireRead(w http.ResponseWriter, r *http.Request) ([]uuid.UUID, bool) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return nil, false
	}
	if err := httpx.RequirePermission(r, "currency:read"); err != nil {
		httpx.WriteError(w, r, err)
		return nil, false
	}
	return scope, true
}

// -----------------------------------------------------------------------------
// Currency Handlers
