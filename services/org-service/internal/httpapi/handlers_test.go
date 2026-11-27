package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/org-service/internal/orgdb"
	"github.com/sinergiinvestasiproperti/erp/services/org-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type fakeOrgStore struct {
	hasChildren bool
	typeCode    string
	attrs       map[string]any
}

func (f *fakeOrgStore) GetOrganizationTypeByID(_ context.Context, scope []uuid.UUID, id uuid.UUID) (orgdb.GetOrganizationTypeByIDRow, error) {
	code := f.typeCode
	if code == "" {
		code = "DIVISION"
	}
	return orgdb.GetOrganizationTypeByIDRow{ID: id, OrganizationID: scope[0], Code: code, Name: code, IsActive: true, RowVersion: 1}, nil
}

func (f *fakeOrgStore) ListAttributes(_ context.Context, _ []uuid.UUID, _ uuid.UUID) (map[string]any, error) {
	if f.attrs == nil {
		return map[string]any{}, nil
	}
	return f.attrs, nil
}

func (f *fakeOrgStore) UpsertAttributes(_ context.Context, _ []uuid.UUID, _, _ uuid.UUID, attrs map[string]any, _ uuid.UUID) error {
	f.attrs = attrs
	return nil
}

func (f *fakeOrgStore) CreateOrganizationType(_ context.Context, _ []uuid.UUID, _ orgdb.CreateOrganizationTypeParams) (orgdb.CreateOrganizationTypeRow, error) {
	return orgdb.CreateOrganizationTypeRow{ID: uuid.MustParse("10000000-0000-0000-0000-000000000040"), RowVersion: 1}, nil
}

func (f *fakeOrgStore) ListOrganizationTypesWithCount(_ context.Context, scope []uuid.UUID, _ orgdb.ListOrganizationTypesParams) ([]orgdb.ListOrganizationTypesRow, int64, error) {
	return []orgdb.ListOrganizationTypesRow{{ID: uuid.New(), OrganizationID: scope[0], Code: "CO", Name: "Company", IsActive: true, RowVersion: 1}}, 1, nil
}

func (f *fakeOrgStore) CreateAndEmit(_ context.Context, _ []uuid.UUID, _ orgdb.CreateOrganizationParams, _ orgdb.InsertOutboxParams) (orgdb.CreateOrganizationRow, error) {
	return orgdb.CreateOrganizationRow{ID: uuid.MustParse("10000000-0000-0000-0000-000000000041"), Status: "Draft", RowVersion: 1}, nil
}

func (f *fakeOrgStore) GetWithHierarchy(_ context.Context, scope []uuid.UUID, id uuid.UUID) (orgdb.OrgOrganization, []orgdb.ListOrganizationDescendantsRow, error) {
	dt, _ := platform.ParseDate("2026-01-01")
	return orgdb.OrgOrganization{
		ID:                 id,
		OrganizationID:     scope[0],
		Code:               "PT-HOLD",
		Name:               "Holding",
		OrganizationTypeID: uuid.MustParse("10000000-0000-0000-0000-000000000040"),
		Path:               "/PT-HOLD",
		Status:             "Draft",
		EffectiveFrom:      dt,
		RowVersion:         1,
	}, []orgdb.ListOrganizationDescendantsRow{
		{ID: id, Code: "PT-HOLD", Name: "Holding", Path: "/PT-HOLD", Status: "Draft", Depth: 0, RowVersion: 1},
	}, nil
}

func (f *fakeOrgStore) ListWithCount(_ context.Context, scope []uuid.UUID, _ orgdb.ListOrganizationsParams) ([]orgdb.ListOrganizationsRow, int64, error) {
	dt, _ := platform.ParseDate("2026-01-01")
	return []orgdb.ListOrganizationsRow{{
		ID:                 uuid.MustParse("10000000-0000-0000-0000-000000000041"),
		OrganizationID:     scope[0],
		Code:               "PT-HOLD",
		Name:               "Holding",
		OrganizationTypeID: uuid.MustParse("10000000-0000-0000-0000-000000000040"),
		Path:               "/PT-HOLD",
		Status:             "Active",
		EffectiveFrom:      dt,
		RowVersion:         1,
	}}, 1, nil
}

func (f *fakeOrgStore) UpdateAndEmit(_ context.Context, _ []uuid.UUID, _ orgdb.UpdateOrganizationParams, _ orgdb.InsertOutboxParams) (orgdb.UpdateOrganizationRow, error) {
	return orgdb.UpdateOrganizationRow{Status: "Draft", RowVersion: 2}, nil
}

func (f *fakeOrgStore) ActivateAndEmit(_ context.Context, _ []uuid.UUID, _ orgdb.ActivateOrganizationParams, _ orgdb.InsertOutboxParams) (int32, error) {
	return 2, nil
}

func (f *fakeOrgStore) DeleteOrganization(_ context.Context, _ []uuid.UUID, _ orgdb.DeleteOrganizationParams) (int64, error) {
	if f.hasChildren {
		return 0, repository.ErrHasChildren
	}
	return 1, nil
}

func orgHdrs(perms string) map[string]string {
	return map[string]string{
		"X-Org-Scope":   "00000000-0000-0000-0000-000000000001",
		"X-User-ID":     "00000000-0000-0000-0000-000000000010",
		"X-Permissions": perms,
	}
}

func callOrg(t *testing.T, h http.Handler, method, path string, body any, hdrs map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	for k, v := range hdrs {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestOrgEndpointsHierarchyActivateAndChildrenGuard(t *testing.T) {
	store := &fakeOrgStore{hasChildren: true}
	mux := http.NewServeMux()
	NewServer(store).Register(mux)
	h := httpx.Middleware(mux)

	if rr := callOrg(t, h, http.MethodGet, "/org-units", nil, orgHdrs("ap:read")); rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without org:read, got %d", rr.Code)
	}

	hdrs := orgHdrs("org:read,org:write")
	hdrs["If-Match"] = `"1"`
	if rr := callOrg(t, h, http.MethodPost, "/organizations/10000000-0000-0000-0000-000000000041/activate", nil, hdrs); rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on activate, got %d (%s)", rr.Code, rr.Body.String())
	}

	if rr := callOrg(t, h, http.MethodGet, "/org-units/10000000-0000-0000-0000-000000000041/hierarchy", nil, hdrs); rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on hierarchy, got %d", rr.Code)
	}

	if rr := callOrg(t, h, http.MethodDelete, "/organizations/10000000-0000-0000-0000-000000000041", nil, hdrs); rr.Code != http.StatusConflict {
		t.Fatalf("expected 409 HAS_CHILDREN, got %d", rr.Code)
	}

	store.hasChildren = false
	if rr := callOrg(t, h, http.MethodDelete, "/organizations/10000000-0000-0000-0000-000000000041", nil, hdrs); rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204 when no children, got %d", rr.Code)
	}
}
