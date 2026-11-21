// Package httpapi wires security-service REST endpoints (RBAC roles,
// permissions catalog, and role-permission assignments).
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/security-service/internal/securitydb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type Store interface {
	CreateAndEmit(ctx context.Context, scope []uuid.UUID, arg securitydb.CreateRoleParams, ev securitydb.InsertOutboxParams) (securitydb.CreateRoleRow, error)
	GetWithPermissions(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (securitydb.SecRole, []securitydb.ListRolePermissionsRow, error)
	ListWithCount(ctx context.Context, scope []uuid.UUID, arg securitydb.ListRolesParams) ([]securitydb.ListRolesRow, int64, error)
	UpdateAndEmit(ctx context.Context, scope []uuid.UUID, arg securitydb.UpdateRoleParams, ev securitydb.InsertOutboxParams) (int32, error)
	DeleteRole(ctx context.Context, scope []uuid.UUID, arg securitydb.DeleteRoleParams) (int64, error)
	CreatePermission(ctx context.Context, scope []uuid.UUID, arg securitydb.CreatePermissionParams) (securitydb.CreatePermissionRow, error)
	ListPermissionsWithCount(ctx context.Context, scope []uuid.UUID, arg securitydb.ListPermissionsParams) ([]securitydb.ListPermissionsRow, int64, error)
	GrantPermissionAndEmit(ctx context.Context, scope []uuid.UUID, arg securitydb.CreateRolePermissionParams, ev securitydb.InsertOutboxParams) (int64, error)
	RevokePermissionAndEmit(ctx context.Context, scope []uuid.UUID, arg securitydb.DeleteRolePermissionParams, ev securitydb.InsertOutboxParams) (int64, error)
}

type Server struct {
	repo Store
	idem *httpx.IdemStore
}

func NewServer(repo Store) *Server {
	return &Server{repo: repo, idem: httpx.NewIdemStore()}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /roles", s.createRole)
	mux.HandleFunc("GET /roles", s.listRoles)
	mux.HandleFunc("GET /roles/{id}", s.getRole)
	mux.HandleFunc("PUT /roles/{id}", s.updateRole)
	mux.HandleFunc("DELETE /roles/{id}", s.deleteRole)
	mux.HandleFunc("GET /roles/{id}/permissions", s.listRolePermissions)
	mux.HandleFunc("POST /roles/{id}/permissions", s.grantRolePermission)
	mux.HandleFunc("DELETE /roles/{id}/permissions/{permissionId}", s.revokeRolePermission)

	mux.HandleFunc("GET /permissions", s.listPermissions)
	mux.HandleFunc("POST /permissions", s.createPermission)
}

type createRoleReq struct {
	Code             string `json:"code"`
	Name             string `json:"name"`
	IsSystem         bool   `json:"isSystem"`
	AdminTier        *int   `json:"adminTier,omitempty"`
	DefaultScopeType string `json:"defaultScopeType,omitempty"`
}

type updateRoleReq struct {
	Name      string `json:"name"`
	IsSystem  bool   `json:"isSystem"`
	AdminTier *int   `json:"adminTier,omitempty"`
}

type createPermReq struct {
	Code   string `json:"code"`
	Module string `json:"module"`
	Action string `json:"action"`
}

type grantPermReq struct {
	PermissionID   uuid.UUID `json:"permissionId"`
	PermissionCode string    `json:"permissionCode,omitempty"`
}

func isReservedAdminRoleCode(code string) bool {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "SYSTEM_ADMIN", "HOLDING_ADMIN", "SUBSIDIARY_ADMIN", "BRANCH_ADMIN":
		return true
	default:
		return false
	}
}

func defaultScopeForRoleCode(code string) string {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "SYSTEM_ADMIN":
		return "all"
	case "HOLDING_ADMIN":
		return "holding"
	case "SUBSIDIARY_ADMIN":
		return "subsidiary"
	case "BRANCH_ADMIN":
		return "branch"
	default:
		return "organization"
	}
}

