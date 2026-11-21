package httpapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/dimension-service/internal/dimensiondb"
	"github.com/sinergiinvestasiproperti/erp/services/dimension-service/internal/httpapi"
	"github.com/sinergiinvestasiproperti/erp/services/dimension-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

// mockStore embeds the Store interface so only exercised methods are implemented.
type mockStore struct {
	httpapi.Store
	createdCC  *dimensiondb.CreateCostCenterParams
	createdARU *dimensiondb.CreateAllocationRuleParams
	execCalls  int
}

func (m *mockStore) GetCostCenter(_ context.Context, _ []uuid.UUID, id uuid.UUID) (dimensiondb.CcCostCenter, error) {
	return dimensiondb.CcCostCenter{ID: id, Code: "CC-100", Name: "Ops", Status: "Active", RowVersion: 3}, nil
}

func (m *mockStore) CreateCostCenter(_ context.Context, _ []uuid.UUID, a dimensiondb.CreateCostCenterParams, _ uuid.UUID) (dimensiondb.CreateCostCenterRow, error) {
	m.createdCC = &a
	return dimensiondb.CreateCostCenterRow{ID: uuid.New(), RowVersion: 1}, nil
}

func (m *mockStore) CreateAllocationRule(_ context.Context, _ []uuid.UUID, a dimensiondb.CreateAllocationRuleParams, _ uuid.UUID) (dimensiondb.CreateAllocationRuleRow, error) {
	m.createdARU = &a
	return dimensiondb.CreateAllocationRuleRow{ID: uuid.New(), RowVersion: 1}, nil
}

func (m *mockStore) ExecuteAllocationRule(_ context.Context, _ []uuid.UUID, id uuid.UUID, _ string, _ *uuid.UUID, _ uuid.UUID) (dimensiondb.CcCostAllocationRule, error) {
	m.execCalls++
	return dimensiondb.CcCostAllocationRule{
		ID: id, SourceCostCenterID: uuid.New(), Name: "Overhead", AllocationBasis: "headcount",
		Targets: []byte(`[{"costCenterId":"33333333-3333-3333-3333-333333333333","percentage":"60"},{"costCenterId":"44444444-4444-4444-4444-444444444444","percentage":"40"}]`),
		IsActive: true, RowVersion: 1,
	}, nil
}

func (m *mockStore) CreateCostCenterAssignment(_ context.Context, _ []uuid.UUID, _ dimensiondb.CreateCostCenterAssignmentParams) (dimensiondb.CreateCostCenterAssignmentRow, error) {
	return dimensiondb.CreateCostCenterAssignmentRow{ID: uuid.New(), RowVersion: 1}, nil
}

func (m *mockStore) CreateProfitCenterAssignment(_ context.Context, _ []uuid.UUID, _ dimensiondb.CreateProfitCenterAssignmentParams) (dimensiondb.CreateProfitCenterAssignmentRow, error) {
	return dimensiondb.CreateProfitCenterAssignmentRow{ID: uuid.New(), RowVersion: 1}, nil
}

func newMux(st httpapi.Store) http.Handler {
	mux := http.NewServeMux()
	httpapi.NewServer(st).Register(mux)
	return httpx.Middleware(mux)
}

func do(h http.Handler, method, path, body string, perms string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("X-Organization-Id", "11111111-1111-1111-1111-111111111111")
	req.Header.Set("X-User-Id", "22222222-2222-2222-2222-222222222222")
	req.Header.Set("Content-Type", "application/json")
	if perms != "" {
		req.Header.Set("X-Permissions", perms)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestCreateCostCenter_OK(t *testing.T) {
	st := &mockStore{}
	rec := do(newMux(st), "POST", "/cost-centers",
		`{"code":"CC-100","name":"Ops","effectiveFrom":"2026-01-01"}`, "dimension:write", nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("want 201 got %d: %s", rec.Code, rec.Body)
	}
	if st.createdCC == nil || st.createdCC.Status != "Draft" {
		t.Fatalf("default status should be Draft, got %+v", st.createdCC)
	}
}

func TestCreateCostCenter_Validation(t *testing.T) {
	cases := map[string]struct {
		body string
		want int
	}{
		"missing code":   {`{"name":"x","effectiveFrom":"2026-01-01"}`, 400},
		"missing name":   {`{"code":"A","effectiveFrom":"2026-01-01"}`, 400},
		"bad date":       {`{"code":"A","name":"x","effectiveFrom":"01/01/2026"}`, 400},
		"bad status":     {`{"code":"A","name":"x","status":"Nope","effectiveFrom":"2026-01-01"}`, 422},
		"to before from": {`{"code":"A","name":"x","effectiveFrom":"2026-02-01","effectiveTo":"2026-01-01"}`, 422},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			rec := do(newMux(&mockStore{}), "POST", "/cost-centers", c.body, "dimension:write", nil)
			if rec.Code != c.want {
				t.Fatalf("want %d got %d: %s", c.want, rec.Code, rec.Body)
			}
		})
	}
}

func TestPermissionDenied(t *testing.T) {
	rec := do(newMux(&mockStore{}), "POST", "/cost-centers",
		`{"code":"A","name":"x","effectiveFrom":"2026-01-01"}`, "dimension:read", nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403 got %d", rec.Code)
	}
}

