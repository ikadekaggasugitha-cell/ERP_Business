package httpapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/financialclose-service/internal/fcldb"
	"github.com/sinergiinvestasiproperti/erp/services/financialclose-service/internal/httpapi"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

type mockFCLStore struct{ httpapi.Store }

func (m *mockFCLStore) CreateClosingRun(_ context.Context, _ []uuid.UUID, _ fcldb.CreateClosingRunParams, _ *uuid.UUID) (fcldb.CreateClosingRunRow, error) {
	return fcldb.CreateClosingRunRow{ID: uuid.New(), Status: "Open", RowVersion: 1}, nil
}
func (m *mockFCLStore) TransitionClosingRunAndEmit(_ context.Context, _ []uuid.UUID, _ uuid.UUID, _ int32, _ string, _, _ uuid.UUID) (int32, error) {
	return 2, nil
}
func (m *mockFCLStore) RequestReopening(_ context.Context, _ []uuid.UUID, _ fcldb.CreatePeriodReopeningParams) (fcldb.CreatePeriodReopeningRow, error) {
	return fcldb.CreatePeriodReopeningRow{ID: uuid.New(), RowVersion: 1}, nil
}

func TestFinancialCloseEndpointsAndValidation(t *testing.T) {
	mux := http.NewServeMux()
	httpapi.NewServer(&mockFCLStore{}).Register(mux)
	h := httpx.Middleware(mux)
	orgID, finOrg, period := uuid.NewString(), uuid.NewString(), uuid.NewString()

	req := httptest.NewRequest("POST", "/closing-runs", strings.NewReader(`{"financialOrganizationRef":"`+finOrg+`","fiscalPeriodRef":"`+period+`","runType":"Invalid"}`))
	req.Header.Set("X-Organization-Id", orgID)
	req.Header.Set("X-User-Id", uuid.NewString())
	req.Header.Set("X-Permissions", "financialclose:write")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 invalid runType, got %d", rec.Code)
	}

	req = httptest.NewRequest("POST", "/closing-runs", strings.NewReader(`{"financialOrganizationRef":"`+finOrg+`","fiscalPeriodRef":"`+period+`","runType":"PeriodEnd"}`))
	req.Header.Set("X-Organization-Id", orgID)
	req.Header.Set("X-User-Id", uuid.NewString())
	req.Header.Set("X-Permissions", "financialclose:write")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 closing run, got %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest("POST", "/closing-runs/"+uuid.NewString()+"/close", nil)
	req.Header.Set("X-Organization-Id", orgID)
	req.Header.Set("X-User-Id", uuid.NewString())
	req.Header.Set("X-Permissions", "financialclose:write")
	req.Header.Set("If-Match", `"1"`)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 close run, got %d %s", rec.Code, rec.Body.String())
	}
}
