// Package httpapi implements Currency, Exchange Rate Type, Exchange Rate,
// Triangulation, and Currency Conversion endpoints with RLS, optimistic locking, and idempotency.
package httpapi

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/currency-service/internal/currencydb"
	"github.com/sinergiinvestasiproperti/erp/services/currency-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
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
// -----------------------------------------------------------------------------

type currencyReq struct {
	Code        string  `json:"code"`
	NumericCode *string `json:"numericCode,omitempty"`
	Name        string  `json:"name"`
	Precision   *int32  `json:"precision,omitempty"`
	IsActive    *bool   `json:"isActive,omitempty"`
}

type currencyResp struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	NumericCode *string   `json:"numericCode,omitempty"`
	Name        string    `json:"name"`
	Precision   int32     `json:"precision"`
	IsActive    bool      `json:"isActive"`
	RowVersion  int32     `json:"rowVersion"`
}

func (s *Server) listCurrencies(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)

	var isActive *bool
	if raw := r.URL.Query().Get("isActive"); raw != "" {
		b := strings.EqualFold(raw, "true") || raw == "1"
		isActive = &b
	}

	rows, total, err := s.repo.ListCurrencies(r.Context(), scope, isActive, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	data := make([]currencyResp, 0, len(rows))
	for _, row := range rows {
		data = append(data, currencyResp{
			ID:          row.ID,
			Code:        row.Code,
			NumericCode: row.NumericCode,
			Name:        row.Name,
			Precision:   row.Precision,
			IsActive:    row.IsActive,
			RowVersion:  row.RowVersion,
		})
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: data,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) createCurrency(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req currencyReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	req.Code = strings.ToUpper(strings.TrimSpace(req.Code))
	if len(req.Code) != 3 {
		httpx.WriteError(w, r, httpx.BadRequest("currency code must be exactly 3 uppercase characters"))
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("currency name is required"))
		return
	}
	precision := int32(2)
	if req.Precision != nil {
		if *req.Precision < 0 {
			httpx.WriteError(w, r, httpx.Unprocessable("INVALID_PRECISION", "precision must be non-negative"))
			return
		}
		precision = *req.Precision
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	correlationID, _ := uuid.Parse(httpx.Correlation(r.Context()))
	created, err := s.repo.CreateCurrency(r.Context(), scope, currencydb.CreateCurrencyParams{
		OrganizationID: scope[0],
		Code:           req.Code,
		NumericCode:    req.NumericCode,
		Name:           strings.TrimSpace(req.Name),
		Precision:      precision,
		IsActive:       isActive,
		CreatedBy:      user,
	}, correlationID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	dto := currencyResp{
		ID:          created.ID,
		Code:        req.Code,
		NumericCode: req.NumericCode,
		Name:        strings.TrimSpace(req.Name),
		Precision:   precision,
		IsActive:    isActive,
		RowVersion:  created.RowVersion,
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, dto)
}

func (s *Server) getCurrency(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("invalid currency id"))
		return
	}

	row, err := s.repo.GetCurrency(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, currencyResp{
		ID:          row.ID,
		Code:        row.Code,
		NumericCode: row.NumericCode,
		Name:        row.Name,
		Precision:   row.Precision,
		IsActive:    row.IsActive,
		RowVersion:  row.RowVersion,
	})
}

