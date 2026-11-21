package httpapi_test

import (
	"context"
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
)

type mockBudStore struct {
	httpapi.Store
	checkRes repository.BudgetCheckResult
}

func (m *mockBudStore) CreateBudget(_ context.Context, _ []uuid.UUID, _ budgetdb.CreateBudgetParams) (budgetdb.CreateBudgetRow, error) {
	return budgetdb.CreateBudgetRow{ID: uuid.New(), Status: "Draft", RowVersion: 1}, nil
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

	req := httptest.NewRequest("POST", "/budgets/check", strings.NewReader(`{"budgetId":"`+bid.String()+`","requestedAmount":"300.0000"}`))
	req.Header.Set("X-Organization-Id", uuid.NewString())
	req.Header.Set("X-User-Id", uuid.NewString())
	req.Header.Set("X-Permissions", "budget:read")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 on HardStop exceeded, got %d %s", rec.Code, rec.Body.String())
	}
}
