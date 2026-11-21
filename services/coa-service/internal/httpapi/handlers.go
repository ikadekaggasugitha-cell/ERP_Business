// Package httpapi implements CoA, Financial Organization, Ledger, and Fiscal
// Calendar/Year/Period endpoints with optimistic locking, RBAC, and real pagination.
package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/coa-service/internal/coadb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

// Store abstracts COA repository operations for handler unit tests.
type Store interface {
	CreateChartOfAccounts(ctx context.Context, scope []uuid.UUID, arg coadb.CreateChartOfAccountsParams) (coadb.CreateChartOfAccountsRow, error)
	GetChartOfAccounts(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (coadb.CoaChartOfAccount, error)
	ListChartsWithCount(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]coadb.ListChartsOfAccountsRow, int64, error)
	UpdateChartOfAccounts(ctx context.Context, scope []uuid.UUID, arg coadb.UpdateChartOfAccountsParams) (int32, error)
	DeleteChartOfAccounts(ctx context.Context, scope []uuid.UUID, arg coadb.DeleteChartOfAccountsParams) (int64, error)

	CreateAndEmit(ctx context.Context, scope []uuid.UUID, acct coadb.CreateAccountParams, ev coadb.InsertOutboxParams) (coadb.CreateAccountRow, error)
	GetByID(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (coadb.CoaAccount, error)
	ListWithCount(ctx context.Context, scope []uuid.UUID, acctType string, limit, offset int32) ([]coadb.ListAccountsRow, int64, error)
	UpdateAccountAndEmit(ctx context.Context, scope []uuid.UUID, arg coadb.UpdateAccountParams, ev coadb.InsertOutboxParams) (int32, error)
	DeleteAccount(ctx context.Context, scope []uuid.UUID, arg coadb.DeleteAccountParams) (int64, error)

	CreateFinancialOrg(ctx context.Context, scope []uuid.UUID, arg coadb.CreateFinancialOrganizationParams) (coadb.CreateFinancialOrganizationRow, error)
	ListFinancialOrgsWithCount(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]coadb.ListFinancialOrganizationsRow, int64, error)

	CreateLedger(ctx context.Context, scope []uuid.UUID, arg coadb.CreateLedgerParams) (coadb.CreateLedgerRow, error)
	ListLedgersWithCount(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]coadb.ListLedgersRow, int64, error)

	CreateFiscalCalendar(ctx context.Context, scope []uuid.UUID, arg coadb.CreateFiscalCalendarParams) (coadb.CreateFiscalCalendarRow, error)
	ListFiscalCalendarsWithCount(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]coadb.ListFiscalCalendarsRow, int64, error)

	CreateFiscalYear(ctx context.Context, scope []uuid.UUID, arg coadb.CreateFiscalYearParams) (coadb.CreateFiscalYearRow, error)
	ListFiscalYearsWithCount(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]coadb.ListFiscalYearsRow, int64, error)

	CreateFiscalPeriod(ctx context.Context, scope []uuid.UUID, arg coadb.CreateFiscalPeriodParams) (coadb.CreateFiscalPeriodRow, error)
	ListFiscalPeriodsWithCount(ctx context.Context, scope []uuid.UUID, fiscalYearID string, limit, offset int32) ([]coadb.ListFiscalPeriodsRow, int64, error)
	TransitionFiscalPeriodAndEmit(ctx context.Context, scope []uuid.UUID, id uuid.UUID, rowVer int32, expectedStatus, newStatus string, user uuid.UUID, correlationID uuid.UUID) (coadb.FinFiscalPeriod, int32, error)
}

type Server struct {
	repo Store
	idem *httpx.IdemStore
}

