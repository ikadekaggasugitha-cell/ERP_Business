package httpapi

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/currency-service/internal/currencydb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

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
		if *req.Precision < 0 || *req.Precision > 8 {
			httpx.WriteError(w, r, httpx.Unprocessable("INVALID_PRECISION", "precision must be between 0 and 8"))
			return
		}
		precision = *req.Precision
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	correlationID := httpx.CorrelationUUID(r)
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

	httpx.WriteIDResponse(w, http.StatusOK, row.RowVersion, currencyResp{
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
		if *req.Precision < 0 || *req.Precision > 8 {
			httpx.WriteError(w, r, httpx.Unprocessable("INVALID_PRECISION", "precision must be between 0 and 8"))
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
