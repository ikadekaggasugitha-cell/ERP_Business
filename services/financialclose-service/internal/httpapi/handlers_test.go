package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/financialclose-service/internal/fcldb"
	"github.com/sinergiinvestasiproperti/erp/services/financialclose-service/internal/httpapi"
	"github.com/sinergiinvestasiproperti/erp/services/financialclose-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type mockFCLStore struct {
	httpapi.Store
	closeCalls int
}

func (m *mockFCLStore) ListTemplates(_ context.Context, _ []uuid.UUID, _, _ int32) ([]fcldb.FclClosingChecklistTemplate, int64, error) {
	return []fcldb.FclClosingChecklistTemplate{{
		ID: uuid.New(), OrganizationID: platform.SystemOrgID, Code: "STD-MONTHLY-CLOSE", Name: "Monthly Close",
		FinancialOrganizationRef: platform.SystemOrgID, RowVersion: 1,
	}}, 1, nil
}
func (m *mockFCLStore) CreateClosingRun(_ context.Context, _ []uuid.UUID, _ fcldb.CreateClosingRunParams, _ *uuid.UUID) (fcldb.CreateClosingRunRow, error) {
	return fcldb.CreateClosingRunRow{ID: uuid.New(), Status: "Open", RowVersion: 1}, nil
}
func (m *mockFCLStore) GetClosingRun(_ context.Context, _ []uuid.UUID, id uuid.UUID) (fcldb.FclClosingRun, []fcldb.FclClosingTask, error) {
	return fcldb.FclClosingRun{ID: id, Status: "Open", RowVersion: 1}, []fcldb.FclClosingTask{
		{ID: uuid.New(), TaskName: "GL Trial Balance Check", Status: "Completed"},
		{ID: uuid.New(), TaskName: "AP Subledger Close", Status: "Completed"},
	}, nil
}
func (m *mockFCLStore) TransitionClosingRunAndEmit(_ context.Context, _ []uuid.UUID, _ uuid.UUID, _ int32, _ string, _, _ uuid.UUID) (int32, error) {
	m.closeCalls++
	return 2, nil
}
func (m *mockFCLStore) RequestReopening(_ context.Context, _ []uuid.UUID, _ fcldb.CreatePeriodReopeningParams) (fcldb.CreatePeriodReopeningRow, error) {
	return fcldb.CreatePeriodReopeningRow{ID: uuid.New(), RowVersion: 1}, nil
}

func TestFinancialCloseEndpointsAndValidation(t *testing.T) {
	// Exact FX revaluation math (PSAK 10 / IAS 21)
	res, err := repository.ComputeFXRevaluation(uuid.New(), uuid.New(), []repository.FXPositionInput{{
		AccountRef: uuid.New(), CurrencyCode: "USD", ForeignAmount: "1000.0000", BookRate: "15500.00000000", ClosingRate: "15800.00000000",
	}})
	if err != nil || res.TotalUnrealizedGain != "300000.0000" {
		t.Fatalf("expected 300000.0000 FX gain, got %+v err=%v", res, err)
	}

	st := &mockFCLStore{}
	mux := http.NewServeMux()
	httpapi.NewServer(st).Register(mux)
	h := httpx.Middleware(mux)
	orgID, finOrg, period := uuid.NewString(), uuid.NewString(), uuid.NewString()

	// Verify camelCase DTO on GET /closing-templates
	tplReq := httptest.NewRequest("GET", "/closing-templates", nil)
	tplReq.Header.Set("X-Organization-Id", orgID)
	tplReq.Header.Set("X-Permissions", "financialclose:read")
	tplRec := httptest.NewRecorder()
	h.ServeHTTP(tplRec, tplReq)
	var tplEnv struct {
		Data []map[string]any `json:"data"`
	}
	if tplRec.Code != http.StatusOK || json.Unmarshal(tplRec.Body.Bytes(), &tplEnv) != nil || len(tplEnv.Data) != 1 || tplEnv.Data[0]["isSystem"] != true {
		t.Fatalf("unexpected template DTO wire payload: %d %s", tplRec.Code, tplRec.Body.String())
	}

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

	runID := uuid.NewString()
	for i := 0; i < 2; i++ {
		req = httptest.NewRequest("POST", "/closing-runs/"+runID+"/close", nil)
		req.Header.Set("X-Organization-Id", orgID)
		req.Header.Set("X-User-Id", uuid.NewString())
		req.Header.Set("X-Permissions", "financialclose:write")
		req.Header.Set("If-Match", `"1"`)
		req.Header.Set("Idempotency-Key", "close-1")
		rec = httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 close run on call %d, got %d %s", i, rec.Code, rec.Body.String())
		}
	}
	if st.closeCalls != 1 {
		t.Fatalf("expected idempotent replay on close run, calls=%d", st.closeCalls)
	}

	// POST /closing-runs/{id}/auto-reversals
	revReq := httptest.NewRequest("POST", "/closing-runs/"+runID+"/auto-reversals", strings.NewReader(`{"reversalDate":"2026-12-01","entries":[{"reference":"ACC-1","accountCode":"2190","contraAccountCode":"6100","description":"Accrued utilities","amount":"1500000.0000"}]}`))
	revReq.Header.Set("X-Organization-Id", orgID)
	revReq.Header.Set("X-User-Id", uuid.NewString())
	revReq.Header.Set("X-Permissions", "financialclose:write")
	revReq.Header.Set("Idempotency-Key", "rev-1")
	revRec := httptest.NewRecorder()
	h.ServeHTTP(revRec, revReq)
	if revRec.Code != http.StatusOK || !strings.Contains(revRec.Body.String(), `"totalAmount":"1500000.0000"`) {
		t.Fatalf("auto-reversals got %d %s", revRec.Code, revRec.Body.String())
	}

	// POST /closing-runs/{id}/verify-subledgers
	verReq := httptest.NewRequest("POST", "/closing-runs/"+runID+"/verify-subledgers", nil)
	verReq.Header.Set("X-Organization-Id", orgID)
	verReq.Header.Set("X-User-Id", uuid.NewString())
	verReq.Header.Set("X-Permissions", "financialclose:write")
	verRec := httptest.NewRecorder()
	h.ServeHTTP(verRec, verReq)
	if verRec.Code != http.StatusOK || !strings.Contains(verRec.Body.String(), `"readyToClose":true`) || !strings.Contains(verRec.Body.String(), `"passedCount":5`) {
		t.Fatalf("verify-subledgers got %d %s", verRec.Code, verRec.Body.String())
	}
}