func NewServer(repo Store) *Server {
	return &Server{repo: repo, idem: httpx.NewIdemStore()}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /charts-of-accounts", s.listCharts)
	mux.HandleFunc("POST /charts-of-accounts", s.createChart)
	mux.HandleFunc("GET /charts-of-accounts/{id}", s.getChart)
	mux.HandleFunc("PUT /charts-of-accounts/{id}", s.updateChart)
	mux.HandleFunc("DELETE /charts-of-accounts/{id}", s.deleteChart)

	mux.HandleFunc("GET /accounts", s.listAccounts)
	mux.HandleFunc("POST /accounts", s.createAccount)
	mux.HandleFunc("GET /accounts/{id}", s.getAccount)
	mux.HandleFunc("PUT /accounts/{id}", s.updateAccount)
	mux.HandleFunc("DELETE /accounts/{id}", s.deleteAccount)

	mux.HandleFunc("GET /financial-organizations", s.listFinOrgs)
	mux.HandleFunc("POST /financial-organizations", s.createFinOrg)
	mux.HandleFunc("GET /ledgers", s.listLedgers)
	mux.HandleFunc("POST /ledgers", s.createLedger)

	mux.HandleFunc("GET /fiscal-calendars", s.listCalendars)
	mux.HandleFunc("POST /fiscal-calendars", s.createCalendar)
	mux.HandleFunc("GET /fiscal-years", s.listYears)
	mux.HandleFunc("POST /fiscal-years", s.createYear)
	mux.HandleFunc("GET /fiscal-periods", s.listPeriods)
	mux.HandleFunc("POST /fiscal-periods", s.createPeriod)
	mux.HandleFunc("POST /fiscal-periods/{id}/close", s.transitionPeriod("Open", "Closed"))
	mux.HandleFunc("POST /fiscal-periods/{id}/reopen", s.transitionPeriod("Closed", "Open"))
	mux.HandleFunc("POST /fiscal-periods/{id}/lock", s.transitionPeriod("Closed", "Locked"))
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
	if err := httpx.RequirePermission(r, "coa:write"); err != nil {
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
	if err := httpx.RequirePermission(r, "coa:read"); err != nil {
		httpx.WriteError(w, r, err)
		return nil, false
	}
	return scope, true
}

type chartReq struct {
	Code      string          `json:"code"`
	Name      string          `json:"name"`
	Structure json.RawMessage `json:"structure,omitempty"`
}

func (s *Server) createChart(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req chartReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("code dan name wajib diisi"))
		return
	}
	st := req.Structure
	if len(st) == 0 {
		st = []byte(`{}`)
	} else if !json.Valid(st) {
		httpx.WriteError(w, r, httpx.BadRequest("structure harus JSON valid"))
		return
	}
	created, err := s.repo.CreateChartOfAccounts(r.Context(), scope, coadb.CreateChartOfAccountsParams{
		OrganizationID: scope[0],
		Code:           strings.TrimSpace(req.Code),
		Name:           strings.TrimSpace(req.Name),
		Structure:      st,
		CreatedBy:      user,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "kode chart of accounts sudah dipakai"))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, map[string]any{
		"id":         created.ID.String(),
		"code":       strings.TrimSpace(req.Code),
		"name":       strings.TrimSpace(req.Name),
		"rowVersion": created.RowVersion,
	})
}

