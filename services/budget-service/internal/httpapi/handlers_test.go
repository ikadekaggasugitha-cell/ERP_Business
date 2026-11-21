package httpapi_test

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/budget-service/internal/budgetdb"
	"github.com/sinergiinvestasiproperti/erp/services/budget-service/internal/httpapi"
	"github.com/sinergiinvestasiproperti/erp/services/budget-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type mockBudStore struct {
	httpapi.Store
	checkRes  repository.BudgetCheckResult
	lineCalls int
}

func (m *mockBudStore) CreateBudget(_ context.Context, _ []uuid.UUID, _ budgetdb.CreateBudgetParams) (budgetdb.CreateBudgetRow, error) {
	return budgetdb.CreateBudgetRow{ID: uuid.New(), Status: "Draft", RowVersion: 1}, nil
}
func (m *mockBudStore) GetBudget(_ context.Context, _ []uuid.UUID, id uuid.UUID) (budgetdb.BudBudget, []budgetdb.BudBudgetLine, error) {
	amt, _ := platform.ParseNumeric("1000000.0000")
	return budgetdb.BudBudget{
			ID: id, OrganizationID: uuid.New(), Code: "BUD-2026", Name: "FY2026", Version: 1, Status: "Approved", IsActive: true, RowVersion: 2,
		}, []budgetdb.BudBudgetLine{{
			ID: uuid.New(), BudgetID: id, AccountRef: uuid.New(), PeriodNo: 1, BudgetAmount: amt, RowVersion: 1,
		}}, nil
}
func (m *mockBudStore) AddBudgetLine(_ context.Context, _ []uuid.UUID, _ budgetdb.CreateBudgetLineParams) (budgetdb.CreateBudgetLineRow, error) {
	m.lineCalls++
	return budgetdb.CreateBudgetLineRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (m *mockBudStore) CheckAvailability(_ context.Context, _ []uuid.UUID, _ uuid.UUID, _ *uuid.UUID, _ *int32, _ *string, _ *big.Rat) (repository.BudgetCheckResult, error) {
	return m.checkRes, nil
}

func TestBudgetPolicyEvaluationAndEndpoints(t *testing.T) {
	bid := uuid.New()
	resHard := repository.EvaluateBudgetPolicy(bid, big.NewRat(1000, 1), big.NewRat(800, 1), big.NewRat(300, 1), "HardStop")
	if resHard.Allowed || resHard.AvailableAmount != "200.0000" {
		t.Fatalf("expected HardStop block, got %+v", resHard)
	}
	resSoft := repository.EvaluateBudgetPolicy(bid, big.NewRat(1000, 1), big.NewRat(800, 1), big.NewRat(300, 1), "SoftWarning")
	if !resSoft.Allowed || !resSoft.Warning {
		t.Fatalf("expected SoftWarning allow+warn, got %+v", resSoft)
	}

	st := &mockBudStore{checkRes: resHard}
	mux := http.NewServeMux()
	httpapi.NewServer(st).Register(mux)
	h := httpx.Middleware(mux)
	orgID := uuid.NewString()

	req := httptest.NewRequest("POST", "/budgets/check", strings.NewReader(`{"budgetId":"`+bid.String()+`","requestedAmount":"300.0000"}`))
	req.Header.Set("X-Organization-Id", orgID)
	req.Header.Set("X-User-Id", uuid.NewString())
	req.Header.Set("X-Permissions", "budget:read")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 on HardStop exceeded, got %d %s", rec.Code, rec.Body.String())
	}

	// Verify camelCase decimal string DTO on GET /budgets/{id} + ETag
	getReq := httptest.NewRequest("GET", "/budgets/"+bid.String(), nil)
	getReq.Header.Set("X-Organization-Id", orgID)
	getReq.Header.Set("X-Permissions", "budget:read")
	getRec := httptest.NewRecorder()
	h.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK || getRec.Header().Get("ETag") != `"2"` {
		t.Fatalf("expected 200 with ETag \"2\", got %d etag=%q", getRec.Code, getRec.Header().Get("ETag"))
	}
	var out struct {
		Budget map[string]any   `json:"budget"`
		Lines  []map[string]any `json:"lines"`
	}
	if err := json.Unmarshal(getRec.Body.Bytes(), &out); err != nil || len(out.Lines) != 1 || out.Lines[0]["budgetAmount"] != "1000000.0000" {
		t.Fatalf("unexpected budget DTO wire payload: %s", getRec.Body.String())
	}

	// Verify idempotency replay on POST /budgets/{id}/lines
	lineBody := `{"accountRef":"` + uuid.NewString() + `","periodNo":1,"budgetAmount":"500000.0000"}`
	for i := 0; i < 2; i++ {
		lReq := httptest.NewRequest("POST", "/budgets/"+bid.String()+"/lines", strings.NewReader(lineBody))
		lReq.Header.Set("X-Organization-Id", orgID)
		lReq.Header.Set("X-User-Id", uuid.NewString())
		lReq.Header.Set("X-Permissions", "budget:write")
		lReq.Header.Set("Idempotency-Key", "line-1")
		lRec := httptest.NewRecorder()
		h.ServeHTTP(lRec, lReq)
		if lRec.Code != http.StatusCreated {
			t.Fatalf("expected 201 on line %d, got %d %s", i, lRec.Code, lRec.Body.String())
		}
	}
	if st.lineCalls != 1 {
		t.Fatalf("expected AddBudgetLine called once via idempotency, got %d", st.lineCalls)
	}
}
