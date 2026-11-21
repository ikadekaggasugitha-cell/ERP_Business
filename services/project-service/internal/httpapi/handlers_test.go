package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/project-service/internal/prjdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

type fakeProjectStore struct {
	approvedRef string
}

func (f *fakeProjectStore) CreateProject(context.Context, []uuid.UUID, prjdb.CreateProjectParams) (prjdb.CreateProjectRow, error) {
	return prjdb.CreateProjectRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (f *fakeProjectStore) GetProject(context.Context, []uuid.UUID, uuid.UUID) (prjdb.PrjProject, error) {
	return prjdb.PrjProject{ID: uuid.New(), RowVersion: 1}, nil
}
func (f *fakeProjectStore) ListProjects(context.Context, []uuid.UUID, *string, int32, int32) ([]prjdb.PrjProject, int64, error) {
	return nil, 0, nil
}
func (f *fakeProjectStore) UpdateProject(context.Context, []uuid.UUID, prjdb.UpdateProjectParams) (int32, error) {
	return 2, nil
}
func (f *fakeProjectStore) TransitionProject(context.Context, []uuid.UUID, uuid.UUID, uuid.UUID, int32, string, string) (int32, error) {
	return 2, nil
}
func (f *fakeProjectStore) DeleteProject(context.Context, []uuid.UUID, uuid.UUID, uuid.UUID, int32) error {
	return nil
}
func (f *fakeProjectStore) CreateWBSElement(context.Context, []uuid.UUID, prjdb.CreateWBSElementParams) (prjdb.CreateWBSElementRow, error) {
	return prjdb.CreateWBSElementRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (f *fakeProjectStore) ListWBSElements(context.Context, []uuid.UUID, uuid.UUID) ([]prjdb.PrjWbsElement, error) {
	return nil, nil
}
func (f *fakeProjectStore) UpdateWBSElement(context.Context, []uuid.UUID, prjdb.UpdateWBSElementParams) (int32, error) {
	return 2, nil
}
func (f *fakeProjectStore) DeleteWBSElement(context.Context, []uuid.UUID, uuid.UUID, uuid.UUID, int32) error {
	return nil
}
func (f *fakeProjectStore) CreateBudget(context.Context, []uuid.UUID, prjdb.CreateProjectBudgetParams) (prjdb.CreateProjectBudgetRow, error) {
	return prjdb.CreateProjectBudgetRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (f *fakeProjectStore) ListBudgets(context.Context, []uuid.UUID, uuid.UUID) ([]prjdb.PrjProjectBudget, error) {
	return nil, nil
}
func (f *fakeProjectStore) ApproveBudget(context.Context, []uuid.UUID, uuid.UUID, uuid.UUID, int32) (int32, error) {
	return 2, nil
}
func (f *fakeProjectStore) CreateCost(context.Context, []uuid.UUID, prjdb.CreateProjectCostParams) (prjdb.CreateProjectCostRow, error) {
	return prjdb.CreateProjectCostRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (f *fakeProjectStore) ListCosts(context.Context, []uuid.UUID, uuid.UUID) ([]prjdb.PrjProjectCost, error) {
	return nil, nil
}
func (f *fakeProjectStore) UpsertProgress(context.Context, []uuid.UUID, prjdb.UpsertProjectProgressParams) (prjdb.UpsertProjectProgressRow, error) {
	return prjdb.UpsertProjectProgressRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (f *fakeProjectStore) ListProgress(context.Context, []uuid.UUID, uuid.UUID) ([]prjdb.PrjProjectProgress, error) {
	return nil, nil
}
func (f *fakeProjectStore) CreateBilling(context.Context, []uuid.UUID, prjdb.CreateProjectBillingParams) (prjdb.CreateProjectBillingRow, error) {
	return prjdb.CreateProjectBillingRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (f *fakeProjectStore) ListBillings(context.Context, []uuid.UUID, *uuid.UUID, int32, int32) ([]prjdb.PrjProjectBilling, int64, error) {
	return nil, 0, nil
}
func (f *fakeProjectStore) ApproveBillingAndEmit(_ context.Context, _ []uuid.UUID, _, _, _, _ uuid.UUID, _ int32, _ *uuid.UUID, _ *string) (int32, string, error) {
	f.approvedRef = "PB-PRJ01-12345678"
	return 2, f.approvedRef, nil
}
func (f *fakeProjectStore) CreateChangeOrder(context.Context, []uuid.UUID, prjdb.CreateChangeOrderParams) (prjdb.CreateChangeOrderRow, error) {
	return prjdb.CreateChangeOrderRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (f *fakeProjectStore) ListChangeOrders(context.Context, []uuid.UUID, uuid.UUID) ([]prjdb.PrjChangeOrder, error) {
	return nil, nil
}
func (f *fakeProjectStore) DecideChangeOrder(context.Context, []uuid.UUID, uuid.UUID, uuid.UUID, int32, string) (int32, error) {
	return 2, nil
}

func newTestMux(store ProjectStore) http.Handler {
	mux := http.NewServeMux()
	NewServer(store).Register(mux)
	return httpx.Middleware(mux)
}

func authHeaders(r *http.Request, perms string) {
	org := uuid.NewString()
	r.Header.Set("X-Organization-Id", org)
	r.Header.Set("X-User-Id", uuid.NewString())
	r.Header.Set("X-Permissions", perms)
	r.Header.Set("Content-Type", "application/json")
}

func TestProjectCreateAndProgressValidation(t *testing.T) {
	h := newTestMux(&fakeProjectStore{})

	req := httptest.NewRequest(http.MethodPost, "/projects", strings.NewReader(`{"code":"PRJ-1","name":"Tower A","projectType":"Construction","currencyCode":"IDR","startDate":"2026-06-01","endDate":"2026-05-01"}`))
	authHeaders(req, "project:write")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for endDate < startDate, got %d: %s", rec.Code, rec.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodPost, "/wbs-elements/"+uuid.NewString()+"/progress", strings.NewReader(`{"periodDate":"2026-05-01","physicalProgressPct":"105"}`))
	authHeaders(req2, "project:write")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for progress > 100, got %d: %s", rec2.Code, rec2.Body.String())
	}
}

func TestProjectBillingApproveEmitsInvoiceRef(t *testing.T) {
	store := &fakeProjectStore{}
	h := newTestMux(store)
	req := httptest.NewRequest(http.MethodPost, "/project-billings/"+uuid.NewString()+"/approve", strings.NewReader(`{}`))
	authHeaders(req, "project:write")
	req.Header.Set("If-Match", `W/"1"`)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || store.approvedRef == "" {
		t.Fatalf("expected 200 and invoice ref, got %d: %s", rec.Code, rec.Body.String())
	}
}
