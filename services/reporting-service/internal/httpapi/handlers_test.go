package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/reporting-service/internal/reportingdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type fakeStore struct {
	rows []reportingdb.ListGLBalanceSnapshotsRow
}

func (f *fakeStore) ListGLWithCount(context.Context, []uuid.UUID, reportingdb.ListGLBalanceSnapshotsParams) ([]reportingdb.ListGLBalanceSnapshotsRow, int64, error) {
	if len(f.rows) > 0 {
		return f.rows, int64(len(f.rows)), nil
	}
	bal, _ := platform.ParseNumeric("500.0000")
	return []reportingdb.ListGLBalanceSnapshotsRow{{
		ID: uuid.New(), FiscalPeriodID: uuid.New(), AccountCode: "JV-1", ClosingBalance: bal,
		RefreshedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}}, 42, nil
}
func (f *fakeStore) ListSalesWithCount(context.Context, []uuid.UUID, reportingdb.ListSalesSummariesParams) ([]reportingdb.RptSalesSummary, int64, error) {
	return nil, 0, nil
}
func (f *fakeStore) ListInventoryWithCount(context.Context, []uuid.UUID, reportingdb.ListInventoryPositionsParams) ([]reportingdb.RptInventoryPosition, int64, error) {
	return nil, 0, nil
}

func TestReportingEndpointsAndPaginationMeta(t *testing.T) {
	mux := http.NewServeMux()
	NewServer(&fakeStore{}).Register(mux)
	h := httpx.Middleware(mux)

	req := httptest.NewRequest(http.MethodGet, "/gl-balance-snapshots?page=2&size=10", nil)
	req.Header.Set("X-Organization-Id", uuid.NewString())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var env httpx.ListEnvelope
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	if env.Page.TotalElements != 42 || env.Page.TotalPages != 5 || env.Page.Number != 2 {
		t.Fatalf("unexpected page meta: %+v", env.Page)
	}
}

func TestFinancialStatementsAndCSVExport(t *testing.T) {
	fpID := uuid.New()
	makeRow := func(code, bal string) reportingdb.ListGLBalanceSnapshotsRow {
		n, _ := platform.ParseNumeric(bal)
		return reportingdb.ListGLBalanceSnapshotsRow{
			ID: uuid.New(), FiscalPeriodID: fpID, AccountCode: code, ClosingBalance: n,
			RefreshedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		}
	}
	// Balanced trial balance:
	// 1100 Cash +1000, 1500 Equipment +500, 2100 AP -300, 3100 Capital -800,
	// 4100 Revenue -1000, 5100 COGS +400, 6100 OpEx +200 -> Sum = 0.
	store := &fakeStore{
		rows: []reportingdb.ListGLBalanceSnapshotsRow{
			makeRow("1100", "1000.0000"),
			makeRow("1500", "500.0000"),
			makeRow("2100", "-300.0000"),
			makeRow("3100", "-800.0000"),
			makeRow("4100", "-1000.0000"),
			makeRow("5100", "400.0000"),
			makeRow("6100", "200.0000"),
		},
	}

	mux := http.NewServeMux()
	NewServer(store).Register(mux)
	h := httpx.Middleware(mux)
	orgID := uuid.NewString()

	// 1. Income Statement
	req := httptest.NewRequest(http.MethodGet, "/financial-statements/income-statement?fiscalPeriodId="+fpID.String(), nil)
	req.Header.Set("X-Organization-Id", orgID)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("income-statement got %d: %s", rec.Code, rec.Body.String())
	}
	var isResp financialStatementResp
	if err := json.Unmarshal(rec.Body.Bytes(), &isResp); err != nil {
		t.Fatal(err)
	}
	if isResp.Totals["revenue"] != "1000.0000" || isResp.Totals["grossProfit"] != "600.0000" || isResp.Totals["netIncome"] != "400.0000" {
		t.Fatalf("unexpected income statement totals: %+v", isResp.Totals)
	}

	// 2. Balance Sheet (Assets 1500 == Liabilities 300 + Equity 800 + NetIncome 400)
	req = httptest.NewRequest(http.MethodGet, "/financial-statements/balance-sheet", nil)
	req.Header.Set("X-Organization-Id", orgID)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("balance-sheet got %d: %s", rec.Code, rec.Body.String())
	}
	var bsResp financialStatementResp
	if err := json.Unmarshal(rec.Body.Bytes(), &bsResp); err != nil {
		t.Fatal(err)
	}
	if bsResp.IsBalanced == nil || !*bsResp.IsBalanced {
		t.Fatalf("expected balanced sheet, got %+v", bsResp)
	}
	if bsResp.Totals["totalAssets"] != "1500.0000" || bsResp.Totals["totalLiabilitiesAndEquity"] != "1500.0000" {
		t.Fatalf("unexpected BS totals: %+v", bsResp.Totals)
	}

	// 3. Cash Flow + CSV export
	req = httptest.NewRequest(http.MethodGet, "/financial-statements/cash-flow?format=csv", nil)
	req.Header.Set("X-Organization-Id", orgID)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Type"), "text/csv") {
		t.Fatalf("cash-flow csv got %d ct=%q", rec.Code, rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(rec.Body.String(), "netCashChange") {
		t.Fatalf("expected netCashChange in CSV body: %s", rec.Body.String())
	}

	// 4. Formula injection guard
	if got := sanitizeCSVCell("=CMD|'/C calc'!A0"); got != "'=CMD|'/C calc'!A0" {
		t.Fatalf("formula injection not neutralized: %q", got)
	}
	if got := sanitizeCSVCell("-500.0000"); got != "-500.0000" {
		t.Fatalf("negative decimal should be preserved: %q", got)
	}
}