func (s *Server) updateCurrency(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("invalid currency id"))
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req currencyReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("currency name is required"))
		return
	}
	precision := int32(2)
	if req.Precision != nil {
		if *req.Precision < 0 {
			httpx.WriteError(w, r, httpx.Unprocessable("INVALID_PRECISION", "precision must be non-negative"))
			return
		}
		precision = *req.Precision
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	nextVer, err := s.repo.UpdateCurrency(r.Context(), scope, currencydb.UpdateCurrencyParams{
		ID:          id,
		RowVersion:  rowVer,
		Name:        strings.TrimSpace(req.Name),
		NumericCode: req.NumericCode,
		Precision:   precision,
		IsActive:    isActive,
		UpdatedBy:   &user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteIDResponse(w, http.StatusOK, nextVer, map[string]any{"rowVersion": nextVer})
}

func (s *Server) deleteCurrency(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("invalid currency id"))
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	_, err = s.repo.DeleteCurrency(r.Context(), scope, currencydb.DeleteCurrencyParams{
		ID:         id,
		RowVersion: rowVer,
		UpdatedBy:  &user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// -----------------------------------------------------------------------------
// Exchange Rate Type Handlers
// -----------------------------------------------------------------------------

type rateTypeReq struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type rateTypeResp struct {
	ID         uuid.UUID `json:"id"`
	Code       string    `json:"code"`
	Name       string    `json:"name"`
	RowVersion int32     `json:"rowVersion"`
}

func (s *Server) listRateTypes(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)

	rows, total, err := s.repo.ListExchangeRateTypes(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	data := make([]rateTypeResp, 0, len(rows))
	for _, row := range rows {
		data = append(data, rateTypeResp{
			ID:         row.ID,
			Code:       row.Code,
			Name:       row.Name,
			RowVersion: row.RowVersion,
		})
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: data,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) createRateType(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req rateTypeReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	req.Code = strings.ToUpper(strings.TrimSpace(req.Code))
	if req.Code == "" {
		httpx.WriteError(w, r, httpx.BadRequest("exchange rate type code is required"))
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("exchange rate type name is required"))
		return
	}

	created, err := s.repo.CreateExchangeRateType(r.Context(), scope, currencydb.CreateExchangeRateTypeParams{
		OrganizationID: scope[0],
		Code:           req.Code,
		Name:           strings.TrimSpace(req.Name),
		CreatedBy:      user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	dto := rateTypeResp{
		ID:         created.ID,
		Code:       req.Code,
		Name:       strings.TrimSpace(req.Name),
		RowVersion: created.RowVersion,
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, dto)
}

func (s *Server) getRateType(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("invalid exchange rate type id"))
		return
	}

	row, err := s.repo.GetExchangeRateType(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, rateTypeResp{
		ID:         row.ID,
		Code:       row.Code,
		Name:       row.Name,
		RowVersion: row.RowVersion,
	})
}

// -----------------------------------------------------------------------------
// Exchange Rate Handlers
// -----------------------------------------------------------------------------

type rateReq struct {
	FromCurrency string `json:"fromCurrency"`
	ToCurrency   string `json:"toCurrency"`
	RateTypeID   string `json:"rateTypeId"`
	Rate         string `json:"rate"`
	ValidFrom    string `json:"validFrom"`
	ValidTo      string `json:"validTo,omitempty"`
}

type rateResp struct {
	ID           uuid.UUID `json:"id"`
	FromCurrency string    `json:"fromCurrency"`
	ToCurrency   string    `json:"toCurrency"`
	RateTypeID   uuid.UUID `json:"rateTypeId"`
	RateTypeCode string    `json:"rateTypeCode,omitempty"`
	RateTypeName string    `json:"rateTypeName,omitempty"`
	Rate         string    `json:"rate"`
	ValidFrom    string    `json:"validFrom"`
	ValidTo      string    `json:"validTo,omitempty"`
	RowVersion   int32     `json:"rowVersion"`
}

func (s *Server) listRates(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)

	fromCurr := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("from")))
	toCurr := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("to")))
	rateTypeParam := strings.TrimSpace(r.URL.Query().Get("type"))

	var onDate pgtype.Date
	if rawDate := strings.TrimSpace(r.URL.Query().Get("onDate")); rawDate != "" {
		d, err := platform.ParseDate(rawDate)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("onDate must be YYYY-MM-DD"))
			return
		}
		onDate = d
	}

	rows, total, err := s.repo.ListExchangeRates(r.Context(), scope, currencydb.ListExchangeRatesParams{
		FromCurrency: fromCurr,
		ToCurrency:   toCurr,
		RateTypeID:   rateTypeParam,
		OnDate:       onDate,
		Limit:        limit,
		Offset:       offset,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	data := make([]rateResp, 0, len(rows))
	for _, row := range rows {
		data = append(data, rateResp{
			ID:           row.ID,
			FromCurrency: row.FromCurrency,
			ToCurrency:   row.ToCurrency,
			RateTypeID:   row.RateTypeID,
			RateTypeCode: row.RateTypeCode,
			RateTypeName: row.RateTypeName,
			Rate:         platform.FormatNumeric(row.Rate, 8),
			ValidFrom:    platform.FormatDate(row.ValidFrom),
			ValidTo:      platform.FormatDate(row.ValidTo),
			RowVersion:   row.RowVersion,
		})
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: data,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) createRate(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req rateReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	req.FromCurrency = strings.ToUpper(strings.TrimSpace(req.FromCurrency))
	req.ToCurrency = strings.ToUpper(strings.TrimSpace(req.ToCurrency))
	if len(req.FromCurrency) != 3 || len(req.ToCurrency) != 3 {
		httpx.WriteError(w, r, httpx.BadRequest("fromCurrency and toCurrency must be 3-letter currency codes"))
		return
	}
	if req.FromCurrency == req.ToCurrency {
		httpx.WriteError(w, r, httpx.Unprocessable("IDENTICAL_CURRENCIES", "fromCurrency and toCurrency cannot be identical"))
		return
	}

	rateTypeID, err := uuid.Parse(strings.TrimSpace(req.RateTypeID))
	if err != nil {
		// Try resolving by code
		t, e := s.repo.GetExchangeRateTypeByCode(r.Context(), scope, strings.ToUpper(strings.TrimSpace(req.RateTypeID)))
		if e != nil {
			httpx.WriteError(w, r, httpx.BadRequest("rateTypeId must be a valid UUID or existing type code"))
			return
		}
		rateTypeID = t.ID
	}

	numRate, err := platform.ParseNumeric(req.Rate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("invalid numeric rate"))
		return
	}
	ratRate := platform.NumericToRat(numRate)
	if ratRate == nil || ratRate.Sign() <= 0 {
		httpx.WriteError(w, r, httpx.Unprocessable("INVALID_RATE", "exchange rate must be greater than zero"))
		return
	}

	validFrom, err := platform.ParseDate(req.ValidFrom)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("validFrom must be YYYY-MM-DD"))
		return
	}

	var validTo pgtype.Date
	if strings.TrimSpace(req.ValidTo) != "" {
		vt, err := platform.ParseDate(req.ValidTo)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("validTo must be YYYY-MM-DD"))
			return
		}
		if vt.Time.Before(validFrom.Time) {
			httpx.WriteError(w, r, httpx.Unprocessable("INVALID_DATE_RANGE", "validTo cannot be earlier than validFrom"))
			return
		}
		validTo = vt
	}

	correlationID, _ := uuid.Parse(httpx.Correlation(r.Context()))
	created, err := s.repo.CreateExchangeRate(r.Context(), scope, currencydb.CreateExchangeRateParams{
		OrganizationID: scope[0],
		FromCurrency:   req.FromCurrency,
		ToCurrency:     req.ToCurrency,
		RateTypeID:     rateTypeID,
		Rate:           numRate,
		ValidFrom:      validFrom,
		ValidTo:        validTo,
		CreatedBy:      user,
	}, correlationID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	dto := rateResp{
		ID:           created.ID,
		FromCurrency: req.FromCurrency,
		ToCurrency:   req.ToCurrency,
		RateTypeID:   rateTypeID,
		Rate:         platform.FormatNumeric(numRate, 8),
		ValidFrom:    platform.FormatDate(validFrom),
		ValidTo:      platform.FormatDate(validTo),
		RowVersion:   created.RowVersion,
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, dto)
}