func TestMissingScopeFailsClosed(t *testing.T) {
	req := httptest.NewRequest("GET", "/cost-centers", nil)
	rec := httptest.NewRecorder()
	newMux(&mockStore{}).ServeHTTP(rec, req)
	if rec.Code < 400 {
		t.Fatalf("expected error without org scope, got %d", rec.Code)
	}
}

func TestUpdateRequiresIfMatch(t *testing.T) {
	id := uuid.New().String()
	rec := do(newMux(&mockStore{}), "PUT", "/cost-centers/"+id,
		`{"name":"x","effectiveFrom":"2026-01-01"}`, "dimension:write", nil)
	if rec.Code < 400 {
		t.Fatalf("expected If-Match error, got %d", rec.Code)
	}
}

func TestUpdateSelfParentRejected(t *testing.T) {
	id := uuid.New()
	body := `{"name":"x","effectiveFrom":"2026-01-01","parentId":"` + id.String() + `"}`
	rec := do(newMux(&mockStore{}), "PUT", "/cost-centers/"+id.String(), body, "dimension:write",
		map[string]string{"If-Match": `"1"`})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422 got %d: %s", rec.Code, rec.Body)
	}
}

func TestAllocationRule_Totals(t *testing.T) {
	src := uuid.New().String()
	a, b := uuid.New().String(), uuid.New().String()
	mk := func(tg string) string {
		return `{"sourceCostCenterId":"` + src + `","name":"r","allocationBasis":"headcount","targets":` + tg + `}`
	}
	cases := map[string]struct {
		body string
		want int
	}{
		"ok 60/40 decimal exact": {mk(`[{"costCenterId":"` + a + `","percentage":"33.3333"},{"costCenterId":"` + b + `","percentage":"66.6667"}]`), 201},
		"sum 99.99":              {mk(`[{"costCenterId":"` + a + `","percentage":"50"},{"costCenterId":"` + b + `","percentage":"49.99"}]`), 422},
		"self target":            {mk(`[{"costCenterId":"` + src + `","percentage":"100"}]`), 422},
		"duplicate":              {mk(`[{"costCenterId":"` + a + `","percentage":"50"},{"costCenterId":"` + a + `","percentage":"50"}]`), 422},
		"negative":               {mk(`[{"costCenterId":"` + a + `","percentage":"-5"},{"costCenterId":"` + b + `","percentage":"105"}]`), 422},
		"empty":                  {mk(`[]`), 400},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			rec := do(newMux(&mockStore{}), "POST", "/allocation-rules", c.body, "dimension:write", nil)
			if rec.Code != c.want {
				t.Fatalf("want %d got %d: %s", c.want, rec.Code, rec.Body)
			}
		})
	}
}

func TestExecuteAllocationAndAssignments(t *testing.T) {
	st := &mockStore{}
	h := newMux(st)

	// GET /cost-centers/{id} sets ETag header
	getRec := do(h, "GET", "/cost-centers/"+uuid.NewString(), "", "dimension:read", nil)
	if getRec.Code != http.StatusOK || getRec.Header().Get("ETag") != `"3"` {
		t.Fatalf("expected 200 with ETag \"3\", got %d etag=%q", getRec.Code, getRec.Header().Get("ETag"))
	}

	// Execute allocation computes exact fixed-point 60/40 distribution + idempotency replay
	ruleID := uuid.NewString()
	rec := do(h, "POST", "/allocation-rules/"+ruleID+"/execute", `{"amount":"1000.0000"}`, "dimension:write", map[string]string{"Idempotency-Key": "exec-1"})
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"600.0000"`) || !strings.Contains(rec.Body.String(), `"400.0000"`) {
		t.Fatalf("unexpected execute allocation response: %d %s", rec.Code, rec.Body.String())
	}
	replay := do(h, "POST", "/allocation-rules/"+ruleID+"/execute", `{"amount":"1000.0000"}`, "dimension:write", map[string]string{"Idempotency-Key": "exec-1"})
	if replay.Code != http.StatusOK || replay.Header().Get("Idempotency-Replayed") != "true" || st.execCalls != 1 {
		t.Fatalf("expected idempotent replay on executeAllocationRule, got %d calls=%d", replay.Code, st.execCalls)
	}

	// Verify rounding remainder assigned to final target so sum == totalAmount
	dists := repository.ComputeDistributions([]byte(`[{"costCenterId":"11111111-1111-1111-1111-111111111111","percentage":"33.3333"},{"costCenterId":"22222222-2222-2222-2222-222222222222","percentage":"33.3333"},{"costCenterId":"33333333-3333-3333-3333-333333333333","percentage":"33.3334"}]`), "100.0000")
	if len(dists) != 3 || dists[0]["amount"] != "33.3333" || dists[1]["amount"] != "33.3333" || dists[2]["amount"] != "33.3334" {
		t.Fatalf("unexpected rounding remainder distribution: %+v", dists)
	}

	// Create cost center & profit center assignments
	ccID, pcID, orgRef := uuid.NewString(), uuid.NewString(), uuid.NewString()
	rec = do(h, "POST", "/cost-center-assignments", `{"costCenterId":"`+ccID+`","organizationRef":"`+orgRef+`"}`, "dimension:write", nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create cc assignment want 201 got %d: %s", rec.Code, rec.Body.String())
	}
	rec = do(h, "POST", "/profit-center-assignments", `{"profitCenterId":"`+pcID+`","organizationRef":"`+orgRef+`"}`, "dimension:write", nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create pc assignment want 201 got %d: %s", rec.Code, rec.Body.String())
	}
}
