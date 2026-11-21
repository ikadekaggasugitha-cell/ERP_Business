package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/security-service/internal/securitydb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

type fakeSecurityStore struct {
	sysRole bool
}

func (f *fakeSecurityStore) CreateAndEmit(_ context.Context, _ []uuid.UUID, _ securitydb.CreateRoleParams, _ securitydb.InsertOutboxParams) (securitydb.CreateRoleRow, error) {
	return securitydb.CreateRoleRow{ID: uuid.MustParse("10000000-0000-0000-0000-000000000021"), RowVersion: 1}, nil
}

func (f *fakeSecurityStore) GetWithPermissions(_ context.Context, scope []uuid.UUID, id uuid.UUID) (securitydb.SecRole, []securitydb.ListRolePermissionsRow, error) {
	return securitydb.SecRole{
		ID:             id,
		OrganizationID: scope[0],
		Code:           "FIN_MGR",
		Name:           "Finance Manager",
		IsSystem:       f.sysRole,
		RowVersion:     1,
	}, []securitydb.ListRolePermissionsRow{{ID: uuid.New(), Code: "gl:read", Module: "gl", Action: "read"}}, nil
}

func (f *fakeSecurityStore) ListWithCount(_ context.Context, scope []uuid.UUID, _ securitydb.ListRolesParams) ([]securitydb.ListRolesRow, int64, error) {
	return []securitydb.ListRolesRow{{
		ID:             uuid.MustParse("10000000-0000-0000-0000-000000000021"),
		OrganizationID: scope[0],
		Code:           "FIN_MGR",
		Name:           "Finance Manager",
		RowVersion:     1,
	}}, 1, nil
}

func (f *fakeSecurityStore) UpdateAndEmit(_ context.Context, _ []uuid.UUID, _ securitydb.UpdateRoleParams, _ securitydb.InsertOutboxParams) (int32, error) {
	return 2, nil
}

func (f *fakeSecurityStore) DeleteRole(_ context.Context, _ []uuid.UUID, _ securitydb.DeleteRoleParams) (int64, error) {
	return 1, nil
}

func (f *fakeSecurityStore) CreatePermission(_ context.Context, _ []uuid.UUID, _ securitydb.CreatePermissionParams) (securitydb.CreatePermissionRow, error) {
	return securitydb.CreatePermissionRow{ID: uuid.New(), RowVersion: 1}, nil
}

func (f *fakeSecurityStore) ListPermissionsWithCount(_ context.Context, scope []uuid.UUID, _ securitydb.ListPermissionsParams) ([]securitydb.ListPermissionsRow, int64, error) {
	return []securitydb.ListPermissionsRow{{
		ID:             uuid.New(),
		OrganizationID: scope[0],
		Code:           "security:read",
		Module:         "security",
		Action:         "read",
		RowVersion:     1,
	}}, 1, nil
}

func (f *fakeSecurityStore) GrantPermissionAndEmit(_ context.Context, _ []uuid.UUID, _ securitydb.CreateRolePermissionParams, _ securitydb.InsertOutboxParams) (int64, error) {
	return 1, nil
}

func (f *fakeSecurityStore) RevokePermissionAndEmit(_ context.Context, _ []uuid.UUID, _ securitydb.DeleteRolePermissionParams, _ securitydb.InsertOutboxParams) (int64, error) {
	return 1, nil
}

func secHdrs(perms string) map[string]string {
	return map[string]string{
		"X-Org-Scope":   "00000000-0000-0000-0000-000000000001",
		"X-User-ID":     "00000000-0000-0000-0000-000000000010",
		"X-Permissions": perms,
	}
}

func sendReq(t *testing.T, h http.Handler, method, path string, body any, hdrs map[string]string) *httptest.ResponseRecorder {
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

func TestSecurityRolesPermissionsAndSystemLock(t *testing.T) {
	store := &fakeSecurityStore{sysRole: true}
	mux := http.NewServeMux()
	NewServer(store).Register(mux)
	h := httpx.Middleware(mux)

	if rr := sendReq(t, h, http.MethodGet, "/permissions", nil, secHdrs("ap:read")); rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without security:read, got %d", rr.Code)
	}

	hdrs := secHdrs("security:read,security:write")
	hdrs["If-Match"] = `"1"`
	if rr := sendReq(t, h, http.MethodDelete, "/roles/10000000-0000-0000-0000-000000000021", nil, hdrs); rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 deleting system role, got %d", rr.Code)
	}
	if rr := sendReq(t, h, http.MethodPut, "/roles/10000000-0000-0000-0000-000000000021", map[string]any{"name": "Tampered", "isSystem": false}, hdrs); rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 updating system role, got %d", rr.Code)
	}
	if rr := sendReq(t, h, http.MethodPost, "/roles/10000000-0000-0000-0000-000000000021/permissions", map[string]any{"permissionId": "20000000-0000-0000-0000-000000000001"}, hdrs); rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 granting permission to system role, got %d", rr.Code)
	}

	store.sysRole = false
	if rr := sendReq(t, h, http.MethodPut, "/roles/10000000-0000-0000-0000-000000000021", map[string]any{"name": "Escalated", "isSystem": true}, hdrs); rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 escalating custom role to system role, got %d", rr.Code)
	}
	if rr := sendReq(t, h, http.MethodDelete, "/roles/10000000-0000-0000-0000-000000000021", nil, hdrs); rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204 deleting custom role, got %d", rr.Code)
	}

	if rr := sendReq(t, h, http.MethodPost, "/roles/10000000-0000-0000-0000-000000000021/permissions", map[string]any{
		"permissionId": "20000000-0000-0000-0000-000000000001",
	}, hdrs); rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 granting permission, got %d", rr.Code)
	}

	if rr := sendReq(t, h, http.MethodPost, "/roles", map[string]any{"code": "ROOT_FAKE", "name": "Fake Root", "isSystem": true}, hdrs); rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 creating system role from tenant endpoint, got %d", rr.Code)
	}

	rr := sendReq(t, h, http.MethodGet, "/permissions", nil, hdrs)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 listing permissions, got %d", rr.Code)
	}
}

