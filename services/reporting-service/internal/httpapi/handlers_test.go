package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/reporting-service/internal/reportingdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type fakeStore struct{}

func (f *fakeStore) ListGLWithCount(context.Context, []uuid.UUID, reportingdb.ListGLBalanceSnapshotsParams) ([]reportingdb.ListGLBalanceSnapshotsRow, int64, error) {
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