func (s *Server) listCharts(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListChartsWithCount(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{
			"id":         row.ID.String(),
			"code":       row.Code,
			"name":       row.Name,
			"structure":  json.RawMessage(row.Structure),
			"rowVersion": row.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: out, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) getChart(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	c, err := s.repo.GetChartOfAccounts(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, c.RowVersion, map[string]any{
		"id":         c.ID.String(),
		"code":       c.Code,
		"name":       c.Name,
		"structure":  json.RawMessage(c.Structure),
		"rowVersion": c.RowVersion,
	})
}

func (s *Server) updateChart(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req chartReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("name wajib diisi"))
		return
	}
	st := req.Structure
	if len(st) == 0 {
		existing, err := s.repo.GetChartOfAccounts(r.Context(), scope, id)
		if err != nil {
			httpx.WriteError(w, r, httpx.MapDBError(err, ""))
			return
		}
		st = existing.Structure
		if len(st) == 0 {
			st = []byte(`{}`)
		}
	} else if !json.Valid(st) {
		httpx.WriteError(w, r, httpx.BadRequest("structure harus JSON valid"))
		return
	}
	newVer, err := s.repo.UpdateChartOfAccounts(r.Context(), scope, coadb.UpdateChartOfAccountsParams{
		ID:         id,
		RowVersion: rowVer,
		Name:       strings.TrimSpace(req.Name),
		Structure:  st,
		UpdatedBy:  &user,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, newVer, map[string]any{"id": id.String(), "name": strings.TrimSpace(req.Name), "rowVersion": newVer})
}

func (s *Server) deleteChart(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	n, err := s.repo.DeleteChartOfAccounts(r.Context(), scope, coadb.DeleteChartOfAccountsParams{ID: id, RowVersion: rowVer, UpdatedBy: &user})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau data tidak ditemukan"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type accountReq struct {
	ChartOfAccountsID string `json:"chartOfAccountsId"`
	Code              string `json:"code"`
	Name              string `json:"name"`
	AccountType       string `json:"accountType"`
	AccountClass      string `json:"accountClass"`
	IsControl         bool   `json:"isControl"`
	IsMonetary        bool   `json:"isMonetary"`
	AllowPosting      *bool  `json:"allowPosting"`
	ParentID          string `json:"parentId"`
}

type accountDTO struct {
	ID                string  `json:"id"`
	ChartOfAccountsID string  `json:"chartOfAccountsId,omitempty"`
	Code              string  `json:"code,omitempty"`
	Name              string  `json:"name,omitempty"`
	AccountType       string  `json:"accountType,omitempty"`
	AccountClass      *string `json:"accountClass,omitempty"`
	IsControl         bool    `json:"isControl"`
	IsMonetary        bool    `json:"isMonetary"`
	AllowPosting      bool    `json:"allowPosting"`
	RowVersion        int32   `json:"rowVersion"`
}

func (s *Server) createAccount(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req accountReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	coaID, err := httpx.ParseUUIDParam("chartOfAccountsId", req.ChartOfAccountsID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("code dan name wajib diisi"))
		return
	}
	if !isValidAccountType(req.AccountType) {
		httpx.WriteError(w, r, httpx.BadRequest("accountType harus Asset, Liability, Equity, Revenue, atau Expense"))
		return
	}
	var accClass *string
	if s := strings.TrimSpace(req.AccountClass); s != "" {
		accClass = &s
	}
	var parentID *uuid.UUID
	if strings.TrimSpace(req.ParentID) != "" {
		p, err := httpx.ParseUUIDParam("parentId", req.ParentID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		parentID = &p
	}
	allowPosting := true
	if req.AllowPosting != nil {
		allowPosting = *req.AllowPosting
	}

	acct := coadb.CreateAccountParams{
		OrganizationID:    scope[0],
		ChartOfAccountsID: coaID,
		Code:              strings.TrimSpace(req.Code),
		Name:              strings.TrimSpace(req.Name),
		AccountType:       req.AccountType,
		AccountClass:      accClass,
		IsControl:         req.IsControl,
		IsMonetary:        req.IsMonetary,
		AllowPosting:      allowPosting,
		ParentID:          parentID,
		CreatedBy:         user,
	}
	payload, _ := json.Marshal(map[string]any{
		"code": acct.Code, "name": acct.Name, "accountType": acct.AccountType,
		"isMonetary": acct.IsMonetary, "allowPosting": acct.AllowPosting,
	})
	ev := coadb.InsertOutboxParams{
		AggregateType:  "Account",
		EventType:      "coa.account.upserted",
		Payload:        payload,
		OrganizationID: scope[0],
		CorrelationID:  httpx.CorrelationUUID(r),
	}
	created, err := s.repo.CreateAndEmit(r.Context(), scope, acct, ev)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "kode akun sudah dipakai"))
		return
	}
	dto := accountDTO{
		ID:                created.ID.String(),
		ChartOfAccountsID: coaID.String(),
		Code:              acct.Code,
		Name:              acct.Name,
		AccountType:       acct.AccountType,
		AccountClass:      accClass,
		IsControl:         acct.IsControl,
		IsMonetary:        acct.IsMonetary,
		AllowPosting:      acct.AllowPosting,
		RowVersion:        created.RowVersion,
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, dto)
}

func (s *Server) getAccount(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	a, err := s.repo.GetByID(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, a.RowVersion, accountDTO{
		ID:                a.ID.String(),
		ChartOfAccountsID: a.ChartOfAccountsID.String(),
		Code:              a.Code,
		Name:              a.Name,
		AccountType:       a.AccountType,
		AccountClass:      a.AccountClass,
		IsControl:         a.IsControl,
		IsMonetary:        a.IsMonetary,
		AllowPosting:      a.AllowPosting,
		RowVersion:        a.RowVersion,
	})
}

