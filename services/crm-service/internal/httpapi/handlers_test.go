package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/crm-service/internal/crmdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

type fakeCRMStore struct {
	won bool
}

func (f *fakeCRMStore) CreateLead(context.Context, []uuid.UUID, crmdb.CreateLeadParams) (crmdb.CreateLeadRow, error) {
	return crmdb.CreateLeadRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (f *fakeCRMStore) GetLead(context.Context, []uuid.UUID, uuid.UUID) (crmdb.CrmLead, error) {
	return crmdb.CrmLead{ID: uuid.New(), RowVersion: 1}, nil
}
func (f *fakeCRMStore) ListLeads(context.Context, []uuid.UUID, *string, int32, int32) ([]crmdb.CrmLead, int64, error) {
	return nil, 0, nil
}
func (f *fakeCRMStore) UpdateLead(context.Context, []uuid.UUID, crmdb.UpdateLeadParams) (int32, error) {
	return 2, nil
}
func (f *fakeCRMStore) TransitionLead(context.Context, []uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, int32, string, string, *uuid.UUID) (int32, *uuid.UUID, error) {
	return 2, nil, nil
}
func (f *fakeCRMStore) LoseLead(context.Context, []uuid.UUID, uuid.UUID, uuid.UUID, int32) (int32, error) {
	return 2, nil
}
func (f *fakeCRMStore) DeleteLead(context.Context, []uuid.UUID, uuid.UUID, uuid.UUID, int32) error {
	return nil
}
func (f *fakeCRMStore) CreateOpportunity(context.Context, []uuid.UUID, crmdb.CreateOpportunityParams) (crmdb.CreateOpportunityRow, error) {
	return crmdb.CreateOpportunityRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (f *fakeCRMStore) GetOpportunity(context.Context, []uuid.UUID, uuid.UUID) (crmdb.CrmOpportunity, error) {
	return crmdb.CrmOpportunity{ID: uuid.New(), RowVersion: 1}, nil
}
func (f *fakeCRMStore) ListOpportunities(context.Context, []uuid.UUID, *string, int32, int32) ([]crmdb.CrmOpportunity, int64, error) {
	return nil, 0, nil
}
func (f *fakeCRMStore) UpdateOpportunity(context.Context, []uuid.UUID, crmdb.UpdateOpportunityParams) (int32, error) {
	return 2, nil
}
func (f *fakeCRMStore) WinOpportunityAndEmit(context.Context, []uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, int32, string) (int32, error) {
	f.won = true
	return 2, nil
}
func (f *fakeCRMStore) LoseOpportunity(context.Context, []uuid.UUID, uuid.UUID, uuid.UUID, int32) (int32, error) {
	return 2, nil
}
func (f *fakeCRMStore) CreateSegment(context.Context, []uuid.UUID, crmdb.CreateCustomerSegmentParams) (crmdb.CreateCustomerSegmentRow, error) {
	return crmdb.CreateCustomerSegmentRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (f *fakeCRMStore) ListSegments(context.Context, []uuid.UUID, int32, int32) ([]crmdb.CrmCustomerSegment, int64, error) {
	return nil, 0, nil
}
func (f *fakeCRMStore) UpdateSegment(context.Context, []uuid.UUID, crmdb.UpdateCustomerSegmentParams) (int32, error) {
	return 2, nil
}
func (f *fakeCRMStore) DeleteSegment(context.Context, []uuid.UUID, uuid.UUID, uuid.UUID, int32) error {
	return nil
}
func (f *fakeCRMStore) CreateInteraction(context.Context, []uuid.UUID, crmdb.CreateInteractionParams) (crmdb.CreateInteractionRow, error) {
	return crmdb.CreateInteractionRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (f *fakeCRMStore) ListInteractions(context.Context, []uuid.UUID, int32, int32) ([]crmdb.CrmInteraction, int64, error) {
	return nil, 0, nil
}
func (f *fakeCRMStore) CreateSupportCase(context.Context, []uuid.UUID, crmdb.CreateSupportCaseParams) (crmdb.CreateSupportCaseRow, error) {
	return crmdb.CreateSupportCaseRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (f *fakeCRMStore) GetSupportCase(context.Context, []uuid.UUID, uuid.UUID) (crmdb.CrmSupportCase, error) {
	return crmdb.CrmSupportCase{ID: uuid.New(), RowVersion: 1}, nil
}
func (f *fakeCRMStore) ListSupportCases(context.Context, []uuid.UUID, *string, int32, int32) ([]crmdb.CrmSupportCase, int64, error) {
	return nil, 0, nil
}
func (f *fakeCRMStore) EscalateSupportCase(context.Context, []uuid.UUID, uuid.UUID, uuid.UUID, int32) (int32, error) {
	return 2, nil
}
func (f *fakeCRMStore) ResolveSupportCase(context.Context, []uuid.UUID, uuid.UUID, uuid.UUID, int32, *string) (int32, error) {
	return 3, nil
}
func (f *fakeCRMStore) CloseSupportCase(context.Context, []uuid.UUID, uuid.UUID, uuid.UUID, int32) (int32, error) {
	return 4, nil
}
func (f *fakeCRMStore) CreateActivityTask(context.Context, []uuid.UUID, crmdb.CreateActivityTaskParams) (crmdb.CreateActivityTaskRow, error) {
	return crmdb.CreateActivityTaskRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (f *fakeCRMStore) ListActivityTasks(context.Context, []uuid.UUID, *string, int32, int32) ([]crmdb.CrmActivityTask, int64, error) {
	return nil, 0, nil
}
func (f *fakeCRMStore) CompleteActivityTask(context.Context, []uuid.UUID, uuid.UUID, uuid.UUID, int32) (int32, error) {
	return 2, nil
}

func newTestMux(store CRMStore) http.Handler {
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

func TestCRMOpportunityValidationAndWin(t *testing.T) {
	store := &fakeCRMStore{}
	h := newTestMux(store)

	req := httptest.NewRequest(http.MethodPost, "/opportunities", strings.NewReader(`{"code":"OPP-1","stage":"Proposal","probabilityPct":"120"}`))
	authHeaders(req, "crm:write")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for probabilityPct > 100, got %d", rec.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/opportunities/"+uuid.NewString()+"/win", strings.NewReader(`{"currencyCode":"IDR"}`))
	authHeaders(req2, "crm:write")
	req2.Header.Set("If-Match", `W/"1"`)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK || !store.won {
		t.Fatalf("expected 200 and won=true, got %d: %s", rec2.Code, rec2.Body.String())
	}
}
