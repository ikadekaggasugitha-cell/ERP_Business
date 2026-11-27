package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/coa-service/internal/coadb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

type stubStore struct {
	accounts []coadb.ListAccountsRow
	total    int64
	period   coadb.FinFiscalPeriod
	nextVer  int32
	transErr error
}

func (s *stubStore) CreateChartOfAccounts(_ context.Context, _ []uuid.UUID, _ coadb.CreateChartOfAccountsParams) (coadb.CreateChartOfAccountsRow, error) {
	return coadb.CreateChartOfAccountsRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (s *stubStore) GetChartOfAccounts(_ context.Context, _ []uuid.UUID, id uuid.UUID) (coadb.CoaChartOfAccount, error) {
	return coadb.CoaChartOfAccount{ID: id, Code: "PSAK", Name: "PSAK", Structure: []byte(`{}`), RowVersion: 1}, nil
}
func (s *stubStore) ListChartsWithCount(_ context.Context, _ []uuid.UUID, _, _ int32) ([]coadb.ListChartsOfAccountsRow, int64, error) {
	return nil, 0, nil
}
func (s *stubStore) UpdateChartOfAccounts(_ context.Context, _ []uuid.UUID, _ coadb.UpdateChartOfAccountsParams) (int32, error) {
	return 2, nil
}
func (s *stubStore) DeleteChartOfAccounts(_ context.Context, _ []uuid.UUID, _ coadb.DeleteChartOfAccountsParams) (int64, error) {
	return 1, nil
}
func (s *stubStore) CreateAndEmit(_ context.Context, _ []uuid.UUID, _ coadb.CreateAccountParams, _ coadb.InsertOutboxParams) (coadb.CreateAccountRow, error) {
	return coadb.CreateAccountRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (s *stubStore) GetByID(_ context.Context, _ []uuid.UUID, id uuid.UUID) (coadb.CoaAccount, error) {
	return coadb.CoaAccount{ID: id, Code: "1100", Name: "Cash", AccountType: "Asset", AllowPosting: true, RowVersion: 1}, nil
}
func (s *stubStore) ListWithCount(_ context.Context, _ []uuid.UUID, _ string, _, _ int32) ([]coadb.ListAccountsRow, int64, error) {
	return s.accounts, s.total, nil
}
func (s *stubStore) UpdateAccountAndEmit(_ context.Context, _ []uuid.UUID, _ coadb.UpdateAccountParams, _ coadb.InsertOutboxParams) (int32, error) {
	return 2, nil
}
func (s *stubStore) DeleteAccount(_ context.Context, _ []uuid.UUID, _ coadb.DeleteAccountParams) (int64, error) {
	return 1, nil
}
func (s *stubStore) CreateFinancialOrg(_ context.Context, _ []uuid.UUID, _ coadb.CreateFinancialOrganizationParams) (coadb.CreateFinancialOrganizationRow, error) {
	return coadb.CreateFinancialOrganizationRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (s *stubStore) ListFinancialOrgsWithCount(_ context.Context, _ []uuid.UUID, _, _ int32) ([]coadb.ListFinancialOrganizationsRow, int64, error) {
	return nil, 0, nil
}
func (s *stubStore) CreateLedger(_ context.Context, _ []uuid.UUID, _ coadb.CreateLedgerParams) (coadb.CreateLedgerRow, error) {
	return coadb.CreateLedgerRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (s *stubStore) ListLedgersWithCount(_ context.Context, _ []uuid.UUID, _, _ int32) ([]coadb.ListLedgersRow, int64, error) {
	return nil, 0, nil
}
func (s *stubStore) UpdateLedger(_ context.Context, _ []uuid.UUID, _ coadb.UpdateLedgerParams) (int32, error) {
	return 2, nil
}
func (s *stubStore) CreateFiscalCalendar(_ context.Context, _ []uuid.UUID, _ coadb.CreateFiscalCalendarParams) (coadb.CreateFiscalCalendarRow, error) {
	return coadb.CreateFiscalCalendarRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (s *stubStore) ListFiscalCalendarsWithCount(_ context.Context, _ []uuid.UUID, _, _ int32) ([]coadb.ListFiscalCalendarsRow, int64, error) {
	return nil, 0, nil
}
func (s *stubStore) CreateFiscalYear(_ context.Context, _ []uuid.UUID, _ coadb.CreateFiscalYearParams) (coadb.CreateFiscalYearRow, error) {
	return coadb.CreateFiscalYearRow{ID: uuid.New(), Status: "Open", RowVersion: 1}, nil
}
func (s *stubStore) ListFiscalYearsWithCount(_ context.Context, _ []uuid.UUID, _, _ int32) ([]coadb.ListFiscalYearsRow, int64, error) {
	return nil, 0, nil
}
func (s *stubStore) UpdateFiscalYear(_ context.Context, _ []uuid.UUID, _ coadb.UpdateFiscalYearParams) (int32, error) {
	return 2, nil
}
func (s *stubStore) CreateFiscalPeriod(_ context.Context, _ []uuid.UUID, _ coadb.CreateFiscalPeriodParams) (coadb.CreateFiscalPeriodRow, error) {
	return coadb.CreateFiscalPeriodRow{ID: uuid.New(), Status: "Open", RowVersion: 1}, nil
}
func (s *stubStore) ListFiscalPeriodsWithCount(_ context.Context, _ []uuid.UUID, _ string, _, _ int32) ([]coadb.ListFiscalPeriodsRow, int64, error) {
	return nil, 0, nil
}
func (s *stubStore) TransitionFiscalPeriodAndEmit(_ context.Context, _ []uuid.UUID, id uuid.UUID, _ int32, _, newStatus string, _ uuid.UUID, _ uuid.UUID) (coadb.FinFiscalPeriod, int32, error) {
	if s.transErr != nil {
		return coadb.FinFiscalPeriod{}, 0, s.transErr
	}
	s.period.ID = id
	s.period.Status = newStatus
	s.period.RowVersion = s.nextVer
	return s.period, s.nextVer, nil
}

func TestCOAEndpointsAndPeriodTransitions(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	store := &stubStore{
		accounts: []coadb.ListAccountsRow{{ID: uuid.New(), Code: "1100", Name: "Cash", AccountType: "Asset", AllowPosting: true, RowVersion: 1}},
		total:    18,
		nextVer:  2,
	}
	mux := http.NewServeMux()
	NewServer(store).Register(mux)
	h := httpx.Middleware(mux)

	// Invalid accountType rejected cleanly.
	badAcct := `{"chartOfAccountsId":"` + uuid.NewString() + `","code":"9999","name":"Bogus","accountType":"Unknown"}`
	req := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(badAcct))
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-User-Id", userID.String())
	req.Header.Set("X-Permissions", "coa:write")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid accountType, got %d", rec.Code)
	}

	// Account list exposes real pagination totalElements & rowVersion.
	req = httptest.NewRequest(http.MethodGet, "/accounts", nil)
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-Permissions", "coa:read")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"totalElements":18`) {
		t.Fatalf("unexpected accounts list: %d %s", rec.Code, rec.Body.String())
	}

	// Fiscal period close transition returns updated status & ETag.
	pid := uuid.NewString()
	req = httptest.NewRequest(http.MethodPost, "/fiscal-periods/"+pid+"/close", nil)
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-User-Id", userID.String())
	req.Header.Set("X-Permissions", "coa:write")
	req.Header.Set("If-Match", `"1"`)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Header().Get("ETag") != `"2"` {
		t.Fatalf("close failed: %d %s", rec.Code, rec.Body.String())
	}
}

func TestAccountHierarchyAndMetadataUpdates(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	rootID := uuid.New()
	childID := uuid.New()

	store := &stubStore{
		accounts: []coadb.ListAccountsRow{
			{ID: rootID, Code: "1000", Name: "Assets", AccountType: "Asset", AllowPosting: false, RowVersion: 1},
			{ID: childID, Code: "1100", Name: "Cash", AccountType: "Asset", ParentID: &rootID, AllowPosting: true, RowVersion: 1},
		},
		total: 2,
	}
	mux := http.NewServeMux()
	NewServer(store).Register(mux)
	h := httpx.Middleware(mux)

	// 1. GET /accounts/hierarchy nests 1100 under 1000
	req := httptest.NewRequest(http.MethodGet, "/accounts/hierarchy", nil)
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-Permissions", "coa:read")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("hierarchy got %d: %s", rec.Code, rec.Body.String())
	}
	var tree struct {
		Data       []accountTreeNode `json:"data"`
		TotalNodes int               `json:"totalNodes"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &tree); err != nil {
		t.Fatal(err)
	}
	if tree.TotalNodes != 2 || len(tree.Data) != 1 || len(tree.Data[0].Children) != 1 || tree.Data[0].Children[0].Code != "1100" {
		t.Fatalf("unexpected hierarchy tree: %+v", tree)
	}

	// 2. PUT /ledgers/{id} with If-Match
	req = httptest.NewRequest(http.MethodPut, "/ledgers/"+uuid.NewString(), strings.NewReader(`{"code":"PRIMARY","ledgerType":"Primary","currencyCode":"IDR"}`))
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-User-Id", userID.String())
	req.Header.Set("X-Permissions", "coa:write")
	req.Header.Set("If-Match", `"1"`)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Header().Get("ETag") != `"2"` {
		t.Fatalf("PUT ledger got %d ETag=%q body=%s", rec.Code, rec.Header().Get("ETag"), rec.Body.String())
	}

	// 3. PUT /fiscal-years/{id} with If-Match
	req = httptest.NewRequest(http.MethodPut, "/fiscal-years/"+uuid.NewString(), strings.NewReader(`{"name":"FY 2026","startDate":"2026-01-01","endDate":"2026-12-31","status":"Open"}`))
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-User-Id", userID.String())
	req.Header.Set("X-Permissions", "coa:write")
	req.Header.Set("If-Match", `"1"`)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Header().Get("ETag") != `"2"` {
		t.Fatalf("PUT fiscal-year got %d ETag=%q body=%s", rec.Code, rec.Header().Get("ETag"), rec.Body.String())
	}
}