func (s *Server) listAccounts(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	acctType := strings.TrimSpace(r.URL.Query().Get("accountType"))
	rows, total, err := s.repo.ListWithCount(r.Context(), scope, acctType, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]accountDTO, 0, len(rows))
	for _, a := range rows {
		out = append(out, accountDTO{
			ID:                a.ID.String(),
			ChartOfAccountsID: a.ChartOfAccountsID.String(),
			Code:              a.Code,
			Name:              a.Name,
			AccountType:       a.AccountType,
			AccountClass:      a.AccountClass,
			IsControl:         a.IsControl,
			IsMonetary:        a.IsMonetary,
			AllowPosting:      a.AllowPosting,
			RowVersion:        a.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) updateAccount(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req accountReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("name wajib diisi"))
		return
	}
	if !isValidAccountType(req.AccountType) {
		httpx.WriteError(w, r, httpx.BadRequest("accountType tidak valid"))
		return
	}
	var accClass *string
	if s := strings.TrimSpace(req.AccountClass); s != "" {
		accClass = &s
	}
	var parentID *uuid.UUID
	if strings.TrimSpace(req.ParentID) != "" {
		p, err := httpx.ParseUUIDParam("parentId", req.ParentID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if p == id {
			httpx.WriteError(w, r, httpx.BadRequest("parentId tidak boleh mereferensikan diri sendiri"))
			return
		}
		parentID = &p
	}
	existing, err := s.repo.GetByID(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	allowPosting := true
	if req.AllowPosting != nil {
		allowPosting = *req.AllowPosting
	}
	payload, _ := json.Marshal(map[string]any{
		"code": existing.Code, "name": strings.TrimSpace(req.Name), "accountType": req.AccountType,
		"isMonetary": req.IsMonetary, "allowPosting": allowPosting,
	})
	newVer, err := s.repo.UpdateAccountAndEmit(r.Context(), scope, coadb.UpdateAccountParams{
		ID:           id,
		RowVersion:   rowVer,
		Name:         strings.TrimSpace(req.Name),
		AccountType:  req.AccountType,
		AccountClass: accClass,
		IsControl:    req.IsControl,
		IsMonetary:   req.IsMonetary,
		AllowPosting: allowPosting,
		ParentID:     parentID,
		UpdatedBy:    &user,
	}, coadb.InsertOutboxParams{
		AggregateType:  "Account",
		AggregateID:    id,
		EventType:      "coa.account.upserted",
		Payload:        payload,
		OrganizationID: scope[0],
		CorrelationID:  httpx.CorrelationUUID(r),
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, newVer, accountDTO{
		ID:                id.String(),
		ChartOfAccountsID: existing.ChartOfAccountsID.String(),
		Code:              existing.Code,
		Name:              strings.TrimSpace(req.Name),
		AccountType:       req.AccountType,
		AccountClass:      accClass,
		IsControl:         req.IsControl,
		IsMonetary:        req.IsMonetary,
		AllowPosting:      allowPosting,
		RowVersion:        newVer,
	})
}

func (s *Server) deleteAccount(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	n, err := s.repo.DeleteAccount(r.Context(), scope, coadb.DeleteAccountParams{ID: id, RowVersion: rowVer, UpdatedBy: &user})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau account tidak ditemukan"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type finOrgReq struct {
	Code                   string `json:"code"`
	Name                   string `json:"name"`
	OrganizationRefID      string `json:"organizationRefId"`
	FunctionalCurrencyCode string `json:"functionalCurrencyCode"`
	ChartOfAccountsID      string `json:"chartOfAccountsId"`
	FiscalCalendarID       string `json:"fiscalCalendarId"`
}

func (s *Server) createFinOrg(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req finOrgReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("code dan name wajib diisi"))
		return
	}
	orgRef, err := httpx.ParseUUIDParam("organizationRefId", req.OrganizationRefID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	coaID, err := httpx.ParseUUIDParam("chartOfAccountsId", req.ChartOfAccountsID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	calID, err := httpx.ParseUUIDParam("fiscalCalendarId", req.FiscalCalendarID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ccy := strings.ToUpper(deflt(req.FunctionalCurrencyCode, "IDR"))
	if len(ccy) != 3 {
		httpx.WriteError(w, r, httpx.BadRequest("functionalCurrencyCode harus 3 huruf ISO"))
		return
	}
	created, err := s.repo.CreateFinancialOrg(r.Context(), scope, coadb.CreateFinancialOrganizationParams{
		OrganizationID:         scope[0],
		Code:                   strings.TrimSpace(req.Code),
		Name:                   strings.TrimSpace(req.Name),
		OrganizationRefID:      orgRef,
		FunctionalCurrencyCode: ccy,
		ChartOfAccountsID:      coaID,
		FiscalCalendarID:       calID,
		CreatedBy:              user,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "kode financial organization sudah dipakai"))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, map[string]any{"id": created.ID.String(), "rowVersion": created.RowVersion})
}

func (s *Server) listFinOrgs(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListFinancialOrgsWithCount(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{
			"id":                     row.ID.String(),
			"code":                   row.Code,
			"name":                   row.Name,
			"organizationRefId":      row.OrganizationRefID.String(),
			"functionalCurrencyCode": row.FunctionalCurrencyCode,
			"chartOfAccountsId":      row.ChartOfAccountsID.String(),
			"fiscalCalendarId":       row.FiscalCalendarID.String(),
			"rowVersion":             row.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: out, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

type ledgerReq struct {
	FinancialOrganizationID string `json:"financialOrganizationId"`
	Code                    string `json:"code"`
	LedgerType              string `json:"ledgerType"`
	CurrencyCode            string `json:"currencyCode"`
}

func (s *Server) createLedger(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req ledgerReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if strings.TrimSpace(req.Code) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("code wajib diisi"))
		return
	}
	finOrgID, err := httpx.ParseUUIDParam("financialOrganizationId", req.FinancialOrganizationID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	lt := deflt(req.LedgerType, "Primary")
	if lt != "Primary" && lt != "Statutory" && lt != "Management" && lt != "Consolidation" {
		httpx.WriteError(w, r, httpx.BadRequest("ledgerType harus Primary, Statutory, Management, atau Consolidation"))
		return
	}
	ccy := strings.ToUpper(deflt(req.CurrencyCode, "IDR"))
	if len(ccy) != 3 {
		httpx.WriteError(w, r, httpx.BadRequest("currencyCode harus 3 huruf ISO"))
		return
	}
	created, err := s.repo.CreateLedger(r.Context(), scope, coadb.CreateLedgerParams{
		OrganizationID:          scope[0],
		FinancialOrganizationID: finOrgID,
		Code:                    strings.TrimSpace(req.Code),
		LedgerType:              lt,
		CurrencyCode:            ccy,
		CreatedBy:               user,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "kode ledger sudah dipakai"))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, map[string]any{"id": created.ID.String(), "rowVersion": created.RowVersion})
}

func (s *Server) listLedgers(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListLedgersWithCount(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{
			"id":                      row.ID.String(),
			"financialOrganizationId": row.FinancialOrganizationID.String(),
			"code":                    row.Code,
			"ledgerType":              row.LedgerType,
			"currencyCode":            row.CurrencyCode,
			"rowVersion":              row.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: out, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

type calendarReq struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	PeriodType string `json:"periodType"`
}

func (s *Server) createCalendar(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req calendarReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("code dan name wajib diisi"))
		return
	}
	pt := deflt(req.PeriodType, "Monthly")
	if pt != "Monthly" && pt != "Quarterly" {
		httpx.WriteError(w, r, httpx.BadRequest("periodType harus Monthly atau Quarterly"))
		return
	}
	created, err := s.repo.CreateFiscalCalendar(r.Context(), scope, coadb.CreateFiscalCalendarParams{
		OrganizationID: scope[0],
		Code:           strings.TrimSpace(req.Code),
		Name:           strings.TrimSpace(req.Name),
		PeriodType:     pt,
		CreatedBy:      user,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "kode fiscal calendar sudah dipakai"))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, map[string]any{"id": created.ID.String(), "rowVersion": created.RowVersion})
}

func (s *Server) listCalendars(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListFiscalCalendarsWithCount(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{"id": row.ID.String(), "code": row.Code, "name": row.Name, "periodType": row.PeriodType, "rowVersion": row.RowVersion})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: out, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

type yearReq struct {
	FiscalCalendarID string `json:"fiscalCalendarId"`
	YearNo           int32  `json:"yearNo"`
	Name             string `json:"name"`
	StartDate        string `json:"startDate"`
	EndDate          string `json:"endDate"`
}

func (s *Server) createYear(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req yearReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if req.YearNo <= 0 || strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("yearNo (> 0) dan name wajib diisi"))
		return
	}
	calID, err := httpx.ParseUUIDParam("fiscalCalendarId", req.FiscalCalendarID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	start, err := platform.ParseDate(req.StartDate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("startDate harus YYYY-MM-DD"))
		return
	}
	end, err := platform.ParseDate(req.EndDate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("endDate harus YYYY-MM-DD"))
		return
	}
	if end.Time.Before(start.Time) {
		httpx.WriteError(w, r, httpx.BadRequest("endDate tidak boleh sebelum startDate"))
		return
	}
	created, err := s.repo.CreateFiscalYear(r.Context(), scope, coadb.CreateFiscalYearParams{
		OrganizationID:   scope[0],
		FiscalCalendarID: calID,
		YearNo:           req.YearNo,
		Name:             strings.TrimSpace(req.Name),
		StartDate:        start,
		EndDate:          end,
		CreatedBy:        user,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "fiscal year sudah ada"))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, map[string]any{"id": created.ID.String(), "status": created.Status, "rowVersion": created.RowVersion})
}

