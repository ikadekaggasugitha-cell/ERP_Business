// Package httpapi implements CoA, Financial Organization, Ledger, and Fiscal
// Calendar/Year/Period endpoints with optimistic locking, RBAC, and real pagination.
package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/coa-service/internal/coadb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
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
	UpdateLedger(ctx context.Context, scope []uuid.UUID, arg coadb.UpdateLedgerParams) (int32, error)

	CreateFiscalCalendar(ctx context.Context, scope []uuid.UUID, arg coadb.CreateFiscalCalendarParams) (coadb.CreateFiscalCalendarRow, error)
	ListFiscalCalendarsWithCount(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]coadb.ListFiscalCalendarsRow, int64, error)

	CreateFiscalYear(ctx context.Context, scope []uuid.UUID, arg coadb.CreateFiscalYearParams) (coadb.CreateFiscalYearRow, error)
	ListFiscalYearsWithCount(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]coadb.ListFiscalYearsRow, int64, error)
	UpdateFiscalYear(ctx context.Context, scope []uuid.UUID, arg coadb.UpdateFiscalYearParams) (int32, error)

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
	mux.HandleFunc("GET /accounts/hierarchy", s.accountHierarchy)
	mux.HandleFunc("POST /accounts", s.createAccount)
	mux.HandleFunc("GET /accounts/{id}", s.getAccount)
	mux.HandleFunc("PUT /accounts/{id}", s.updateAccount)
	mux.HandleFunc("DELETE /accounts/{id}", s.deleteAccount)

	mux.HandleFunc("GET /financial-organizations", s.listFinOrgs)
	mux.HandleFunc("POST /financial-organizations", s.createFinOrg)
	mux.HandleFunc("GET /ledgers", s.listLedgers)
	mux.HandleFunc("POST /ledgers", s.createLedger)
	mux.HandleFunc("PUT /ledgers/{id}", s.updateLedger)

	mux.HandleFunc("GET /fiscal-calendars", s.listCalendars)
	mux.HandleFunc("POST /fiscal-calendars", s.createCalendar)
	mux.HandleFunc("GET /fiscal-years", s.listYears)
	mux.HandleFunc("POST /fiscal-years", s.createYear)
	mux.HandleFunc("PUT /fiscal-years/{id}", s.updateFiscalYear)
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