func (s *Server) createRole(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "security:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	actor, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req createRoleReq
	key, hash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	code := strings.TrimSpace(req.Code)
	name := strings.TrimSpace(req.Name)
	if code == "" || name == "" {
		httpx.WriteError(w, r, httpx.BadRequest("code dan name wajib diisi"))
		return
	}
	if req.IsSystem {
		httpx.WriteError(w, r, httpx.Forbidden("tidak diizinkan membuat system role melalui endpoint tenant"))
		return
	}
	if len(httpx.Roles(r.Context())) > 0 {
		callerTier := httpx.CallerAdminTier(r.Context())
		if isReservedAdminRoleCode(code) && callerTier > httpx.AdminTierSystem {
			httpx.WriteError(w, r, httpx.Forbidden("kode role admin berjenjang hanya boleh dikelola oleh SYSTEM_ADMIN"))
			return
		}
		targetTier := httpx.AdminTierNone
		if req.AdminTier != nil {
			targetTier = *req.AdminTier
		}
		if !httpx.CanDelegateTier(callerTier, targetTier) {
			httpx.WriteError(w, r, httpx.Forbidden("admin tier tidak mencukupi untuk membuat role pada tingkat ini"))
			return
		}
	}
	payload, _ := json.Marshal(map[string]any{"code": code, "name": name, "isSystem": false})
	ev := securitydb.InsertOutboxParams{
		AggregateType:  "Role",
		EventType:      "security.role.upserted",
		Payload:        payload,
		OrganizationID: scope[0],
		CorrelationID:  httpx.CorrelationUUID(r),
	}
	created, err := s.repo.CreateAndEmit(r.Context(), scope, securitydb.CreateRoleParams{
		OrganizationID: scope[0],
		Code:           code,
		Name:           name,
		IsSystem:       false,
		CreatedBy:      actor,
	}, ev)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "kode role sudah ada"))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, key, hash, http.StatusCreated, created.RowVersion, map[string]any{
		"id":               created.ID,
		"code":             code,
		"name":             name,
		"isSystem":         false,
		"adminTier":        httpx.AdminTierForRoles([]string{code}),
		"defaultScopeType": defaultScopeForRoleCode(code),
		"rowVersion":       created.RowVersion,
	})
}