func (s *Server) getRate(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("invalid exchange rate id"))
		return
	}

	row, err := s.repo.GetExchangeRate(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, rateResp{
		ID:           row.ID,
		FromCurrency: row.FromCurrency,
		ToCurrency:   row.ToCurrency,
		RateTypeID:   row.RateTypeID,
		RateTypeCode: row.RateTypeCode,
		RateTypeName: row.RateTypeName,
		Rate:         platform.FormatNumeric(row.Rate, 8),
		ValidFrom:    platform.FormatDate(row.ValidFrom),
		ValidTo:      platform.FormatDate(row.ValidTo),
		RowVersion:   row.RowVersion,
	})
}

func (s *Server) updateRate(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("invalid exchange rate id"))
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		Rate    string `json:"rate"`
		ValidTo string `json:"validTo,omitempty"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	numRate, err := platform.ParseNumeric(req.Rate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("invalid numeric rate"))
		return
	}
	ratRate := platform.NumericToRat(numRate)
	if ratRate == nil || ratRate.Sign() <= 0 {
		httpx.WriteError(w, r, httpx.Unprocessable("INVALID_RATE", "exchange rate must be greater than zero"))
		return
	}

	var validTo pgtype.Date
	if strings.TrimSpace(req.ValidTo) != "" {
		vt, err := platform.ParseDate(req.ValidTo)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("validTo must be YYYY-MM-DD"))
			return
		}
		validTo = vt
	}

	correlationID, _ := uuid.Parse(httpx.Correlation(r.Context()))
	nextVer, err := s.repo.UpdateExchangeRate(r.Context(), scope, currencydb.UpdateExchangeRateParams{
		ID:         id,
		RowVersion: rowVer,
		Rate:       numRate,
		ValidTo:    validTo,
		UpdatedBy:  &user,
	}, correlationID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteIDResponse(w, http.StatusOK, nextVer, map[string]any{"rowVersion": nextVer})
}

func (s *Server) deleteRate(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("invalid exchange rate id"))
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	_, err = s.repo.DeleteExchangeRate(r.Context(), scope, currencydb.DeleteExchangeRateParams{
		ID:         id,
		RowVersion: rowVer,
		UpdatedBy:  &user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// -----------------------------------------------------------------------------
// Convert Handler (Triangulation Engine)
// -----------------------------------------------------------------------------

type convertReq struct {
	FromCurrency string `json:"fromCurrency"`
	ToCurrency   string `json:"toCurrency"`
	RateTypeID   string `json:"rateTypeId,omitempty"`
	RateTypeCode string `json:"rateTypeCode,omitempty"`
	Amount       string `json:"amount"`
	OnDate       string `json:"onDate,omitempty"`
	Anchor       string `json:"anchor,omitempty"`
}

func (s *Server) convertRate(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	var req convertReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	req.FromCurrency = strings.ToUpper(strings.TrimSpace(req.FromCurrency))
	req.ToCurrency = strings.ToUpper(strings.TrimSpace(req.ToCurrency))
	if req.FromCurrency == "" || req.ToCurrency == "" {
		httpx.WriteError(w, r, httpx.BadRequest("fromCurrency and toCurrency are required"))
		return
	}
	if strings.TrimSpace(req.Amount) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("amount is required"))
		return
	}

	// Rate Type resolution
	var rateTypeID uuid.UUID
	if strings.TrimSpace(req.RateTypeID) != "" {
		id, err := uuid.Parse(strings.TrimSpace(req.RateTypeID))
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid rateTypeId format"))
			return
		}
		rateTypeID = id
	} else {
		rateCode := "SPOT"
		if strings.TrimSpace(req.RateTypeCode) != "" {
			rateCode = strings.ToUpper(strings.TrimSpace(req.RateTypeCode))
		}
		rt, err := s.repo.GetExchangeRateTypeByCode(r.Context(), scope, rateCode)
		if err != nil {
			httpx.WriteError(w, r, httpx.NotFound("exchange rate type not found: "+rateCode))
			return
		}
		rateTypeID = rt.ID
	}

	// Date resolution
	dateStr := strings.TrimSpace(req.OnDate)
	if dateStr == "" {
		dateStr = time.Now().UTC().Format("2006-01-02")
	}
	onDate, err := platform.ParseDate(dateStr)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("onDate must be YYYY-MM-DD"))
		return
	}

	res, err := s.repo.ConvertAmount(
		r.Context(),
		scope,
		req.FromCurrency,
		req.ToCurrency,
		rateTypeID,
		onDate,
		req.Amount,
		strings.TrimSpace(req.Anchor),
	)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, res)
}

