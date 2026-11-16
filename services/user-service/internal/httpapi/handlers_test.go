package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/user-service/internal/userdb"
)

type fakeUserStore struct{}

func (f *fakeUserStore) CreateAndEmit(_ context.Context, _ []uuid.UUID, _ userdb.CreateAppUserParams, _ userdb.InsertOutboxParams) (userdb.CreateAppUserRow, error) {
	return userdb.CreateAppUserRow{ID: uuid.MustParse("10000000-0000-0000-0000-000000000011"), Status: "Active", RowVersion: 1}, nil
}

func (f *fakeUserStore) GetWithRoles(_ context.Context, scope []uuid.UUID, id uuid.UUID) (userdb.UsrAppUser, []userdb.ListUserOrgAssignmentsRow, error) {
	return userdb.UsrAppUser{
		ID:             id,
		OrganizationID: scope[0],
		Username:       "alice",
		Email:          "alice@ex.co",
		FullName:       "Alice",
		Status:         "Active",
		RowVersion:     1,
	}, []userdb.ListUserOrgAssignmentsRow{{ID: uuid.New(), UserID: id, OrganizationRefID: scope[0], RoleID: uuid.New(), RowVersion: 1}}, nil
}

func (f *fakeUserStore) ListWithCount(_ context.Context, scope []uuid.UUID, _ userdb.ListAppUsersParams) ([]userdb.ListAppUsersRow, int64, error) {
	return []userdb.ListAppUsersRow{{
		ID:             uuid.MustParse("10000000-0000-0000-0000-000000000011"),
		OrganizationID: scope[0],
		Username:       "alice",
		Email:          "alice@ex.co",
		FullName:       "Alice",
		Status:         "Active",
		RowVersion:     1,
	}}, 1, nil
}

func (f *fakeUserStore) UpdateAndEmit(_ context.Context, _ []uuid.UUID, _ userdb.UpdateAppUserParams, _ userdb.InsertOutboxParams) (userdb.UpdateAppUserRow, error) {
	return userdb.UpdateAppUserRow{Status: "Active", RowVersion: 2}, nil
}

func (f *fakeUserStore) ChangeStatusAndEmit(_ context.Context, _ []uuid.UUID, _ userdb.ChangeUserStatusParams, _ userdb.InsertOutboxParams) (userdb.ChangeUserStatusRow, error) {
	return userdb.ChangeUserStatusRow{Status: "Inactive", RowVersion: 2}, nil
}

func (f *fakeUserStore) DeleteUser(_ context.Context, _ []uuid.UUID, _ userdb.DeleteAppUserParams) (int64, error) {
	return 1, nil
}

func (f *fakeUserStore) AssignRoleAndEmit(_ context.Context, _ []uuid.UUID, _ userdb.CreateUserOrgAssignmentParams, _ userdb.InsertOutboxParams) (userdb.CreateUserOrgAssignmentRow, error) {
	return userdb.CreateUserOrgAssignmentRow{ID: uuid.New(), RowVersion: 1}, nil
}

func (f *fakeUserStore) RemoveRoleAndEmit(_ context.Context, _ []uuid.UUID, _ userdb.RemoveUserOrgAssignmentParams, _ userdb.InsertOutboxParams) (int64, error) {
	return 1, nil
}

func userHeaders(perms string) map[string]string {
	return map[string]string{
		"X-Org-Scope":   "00000000-0000-0000-0000-000000000001",
		"X-User-ID":     "00000000-0000-0000-0000-000000000010",
		"X-Permissions": perms,
	}
}

func callHTTP(t *testing.T, h http.Handler, method, path string, body any, hdrs map[string]string) *httptest.ResponseRecorder {
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

func TestUserEndpointsRBACCRUDDisableAndRoles(t *testing.T) {
	mux := http.NewServeMux()
	NewServer(&fakeUserStore{}).Register(mux)
	h := httpx.Middleware(mux)

	if rr := callHTTP(t, h, http.MethodGet, "/users", nil, userHeaders("ap:read")); rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without user:read, got %d", rr.Code)
	}

	if rr := callHTTP(t, h, http.MethodPost, "/users", map[string]any{
		"username": "bad",
		"email":    "not-an-email",
		"fullName": "Bad",
	}, userHeaders("user:write")); rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad email, got %d", rr.Code)
	}

	hdrs := userHeaders("user:read,user:write")
	rr := callHTTP(t, h, http.MethodPost, "/users", map[string]any{
		"username": "alice",
		"email":    "alice@ex.co",
		"fullName": "Alice",
	}, hdrs)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 on create user, got %d (%s)", rr.Code, rr.Body.String())
	}

	rr = callHTTP(t, h, http.MethodGet, "/users", nil, hdrs)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on list users, got %d", rr.Code)
	}
	var env httpx.ListEnvelope
	_ = json.Unmarshal(rr.Body.Bytes(), &env)
	if env.Page.TotalElements != 1 {
		t.Fatalf("expected totalElements=1, got %+v", env.Page)
	}

	hdrs["If-Match"] = `"1"`
	rr = callHTTP(t, h, http.MethodPost, "/users/10000000-0000-0000-0000-000000000011/disable", nil, hdrs)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on disable, got %d (%s)", rr.Code, rr.Body.String())
	}

	rr = callHTTP(t, h, http.MethodPost, "/users/10000000-0000-0000-0000-000000000011/roles", map[string]any{
		"roleId": "20000000-0000-0000-0000-000000000001",
	}, hdrs)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 on assign role, got %d (%s)", rr.Code, rr.Body.String())
	}

	rr = callHTTP(t, h, http.MethodDelete, "/users/10000000-0000-0000-0000-000000000011/roles/20000000-0000-0000-0000-000000000001", nil, hdrs)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204 on remove role, got %d", rr.Code)
	}
}