func (s *Server) listRoles(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "security:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var isSys *bool
	if v := strings.TrimSpace(r.URL.Query().Get("isSystem")); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("isSystem harus true/false"))
			return
		}
		isSys = &b
	}
	rows, total, err := s.repo.ListWithCount(r.Context(), scope, securitydb.ListRolesParams{
		Limit:    limit,
		Offset:   offset,
		IsSystem: isSys,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, rl := range rows {
		out = append(out, map[string]any{
			"id":               rl.ID,
			"organizationId":   rl.OrganizationID,
			"code":             rl.Code,
			"name":             rl.Name,
			"isSystem":         rl.IsSystem,
			"adminTier":        httpx.AdminTierForRoles([]string{rl.Code}),
			"defaultScopeType": defaultScopeForRoleCode(rl.Code),
			"rowVersion":       rl.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) getRole(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "security:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	role, perms, err := s.repo.GetWithPermissions(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("role tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, role.RowVersion, map[string]any{
		"id":             role.ID,
		"organizationId": role.OrganizationID,
		"code":           role.Code,
		"name":           role.Name,
		"isSystem":       role.IsSystem,
		"rowVersion":     role.RowVersion,
		"permissions":    formatPerms(perms),
	})
}

func (s *Server) updateRole(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "security:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	actor, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ver, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req updateRoleReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		httpx.WriteError(w, r, httpx.BadRequest("name wajib diisi"))
		return
	}
	if req.IsSystem {
		httpx.WriteError(w, r, httpx.Forbidden("tidak diizinkan mengubah role menjadi system role"))
		return
	}
	existing, _, err := s.repo.GetWithPermissions(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("role tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if existing.IsSystem || existing.OrganizationID == platform.SystemOrgID {
		httpx.WriteError(w, r, httpx.Forbidden("system role tidak dapat dimodifikasi"))
		return
	}
	if existing.RowVersion != ver {
		httpx.WriteError(w, r, httpx.Conflict("VERSION_CONFLICT", "row_version role tidak cocok"))
		return
	}
	payload, _ := json.Marshal(map[string]any{"code": existing.Code, "name": name, "isSystem": false})
	newVer, err := s.repo.UpdateAndEmit(r.Context(), scope, securitydb.UpdateRoleParams{
		ID:         id,
		RowVersion: ver,
		Name:       name,
		UpdatedBy:  &actor,
	}, securitydb.InsertOutboxParams{
		AggregateType:  "Role",
		AggregateID:    id,
		EventType:      "security.role.upserted",
		Payload:        payload,
		OrganizationID: existing.OrganizationID,
		CorrelationID:  httpx.CorrelationUUID(r),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.Conflict("VERSION_CONFLICT", "role berubah secara bersamaan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, newVer, map[string]any{
		"id":         id,
		"code":       existing.Code,
		"name":       name,
		"isSystem":   false,
		"rowVersion": newVer,
	})
}

func (s *Server) deleteRole(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "security:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	actor, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ver, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	existing, _, err := s.repo.GetWithPermissions(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("role tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if existing.IsSystem || existing.OrganizationID == platform.SystemOrgID {
		httpx.WriteError(w, r, httpx.Forbidden("role sistem tidak boleh dihapus"))
		return
	}
	if existing.RowVersion != ver {
		httpx.WriteError(w, r, httpx.Conflict("VERSION_CONFLICT", "row_version role tidak cocok"))
		return
	}
	n, err := s.repo.DeleteRole(r.Context(), scope, securitydb.DeleteRoleParams{
		ID:         id,
		RowVersion: ver,
		UpdatedBy:  &actor,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.Conflict("VERSION_CONFLICT", "role berubah secara bersamaan"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listPermissions(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "security:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var modPtr *string
	if m := strings.TrimSpace(r.URL.Query().Get("module")); m != "" {
		modPtr = &m
	}
	rows, total, err := s.repo.ListPermissionsWithCount(r.Context(), scope, securitydb.ListPermissionsParams{
		Limit:  limit,
		Offset: offset,
		Module: modPtr,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, p := range rows {
		out = append(out, map[string]any{
			"id":             p.ID,
			"organizationId": p.OrganizationID,
			"code":           p.Code,
			"module":         p.Module,
			"action":         p.Action,
			"rowVersion":     p.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) createPermission(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "security:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	actor, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req createPermReq
	key, hash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	code := strings.TrimSpace(req.Code)
	mod := strings.TrimSpace(req.Module)
	act := strings.TrimSpace(req.Action)
	if code == "" || mod == "" || act == "" {
		httpx.WriteError(w, r, httpx.BadRequest("code, module, dan action wajib diisi"))
		return
	}
	res, err := s.repo.CreatePermission(r.Context(), scope, securitydb.CreatePermissionParams{
		OrganizationID: scope[0],
		Code:           code,
		Module:         mod,
		Action:         act,
		CreatedBy:      actor,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "kode permission sudah ada"))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, key, hash, http.StatusCreated, res.RowVersion, map[string]any{
		"id":         res.ID,
		"code":       code,
		"module":     mod,
		"action":     act,
		"rowVersion": res.RowVersion,
	})
}

func (s *Server) listRolePermissions(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "security:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	_, perms, err := s.repo.GetWithPermissions(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("role tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := formatPerms(perms)
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(0, len(out), len(out)),
	})
}

func (s *Server) grantRolePermission(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "security:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	actor, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	roleID, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req grantPermReq
	key, hash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if req.PermissionID == uuid.Nil {
		httpx.WriteError(w, r, httpx.BadRequest("permissionId wajib diisi"))
		return
	}
	existing, _, err := s.repo.GetWithPermissions(r.Context(), scope, roleID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("role tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if existing.IsSystem || existing.OrganizationID == platform.SystemOrgID {
		httpx.WriteError(w, r, httpx.Forbidden("system role tidak dapat dimodifikasi"))
		return
	}
	if len(httpx.Roles(r.Context())) > 0 && httpx.CallerAdminTier(r.Context()) >= httpx.AdminTierSubsidiary {
		pCode := strings.ToLower(strings.TrimSpace(req.PermissionCode))
		if strings.HasPrefix(pCode, "consolidation.") || strings.HasPrefix(pCode, "consolidation:") {
			httpx.WriteError(w, r, httpx.Forbidden("permission konsolidasi hanya boleh diberikan oleh Head Office / Holding Admin"))
			return
		}
	}
	payload, _ := json.Marshal(map[string]any{"roleId": roleID, "permissionId": req.PermissionID})
	_, err = s.repo.GrantPermissionAndEmit(r.Context(), scope, securitydb.CreateRolePermissionParams{
		OrganizationID: existing.OrganizationID,
		RoleID:         roleID,
		PermissionID:   req.PermissionID,
		CreatedBy:      actor,
	}, securitydb.InsertOutboxParams{
		AggregateType:  "Role",
		AggregateID:    roleID,
		EventType:      "security.role.upserted",
		Payload:        payload,
		OrganizationID: existing.OrganizationID,
		CorrelationID:  httpx.CorrelationUUID(r),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("role tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, key, hash, http.StatusCreated, 0, map[string]any{
		"roleId":       roleID,
		"permissionId": req.PermissionID,
	})
}

func (s *Server) revokeRolePermission(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "security:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	roleID, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	permID, err := httpx.ParseUUIDParam("permissionId", r.PathValue("permissionId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	existing, _, err := s.repo.GetWithPermissions(r.Context(), scope, roleID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("role tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if existing.IsSystem || existing.OrganizationID == platform.SystemOrgID {
		httpx.WriteError(w, r, httpx.Forbidden("system role tidak dapat dimodifikasi"))
		return
	}
	payload, _ := json.Marshal(map[string]any{"roleId": roleID, "permissionId": permID, "revoked": true})
	n, err := s.repo.RevokePermissionAndEmit(r.Context(), scope, securitydb.DeleteRolePermissionParams{
		RoleID:       roleID,
		PermissionID: permID,
	}, securitydb.InsertOutboxParams{
		AggregateType:  "Role",
		AggregateID:    roleID,
		EventType:      "security.role.upserted",
		Payload:        payload,
		OrganizationID: existing.OrganizationID,
		CorrelationID:  httpx.CorrelationUUID(r),
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.NotFound("permission pada role tidak ditemukan"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func formatPerms(rows []securitydb.ListRolePermissionsRow) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, p := range rows {
		out = append(out, map[string]any{
			"id":     p.ID,
			"code":   p.Code,
			"module": p.Module,
			"action": p.Action,
		})
	}
	return out
}