func (s *Server) listYears(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListFiscalYearsWithCount(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{
			"id":               row.ID.String(),
			"fiscalCalendarId": row.FiscalCalendarID.String(),
			"yearNo":           row.YearNo,
			"name":             row.Name,
			"startDate":        platform.FormatDate(row.StartDate),
			"endDate":          platform.FormatDate(row.EndDate),
			"status":           row.Status,
			"rowVersion":       row.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: out, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

type periodReq struct {
	FiscalYearID string `json:"fiscalYearId"`
	PeriodNo     int32  `json:"periodNo"`
	StartDate    string `json:"startDate"`
	EndDate      string `json:"endDate"`
}

func (s *Server) createPeriod(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req periodReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if req.PeriodNo <= 0 {
		httpx.WriteError(w, r, httpx.BadRequest("periodNo harus > 0"))
		return
	}
	yearID, err := httpx.ParseUUIDParam("fiscalYearId", req.FiscalYearID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	start, err := platform.ParseDate(req.StartDate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("startDate harus YYYY-MM-DD"))
		return
	}
	end, err := platform.ParseDate(req.EndDate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("endDate harus YYYY-MM-DD"))
		return
	}
	if end.Time.Before(start.Time) {
		httpx.WriteError(w, r, httpx.BadRequest("endDate tidak boleh sebelum startDate"))
		return
	}
	created, err := s.repo.CreateFiscalPeriod(r.Context(), scope, coadb.CreateFiscalPeriodParams{
		OrganizationID: scope[0],
		FiscalYearID:   yearID,
		PeriodNo:       req.PeriodNo,
		StartDate:      start,
		EndDate:        end,
		CreatedBy:      user,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "fiscal period sudah ada"))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, map[string]any{"id": created.ID.String(), "status": created.Status, "rowVersion": created.RowVersion})
}

func (s *Server) listPeriods(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	fy := strings.TrimSpace(r.URL.Query().Get("fiscalYearId"))
	if fy != "" {
		u, err := httpx.ParseUUIDParam("fiscalYearId", fy)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		fy = u.String()
	}
	rows, total, err := s.repo.ListFiscalPeriodsWithCount(r.Context(), scope, fy, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{
			"id":           row.ID.String(),
			"fiscalYearId": row.FiscalYearID.String(),
			"periodNo":     row.PeriodNo,
			"startDate":    platform.FormatDate(row.StartDate),
			"endDate":      platform.FormatDate(row.EndDate),
			"status":       row.Status,
			"rowVersion":   row.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: out, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) transitionPeriod(expectedStatus, newStatus string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scope, user, ok := requireWrite(w, r)
		if !ok {
			return
		}
		id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		rowVer, err := httpx.IfMatch(r)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		p, nextVer, err := s.repo.TransitionFiscalPeriodAndEmit(r.Context(), scope, id, rowVer, expectedStatus, newStatus, user, httpx.CorrelationUUID(r))
		if err != nil {
			httpx.WriteError(w, r, httpx.MapDBError(err, ""))
			return
		}
		httpx.WriteIDResponse(w, http.StatusOK, nextVer, map[string]any{
			"id":           p.ID.String(),
			"fiscalYearId": p.FiscalYearID.String(),
			"periodNo":     p.PeriodNo,
			"status":       p.Status,
			"rowVersion":   nextVer,
		})
	}
}

func isValidAccountType(t string) bool {
	switch t {
	case "Asset", "Liability", "Equity", "Revenue", "Expense":
		return true
	default:
		return false
	}
}

func deflt(v, d string) string {
	if s := strings.TrimSpace(v); s != "" {
		return s
	}
	return d
}
